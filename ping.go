package kekahu

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// SendNPings is a helper function that looks up the neighbors from the API,
// then sends N pings to them, keeping track of internal metrics. This method
// is meant to be run from the command line, so it doesn't use the standard
// logger but instead directly prints to the command line.
func (k *KeKahu) SendNPings(n uint64) error {
	// Fetch the source and the targets. If there is no response, or no targets
	// then return, we're not going to be doing any work!
	source, targets := k.Neighbors()
	if source == "" || targets == nil || len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "no active neighbors to ping")
		return nil
	}

	fmt.Fprintf(os.Stderr, "sending %d pings to %d neighbors ...\n", n, len(targets))

	// Execute the pings against each of the returned sources
	group := new(sync.WaitGroup)
	for i := uint64(0); i < n; i++ {
		for _, target := range targets {
			group.Add(1)
			go func(target *Neighbor) {
				defer group.Done()

				// Send the ping and record the duration
				sequence := k.network.Next(target.Hostname)
				latency, err := k.Ping(source, target.Hostname, target.IPAddr, sequence)
				if err != nil {
					fmt.Fprint(os.Stderr, "x")
					latency = time.Duration(0)
				} else {
					fmt.Fprint(os.Stderr, ".")
				}

				// Update the metrics
				k.network.Update(target.Hostname, latency)

			}(target)
		}
	}

	// Wait for all pings to complete and clear stderr buffer
	group.Wait()
	fmt.Fprint(os.Stderr, "\n")
	return nil
}
