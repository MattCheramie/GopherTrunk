package soapyremote

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sort"
	"time"
)

// SoapyRemote RPC wire format (reverse-engineered from pothosware/SoapyRemote
// @master, common/SoapyRPCPacket.hpp + SoapyRPCPacker.cpp + SoapyRPCUnpacker.cpp).
//
// Every RPC message is: 12-byte header + payload + 4-byte trailer, all framed
// integers big-endian (network order):
//
//	header  = headerWord uint32 ("SRPC") | version uint32 | length uint32
//	trailer = trailerWord uint32 ("CPRS")
//
// length is the TOTAL message size, INCLUDING header and trailer.
//
// The payload is a sequence of type-tagged values. Each value starts with a
// 1-byte type tag (SoapyRemoteTypes) followed by a body. Crucially there are
// no "bare" integers: every nested integer carries its own INT32/INT64 tag.
// So e.g. a CALL is [CALL][INT32 tag][id], a STRING is [STRING][INT32 tag]
// [len][raw bytes], and a FLOAT64 is [FLOAT64][INT32 tag][exp][INT64 tag][man].
const (
	rpcHeaderWord  uint32 = 0x53525043 // "SRPC"
	rpcTrailerWord uint32 = 0x43505253 // "CPRS"
	rpcVersion     uint32 = 0x000400
	rpcHeaderSize         = 12
	rpcTrailerSize        = 4
)

// SoapyRemoteTypes — the 1-byte value type tags.
const (
	tChar       byte = 0
	tBool       byte = 1
	tInt32      byte = 2
	tInt64      byte = 3
	tFloat64    byte = 4
	tComplex128 byte = 5
	tString     byte = 6
	tRange      byte = 7
	tRangeList  byte = 8
	tStringList byte = 9
	tFloat64Lst byte = 10
	tKwargs     byte = 11
	tKwargsList byte = 12
	tException  byte = 13
	tVoid       byte = 14
	tCall       byte = 15
	tSizeList   byte = 16
	tArgInfo    byte = 17
	tArgInfoLst byte = 18
)

// SoapyRemoteCalls — the subset of RPC call IDs this client issues.
const (
	callFind                   int32 = 0
	callMake                   int32 = 1
	callUnmake                 int32 = 2
	callHangup                 int32 = 3
	callGetServerID            int32 = 20
	callGetHardwareKey         int32 = 101
	callSetupStream            int32 = 300
	callCloseStream            int32 = 301
	callActivateStream         int32 = 302
	callDeactivateStream       int32 = 303
	callGetNativeStreamFormat  int32 = 305
	callSetFrequencyCorrection int32 = 504
	callSetGainMode            int32 = 701
	callSetGain                int32 = 703
	callSetFrequency           int32 = 800
	callSetSampleRate          int32 = 900
	callGetSampleRate          int32 = 901
	callGetHardwareTime        int32 = 1101
	callWriteSetting           int32 = 1400
)

// dirRX is SoapySDR's SOAPY_SDR_RX direction (TX is 0).
const dirRX byte = 1

// soapyFlagHasTime is SoapySDR's SOAPY_SDR_HAS_TIME stream flag: the timeNs
// argument of ACTIVATE_STREAM carries a valid hardware timestamp, so the
// remote driver schedules the stream start instead of starting immediately.
const soapyFlagHasTime int32 = 1 << 2

// dblMantDig mirrors C's DBL_MANT_DIG used in the frexp/ldexp double codec.
const dblMantDig = 53

// packer builds one RPC message. The first rpcHeaderSize bytes are reserved
// for the header (filled in by writeTo); values append after.
type packer struct {
	buf []byte
}

func newPacker() *packer {
	return &packer{buf: make([]byte, rpcHeaderSize, 128)}
}

func (p *packer) raw8(b byte)     { p.buf = append(p.buf, b) }
func (p *packer) rawU32(v uint32) { p.buf = binary.BigEndian.AppendUint32(p.buf, v) }
func (p *packer) rawU64(v uint64) { p.buf = binary.BigEndian.AppendUint64(p.buf, v) }

