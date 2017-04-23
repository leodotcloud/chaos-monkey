package types

import (
	"github.com/rancher/go-rancher/v2"
)

// SharedInfo ...
type SharedInfo struct {
	Client                  *client.RancherClient
	Rawclient               *client.RancherClient
	DockerProxies           map[string]string
	UseDigitalOcean         bool
	DigitalOceanAccessToken string
	UseAWS                  bool
	AWSAccessKeyID          string
	AWSSecretAccessKey      string
	UsePacket               bool
	PacketProjectID         string
	PacketToken             string
	StartClusterSize        int
	MinClusterSize          int
	MaxClusterSize          int
	DisableAddHostScenario  bool
	DisableDelHostScenario  bool
}
