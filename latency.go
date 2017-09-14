package kekahu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Latency is a hard working method that sends a request to the Kahu server for
// all targets associated with the current host, then sends a ping request to
// each of them, measuring the latency of the ping. It then reports the results
// of the pings back to Kahu.
//
// Latency is called routinely from the heartbeat method, and will only be
// executed if the host is active and the heartbeat was successful.
func (k *KeKahu) Latency(report bool) {
	trace("executing latency measures to neighbors")

	// Fetch the source and the targets. If there is no response, or no targets
	// then return, we're not going to be doing any work!
	source, targets := k.Neighbors()
	if source == "" || targets == nil {
		return
	}

	// Execute the pings against each of the returned sources
	group := new(sync.WaitGroup)
	for _, target := range targets {
		group.Add(1)
		go func(target map[string]string) {
			defer group.Done()

			// Send the ping and record the duration
			sequence := k.network.Next(target["hostname"])
			latency, err := k.Ping(source, target["hostname"], target["addr"], sequence)
			if err != nil {
				k.echan <- err
				latency = time.Duration(0)
			}

			// Update the metrics
			k.network.Update(target["hostname"], latency)

			// Send the metrics back to Kahu if report is true
			if report {
				if err := k.latency(target["hostname"]); err != nil {
					k.echan <- err
					return
				}
			}

		}(target)
	}

	// Wait for all pings to complete
	group.Wait()
}

// latency is a helper method to send the latency information for the
// specified host to KeKahu -- sending the summary statistics for each.
func (k *KeKahu) latency(host string) error {
	// Compose JSON to post
	report := k.network.Serialize(host)

	// Create encoder and buffer
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(report); err != nil {
		return fmt.Errorf("could not encode latency post body: %s", err)
	}

	// Create the request and post
	req, err := k.newRequest(http.MethodPost, "/api/latency", buf)
	if err != nil {
		return err
	}

	// Perform the request
	res, err := k.doRequest(req)
	if err != nil {
		return err
	}

	// Read the response from Kahu
	info, err := k.parseResponse(res)
	if err != nil {
		return err
	}

	// Log the response if in debug mode
	debug(
		"updated latency statics from %s to %s (success: %t)",
		info["source"].(string), info["target"].(string), info["success"].(bool),
	)

	return nil
}

// Neighbors fetches the targets information from the Kahu server by performing
// a GET request against the /api/latency endpoint. It returns the source name
// of the requesting server as well as a list of target information.
func (k *KeKahu) Neighbors() (source string, targets []map[string]string) {

	type Response struct {
		Source  string
		Targets []map[string]string
	}

	// Create the request and post
	req, err := k.newRequest(http.MethodGet, "/api/latency", nil)
	if err != nil {
		k.echan <- fmt.Errorf("could not create request: %s", err)
		return "", nil
	}

	// Perform the request
	res, err := k.doRequest(req)
	if err != nil {
		k.echan <- fmt.Errorf("could make http request: %s", err)
		return "", nil
	}

	// Read the response from Kahu
	defer res.Body.Close()
	info := new(Response)
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		k.echan <- fmt.Errorf("could not parse kahu response: %s", err)
		return "", nil
	}

	return info.Source, info.Targets
}

// Metrics returns access to the latency metrics so that the command line
// can print them out on demand.
func (k *KeKahu) Metrics() map[string]map[string]interface{} {
	return k.network.Report()
}
