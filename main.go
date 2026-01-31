package main

import (
	"fmt"

	"crown_and_coin/engine"
	"crown_and_coin/events"
)

func main() {
	fmt.Println("=== Crown & Coin Game Engine Demo ===")
	fmt.Println()

	// Create a dice roller (seeded for reproducibility in demo)
	dice := engine.NewSeededDice(42)

	// Create the game engine
	gameEngine := engine.NewEngine(dice)

	// Subscribe to events for logging
	gameEngine.GetEventBus().Subscribe(func(e events.Event) {
		fmt.Printf("  [EVENT] %s\n", e)
	})

	// Setup initial game state
	// Create two countries with their monarchs
	countryA := engine.NewCountry("Avalon", "monarch_alice")
	countryB := engine.NewCountry("Britannia", "monarch_bob")

	// Create merchants
	merchant1 := engine.NewMerchant("merchant_charlie", "Avalon")
	merchant2 := engine.NewMerchant("merchant_diana", "Avalon")
	merchant3 := engine.NewMerchant("merchant_eve", "Britannia")

	gameEngine.SetupGame(
		[]*engine.Country{countryA, countryB},
		[]*engine.Merchant{merchant1, merchant2, merchant3},
	)

	// Print initial state
	printState(gameEngine.GetState())

	// Register phase handlers using the adapter pattern
	// For this demo, we'll create simplified phase executors
	gameEngine.RegisterPhase(engine.NewPhaseAdapter(
		"Taxation",
		engine.PhaseTaxation,
		taxationExecutor(dice),
	))

	gameEngine.RegisterPhase(engine.NewPhaseAdapter(
		"Spending",
		engine.PhaseSpending,
		spendingExecutor(dice),
	))

	gameEngine.RegisterPhase(engine.NewPhaseAdapter(
		"War",
		engine.PhaseWar,
		warExecutor(dice),
	))

	gameEngine.RegisterPhase(engine.NewPhaseAdapter(
		"Assessment",
		engine.PhaseAssessment,
		assessmentExecutor(dice),
	))

	// Simulate a few turns
	fmt.Println("\n=== Simulating Turn 1 ===")

	// For demo, we'll just run the phases with no player actions
	// In a real game, actions would come from players
	_, err := gameEngine.RunTurn(make(map[engine.PhaseType][]engine.Action))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print state after turn 1
	printState(gameEngine.GetState())

	fmt.Println("\n=== Demo Complete ===")
	fmt.Println("The engine is ready for integration with player input systems.")
}

// printState displays the current game state
func printState(state *engine.GameState) {
	fmt.Printf("\n--- Game State (Turn %d, Phase: %s) ---\n", state.Turn, state.Phase)

	fmt.Println("\nCountries:")
	for _, c := range state.Countries {
		status := "Monarchy"
		if c.IsRepublic {
			status = "Republic"
		}
		if !c.IsAlive() {
			status = "DEFEATED"
		}
		fmt.Printf("  %s: HP=%d, Army=%d, Gold=%d, Peasants=%d (%s)\n",
			c.ID, c.HP, c.ArmyStrength, c.Gold, c.Peasants, status)
	}

	fmt.Println("\nMerchants:")
	for _, m := range state.Merchants {
		fmt.Printf("  %s (in %s): Stored=%d, Invested=%d\n",
			m.ID, m.CountryID, m.StoredGold, m.InvestedGold)
	}
}

// Phase executors - these implement the game logic
// In a full implementation, these would use the phases package

func taxationExecutor(dice engine.DiceRoller) engine.PhaseExecutor {
	return func(state *engine.GameState, actions []engine.Action) (*engine.GameState, []events.Event) {
		newState := state.Clone()
		var evts []events.Event

		// Automatic: Pay out investments from previous turn
		for _, merchant := range newState.Merchants {
			if merchant.InvestedGold > 0 {
				payout := merchant.CollectInvestment()
				evts = append(evts, events.NewInvestmentPayoutEvent(merchant.ID, payout))
			}
		}

		// Automatic: All merchants receive 5 gold income
		for _, merchant := range newState.Merchants {
			merchant.ReceiveIncome(5)
			evts = append(evts, events.NewMerchantIncomeEvent(merchant.ID, 5))
		}

		// For demo: monarchs collect low tax (5 gold per peasant)
		for _, country := range newState.Countries {
			if country.IsAlive() && !country.IsRepublic {
				taxAmount := 5 * country.Peasants
				country.AddGold(taxAmount)
				evts = append(evts, events.NewPeasantTaxEvent(country.ID, taxAmount, false))
			}
		}

		return newState, evts
	}
}

func spendingExecutor(dice engine.DiceRoller) engine.PhaseExecutor {
	return func(state *engine.GameState, actions []engine.Action) (*engine.GameState, []events.Event) {
		newState := state.Clone()
		var evts []events.Event

		// For demo: countries build some army
		for _, country := range newState.Countries {
			if country.IsAlive() && country.Gold > 0 {
				armyBuild := country.Gold / 2 // Spend half on army
				if armyBuild > 0 {
					country.SpendGold(armyBuild)
					country.AddArmy(armyBuild)
					evts = append(evts, events.NewArmyBuiltEvent(country.ID, armyBuild, country.ArmyStrength))
				}
			}
		}

		// For demo: merchants invest half their gold
		for _, merchant := range newState.Merchants {
			if merchant.StoredGold > 0 {
				investAmount := merchant.StoredGold / 2
				if investAmount > 0 {
					merchant.Invest(investAmount)
					evt := events.NewBaseEvent(events.EventInvestmentMade)
					evt.Set("merchant_id", merchant.ID)
					evt.Set("amount", investAmount)
					evts = append(evts, evt)
				}
			}
		}

		return newState, evts
	}
}

func warExecutor(dice engine.DiceRoller) engine.PhaseExecutor {
	return func(state *engine.GameState, actions []engine.Action) (*engine.GameState, []events.Event) {
		newState := state.Clone()
		var evts []events.Event

		// For demo: no attacks, just maintenance
		for _, country := range newState.Countries {
			if country.IsAlive() && country.ArmyStrength > 0 {
				oldStrength := country.ArmyStrength
				country.HalveArmy()
				evts = append(evts, events.NewArmyMaintenanceEvent(country.ID, oldStrength, country.ArmyStrength))
			}
		}

		return newState, evts
	}
}

func assessmentExecutor(dice engine.DiceRoller) engine.PhaseExecutor {
	return func(state *engine.GameState, actions []engine.Action) (*engine.GameState, []events.Event) {
		// For demo: no merchant actions
		return state.Clone(), nil
	}
}
