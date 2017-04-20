package utils

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	dtypes "github.com/docker/docker/api/types"
	dc "github.com/docker/docker/client"
	"github.com/leodotcloud/chaos-monkey/types"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-docker-api-proxy"
)

// GetClientForProject gets the client for a specific Rancher Project.
// TODO: validates the credentials provided
func GetClientForProject(url, envID, accessKey, secretKey string) (*client.RancherClient, error) {
	url = url + "/v2-beta/projects/" + envID + "/schemas"
	c, err := client.NewRancherClient(&client.ClientOpts{
		Url:       url,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})

	if err != nil {
		return nil, err
	}

	//projectCollection, err := c.Project.List(&client.ListOpts{})
	//if err != nil {
	//	return nil, err
	//}

	//if !(len(projectCollection.Data) > 0) {
	//	return nil, fmt.Errorf("no valid project found")
	//}

	//currentProj := projectCollection.Data[0]
	//updates := map[string]string{
	//	"name": currentProj.Name,
	//}
	//_, err = c.Project.Update(&currentProj, updates)
	//if err != nil {
	//	return nil, err
	//}

	return c, nil
}

func determineAPIVersion(host *client.Host) string {
	version := host.Labels["io.rancher.host.docker_version"]
	parts := strings.Split(fmt.Sprint(version), ".")
	if len(parts) != 2 {
		return ""
	}
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return ""
	}

	return fmt.Sprintf("1.%d", num+12)
}

// ReloadRandomInstanceUsingAPI ...
func ReloadRandomInstanceUsingAPI(c *client.RancherClient, instances []client.Instance) error {
	length := len(instances)
	if !(length > 0) {
		return fmt.Errorf("no instances available for reload")
	}

	randomInstance := instances[rand.Intn(length)]
	logrus.Debugf("reloading instance using API: %v", randomInstance.Name)
	_, err := c.Instance.ActionRestart(&randomInstance)
	if err != nil {
		return err
	}
	return nil
}

// RemoveRandomInstanceUsingAPI ...
func RemoveRandomInstanceUsingAPI(c *client.RancherClient, instances []client.Instance) error {
	length := len(instances)
	if !(length > 0) {
		return fmt.Errorf("no instances available to remove")
	}

	randomInstance := instances[rand.Intn(length)]
	logrus.Debugf("removing instance using docker: %v", randomInstance.Name)

	_, err := c.Instance.ActionRemove(&randomInstance)
	if err != nil {
		return err
	}
	return nil
}

// RemoveRandomInstanceUsingDocker ...
func RemoveRandomInstanceUsingDocker(si *types.SharedInfo, instances []client.Instance) error {
	length := len(instances)
	if !(length > 0) {
		return fmt.Errorf("no instances available to remove")
	}

	randomInstance := instances[rand.Intn(length)]
	logrus.Debugf("removing instance using docker: %v", randomInstance.Name)

	dockerClient, err := GetDockerClientForHost(si, randomInstance.HostId)
	if err != nil {
		return err
	}

	removeOpts := dtypes.ContainerRemoveOptions{
		Force: true,
	}
	err = dockerClient.ContainerRemove(context.Background(), randomInstance.ExternalId, removeOpts)
	if err != nil {
		return nil
	}

	return nil
}

// GetDockerProxyInfoForHost ...
func GetDockerProxyInfoForHost(si *types.SharedInfo, hostID string) (string, error) {
	proxy, ok := si.DockerProxies[hostID]
	if !ok {
		return StartDockerProxyForHost(si, hostID)
	}

	return proxy, nil
}

// GetDockerClientForHost ...
// TODO: Should I cache the docker client as well???
func GetDockerClientForHost(si *types.SharedInfo, hostID string) (*dc.Client, error) {
	proxy, err := GetDockerProxyInfoForHost(si, hostID)
	if err != nil {
		return nil, err
	}

	// TODO: Fix this later
	apiVersion := "1.24"
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	dockerClient, err := dc.NewClient(proxy, apiVersion, nil, defaultHeaders)
	if err != nil {
		return nil, err
	}

	return dockerClient, nil
}

// StartDockerProxyForHost ...
func StartDockerProxyForHost(si *types.SharedInfo, hostID string) (string, error) {
	host, err := si.Client.Host.ById(hostID)
	if err != nil {
		return "", err
	}

	if host.State != "active" {
		return "", fmt.Errorf("Can not contact host %s in state %s", host.Hostname, host.State)
	}

	tempfile, err := ioutil.TempFile("", "docker-sock")
	if err != nil {
		return "", err
	}

	if err := tempfile.Close(); err != nil {
		return "", err
	}

	// TODO: Have an exit channel???
	started := make(chan bool)
	go func(tempfileName string) {
		dockerHost := "unix://" + tempfileName

		logrus.Infof("goroutine: starting proxy for dockerHost: %v", dockerHost)
		proxy := dockerapiproxy.NewProxy(si.Client, host.Id, dockerHost)
		if err := proxy.Listen(); err != nil {
			logrus.Errorf("error listening for proxy: %v", err)
			return
		}
		started <- true
		logrus.Debugf("docker proxy started on %v", tempfileName)
		logrus.Fatal(proxy.Serve())
		os.Remove(tempfileName)
	}(tempfile.Name())

	<-started

	return "unix://" + tempfile.Name(), nil

}

