package kekahu

import "net/http"

// Health reports the system status to Kahu using the system HealthCheck.
func (k *KeKahu) Health() {
	trace("executing system health check")

	// Get the health check form the system
	health, err := HealthCheck(true)
	if err != nil {
		// TODO: should we really be logging these errors if we're going to fail?
		k.echan <- err
		return
	}

	// Create encoder and buffer
	body, err := encodeRequest(health)
	if err != nil {
		k.echan <- err
		return
	}

	// Create the request and post
	req, err := k.newRequest(http.MethodPost, HealthEndpoint, body)
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

	// Log the response if in debug mode
	debug("health status report: %d %s", res.StatusCode, res.Status)

}
