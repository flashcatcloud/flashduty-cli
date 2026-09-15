//go:build unix

package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"
)

// errBrokerUnsupported is returned when broker mode is requested on a build that
// cannot provide it. On unix this is effectively unreachable (newBrokerHTTPClient
// never returns nil), but defaultNewClient references it on every platform.
var errBrokerUnsupported = errors.New("flashduty: broker mode is not supported on this platform")

// errBrokerClosed is returned (wrapped) when the runner-side broker control
// channel is gone: the runner exited, or reclaimed the channel once the
// command that started this process finished, so fduty calls from a
// long-lived background process fail this way. A live broker declining a
// dial is a different failure (see the 0xFF path in dial).
//
// The distinction rides the user-visible message, not error identity: the
// SDK flattens dial errors into text ("%v"), so only in-module tests
// classify via errors.Is; humans and agents reading the output tell the two
// failures apart by wording.
//
// It covers the handshake phase only. When an already-dispatched connection
// is torn down, in-flight requests see EOF/reset first; the next dial
// reports this error.
var errBrokerClosed = errors.New("broker channel closed: the command that started this process has finished — rerun it in the foreground of a live session")

// brokerEgressCapable reports whether this build can act as a broker-mode client
// (read FLASHDUTY_CRED_FD and dial over the inherited control fd). The runner
// probes it via `fduty version --json` and only advertises broker mode to safari
// when true — otherwise safari would deliver the per-person key out-of-band to an
// fduty that can't read it. True on unix, where this file is built.
const brokerEgressCapable = true

// brokerDialer owns the inherited control fd and serializes per-dial handshakes.
// Each Dial sends a 1-byte request datagram on the control channel and receives
// one dedicated SOCK_STREAM fd back via SCM_RIGHTS.
type brokerDialer struct {
	mu     sync.Mutex // serialize send+recv so concurrent dials don't cross fds
	credFD int
}

func (d *brokerDialer) dial(_ context.Context, _, _ string) (net.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := syscall.Sendmsg(d.credFD, []byte{0x01}, nil, nil, 0); err != nil {
		// A dead control channel surfaces on send as EPIPE on the
		// production Linux SEQPACKET socket, or ECONNREFUSED when a
		// datagram-style peer is gone; classify both as errBrokerClosed so
		// "broker gone" reads differently from "broker alive but declined
		// this dial" (the 0xFF path below).
		if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNREFUSED) {
			return nil, fmt.Errorf("%w (handshake send: %v)", errBrokerClosed, err)
		}
		return nil, fmt.Errorf("broker handshake send: %w", err)
	}
	body := make([]byte, 1)
	oob := make([]byte, syscall.CmsgSpace(4)) // room for exactly one fd
	n, oobn, _, _, err := syscall.Recvmsg(d.credFD, body, oob, 0)
	if err != nil {
		return nil, fmt.Errorf("broker handshake recv: %w", err)
	}
	if n == 0 {
		// Orderly EOF: the broker closed the control channel.
		return nil, fmt.Errorf("%w (recvmsg: EOF)", errBrokerClosed)
	}
	if body[0] == 0xFF {
		// ctrlRespErr: the broker is alive but declined this dial (e.g. it
		// could not mint a connection). Deliberately not errBrokerClosed.
		return nil, errors.New("broker refused the dial request (no request reached Flashduty; retrying is safe)")
	}
	if body[0] != 0x01 {
		return nil, fmt.Errorf("broker handshake: unexpected response byte 0x%02x", body[0])
	}
	scms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return nil, fmt.Errorf("broker parse scm: %w", err)
	}
	if len(scms) == 0 {
		return nil, fmt.Errorf("broker sent no fd")
	}
	fds, err := syscall.ParseUnixRights(&scms[0])
	if err != nil {
		return nil, fmt.Errorf("broker parse rights: %w", err)
	}
	if len(fds) == 0 {
		return nil, fmt.Errorf("broker sent no usable fd")
	}
	f := os.NewFile(uintptr(fds[0]), "broker-conn")
	conn, err := net.FileConn(f) // dups + registers with the netpoller
	_ = f.Close()
	if err != nil {
		return nil, fmt.Errorf("broker fileconn: %w", err)
	}
	return conn, nil
}

// newBrokerHTTPClient builds an *http.Client whose Transport.DialContext routes
// every connection over the inherited control fd. Timeout matches the SDK's
// historical default (30s) so behavior is unchanged for non-streaming calls;
// streaming export relies on request context like before.
func newBrokerHTTPClient(credFD int) *http.Client {
	d := &brokerDialer{credFD: credFD}
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:        d.dial,
			DisableCompression: false,
			MaxIdleConns:       0,
			// All dials target the same logical host (the broker sentinel base
			// URL) over the one control fd, so a single idle keep-alive conn is
			// enough for pagination loops; cap it so dispatched conns don't linger.
			MaxIdleConnsPerHost:   1,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 0,
		},
	}
}
