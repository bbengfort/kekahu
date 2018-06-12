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
		go func(target *Neighbor) {
			defer group.Done()

			// Send the ping and record the duration
			sequence := k.network.Next(target.Hostname)
			latency, err := k.Ping(source, target.Hostname, target.IPAddr, sequence)
			if err != nil {
				k.echan <- err
				latency = time.Duration(0)
			}

			// Update the metrics
			k.network.Update(target.Hostname, latency)

			// Send the metrics back to Kahu if report is true
			if report {
				if err := k.latency(target.Hostname, latency); err != nil {
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
// specified host to the Kahu API.
func (k *KeKahu) latency(target string, latency time.Duration) error {
	// Compose JSON to post
	data := make(UpdateLatencyRequests, 0)
	update := new(UpdateLatencyRequest)
	update.Init(target, latency)
	data = append(data, update)

	// Create encoder and buffer
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		return fmt.Errorf("could not encode latency post body: %s", err)
	}

	// Create the request and post
	req, err := k.newRequest(http.MethodPost, LatencyEndpoint, buf)
	if err != nil {
		return err
	}

	// Perform the request
	res, err := k.doRequest(req)
	if err != nil {
		return err
	}

	// Read the response from Kahu
	defer res.Body.Close()
	info := make(UpdateLatencyResponses, 0)
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return fmt.Errorf("could not parse kahu response: %s", err)
	}

	// Log the response if in debug mode
	debug(
		"updated latency statistics from %d pings", len(info),
	)

	return nil
}

// Neighbors fetches the targets information from the Kahu server by performing
// a GET request against the /api/latency endpoint. It returns the source name
// of the requesting server as well as a list of target information.
func (k *KeKahu) Neighbors() (source string, targets []*Neighbor) {

	// Create the request and post
	req, err := k.newRequest(http.MethodGet, NeighborsEndpoint, nil)
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
	info := new(NeighborsResponse)
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

//===========================================================================
// Latency Request and Response Objects
//===========================================================================

// NeighborsResponse from the Kahu API with active targets and addresses to
// send pings and then post latencies for.
type NeighborsResponse struct {
	Source  string      `json:"source"`  // the unique name identifying the local host
	Targets []*Neighbor `json:"targets"` // a list of neighbors on the network to ping
}

// Neighbor represents a host on the network to send a ping to.
type Neighbor struct {
	Hostname string `json:"name"`       // unique name for the target host
	State    string `json:"state"`      // the current health of the target
	IPAddr   string `json:"ip_address"` // the external IP address of the target
	Domain   string `json:"domain"`     // the external domain name of the target
}

// UpdateLatencyRequests to POST multiple ping records to Kahu.
type UpdateLatencyRequests []*UpdateLatencyRequest

// UpdateLatencyRequest sends a record of a ping to the target to Kahu.
type UpdateLatencyRequest struct {
	Target  string  `json:"target"`  // unique name of target host
	Latency float64 `json:"latency"` // ping latency in milliseconds
	Timeout bool    `json:"timeout"` // whether or not the ping timed out
}

// Init the update latency request with a ping duration and target.
func (req *UpdateLatencyRequest) Init(target string, latency time.Duration) {
	req.Target = target

	if latency == 0 {
		req.Timeout = true
		req.Latency = 0
	} else {
		req.Timeout = false
		req.Latency = float64(latency) / float64(time.Millisecond)
	}
}

// UpdateLatencyResponses for each target posted in the request.
type UpdateLatencyResponses []*UpdateLatencyResponse

// UpdateLatencyResponse is returned from the Kahu API with details about the
// current distribution of latencies to the targets specified in the request.
type UpdateLatencyResponse struct {
	Source   string  `json:"source"`   // the current local host
	Target   string  `json:"target"`   // the target of the pings
	Messages uint64  `json:"messages"` // number of messages sent
	Timeouts uint64  `json:"timeouts"` // number of timeouts
	Fastest  float64 `json:"fastest"`  // fastest ping in ms
	Slowest  float64 `json:"slowest"`  // slowest ping in ms
	Mean     float64 `json:"mean"`     // average ping time in ms
	StdDev   float64 `json:"stddev"`   // standard deviation of ping time in ms
	Range    float64 `json:"range"`    // range of ping time in ms
}
