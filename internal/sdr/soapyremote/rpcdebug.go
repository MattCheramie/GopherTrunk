package soapyremote

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
)

// Verbose RPC tracing for the SoapyRemote control channel, enabled per endpoint
// with sdr.soapy_remote[].verbose_debug.
//
// The wire carries no schema — a message is a call id followed by bare tagged
// values — so when SoapySDRServer complains "~SoapyRPCUnpacker: Unconsumed
// payload bytes N" it is saying the handler it dispatched to did not want as
// many arguments as arrived, and it does not say which call. Without a client
// trace the only way to work out which RPC it was is to guess. That guessing is
// what let callSetAntenna sit on opcode 600 (HAS_DC_OFFSET_MODE) through a
// release: the server logged unconsumed bytes, replied with a bool, and the
// bool passed as success.
//
// A hex dump alone would not have closed that either — the decoded argument
// list is the part that lines up against upstream's ClientHandler.cpp.

// traceMaxHexBytes caps the hex dump per frame. Long enough for any control RPC
// this client sends (the largest is SETUP_STREAM with its kwargs).
const traceMaxHexBytes = 96

// rpcTracer renders one line per RPC request and response at DEBUG. nil when
// verbose_debug is off, and every call site is nil-guarded, so the disabled
// path is byte-identical to having no tracer at all.
type rpcTracer struct {
	log  *slog.Logger
	addr string
	seq  atomic.Uint64
}

func newRPCTracer(log *slog.Logger, addr string) *rpcTracer {
	return &rpcTracer{log: log, addr: addr}
}

// next hands out the sequence number that pairs a request line with its
// response line.
func (t *rpcTracer) next() uint64 { return t.seq.Add(1) }

// request traces an outgoing frame. buf is the complete on-wire message,
// header and trailer included, as writeTo is about to send it.
func (t *rpcTracer) request(seq uint64, buf []byte) {
	if t == nil {
		return
	}
	payload := framePayload(buf)
	t.log.Debug("soapyremote: rpc >",
		"addr", t.addr,
		"seq", seq,
		"call", describeCall(payload),
		"args", describePayload(payload),
		"bytes", len(buf),
		"hex", hexBytes(buf, traceMaxHexBytes))
}

// response traces an incoming frame. payload is the message body with header
// and trailer already stripped (what readResponse hands back).
func (t *rpcTracer) response(seq uint64, callID int32, payload []byte, err error) {
	if t == nil {
		return
	}
	attrs := []any{
		"addr", t.addr,
		"seq", seq,
		"call", callName(callID),
	}
	if err != nil {
		attrs = append(attrs, "err", err)
		t.log.Debug("soapyremote: rpc <", attrs...)
		return
	}
	attrs = append(attrs,
		"args", describePayload(payload),
		"bytes", len(payload),
		"hex", hexBytes(payload, traceMaxHexBytes))
	t.log.Debug("soapyremote: rpc <", attrs...)
}

// framePayload strips the 12-byte header and 4-byte trailer from a complete
// on-wire message. Returns nil for anything too short to be one.
func framePayload(buf []byte) []byte {
	if len(buf) < rpcHeaderSize+rpcTrailerSize {
		return nil
	}
	return buf[rpcHeaderSize : len(buf)-rpcTrailerSize]
}

// describeCall names the leading CALL value of a request payload, or "" when
// the payload does not start with one (every response).
func describeCall(payload []byte) string {
	u := &unpacker{buf: payload}
	if t, ok := u.peekTag(); !ok || t != tCall {
		return ""
	}
	id, err := u.call()
	if err != nil {
		return ""
	}
	return callName(id)
}

// callName renders an opcode as NAME(id), or just the id when unknown — an
// unknown id in a trace is itself the finding.
func callName(id int32) string {
	if n, ok := callNames[id]; ok {
		return fmt.Sprintf("%s(%d)", n, id)
	}
	return fmt.Sprintf("UNKNOWN(%d)", id)
}

// callNames covers the calls this client issues. Kept beside the opcode
// constants in rpc.go — add both together.
var callNames = map[int32]string{
	callFind:                   "FIND",
	callMake:                   "MAKE",
	callUnmake:                 "UNMAKE",
	callHangup:                 "HANGUP",
	callGetServerID:            "GET_SERVER_ID",
	callGetHardwareKey:         "GET_HARDWARE_KEY",
	callSetupStream:            "SETUP_STREAM",
	callCloseStream:            "CLOSE_STREAM",
	callActivateStream:         "ACTIVATE_STREAM",
	callDeactivateStream:       "DEACTIVATE_STREAM",
	callGetNativeStreamFormat:  "GET_NATIVE_STREAM_FORMAT",
	callListAntennas:           "LIST_ANTENNAS",
	callSetAntenna:             "SET_ANTENNA",
	callGetAntenna:             "GET_ANTENNA",
	callSetFrequencyCorrection: "SET_FREQUENCY_CORRECTION",
	callSetGainMode:            "SET_GAIN_MODE",
	callSetGain:                "SET_GAIN",
	callSetFrequency:           "SET_FREQUENCY",
	callSetSampleRate:          "SET_SAMPLE_RATE",
	callGetSampleRate:          "GET_SAMPLE_RATE",
	callGetHardwareTime:        "GET_HARDWARE_TIME",
	callWriteSetting:           "WRITE_SETTING",
}

