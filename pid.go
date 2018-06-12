// Handling process id information and enables cross-process communication
// between the server and the command line client.

package kekahu

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

//===========================================================================
// OS Signal Handlers
//===========================================================================

func signalHandler(shutdown func() error) {
	// Make signal channel and register notifiers for Interupt and Terminate
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive a signal on the channel
	<-sigchan

	// Shutdown now that we've received the signal
	if err := shutdown(); err != nil {
		msg := fmt.Sprintf("shutdown error: %s", err.Error())
		log.Fatal(msg)
	}

	// Make a clean exit
	os.Exit(0)
}
