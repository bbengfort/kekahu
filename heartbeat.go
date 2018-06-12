package kekahu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bbengfort/x/net"
)

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

	// Compose JSON to post
	data := new(HeartbeatRequest)
	if err := data.Load(); err != nil {
		k.echan <- err
		return
	}

	// Create encoder and buffer
	body, err := encodeRequest(data)
	if err != nil {
		k.echan <- err
		return
	}

	// Create the request and post
	req, err := k.newRequest(http.MethodPost, HeartbeatEndpoint, body)
	if err != nil {
		k.echan <- err
		return
	}

	// Perform the request
	res, err := k.doRequest(req)
	if err != nil {
		k.echan <- err
		return
	}

	// Read the response from Kahu
	hb := new(HeartbeatResponse)
	if err := hb.Parse(res); err != nil {
		k.echan <- err
		return
	}

	// Log the response if in debug mode
	debug("%s", hb)

	// If we're active and the heartbeat was successful then run ping routine
	// to collect latency measurements from all other active hosts.
	if hb.Success && hb.Active {
		go k.Latency(true)
	}

}

//===========================================================================
// Heartbeat JSON Resquest and Response Objects
//===========================================================================

// HeartbeatRequest JSON data structure to POST to Kahu /api/heartbeat/
type HeartbeatRequest struct {
	IPAddr   string `json:"ip_address"`
	Hostname string `json:"hostname"`
}

// Load the HeartbeatRequest by looking up the current hostname and external
// IP address using system utilities.
func (hb *HeartbeatRequest) Load() (err error) {
	// First collect the public IP address of the host
	hb.IPAddr, err = net.PublicIP()
	if err != nil {
		return fmt.Errorf("could not get public IP: %s", err)
	}
	debug("public ip address is %s", hb.IPAddr)

	// Then collect the hostname of the host
	hb.Hostname, err = os.Hostname()
	if err != nil {
		return fmt.Errorf("could not get hostname: %s", err)
	}
	debug("hostname is %s", hb.Hostname)

	return nil
}

// HeartbeatResponse JSON data struct to parse Kahu /api/heartbeat/ response.
type HeartbeatResponse struct {
	Success bool   `json:"success"`
	Replica string `json:"replica"`
	Active  bool   `json:"active"`
}

// Parse the Kahu heartbeat HTTP response body
func (hb *HeartbeatResponse) Parse(res *http.Response) error {
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&hb); err != nil {
		return fmt.Errorf("could not parse kahu response: %s", err)
	}

	return nil
}

func (hb *HeartbeatResponse) String() string {
	return fmt.Sprintf(
		"updated %s success: %t active: %t",
		hb.Replica, hb.Success, hb.Active,
	)
}
