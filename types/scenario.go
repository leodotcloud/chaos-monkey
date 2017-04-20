package types

import (
//"github.com/rancher/go-rancher/v2"
)

// Scenario ...
type Scenario interface {
	// GetName ...
	GetName() string
	// Run ...
	Run(*SharedInfo) error
	// IsSkip ...
	IsSkip() bool
}
