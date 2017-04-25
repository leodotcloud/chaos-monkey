package utils

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	dtypes "github.com/docker/docker/api/types"
	dc "github.com/docker/docker/client"
	"github.com/leodotcloud/chaos-monkey/types"
	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/rancher-docker-api-proxy"
)

// GetParsedBaseURL ...
func GetParsedBaseURL(inputURL string) (string, error) {
	u, err := url.Parse(inputURL)
	if err != nil {
		return "", err
	}
	newURL := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}
	return newURL.String(), nil
}

// GetRawClient ...
func GetRawClient(url, accessKey, secretKey string) (*client.RancherClient, error) {
	url = url + "/v2-beta"
	c, err := client.NewRancherClient(&client.ClientOpts{
		Url:       url,
		AccessKey: accessKey,
		SecretKey: secretKey,
	})

	if err != nil {
		return nil, err
	}

	return c, nil
}

// GetClientForProject gets the client for a specific Rancher Project.
// TODO: validates the credentials provided
func GetClientForProject(url, projectID, accessKey, secretKey string) (*client.RancherClient, error) {
	if projectID == "" {
		return nil, fmt.Errorf("no project ID specified")
	}

	url = url + "/v2-beta/projects/" + projectID + "/schemas"
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

// AddHostsUsingAPIWithoutAnyChecks ...
func AddHostsUsingAPIWithoutAnyChecks(si *types.SharedInfo, N int) error {
	// TODO: Fix for other clouds?
	return AddDigitalOceanHostsUsingAPI(si, N)

}

// AddHostsUsingAPI ...
func AddHostsUsingAPI(si *types.SharedInfo, N, expectedMaxSize int) error {
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

	if currentNumOfHosts > expectedMaxSize {
		return fmt.Errorf("current number of hosts(%v) is more than maximum size(%v), can't add",
			currentNumOfHosts, expectedMaxSize)
	}

	if N == 0 {
		N = expectedMaxSize - currentNumOfHosts
	}

	afterAddNumOfHosts := currentNumOfHosts + N
	if afterAddNumOfHosts > expectedMaxSize {
		logrus.Infof("N(%v)to add will make the cluster size > maximum size (%v)",
			N, si.MinClusterSize)
		newN := expectedMaxSize - currentNumOfHosts
		logrus.Infof("hence adding only %v hosts, instead of %v", newN, N)
		N = newN
	}

	return AddHostsUsingAPIWithoutAnyChecks(si, N)
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

func getDORandomHostImageName() string {
	images := []string{"centos-7-x64", "ubuntu-16-04-x64", "ubuntu-14-04-x64", "fedora-24-x64"}
	return images[rand.Intn(len(images))]
}

func getDORandomHostSize() string {
	//sizes := []string{"1gb", "2gb", "4gb", "8gb", "16gb", "m-16gb"}
	sizes := []string{"1gb", "2gb", "4gb", "8gb"}
	return sizes[rand.Intn(len(sizes))]
}

func getDORandomRegion() string {
	locations := []string{"sfo1", "sfo2", "nyc1", "nyc2", "nyc3"}
	return locations[rand.Intn(len(locations))]
}

// AddDigitalOceanHostsUsingAPI ...
// If N=0, random number depends on the logic
func AddDigitalOceanHostsUsingAPI(si *types.SharedInfo, N int) error {
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
			Image:             getDORandomHostImageName(),
			PrivateNetworking: false,
			Region:            getDORandomRegion(),
			Size:              getDORandomHostSize(),
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
	logrus.Debugf("SetupCluster")

	err := AddHostsUsingAPI(si, si.StartClusterSize, si.StartClusterSize)
	if err != nil {
		return err
	}

	stack, err := AddStack(si, "cmstack-long")
	if err != nil {
		return err
	}

	_, err = AddService(si, stack.Id, "cmservice-long", false)
	if err != nil {
		return err
	}

	return nil
}

// AddStack creates an empty stack and start it
func AddStack(si *types.SharedInfo, stackName string) (*client.Stack, error) {
	logrus.Debugf("AddStack: %v", stackName)
	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_eq": stackName,
		},
	}

	collection, err := si.Client.Stack.List(listOpts)
	if err != nil {
		return nil, err
	}

	if len(collection.Data) > 0 {
		return &collection.Data[0], nil
	}

	stack := client.Stack{
		Name:          stackName,
		StartOnCreate: true,
	}
	return si.Client.Stack.Create(&stack)
}

