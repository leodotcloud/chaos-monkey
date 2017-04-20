package dns

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/leodotcloud/chaos-monkey/types"
	//"github.com/rancher/go-rancher/v2"
)

// ReloadOneRandomDNSContainerUsingAPI ...
type ReloadOneRandomDNSContainerUsingAPI struct{ types.BaseScenario }

// Run ...
func (s *ReloadOneRandomDNSContainerUsingAPI) Run(si *types.SharedInfo) error {
	logrus.Debugf("Running Scenario: %v", s.Name)

	return fmt.Errorf("Not implemented")
}
