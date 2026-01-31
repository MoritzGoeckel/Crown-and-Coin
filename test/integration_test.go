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
	t.Log("=== SETUP ===")

	assertJSONResponse(t, api, `{
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
	}`, `{
		"success": true,
		"state": {
			"turn": 1,
			"phase": "taxation",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 0, "gold": 0, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 10, "army_strength": 0, "gold": 0, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 0, "invested_gold": 0},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 0, "invested_gold": 0},
				"diana": {"id": "diana", "country_id": "Britannia", "stored_gold": 0, "invested_gold": 0},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 0, "invested_gold": 0}
			}
		}
	}`)

	// ========================================
	// ROUND 1
	// ========================================
	t.Log("=== ROUND 1 ===")

	// ----------------------------------------
	// ROUND 1, PHASE 1: TAXATION
	// ----------------------------------------
	t.Log("--- Round 1, Phase 1: Taxation ---")

	// Verify monarch has tax options
	assertJSONResponse(t, api, `{"type": "get_actions", "player_id": "alice"}`, `{
		"success": true,
		"player_id": "alice",
		"phase": "taxation",
		"actions": [
			{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"},
			{"type": "tax_peasants_high", "player_id": "alice", "country_id": "Avalon"},
			{"type": "tax_merchants", "player_id": "alice", "country_id": "Avalon", "merchant_id": "charlie", "amount": "<AMOUNT:0-0>"},
			{"type": "tax_merchants", "player_id": "alice", "country_id": "Avalon", "merchant_id": "eve", "amount": "<AMOUNT:0-0>"}
		]
	}`)

	// Verify merchants have no actions in taxation phase
	assertJSONResponse(t, api, `{"type": "get_actions", "player_id": "charlie"}`, `{
		"success": true,
		"player_id": "charlie",
		"phase": "taxation",
		"actions": []
	}`)

	// Submit tax actions
	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_high", "player_id": "alice", "country_id": "Avalon"},
			{"type": "tax_peasants_low", "player_id": "bob", "country_id": "Britannia"}
		]
	}`, `{"success": true, "queued_actions": 2, "phase": "taxation"}`)

	// Advance - merchants get 5 gold income, taxes collected
	// With seed 12345, high tax does NOT trigger revolt (dice roll > 2)
	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "taxation",
		"current_phase": "negotiation",
		"turn": 1,
		"state": {
			"turn": 1,
			"phase": "negotiation",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 0, "gold": 10, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 10, "army_strength": 0, "gold": 5, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 0},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 0},
				"diana": {"id": "diana", "country_id": "Britannia", "stored_gold": 5, "invested_gold": 0},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 5, "invested_gold": 0}
			}
		}
	}`)

	// ----------------------------------------
	// ROUND 1, PHASE 2: NEGOTIATION (skip)
	// ----------------------------------------
	t.Log("--- Round 1, Phase 2: Negotiation ---")

	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "negotiation",
		"current_phase": "spending",
		"turn": 1,
		"state": {
			"turn": 1,
			"phase": "spending",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 0, "gold": 10, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 10, "army_strength": 0, "gold": 5, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 0},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 0},
				"diana": {"id": "diana", "country_id": "Britannia", "stored_gold": 5, "invested_gold": 0},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 5, "invested_gold": 0}
			}
		}
	}`)

	// ----------------------------------------
	// ROUND 1, PHASE 3: SPENDING
	// ----------------------------------------
	t.Log("--- Round 1, Phase 3: Spending ---")

	// Verify merchant spending options
	assertJSONResponse(t, api, `{"type": "get_actions", "player_id": "charlie"}`, `{
		"success": true,
		"player_id": "charlie",
		"phase": "spending",
		"actions": [
			{"type": "merchant_invest", "player_id": "charlie", "merchant_id": "charlie", "amount": "<AMOUNT:0-5>"},
			{"type": "merchant_hide", "player_id": "charlie", "merchant_id": "charlie"}
		]
	}`)

	// MGDO why does merchant_hide not have amount? Maybe add that or remove amount from invest?

	// Submit spending actions
	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "build_army", "player_id": "alice", "country_id": "Avalon", "amount": 8},
			{"type": "build_army", "player_id": "bob", "country_id": "Britannia", "amount": 3},
			{"type": "merchant_invest", "player_id": "charlie", "merchant_id": "charlie", "amount": 3},
			{"type": "merchant_hide", "player_id": "eve", "merchant_id": "eve"},
			{"type": "merchant_invest", "player_id": "diana", "merchant_id": "diana", "amount": 4},
			{"type": "merchant_invest", "player_id": "frank", "merchant_id": "frank", "amount": 2}
		]
	}`, `{"success": true, "queued_actions": 6, "phase": "spending"}`)

	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "spending",
		"current_phase": "war",
		"turn": 1,
		"state": {
			"turn": 1,
			"phase": "war",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 8, "gold": 2, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 10, "army_strength": 3, "gold": 2, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 2, "invested_gold": 3},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 0},
				"diana": {"id": "diana", "country_id": "Britannia", "stored_gold": 1, "invested_gold": 4},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 3, "invested_gold": 2}
			}
		}
	}`)

	// ----------------------------------------
	// ROUND 1, PHASE 4: WAR
	// ----------------------------------------
	t.Log("--- Round 1, Phase 4: War ---")

	// Alice attacks Britannia
	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "attack", "player_id": "alice", "country_id": "Avalon", "target_id": "Britannia"}
		]
	}`, `{"success": true, "queued_actions": 1, "phase": "war"}`)

	// Battle: Avalon (8) vs Britannia (3) -> Avalon wins, Britannia takes 5 damage
	// Avalon gets 5 gold victory bonus, armies halved
	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "war",
		"current_phase": "assessment",
		"turn": 1,
		"state": {
			"turn": 1,
			"phase": "assessment",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 4, "gold": 7, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 5, "army_strength": 1, "gold": 2, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 2, "invested_gold": 3},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 0},
				"diana": {"id": "diana", "country_id": "Britannia", "stored_gold": 1, "invested_gold": 4},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 3, "invested_gold": 2}
			}
		}
	}`)

	// ----------------------------------------
	// ROUND 1, PHASE 5: ASSESSMENT
	// ----------------------------------------
	t.Log("--- Round 1, Phase 5: Assessment ---")

	// Diana flees to Avalon, others remain
	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "remain", "player_id": "charlie", "merchant_id": "charlie"},
			{"type": "remain", "player_id": "eve", "merchant_id": "eve"},
			{"type": "flee", "player_id": "diana", "merchant_id": "diana", "target_id": "Avalon"},
			{"type": "remain", "player_id": "frank", "merchant_id": "frank"}
		]
	}`, `{"success": true, "queued_actions": 4, "phase": "assessment"}`)

	// Diana loses invested gold when fleeing
	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "assessment",
		"current_phase": "taxation",
		"turn": 2,
		"state": {
			"turn": 2,
			"phase": "taxation",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 4, "gold": 7, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 5, "army_strength": 1, "gold": 2, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 2, "invested_gold": 3},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 0},
				"diana": {"id": "diana", "country_id": "Avalon", "stored_gold": 1, "invested_gold": 0},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 3, "invested_gold": 2}
			}
		}
	}`)

	t.Log("=== ROUND 1 COMPLETE ===")

	// ========================================
	// ROUND 2
	// ========================================
	t.Log("=== ROUND 2 ===")

	// ----------------------------------------
	// ROUND 2, PHASE 1: TAXATION
	// ----------------------------------------
	t.Log("--- Round 2, Phase 1: Taxation ---")

	// Investments pay out: charlie 3->6, frank 2->4
	// All merchants get 5 income
	// Alice taxes charlie for 3 gold
	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"},
			{"type": "tax_peasants_low", "player_id": "bob", "country_id": "Britannia"},
			{"type": "tax_merchants", "player_id": "alice", "country_id": "Avalon", "merchant_id": "charlie", "amount": 3}
		]
	}`, `{"success": true, "queued_actions": 3, "phase": "taxation"}`)

	// Charlie: 2 stored + 6 payout + 5 income - 3 tax = 10
	// Eve: 5 stored + 5 income = 10
	// Diana: 1 stored + 5 income = 6
	// Frank: 3 stored + 4 payout + 5 income = 12
	// Avalon: 7 + 5 (tax) + 3 (merchant tax) = 15
	// Britannia: 2 + 5 (tax) = 7
	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "taxation",
		"current_phase": "negotiation",
		"turn": 2,
		"state": {
			"turn": 2,
			"phase": "negotiation",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 4, "gold": 15, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 5, "army_strength": 1, "gold": 7, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 10, "invested_gold": 0},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 10, "invested_gold": 0},
				"diana": {"id": "diana", "country_id": "Avalon", "stored_gold": 6, "invested_gold": 0},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 12, "invested_gold": 0}
			}
		}
	}`)

	// ----------------------------------------
	// ROUND 2, PHASE 2: NEGOTIATION (skip)
	// ----------------------------------------
	t.Log("--- Round 2, Phase 2: Negotiation ---")
	sendMessage(t, api, `{"type": "advance"}`)

	// ----------------------------------------
	// ROUND 2, PHASE 3: SPENDING
	// ----------------------------------------
	t.Log("--- Round 2, Phase 3: Spending ---")

	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "build_army", "player_id": "alice", "country_id": "Avalon", "amount": 12},
			{"type": "build_army", "player_id": "bob", "country_id": "Britannia", "amount": 7},
			{"type": "merchant_invest", "player_id": "charlie", "merchant_id": "charlie", "amount": 5},
			{"type": "merchant_invest", "player_id": "eve", "merchant_id": "eve", "amount": 5},
			{"type": "merchant_invest", "player_id": "diana", "merchant_id": "diana", "amount": 3},
			{"type": "merchant_invest", "player_id": "frank", "merchant_id": "frank", "amount": 6}
		]
	}`, `{"success": true, "queued_actions": 6, "phase": "spending"}`)

	// Avalon: 4 + 12 = 16 army, 15 - 12 = 3 gold
	// Britannia: 1 + 7 = 8 army, 7 - 7 = 0 gold
	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "spending",
		"current_phase": "war",
		"turn": 2,
		"state": {
			"turn": 2,
			"phase": "war",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 16, "gold": 3, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 5, "army_strength": 8, "gold": 0, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": false}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"diana": {"id": "diana", "country_id": "Avalon", "stored_gold": 3, "invested_gold": 3},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 6, "invested_gold": 6}
			}
		}
	}`)

	// ----------------------------------------
	// ROUND 2, PHASE 4: WAR
	// ----------------------------------------
	t.Log("--- Round 2, Phase 4: War ---")

	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "attack", "player_id": "alice", "country_id": "Avalon", "target_id": "Britannia"}
		]
	}`, `{"success": true, "queued_actions": 1, "phase": "war"}`)

	// Battle: Avalon (16) vs Britannia (8) -> Britannia takes 8 damage
	// Britannia: 5 HP - 8 = -3, revives to 1 HP (first death)
	// Avalon: 3 + 5 victory = 8 gold, army halved to 8
	// Britannia: army halved to 4
	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "war",
		"current_phase": "assessment",
		"turn": 2,
		"state": {
			"turn": 2,
			"phase": "assessment",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 8, "gold": 8, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": 1, "army_strength": 4, "gold": 0, "peasants": 1, "is_republic": false, "monarch_id": "bob", "died_once": true}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"diana": {"id": "diana", "country_id": "Avalon", "stored_gold": 3, "invested_gold": 3},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 6, "invested_gold": 6}
			}
		}
	}`)

	// ----------------------------------------
	// ROUND 2, PHASE 5: ASSESSMENT
	// ----------------------------------------
	t.Log("--- Round 2, Phase 5: Assessment ---")

	// Frank revolts! Frank (6 gold) > Bob (0 gold) -> success
	assertJSONResponse(t, api, `{
		"type": "submit",
		"actions": [
			{"type": "remain", "player_id": "charlie", "merchant_id": "charlie"},
			{"type": "remain", "player_id": "eve", "merchant_id": "eve"},
			{"type": "remain", "player_id": "diana", "merchant_id": "diana"},
			{"type": "revolt", "player_id": "frank", "merchant_id": "frank", "country_id": "Britannia"}
		]
	}`, `{"success": true, "queued_actions": 4, "phase": "assessment"}`)

	// Revolt succeeds: Britannia becomes republic, loses 2 HP (1 -> -1)
	assertJSONResponseWithEvents(t, api, `{"type": "advance"}`, `{
		"success": true,
		"previous_phase": "assessment",
		"current_phase": "taxation",
		"turn": 3,
		"state": {
			"turn": 3,
			"phase": "taxation",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 8, "gold": 8, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": -1, "army_strength": 4, "gold": 0, "peasants": 1, "is_republic": true, "monarch_id": "", "died_once": true}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"diana": {"id": "diana", "country_id": "Avalon", "stored_gold": 3, "invested_gold": 3},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 6, "invested_gold": 6}
			}
		}
	}`)

	t.Log("=== ROUND 2 COMPLETE ===")

	// Final state check
	t.Log("=== FINAL STATE ===")
	assertJSONResponse(t, api, `{"type": "get_state"}`, `{
		"success": true,
		"state": {
			"turn": 3,
			"phase": "taxation",
			"countries": {
				"Avalon": {"id": "Avalon", "hp": 10, "army_strength": 8, "gold": 8, "peasants": 1, "is_republic": false, "monarch_id": "alice", "died_once": false},
				"Britannia": {"id": "Britannia", "hp": -1, "army_strength": 4, "gold": 0, "peasants": 1, "is_republic": true, "monarch_id": "", "died_once": true}
			},
			"merchants": {
				"charlie": {"id": "charlie", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"eve": {"id": "eve", "country_id": "Avalon", "stored_gold": 5, "invested_gold": 5},
				"diana": {"id": "diana", "country_id": "Avalon", "stored_gold": 3, "invested_gold": 3},
				"frank": {"id": "frank", "country_id": "Britannia", "stored_gold": 6, "invested_gold": 6}
			}
		}
	}`)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func sendMessage(t *testing.T, api *jsonapi.GameAPI, request string) []byte {
	t.Helper()
	resp, err := api.ProcessMessage([]byte(request))
	if err != nil {
		t.Fatalf("ProcessMessage error: %v", err)
	}
	return resp
}

// assertJSONResponse compares the API response to expected JSON (ignoring formatting and events)
func assertJSONResponse(t *testing.T, api *jsonapi.GameAPI, request, expectedJSON string) {
	t.Helper()
	resp := sendMessage(t, api, request)

	if !jsonEqual(t, resp, expectedJSON) {
		t.Errorf("JSON mismatch\nRequest: %s\nExpected: %s\nActual: %s",
			normalizeJSON(t, request),
			normalizeJSON(t, expectedJSON),
			normalizeJSON(t, string(resp)))
	}
}

// assertJSONResponseWithEvents compares response but ignores the events field
func assertJSONResponseWithEvents(t *testing.T, api *jsonapi.GameAPI, request, expectedJSON string) {
	t.Helper()
	resp := sendMessage(t, api, request)

	// Parse both and remove events field before comparison
	var actual, expected map[string]interface{}
	if err := json.Unmarshal(resp, &actual); err != nil {
		t.Fatalf("Failed to parse actual response: %v", err)
	}
	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		t.Fatalf("Failed to parse expected JSON: %v", err)
	}

	// Remove events from both (we don't compare events in detail)
	delete(actual, "events")
	delete(expected, "events")

	actualBytes, _ := json.Marshal(actual)
	expectedBytes, _ := json.Marshal(expected)

	if !jsonEqual(t, actualBytes, string(expectedBytes)) {
		t.Errorf("JSON mismatch (excluding events)\nRequest: %s\nExpected: %s\nActual: %s",
			normalizeJSON(t, request),
			normalizeJSON(t, string(expectedBytes)),
			normalizeJSON(t, string(actualBytes)))
	}
}

// jsonEqual compares two JSON values for equality (ignoring formatting)
func jsonEqual(t *testing.T, actual []byte, expected string) bool {
	t.Helper()

	var actualObj, expectedObj interface{}
	if err := json.Unmarshal(actual, &actualObj); err != nil {
		t.Fatalf("Failed to parse actual JSON: %v\nJSON: %s", err, string(actual))
	}
	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("Failed to parse expected JSON: %v\nJSON: %s", err, expected)
	}

	actualNorm, _ := json.Marshal(actualObj)
	expectedNorm, _ := json.Marshal(expectedObj)

	return string(actualNorm) == string(expectedNorm)
}

// normalizeJSON formats JSON for readable error output
func normalizeJSON(t *testing.T, jsonStr string) string {
	t.Helper()
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return jsonStr // Return as-is if not valid JSON
	}
	formatted, _ := json.MarshalIndent(obj, "", "  ")
	return string(formatted)
}
