package ka9qradio

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"strings"
	"time"
)

// mDNS link-local group (RFC 6762).
var mdnsGroup = netip.AddrPortFrom(netip.AddrFrom4([4]byte{224, 0, 0, 251}), 5353)

// DNS record types / class.
const (
	dnsTypeA    = 1
	dnsTypeAAAA = 28
	dnsClassIN  = 1
	// qClassUnicast is the top bit of the question's class, the mDNS "QU"
	// flag asking the responder to reply unicast to our source port so a
	// plain (non-group-joined) socket sees the answer (RFC 6762 §5.4).
	qClassUnicast = 0x8000
)

// resolveMDNS resolves a single-label `.local` hostname (radiod registers the
// channel's status multicast group address as the name's A record) to an IP by
// querying the mDNS group and reading the first matching A/AAAA answer. It is a
// deliberately small, dependency-free resolver: it sends one query for A and
// one for AAAA, prefers the IPv4 answer (radiod groups are IPv4 239.x.x.x), and
// gives up after timeout.
func resolveMDNS(host string, timeout time.Duration) (netip.Addr, error) {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return netip.Addr{}, fmt.Errorf("ka9qradio: mdns listen: %w", err)
	}
	defer conn.Close()

	group := net.UDPAddr{IP: mdnsGroup.Addr().AsSlice(), Port: int(mdnsGroup.Port())}
	for _, qt := range []uint16{dnsTypeA, dnsTypeAAAA} {
		q, err := buildQuery(host, qt)
		if err != nil {
			return netip.Addr{}, err
		}
		if _, err := conn.WriteToUDP(q, &group); err != nil {
			return netip.Addr{}, fmt.Errorf("ka9qradio: mdns query: %w", err)
		}
	}

	deadline := time.Now().Add(timeout)
	_ = conn.SetReadDeadline(deadline)
	want := dnsName(host)
	buf := make([]byte, 1500)
	for time.Now().Before(deadline) {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			// Timeout (or transient) — stop trying.
			break
		}
		if addr, ok := parseAnswer(buf[:n], want); ok {
			return addr, nil
		}
	}
	return netip.Addr{}, fmt.Errorf("ka9qradio: mdns: no answer for %q", host)
}

// dnsName lowercases and trims a hostname for case-insensitive comparison.
func dnsName(host string) string {
	return strings.ToLower(strings.TrimSuffix(host, "."))
}

// buildQuery builds a single-question mDNS query for host with the QU bit set.
func buildQuery(host string, qtype uint16) ([]byte, error) {
	var b []byte
	id := uint16(rand.Uint32())
	b = append(b, byte(id>>8), byte(id)) // ID
	b = append(b, 0x00, 0x00)            // flags: standard query
	b = append(b, 0x00, 0x01)            // QDCOUNT=1
	b = append(b, 0x00, 0x00)            // ANCOUNT
	b = append(b, 0x00, 0x00)            // NSCOUNT
	b = append(b, 0x00, 0x00)            // ARCOUNT
	for _, label := range strings.Split(dnsName(host), ".") {
		if len(label) == 0 || len(label) > 63 {
			return nil, fmt.Errorf("ka9qradio: invalid mdns label in %q", host)
		}
		b = append(b, byte(len(label)))
		b = append(b, label...)
	}
	b = append(b, 0x00) // root label
	b = append(b, byte(qtype>>8), byte(qtype))
	cls := uint16(dnsClassIN | qClassUnicast)
	b = append(b, byte(cls>>8), byte(cls))
	return b, nil
}

// parseAnswer scans a DNS response for the first A/AAAA record whose owner name
// matches want, returning its address. Name compression is handled when reading
// owner names; RDATA is read by length.
func parseAnswer(msg []byte, want string) (netip.Addr, bool) {
	if len(msg) < 12 {
		return netip.Addr{}, false
	}
	qd := int(msg[4])<<8 | int(msg[5])
	an := int(msg[6])<<8 | int(msg[7])
	off := 12
	// Skip the question section.
	for i := 0; i < qd; i++ {
		var ok bool
		_, off, ok = readName(msg, off)
		if !ok || off+4 > len(msg) {
			return netip.Addr{}, false
		}
		off += 4 // QTYPE + QCLASS
	}
	for i := 0; i < an; i++ {
		name, noff, ok := readName(msg, off)
		if !ok {
			return netip.Addr{}, false
		}
		off = noff
		if off+10 > len(msg) {
			return netip.Addr{}, false
		}
		rtype := int(msg[off])<<8 | int(msg[off+1])
		rdlen := int(msg[off+8])<<8 | int(msg[off+9])
		off += 10
		if off+rdlen > len(msg) {
			return netip.Addr{}, false
		}
		rdata := msg[off : off+rdlen]
		off += rdlen
		if !strings.EqualFold(name, want) {
			continue
		}
		switch rtype {
		case dnsTypeA:
			if rdlen == 4 {
				return netip.AddrFrom4([4]byte{rdata[0], rdata[1], rdata[2], rdata[3]}), true
			}
		case dnsTypeAAAA:
			if rdlen == 16 {
				var a [16]byte
				copy(a[:], rdata)
				return netip.AddrFrom16(a), true
			}
		}
	}
	return netip.Addr{}, false
}

// readName decodes a DNS name starting at off, following compression pointers,
// and returns the dotted name plus the offset just past the name in the record
// stream (i.e. past the first pointer, not into the pointed-to data).
func readName(msg []byte, off int) (string, int, bool) {
	var labels []string
	jumped := false
	next := off
	// Bound the loop to the message size to defeat pointer loops.
	for steps := 0; steps < len(msg); steps++ {
		if off >= len(msg) {
			return "", 0, false
		}
		b := int(msg[off])
		if b == 0 {
			off++
			if !jumped {
				next = off
			}
			return strings.Join(labels, "."), next, true
		}
		if b&0xc0 == 0xc0 {
			if off+1 >= len(msg) {
				return "", 0, false
			}
			ptr := (b&0x3f)<<8 | int(msg[off+1])
			if !jumped {
				next = off + 2
			}
			jumped = true
			off = ptr
			continue
		}
		off++
		if off+b > len(msg) {
			return "", 0, false
		}
		labels = append(labels, string(msg[off:off+b]))
		off += b
	}
	return "", 0, false
}
