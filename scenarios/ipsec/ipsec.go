package ipsec

import (
	//"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/leodotcloud/chaos-monkey/types"
	"github.com/leodotcloud/chaos-monkey/utils"
	"github.com/rancher/go-rancher/v2"
)

// ReloadOneRandomIPSecContainerUsingAPI ...
type ReloadOneRandomIPSecContainerUsingAPI struct{ types.BaseScenario }

// Run ...
func (s *ReloadOneRandomIPSecContainerUsingAPI) Run(si *types.SharedInfo) error {
	logrus.Debugf("Running Scenario: %v", s.Name)

	// TODO: state: running
	instanceListOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name" + "_like": "%ipsec-router%",
		},
	}
	instanceCollection, err := si.Client.Instance.List(instanceListOpts)
	if err != nil {
		return err
	}

	err = utils.ReloadRandomInstanceUsingAPI(si.Client, instanceCollection.Data)
	if err != nil {
		return err
	}

	return nil
}

// RemoveOneRandomIPSecContainerUsingAPI ...
type RemoveOneRandomIPSecContainerUsingAPI struct{ types.BaseScenario }

// Run ...
func (s *RemoveOneRandomIPSecContainerUsingAPI) Run(si *types.SharedInfo) error {
	logrus.Debugf("Running Scenario: %v", s.Name)

	// TODO: state: running
	instanceListOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name" + "_like": "%ipsec-router%",
		},
	}
	instanceCollection, err := si.Client.Instance.List(instanceListOpts)
	if err != nil {
		return err
	}

	err = utils.RemoveRandomInstanceUsingAPI(si.Client, instanceCollection.Data)
	if err != nil {
		return err
	}

	return nil
}

// RemoveOneRandomIPSecContainerUsingDocker ...
type RemoveOneRandomIPSecContainerUsingDocker struct{ types.BaseScenario }

// Run ...
func (s *RemoveOneRandomIPSecContainerUsingDocker) Run(si *types.SharedInfo) error {
	logrus.Debugf("Running Scenario: %v", s.Name)

	// TODO: state: running
	instanceListOpts := &client.ListOpts{
		Filters: map[string]interface{}{
			"name" + "_like": "%ipsec-router%",
		},
	}
	instanceCollection, err := si.Client.Instance.List(instanceListOpts)
	if err != nil {
		return err
	}

	err = utils.RemoveRandomInstanceUsingDocker(si, instanceCollection.Data)
	if err != nil {
		return err
	}

	return nil
}
