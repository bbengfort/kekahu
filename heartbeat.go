package kekahu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

	// First collect the public IP address of the host
	ipaddr, err := net.PublicIP()
	if err != nil {
		k.echan <- fmt.Errorf("could not get public IP: %s", err)
		return
	}

	// Compose JSON to post
	debug("public ip address is %s", ipaddr)
	data := make(map[string]string)
	data["ip_address"] = ipaddr

	// Create encoder and buffer
	buf := new(bytes.Buffer)
	if err = json.NewEncoder(buf).Encode(data); err != nil {
		k.echan <- fmt.Errorf("could not encode heartbeat post body: %s", err)
		return
	}

	// Create the request and post
	req, err := k.newRequest(http.MethodPost, "/api/heartbeat", buf)
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
	hb, err := k.parseResponse(res)
	if err != nil {
		k.echan <- err
	}

	// Get the values from the response
	success := hb["success"].(bool)
	active := hb["active"].(bool)

	// Log the response if in debug mode
	debug(
		"updated %s (%s) success: %t active: %t\n",
		hb["machine"].(string), hb["ipaddr"].(string), success, active,
	)

	// If we're active and the heartbeat was successful then run ping routine
	// to collect latency measurements from all other active hosts.
	if success && active {
		go k.Latency(true)
	}

}
