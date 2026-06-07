//go:build unix

package cli

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
			// Blocks until a handshake datagram arrives or parentFD is closed by
			// stop() (which unblocks Recvmsg with EBADF/EOF → clean exit, no leak).
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
		// Closing parentFD unblocks the control goroutine's Recvmsg so it exits
		// instead of leaking across -count iterations.
		_ = syscall.Close(parentFD)
		<-ctlGone
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
	defer syscall.Close(childFD)
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

// serveProxyConn is a tiny test upstream-proxy used by fakeBroker; the real
// implementation lives in the runner, this mirrors it for the CLI test.
func serveProxyConn(conn net.Conn, upstream, realKey string) {
	defer conn.Close()
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
