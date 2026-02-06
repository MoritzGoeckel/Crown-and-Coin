package test

import (
	"encoding/json"
	"os"
	"testing"

	"crown_and_coin/engine"
	"crown_and_coin/jsonapi"
)

type Scenario struct {
	Name  string `json:"name"`
	Seed  int64  `json:"seed"`
	Steps []Step `json:"steps"`
}

type Step struct {
	Request  json.RawMessage `json:"request"`
	Expected json.RawMessage `json:"expected"`
}

func TestScenarios(t *testing.T) {
	scenarios := loadScenarios(t, "scenarios.json")

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			runScenario(t, scenario)
		})
	}
}

func loadScenarios(t *testing.T, filename string) []Scenario {
	t.Helper()

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read scenarios file: %v", err)
	}

	var scenarios []Scenario
	if err := json.Unmarshal(data, &scenarios); err != nil {
		t.Fatalf("Failed to parse scenarios JSON: %v", err)
	}

	return scenarios
}

func runScenario(t *testing.T, scenario Scenario) {
	t.Helper()

	api := jsonapi.NewGameAPIWithDice(engine.NewSeededDice(scenario.Seed))

	for i, step := range scenario.Steps {
		resp, err := api.ProcessMessage(step.Request)
		if err != nil {
			t.Fatalf("Step %d: ProcessMessage error: %v\nRequest: %s", i+1, err, string(step.Request))
		}

		if !jsonEqualIgnoreEvents(t, resp, step.Expected) {
			t.Errorf("Step %d: JSON mismatch\nRequest:  %s\nExpected: %s\nActual:   %s",
				i+1,
				formatJSON(step.Request),
				formatJSON(step.Expected),
				formatJSON(resp))
		}
	}
}

func jsonEqualIgnoreEvents(t *testing.T, actual []byte, expected []byte) bool {
	t.Helper()

	var actualObj, expectedObj map[string]interface{}
	if err := json.Unmarshal(actual, &actualObj); err != nil {
		t.Fatalf("Failed to parse actual JSON: %v\nJSON: %s", err, string(actual))
	}
	if err := json.Unmarshal(expected, &expectedObj); err != nil {
		t.Fatalf("Failed to parse expected JSON: %v\nJSON: %s", err, string(expected))
	}

	// Remove events field from both before comparison
	delete(actualObj, "events")
	delete(expectedObj, "events")

	return deepEqual(actualObj, expectedObj)
}

func deepEqual(a, b interface{}) bool {
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		for k, v := range aVal {
			if !deepEqual(v, bVal[k]) {
				return false
			}
		}
		return true
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok || len(aVal) != len(bVal) {
			return false
		}
		// Compare arrays as unordered sets
		used := make([]bool, len(bVal))
		for _, aItem := range aVal {
			found := false
			for j, bItem := range bVal {
				if !used[j] && deepEqual(aItem, bItem) {
					used[j] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}

func formatJSON(data []byte) string {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return string(data)
	}
	formatted, _ := json.MarshalIndent(obj, "", "  ")
	return string(formatted)
}
