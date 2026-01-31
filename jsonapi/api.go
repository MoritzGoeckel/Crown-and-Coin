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
	case RequestSetup:
		response = api.handleSetup(req.(*SetupRequest))
	case RequestGetState:
		response = api.handleGetState()
	case RequestGetActions:
		response = api.handleGetActions(req.(*GetActionsRequest))
	case RequestSubmit:
		response = api.handleSubmit(req.(*SubmitRequest))
	case RequestGetQueued:
		response = api.handleGetQueued()
	case RequestAdvance:
		response = api.handleAdvance()
	default:
		return api.errorResponse(fmt.Sprintf("unknown request type: %s", reqType))
	}

	return json.Marshal(response)
}

func (api *GameAPI) handleSetup(req *SetupRequest) *SetupResponse {
	// Create countries
	countries := make([]*engine.Country, len(req.Countries))
	for i, c := range req.Countries {
		countries[i] = engine.NewCountry(c.ID, c.MonarchID)
	}

	// Create merchants
	merchants := make([]*engine.Merchant, len(req.Merchants))
	for i, m := range req.Merchants {
		merchants[i] = engine.NewMerchant(m.ID, m.CountryID)
	}

	// Setup the game
	api.engine.SetupGame(countries, merchants)

	// Clear action queue
	api.actionQueue = make([]actions.Action, 0)

	return &SetupResponse{
		Success: true,
		State:   SerializeState(api.engine.GetState()),
	}
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

func (api *GameAPI) handleGetQueued() *QueuedResponse {
	state := api.engine.GetState()

	actionJSONs := make([]ActionJSON, len(api.actionQueue))
	for i, action := range api.actionQueue {
		actionJSONs[i] = SerializeAction(action, state)
	}

	return &QueuedResponse{
		Success: true,
		Phase:   state.Phase.String(),
		Actions: actionJSONs,
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
