package jsonapi

import (
	"encoding/json"
	"testing"

	"crown_and_coin/engine"
)

// Helper to send a JSON message and get a typed response
func sendMessage[T any](t *testing.T, api *GameAPI, msg string) T {
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

// setupGame is a helper that adds countries and merchants for tests
func setupGame(t *testing.T, api *GameAPI) {
	t.Helper()
	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Britannia", "monarch_id": "bob"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "diana", "country_id": "Britannia"}`)
}

func TestAddCountry(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	resp := sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	if !resp.Success {
		t.Fatal("AddCountry should succeed")
	}

	// Duplicate should fail
	resp = sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "bob"}`)
	if resp.Success {
		t.Fatal("Duplicate country_id should fail")
	}
}

func TestAddMerchant(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)

	resp := sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)
	if !resp.Success {
		t.Fatal("AddMerchant should succeed")
	}

	// Duplicate should fail
	resp = sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)
	if resp.Success {
		t.Fatal("Duplicate player_id should fail")
	}
}

func TestRemoveMerchant(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	resp := sendMessage[RemoveMerchantResponse](t, api, `{"type": "remove_merchant", "player_id": "charlie"}`)
	if !resp.Success {
		t.Fatal("RemoveMerchant should succeed")
	}

	// Removing again should fail
	resp = sendMessage[RemoveMerchantResponse](t, api, `{"type": "remove_merchant", "player_id": "charlie"}`)
	if resp.Success {
		t.Fatal("Removing non-existent merchant should fail")
	}
}

func TestGetPlayers(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	resp := sendMessage[GetPlayersResponse](t, api, `{"type": "get_players"}`)
	if !resp.Success {
		t.Fatal("GetPlayers should succeed")
	}
	if len(resp.Players) != 2 {
		t.Errorf("Expected 2 players, got %d", len(resp.Players))
	}

	alice := resp.Players["alice"]
	if alice == nil {
		t.Fatal("Player alice not found")
	}
	if alice.Role != "monarch" {
		t.Errorf("Expected alice role 'monarch', got '%s'", alice.Role)
	}
	if alice.CountryID != "Avalon" {
		t.Errorf("Expected alice country 'Avalon', got '%s'", alice.CountryID)
	}

	charlie := resp.Players["charlie"]
	if charlie == nil {
		t.Fatal("Player charlie not found")
	}
	if charlie.Role != "merchant" {
		t.Errorf("Expected charlie role 'merchant', got '%s'", charlie.Role)
	}
}

func TestGetState(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	resp := sendMessage[StateResponse](t, api, `{"type": "get_state"}`)

	if !resp.Success {
		t.Fatal("GetState should succeed")
	}
	if resp.State.Turn != 1 {
		t.Errorf("Expected turn 1, got %d", resp.State.Turn)
	}

	country := resp.State.Countries["Avalon"]
	if country == nil {
		t.Fatal("Country Avalon not found")
	}
	if country.HP != 10 {
		t.Errorf("Expected HP 10, got %d", country.HP)
	}
}

func TestGetActions(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	// Get actions for monarch
	resp := sendMessage[ActionsResponse](t, api, `{"type": "get_actions", "player_id": "alice"}`)

	if !resp.Success {
		t.Fatal("GetActions should succeed")
	}
	if resp.PlayerID != "alice" {
		t.Errorf("Expected player_id 'alice', got '%s'", resp.PlayerID)
	}
	if resp.Phase != "taxation" {
		t.Errorf("Expected phase 'taxation', got '%s'", resp.Phase)
	}

	// Should have tax options
	if len(resp.Actions) == 0 {
		t.Error("Expected some actions for monarch")
	}

	// Check that actions are executable templates
	foundTaxLow := false
	foundTaxHigh := false
	for _, action := range resp.Actions {
		if action.Type == "tax_peasants_low" {
			foundTaxLow = true
			if action.PlayerID != "alice" {
				t.Errorf("Action should have player_id 'alice', got '%s'", action.PlayerID)
			}
		}
		if action.Type == "tax_peasants_high" {
			foundTaxHigh = true
		}
	}
	if !foundTaxLow {
		t.Error("Expected tax_peasants_low action")
	}
	if !foundTaxHigh {
		t.Error("Expected tax_peasants_high action")
	}
}

func TestSubmitActions(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	// Submit a tax action
	submitMsg := `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"}
		]
	}`
	resp := sendMessage[SubmitResponse](t, api, submitMsg)

	if !resp.Success {
		t.Fatal("Submit should succeed")
	}
	if resp.QueuedActions != 1 {
		t.Errorf("Expected 1 queued action, got %d", resp.QueuedActions)
	}
}

func TestGetQueued(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	// Submit action
	submitMsg := `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"}
		]
	}`
	sendMessage[SubmitResponse](t, api, submitMsg)

	// Get queued
	resp := sendMessage[QueuedResponse](t, api, `{"type": "get_queued"}`)

	if !resp.Success {
		t.Fatal("GetQueued should succeed")
	}
	if len(resp.Actions) != 1 {
		t.Errorf("Expected 1 queued action, got %d", len(resp.Actions))
	}
	if resp.Actions[0].Type != "tax_peasants_low" {
		t.Errorf("Expected action type 'tax_peasants_low', got '%s'", resp.Actions[0].Type)
	}
}

func TestAdvance(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	// Submit tax action
	submitMsg := `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"}
		]
	}`
	sendMessage[SubmitResponse](t, api, submitMsg)

	// Advance to next phase
	resp := sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`)

	if !resp.Success {
		t.Fatal("Advance should succeed")
	}
	if resp.PreviousPhase != "taxation" {
		t.Errorf("Expected previous phase 'taxation', got '%s'", resp.PreviousPhase)
	}
	if resp.CurrentPhase != "negotiation" {
		t.Errorf("Expected current phase 'negotiation', got '%s'", resp.CurrentPhase)
	}

	// Check that events were generated
	if len(resp.Events) == 0 {
		t.Error("Expected some events from taxation phase")
	}

	// Check that gold was collected (5 per peasant for low tax)
	country := resp.State.Countries["Avalon"]
	if country.Gold != 5 {
		t.Errorf("Expected country gold to be 5 after low tax, got %d", country.Gold)
	}

	// Check that merchant got income
	merchant := resp.State.Merchants["charlie"]
	if merchant.StoredGold != 5 {
		t.Errorf("Expected merchant stored gold to be 5 after income, got %d", merchant.StoredGold)
	}
}

