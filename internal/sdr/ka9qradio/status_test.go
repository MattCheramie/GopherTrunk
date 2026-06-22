package ka9qradio

import (
	"math"
	"net/netip"
	"testing"
)

func TestIntTLVRoundTrip(t *testing.T) {
	cases := []uint64{0, 1, 255, 256, 24000, 162550000, math.MaxUint32}
	for _, v := range cases {
		buf := appendInt(nil, stOutputSamprate, v)
		tlvs, err := parseTLVs(buf)
		if err != nil {
			t.Fatalf("parseTLVs(%d): %v", v, err)
		}
		if len(tlvs) != 1 || tlvs[0].typ != stOutputSamprate {
			t.Fatalf("v=%d: unexpected tlvs %+v", v, tlvs)
		}
		if got := decodeUint(tlvs[0].val); got != v {
			t.Fatalf("v=%d: decoded %d", v, got)
		}
	}
	// Zero compresses to a zero-length value.
	if buf := appendInt(nil, stOutputSSRC, 0); len(buf) != 2 || buf[1] != 0 {
		t.Fatalf("zero not compressed: % x", buf)
	}
}

func TestDoubleTLVRoundTrip(t *testing.T) {
	for _, f := range []float64{0, 162550000, 446.00625e6, 1e9} {
		buf := appendDouble(nil, stRadioFrequency, f)
		tlvs, _ := parseTLVs(buf)
		if len(tlvs) != 1 {
			t.Fatalf("f=%v: tlvs %+v", f, tlvs)
		}
		if got := decodeFloat(tlvs[0].val); got != f {
			t.Fatalf("f=%v: decoded %v", f, got)
		}
	}
}

func TestDecodeSocket(t *testing.T) {
	// IPv4: 239.10.20.30:5004.
	v4 := []byte{239, 10, 20, 30, 0x13, 0x8c} // 0x138c = 5004
	ap, err := decodeSocket(v4)
	if err != nil {
		t.Fatalf("decodeSocket v4: %v", err)
	}
	if ap.String() != "239.10.20.30:5004" {
		t.Fatalf("v4 got %s", ap)
	}
	// Wrong length.
	if _, err := decodeSocket([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected error on bad socket length")
	}
}

// buildStatusReply assembles a radiod-style STATUS packet for one channel.
func buildStatusReply(ssrc uint32, dataIP netip.Addr, dataPort uint16, samprate uint32, enc encoding, channels int, freq float64) []byte {
	buf := []byte{pktStatus}
	buf = appendInt(buf, stOutputSSRC, uint64(ssrc))
	sock := make([]byte, 6)
	a4 := dataIP.As4()
	copy(sock[0:4], a4[:])
	sock[4] = byte(dataPort >> 8)
	sock[5] = byte(dataPort)
	buf = appendTLV(buf, stOutputDataDestSocket, sock)
	buf = appendInt(buf, stOutputSamprate, uint64(samprate))
	buf = appendInt(buf, stOutputEncoding, uint64(enc))
	buf = appendInt(buf, stOutputChannels, uint64(channels))
	buf = appendDouble(buf, stRadioFrequency, freq)
	return appendEOL(buf)
}

func TestParseStatus(t *testing.T) {
	pkt := buildStatusReply(162550, netip.AddrFrom4([4]byte{239, 1, 2, 3}), 5004, 24000, encF32LE, 2, 162550000)
	st, err := parseStatus(pkt)
	if err != nil {
		t.Fatalf("parseStatus: %v", err)
	}
	if !st.hasSSRC || st.ssrc != 162550 {
		t.Fatalf("ssrc: %+v", st)
	}
	if !st.hasDataDest || st.dataDest.String() != "239.1.2.3:5004" {
		t.Fatalf("dataDest: %+v", st)
	}
	if st.sampleRate != 24000 || st.encoding != encF32LE || st.channels != 2 {
		t.Fatalf("params: %+v", st)
	}
	if st.radioFreqHz != 162550000 {
		t.Fatalf("freq: %v", st.radioFreqHz)
	}
}

func TestBuildPollAndSetFreq(t *testing.T) {
	poll := buildPoll(0x1234, 0xABCD)
	if poll[0] != pktCommand {
		t.Fatalf("poll not a command: % x", poll)
	}
	tlvs, err := parseTLVs(poll[1:])
	if err != nil {
		t.Fatalf("parse poll: %v", err)
	}
	var sawTag, sawSSRC bool
	for _, tv := range tlvs {
		switch tv.typ {
		case stCommandTag:
			sawTag = decodeUint(tv.val) == 0xABCD
		case stOutputSSRC:
			sawSSRC = decodeUint(tv.val) == 0x1234
		}
	}
	if !sawTag || !sawSSRC {
		t.Fatalf("poll missing tag/ssrc: %+v", tlvs)
	}

	cmd := buildSetFreq(0x1234, 1, 446006250)
	st, err := parseStatus(cmd) // parseStatus reads any TLV packet
	if err != nil {
		t.Fatalf("parse setfreq: %v", err)
	}
	if st.radioFreqHz != 446006250 {
		t.Fatalf("setfreq carried %v", st.radioFreqHz)
	}
}

func TestParseTLVsTruncated(t *testing.T) {
	// type with a length claiming more bytes than present.
	if _, err := parseTLVs([]byte{byte(stOutputSamprate), 4, 0, 0}); err == nil {
		t.Fatal("expected overrun error")
	}
}
