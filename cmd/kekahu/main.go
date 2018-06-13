package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bbengfort/kekahu"
	"github.com/joho/godotenv"
	"github.com/urfave/cli"
)

func main() {
	// Load the .env file if it exists
	godotenv.Load()

	// Instantiate the command line application
	// TODO: keep KeKahu version consistent with Kahu version
	app := cli.NewApp()
	app.Name = "kekahu"
	app.Version = "1.2"
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
					Value:  kekahu.DefaultInterval.String(),
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
					Value:  kekahu.DefaultKahuURL,
					EnvVar: "KEKAHU_URL",
				},
				cli.UintFlag{
					Name:   "verbosity",
					Usage:  "set log level from 0-4, lower is more verbose",
					Value:  2,
					EnvVar: "ALIA_VERBOSITY",
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
					Value:  kekahu.DefaultKahuURL,
					EnvVar: "KEKAHU_URL",
				},
				cli.UintFlag{
					Name:   "verbosity",
					Usage:  "set log level from 0-4, lower is more verbose",
					Value:  2,
					EnvVar: "ALIA_VERBOSITY",
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
					Value:  kekahu.DefaultKahuURL,
					EnvVar: "KEKAHU_URL",
				},
				cli.UintFlag{
					Name:   "verbosity",
					Usage:  "set log level from 0-4, lower is more verbose",
					Value:  2,
					EnvVar: "ALIA_VERBOSITY",
				},
			},
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
	// Set the logging level
	verbose := c.Uint("verbosity")
	kekahu.SetLogLevel(uint8(verbose))

	var err error
	if client, err = kekahu.New(c.String("key"), c.String("url")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

// Run the keep-alive server
func run(c *cli.Context) error {
	// Set the logging level
	verbose := c.Uint("verbosity")
	kekahu.SetLogLevel(uint8(verbose))

	delay, err := time.ParseDuration(c.String("delay"))
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if err := client.Run(delay, c.String("pid")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

// Sync the local peers.json file
func sync(c *cli.Context) error {
	// Set the logging level
	verbose := c.Uint("verbosity")
	kekahu.SetLogLevel(uint8(verbose))

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
