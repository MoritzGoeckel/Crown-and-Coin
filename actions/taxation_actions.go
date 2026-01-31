package actions

import (
	"errors"
	"crown_and_coin/engine"
	"crown_and_coin/events"
)

// TaxPeasantsAction - Monarch taxes peasants (low or high)
type TaxPeasantsAction struct {
	BaseAction
	CountryID string
	HighTax   bool // false = 5 gold safe, true = 10 gold with revolt risk
}

func NewTaxPeasantsAction(playerID, countryID string, highTax bool) *TaxPeasantsAction {
	actionType := ActionTaxPeasantsLow
	if highTax {
		actionType = ActionTaxPeasantsHigh
	}
	return &TaxPeasantsAction{
		BaseAction: BaseAction{actionType: actionType, playerID: playerID},
		CountryID:  countryID,
		HighTax:    highTax,
	}
}

func (a *TaxPeasantsAction) Validate(state *engine.GameState) error {
	country := state.GetCountry(a.CountryID)
	if country == nil {
		return errors.New("country not found")
	}
	if country.IsRepublic {
		return errors.New("republics cannot use monarch tax actions")
	}
	if country.MonarchID != a.playerID {
		return errors.New("only the monarch can tax peasants")
	}
	return nil
}

func (a *TaxPeasantsAction) Apply(state *engine.GameState, roller DiceRoller) (*engine.GameState, []events.Event) {
	newState := state.Clone()
	country := newState.GetCountry(a.CountryID)
	var evts []events.Event

	goldPerPeasant := 5
	if a.HighTax {
		goldPerPeasant = 10
	}

	totalGold := goldPerPeasant * country.Peasants
	country.AddGold(totalGold)
	evts = append(evts, events.NewPeasantTaxEvent(a.CountryID, totalGold, a.HighTax))

	// Check for revolt on high tax (2/6 chance)
	if a.HighTax {
		roll := roller.Roll(6)
		if roll <= 2 {
			damage := 2
			country.TakeDamage(damage)
			evts = append(evts, events.NewPeasantRevoltEvent(a.CountryID, damage))
		}
	}

	return newState, evts
}

// TaxMerchantsAction - Monarch collects tax from merchants
type TaxMerchantsAction struct {
	BaseAction
	CountryID  string
	MerchantID string
	Amount     int
}

func NewTaxMerchantsAction(playerID, countryID, merchantID string, amount int) *TaxMerchantsAction {
	return &TaxMerchantsAction{
		BaseAction: BaseAction{actionType: ActionTaxMerchants, playerID: playerID},
		CountryID:  countryID,
		MerchantID: merchantID,
		Amount:     amount,
	}
}

func (a *TaxMerchantsAction) Validate(state *engine.GameState) error {
	country := state.GetCountry(a.CountryID)
	if country == nil {
		return errors.New("country not found")
	}
	if country.IsRepublic {
		return errors.New("republics handle taxes differently")
	}
	if country.MonarchID != a.playerID {
		return errors.New("only the monarch can tax merchants")
	}
	merchant := state.GetMerchant(a.MerchantID)
	if merchant == nil {
		return errors.New("merchant not found")
	}
	if merchant.CountryID != a.CountryID {
		return errors.New("merchant does not belong to this country")
	}
	if a.Amount < 0 {
		return errors.New("tax amount cannot be negative")
	}
	return nil
}

func (a *TaxMerchantsAction) Apply(state *engine.GameState, roller DiceRoller) (*engine.GameState, []events.Event) {
	newState := state.Clone()
	country := newState.GetCountry(a.CountryID)
	merchant := newState.GetMerchant(a.MerchantID)
	var evts []events.Event

	// Merchant pays what they can
	actualPaid := merchant.PayTax(a.Amount)
	country.AddGold(actualPaid)

	evts = append(evts, events.NewMerchantTaxEvent(a.CountryID, a.MerchantID, actualPaid))

	return newState, evts
}
