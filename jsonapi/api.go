package jsonapi

import (
	"encoding/json"
	"fmt"

	"crown_and_coin/actions"
	"crown_and_coin/engine"
	"crown_and_coin/events"
	"crown_and_coin/phases"
)

// GameAPI provides a JSON message-based interface to the game engine
type GameAPI struct {
	engine      *engine.Engine
	actionQueue []actions.Action
	phases      map[engine.PhaseType]phases.Phase
	dice        engine.DiceRoller
}

// NewGameAPI creates a new JSON API wrapper
func NewGameAPI() *GameAPI {
	return NewGameAPIWithDice(engine.NewRandomDice())
}

// NewGameAPIWithDice creates a new JSON API with a specific dice roller (for testing)
func NewGameAPIWithDice(dice engine.DiceRoller) *GameAPI {
	api := &GameAPI{
		engine:      engine.NewEngine(dice),
		actionQueue: make([]actions.Action, 0),
		phases:      make(map[engine.PhaseType]phases.Phase),
		dice:        dice,
	}

	// Register all phases
	api.phases[engine.PhaseTaxation] = phases.NewTaxationPhase(dice)
	api.phases[engine.PhaseSpending] = phases.NewSpendingPhase(dice)
	api.phases[engine.PhaseWar] = phases.NewWarPhase(dice)
	api.phases[engine.PhaseAssessment] = phases.NewAssessmentPhase(dice)

	return api
}

// ProcessMessage handles a JSON message and returns a JSON response
func (api *GameAPI) ProcessMessage(data []byte) ([]byte, error) {
	reqType, req, err := ParseRequest(data)
	if err != nil {
		return api.errorResponse(fmt.Sprintf("invalid request: %v", err))
	}

	var response interface{}

	switch reqType {
	case RequestGetPlayers:
		response = api.handleGetPlayers()
	case RequestAddMerchant:
		response = api.handleAddMerchant(req.(*AddMerchantRequest))
	case RequestAddCountry:
		response = api.handleAddCountry(req.(*AddCountryRequest))
	case RequestRemoveMerchant:
		response = api.handleRemoveMerchant(req.(*RemoveMerchantRequest))
	case RequestGetState:
		response = api.handleGetState()
	case RequestGetActions:
		response = api.handleGetActions(req.(*GetActionsRequest))
	case RequestSubmit:
		response = api.handleSubmit(req.(*SubmitRequest))
	case RequestGetQueued:
		response = api.handleGetQueued(req.(*GetQueuedRequest))
	case RequestAdvance:
		response = api.handleAdvance()
	default:
		return api.errorResponse(fmt.Sprintf("unknown request type: %s", reqType))
	}

	return json.Marshal(response)
}

func (api *GameAPI) handleGetPlayers() *GetPlayersResponse {
	state := api.engine.GetState()
	players := make(map[string]*PlayerInfo)

	// Add monarchs
	for _, country := range state.Countries {
		if !country.IsRepublic && country.MonarchID != "" {
			players[country.MonarchID] = &PlayerInfo{
				CountryID: country.ID,
				Role:      "monarch",
			}
		}
	}

	// Add merchants
	for _, merchant := range state.Merchants {
		players[merchant.ID] = &PlayerInfo{
			CountryID: merchant.CountryID,
			Role:      "merchant",
		}
	}

	return &GetPlayersResponse{
		Success: true,
		Players: players,
	}
}

func (api *GameAPI) handleAddMerchant(req *AddMerchantRequest) *AddMerchantResponse {
	state := api.engine.GetState()

	// Check if player_id is already in use (as merchant or monarch)
	if state.GetMerchant(req.PlayerID) != nil {
		return &AddMerchantResponse{
			Success: false,
			Error:   fmt.Sprintf("player_id '%s' already exists as a merchant", req.PlayerID),
		}
	}
	for _, country := range state.Countries {
		if country.MonarchID == req.PlayerID {
			return &AddMerchantResponse{
				Success: false,
				Error:   fmt.Sprintf("player_id '%s' already exists as a monarch", req.PlayerID),
			}
		}
	}

	// Check if country exists
	if state.GetCountry(req.CountryID) == nil {
		return &AddMerchantResponse{
			Success: false,
			Error:   fmt.Sprintf("country_id '%s' not found", req.CountryID),
		}
	}

	merchant := engine.NewMerchant(req.PlayerID, req.CountryID)
	state.AddMerchant(merchant)

	return &AddMerchantResponse{Success: true}
}

