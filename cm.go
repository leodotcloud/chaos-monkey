package main

import (
	"math/rand"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/leodotcloud/chaos-monkey/scenarios"
	"github.com/leodotcloud/chaos-monkey/types"
	"github.com/leodotcloud/chaos-monkey/utils"
	//"github.com/rancher/go-rancher/v2"
)

const (
	// DefaultMinWaitTime ...
	DefaultMinWaitTime = 120
	// DefaultMaxWaitTime ...
	DefaultMaxWaitTime = 600
	// DefaultStartClusterSize ...
	DefaultStartClusterSize = 10
	// DefaultMinimumClusterSize ...
	DefaultMinimumClusterSize = 5
	// DefaultMaximumClusterSize ...
	DefaultMaximumClusterSize = 15
)

func test() {
	logrus.Infof("hello")
}

// ChaosMonkey continoulsy runs one of the chaos scenarios at random intervals
type ChaosMonkey struct {
	url             string
	cattleAccessKey string
	cattleSecretKey string
	cattleProjectID string
	minWait         int
	maxWait         int
	seed            int64
	sharedInfo      *types.SharedInfo
}

// NewChaosMonkey returns a new instance of ChaosMonkey
func NewChaosMonkey(url, cattleProjectID, cattleAccessKey, cattleSecretKey string,
	minWait, maxWait int, seed int64,
	sharedInfo *types.SharedInfo) (*ChaosMonkey, error) {
	// TODO: check if valid URL
	// TODO: check if access key/secret key are working

	parsedURL, err := utils.GetParsedBaseURL(url)
	if err != nil {
		return nil, err
	}

	rawClient, err := utils.GetRawClient(parsedURL, cattleAccessKey, cattleSecretKey)
	if err != nil {
		return nil, err
	}

	// TODO: if using same setup, check if current environment has allowSystemRole.
	// If already given a projectID, then ignore
	if cattleProjectID == "" {
		cattleProjectID, err = utils.GetChaosMonkeyProjectID(rawClient)
		if err != nil {
			return nil, err
		}
	}
	logrus.Infof("using project id: %v", cattleProjectID)

	client, err := utils.GetClientForProject(parsedURL, cattleProjectID, cattleAccessKey, cattleSecretKey)
	if err != nil {
		return nil, err
	}
	sharedInfo.Client = client
	// TODO: If no cloud provider is specified, disable other options dependent on that.

	if seed == 0 {
		seed = time.Now().UTC().UnixNano()
	}
	logrus.Infof("Using seed: %v", seed)

	// TODO: Check which are actually needed
	return &ChaosMonkey{
		url:             url,
		cattleAccessKey: cattleAccessKey,
		cattleSecretKey: cattleSecretKey,
		cattleProjectID: cattleProjectID,
		minWait:         minWait,
		maxWait:         maxWait,
		seed:            seed,
		sharedInfo:      sharedInfo,
	}, nil
}

// Run starts the chaos tests against the provided URL
func (cm *ChaosMonkey) Run() error {
	logrus.Infof("Running ChaosMonkey")
	rand.Seed(cm.seed)

	if err := cm.Setup(); err != nil {
		logrus.Errorf("error setting up: %v", err)
		return err
	}

	// Initialize the scenarios
	scenarios := scenarios.GetScenarios()

	for {
		randomPick := rand.Intn(len(scenarios))
		randomScenario := scenarios[randomPick]

		if randomScenario.IsSkip() {
			logrus.Debugf("Skip scenario: %v", randomScenario.GetName())
			continue
		}

		logrus.Infof("Triggering scenario: %v", randomScenario.GetName())
		if err := randomScenario.Run(cm.sharedInfo); err != nil {
			logrus.Infof("Error running scenario %v: %v", randomScenario.GetName(), err)
		}

		// TODO: Notify interested parties?

		randomInterval := cm.minWait + rand.Intn(cm.maxWait-cm.minWait)
		logrus.Debugf("sleeping for randomInterval: %v before next run", randomInterval)
		time.Sleep(time.Duration(randomInterval) * time.Second)
	}
}

// Setup does the initial setup of the cluster with the needed size
// etc.
func (cm *ChaosMonkey) Setup() error {
	logrus.Debugf("Doing Setup for ChaosMonkey")
	utils.SetupCluster(cm.sharedInfo)
	return nil
}
