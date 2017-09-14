package kekahu

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bbengfort/kekahu/ping"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// DefaultAddr is the default port that the server listens on.
const DefaultAddr = ":3284"

// DefaultPingTimeout to wait for an echo response from a server (e.g. TTL)
const DefaultPingTimeout = time.Second * 2

//===========================================================================
// Echo Server
//===========================================================================

// Server implements the Echo service to respond to ping requests from other
// hosts in order to measure inter-host latencies over time.
type Server struct {
	name     string // host information for the server
	addr     string // address to bind the server to
	messages uint64 // number of messages responded to
}

// Init the server with the name and address. If name is empty, use hostname.
// If addr is empty string, then use the default address.
func (s *Server) Init(addr, name string) {
	s.addr = addr
	s.name = name

	if s.name == "" {
		s.name, _ = os.Hostname()
	}

	if s.addr == "" {
		s.addr = DefaultAddr
	}
}

// Run the server on the specified address, listening for Ping requests and
// responding to them as quickly as possible.
func (s *Server) Run(echan chan<- error) error {
	// Create the TCP socket to listen on
	sock, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("could not listen on '%s': %s", s.addr, err)
	}

	// Log taht we're listening on the socket
	status("listening for pings on %s", s.addr)

	// Create the gRPC server and handler
	srv := grpc.NewServer()
	ping.RegisterEchoServer(srv, s)

	// Run the server in its own go routine
	go func() {
		defer sock.Close()
		if err = srv.Serve(sock); err != nil {
			echan <- err
		}
	}()

	return nil
}

// Shutdown the server with a status message
func (s *Server) Shutdown() error {
	status("replied to %d pings", s.messages)
	return nil
}

// Ping implements the ping.EchoServer interface. Server handling is simply to
// log the message has been received and to
func (s *Server) Ping(ctx context.Context, in *ping.Packet) (*ping.Packet, error) {
	// Log that we've received the message
	s.messages++
	info("received ping %d from %s", in.Sequence, in.Source)

	// Send the reply
	in.Target = s.name
	return in, nil
}

//===========================================================================
// Echo Client
//===========================================================================

// Ping from the specified source to the specified target at the given
// addr (note that if the addr doesn't contain a port, the DefaultAddr port is
// appended to the addr). This method returns the latency of the message from
// one endpoint to the other, or it returns 0 if the message times out.
//
// This method is quite heavyweight but is fast enough since it is not called
// often. In the future we can abstract this to resusable components so we're
// not building the request every time. Ensure, however, that the latency is
// only computing the time it takes to send and receive a message.
func (k *KeKahu) Ping(source, target, addr string, seq uint64) (time.Duration, error) {
	// First compose the address
	addr = resolveAddr(addr)

	// Create the message
	msg := &ping.Packet{
		Source:   source,
		Target:   target,
		Sequence: seq,
	}

	// Create the connection
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return 0, fmt.Errorf("could not connect to '%s': %s", addr, err)
	}
	defer conn.Close()

	// Create the grpc client and send the ping
	client := ping.NewEchoClient(conn)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultPingTimeout)
	defer cancel()

	if _, err = client.Ping(ctx, msg); err != nil {
		return 0, fmt.Errorf("could not send ping to %s: %s", addr, err)
	}

	// Compute the latency immediately
	latency := time.Since(start)
	info("ping from %s to %s in %s", source, target, latency)
	return latency, nil
}

// Resolves the address by appending the default port if one isn't on it. This
// method simply splits on : and if no colon is found, then appends the default
// addr constant.
func resolveAddr(addr string) string {
	parts := strings.Split(addr, ":")
	if len(parts) == 1 {
		return addr + DefaultAddr
	}
	return addr
}
