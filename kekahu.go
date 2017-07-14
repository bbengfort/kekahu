package kekahu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bbengfort/x/net"
	"github.com/bbengfort/x/peers"
)

// DefaultKahuURL to communicate with the heartbeat service
const DefaultKahuURL = "https://kahu.herokuapp.com"

// DefaultAPITimeout to wait for responses from the Kahu server
const DefaultAPITimeout = time.Second * 5

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

	kekahu := &KeKahu{url: url, apikey: api, client: client}
	return kekahu, nil
}

//===========================================================================
// KeKahu Struct and Methods
//===========================================================================

// KeKahu is the Kahu client that performs service requests to Kahu. It's
// state manages the URL and API Key that should be passed in via New()
type KeKahu struct {
	url    *url.URL      // URL of the Kahu service
	apikey string        // API Key to access the Kahu service with
	client *http.Client  // HTTP client to perform requests
	pid    *PID          // Reference to current PID file
	delay  time.Duration // Interval between Heartbeats
	echan  chan error    // Channel to listen for non-fatal errors on
	done   chan bool     // Channel to listen for shutdown signal
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

	// Run the OS signal handlers
	go k.signalHandler()

	// Start the heartbeat parameters
	k.delay = delay
	k.echan = make(chan error)
	k.done = make(chan bool)
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
func (k *KeKahu) Shutdown() error {
	info("shutting down")
	k.done <- true // Shutdown the running listener
	return k.pid.Free()
}

// Sync the peers.json file from Kahu. If no path is specified then the peers
// file will be synced to the path specified by the peers package, most
// likely ~/.fluidfs/peers.json unless the $PEERS_PATH is set.
func (k *KeKahu) Sync(path string) error {
	// Determine the path to synchronize the peers to.
	if path == "" {
		path = peers.Path()
	}

	// Create the request to the Kahu service
	req, err := k.newRequest(http.MethodGet, "/api/replicas", nil)
	if err != nil {
		return err
	}

	// Perform the GET request
	res, err := k.doRequest(req)
	if err != nil {
		return fmt.Errorf("kahu error: %s", err)
	}

	// Ensure connection is closed on complete
	defer res.Body.Close()

	// Parse the JSON into a peers struct
	peers := new(peers.Peers)
	if err := json.NewDecoder(res.Body).Decode(&peers); err != nil {
		return fmt.Errorf("could not parse Kahu response %s", err)
	}

	// Save the peers to disk at the specified path
	return peers.Dump(path)
}

// Heartbeat sends a heartbeat POST message to the Kahu endpoint, notifying
// the management service that the localhost is alive and well. It then
// schedules the next heartbeat message to be sent after the specified delay.
//
// Any http errors that occur are sent on the error channel to be logged by
// the application. These errors are not fatal and do not cause the heartbeat
// interval to stop.
func (k *KeKahu) Heartbeat() {
	trace("executing heartbeat")

	// Schedule the next heartbeat after this function is complete
	defer time.AfterFunc(k.delay, k.Heartbeat)

	// First collect the public IP address of the host
	ipaddr, err := net.PublicIP()
	if err != nil {
		k.echan <- fmt.Errorf("could not get public IP: %s", err)
		return
	}

	// Compose JSON to post
	data := make(map[string]string)
	data["ip_address"] = ipaddr

	// Create encoder and buffer
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(data); err != nil {
		k.echan <- fmt.Errorf("could not encode heartbeat post body: %s", err)
		return
	}

	// Create the request and post
	req, err := k.newRequest(http.MethodPost, "/api/heartbeat", buf)
	if err != nil {
		k.echan <- fmt.Errorf("could not create request: %s", err)
		return
	}

	// Perform the request
	res, err := k.doRequest(req)
	if err != nil {
		k.echan <- fmt.Errorf("could make http request: %s", err)
		return
	}

	// Read the response from Kahu
	defer res.Body.Close()
	hb := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&hb); err != nil {
		k.echan <- fmt.Errorf("could not parse kahu response: %s", err)
		return
	}

	// Log the response if in debug mode
	debug(
		"updated %s (%s) success: %t\n",
		hb["machine"].(string),
		hb["ipaddr"].(string),
		hb["success"].(bool),
	)
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
		return nil, err
	}

	// Resolve the URL reference
	url := k.url.ResolveReference(ep)

	// Construct the request
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, err
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

//===========================================================================
// OS Signal Handlers
//===========================================================================

func (k *KeKahu) signalHandler() {
	// Make signal channel and register notifiers for Interupt and Terminate
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, syscall.SIGTERM)

	// Block until we receive a signal on the channel
	<-sigchan

	// Defer the clean exit until the end of the function
	defer os.Exit(0)

	// Shutdown now that we've received the signal
	err := k.Shutdown()
	if err != nil {
		msg := fmt.Sprintf("shutdown error: %s", err.Error())
		log.Fatal(msg)
		os.Exit(1)
	}
}
