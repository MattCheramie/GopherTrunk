// Package ka9qradio implements an sdr.Driver that consumes a channel from a
// remote ka9q-radio `radiod` instance over IP multicast, in pure Go with no
// CGO. radiod (https://github.com/ka9q/ka9q-radio) is a multichannel SDR
// daemon that reads a front end (RX888, Airspy, RTL-SDR, …), runs
// fast-convolution digital downconverters, and publishes each channel as an
// RTP/IP-multicast stream. A channel configured for raw "linear" IQ output
// (`OUTPUT_CHANNELS = 2`) emits interleaved complex samples that GopherTrunk
// can demodulate exactly like a local dongle — so one well-sited radiod can
// feed many decoders across a LAN (issue #765).
//
// Two multicast groups are involved, mirroring radiod itself:
//
//   - A status + command group (UDP port 5006). radiod multicasts per-channel
//     metadata here as binary TLV (type/length/value) packets, and clients
//     send commands to the same group. Each channel is identified by its RTP
//     SSRC. This driver POLLs the group for its configured SSRC to learn the
//     channel's data multicast socket, sample rate, encoding and centre
//     frequency, and sends RADIO_FREQUENCY commands to retune (see status.go).
//   - A data group (UDP port 5004) carrying standard RTP packets whose payload
//     is the channel's interleaved IQ in the negotiated encoding (see rtp.go).
//
// radiod instances are named with mDNS `.local` hostnames whose A/AAAA record
// is the status multicast group address; mdns.go resolves them. Plain
// `host:port` and literal `239.x.x.x:port` addresses bypass mDNS.
//
// The wire formats (TLV grammar, integer/double leading-zero compression, RTP
// header, socket TLV layout, and the status_type / encoding enums) are
// transcribed from ka9q-radio @ main (src/status.{c,h}, src/rtp.{c,h}) so the
// values stay byte-compatible. It is exercised by synthetic packets in the
// tests; validate against a live radiod before relying on it.
//
// Limitations:
//   - Raw IQ ("linear", OUTPUT_CHANNELS=2) channels only. Opus / μ-law / A-law
//     / AX.25 payloads are audio/packet encodings, not IQ, and are rejected.
//   - The operator pre-configures the channel in radiod@.conf; this driver
//     consumes and retunes an existing channel rather than creating one.
//   - Plaintext multicast, like rtl_tcp / soapy_remote. Use on trusted LANs.
package ka9qradio

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/netip"
)

// UDP ports radiod uses by default (src/rtp.h: DEFAULT_RTP_PORT /
// DEFAULT_STAT_PORT).
const (
	defaultDataPort   = 5004
	defaultStatusPort = 5006
)

// pktType is the first byte of every status/command packet
// (src/status.h enum pkt_type).
const (
	pktStatus  byte = 0
	pktCommand byte = 1
)

// statusType is the TLV type byte (src/status.h enum status_type). The enum is
// transcribed in full and in order: only EOL is fixed at 0 and every other
// value is positional, so the complete list is required to keep the handful we
// use (COMMAND_TAG, OUTPUT_SSRC, OUTPUT_DATA_DEST_SOCKET, OUTPUT_SAMPRATE,
// RADIO_FREQUENCY, OUTPUT_CHANNELS, OUTPUT_ENCODING) at the correct offsets.
type statusType uint8

