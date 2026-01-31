package phases

import (
	"crown_and_coin/actions"
	"crown_and_coin/engine"
	"crown_and_coin/events"
)

// Phase defines the interface for a game phase
type Phase interface {
	// Name returns the display name of this phase
	Name() string

	// Type returns the phase type enum
	Type() engine.PhaseType

	// ValidActions returns all valid actions for a player in this phase
	ValidActions(state *engine.GameState, playerID string) []actions.Action

	// Execute runs the phase with the given actions and returns the new state and any events
	Execute(state *engine.GameState, playerActions []actions.Action) (*engine.GameState, []events.Event)
}

// BasePhase provides common functionality for phases
type BasePhase struct {
	name      string
	phaseType engine.PhaseType
}

func (p *BasePhase) Name() string {
	return p.name
}

func (p *BasePhase) Type() engine.PhaseType {
	return p.phaseType
}
