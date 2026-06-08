//go:build !unix

package cli

import (
	"errors"
	"net/http"
)

func newBrokerHTTPClient(int) *http.Client { return nil }

var errBrokerUnsupported = errors.New("flashduty: broker mode is not supported on this platform")

// brokerEgressCapable reports whether this build can act as a broker-mode client.
// False on non-unix builds, which lack the FLASHDUTY_CRED_FD dial path. The
// runner reads this (via `fduty version --json`) to decide whether to advertise
// broker mode to safari. See broker_dial_unix.go for the rationale.
const brokerEgressCapable = false
