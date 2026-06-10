package voice

import "testing"

// TestDormantSessionCloseNoPanic reproduces the shutdown crash where
// Recorder.Close finalized a dormant post-segment session (wav == nil)
// and dereferenced the nil WavWriter at wav.go's Close — a nil-pointer
// panic. close() must treat a nil writer as a no-op.
func TestDormantSessionCloseNoPanic(t *testing.T) {
	s := &recordingSession{} // dormant: wav, raw, vocoder all nil
	if err := s.close(); err != nil {
		t.Fatalf("dormant session close() = %v, want nil", err)
	}
}

// TestWavWriterNilClose pins the defensive nil-receiver guard on
// WavWriter.Close.
func TestWavWriterNilClose(t *testing.T) {
	var w *WavWriter
	if err := w.Close(); err != nil {
		t.Fatalf("(*WavWriter)(nil).Close() = %v, want nil", err)
	}
}
