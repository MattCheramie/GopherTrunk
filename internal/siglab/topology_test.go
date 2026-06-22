package siglab

import (
	"bytes"
	"testing"

	"github.com/MattCheramie/GopherTrunk/internal/scanner/ccdecoder"
	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// fakeTopoPipeline is a no-op pipeline that reports a fixed topology, used to
// guard the generic factory path's pipe.(TopologyProvider) hook in engine.go
// (the P25 path uses the deep-bundle closure instead).
type fakeTopoPipeline struct{}

func (fakeTopoPipeline) Process([]complex64) {}
func (fakeTopoPipeline) Reset()              {}
func (fakeTopoPipeline) Close() error        { return nil }
func (fakeTopoPipeline) TopologySnapshot() *trunking.TopologySnapshot {
	return &trunking.TopologySnapshot{SystemID: 0x49A, ColorCode: 5}
}

func TestGenericPipelineTopologyAttached(t *testing.T) {
	restore := ccdecoder.SetTestFactory(trunking.ProtocolNXDN,
		func(ccdecoder.PipelineOptions) (ccdecoder.ProtocolPipeline, error) {
			return fakeTopoPipeline{}, nil
		})
	defer restore()

	buf := EncodeCapture(make([]complex64, 4096), FormatF32)
	res, err := RunReader(bytes.NewReader(buf), "fake", Config{
		Protocol:     trunking.ProtocolNXDN,
		SampleRateHz: 48_000,
		Format:       FormatF32,
	})
	if err != nil {
		t.Fatalf("RunReader: %v", err)
	}
	if res.Topology == nil {
		t.Fatal("Result.Topology not attached from the generic TopologyProvider hook")
	}
	if res.Topology.SystemID != 0x49A || res.Topology.ColorCode != 5 {
		t.Errorf("topology = %+v, want SystemID 49A ColorCode 5", res.Topology)
	}
}

func TestTopologySnapshotEmpty(t *testing.T) {
	if !(&TopologySnapshot{}).Empty() {
		t.Error("zero TopologySnapshot should be Empty")
	}
	if (*TopologySnapshot)(nil).Empty() != true {
		t.Error("nil TopologySnapshot should be Empty")
	}
	if (&TopologySnapshot{Neighbors: []NeighborRef{{Site: 1}}}).Empty() {
		t.Error("snapshot with a neighbor should not be Empty")
	}
}

// TestIdentifyReaderMatchesIdentify confirms the in-memory IdentifyReader
// produces the same winner as the file-based Identify over the same capture —
// the parity guarantee the live sweep relies on.
func TestIdentifyReaderMatchesIdentify(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}

	cfg := IdentifyConfig{SampleRateHz: meta.SampleRateHz, Format: FormatF32}
	fromIQ, err := IdentifyIQ(iq, "p25.iq", cfg)
	if err != nil {
		t.Fatalf("IdentifyIQ: %v", err)
	}
	if fromIQ.Winner != trunking.ProtocolP25.String() {
		t.Errorf("IdentifyIQ winner = %q, want p25", fromIQ.Winner)
	}
	if fromIQ.Inconclusive {
		t.Errorf("IdentifyIQ inconclusive (conf %.2f)", fromIQ.Confidence)
	}

	// Same buffer through a file → Identify should agree on the winner.
	dir := t.TempDir()
	path := dir + "/p25.cfile"
	if err := WriteCapture(path, iq, FormatF32); err != nil {
		t.Fatalf("WriteCapture: %v", err)
	}
	fromFile, err := Identify(path, cfg)
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if fromIQ.Winner != fromFile.Winner {
		t.Errorf("winner mismatch: IdentifyIQ=%q Identify=%q", fromIQ.Winner, fromFile.Winner)
	}
	if len(fromIQ.Candidates) != len(fromFile.Candidates) {
		t.Errorf("candidate count mismatch: IQ=%d file=%d", len(fromIQ.Candidates), len(fromFile.Candidates))
	}
}