// taggedI32 / taggedI64 write a self-tagged integer, matching the C++
// operator&(int) / operator&(long long).
func (p *packer) taggedI32(v int32) { p.raw8(tInt32); p.rawU32(uint32(v)) }
func (p *packer) taggedI64(v int64) { p.raw8(tInt64); p.rawU64(uint64(v)) }

func (p *packer) call(id int32) { p.raw8(tCall); p.taggedI32(id) }
func (p *packer) char(c byte)   { p.raw8(tChar); p.raw8(c) }

func (p *packer) boolean(b bool) {
	p.raw8(tBool)
	if b {
		p.raw8(1)
	} else {
		p.raw8(0)
	}
}

func (p *packer) i32(v int32) { p.taggedI32(v) }
func (p *packer) i64(v int64) { p.taggedI64(v) }

// f64 packs a double exactly as SoapyRemote does: the raw frexp exponent as a
// tagged int32 and the ldexp(frac, DBL_MANT_DIG) mantissa as a tagged int64.
// The (long long) cast in C truncates toward zero; int64(float64) in Go does
// the same, so the round-trip is bit-identical.
func (p *packer) f64(v float64) {
	p.raw8(tFloat64)
	frac, exp := math.Frexp(v)
	man := int64(math.Ldexp(frac, dblMantDig))
	p.taggedI32(int32(exp))
	p.taggedI64(man)
}

func (p *packer) str(s string) {
	p.raw8(tString)
	p.taggedI32(int32(len(s)))
	p.buf = append(p.buf, s...)
}

func (p *packer) sizeList(v []int) {
	p.raw8(tSizeList)
	p.taggedI32(int32(len(v)))
	for _, x := range v {
		p.taggedI32(int32(x))
	}
}

// kwargs packs a map. Keys are sorted so the wire bytes are deterministic
// (the server rebuilds an unordered map, so order is purely cosmetic, but
// determinism keeps tests stable).
func (p *packer) kwargs(m map[string]string) {
	p.raw8(tKwargs)
	p.taggedI32(int32(len(m)))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		p.str(k)
		p.str(m[k])
	}
}

// writeTo finalizes the message (length + trailer) and writes it to conn under
// the given deadline.
func (p *packer) writeTo(conn net.Conn, timeout time.Duration) error {
	p.rawU32(rpcTrailerWord) // trailer
	total := uint32(len(p.buf))
	binary.BigEndian.PutUint32(p.buf[0:4], rpcHeaderWord)
	binary.BigEndian.PutUint32(p.buf[4:8], rpcVersion)
	binary.BigEndian.PutUint32(p.buf[8:12], total)
	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
		defer conn.SetWriteDeadline(time.Time{})
	}
	if _, err := conn.Write(p.buf); err != nil {
		return fmt.Errorf("soapyremote: write rpc: %w", err)
	}
	return nil
}

// unpacker reads tagged values out of an RPC response payload (header and
// trailer already stripped).
type unpacker struct {
	buf []byte
	off int
}

var errShortResponse = errors.New("soapyremote: short rpc response")

func (u *unpacker) peekTag() (byte, bool) {
	if u.off >= len(u.buf) {
		return 0, false
	}
	return u.buf[u.off], true
}

func (u *unpacker) tag() (byte, error) {
	if u.off >= len(u.buf) {
		return 0, errShortResponse
	}
	t := u.buf[u.off]
	u.off++
	return t, nil
}

func (u *unpacker) expect(want byte) error {
	t, err := u.tag()
	if err != nil {
		return err
	}
	if t != want {
		return fmt.Errorf("soapyremote: rpc type tag %d, want %d", t, want)
	}
	return nil
}

func (u *unpacker) u32() (uint32, error) {
	if u.off+4 > len(u.buf) {
		return 0, errShortResponse
	}
	v := binary.BigEndian.Uint32(u.buf[u.off:])
	u.off += 4
	return v, nil
}