func TestFullTurnSimulation(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	setupGame(t, api)

	// Phase 1: Taxation
	// Both monarchs collect low tax
	sendMessage[SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"},
			{"type": "tax_peasants_low", "player_id": "bob", "country_id": "Britannia"}
		]
	}`)
	sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`)

	// Phase 2: Negotiation (skip)
	sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`)

	// Phase 3: Spending
	// Alice builds army, Bob saves
	sendMessage[SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "build_army", "player_id": "alice", "country_id": "Avalon", "amount": 5}
		]
	}`)
	sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`)

	// Phase 4: War
	// No attacks
	sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`)

	// Phase 5: Assessment
	// Merchants remain
	sendMessage[SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [
			{"type": "remain", "player_id": "charlie", "merchant_id": "charlie"},
			{"type": "remain", "player_id": "diana", "merchant_id": "diana"}
		]
	}`)
	advResp := sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`)

	// Should be turn 2 now
	if advResp.Turn != 2 {
		t.Errorf("Expected turn 2 after full cycle, got %d", advResp.Turn)
	}
	if advResp.CurrentPhase != "taxation" {
		t.Errorf("Expected to be back at taxation phase, got '%s'", advResp.CurrentPhase)
	}

	// Verify state
	avalon := advResp.State.Countries["Avalon"]
	if avalon.ArmyStrength != 2 { // 5 built, then halved = 2
		t.Errorf("Expected Avalon army strength 2 (5 halved), got %d", avalon.ArmyStrength)
	}
}

func TestMerchantActions(t *testing.T) {
	api := NewGameAPIWithDice(engine.NewSeededDice(42))

	sendMessage[AddCountryResponse](t, api, `{"type": "add_country", "country_id": "Avalon", "monarch_id": "alice"}`)
	sendMessage[AddMerchantResponse](t, api, `{"type": "add_merchant", "player_id": "charlie", "country_id": "Avalon"}`)

	// Advance through taxation (merchant gets 5 gold income)
	sendMessage[SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [{"type": "tax_peasants_low", "player_id": "alice", "country_id": "Avalon"}]
	}`)
	sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`) // taxation -> negotiation
	sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`) // negotiation -> spending

	// Get merchant actions in spending phase
	resp := sendMessage[ActionsResponse](t, api, `{"type": "get_actions", "player_id": "charlie"}`)

	if len(resp.Actions) == 0 {
		t.Fatal("Merchant should have actions in spending phase")
	}

	// Should have invest and hide options
	foundInvest := false
	foundHide := false
	for _, action := range resp.Actions {
		if action.Type == "merchant_invest" {
			foundInvest = true
			// Check for amount placeholder
			if amountStr, ok := action.Amount.(string); ok {
				if amountStr == "" {
					t.Error("Invest action should have amount placeholder")
				}
			}
		}
		if action.Type == "merchant_hide" {
			foundHide = true
		}
	}
	if !foundInvest {
		t.Error("Expected merchant_invest action")
	}
	if !foundHide {
		t.Error("Expected merchant_hide action")
	}

	// Submit invest action
	sendMessage[SubmitResponse](t, api, `{
		"type": "submit",
		"actions": [{"type": "merchant_invest", "player_id": "charlie", "merchant_id": "charlie", "amount": 3}]
	}`)
	advResp := sendMessage[AdvanceResponse](t, api, `{"type": "advance"}`)

	// Verify merchant invested
	merchant := advResp.State.Merchants["charlie"]
	if merchant.InvestedGold != 3 {
		t.Errorf("Expected invested gold 3, got %d", merchant.InvestedGold)
	}
	if merchant.StoredGold != 2 { // 5 - 3 = 2
		t.Errorf("Expected stored gold 2, got %d", merchant.StoredGold)
	}
}
