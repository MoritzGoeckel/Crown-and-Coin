package test

import (
	"encoding/json"
	"testing"

	"crown_and_coin/engine"
	"crown_and_coin/jsonapi"
)

// TestRealisticGameScenario runs through a full 2-round game with 2 countries
// Each country has 1 monarch and 2 merchants (6 players total)
func TestRealisticGameScenario(t *testing.T) {
	// Use seeded dice for deterministic results
	api := jsonapi.NewGameAPIWithDice(engine.NewSeededDice(12345))

	// ========================================
	// SETUP: 2 countries, 2 merchants each
	// ========================================
	// Avalon: King Alice, merchants Charlie and Eve
	// Britannia: King Bob, merchants Diana and Frank

	setupResp := sendJSON[jsonapi.SetupResponse](t, api, `{
		"type": "setup",
		"countries": [
			{"id": "Avalon", "monarch_id": "alice"},
			{"id": "Britannia", "monarch_id": "bob"}
		],
		"merchants": [
			{"id": "charlie", "country_id": "Avalon"},
			{"id": "eve", "country_id": "Avalon"},
			{"id": "diana", "country_id": "Britannia"},
			{"id": "frank", "country_id": "Britannia"}
		]
	}`)

	assertSuccess(t, setupResp.Success, "Setup")
	assertEqual(t, "turn", 1, setupResp.State.Turn)
	assertEqual(t, "phase", "taxation", setupResp.State.Phase)
	assertEqual(t, "country count", 2, len(setupResp.State.Countries))
	assertEqual(t, "merchant count", 4, len(setupResp.State.Merchants))

	// Verify initial country state
	assertCountryState(t, setupResp.State, "Avalon", 10, 0, 0, 1)
	assertCountryState(t, setupResp.State, "Britannia", 10, 0, 0, 1)

	// Verify initial merchant state
	assertMerchantState(t, setupResp.State, "charlie", "Avalon", 0, 0)
	assertMerchantState(t, setupResp.State, "eve", "Avalon", 0, 0)
	assertMerchantState(t, setupResp.State, "diana", "Britannia", 0, 0)
	assertMerchantState(t, setupResp.State, "frank", "Britannia", 0, 0)

	// ========================================
	// ROUND 1
	// ========================================
	t.Log("=== ROUND 1 ===")

	// ----------------------------------------
	// ROUND 1, PHASE 1: TAXATION
	// ----------------------------------------
	t.Log("--- Round 1, Phase 1: Taxation ---")

	// Verify monarchs have tax options
	aliceActions := sendJSON[jsonapi.ActionsResponse](t, api, `{"type": "get_actions", "player_id": "alice"}`)
	assertSuccess(t, aliceActions.Success, "GetActions alice")
	assertEqual(t, "alice phase", "taxation", aliceActions.Phase)
	assertContainsActionType(t, aliceActions.Actions, "tax_peasants_low")
	assertContainsActionType(t, aliceActions.Actions, "tax_peasants_high")
	assertContainsActionType(t, aliceActions.Actions, "tax_merchants")

	// Verify merchants have no actions in taxation phase
	charlieActions := sendJSON[jsonapi.ActionsResponse](t, api, `{"type": "get_actions", "player_id": "charlie"}`)
	assertEqual(t, "charlie actions in taxation", 0, len(charlieActions.Actions))

	// Alice (Avalon) chooses HIGH tax (10 gold, risk of revolt)
	// Bob (Britannia) chooses LOW tax (5 gold, safe)
	// Alice also taxes merchant Charlie for 2 gold
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_high", "player_id": "alice", "country_id": "Avalon"},
			{"type": "tax_peasants_low", "player_id": "bob", "country_id": "Britannia"}
		]
	}`)

	taxationResp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)
	assertSuccess(t, taxationResp.Success, "Advance taxation")
	assertEqual(t, "previous phase", "taxation", taxationResp.PreviousPhase)
	assertEqual(t, "current phase", "negotiation", taxationResp.CurrentPhase)

	// After taxation:
	// - All merchants received 5 gold income
	// - Avalon: 10 gold (high tax, 1 peasant * 10) - may have revolt damage
	// - Britannia: 5 gold (low tax, 1 peasant * 5)
	assertMerchantState(t, taxationResp.State, "charlie", "Avalon", 5, 0)
	assertMerchantState(t, taxationResp.State, "eve", "Avalon", 5, 0)
	assertMerchantState(t, taxationResp.State, "diana", "Britannia", 5, 0)
	assertMerchantState(t, taxationResp.State, "frank", "Britannia", 5, 0)

	// Avalon got 10 gold from high tax
	avalonGold := taxationResp.State.Countries["Avalon"].Gold
	assertEqual(t, "Avalon gold after high tax", 10, avalonGold)

	// Britannia got 5 gold from low tax
	assertEqual(t, "Britannia gold", 5, taxationResp.State.Countries["Britannia"].Gold)

	// Check if Avalon had a peasant revolt (2/6 chance with seeded dice)
	avalonHP := taxationResp.State.Countries["Avalon"].HP
	t.Logf("Avalon HP after high tax: %d (revolt if < 10)", avalonHP)

	// ----------------------------------------
	// ROUND 1, PHASE 2: NEGOTIATION (skip)
	// ----------------------------------------
	t.Log("--- Round 1, Phase 2: Negotiation ---")

	negotiationResp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)
	assertEqual(t, "current phase after negotiation", "spending", negotiationResp.CurrentPhase)

	// ----------------------------------------
	// ROUND 1, PHASE 3: SPENDING
	// ----------------------------------------
	t.Log("--- Round 1, Phase 3: Spending ---")

	// Verify monarch spending options
	aliceSpendActions := sendJSON[jsonapi.ActionsResponse](t, api, `{"type": "get_actions", "player_id": "alice"}`)
	assertEqual(t, "alice phase", "spending", aliceSpendActions.Phase)
	assertContainsActionType(t, aliceSpendActions.Actions, "build_army")

	// Verify merchant spending options
	charlieSpendActions := sendJSON[jsonapi.ActionsResponse](t, api, `{"type": "get_actions", "player_id": "charlie"}`)
	assertContainsActionType(t, charlieSpendActions.Actions, "merchant_invest")
	assertContainsActionType(t, charlieSpendActions.Actions, "merchant_hide")

	// Actions:
	// Alice: Build 8 army (spend 8 gold, keep 2)
	// Bob: Build 3 army (spend 3 gold, keep 2)
	// Charlie: Invest 3 gold
	// Eve: Hide (keep savings)
	// Diana: Invest 4 gold
	// Frank: Invest 2 gold
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "build_army", "player_id": "alice", "country_id": "Avalon", "amount": 8},
			{"type": "build_army", "player_id": "bob", "country_id": "Britannia", "amount": 3},
			{"type": "merchant_invest", "player_id": "charlie", "merchant_id": "charlie", "amount": 3},
			{"type": "merchant_hide", "player_id": "eve", "merchant_id": "eve"},
			{"type": "merchant_invest", "player_id": "diana", "merchant_id": "diana", "amount": 4},
			{"type": "merchant_invest", "player_id": "frank", "merchant_id": "frank", "amount": 2}
		]
	}`)

	spendingResp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)
	assertEqual(t, "current phase after spending", "war", spendingResp.CurrentPhase)

	// Verify spending results
	assertEqual(t, "Avalon army", 8, spendingResp.State.Countries["Avalon"].ArmyStrength)
	assertEqual(t, "Avalon gold after spending", 2, spendingResp.State.Countries["Avalon"].Gold)
	assertEqual(t, "Britannia army", 3, spendingResp.State.Countries["Britannia"].ArmyStrength)
	assertEqual(t, "Britannia gold after spending", 2, spendingResp.State.Countries["Britannia"].Gold)

	// Verify merchant investments
	assertMerchantState(t, spendingResp.State, "charlie", "Avalon", 2, 3) // 5-3=2 stored, 3 invested
	assertMerchantState(t, spendingResp.State, "eve", "Avalon", 5, 0)     // kept all
	assertMerchantState(t, spendingResp.State, "diana", "Britannia", 1, 4) // 5-4=1 stored, 4 invested
	assertMerchantState(t, spendingResp.State, "frank", "Britannia", 3, 2) // 5-2=3 stored, 2 invested

	// ----------------------------------------
	// ROUND 1, PHASE 4: WAR
	// ----------------------------------------
	t.Log("--- Round 1, Phase 4: War ---")

	// Verify war options
	aliceWarActions := sendJSON[jsonapi.ActionsResponse](t, api, `{"type": "get_actions", "player_id": "alice"}`)
	assertEqual(t, "alice phase", "war", aliceWarActions.Phase)
	assertContainsActionType(t, aliceWarActions.Actions, "attack")
	assertContainsActionType(t, aliceWarActions.Actions, "no_attack")

	// Alice attacks Britannia (8 vs 3 army)
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "attack", "player_id": "alice", "country_id": "Avalon", "target_id": "Britannia"}
		]
	}`)

	warResp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)
	assertEqual(t, "current phase after war", "assessment", warResp.CurrentPhase)

	// Battle result: Avalon (8) vs Britannia (3)
	// Avalon wins, Britannia takes 5 damage (8-3)
	// Avalon gets 5 gold victory bonus
	// Armies are halved after war: 8->4, 3->1
	assertEqual(t, "Britannia HP after battle", 5, warResp.State.Countries["Britannia"].HP) // 10 - 5 = 5
	assertEqual(t, "Avalon gold after victory", 7, warResp.State.Countries["Avalon"].Gold)  // 2 + 5 = 7
	assertEqual(t, "Avalon army after maintenance", 4, warResp.State.Countries["Avalon"].ArmyStrength)
	assertEqual(t, "Britannia army after maintenance", 1, warResp.State.Countries["Britannia"].ArmyStrength)

	// ----------------------------------------
	// ROUND 1, PHASE 5: ASSESSMENT
	// ----------------------------------------
	t.Log("--- Round 1, Phase 5: Assessment ---")

	// Verify merchant assessment options
	charlieAssessActions := sendJSON[jsonapi.ActionsResponse](t, api, `{"type": "get_actions", "player_id": "charlie"}`)
	assertEqual(t, "charlie phase", "assessment", charlieAssessActions.Phase)
	assertContainsActionType(t, charlieAssessActions.Actions, "remain")
	assertContainsActionType(t, charlieAssessActions.Actions, "flee")
	assertContainsActionType(t, charlieAssessActions.Actions, "revolt")

	// Diana considers fleeing to Avalon (stronger country)
	// All others remain
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "remain", "player_id": "charlie", "merchant_id": "charlie"},
			{"type": "remain", "player_id": "eve", "merchant_id": "eve"},
			{"type": "flee", "player_id": "diana", "merchant_id": "diana", "target_id": "Avalon"},
			{"type": "remain", "player_id": "frank", "merchant_id": "frank"}
		]
	}`)

	assessmentResp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)
	assertEqual(t, "turn after round 1", 2, assessmentResp.Turn)
	assertEqual(t, "phase after round 1", "taxation", assessmentResp.CurrentPhase)

	// Diana fled to Avalon - loses invested gold
	assertMerchantState(t, assessmentResp.State, "diana", "Avalon", 1, 0) // kept stored, lost invested

	t.Log("=== ROUND 1 COMPLETE ===")

	// ========================================
	// ROUND 2
	// ========================================
	t.Log("=== ROUND 2 ===")

	// ----------------------------------------
	// ROUND 2, PHASE 1: TAXATION
	// ----------------------------------------
	t.Log("--- Round 2, Phase 1: Taxation ---")

	// Investments from round 1 should pay out (double)
	// Charlie: 3 invested -> 6 payout
	// Frank: 2 invested -> 4 payout
	// (Diana lost investment by fleeing, Eve didn't invest)

	// Both monarchs choose low tax this round
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"},
			{"type": "tax_peasants_low", "player_id": "bob", "country_id": "Britannia"},
			{"type": "tax_merchants", "player_id": "alice", "country_id": "Avalon", "merchant_id": "charlie", "amount": 3}
		]
	}`)

	taxation2Resp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)

	// Charlie: had 2 stored + 3 invested
	// -> investment payout: 2 + 6 = 8
	// -> merchant income: 8 + 5 = 13
	// -> taxed 3: 13 - 3 = 10
	assertMerchantState(t, taxation2Resp.State, "charlie", "Avalon", 10, 0)

	// Eve: had 5 stored, 0 invested
	// -> income: 5 + 5 = 10
	assertMerchantState(t, taxation2Resp.State, "eve", "Avalon", 10, 0)

	// Diana: had 1 stored (fled, lost invested)
	// -> income: 1 + 5 = 6
	assertMerchantState(t, taxation2Resp.State, "diana", "Avalon", 6, 0)

	// Frank: had 3 stored + 2 invested
	// -> investment payout: 3 + 4 = 7
	// -> income: 7 + 5 = 12
	assertMerchantState(t, taxation2Resp.State, "frank", "Britannia", 12, 0)

	// Avalon: had 7 gold + 5 (low tax) + 3 (merchant tax) = 15
	assertEqual(t, "Avalon gold round 2", 15, taxation2Resp.State.Countries["Avalon"].Gold)

	// Britannia: had 2 gold + 5 (low tax) = 7
	assertEqual(t, "Britannia gold round 2", 7, taxation2Resp.State.Countries["Britannia"].Gold)

	// ----------------------------------------
	// ROUND 2, PHASE 2: NEGOTIATION (skip)
	// ----------------------------------------
	t.Log("--- Round 2, Phase 2: Negotiation ---")
	sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)

	// ----------------------------------------
	// ROUND 2, PHASE 3: SPENDING
	// ----------------------------------------
	t.Log("--- Round 2, Phase 3: Spending ---")

	// Alice builds big army for final attack
	// Bob builds defense
	// Merchants invest for future
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "build_army", "player_id": "alice", "country_id": "Avalon", "amount": 12},
			{"type": "build_army", "player_id": "bob", "country_id": "Britannia", "amount": 7},
			{"type": "merchant_invest", "player_id": "charlie", "merchant_id": "charlie", "amount": 5},
			{"type": "merchant_invest", "player_id": "eve", "merchant_id": "eve", "amount": 5},
			{"type": "merchant_invest", "player_id": "diana", "merchant_id": "diana", "amount": 3},
			{"type": "merchant_invest", "player_id": "frank", "merchant_id": "frank", "amount": 6}
		]
	}`)

	spending2Resp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)

	// Avalon: 4 (from last round) + 12 = 16 army, 15 - 12 = 3 gold
	assertEqual(t, "Avalon army round 2", 16, spending2Resp.State.Countries["Avalon"].ArmyStrength)
	assertEqual(t, "Avalon gold after spending round 2", 3, spending2Resp.State.Countries["Avalon"].Gold)

	// Britannia: 1 (from last round) + 7 = 8 army, 7 - 7 = 0 gold
	assertEqual(t, "Britannia army round 2", 8, spending2Resp.State.Countries["Britannia"].ArmyStrength)
	assertEqual(t, "Britannia gold after spending round 2", 0, spending2Resp.State.Countries["Britannia"].Gold)

	// ----------------------------------------
	// ROUND 2, PHASE 4: WAR
	// ----------------------------------------
	t.Log("--- Round 2, Phase 4: War ---")

	// Alice attacks again to try to defeat Britannia
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "attack", "player_id": "alice", "country_id": "Avalon", "target_id": "Britannia"}
		]
	}`)

	war2Resp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)

	// Battle: Avalon (16) vs Britannia (8)
	// Avalon wins, Britannia takes 8 damage
	// Britannia had 5 HP, takes 8 damage -> would die, but has "died once" revival
	// Actually Britannia hasn't died yet, so first death revives to 1 HP
	britanniaHP := war2Resp.State.Countries["Britannia"].HP
	t.Logf("Britannia HP after second battle: %d", britanniaHP)

	// Britannia takes 8 damage (16-8), had 5 HP -> 5-8 = -3 -> revives to 1 (first death)
	assertEqual(t, "Britannia HP after defeat", 1, britanniaHP)
	assertEqual(t, "Britannia died_once flag", true, war2Resp.State.Countries["Britannia"].DiedOnce)

	// Avalon gets victory bonus
	assertEqual(t, "Avalon gold after victory", 8, war2Resp.State.Countries["Avalon"].Gold) // 3 + 5 = 8

	// Armies halved
	assertEqual(t, "Avalon army after maintenance", 8, war2Resp.State.Countries["Avalon"].ArmyStrength)  // 16 / 2
	assertEqual(t, "Britannia army after maintenance", 4, war2Resp.State.Countries["Britannia"].ArmyStrength) // 8 / 2

	// ----------------------------------------
	// ROUND 2, PHASE 5: ASSESSMENT
	// ----------------------------------------
	t.Log("--- Round 2, Phase 5: Assessment ---")

	// Frank considers revolt (Britannia is weak)
	// But Bob (monarch) has 0 gold, Frank has 6 stored gold
	// Revolt requirement: merchants must have MORE gold than monarch
	// Frank has 6 > 0, so revolt would succeed!

	// Check Frank's options
	frankAssessActions := sendJSON[jsonapi.ActionsResponse](t, api, `{"type": "get_actions", "player_id": "frank"}`)
	assertContainsActionType(t, frankAssessActions.Actions, "revolt")

	// Frank revolts!
	sendJSON[jsonapi.SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "remain", "player_id": "charlie", "merchant_id": "charlie"},
			{"type": "remain", "player_id": "eve", "merchant_id": "eve"},
			{"type": "remain", "player_id": "diana", "merchant_id": "diana"},
			{"type": "revolt", "player_id": "frank", "merchant_id": "frank", "country_id": "Britannia"}
		]
	}`)

	assessment2Resp := sendJSON[jsonapi.AdvanceResponse](t, api, `{"type": "advance"}`)

	// Revolt succeeds: Frank (6 gold) > Bob (0 gold)
	// Britannia loses 2 HP (1 -> -1, but already died once, so truly dead now...
	// Wait, the revival is already used. Let me check the logic.)
	// Actually: Britannia HP was 1, takes 2 damage from revolt -> -1
	// DiedOnce is true, so no revival -> HP stays at -1 (dead)

	// Actually let me re-check: revolt damage is 2 HP
	// Britannia had 1 HP, DiedOnce=true
	// After 2 damage: 1 - 2 = -1, and since DiedOnce is true, no revival

	// Hmm, but looking at the game rules, a successful revolt also makes it a republic
	// Let me verify the state

	assertEqual(t, "turn after round 2", 3, assessment2Resp.Turn)

	// Check if Britannia is now a republic (revolt succeeded)
	britannia := assessment2Resp.State.Countries["Britannia"]
	t.Logf("Britannia after revolt: HP=%d, IsRepublic=%v, DiedOnce=%v",
		britannia.HP, britannia.IsRepublic, britannia.DiedOnce)

	// The revolt should succeed (Frank 6 gold > Bob 0 gold)
	// This makes Britannia a republic and costs 2 HP
	assertEqual(t, "Britannia is republic after revolt", true, britannia.IsRepublic)

	t.Log("=== ROUND 2 COMPLETE ===")

	// Final state summary
	t.Log("=== FINAL STATE ===")
	finalState := sendJSON[jsonapi.StateResponse](t, api, `{"type": "get_state"}`)

	t.Logf("Turn: %d, Phase: %s", finalState.State.Turn, finalState.State.Phase)
	for id, c := range finalState.State.Countries {
		t.Logf("Country %s: HP=%d, Army=%d, Gold=%d, Peasants=%d, Republic=%v",
			id, c.HP, c.ArmyStrength, c.Gold, c.Peasants, c.IsRepublic)
	}
	for id, m := range finalState.State.Merchants {
		t.Logf("Merchant %s: Country=%s, Stored=%d, Invested=%d",
			id, m.CountryID, m.StoredGold, m.InvestedGold)
	}
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func sendJSON[T any](t *testing.T, api *jsonapi.GameAPI, msg string) T {
	t.Helper()
	respBytes, err := api.ProcessMessage([]byte(msg))
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}

	var resp T
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v\nResponse: %s", err, string(respBytes))
	}
	return resp
}

