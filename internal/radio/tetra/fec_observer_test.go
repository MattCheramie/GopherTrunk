package tetra

import (
	"io"
	"log/slog"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/events"
	"github.com/MattCheramie/GopherTrunk/internal/radio/framing"
)

// buildSBStreamWithBSCHErrors assembles a synchronisation burst (BSCH +
// STS + broadcast + BNCH SYSINFO) as a hard dibit stream, flipping the
// first nBitErrors bits of the 120-bit BSCH type-5 block. The BNCH is
// left clean. Returns the stream plus the cell's extended colour code.
//
// The geometry mirrors processSB: pad, BSCH[L-60,L), STS, broadcast(15),
// BNCH[L+34,L+142), trailing margin so the look-ahead is satisfied.
func buildSBStreamWithBSCHErrors(t *testing.T, cc6 uint8, mcc, mnc, la uint16, nBitErrors int) ([]uint8, uint32) {
	t.Helper()
	syncInfo := make([]byte, 60)
	putBits(syncInfo, 4, 6, uint32(cc6))
	putBits(syncInfo, 31, 10, uint32(mcc))
	putBits(syncInfo, 41, 14, uint32(mnc))

	bschBits := EncodeBSCH(syncInfo) // 120 type-5 bits
	for i := 0; i < nBitErrors && i < len(bschBits); i++ {
		bschBits[i] ^= 1
	}
	bschDibits := TetraBitsToDibits(bschBits)

	ext := ExtendedColourCode(mcc, mnc, cc6)
	payload := make([]byte, 11)
	payload[3] = byte((la >> 6) & 0xFF)
	payload[4] = byte((la & 0x3F) << 2)
	pdu := PDU{Disc: DiscMLE, Type: uint8(MLESystemInfo), Payload: payload}
	bnchBits := EncodeSCHHD(framing.UnpackBitsMSB(AssemblePDU(pdu), 124), ext)
	bnchDibits := TetraBitsToDibits(bnchBits)

	var stream []uint8
	stream = append(stream, make([]uint8, 50)...)                // pad before BSCH
	stream = append(stream, bschDibits...)                       // BSCH (60)
	stream = append(stream, SyncTrainingDibits()...)             // STS (19)
	stream = append(stream, make([]uint8, sbBroadcastDibits)...) // broadcast (15)
	stream = append(stream, bnchDibits...)                       // BNCH (108)
	stream = append(stream, make([]uint8, 300)...)               // trailing look-ahead margin
	return stream, ext
}

// TestFECObserverReportsBSCHCorrectionDepth confirms the FEC observer
// receives the exact number of channel bits the §8.3.1 BSCH decode chain
// corrected: zero on a clean burst, k when k recoverable bit errors are
// injected. This is the data behind the opt-in
// gophertrunk_tetra_viterbi_corrections histogram (issue #648 follow-up).
func TestFECObserverReportsBSCHCorrectionDepth(t *testing.T) {
	for _, nErr := range []int{0, 1, 2} {
		t.Run("", func(t *testing.T) {
			type sample struct {
				channel string
				corr    int
			}
			var got []sample

			bus := events.NewBus(8)
			defer bus.Close()
			cc := New(Options{
				Bus:         bus,
				Log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
				SystemName:  "Sys",
				FrequencyHz: 412_000_000,
				FECObserver: func(channel string, corrections int) {
					got = append(got, sample{channel, corrections})
				},
			})
			cc.SetChannelCoding(ChannelCodingOn)

			stream, wantColour := buildSBStreamWithBSCHErrors(t, 0x2D, 0x0CE, 0x1234, 0x2B7, nErr)
			cc.Process(stream, 0)

			// BSCH correction depth must equal the injected error count.
			var sawBSCH bool
			for _, s := range got {
				if s.channel == "bsch" {
					sawBSCH = true
					if s.corr != nErr {
						t.Errorf("bsch correction depth = %d, want %d", s.corr, nErr)
					}
				}
			}
			if !sawBSCH {
				t.Fatalf("FEC observer saw no bsch sample (got %+v); BSCH failed to decode at %d injected errors", got, nErr)
			}

			// The clean BNCH should report zero corrections.
			for _, s := range got {
				if s.channel == "schhd" && s.corr != 0 {
					t.Errorf("clean schhd correction depth = %d, want 0", s.corr)
				}
			}

			// Sanity: the colour code was still learned despite the errors.
			if topo := cc.Topology(); topo.ColourCode != wantColour {
				t.Errorf("learned colour = %#x, want %#x", topo.ColourCode, wantColour)
			}
		})
	}
}

// TestFECObserverNilByDefault confirms the correction-depth path is inert
// when no observer is wired — the production default unless
// metrics.detailed_fec is enabled.
func TestFECObserverNilByDefault(t *testing.T) {
	bus := events.NewBus(8)
	defer bus.Close()
	cc := New(Options{Bus: bus, Log: slog.Default(), SystemName: "Sys", FrequencyHz: 412_000_000})
	cc.SetChannelCoding(ChannelCodingOn)
	stream, _ := buildSBStreamWithBSCHErrors(t, 0x11, 0x123, 0x2345, 0x0AB, 0)
	// Must not panic with a nil observer; the SB burst still locks.
	cc.Process(stream, 0)
}