const (
	stEOL        statusType = iota // 0: end-of-list sentinel
	stCommandTag                   // echoes the requester's tag
	stCmdCnt
	stGPSTime

	stDescription
	stStatusDestSocket
	stSetopts
	stClearopts
	stRTPTimesnap
	stBinByteData
	stInputSamprate
	stSpectrumBase
	stSpectrumAvg
	stInputSamples
	stWindowType
	stNoiseBW

	stOutputDataSourceSocket
	stOutputDataDestSocket // multicast group:port carrying this channel's IQ
	stOutputSSRC
	stOutputTTL
	stOutputSamprate // output sample rate, Hz (integer)
	stOutputMetadataPackets
	stOutputDataPackets
	stOutputErrors

	stCalibrate
	stLNAGain
	stMixerGain
	stIFGain

	stDCIOffset
	stDCQOffset
	stIQImbalance
	stIQPhase
	stDirectConversion

	stRadioFrequency // channel centre frequency, Hz (double)
	stFirstLOFrequency
	stSecondLOFrequency
	stShiftFrequency
	stDopplerFrequency
	stDopplerFrequencyRate

	stLowEdge
	stHighEdge
	stKaiserBeta
	stFilterBlocksize
	stFilterFIRLength
	stFilter2

	stIFPower
	stBasebandPower
	stNoiseDensity

	stDemodType      // 0 = linear (raw IQ), 1 = FM, 2 = WFM, 3 = spectrum
	stOutputChannels // 1 or 2; raw IQ uses 2 (I,Q)
	stIndependentSideband
	stPLLEnable
	stPLLLock
	stPLLSquare
	stPLLPhase
	stPLLBW
	stEnvelope
	stSNRSquelch

	stPLLSNR
	stFreqOffset
	stPeakDeviation
	stPLTone

	stAGCEnable
	stHeadroom
	stAGCHangtime
	stAGCRecoveryRate
	stFMSNR
	stAGCThreshold

	stGain
	stOutputLevel
	stOutputSamples

	stOpusBitRate
	stMaxdelay
	stFilter2Blocksize
	stFilter2FIRLength
	stFilter2KaiserBeta
	stSpectrumFFTN

	stFilterDrops
	stLock

	stTP1
	stTP2

	stUnused4
	stADBitsPerSample
	stSquelchOpen
	stSquelchClose
	stPreset
	stDeemphTC
	stDeemphGain
	stUnused3
	stPLDeviation
	stThreshExtend

	stSpectrumShape
	stUnused2
	stResolutionBW
	stBinCount
	stCrossover
	stBinData

	stRFAtten
	stRFGain
	stRFAGC
	stFELowEdge
	stFEHighEdge
	stFEIsReal
	stUnused
	stADOver
	stRTPPT
	stStatusInterval
	stOutputEncoding // enum encoding of the output samples
	stSamplesSinceOver
	stPLLWraps
	stRFLevelCal
	stOpusDTX
	stOpusApplication
	stOpusBandwidth
	stOpusFEC
	stSpectrumStep
	stSpectrumOverlap
	stLifetime
)

// encoding is the on-wire sample encoding (src/rtp.h enum encoding).
type encoding uint8

const (
	encNone encoding = iota // NO_ENCODING
	encS16LE
	encS16BE
	encOpus
	encF32LE
	encAX25
	encF16LE
	encOpusVOIP
	encF32BE
	encF16BE
	encMulaw
	encAlaw
)

// parseEncoding maps the config/string names radiod accepts (src/rtp.c
// parse_encoding) to the wire enum. Only the encodings that carry raw IQ are
// meaningful here; the rest are accepted for completeness so a status reply
// that advertises them parses, and isIQEncoding rejects them later.
func parseEncoding(s string) (encoding, bool) {
	switch s {
	case "s16be", "s16", "int":
		return encS16BE, true
	case "s16le":
		return encS16LE, true
	case "f32", "float", "f32le":
		return encF32LE, true
	case "f32be":
		return encF32BE, true
	case "f16le", "f16":
		return encF16LE, true
	case "f16be":
		return encF16BE, true
	case "":
		return encNone, true
	default:
		return encNone, false
	}
}

func (e encoding) String() string {
	switch e {
	case encS16LE:
		return "s16le"
	case encS16BE:
		return "s16be"
	case encF32LE:
		return "f32le"
	case encF32BE:
		return "f32be"
	case encF16LE:
		return "f16le"
	case encF16BE:
		return "f16be"
	case encOpus:
		return "opus"
	case encOpusVOIP:
		return "opus-voip"
	case encAX25:
		return "ax.25"
	case encMulaw:
		return "mulaw"
	case encAlaw:
		return "alaw"
	default:
		return "none"
	}
}