// describePayload walks the tagged values in a message body and renders them
// as a comma-separated argument list. Decoding stops at the first value whose
// tag this client never handles — the remaining bytes are reported as a count,
// since a tag with an unknown length cannot be skipped over.
//
// This is deliberately independent of what the caller BELIEVES it packed: the
// point of the trace is to show what actually went on the wire.
func describePayload(payload []byte) string {
	if len(payload) == 0 {
		return "<empty>"
	}
	u := &unpacker{buf: payload}
	var parts []string
	for {
		tag, ok := u.peekTag()
		if !ok {
			break
		}
		s, err := describeValue(u, tag)
		if err != nil {
			parts = append(parts, fmt.Sprintf("<tag %d undecodable: %v; %d bytes left>",
				tag, err, len(payload)-u.off))
			break
		}
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "<empty>"
	}
	return strings.Join(parts, ", ")
}

func describeValue(u *unpacker, tag byte) (string, error) {
	switch tag {
	case tCall:
		id, err := u.call()
		if err != nil {
			return "", err
		}
		return "call " + callName(id), nil
	case tChar:
		c, err := u.char()
		if err != nil {
			return "", err
		}
		if c >= 0x20 && c < 0x7f {
			return fmt.Sprintf("char %q", rune(c)), nil
		}
		return fmt.Sprintf("char %#02x", c), nil
	case tBool:
		b, err := u.boolean()
		if err != nil {
			return "", err
		}
		return "bool " + strconv.FormatBool(b), nil
	case tInt32:
		v, err := u.i32()
		if err != nil {
			return "", err
		}
		return "i32 " + strconv.FormatInt(int64(v), 10), nil
	case tInt64:
		v, err := u.i64()
		if err != nil {
			return "", err
		}
		return "i64 " + strconv.FormatInt(v, 10), nil
	case tFloat64:
		v, err := u.f64()
		if err != nil {
			return "", err
		}
		return "f64 " + strconv.FormatFloat(v, 'g', -1, 64), nil
	case tString:
		s, err := u.str()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("str %q", s), nil
	case tStringList:
		v, err := u.strList()
		if err != nil {
			return "", err
		}
		quoted := make([]string, len(v))
		for i, s := range v {
			quoted[i] = strconv.Quote(s)
		}
		return "strList[" + strings.Join(quoted, " ") + "]", nil
	case tSizeList:
		v, err := describeSizeList(u)
		if err != nil {
			return "", err
		}
		return v, nil
	case tKwargs:
		m, err := u.kwargs()
		if err != nil {
			return "", err
		}
		return "kwargs{" + formatKwargs(m) + "}", nil
	case tException:
		// Framed as a string; checkException reads it the same way.
		if err := u.expect(tException); err != nil {
			return "", err
		}
		msg, err := u.str()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("exception %q", msg), nil
	case tVoid:
		if _, err := u.tag(); err != nil {
			return "", err
		}
		return "void", nil
	default:
		return "", fmt.Errorf("this client does not decode tag %d", tag)
	}
}

// describeSizeList mirrors packer.sizeList — the tag, a tagged count, then that
// many tagged int32s. The unpacker has no reader for it (only the client packs
// one), so the trace decodes it here.
func describeSizeList(u *unpacker) (string, error) {
	if err := u.expect(tSizeList); err != nil {
		return "", err
	}
	n, err := u.i32()
	if err != nil {
		return "", err
	}
	if n < 0 {
		return "", errShortResponse
	}
	vals := make([]string, 0, n)
	for i := int32(0); i < n; i++ {
		v, err := u.i32()
		if err != nil {
			return "", err
		}
		vals = append(vals, strconv.FormatInt(int64(v), 10))
	}
	return "sizeList[" + strings.Join(vals, " ") + "]", nil
}

// formatKwargs renders a kwargs map with sorted keys so two traces of the same
// call are diffable.
func formatKwargs(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// hexBytes renders up to max bytes as lowercase hex, noting how many were
// elided. Mirrors the rtlsdr USB debug transport's dump format.
func hexBytes(b []byte, max int) string {
	n := len(b)
	if n > max {
		n = max
	}
	var sb strings.Builder
	sb.Grow(n*3 + 16)
	const hexDigits = "0123456789abcdef"
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteByte(hexDigits[b[i]>>4])
		sb.WriteByte(hexDigits[b[i]&0x0f])
	}
	if len(b) > n {
		fmt.Fprintf(&sb, " ... (%d more)", len(b)-n)
	}
	return sb.String()
}
