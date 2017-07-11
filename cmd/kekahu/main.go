package main

import (
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
	app := cli.NewApp()
	app.Name = "kekahu"
	app.Version = "0.1"
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
					Value:  "1m",
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
	var err error
	if client, err = kekahu.New(c.String("key"), c.String("url")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

// Run the keep-alive server
func run(c *cli.Context) error {
	delay, err := time.ParseDuration(c.String("delay"))
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if err := client.Run(delay); err != nil {
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
