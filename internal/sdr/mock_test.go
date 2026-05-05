package sdr

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMockDeviceReplaysU8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "iq.cfile")
	const samples = 8192
	buf := make([]byte, samples*2)
	for i := range buf {
		buf[i] = byte(127 + (i % 7) - 3)
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		t.Fatal(err)
	}

	drv := &MockDriver{Files: []string{path}}
	dev, err := drv.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := dev.SetSampleRate(8_000_000); err != nil { // run faster than realtime
		t.Fatal(err)
	}
	defer dev.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := dev.StreamIQ(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got := 0
	for chunk := range ch {
		got += len(chunk)
		if got >= samples {
			break
		}
	}
	if got < samples {
		t.Errorf("received %d samples, want >= %d", got, samples)
	}
}

func TestMockDriverEnumerate(t *testing.T) {
	drv := &MockDriver{Files: []string{"a.cfile", "b.cfile"}}
	infos, err := drv.Enumerate()
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 2 || infos[1].Index != 1 {
		t.Errorf("infos = %+v", infos)
	}
}
