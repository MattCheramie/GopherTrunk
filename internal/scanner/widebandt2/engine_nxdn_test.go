package widebandt2

import (
	"encoding/binary"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
	"github.com/MattCheramie/GopherTrunk/internal/radio/nxdn"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// buildNXDNSpecStream assembles a ViterbiSpec on-air RCCH dibit stream:
// padding + outbound FSW + LICH + the spec-encoded 150-dibit CAC carrying a
// SITEINFO — built entirely from the exported nxdn encode chain, the same
// fixture shape the in-package process tests use.
func buildNXDNSpecStream(t *testing.T) []uint8 {
	t.Helper()
	lichInfo := nxdn.AssembleLICH(nxdn.LICH{RFCh: nxdn.RFChControl})
	lichDibits := framing.BitsToDibits(nxdn.EncodeLICHWire(lichInfo))

	var payload [8]byte
	binary.BigEndian.PutUint16(payload[0:2], 0xAAAA) // LocationID
	binary.BigEndian.PutUint16(payload[2:4], 0x1234) // SiteID
	binary.BigEndian.PutUint16(payload[4:6], 0x5678) // SystemID
	info := make([]byte, nxdn.CACInfoBits)
	typ := byte(nxdn.RCCHSITEINFO)
	for i := 0; i < 8; i++ {
		info[8+i] = (typ >> uint(7-i)) & 1
	}
	for b := 0; b < 8; b++ {
		for i := 0; i < 8; i++ {
			info[16+8*b+i] = (payload[b] >> uint(7-i)) & 1
		}
	}
	cacChannel := nxdn.EncodeCACChannel(info)
	if cacChannel == nil {
		t.Fatal("EncodeCACChannel returned nil")
	}

	stream := make([]uint8, 30)
	stream = append(stream, nxdn.FSWDibitsOutbound...)
	stream = append(stream, lichDibits...)
	stream = append(stream, framing.BitsToDibits(cacChannel)...)
	return stream
}

// TestBuildChannelNXDN pins the wideband NXDN channel: buildChannel accepts
// an NXDN system on a declared control channel, wires the spec Viterbi mode
// by default, and the tap's control channel decodes a SITEINFO frame to a
// lock plus a DecodedFrames bump (the diagnostics gate's counter).
func TestBuildChannelNXDN(t *testing.T) {
	bus := events.NewBus(16)
	defer bus.Close()
	sub := bus.Subscribe()
	defer sub.Close()
	log := slog.New(slog.NewTextHandler(discardWriter{}, nil))

	const ccHz = 453_125_000
	sys := trunking.System{
		Name:            "nxdn-sys",
		Protocol:        trunking.ProtocolNXDN,
		ControlChannels: []uint32{ccHz},
	}

	// Off-CC frequency must be rejected (requireControlChannel).
	if _, err := buildChannel(sys, ChannelConfig{FrequencyHz: ccHz + 25_000, SystemName: sys.Name}, narrowbandRateHz, bus, log, nil); err == nil {
		t.Fatal("buildChannel accepted an NXDN tap off the declared control channel")
	}

	ec, err := buildChannel(sys, ChannelConfig{FrequencyHz: ccHz, SystemName: sys.Name}, narrowbandRateHz, bus, log, nil)
	if err != nil {
		t.Fatalf("buildChannel(NXDN): %v", err)
	}
	if ec.protoTag != "nxdn" {
		t.Errorf("protoTag = %q, want %q", ec.protoTag, "nxdn")
	}
	cc, ok := ec.processor.(*nxdn.ControlChannel)
	if !ok {
		t.Fatalf("processor type = %T, want *nxdn.ControlChannel", ec.processor)
	}
	if got := cc.ViterbiMode(); got != nxdn.ViterbiSpec {
		t.Errorf("ViterbiMode = %v, want ViterbiSpec (the production default)", got)
	}

	cc.Process(buildNXDNSpecStream(t), 0)
	if got := ec.decoded(); got == 0 {
		t.Error("decoded() = 0 after a CRC-clean SITEINFO frame (diagnostics gate counter not wired)")
	}
	locked := false
	for {
		done := false
		select {
		case ev := <-sub.C:
			if ev.Kind == events.KindCCLocked {
				locked = true
			}
		default:
			done = true
		}
		if done {
			break
		}
	}
	if !locked {
		t.Error("no KindCCLocked after a valid SITEINFO frame through the wideband NXDN tap")
	}
}
