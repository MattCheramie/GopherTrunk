package iio

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"
)

// dialFailure says, in operator terms, which layer a failed iiod connect
// broke at. "Is the board reachable, and is iiod running on it?" covered
// both, and #1329's PlutoSky answered neither ping nor TCP: that is the
// board's network link or firmware, not iiod and not GopherTrunk, and the
// operator needs to be told so instead of being sent after iiod.
type dialFailure int

const (
	dialOther       dialFailure = iota
	dialUnreachable             // no route / host down: nothing answers at the IP level
	dialRefused                 // host answered, nothing listening on the port
	dialTimeout                 // no answer at all within the timeout
)

func classifyDialError(err error) dialFailure {
	switch {
	case errors.Is(err, syscall.EHOSTUNREACH), errors.Is(err, syscall.ENETUNREACH),
		errors.Is(err, syscall.EHOSTDOWN):
		return dialUnreachable
	case errors.Is(err, syscall.ECONNREFUSED):
		return dialRefused
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return dialTimeout
	}
	// Windows reports WSA error codes whose text differs from the POSIX
	// errnos above; match the message as a fallback.
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no route to host"), strings.Contains(msg, "network is unreachable"),
		strings.Contains(msg, "unreachable host"), strings.Contains(msg, "unreachable network"):
		return dialUnreachable
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "actively refused"):
		return dialRefused
	case strings.Contains(msg, "timed out"), strings.Contains(msg, "timeout"):
		return dialTimeout
	}
	return dialOther
}

// explainDialError wraps a failed dial to addr with what to check next.
func explainDialError(addr string, err error) error {
	host := addr
	if h, _, splitErr := net.SplitHostPort(addr); splitErr == nil {
		host = h
	}
	var hint string
	switch classifyDialError(err) {
	case dialUnreachable:
		hint = fmt.Sprintf("nothing answers at %s at the network level. This is before iiod and GopherTrunk: "+
			"check that `ping %s` works. If it does not, check the cable or USB link, that the host has an "+
			"address on the board's subnet (USB networking puts the board at 192.168.2.1), and the board's "+
			"network settings; a board that never answers ping has a link or firmware problem", host, host)
	case dialRefused:
		hint = fmt.Sprintf("%s answered but nothing is listening on the iiod port. The board is up, so check "+
			"that iiod is running on it and that the port is 30431 (or the one given in -addr)", host)
	case dialTimeout:
		hint = fmt.Sprintf("no answer from %s within the timeout. Check `ping %s`: if ping works, a firewall "+
			"may be dropping TCP port 30431; if not, see the network checks for an unreachable board", host, host)
	default:
		hint = "is the board reachable, and is iiod running on it?"
	}
	return fmt.Errorf("iio: dial %s: %w (%s)", addr, err, hint)
}
