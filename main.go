package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/leodotcloud/chaos-monkey/types"
	"github.com/urfave/cli"
)

// VERSION of the binary, that can be changed during build
var VERSION = "v0.0.0-dev"

func main() {
	app := cli.NewApp()
	app.Name = "chaos-monkey"
	app.Version = VERSION
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "rancher-url",
			Value:  "",
			EnvVar: "RANCHER_URL",
		},
		cli.StringFlag{
			Name:   "rancher-access-key",
			Value:  "",
			EnvVar: "RANCHER_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "rancher-project-id",
			Value:  "1a5",
			EnvVar: "RANCHER_PROJECT_ID",
		},
		cli.StringFlag{
			Name:   "rancher-secret-key",
			Value:  "",
			EnvVar: "RANCHER_SECRET_KEY",
		},
		cli.IntFlag{
			Name:  "start-cluster-size",
			Value: DefaultStartClusterSize,
		},
		cli.IntFlag{
			Name:  "min-cluster-size",
			Value: DefaultMinimumClusterSize,
		},
		cli.IntFlag{
			Name:  "max-cluster-size",
			Value: DefaultMaximumClusterSize,
		},
		cli.IntFlag{
			Name:  "min-wait",
			Value: DefaultMinWaitTime,
		},
		cli.IntFlag{
			Name:  "max-wait",
			Value: DefaultMaxWaitTime,
		},
		cli.Int64Flag{
			Name: "seed",
		},
		cli.BoolFlag{
			Name:   "use-digitalocean",
			Usage:  "Use DigitalOcean Cloud Provider",
			EnvVar: "USE_DIGITALOCEAN",
		},
		cli.StringFlag{
			Name:   "digitalocean-access-token",
			EnvVar: "DIGITALOCEAN_ACCESS_TOKEN",
		},
		cli.BoolFlag{
			Name:   "use-aws",
			Usage:  "Use AWS Cloud Provider",
			EnvVar: "USE_AWS",
		},
		cli.StringFlag{
			Name:   "aws-secret-key-id",
			EnvVar: "AWS_SECRET_KEY_ID",
		},
		cli.StringFlag{
			Name:   "aws-secret-access-key",
			EnvVar: "AWS_SECRET_ACCESS_KEY",
		},
		cli.BoolFlag{
			Name:   "use-packet",
			Usage:  "Use Packet Cloud Provider",
			EnvVar: "USE_PACKET",
		},
		cli.StringFlag{
			Name:   "packet-project-id",
			EnvVar: "PACKET_PROJECT_ID",
		},
		cli.StringFlag{
			Name:   "packet-token",
			EnvVar: "PACKET_TOKEN",
		},
		cli.BoolFlag{
			Name:   "disable-host-add-scenario",
			Usage:  "Disable adding of Hosts during testing",
			EnvVar: "DISALBLE_HOST_ADD_SCENARIO",
		},
		cli.BoolFlag{
			Name:   "disable-host-del-scenario",
			Usage:  "Disable deleting of Hosts during testing",
			EnvVar: "DISALBLE_HOST_DEL_SCENARIO",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Turn on debug logging",
		},
	}
	app.Action = run
	app.Run(os.Args)
}

func run(c *cli.Context) error {
	var err error
	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	rancherURL := c.String("rancher-url")
	rancherAccessKey := c.String("rancher-access-key")
	rancherSecretKey := c.String("rancher-secret-key")
	rancherProjectID := c.String("rancher-project-id")
	minWait := c.Int("min-wait")
	maxWait := c.Int("max-wait")
	seed := c.Int64("seed")

	sharedInfo := &types.SharedInfo{
		UseDigitalOcean:         c.Bool("use-digitalocean"),
		DigitalOceanAccessToken: c.String("digitalocean-access-token"),
		UseAWS:                  c.Bool("use-aws"),
		AWSAccessKeyID:          c.String("aws-access-key-id"),
		AWSSecretAccessKey:      c.String("aws-secret-access-key"),
		UsePacket:               c.Bool("use-packet"),
		PacketProjectID:         c.String("packet-project-id"),
		PacketToken:             c.String("packet-token"),
		DisableAddHostScenario:  c.Bool("disable-host-add-scenario"),
		DisableDelHostScenario:  c.Bool("disable-host-del-scenario"),
		StartClusterSize:        c.Int("start-cluster-size"),
		MinClusterSize:          c.Int("min-cluster-size"),
		MaxClusterSize:          c.Int("max-cluster-size"),
	}

	if rancherURL == "" {
		err = fmt.Errorf("Rancher URL not specified")
		logrus.Errorf("error: %v", err)
		return err
	}

	//if rancherAccessKey == "" {
	//	err = fmt.Errorf("Rancher Access Key not specified")
	//	logrus.Errorf("error: %v", err)
	//	return err
	//}

	//if rancherSecretKey == "" {
	//	err = fmt.Errorf("Rancher Secret Key not specified")
	//	logrus.Errorf("error: %v", err)
	//	return err
	//}

	logrus.Debugf("rancher-url: %v", rancherURL)

	cm, err := NewChaosMonkey(rancherURL, rancherProjectID, rancherAccessKey, rancherSecretKey,
		minWait, maxWait, seed,
		sharedInfo)
	if err != nil {
		logrus.Errorf("error creating chaos monkey: %v", err)
		return err
	}

	if err := cm.Run(); err != nil {
		logrus.Errorf("error running chaos monkey: %v", err)
		return err
	}

	//<-make(chan struct{})
	return nil
}
