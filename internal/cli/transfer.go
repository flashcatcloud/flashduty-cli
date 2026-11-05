package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// Timeouts differ by kind of call. A JSON call is small, so the SDK client's
// overall timeout (30s) bounds it. A byte transfer — file upload, file
// download, transcript export — runs as long as its size and link need, so the
// SDK serves it without an overall timeout and the CLI bounds it by progress
// instead: it is cancelled only after transferIdleTimeout with no bytes moved.
// The idle clock also covers the wait for the response to start and for the
// reply after the last byte is sent.
var transferIdleTimeout = 60 * time.Second

// errTransferStalled reports a transfer cancelled for lack of progress.
var errTransferStalled = errors.New("transfer stalled")

// transfer is the context of one byte transfer: pass ctx to the SDK call, wrap
// the transferred stream with track, and pass the call's error through done.
type transfer struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	timer  *time.Timer
}

func startTransfer(parent context.Context) *transfer {
	ctx, cancel := context.WithCancelCause(parent)
	return &transfer{
		ctx:    ctx,
		cancel: cancel,
		timer:  time.AfterFunc(transferIdleTimeout, func() { cancel(errTransferStalled) }),
	}
}

// track wraps r so every read that moves bytes restarts the idle clock.
func (t *transfer) track(r io.Reader) io.Reader {
	return &progressReader{r: r, timer: t.timer}
}

// done stops the idle clock and, when the transfer failed because it stalled,
// says so instead of the bare "context canceled" the stall produced.
func (t *transfer) done(err error) error {
	t.timer.Stop()
	stalled := errors.Is(context.Cause(t.ctx), errTransferStalled)
	t.cancel(nil)
	if err != nil && stalled {
		return fmt.Errorf("%w: no bytes moved for %s, so it was cancelled; retry, and check the network path if it repeats", errTransferStalled, transferIdleTimeout)
	}
	return err
}

type progressReader struct {
	r     io.Reader
	timer *time.Timer
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.timer.Reset(transferIdleTimeout)
	}
	return n, err
}
