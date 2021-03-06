package kekahu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"
)

// PackageVersion of the KeKahu application
const PackageVersion = "1.6"

// Endpoints on the Kahu RESTful API
const (
	HeartbeatEndpoint = "/api/heartbeat/"
	LatencyEndpoint   = "/api/latency/"
	NeighborsEndpoint = "/api/latency/neighbors/"
	ReplicasEndpoint  = "/api/replicas/"
	HealthEndpoint    = "/api/health/"
)

//===========================================================================
// Package Initialization
//===========================================================================

// Initialize the package and random numbers, etc.
func init() {
	// Set the random seed to something different each time.
	rand.Seed(time.Now().UnixNano())

	// Initialize our debug logging with our prefix
	logger = log.New(os.Stdout, "[kekahu] ", log.Lmicroseconds)
}

//===========================================================================
// Kekahu Client
//===========================================================================

// New constructs a KeKahu client from an api key and url pair. If a URL is
// not specified (e.g. an empty string) then the DefaultKahuURL is used. This
// function returns an error if no API key is provided.
func New(options *Config) (*KeKahu, error) {
	// Create default configuration
	config := new(Config)
	if err := config.Load(); err != nil {
		return nil, err
	}

	// Update the configuration from the options
	if err := config.Update(options); err != nil {
		return nil, err
	}

	// Set the logging level
	SetLogLevel(uint8(config.Verbosity))

	// Create the HTTP client
	timeout, _ := config.GetAPITimeout()
	client := &http.Client{Timeout: timeout}

	// Create the Echo server
	server := new(Server)
	server.Init("", "")

	// Create the ping latencies map
	network := new(Network)
	network.Init()

	kekahu := &KeKahu{config: config, client: client, server: server, network: network}
	return kekahu, nil
}

//===========================================================================
// KeKahu Struct and Methods
//===========================================================================

// KeKahu is the Kahu client that performs service requests to Kahu. It's
// state manages the URL and API Key that should be passed in via New()
type KeKahu struct {
	config  *Config       // KeKahu service configuration
	client  *http.Client  // HTTP client to perform requests
	server  *Server       // Echo server to respond to ping requests
	delay   time.Duration // Interval between Heartbeats
	jitter  time.Duration // Range before and after interval to jitter the heartbeat
	echan   chan error    // Channel to listen for non-fatal errors on
	done    chan bool     // Channel to listen for shutdown signal
	network *Network      // Ping latency to other peers in the network
}

// Run the keep-alive heartbeat service with the interval specified. The
// service will log any http errors to to standard out and any other errors
// as fatal, exiting the program - otherwise it will continue running until
// it is shutdown by OS signals.
func (k *KeKahu) Run() (err error) {
	// Initialize the listener channels
	k.echan = make(chan error)
	k.done = make(chan bool, 1)

	// Run the OS signal handlers
	go signalHandler(k.Shutdown)

	// Start the local echo server
	if err = k.server.Run(k.echan); err != nil {
		return err
	}

	// Start the heartbeat
	k.delay, err = k.config.GetInterval()
	if err != nil {
		return err
	}
	k.jitter, err = k.config.GetJitter()
	if err != nil {
		return err
	}
	go k.Heartbeat()

	// Wait for any errors and log them
outer:
	for {
		select {
		case err := <-k.echan:
			warne(err)
		case done := <-k.done:
			if done {
				break outer
			}
		}
	}

	return nil
}

// Shutdown the KeKahu service and clean up the PID file.
func (k *KeKahu) Shutdown() (err error) {
	info("shutting down the kekahu service")

	// Shutdown the server
	if err = k.server.Shutdown(); err != nil {
		k.echan <- err
	}

	// Notify the run method we're done
	// NOTE: do this last or the cleanup proceedure won't be done.
	k.done <- true
	return nil
}

//===========================================================================
// Internal Methods
//===========================================================================

// Construct a URL from the given endpoint and add API key header to the
// http request -- all things required to perform an Kahu API request.
func (k *KeKahu) newRequest(method, endpoint string, body io.Reader) (*http.Request, error) {

	// Parse the endpoint
	ep, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("could not parse endpoint: %s", err)
	}

	// Resolve the URL reference
	baseURL, err := k.config.GetURL()
	if err != nil {
		return nil, err
	}
	url := baseURL.ResolveReference(ep)

	// Construct the request
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	// Add the headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", k.config.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	trace("created %s request to %s", method, url)
	return req, nil
}

// Do the request and also return an error for non 200 status
func (k *KeKahu) doRequest(req *http.Request) (*http.Response, error) {
	res, err := k.client.Do(req)
	if err != nil {
		err = fmt.Errorf("could not make http request: %s", err)
		return res, err
	}

	debug("%s %s %s", req.Method, req.URL.String(), res.Status)

	// Check the status from the client
	if res.StatusCode < 200 || res.StatusCode > 299 {
		res.Body.Close()
		return res, fmt.Errorf("could not access Kahu service: %s", res.Status)
	}

	return res, nil
}

// Encode a generic request to the Kahu API into a buffer with JSON data
func encodeRequest(data interface{}) (body io.Reader, err error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		return nil, fmt.Errorf("could not encode request: %s", err)
	}
	return buf, nil
}

// Parse a generic response from the Kahu API into a JSON map interface object
func parseResponse(res *http.Response) (map[string]interface{}, error) {
	defer res.Body.Close()
	info := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("could not parse kahu response: %s", err)
	}

	return info, nil
}
