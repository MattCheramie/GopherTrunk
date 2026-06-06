package soapyremote

import (
	"encoding/binary"
	"fmt"
	"math"
)

// Stream datagram wire format (common/SoapyStreamEndpoint.cpp). Every transfer
// (one UDP datagram, or one framed unit on a TCP stream socket) begins with a
// 24-byte header. The header's integer fields are network byte order; the IQ
// sample payload that follows is copied RAW (native little-endian on the
// x86/ARM hosts that run SoapySDRServer) and is NOT byte-swapped.
const streamHeaderSize = 24

// streamHeader is the decoded 24-byte datagram header.
type streamHeader struct {
	bytes    uint32 // total transfer size including this header
	sequence uint32 // sender's monotonically increasing sequence
	elems    int32  // element count per channel, or a negative SoapySDR error code
	flags    int32  // SOAPY_SDR_* stream flags
	time     int64  // timestamp (ns)
}

// SoapyRemote stream flow control (common/SoapyStreamEndpoint.cpp). On an RX
// stream the *receiver* (this client) must send flow-control ACK datagrams back
// to the server over the stream/data socket, or the server streams nothing:
//
//   - The server's sender thread blocks in waitSend() while
//     "not _receiveInitial", so it will not transmit a single sample until it
//     has received an initial ACK from us.
//   - Thereafter it may only run _maxInFlightSeqs datagrams ahead of the last
//     sequence we have ACKed, so we must keep ACKing as data arrives.
//
// The receiver normally sends the initial ACK in its constructor and a
// gratuitous ACK every _triggerAckWindow = _maxInFlightSeqs/_numBuffs received
// datagrams (acquireRecv). We mirror that cadence here. Without these ACKs the
// server never sends, our reads time out with zero bytes, and the radio's RX
// path overruns (issue #542 follow-up).
const (
	// streamMTU matches SoapyRemote's default endpoint MTU.
	streamMTU = 1500
	// streamNumBuffs mirrors SOAPY_REMOTE_ENDPOINT_NUM_BUFFS.
	streamNumBuffs = 8
	// streamWindowBytes is the receiver window we advertise as the in-flight
	// credit ceiling. The server gates its sender on the value we send, not on
	// any local buffer, so this only needs to be comfortably large.
	streamWindowBytes = 8 * 1024 * 1024
	// maxInFlightSeqs is the credit window advertised to the server (the ACK's
	// elems field); it gates how far ahead the server may stream (window/mtu).
	maxInFlightSeqs = streamWindowBytes / streamMTU // ≈5592
	// triggerAckWindow is how many received datagrams elapse between ACKs
	// (_maxInFlightSeqs/_numBuffs), matching upstream's gratuitous-ACK cadence.
	triggerAckWindow = maxInFlightSeqs / streamNumBuffs // ≈699
)

// encodeStreamACK builds the 24-byte flow-control ACK datagram: a bare,
// payload-less header whose sequence is the last received sequence and whose
// elems carries the advertised in-flight window. Big-endian, byte-matching
// SoapyStreamEndpoint::sendACK.
func encodeStreamACK(seq uint32) []byte {
	b := make([]byte, streamHeaderSize)
	binary.BigEndian.PutUint32(b[0:4], streamHeaderSize)
	binary.BigEndian.PutUint32(b[4:8], seq)
	binary.BigEndian.PutUint32(b[8:12], maxInFlightSeqs)
	binary.BigEndian.PutUint32(b[12:16], 0)
	binary.BigEndian.PutUint64(b[16:24], 0)
	return b
}

func decodeStreamHeader(b []byte) streamHeader {
	return streamHeader{
		bytes:    binary.BigEndian.Uint32(b[0:4]),
		sequence: binary.BigEndian.Uint32(b[4:8]),
		elems:    int32(binary.BigEndian.Uint32(b[8:12])),
		flags:    int32(binary.BigEndian.Uint32(b[12:16])),
		time:     int64(binary.BigEndian.Uint64(b[16:24])),
	}
}

// sampleFormat identifies the wire sample encoding the server was asked to send.
type sampleFormat int

const (
	formatCS16 sampleFormat = iota // interleaved little-endian int16 I,Q
	formatCF32                     // interleaved little-endian float32 I,Q
)

// soapyName returns the SoapySDR format string for the SETUP_STREAM call.
func (f sampleFormat) soapyName() string {
	if f == formatCF32 {
		return "CF32"
	}
	return "CS16"
}

// bytesPerSample is the size of one complex sample on the wire.
func (f sampleFormat) bytesPerSample() int {
	if f == formatCF32 {
		return 8
	}
	return 4
}

func parseFormat(s string) (sampleFormat, error) {
	switch s {
	case "", "CS16", "cs16":
		return formatCS16, nil
	case "CF32", "cf32":
		return formatCF32, nil
	default:
		return 0, fmt.Errorf("soapyremote: unsupported format %q (want CS16 or CF32)", s)
	}
}

// convert turns one raw sample-payload buffer into complex64 normalized to
// roughly [-1, 1), matching every other GopherTrunk SDR backend.
func (f sampleFormat) convert(buf []byte) []complex64 {
	if f == formatCF32 {
		return convertCF32(buf)
	}
	return convertCS16(buf)
}

// convertCS16 maps interleaved little-endian int16 I,Q to complex64.
// Mirrors baseband.decodeIQ16 / rtltcp.convertU8 normalization.
func convertCS16(buf []byte) []complex64 {
	n := len(buf) / 4
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := int16(binary.LittleEndian.Uint16(buf[4*i:]))
		qv := int16(binary.LittleEndian.Uint16(buf[4*i+2:]))
		out[i] = complex(float32(iv)/32768, float32(qv)/32768)
	}
	return out
}

// convertCF32 maps interleaved little-endian float32 I,Q to complex64. The
// server already delivers samples in the conventional [-1, 1] float range.
func convertCF32(buf []byte) []complex64 {
	n := len(buf) / 8
	out := make([]complex64, n)
	for i := 0; i < n; i++ {
		iv := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i:]))
		qv := math.Float32frombits(binary.LittleEndian.Uint32(buf[8*i+4:]))
		out[i] = complex(iv, qv)
	}
	return out
}
