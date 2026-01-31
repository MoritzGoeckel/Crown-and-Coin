package phases

import (
	"crown_and_coin/actions"
	"crown_and_coin/engine"
	"crown_and_coin/events"
)

// WarPhase handles Phase 4: War
type WarPhase struct {
	BasePhase
	dice engine.DiceRoller
}

func NewWarPhase(dice engine.DiceRoller) *WarPhase {
	return &WarPhase{
		BasePhase: BasePhase{
			name:      "War",
			phaseType: engine.PhaseWar,
		},
		dice: dice,
	}
}

func (p *WarPhase) ValidActions(state *engine.GameState, playerID string) []actions.Action {
	var validActions []actions.Action

	// Check if player is a monarch
	for _, country := range state.Countries {
		if country.MonarchID == playerID && !country.IsRepublic && country.IsAlive() {
			// Can choose not to attack
			validActions = append(validActions,
				actions.NewNoAttackAction(playerID, country.ID),
			)

			// Can attack any other alive country
			for _, target := range state.GetAliveCountries() {
				if target.ID != country.ID {
					validActions = append(validActions,
						actions.NewAttackAction(playerID, country.ID, target.ID),
					)
				}
			}
		}
	}

	return validActions
}

func (p *WarPhase) Execute(state *engine.GameState, playerActions []actions.Action) (*engine.GameState, []events.Event) {
	newState := state.Clone()
	var allEvents []events.Event

	// Process all attack actions
	for _, action := range playerActions {
		if err := action.Validate(newState); err != nil {
			continue // Skip invalid actions
		}

		var actionEvents []events.Event
		newState, actionEvents = action.Apply(newState, p.dice)
		allEvents = append(allEvents, actionEvents...)
	}

	// After all battles, halve all armies (maintenance cost)
	for _, country := range newState.Countries {
		if country.IsAlive() && country.ArmyStrength > 0 {
			oldStrength := country.ArmyStrength
			country.HalveArmy()
			allEvents = append(allEvents, events.NewArmyMaintenanceEvent(
				country.ID, oldStrength, country.ArmyStrength,
			))
		}
	}

	return newState, allEvents
}
