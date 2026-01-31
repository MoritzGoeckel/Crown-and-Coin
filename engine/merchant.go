package engine

// Merchant represents a merchant player in the game
type Merchant struct {
	ID           string `json:"id"`
	CountryID    string `json:"country_id"`    // Which country this merchant belongs to
	StoredGold   int    `json:"stored_gold"`   // Personal savings (can take when fleeing)
	InvestedGold int    `json:"invested_gold"` // Gold invested (doubles next turn, lost if fleeing)
}

// NewMerchant creates a new merchant with default values
func NewMerchant(id string, countryID string) *Merchant {
	return &Merchant{
		ID:           id,
		CountryID:    countryID,
		StoredGold:   0,
		InvestedGold: 0,
	}
}

// TotalGold returns the sum of stored and invested gold
func (m *Merchant) TotalGold() int {
	return m.StoredGold + m.InvestedGold
}

// ReceiveIncome adds gold to stored gold (automatic 5 gold per turn)
func (m *Merchant) ReceiveIncome(amount int) {
	m.StoredGold += amount
}

// PayTax removes gold from stored gold, returns actual amount paid
func (m *Merchant) PayTax(amount int) int {
	if m.StoredGold >= amount {
		m.StoredGold -= amount
		return amount
	}
	paid := m.StoredGold
	m.StoredGold = 0
	return paid
}

// Invest moves gold from stored to invested
func (m *Merchant) Invest(amount int) bool {
	if m.StoredGold < amount {
		return false
	}
	m.StoredGold -= amount
	m.InvestedGold += amount
	return true
}

// CollectInvestment doubles invested gold and moves it to stored
func (m *Merchant) CollectInvestment() int {
	payout := m.InvestedGold * 2
	m.StoredGold += payout
	m.InvestedGold = 0
	return payout
}

// Hide is an alias for keeping gold in stored (no-op, gold stays in StoredGold)
func (m *Merchant) Hide(amount int) bool {
	// Gold is already in StoredGold, this is just for action clarity
	return m.StoredGold >= amount
}

// FleeToCountry moves merchant to a new country, losing invested gold
func (m *Merchant) FleeToCountry(newCountryID string) {
	m.CountryID = newCountryID
	m.InvestedGold = 0 // Lose investments when fleeing
}

// LoseAllGold transfers all gold away (used in failed revolt)
func (m *Merchant) LoseAllGold() int {
	total := m.StoredGold + m.InvestedGold
	m.StoredGold = 0
	m.InvestedGold = 0
	return total
}

// Clone creates a deep copy of the merchant
func (m *Merchant) Clone() *Merchant {
	return &Merchant{
		ID:           m.ID,
		CountryID:    m.CountryID,
		StoredGold:   m.StoredGold,
		InvestedGold: m.InvestedGold,
	}
}
