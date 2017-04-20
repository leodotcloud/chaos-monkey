package types

import (
	"github.com/rancher/go-rancher/v2"
)

// SharedInfo ...
type SharedInfo struct {
	Client                  *client.RancherClient
	Rawclient               *client.RancherClient
	DockerProxies           map[string]string
	DigitalOceanAccessToken string
	StartClusterSize        int
	MinClusterSize          int
	MaxClusterSize          int
}