func (u *unpacker) i32() (int32, error) {
	if err := u.expect(tInt32); err != nil {
		return 0, err
	}
	v, err := u.u32()
	return int32(v), err
}

func (u *unpacker) char() (byte, error) {
	if err := u.expect(tChar); err != nil {
		return 0, err
	}
	return u.tag() // a raw byte follows the CHAR tag; reuse tag() to read one
}

func (u *unpacker) boolean() (bool, error) {
	if err := u.expect(tBool); err != nil {
		return false, err
	}
	b, err := u.tag()
	return b != 0, err
}

func (u *unpacker) call() (int32, error) {
	if err := u.expect(tCall); err != nil {
		return 0, err
	}
	return u.i32()
}

func (u *unpacker) u64() (uint64, error) {
	if u.off+8 > len(u.buf) {
		return 0, errShortResponse
	}
	v := binary.BigEndian.Uint64(u.buf[u.off:])
	u.off += 8
	return v, nil
}

func (u *unpacker) i64() (int64, error) {
	if err := u.expect(tInt64); err != nil {
		return 0, err
	}
	v, err := u.u64()
	return int64(v), err
}

func (u *unpacker) f64() (float64, error) {
	if err := u.expect(tFloat64); err != nil {
		return 0, err
	}
	exp, err := u.i32()
	if err != nil {
		return 0, err
	}
	if err := u.expect(tInt64); err != nil {
		return 0, err
	}
	man, err := u.u64()
	if err != nil {
		return 0, err
	}
	return math.Ldexp(float64(int64(man)), int(exp)-dblMantDig), nil
}

func (u *unpacker) str() (string, error) {
	if err := u.expect(tString); err != nil {
		return "", err
	}
	n, err := u.i32()
	if err != nil {
		return "", err
	}
	if n < 0 || u.off+int(n) > len(u.buf) {
		return "", errShortResponse
	}
	s := string(u.buf[u.off : u.off+int(n)])
	u.off += int(n)
	return s, nil
}

// checkException returns a non-nil error if the response leads with a
// SOAPY_REMOTE_EXCEPTION value (the server's way of reporting a failed call).
// It is a no-op for VOID and normal return values.
func (u *unpacker) checkException() error {
	t, ok := u.peekTag()
	if !ok || t != tException {
		return nil
	}
	u.off++ // consume exception tag
	msg, err := u.str()
	if err != nil {
		return fmt.Errorf("soapyremote: remote exception (unreadable): %w", err)
	}
	return fmt.Errorf("soapyremote: remote exception: %s", msg)
}

// readResponse reads one framed RPC response off conn and returns an unpacker
// over its payload.
func readResponse(conn net.Conn, timeout time.Duration) (*unpacker, error) {
	if timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		defer conn.SetReadDeadline(time.Time{})
	}
	var hdr [rpcHeaderSize]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, fmt.Errorf("soapyremote: read rpc header: %w", err)
	}
	if got := binary.BigEndian.Uint32(hdr[0:4]); got != rpcHeaderWord {
		return nil, fmt.Errorf("soapyremote: rpc header magic %#08x, want %#08x", got, rpcHeaderWord)
	}
	length := binary.BigEndian.Uint32(hdr[8:12])
	if length < rpcHeaderSize+rpcTrailerSize {
		return nil, fmt.Errorf("soapyremote: rpc length %d too small", length)
	}
	body := make([]byte, length-rpcHeaderSize)
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, fmt.Errorf("soapyremote: read rpc body: %w", err)
	}
	trailer := binary.BigEndian.Uint32(body[len(body)-rpcTrailerSize:])
	if trailer != rpcTrailerWord {
		return nil, fmt.Errorf("soapyremote: rpc trailer %#08x, want %#08x", trailer, rpcTrailerWord)
	}
	return &unpacker{buf: body[:len(body)-rpcTrailerSize]}, nil
}