func (api *GameAPI) handleAddCountry(req *AddCountryRequest) *AddCountryResponse {
	state := api.engine.GetState()

	// Check if country_id is already in use
	if state.GetCountry(req.CountryID) != nil {
		return &AddCountryResponse{
			Success: false,
			Error:   fmt.Sprintf("country_id '%s' already exists", req.CountryID),
		}
	}

	// Check if monarch_id is already in use
	for _, country := range state.Countries {
		if country.MonarchID == req.MonarchID {
			return &AddCountryResponse{
				Success: false,
				Error:   fmt.Sprintf("monarch_id '%s' already exists as a monarch", req.MonarchID),
			}
		}
	}
	if state.GetMerchant(req.MonarchID) != nil {
		return &AddCountryResponse{
			Success: false,
			Error:   fmt.Sprintf("monarch_id '%s' already exists as a merchant", req.MonarchID),
		}
	}

	country := engine.NewCountry(req.CountryID, req.MonarchID)
	state.AddCountry(country)

	return &AddCountryResponse{Success: true}
}

func (api *GameAPI) handleRemoveMerchant(req *RemoveMerchantRequest) *RemoveMerchantResponse {
	state := api.engine.GetState()

	if !state.RemoveMerchant(req.PlayerID) {
		return &RemoveMerchantResponse{
			Success: false,
			Error:   fmt.Sprintf("player_id '%s' not found", req.PlayerID),
		}
	}

	return &RemoveMerchantResponse{Success: true}
}

func (api *GameAPI) handleGetState() *StateResponse {
	return &StateResponse{
		Success: true,
		State:   SerializeState(api.engine.GetState()),
	}
}

func (api *GameAPI) handleGetActions(req *GetActionsRequest) *ActionsResponse {
	state := api.engine.GetState()
	currentPhase := state.Phase

	// Get the phase handler
	phase, ok := api.phases[currentPhase]
	if !ok {
		// No actions for this phase (e.g., Negotiation)
		return &ActionsResponse{
			Success:  true,
			PlayerID: req.PlayerID,
			Phase:    currentPhase.String(),
			Actions:  []ActionJSON{},
		}
	}

	// Get valid actions for this player
	validActions := phase.ValidActions(state, req.PlayerID)

	// Convert to JSON format
	actionJSONs := make([]ActionJSON, len(validActions))
	for i, action := range validActions {
		actionJSONs[i] = SerializeAction(action, state)
	}

	return &ActionsResponse{
		Success:  true,
		PlayerID: req.PlayerID,
		Phase:    currentPhase.String(),
		Actions:  actionJSONs,
	}
}

func (api *GameAPI) handleSubmit(req *SubmitRequest) *SubmitResponse {
	state := api.engine.GetState()

	for _, aj := range req.Actions {
		action, err := DeserializeAction(aj)
		if err != nil {
			continue // Skip invalid actions
		}

		// Validate against current state
		if err := action.Validate(state); err != nil {
			continue // Skip actions that fail validation
		}

		api.actionQueue = append(api.actionQueue, action)
	}

	return &SubmitResponse{
		Success:       true,
		QueuedActions: len(api.actionQueue),
		Phase:         state.Phase.String(),
	}
}

func (api *GameAPI) handleGetQueued(req *GetQueuedRequest) *QueuedResponse {
	state := api.engine.GetState()

	var filteredActions []ActionJSON
	for _, action := range api.actionQueue {
		// If player_id is specified, only include actions from that player
		if req.PlayerID != "" && action.PlayerID() != req.PlayerID {
			continue
		}
		filteredActions = append(filteredActions, SerializeAction(action, state))
	}

	return &QueuedResponse{
		Success: true,
		Phase:   state.Phase.String(),
		Actions: filteredActions,
	}
}

func (api *GameAPI) handleAdvance() *AdvanceResponse {
	state := api.engine.GetState()
	previousPhase := state.Phase

	// Get the phase handler
	phase, ok := api.phases[previousPhase]
	var allEvents []events.Event

	if ok {
		// Execute the phase with queued actions
		newState, phaseEvents := phase.Execute(state, api.actionQueue)
		api.engine.SetState(newState)
		allEvents = phaseEvents
	}

	// Clear action queue for next phase
	api.actionQueue = make([]actions.Action, 0)

	// Advance to next phase
	api.engine.GetState().NextPhase()
	newState := api.engine.GetState()

	return &AdvanceResponse{
		Success:       true,
		PreviousPhase: previousPhase.String(),
		CurrentPhase:  newState.Phase.String(),
		Turn:          newState.Turn,
		Events:        SerializeEvents(allEvents),
		State:         SerializeState(newState),
	}
}

func (api *GameAPI) errorResponse(message string) ([]byte, error) {
	return json.Marshal(&ErrorResponse{
		Success: false,
		Error:   message,
	})
}

// GetEngine returns the underlying engine (for testing)
func (api *GameAPI) GetEngine() *engine.Engine {
	return api.engine
}