// isIQEncoding reports whether an encoding carries raw IQ samples this driver
// can decode (signed-16 or 32-bit float, either byte order). Opus/μ-law/A-law
// are lossy audio codecs and AX.25 is packet data — none are IQ.
func isIQEncoding(e encoding) bool {
	switch e {
	case encS16LE, encS16BE, encF32LE, encF32BE:
		return true
	default:
		return false
	}
}

// --- TLV codec (src/status.c) ---------------------------------------------
//
// Each TLV is a 1-byte type, a length field, then the value. A length < 128 is
// a single byte; a length >= 128 is encoded as 0x80|N followed by N big-endian
// length bytes. Integers (and the IEEE bits of floats/doubles) are big-endian
// with leading zero bytes dropped, so a zero value is length 0.

// appendTLV writes one type/length/value triple.
func appendTLV(buf []byte, typ statusType, val []byte) []byte {
	buf = append(buf, byte(typ))
	n := len(val)
	switch {
	case n < 128:
		buf = append(buf, byte(n))
	case n < 1<<16:
		buf = append(buf, 0x80|2, byte(n>>8), byte(n))
	case n < 1<<24:
		buf = append(buf, 0x80|3, byte(n>>16), byte(n>>8), byte(n))
	default:
		buf = append(buf, 0x80|4, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	return append(buf, val...)
}

// appendInt encodes an unsigned integer with leading-zero compression, matching
// radiod's encode_int64.
func appendInt(buf []byte, typ statusType, v uint64) []byte {
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], v)
	i := 0
	for i < 8 && tmp[i] == 0 {
		i++
	}
	return appendTLV(buf, typ, tmp[i:])
}

// appendDouble encodes a float64 as its big-endian IEEE-754 bits (compressed),
// matching radiod's encode_double.
func appendDouble(buf []byte, typ statusType, f float64) []byte {
	return appendInt(buf, typ, math.Float64bits(f))
}

// appendEOL terminates a TLV list with a single null type byte.
func appendEOL(buf []byte) []byte { return append(buf, byte(stEOL)) }

// decodeUint reads a big-endian, possibly leading-zero-compressed unsigned
// integer (radiod's decode_int64).
func decodeUint(val []byte) uint64 {
	var r uint64
	for _, b := range val {
		r = (r << 8) | uint64(b)
	}
	return r
}

// decodeFloat reads a compressed float or double value (radiod's decode_float):
// up to 4 bytes is a float32, more is a float64. Both reconstruct because
// leading-zero compression only drops high bytes.
func decodeFloat(val []byte) float64 {
	switch {
	case len(val) == 0:
		return 0
	case len(val) <= 4:
		return float64(math.Float32frombits(uint32(decodeUint(val))))
	default:
		return math.Float64frombits(decodeUint(val))
	}
}

// decodeSocket reads an OUTPUT_DATA_DEST_SOCKET-style value: 6 bytes for IPv4
// (4-byte addr + 2-byte port, both network order) or 18 bytes for IPv6
// (16-byte addr + 2-byte port). The address family is inferred from the length,
// exactly as radiod's decode_socket does.
func decodeSocket(val []byte) (netip.AddrPort, error) {
	switch len(val) {
	case 6:
		addr := netip.AddrFrom4([4]byte{val[0], val[1], val[2], val[3]})
		port := binary.BigEndian.Uint16(val[4:6])
		return netip.AddrPortFrom(addr, port), nil
	case 18:
		var a [16]byte
		copy(a[:], val[:16])
		addr := netip.AddrFrom16(a)
		port := binary.BigEndian.Uint16(val[16:18])
		return netip.AddrPortFrom(addr, port), nil
	default:
		return netip.AddrPort{}, fmt.Errorf("ka9qradio: socket TLV has %d bytes, want 6 or 18", len(val))
	}
}

