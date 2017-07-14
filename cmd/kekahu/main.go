package main

import (
	"fmt"
	"os"
	"syscall"
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
	app.Version = "0.3"
	app.Usage = "Keep alive client for the Kahu service"

	app.Commands = []cli.Command{
		{
			Name:   "start",
			Usage:  "run the kahu heartbeat program",
			Before: initClient,
			Action: start,
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
				cli.StringFlag{
					Name:   "p, pid",
					Usage:  "path to PID file or empty for standard location",
					Value:  "",
					EnvVar: "KEKAHU_PID_PATH",
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
		{
			Name:   "stop",
			Usage:  "if a pid file exists kill the kekahu service",
			Action: stop,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "p, pid",
					Usage:  "path to PID file or empty for standard location",
					Value:  "",
					EnvVar: "KEKAHU_PID_PATH",
				},
			},
		},
		{
			Name:   "reload",
			Usage:  "stop then start the kekahu service",
			Before: initClient,
			Action: reload,
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
				cli.StringFlag{
					Name:   "p, pid",
					Usage:  "path to PID file or empty for standard location",
					Value:  "",
					EnvVar: "KEKAHU_PID_PATH",
				},
			},
		},
		{
			Name:   "status",
			Usage:  "print the pid status of the kekahu service",
			Action: status,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "p, pid",
					Usage:  "path to PID file or empty for standard location",
					Value:  "",
					EnvVar: "KEKAHU_PID_PATH",
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
func start(c *cli.Context) error {
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

	if err := client.Sync(c.String("path")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	return nil
}

//===========================================================================
// Commands
//===========================================================================

// Send a kill signal to the process defined by the PID
func stop(c *cli.Context) error {
	pid := kekahu.NewPID(c.String("pid"))
	if err := pid.Load(); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	// Get the process from the os
	proc, err := os.FindProcess(pid.PID)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	// Kill the process
	fmt.Printf("stopping process %d\n", pid.PID)
	if err = proc.Signal(syscall.SIGTERM); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	// Wait for the state to change in the process
	state, err := proc.Wait()
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	fmt.Printf("process exited with %s", state)
	return nil
}

// Call the stop function, then call the start function.
// Don't use this probably.
func reload(c *cli.Context) error {
	stop(c)
	time.Sleep(1 * time.Second)
	start(c)
	return nil
}

// Indicate the status of the service based on the pid
func status(c *cli.Context) error {
	pid := kekahu.NewPID(c.String("pid"))
	if err := pid.Load(); err != nil {
		fmt.Printf("kekahu is not running; no pid exists at %s\n", pid.Path())
	} else {
		fmt.Printf("kekahu is running with pid %d from %s\n", pid.PID, pid.Path())
	}

	return nil
}
