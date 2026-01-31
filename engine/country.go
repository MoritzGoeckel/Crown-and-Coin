package engine

// Country represents a nation in the game
type Country struct {
	ID           string
	HP           int    // Health points, starts at 10
	ArmyStrength int    // Military power, starts at 0
	Gold         int    // Treasury
	Peasants     int    // Tax base, starts at 1
	IsRepublic   bool   // false = monarchy, true = merchant republic
	MonarchID    string // Player controlling the country (if monarchy)
	DiedOnce     bool   // Tracks if country already used its "revival"
}

// NewCountry creates a new country with default starting values
func NewCountry(id string, monarchID string) *Country {
	return &Country{
		ID:           id,
		HP:           10,
		ArmyStrength: 0,
		Gold:         0,
		Peasants:     1,
		IsRepublic:   false,
		MonarchID:    monarchID,
		DiedOnce:     false,
	}
}

// IsAlive returns true if the country still has HP
func (c *Country) IsAlive() bool {
	return c.HP > 0
}

// TakeDamage reduces HP by the given amount, handling the first-death revival
func (c *Country) TakeDamage(damage int) {
	c.HP -= damage
	if c.HP <= 0 && !c.DiedOnce {
		c.HP = 1
		c.DiedOnce = true
	}
}

// AddArmy increases army strength
func (c *Country) AddArmy(amount int) {
	c.ArmyStrength += amount
}

// HalveArmy reduces army strength by half (maintenance cost)
func (c *Country) HalveArmy() {
	c.ArmyStrength = c.ArmyStrength / 2
}

// AddGold adds gold to the treasury
func (c *Country) AddGold(amount int) {
	c.Gold += amount
}

// SpendGold removes gold from the treasury, returns false if insufficient
func (c *Country) SpendGold(amount int) bool {
	if c.Gold < amount {
		return false
	}
	c.Gold -= amount
	return true
}

// AddPeasant increases the peasant count
func (c *Country) AddPeasant() {
	c.Peasants++
}

// BecomeRepublic converts the country to a merchant republic
func (c *Country) BecomeRepublic() {
	c.IsRepublic = true
	c.MonarchID = ""
}

// Clone creates a deep copy of the country
func (c *Country) Clone() *Country {
	return &Country{
		ID:           c.ID,
		HP:           c.HP,
		ArmyStrength: c.ArmyStrength,
		Gold:         c.Gold,
		Peasants:     c.Peasants,
		IsRepublic:   c.IsRepublic,
		MonarchID:    c.MonarchID,
		DiedOnce:     c.DiedOnce,
	}
}