// tlv is one decoded type/value pair.
type tlv struct {
	typ statusType
	val []byte
}

// parseTLVs walks a status/command payload (the bytes AFTER the leading
// pkt_type byte) into its TLVs, stopping at the EOL sentinel. A truncated
// length or value returns an error rather than reading out of bounds.
func parseTLVs(p []byte) ([]tlv, error) {
	var out []tlv
	for i := 0; i < len(p); {
		typ := statusType(p[i])
		i++
		if typ == stEOL {
			break
		}
		if i >= len(p) {
			return nil, fmt.Errorf("ka9qradio: truncated TLV length for type %d", typ)
		}
		optlen := int(p[i])
		i++
		if optlen&0x80 != 0 {
			lol := optlen & 0x7f
			if i+lol > len(p) {
				return nil, fmt.Errorf("ka9qradio: truncated extended length")
			}
			optlen = 0
			for k := 0; k < lol; k++ {
				optlen = (optlen << 8) | int(p[i])
				i++
			}
		}
		if i+optlen > len(p) {
			return nil, fmt.Errorf("ka9qradio: TLV type %d length %d overruns packet", typ, optlen)
		}
		out = append(out, tlv{typ: typ, val: p[i : i+optlen]})
		i += optlen
	}
	return out, nil
}

// channelStatus is the subset of a radiod status reply this driver needs.
type channelStatus struct {
	ssrc        uint32
	hasSSRC     bool
	dataDest    netip.AddrPort
	hasDataDest bool
	sampleRate  uint32
	encoding    encoding
	channels    int
	radioFreqHz float64
}

// parseStatus decodes a status packet (including the leading pkt_type byte) for
// the fields this driver tracks. It does not require the packet to be a STATUS
// (vs CMD) so the same routine can introspect either.
func parseStatus(pkt []byte) (channelStatus, error) {
	var st channelStatus
	if len(pkt) == 0 {
		return st, fmt.Errorf("ka9qradio: empty status packet")
	}
	tlvs, err := parseTLVs(pkt[1:])
	if err != nil {
		return st, err
	}
	for _, t := range tlvs {
		switch t.typ {
		case stOutputSSRC:
			st.ssrc = uint32(decodeUint(t.val))
			st.hasSSRC = true
		case stOutputDataDestSocket:
			ap, err := decodeSocket(t.val)
			if err != nil {
				return st, err
			}
			st.dataDest = ap
			st.hasDataDest = true
		case stOutputSamprate:
			st.sampleRate = uint32(decodeUint(t.val))
		case stOutputEncoding:
			st.encoding = encoding(decodeUint(t.val))
		case stOutputChannels:
			st.channels = int(decodeUint(t.val))
		case stRadioFrequency:
			st.radioFreqHz = decodeFloat(t.val)
		}
	}
	return st, nil
}

// buildPoll builds a command packet that queries the channel's current status:
// a CMD with a command tag and the target SSRC, terminated by EOL. radiod
// replies on the status group with a STATUS packet for that SSRC.
func buildPoll(ssrc, tag uint32) []byte {
	buf := []byte{pktCommand}
	buf = appendInt(buf, stCommandTag, uint64(tag))
	buf = appendInt(buf, stOutputSSRC, uint64(ssrc))
	return appendEOL(buf)
}

// buildSetFreq builds a command packet that retunes the channel to hz, matching
// the `tune` client's RADIO_FREQUENCY command.
func buildSetFreq(ssrc, tag uint32, hz float64) []byte {
	buf := []byte{pktCommand}
	buf = appendInt(buf, stCommandTag, uint64(tag))
	buf = appendInt(buf, stOutputSSRC, uint64(ssrc))
	buf = appendDouble(buf, stRadioFrequency, hz)
	return appendEOL(buf)
}
