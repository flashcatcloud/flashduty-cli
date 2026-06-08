//go:build linux

package cli

import "syscall"

// controlSockType is the control-channel socket type used by the broker dial
// test. On Linux (production) the runner uses SOCK_SEQPACKET.
const controlSockType = syscall.SOCK_SEQPACKET
