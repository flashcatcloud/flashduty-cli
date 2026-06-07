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
		return nil, fmt.Errorf("broker handshake send: %w", err)
	}
	body := make([]byte, 1)
	oob := make([]byte, syscall.CmsgSpace(4)) // room for exactly one fd
	n, oobn, _, _, err := syscall.Recvmsg(d.credFD, body, oob, 0)
	if err != nil {
		return nil, fmt.Errorf("broker handshake recv: %w", err)
	}
	if n < 1 || body[0] != 0x01 {
		return nil, fmt.Errorf("broker refused connection (code %v)", body[:n])
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
			DialContext:           d.dial,
			DisableCompression:    false,
			MaxIdleConns:          0,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 0,
		},
	}
}
