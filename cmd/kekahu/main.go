package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bbengfort/kekahu"
	"github.com/joho/godotenv"
	"github.com/koding/multiconfig"
	"github.com/urfave/cli"
)

func main() {
	// Load the .env file if it exists
	godotenv.Load()

	// Instantiate the command line application
	// TODO: keep KeKahu version consistent with Kahu version
	app := cli.NewApp()
	app.Name = "kekahu"
	app.Version = kekahu.PackageVersion
	app.Usage = "Keep alive client for the Kahu service"

	app.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "run the kahu heartbeat program",
			Before: initClient,
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "d, delay",
					Usage:  "parsable duration of the delay between heartbeats",
					EnvVar: "KEKAHU_INTERVAL",
				},
				cli.StringFlag{
					Name:   "k, key",
					Usage:  "api key of the local host",
					EnvVar: "KEKAHU_API_KEY",
				},
				cli.StringFlag{
					Name:   "u, url",
					Usage:  "kahu service url if different from default",
					EnvVar: "KEKAHU_URL",
				},
				cli.IntFlag{
					Name:   "verbosity",
					Usage:  "set log level from 0-4, lower is more verbose",
					EnvVar: "KEKAHU_VERBOSITY",
				},
			},
		},
		{
			Name:   "sync",
			Usage:  "synchronize the local peers definition",
			Before: initClient,
			Action: sync,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "p, path",
					Usage:  "path to write the peers.json file (if empty writes to home directory)",
					Value:  "",
					EnvVar: "PEERS_PATH",
				},
				cli.StringFlag{
					Name:   "k, key",
					Usage:  "api key of the local host",
					EnvVar: "KEKAHU_API_KEY",
				},
				cli.StringFlag{
					Name:   "u, url",
					Usage:  "kahu service url",
					EnvVar: "KEKAHU_URL",
				},
				cli.IntFlag{
					Name:   "verbosity",
					Usage:  "set log level from 0-4, lower is more verbose",
					EnvVar: "KEKAHU_VERBOSITY",
				},
			},
		},
		{
			Name:   "ping",
			Usage:  "ping another kekahu server to determine latency",
			Before: initClient,
			Action: ping,
			Flags: []cli.Flag{
				cli.Uint64Flag{
					Name:  "n, number",
					Usage: "number of pings to send",
					Value: 1,
				},
				cli.StringFlag{
					Name:   "k, key",
					Usage:  "api key of the local host",
					EnvVar: "KEKAHU_API_KEY",
				},
				cli.StringFlag{
					Name:   "u, url",
					Usage:  "kahu service url",
					EnvVar: "KEKAHU_URL",
				},
				cli.IntFlag{
					Name:   "verbosity",
					Usage:  "set log level from 0-4, lower is more verbose",
					EnvVar: "KEKAHU_VERBOSITY",
				},
			},
		},
		{
			Name:   "config",
			Usage:  "print the current KeKahu configuration",
			Action: config,
		},
		{
			Name:   "health",
			Usage:  "print out KeKahu's view of the system status",
			Action: health,
		},
	}

	// Run the CLI program
	app.Run(os.Args)
}

//===========================================================================
// Commands
//===========================================================================

var client *kekahu.KeKahu

// Initialize the kekahu client
func initClient(c *cli.Context) error {
	config := &kekahu.Config{
		Interval:  c.String("delay"),
		URL:       c.String("url"),
		Verbosity: c.Int("verbosity"),
		APIKey:    c.String("key"),
	}

	var err error
	if client, err = kekahu.New(config); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

// Print the current configuration of KeKahu
func config(c *cli.Context) error {
	conf := new(kekahu.Config)
	if err := conf.Load(); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if path, err := kekahu.FindConfigPath(); err == nil {
		fmt.Println("\nConfig File\n-----------")
		fmt.Printf("  %s\n\n", path)
	}

	fmt.Println("JSON Configuration\n------------------")
	fmt.Println(string(data))
	fmt.Println("\nEnvironment Variables\n---------------------")
	env := &multiconfig.EnvironmentLoader{Prefix: "KEKAHU", CamelCase: true}
	env.PrintEnvs(conf)
	return nil
}

// Run the keep-alive server
func run(c *cli.Context) error {
	if err := client.Run(); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

// Sync the local peers.json file
func sync(c *cli.Context) error {
	if err := client.Sync(c.String("path")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

// Ping the remote host to determine latency
func ping(c *cli.Context) error {
	kekahu.SetLogLevel(kekahu.Silent)

	// Send the pings
	if err := client.SendNPings(c.Uint64("number")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	// Report the averages
	metrics := client.Metrics()
	data, _ := json.MarshalIndent(metrics, "", "  ")
	fmt.Println(string(data))

	return nil
}

// Perform a health check and view the system status
func health(c *cli.Context) error {
	status, err := kekahu.HealthCheck(true)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	data, err := status.Dump(2)
	if err != nil {
		return cli.NewExitError("couldn't dump status to JSON", 1)
	}

	fmt.Println(string(data))
	return nil
}
