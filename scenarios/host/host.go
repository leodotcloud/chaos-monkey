package host

import (
	"github.com/Sirupsen/logrus"
	"github.com/leodotcloud/chaos-monkey/types"
	"github.com/leodotcloud/chaos-monkey/utils"
	//"github.com/rancher/go-rancher/v2"
)

// AddHostUsingAPI ...
type AddHostUsingAPI struct{ types.BaseScenario }

// DeleteHostUsingAPI ...
type DeleteHostUsingAPI struct{ types.BaseScenario }

// Run ...
func (s *AddHostUsingAPI) Run(si *types.SharedInfo) error {
	logrus.Debugf("Running Scenario: %v", s.Name)

	return utils.AddHostsUsingAPI(si, 1, si.MaxClusterSize)
}

// Run ...
func (s *DeleteHostUsingAPI) Run(si *types.SharedInfo) error {
	logrus.Debugf("Running Scenario: %v", s.Name)

	return utils.DeleteHostsUsingAPI(si, 1)
}
