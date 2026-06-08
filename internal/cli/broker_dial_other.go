//go:build !unix

package cli

import (
	"errors"
	"net/http"
)

func newBrokerHTTPClient(int) *http.Client { return nil }

var errBrokerUnsupported = errors.New("flashduty: broker mode is not supported on this platform")
