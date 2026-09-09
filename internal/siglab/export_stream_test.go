package siglab

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/MattCheramie/GopherTrunk/internal/trunking"
)

// TestJSONLStreamerMatchesBatchExport pins that streaming events live through
// JSONLEventStreamer + WriteJSONLSummary produces output byte-identical to the
// batch FormatJSONL export — the contract that lets `replay -out-format jsonl`
// stream live on a pipe (issue #314) without changing what file-based callers
// parse.
func TestJSONLStreamerMatchesBatchExport(t *testing.T) {
	var live bytes.Buffer
	streamer := NewJSONLEventStreamer(&live)
	res, _, err := SynthesizeAndAnalyze(
		SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32},
		Config{},
		streamer.OnEvent,
	)
	if err != nil {
		t.Fatalf("SynthesizeAndAnalyze: %v", err)
	}
	if err := streamer.Err(); err != nil {
		t.Fatalf("streamer error: %v", err)
	}
	if err := WriteJSONLSummary(&live, res); err != nil {
		t.Fatalf("WriteJSONLSummary: %v", err)
	}

	var batch bytes.Buffer
	if err := WriteResult(&batch, res, FormatJSONL); err != nil {
		t.Fatalf("WriteResult(FormatJSONL): %v", err)
	}
	if !bytes.Equal(live.Bytes(), batch.Bytes()) {
		t.Fatalf("live-streamed JSONL differs from batch export:\nlive:\n%s\nbatch:\n%s",
			live.String(), batch.String())
	}
	if len(res.Events) == 0 {
		t.Fatal("fixture produced no events; the comparison proved nothing")
	}
}

// gatedEOFReader serves its data normally, then BLOCKS before returning EOF
// until release is closed — modelling a live pipe that has delivered samples
// but not hung up. It lets the test below distinguish "events are delivered as
// they are decoded" from "events only appear once the stream closes".
type gatedEOFReader struct {
	data    *bytes.Reader
	release <-chan struct{}
	timeout time.Duration
}

func (g *gatedEOFReader) Read(p []byte) (int, error) {
	n, err := g.data.Read(p)
	if err != io.EOF {
		return n, err
	}
	select {
	case <-g.release:
	case <-time.After(g.timeout):
	}
	return n, io.EOF
}

// TestRunReaderStreamDeliversEventsBeforeEOF pins the live-pipe contract behind
// `replay -in - -out-format jsonl` (issue #314): onEvent fires while the input
// stream is still open. A batch implementation that buffers events until EOF
// would time out here.
func TestRunReaderStreamDeliversEventsBeforeEOF(t *testing.T) {
	iq, meta, err := Synthesize(SynthOptions{Protocol: trunking.ProtocolP25, Format: FormatF32})
	if err != nil {
		t.Fatalf("Synthesize: %v", err)
	}
	proto, err := ParseProtocolCLI(meta.Protocol)
	if err != nil {
		t.Fatalf("ParseProtocolCLI(%q): %v", meta.Protocol, err)
	}
	cfg := Config{
		Protocol:     proto,
		SystemName:   "stream-test",
		SampleRateHz: meta.SampleRateHz,
		Format:       FormatF32,
	}

	release := make(chan struct{})
	var once sync.Once
	gotEvent := make(chan struct{})
	onEvent := func(EventRecord) {
		once.Do(func() {
			close(gotEvent)
			close(release) // first event seen while EOF is held back: let the stream end
		})
	}
	r := &gatedEOFReader{
		data:    bytes.NewReader(EncodeCapture(iq, FormatF32)),
		release: release,
		timeout: 30 * time.Second,
	}
	res, err := RunReaderStream(r, "gated-pipe", cfg, onEvent)
	if err != nil {
		t.Fatalf("RunReaderStream: %v", err)
	}
	select {
	case <-gotEvent:
	default:
		t.Fatalf("no event was delivered while the stream was still open (events=%d after EOF) — live streaming is broken for unbounded pipes (#314)", len(res.Events))
	}
}
