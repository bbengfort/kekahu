package kekahu

import (
	"sync"
	"time"

	"github.com/bbengfort/x/stats"
)

// Network keeps track of latency statistics between peers when running the
// echo ping protocol on each heartbeat. This struct serves primarily as a
// thread-safe access to a map of hostnames to stats.Benchmark objects.
type Network struct {
	sync.RWMutex
	metrics map[string]*stats.Benchmark
}

// Init the internal mapping of metrics objects.
func (n *Network) Init() {
	n.Lock()
	defer n.Unlock()
	n.metrics = make(map[string]*stats.Benchmark)
}

// Update the network with the latencies for the given host.
func (n *Network) Update(host string, latencies ...time.Duration) {
	n.Lock()
	defer n.Unlock()
	metrics := n.get(host)
	metrics.Update(latencies...)
}

// Next returns the next sequence id for the specified host.
func (n *Network) Next(host string) uint64 {
	n.RLock()
	defer n.RUnlock()
	metrics := n.get(host)
	return metrics.N() + 1
}

// Serialize the benchmark for a specific host to post to Kahu. Note that
// this returns float values in milliseconds for timing purposes.
func (n *Network) Serialize(host string) map[string]interface{} {
	n.RLock()
	defer n.RUnlock()

	// Instantiate data structures
	metrics := n.get(host)
	data := make(map[string]interface{})

	// Add information in milliseconds to the data structure
	data["target"] = host
	data["messages"] = metrics.N()
	data["timeouts"] = metrics.Timeouts()
	data["total"] = metrics.Statistics.Total() * 1000.0
	data["mean"] = metrics.Statistics.Mean() * 1000.0
	data["stddev"] = metrics.Statistics.StdDev() * 1000.0
	data["variance"] = metrics.Statistics.Variance() * 1000.0
	data["fastest"] = metrics.Statistics.Minimum() * 1000.0
	data["slowest"] = metrics.Statistics.Maximum() * 1000.0
	data["range"] = metrics.Statistics.Range() * 1000.0

	return data
}

// Report returns a JSON representation of the metrics.
func (n *Network) Report() map[string]map[string]interface{} {
	n.RLock()
	defer n.RUnlock()
	data := make(map[string]map[string]interface{})
	for host, bench := range n.metrics {
		data[host] = bench.Serialize()
	}
	return data
}

// metrics returns the benchmark for the specified host (not thread-safe).
func (n *Network) get(host string) *stats.Benchmark {
	// Get the stats object from the map
	metrics, ok := n.metrics[host]
	if !ok {
		metrics = new(stats.Benchmark)
		n.metrics[host] = metrics
	}

	return metrics
}
