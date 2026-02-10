package engine

import (
	"crown_and_coin/events"
)

// ActionProvider interface for getting player actions
// This allows different input methods (AI, network, CLI, etc.)
type ActionProvider interface {
	// GetActions returns the actions for a given phase
	// The provider receives the current state and phase, and returns actions
	GetActions(state *GameState, phase PhaseType) []Action
}

// Action interface (simplified for engine package)
type Action interface {
	Type() interface{}
	PlayerID() string
	Validate(state *GameState) error
	Apply(state *GameState, roller DiceRoller) (*GameState, []events.Event)
}

// Phase interface for the engine
type Phase interface {
	Name() string
	Type() PhaseType
	Execute(state *GameState, actions []Action) (*GameState, []events.Event)
}

// Engine orchestrates the game loop
type Engine struct {
	state          *GameState
	phases         map[PhaseType]Phase
	dice           DiceRoller
	eventBus       *events.EventBus
	pendingActions []interface{}
}

// NewEngine creates a new game engine
func NewEngine(dice DiceRoller) *Engine {
	return &Engine{
		state:          NewGameState(),
		phases:         make(map[PhaseType]Phase),
		dice:           dice,
		eventBus:       events.NewEventBus(),
		pendingActions: make([]interface{}, 0),
	}
}

// RegisterPhase adds a phase handler to the engine
func (e *Engine) RegisterPhase(phase Phase) {
	e.phases[phase.Type()] = phase
}

// GetState returns the current game state
func (e *Engine) GetState() *GameState {
	return e.state
}

// SetState sets the game state (useful for loading games)
func (e *Engine) SetState(state *GameState) {
	e.state = state
}

// GetEventBus returns the event bus for subscribing to events
func (e *Engine) GetEventBus() *events.EventBus {
	return e.eventBus
}

// SetupGame initializes the game with countries and merchants
func (e *Engine) SetupGame(countries []*Country, merchants []*Merchant) {
	e.state = NewGameState()
	for _, c := range countries {
		e.state.AddCountry(c)
	}
	for _, m := range merchants {
		e.state.AddMerchant(m)
	}
}

// RunPhase executes a single phase with the given actions
func (e *Engine) RunPhase(actions []Action) ([]events.Event, error) {
	phase, ok := e.phases[e.state.Phase]
	if !ok {
		// Skip phases without handlers (like Negotiation)
		e.state.NextPhase()
		return nil, nil
	}

	newState, evts := phase.Execute(e.state, actions)
	e.state = newState
	e.eventBus.Publish(evts...)

	// Advance to next phase
	e.state.NextPhase()

	return evts, nil
}

// RunTurn executes a complete turn with actions for each phase
func (e *Engine) RunTurn(actionsByPhase map[PhaseType][]Action) ([]events.Event, error) {
	var allEvents []events.Event

	// Emit turn start event
	turnStartEvt := events.NewBaseEvent(events.EventTurnStarted)
	turnStartEvt.Set("turn", e.state.Turn)
	e.eventBus.Publish(turnStartEvt)
	allEvents = append(allEvents, turnStartEvt)

	// Run through all phases
	startingPhase := e.state.Phase
	for {
		currentPhase := e.state.Phase

		// Get actions for this phase
		actions := actionsByPhase[currentPhase]

		// Run the phase
		phaseEvents, err := e.RunPhase(actions)
		if err != nil {
			return allEvents, err
		}
		allEvents = append(allEvents, phaseEvents...)

		// Check if we've completed a full turn (back to starting phase with higher turn number)
		if e.state.Phase == startingPhase {
			break
		}
	}

	// Emit turn end event
	turnEndEvt := events.NewBaseEvent(events.EventTurnEnded)
	turnEndEvt.Set("turn", e.state.Turn-1) // Turn was already incremented
	e.eventBus.Publish(turnEndEvt)
	allEvents = append(allEvents, turnEndEvt)

	return allEvents, nil
}

// IsGameOver checks if the game has ended (only one country remains)
func (e *Engine) IsGameOver() bool {
	aliveCount := 0
	for _, c := range e.state.Countries {
		if c.IsAlive() {
			aliveCount++
		}
	}
	return aliveCount <= 1
}

// GetWinner returns the winning country (if game is over)
func (e *Engine) GetWinner() *Country {
	if !e.IsGameOver() {
		return nil
	}
	for _, c := range e.state.Countries {
		if c.IsAlive() {
			return c
		}
	}
	return nil
}

// SubmitAction adds an action to the pending actions queue
func (e *Engine) SubmitAction(action interface{}) {
	e.pendingActions = append(e.pendingActions, action)
}

// GetPendingActions returns all pending actions
func (e *Engine) GetPendingActions() []interface{} {
	return e.pendingActions
}

// ClearPendingActions removes all pending actions
func (e *Engine) ClearPendingActions() {
	e.pendingActions = make([]interface{}, 0)
}
