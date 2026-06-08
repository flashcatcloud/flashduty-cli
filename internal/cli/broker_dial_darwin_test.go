//go:build darwin

package cli

import "syscall"

// controlSockType is the control-channel socket type used by the broker dial
// test. darwin's AF_UNIX has no SOCK_SEQPACKET support, so the native test
// falls back to SOCK_DGRAM, which preserves datagram boundaries identically for
// the CLI dialer's Sendmsg/Recvmsg+SCM_RIGHTS path. Production runners are
// Linux-only (SOCK_SEQPACKET).
const controlSockType = syscall.SOCK_DGRAM
