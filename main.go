package main

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
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
		cli.StringFlag{
			Name:   "digitalocean-access-token",
			Value:  "",
			EnvVar: "DIGITALOCEAN_ACCESS_TOKEN",
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
	digitaloceanAccessToken := c.String("digitalocean-access-token")
	startClusterSize := c.Int("start-cluster-size")
	minClusterSize := c.Int("min-cluster-size")
	maxClusterSize := c.Int("max-cluster-size")
	minWait := c.Int("min-wait")
	maxWait := c.Int("max-wait")
	seed := c.Int64("seed")

	if rancherURL == "" {
		err = fmt.Errorf("Rancher URL not specified")
		logrus.Errorf("error: %v", err)
		return err
	}

	if rancherAccessKey == "" {
		err = fmt.Errorf("Rancher Access Key not specified")
		logrus.Errorf("error: %v", err)
		return err
	}

	if rancherSecretKey == "" {
		err = fmt.Errorf("Rancher Secret Key not specified")
		logrus.Errorf("error: %v", err)
		return err
	}

	logrus.Debugf("rancher-url: %v", rancherURL)

	cm, err := NewChaosMonkey(rancherURL, rancherProjectID, rancherAccessKey, rancherSecretKey,
		digitaloceanAccessToken,
		startClusterSize, minClusterSize, maxClusterSize,
		minWait, maxWait, seed)
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
