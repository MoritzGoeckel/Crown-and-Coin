package phases

import (
	"crown_and_coin/actions"
	"crown_and_coin/engine"
	"crown_and_coin/events"
)

// TaxationPhase handles Phase 1: Taxation
type TaxationPhase struct {
	BasePhase
	dice engine.DiceRoller
}

func NewTaxationPhase(dice engine.DiceRoller) *TaxationPhase {
	return &TaxationPhase{
		BasePhase: BasePhase{
			name:      "Taxation",
			phaseType: engine.PhaseTaxation,
		},
		dice: dice,
	}
}

func (p *TaxationPhase) ValidActions(state *engine.GameState, playerID string) []actions.Action {
	var validActions []actions.Action

	// Check if player is a monarch
	for _, country := range state.Countries {
		if country.MonarchID == playerID && !country.IsRepublic && country.IsAlive() {
			// Monarch can choose low or high peasant tax
			validActions = append(validActions,
				actions.NewTaxPeasantsAction(playerID, country.ID, false), // Low tax
				actions.NewTaxPeasantsAction(playerID, country.ID, true),  // High tax
			)

			// Monarch can tax each merchant
			for _, merchant := range state.GetMerchantsByCountry(country.ID) {
				// Offer to tax any amount up to merchant's stored gold
				validActions = append(validActions,
					actions.NewTaxMerchantsAction(playerID, country.ID, merchant.ID, merchant.StoredGold),
				)
			}
		}
	}

	return validActions
}

func (p *TaxationPhase) Execute(state *engine.GameState, playerActions []actions.Action) (*engine.GameState, []events.Event) {
	newState := state.Clone()
	var allEvents []events.Event

	// Process monarch tax actions
	// Group actions by country to handle peasant revolt at end of phase
	revoltChecks := make(map[string]bool) // countryID -> had high tax

	for _, action := range playerActions {
		if err := action.Validate(newState); err != nil {
			continue // Skip invalid actions
		}

		var actionEvents []events.Event
		newState, actionEvents = action.Apply(newState, p.dice)
		allEvents = append(allEvents, actionEvents...)

		// Track if any peasant revolt events occurred
		for _, evt := range actionEvents {
			if evt.Type() == events.EventPeasantRevolt {
				revoltChecks[evt.Data()["country_id"].(string)] = true
			}
		}
	}

	return newState, allEvents
}
