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
			Usage:  "run the throughput experiment",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "i, interval",
					Usage:  "parsable duration of heartbeat interval",
					Value:  "1m",
					EnvVar: "KEKAHU_INTERVAL",
				},
				cli.StringFlag{
					Name:   "k, key",
					Usage:  "api key of the local host",
					EnvVar: "KEKAHU_API_KEY",
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

// Run the keep-alive server
func run(c *cli.Context) error {
	key := c.String("key")
	duration, err := time.ParseDuration(c.String("interval"))
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	kekahu.Heartbeat(key, duration)
	return nil
}
