package scenarios

import (
	"github.com/Sirupsen/logrus"
	"github.com/leodotcloud/chaos-monkey/scenarios/dns"
	"github.com/leodotcloud/chaos-monkey/scenarios/host"
	"github.com/leodotcloud/chaos-monkey/scenarios/ipsec"
	"github.com/leodotcloud/chaos-monkey/scenarios/metadata"
	"github.com/leodotcloud/chaos-monkey/types"
)

// GetScenarios collectes all the various scenarios available
func GetScenarios() []types.Scenario {
	logrus.Debugf("collecting scenarios")
	scenarios := []types.Scenario{
		&host.AddHostUsingAPI{types.BaseScenario{Skip: false, Name: "Add a Host using Rancher API"}},
		&host.DeleteHostUsingAPI{types.BaseScenario{Skip: false, Name: "Delete a Host using Rancher API"}},

		&dns.ReloadOneRandomDNSContainerUsingAPI{types.BaseScenario{Skip: false, Name: "Reload a random DNS container using API"}},

		&metadata.ReloadOneRandomMetadataContainerUsingAPI{types.BaseScenario{Skip: true, Name: "Reload a random Metadata container using API"}},

		&ipsec.ReloadOneRandomIPSecContainerUsingAPI{types.BaseScenario{Skip: false, Name: "Reload a random IPSec router container using API"}},
		&ipsec.RemoveOneRandomIPSecContainerUsingDocker{types.BaseScenario{Skip: false, Name: "Remove a random IPSec router container using Docker"}},
	}

	return scenarios
}
