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

// ErrBrokerClosed is returned (wrapped) when the runner-side broker control
// channel is gone: the runner exited, or reclaimed the channel once the
// command that started this process finished, so fduty calls from a
// long-lived background process fail this way. A live broker declining a
// dial is a different failure (see the 0xFF path in dial). Callers and
// automation can tell the two apart with errors.Is(err, ErrBrokerClosed);
// the wrapping by http.Transport and url.Error preserves that.
var ErrBrokerClosed = errors.New("flashduty: broker control channel closed: the broker that started this process is no longer available")

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
		// A dead control channel surfaces on send as EPIPE (stream-style
		// sockets) or ECONNREFUSED (datagram-style sockets, where the closed
		// peer answers the datagram): classify both as ErrBrokerClosed so
		// callers can tell "broker gone" apart from "broker alive but declined
		// this dial" (the 0xFF path below).
		if errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNREFUSED) {
			return nil, fmt.Errorf("%w (handshake send: %v)", ErrBrokerClosed, err)
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
		return nil, fmt.Errorf("%w (handshake recv: connection closed by peer)", ErrBrokerClosed)
	}
	if body[0] == 0xFF {
		// ctrlRespErr: the broker is alive but declined this dial (e.g. it
		// could not mint a connection). Deliberately not ErrBrokerClosed.
		return nil, errors.New("flashduty: broker refused the dial request")
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
	if err != nil || len(fds) == 0 {
		return nil, fmt.Errorf("broker parse rights: %w", err)
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