func assertSuccess(t *testing.T, success bool, context string) {
	t.Helper()
	if !success {
		t.Fatalf("%s: expected success=true", context)
	}
}

func assertEqual[T comparable](t *testing.T, name string, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", name, expected, actual)
	}
}

func assertCountryState(t *testing.T, state *jsonapi.StateJSON, countryID string, hp, army, gold, peasants int) {
	t.Helper()
	country := state.Countries[countryID]
	if country == nil {
		t.Fatalf("Country %s not found", countryID)
	}
	if country.HP != hp {
		t.Errorf("Country %s HP: expected %d, got %d", countryID, hp, country.HP)
	}
	if country.ArmyStrength != army {
		t.Errorf("Country %s Army: expected %d, got %d", countryID, army, country.ArmyStrength)
	}
	if country.Gold != gold {
		t.Errorf("Country %s Gold: expected %d, got %d", countryID, gold, country.Gold)
	}
	if country.Peasants != peasants {
		t.Errorf("Country %s Peasants: expected %d, got %d", countryID, peasants, country.Peasants)
	}
}

func assertMerchantState(t *testing.T, state *jsonapi.StateJSON, merchantID, countryID string, stored, invested int) {
	t.Helper()
	merchant := state.Merchants[merchantID]
	if merchant == nil {
		t.Fatalf("Merchant %s not found", merchantID)
	}
	if merchant.CountryID != countryID {
		t.Errorf("Merchant %s Country: expected %s, got %s", merchantID, countryID, merchant.CountryID)
	}
	if merchant.StoredGold != stored {
		t.Errorf("Merchant %s StoredGold: expected %d, got %d", merchantID, stored, merchant.StoredGold)
	}
	if merchant.InvestedGold != invested {
		t.Errorf("Merchant %s InvestedGold: expected %d, got %d", merchantID, invested, merchant.InvestedGold)
	}
}

func assertContainsActionType(t *testing.T, actions []jsonapi.ActionJSON, actionType string) {
	t.Helper()
	for _, a := range actions {
		if a.Type == actionType {
			return
		}
	}
	t.Errorf("Expected action type %s not found in actions", actionType)
}
