package kekahu

import (
	"fmt"
	"time"
)

// Heartbeat posts a status message to the Kahu server then delays sending
// the next heartbeat message.
func Heartbeat(key string, delay time.Duration) {

	// Schedule the next heartbeat after this completes
	defer time.AfterFunc(delay, func() {
		Heartbeat(key, delay)
	})

	fmt.Println("heartbeat!")
}