// DeleteStack creates an empty stack and start it
func DeleteStack(si *types.SharedInfo, stackName string) error {
	logrus.Debugf("DeleteStack: %v", stackName)
	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_eq": stackName,
		},
	}

	collection, err := si.Client.Stack.List(listOpts)
	if err != nil {
		return err
	}

	if !(len(collection.Data) > 0) {
		return fmt.Errorf("stack doesn't exist with given name: %v", stackName)
	}

	stack := collection.Data[0]
	err = si.Client.Stack.Delete(&stack)
	if err != nil {
		return err
	}

	return nil
}

// AddService ...
func AddService(si *types.SharedInfo, stackID, serviceName string, enableHealthCheck bool) (*client.Service, error) {
	logrus.Debugf("AddService: %v", serviceName)

	service, err := getServiceByName(si, serviceName)
	if err == nil {
		return service, nil
	}

	service = &client.Service{
		StackId:       stackID,
		Name:          serviceName,
		Scale:         1,
		StartOnCreate: true,
		LaunchConfig: &client.LaunchConfig{
			ImageUuid:             "docker:leodotcloud/self-health-status:dev",
			StdinOpen:             true,
			StartOnCreate:         true,
			InstanceTriggeredStop: "stop",
			Vcpu: 1,
			Labels: map[string]interface{}{
				"io.rancher.container.pull_image": "always",
			},
		},
	}

	if enableHealthCheck {
		service.LaunchConfig.HealthCheck = &client.InstanceHealthCheck{
			HealthyThreshold:    2,
			InitializingTimeout: 60000,
			Interval:            2000,
			Port:                80,
			ReinitializingTimeout: 60000,
			RequestLine:           `GET "/v1/healthcheck" "HTTP/1.0"`,
			ResponseTimeout:       2000,
			Strategy:              "none",
			UnhealthyThreshold:    3,
		}
	}

	return si.Client.Service.Create(service)
}

func getServiceByName(si *types.SharedInfo, serviceName string) (*client.Service, error) {
	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_eq": serviceName,
		},
	}

	collection, err := si.Client.Service.List(listOpts)
	if err != nil {
		return nil, err
	}

	if !(len(collection.Data) > 0) {
		return nil, fmt.Errorf("service doesn't exist with given name: %v", serviceName)
	}

	service := collection.Data[0]

	return &service, nil
}

// DeleteServiceByName ...
func DeleteServiceByName(si *types.SharedInfo, serviceName string) error {
	logrus.Debugf("DeleteService: %v", serviceName)

	service, err := getServiceByName(si, serviceName)
	if err != nil {
		return err
	}

	err = si.Client.Service.Delete(service)
	if err != nil {
		return err
	}
	return nil
}

// DeleteServiceByID ...
func DeleteServiceByID(si *types.SharedInfo, serviceID string) error {
	logrus.Debugf("DeleteService: %v", serviceID)

	service, err := si.Client.Service.ById(serviceID)
	if err != nil {
		return err
	}

	err = si.Client.Service.Delete(service)
	if err != nil {
		return err
	}

	return nil
}

// ChangeServiceScale ...
func ChangeServiceScale(si *types.SharedInfo, serviceName string, newScale int) error {
	logrus.Debugf("ChangeServiceScale of %v to %v", serviceName, newScale)

	service, err := getServiceByName(si, serviceName)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"scale": newScale,
	}
	_, err = si.Client.Service.Update(service, updates)
	if err != nil {
		return err
	}

	return nil
}

// AddAPIAccountKey ...
func AddAPIAccountKey(si *types.SharedInfo) error {
	ak, err := si.Client.ApiKey.Create(&client.ApiKey{
		Name: "juliet",
	})
	if err != nil {
		return err
	}

	logrus.Debugf("ak: %#+v", *ak)
	return nil
}

// EnableSystemRole ...
func EnableSystemRole(si *types.SharedInfo) error {
	return nil
}

// AddLongRunningStack creates a stack with the name cmstack-long-...
// so that this stack is never deleted in any of the chaos tests
func AddLongRunningStack() {
}

func getProjectList(si *types.SharedInfo) error {
	listOpts := &client.ListOpts{Filters: map[string]interface{}{}}
	collection, err := si.RawClient.Project.List(listOpts)
	if err != nil {
		return err
	}

	logrus.Infof("projects: %v", collection.Data)
	return nil
}

