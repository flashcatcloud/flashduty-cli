//go:build unix

package cli

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
)

// fakeBroker mimics the runner: it owns the parent end of a SEQPACKET control
// socket, and for each 1-byte handshake it dispatches a dedicated STREAM conn
// (via SCM_RIGHTS) that proxies to the given upstream URL, overwriting app_key.
func fakeBroker(t *testing.T, parentFD int, upstream string, realKey string) (stop func()) {
	t.Helper()
	var (
		mu      sync.Mutex
		conns   []net.Conn
		ctlGone = make(chan struct{})
	)
	go func() {
		defer close(ctlGone)
		buf := make([]byte, 8)
		for {
			// Blocks until a handshake datagram arrives or stop() shuts down
			// parentFD (recvmsg then returns EOF → clean exit, no leak). NOTE: a
			// bare Close(parentFD) does NOT wake a blocked recvmsg on Linux (only on
			// darwin/BSD), so stop() uses Shutdown — see the stop func below.
			n, _, _, _, err := syscall.Recvmsg(parentFD, buf, nil, 0)
			if err != nil || n == 0 {
				return
			}
			// Create the dedicated STREAM pair; keep one end, send the other.
			pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
			if err != nil {
				_ = syscall.Sendmsg(parentFD, []byte{0xFF}, nil, nil, 0)
				continue
			}
			rights := syscall.UnixRights(pair[1])
			if serr := syscall.Sendmsg(parentFD, []byte{0x01}, rights, nil, 0); serr != nil {
				_ = syscall.Close(pair[0])
				_ = syscall.Close(pair[1])
				return
			}
			_ = syscall.Close(pair[1])
			// Serve HTTP on pair[0], proxying to upstream with the real key.
			myEnd := os.NewFile(uintptr(pair[0]), "broker-end")
			conn, _ := net.FileConn(myEnd)
			_ = myEnd.Close()
			if conn == nil {
				continue
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
			go serveProxyConn(conn, upstream, realKey)
		}
	}()
	return func() {
		// Shutdown (NOT a bare Close) wakes the control goroutine's blocked
		// Recvmsg portably — on Linux, closing an fd does not interrupt a recvmsg
		// blocked on it in another goroutine; shutdown returns EOF on both Linux
		// and darwin. Join, then close.
		_ = syscall.Shutdown(parentFD, syscall.SHUT_RDWR)
		<-ctlGone
		_ = syscall.Close(parentFD)
		mu.Lock()
		for _, c := range conns {
			_ = c.Close()
		}
		mu.Unlock()
	}
}

func TestBrokerHTTPClient_DialAndRewrite(t *testing.T) {
	// Upstream asserts the real key arrived (sentinel was overwritten).
	gotKey := make(chan string, 16)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey <- r.URL.Query().Get("app_key")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	// Control channel: child end → CLI; parent end → fake broker. Production
	// (Linux runner) uses SOCK_SEQPACKET, but darwin's AF_UNIX has no SEQPACKET
	// support, so the native test uses SOCK_DGRAM — both preserve the datagram
	// boundaries the 1-byte handshake relies on, and the CLI dialer's
	// Sendmsg/Recvmsg+SCM_RIGHTS path is identical for either socket type.
	pair, err := syscall.Socketpair(syscall.AF_UNIX, controlSockType, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	childFD, parentFD := pair[0], pair[1]
	stop := fakeBroker(t, parentFD, upstream.URL, "REAL-KEY") // owns + closes parentFD
	defer func() { _ = syscall.Close(childFD) }()
	defer stop()

	client := newBrokerHTTPClient(childFD)
	if client == nil {
		t.Fatal("newBrokerHTTPClient returned nil")
	}
	defer client.CloseIdleConnections() // release dispatched keep-alive conns
	// The CLI's base URL is an http placeholder; broker rewrites host.
	req, _ := http.NewRequestWithContext(context.Background(), "GET",
		"http://flashduty.broker.local/incident/channels?app_key=SENTINEL", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do: %v", err)
	}
	_, _ = io.Copy(io.Discard, resp.Body) // drain so the conn is reusable
	_ = resp.Body.Close()
	if got := <-gotKey; got != "REAL-KEY" {
		t.Fatalf("upstream saw app_key=%q, want REAL-KEY (sentinel not overwritten)", got)
	}

	// Concurrency: 8 parallel requests each get their own dispatched conn.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _ := http.NewRequest("GET", "http://flashduty.broker.local/x?app_key=SENTINEL", nil)
			rsp, e := client.Do(r)
			if e != nil {
				errs <- e
				return
			}
			_, _ = io.Copy(io.Discard, rsp.Body)
			_ = rsp.Body.Close()
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("concurrent client.Do: %v", e)
	}
	for i := 0; i < 8; i++ {
		if got := <-gotKey; got != "REAL-KEY" {
			t.Fatalf("concurrent req saw app_key=%q", got)
		}
	}
}

// TestDefaultNewClient_BrokerMode covers the defaultNewClient wiring: with
// FLASHDUTY_CRED_FD set it builds a client with no configured app_key (the
// broker supplies the real key), and a malformed fd value is a clean error.
func TestDefaultNewClient_BrokerMode(t *testing.T) {
	// Hermetic config: empty HOME (no config file) + no env app key.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("FLASHDUTY_APP_KEY", "")

	// A real, open fd so newBrokerHTTPClient gets a usable control channel.
	pair, err := syscall.Socketpair(syscall.AF_UNIX, controlSockType, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	defer func() { _ = syscall.Close(pair[0]) }()
	defer func() { _ = syscall.Close(pair[1]) }()

	t.Setenv("FLASHDUTY_CRED_FD", strconv.Itoa(pair[0]))
	client, err := defaultNewClient()
	if err != nil {
		t.Fatalf("broker mode with no app key must succeed, got: %v", err)
	}
	if client == nil {
		t.Fatal("broker mode returned nil client")
	}

	// A malformed fd value is rejected up front (not silently ignored).
	t.Setenv("FLASHDUTY_CRED_FD", "not-a-number")
	if _, err := defaultNewClient(); err == nil {
		t.Fatal("invalid FLASHDUTY_CRED_FD must error")
	}

	// Both a configured app key AND a control fd: broker mode wins. The client
	// builds with the sentinel key (the broker overwrites it with the real
	// per-person key), so the configured app key never reaches the wire.
	t.Setenv("FLASHDUTY_APP_KEY", "ENV-KEY-SHOULD-NOT-BE-USED")
	t.Setenv("FLASHDUTY_CRED_FD", strconv.Itoa(pair[0]))
	if c, err := defaultNewClient(); err != nil || c == nil {
		t.Fatalf("both app key + cred fd set: want broker client, got client=%v err=%v", c, err)
	}
}

// TestDefaultNewClient_RejectsStdioFD verifies the fd>=3 guard: fds 0/1/2 are
// stdio and can never be the runner-injected control end, so they are rejected
// rather than handshaking on stdin/stdout.
func TestDefaultNewClient_RejectsStdioFD(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("FLASHDUTY_APP_KEY", "")
	for _, fd := range []string{"-1", "0", "1", "2"} {
		t.Setenv("FLASHDUTY_CRED_FD", fd)
		if _, err := defaultNewClient(); err == nil {
			t.Fatalf("FLASHDUTY_CRED_FD=%q must be rejected (stdio/invalid)", fd)
		}
	}
}

// TestBrokerHTTPClient_RefusedReturnsError verifies the dialer surfaces the
// broker's 0xFF refusal (e.g. the runner failed to mint a connection) as a real
// error instead of hanging or wrapping a nil conn.
func TestBrokerHTTPClient_RefusedReturnsError(t *testing.T) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, controlSockType, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	childFD, parentFD := pair[0], pair[1]
	defer func() { _ = syscall.Close(childFD) }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 8)
		for {
			n, _, _, _, rerr := syscall.Recvmsg(parentFD, buf, nil, 0)
			if rerr != nil || n == 0 {
				return
			}
			_ = syscall.Sendmsg(parentFD, []byte{0xFF}, nil, nil, 0) // always refuse
		}
	}()

	client := newBrokerHTTPClient(childFD)
	req, _ := http.NewRequestWithContext(context.Background(), "GET",
		"http://flashduty.broker.local/x?app_key=SENTINEL", nil)
	if _, err := client.Do(req); err == nil {
		t.Fatal("client.Do must fail when broker refuses with 0xFF")
	}
	// Shutdown (not bare Close) to wake the goroutine's blocked Recvmsg on Linux.
	_ = syscall.Shutdown(parentFD, syscall.SHUT_RDWR)
	<-done
	_ = syscall.Close(parentFD)
}

