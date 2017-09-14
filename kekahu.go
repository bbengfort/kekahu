package kekahu

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"
)

// DefaultKahuURL to communicate with the heartbeat service
const DefaultKahuURL = "https://kahu.herokuapp.com"

// DefaultAPITimeout to wait for responses from the Kahu server
const DefaultAPITimeout = time.Second * 5

// DefaultInterval between heartbeat messages
const DefaultInterval = time.Minute * 1

//===========================================================================
// Package Initialization
//===========================================================================

// Initialize the package and random numbers, etc.
func init() {
	// Set the random seed to something different each time.
	rand.Seed(time.Now().Unix())

	// Initialize our debug logging with our prefix
	logger = log.New(os.Stdout, "[kekahu] ", log.Lmicroseconds)
}

//===========================================================================
// Kekahu Client
//===========================================================================

// New constructs a KeKahu client from an api key and url pair. If a URL is
// not specified (e.g. an empty string) then the DefaultKahuURL is used. This
// function returns an error if no API key is provided.
func New(api, kahuURL string) (*KeKahu, error) {

	// An API key is required.
	if api == "" {
		return nil, errors.New("an API key is required to access the kahu service")
	}

	// Use the DefaultKahuURL if necessary.
	if kahuURL == "" {
		kahuURL = DefaultKahuURL
	}

	// Parse the URL
	url, err := url.Parse(kahuURL)
	if err != nil {
		return nil, err
	}

	// Create the HTTP client
	client := &http.Client{Timeout: DefaultAPITimeout}

	// Create the Echo server
	server := new(Server)
	server.Init("", "")

	// Create the ping latencies map
	network := new(Network)
	network.Init()

	kekahu := &KeKahu{url: url, apikey: api, client: client, server: server, network: network}
	return kekahu, nil
}

//===========================================================================
// KeKahu Struct and Methods
//===========================================================================

// KeKahu is the Kahu client that performs service requests to Kahu. It's
// state manages the URL and API Key that should be passed in via New()
type KeKahu struct {
	url     *url.URL      // URL of the Kahu service
	apikey  string        // API Key to access the Kahu service with
	client  *http.Client  // HTTP client to perform requests
	server  *Server       // Echo server to respond to ping requests
	pid     *PID          // Reference to current PID file
	delay   time.Duration // Interval between Heartbeats
	echan   chan error    // Channel to listen for non-fatal errors on
	done    chan bool     // Channel to listen for shutdown signal
	network *Network      // Ping latency to other peers in the network
}

// Run the keep-alive heartbeat service with the interval specified. The
// service will log any http errors to to standard out and any other errors
// as fatal, exiting the program - otherwise it will continue running until
// it is shutdown by OS signals.
func (k *KeKahu) Run(delay time.Duration, pid string) error {
	// Create the PID file
	k.pid = NewPID(pid)
	if err := k.pid.Save(); err != nil {
		return err
	}
	debug("pid file saved to %s", k.pid.Path())

	// Initialize the listener channels
	k.echan = make(chan error)
	k.done = make(chan bool, 1)

	// Run the OS signal handlers
	go signalHandler(k.Shutdown)

	// Start the local echo server
	if err := k.server.Run(k.echan); err != nil {
		return err
	}

	// Start the heartbeat
	k.delay = delay
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

	// Free the PID file
	if err = k.pid.Free(); err != nil {
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
	url := k.url.ResolveReference(ep)

	// Construct the request
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	// Add the headers
	req.Header.Set("X-Api-Key", k.apikey)
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
	if res.StatusCode != 200 {
		res.Body.Close()
		return res, fmt.Errorf("could not access Kahu service: %s", res.Status)
	}

	return res, nil
}

// Parse the response into a JSON map interface object
func (k *KeKahu) parseResponse(res *http.Response) (map[string]interface{}, error) {
	defer res.Body.Close()
	info := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("could not parse kahu response: %s", err)
	}

	return info, nil
}
