package types

// BaseScenario ...
type BaseScenario struct {
	Name string
	Skip bool
}

// GetName returns the name of the Scenario
func (bs *BaseScenario) GetName() string {
	return bs.Name
}

// IsSkip ...
func (bs *BaseScenario) IsSkip() bool {
	return bs.Skip
}