// AddHostsUsingAPI ...
func AddHostsUsingAPI(si *types.SharedInfo, N int) error {
	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_prefix": "cmhost",
			//"state_eq":    "active",  // TODO: Fix with correct states
		},
	}

	collection, err := si.Client.Host.List(listOpts)
	if err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}

	currentNumOfHosts := len(collection.Data)

	if currentNumOfHosts > si.MaxClusterSize {
		return fmt.Errorf("current number of hosts(%v) is more than maximum size(%v), can't add",
			currentNumOfHosts, si.MaxClusterSize)
	}

	if N == 0 {
		N = si.MaxClusterSize - currentNumOfHosts
	}

	afterAddNumOfHosts := currentNumOfHosts + N
	if afterAddNumOfHosts > si.MaxClusterSize {
		logrus.Infof("N(%v)to add will make the cluster size > maximum size (%v)",
			N, si.MinClusterSize)
		newN := si.MaxClusterSize - currentNumOfHosts
		logrus.Infof("hence adding only %v hosts, instead of %v", newN, N)
		N = newN
	}

	// TODO: Fix for other clouds?
	return AddDigitalOceanHosts(si, N)
}

// DeleteHostsUsingAPI ...
func DeleteHostsUsingAPI(si *types.SharedInfo, N int) error {
	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_prefix": "cmhost",
			"state_eq":    "active",
		},
	}

	collection, err := si.Client.Host.List(listOpts)
	if err != nil {
		logrus.Errorf("error: %v", err)
		return err
	}

	currentNumOfHosts := len(collection.Data)
	if !(currentNumOfHosts > 0) {
		return fmt.Errorf("no hosts found in the cluster")
	}

	if currentNumOfHosts < si.MinClusterSize {
		return fmt.Errorf("current number of hosts(%v) is less than minimum size(%v), can't delete",
			currentNumOfHosts, si.MinClusterSize)
	}

	afterDeleteNumOfHosts := currentNumOfHosts - N
	if afterDeleteNumOfHosts < si.MinClusterSize {
		logrus.Infof("N(%v)to delete will make the cluster size < minimum size (%v)",
			N, si.MinClusterSize)
		newN := currentNumOfHosts - si.MinClusterSize
		logrus.Infof("hence deleting only %v hosts, instead of %v", newN, N)
		N = newN
	}

	indicesToDelete := GetNRandomPicksFromPool(N, currentNumOfHosts)
	for index := range indicesToDelete {
		host := &collection.Data[index]
		_, err = si.Client.Host.ActionDeactivate(host)
		if err != nil {
			logrus.Errorf("couldn't deactiviate the host %v: %v", host.Name, err)
			continue
		}
		err = si.Client.Host.Delete(host)
		if err != nil {
			logrus.Errorf("couldn't delete the host %v: %v", host.Name, err)
			continue
		}
	}
	return nil
}

// GetNRandomPicksFromPool ...
func GetNRandomPicksFromPool(N, poolSize int) map[int]int {
	picks := make(map[int]int)

	i := 0
	for i < N {
		index := rand.Intn(poolSize)
		_, ok := picks[index]
		if !ok {
			picks[index] = index
			i++
		}
	}
	return picks
}

// AddDigitalOceanHosts ...
// If N=0, random number depends on the logic
func AddDigitalOceanHosts(si *types.SharedInfo, N int) error {
	if N == 0 {
		// TODO: Fix this
		N = 1
	}

	for i := 0; i < N; i++ {
		doHost := &client.Host{}

		rt := RandomToken()
		doHost.Hostname = "cmhost-" + rt
		doHost.Name = "cmhost-" + rt
		doHost.EngineInstallUrl = "https://releases.rancher.com/install-docker/1.12.sh"

		// TODO: Make this configurable?
		// docker URL
		doConfig := &client.DigitaloceanConfig{
			AccessToken:       si.DigitalOceanAccessToken,
			Backups:           false,
			Image:             "ubuntu-16-04-x64",
			PrivateNetworking: false,
			Region:            "sfo1",
			Size:              "1gb",
			SshUser:           "root",
		}
		doHost.DigitaloceanConfig = doConfig

		h, err := si.Client.Host.Create(doHost)
		if err != nil {
			logrus.Errorf("error: %v", err)
			continue
		}
		logrus.Debugf("created host: %#v", h)
	}

	return nil
}

// RandomToken ...
func RandomToken() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// SetupCluster ...
func SetupCluster(si *types.SharedInfo) error {
	return AddHostsUsingAPI(si, si.StartClusterSize)
}