// TestBrokerDialer_RecvEOF_ClassifiesBrokerClosed covers a broker that closes
// the control channel mid-handshake: Recvmsg returns 0 (orderly EOF) and the
// dial must classify as ErrBrokerClosed with a message that explains the
// channel is gone.
func TestBrokerDialer_RecvEOF_ClassifiesBrokerClosed(t *testing.T) {
	// SOCK_STREAM because closing the peer end must wake the dialer's blocked
	// Recvmsg with EOF on both darwin and Linux; datagram sockets carry no EOF
	// signalling. Production SEQPACKET hits this same n==0 code path.
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	childFD, parentFD := pair[0], pair[1]
	defer func() { _ = syscall.Close(childFD) }()

	parentGone := make(chan struct{})
	go func() {
		defer close(parentGone)
		defer func() { _ = syscall.Close(parentFD) }()
		buf := make([]byte, 1)
		// Consume the handshake byte, then close: the dialer's Recvmsg
		// returns EOF.
		_, _, _, _, _ = syscall.Recvmsg(parentFD, buf, nil, 0)
	}()

	d := &brokerDialer{credFD: childFD}
	_, err = d.dial(context.Background(), "", "")
	<-parentGone
	if !errors.Is(err, ErrBrokerClosed) {
		t.Fatalf("dial after broker close: want ErrBrokerClosed, got: %v", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "broker control channel") {
		t.Fatalf("user-facing message must mention the closed control channel, got: %v", msg)
	}
}

// TestBrokerDialer_SendEPIPE_ClassifiesBrokerClosed covers a broker whose
// control channel is already gone when the handshake starts: Sendmsg fails
// with EPIPE (stream sockets) and the dial must classify as ErrBrokerClosed.
func TestBrokerDialer_SendEPIPE_ClassifiesBrokerClosed(t *testing.T) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	childFD, parentFD := pair[0], pair[1]
	defer func() { _ = syscall.Close(childFD) }()
	_ = syscall.Close(parentFD) // peer gone before the handshake

	d := &brokerDialer{credFD: childFD}
	_, err = d.dial(context.Background(), "", "")
	if !errors.Is(err, ErrBrokerClosed) {
		t.Fatalf("dial with dead broker: want ErrBrokerClosed, got: %v", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "no longer available") {
		t.Fatalf("user-facing message must state the broker is gone, got: %v", msg)
	}
}

// TestBrokerDialer_RefusalIsNotBrokerClosed covers a live broker that answers
// the handshake with the 0xFF refusal byte: it must NOT classify as
// ErrBrokerClosed, and the message must read as a refusal.
func TestBrokerDialer_RefusalIsNotBrokerClosed(t *testing.T) {
	pair, err := syscall.Socketpair(syscall.AF_UNIX, controlSockType, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}
	childFD, parentFD := pair[0], pair[1]
	defer func() { _ = syscall.Close(childFD) }()
	defer func() { _ = syscall.Close(parentFD) }()

	// Queue the refusal before the dial: the dialer's Sendmsg succeeds and its
	// Recvmsg consumes the already-buffered 0xFF datagram, so no goroutine is
	// needed.
	if err := syscall.Sendmsg(parentFD, []byte{0xFF}, nil, nil, 0); err != nil {
		t.Fatalf("queue refusal: %v", err)
	}

	d := &brokerDialer{credFD: childFD}
	_, err = d.dial(context.Background(), "", "")
	if err == nil {
		t.Fatal("dial must fail when the broker refuses")
	}
	if errors.Is(err, ErrBrokerClosed) {
		t.Fatalf("a live broker's refusal must not classify as ErrBrokerClosed: %v", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "broker refused the dial request") {
		t.Fatalf("user-facing message must say the dial was refused, got: %v", msg)
	}
}

// serveProxyConn is a tiny test upstream-proxy used by fakeBroker; the real
// implementation lives in the runner, this mirrors it for the CLI test.
func serveProxyConn(conn net.Conn, upstream, realKey string) {
	defer func() { _ = conn.Close() }()
	br := newReadProxy(conn, upstream, realKey)
	br.run()
}

// readProxy is a minimal test-only HTTP proxy: it reads requests off conn,
// overwrites the app_key query param with realKey, forwards to upstream, and
// copies each response back. Serves sequential keep-alive requests until the
// connection closes.
type readProxy struct {
	conn     net.Conn
	upstream *url.URL
	realKey  string
}

func newReadProxy(conn net.Conn, upstream, realKey string) *readProxy {
	u, _ := url.Parse(upstream)
	return &readProxy{conn: conn, upstream: u, realKey: realKey}
}

func (p *readProxy) run() {
	br := bufio.NewReader(p.conn)
	for {
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}
		// Point the request at the real upstream and overwrite the sentinel.
		req.URL.Scheme = p.upstream.Scheme
		req.URL.Host = p.upstream.Host
		req.Host = p.upstream.Host
		q := req.URL.Query()
		q.Set("app_key", p.realKey)
		req.URL.RawQuery = q.Encode()
		req.RequestURI = ""
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			return
		}
		if err := resp.Write(p.conn); err != nil {
			_ = resp.Body.Close()
			return
		}
		_ = resp.Body.Close()
	}
}
