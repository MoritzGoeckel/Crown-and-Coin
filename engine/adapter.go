package engine

import (
	"crown_and_coin/events"
)

// PhaseAdapter wraps a phase implementation to work with the engine
type PhaseAdapter struct {
	name      string
	phaseType PhaseType
	executor  PhaseExecutor
}

// PhaseExecutor is the function signature for phase execution
type PhaseExecutor func(state *GameState, actions []Action) (*GameState, []events.Event)

// NewPhaseAdapter creates a new phase adapter
func NewPhaseAdapter(name string, phaseType PhaseType, executor PhaseExecutor) *PhaseAdapter {
	return &PhaseAdapter{
		name:      name,
		phaseType: phaseType,
		executor:  executor,
	}
}

func (p *PhaseAdapter) Name() string {
	return p.name
}

func (p *PhaseAdapter) Type() PhaseType {
	return p.phaseType
}

func (p *PhaseAdapter) Execute(state *GameState, actions []Action) (*GameState, []events.Event) {
	return p.executor(state, actions)
}