// GetSelfProjectUUID ...
func GetSelfProjectUUID() (string, error) {
	mc, err := metadata.NewClientAndWait("http://rancher-metadata/2016-07-29")
	if err != nil {
		logrus.Errorf("error creating metadata client: %v", err)
		return "", err
	}

	self, err := mc.GetSelfContainer()
	if err != nil {
		logrus.Errorf("error getting self container from metadata: %v", err)
		return "", err
	}

	return self.EnvironmentUUID, nil
}

// GetSelfProjectID ...
func GetSelfProjectID(rawClient *client.RancherClient) (string, error) {
	selfProjectUUID, err := GetSelfProjectUUID()
	if err != nil {
		return "", err
	}
	logrus.Debugf("got selfProjectUUID from metadata: %v", selfProjectUUID)

	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"uuid_eq": selfProjectUUID,
		},
	}

	collection, err := rawClient.Project.List(listOpts)
	if err != nil {
		logrus.Errorf("error getting self project: %v", err)
		return "", fmt.Errorf("probably access/secrey keys provided are not associated with 'isSystemRole: true'")
	}

	if len(collection.Data) == 0 {
		e := fmt.Errorf("error finding self project using the API")
		logrus.Errorf("%v", e)
		return "", e
	}

	if len(collection.Data) != 1 {
		e := fmt.Errorf("expecting only one match for self container but found: %v", len(collection.Data))
		logrus.Errorf("error: %v", e)
		return "", e
	}

	return collection.Data[0].Id, nil
}

// GetChaosMonkeyProjectID ...
func GetChaosMonkeyProjectID(rawClient *client.RancherClient) (string, error) {
	defaultChaosMonkeyProjectName := "chaosmonkey"

	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_eq":  defaultChaosMonkeyProjectName,
			"state_eq": "active",
		},
	}

	collection, err := rawClient.Project.List(listOpts)
	if err != nil {
		logrus.Errorf("error getting self project: %v", err)
		return "", err
	}
	logrus.Infof("collection: %+v", collection)

	l := len(collection.Data)
	if l > 1 {
		e := fmt.Errorf("expecting only one chaosmonkey environment but found: %v", l)
		logrus.Errorf("%v", e)
		return "", e
	} else if l == 1 {
		return collection.Data[0].Id, nil
	}

	// TODO: support for custom catalog
	p, err := CreateProject(rawClient, defaultChaosMonkeyProjectName, "Cattle", "library")
	if err != nil {
		return "", err
	}

	return p.Id, nil
}

// CreateProject ...
// TODO: Probably needs work for custom template
func CreateProject(rawClient *client.RancherClient, projectName, projectTemplateName, catalogName string) (*client.Project, error) {
	logrus.Debugf("CreateProject: projectName=%v projectTemplateName=%v catalogName=%v",
		projectName, projectTemplateName, catalogName)

	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_like":       "%" + projectTemplateName + "%",
			"externalId_like": "catalog://" + catalogName + "%",
		},
	}

	collection, err := rawClient.ProjectTemplate.List(listOpts)
	if err != nil {
		logrus.Errorf("error getting self project: %v", err)
		return nil, err
	}

	l := len(collection.Data)
	if l == 0 {
		return nil, fmt.Errorf("no project templates found")
	} else if l > 1 {
		return nil, fmt.Errorf("expecting only one project template but found: %v", l)
	}

	template := collection.Data[0]

	// TODO: Needs work for authentication
	p := &client.Project{
		Name:              projectName,
		ProjectTemplateId: template.Id,
		AllowSystemRole:   true,
	}

	p, err = rawClient.Project.Create(p)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("created project: %v", p)
	return p, nil
}

// DeleteProject ...
// TODO: Probably needs work for custom template
func DeleteProject(rawClient *client.RancherClient, projectName string) error {
	logrus.Debugf("DeleteProject: projectName=%v", projectName)

	listOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name_eq":   projectName,
			"state_neq": "removed",
		},
	}

	collection, err := rawClient.Project.List(listOpts)
	if err != nil {
		logrus.Errorf("error getting self project: %v", err)
		return err
	}

	l := len(collection.Data)
	if l == 0 {
		return fmt.Errorf("no project found with name: %v", projectName)
	} else if l > 1 {
		return fmt.Errorf("expecting only one project with name %v but found: %v", projectName, l)
	}

	p := &collection.Data[0]

	if p.State != "inactive" {
		_, err = rawClient.Project.ActionDeactivate(p)
		if err != nil {
			return err
		}
	}

	//TODO: Not working
	err = rawClient.Project.Delete(p)
	if err != nil {
		return err
	}

	logrus.Infof("deleted project: %v", projectName)
	return nil
}
