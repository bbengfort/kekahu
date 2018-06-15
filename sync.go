package kekahu

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	replicas := make([]*peers.Peer, 0)
	if err := json.NewDecoder(res.Body).Decode(&replicas); err != nil {
		return fmt.Errorf("could not parse Kahu response %s", err)
	}

	info := make(map[string]interface{})
	info["num_replicas"] = len(replicas)
	info["updated"] = time.Now()

	peers := &peers.Peers{
		Info:  info,
		Peers: replicas,
	}

	// Save the peers to disk at the specified path
	return peers.Dump(path)
}
