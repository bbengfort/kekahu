package kekahu

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bbengfort/x/peers"
)

// Sync the peers.json file from Kahu. If no path is specified then the peers
// file will be synced to the path specified by the peers package, most
// likely ~/.fluidfs/peers.json unless the $PEERS_PATH is set.
func (k *KeKahu) Sync(path string) error {
	// Determine the path to synchronize the peers to.
	if path == "" {
		path = k.config.PeersPath
	}

	// Create the request to the Kahu service
	req, err := k.newRequest(http.MethodGet, ReplicasEndpoint, nil)
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
