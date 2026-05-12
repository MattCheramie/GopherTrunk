//go:build linux

package player

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// listPlatformDevices walks /dev/snd looking for playback PCM nodes
// matching pcmC{card}D{device}p. Each entry is returned as a string
// the operator can drop straight into audio.device. Empty when the
// host has no sound subsystem mounted (CI containers).
func listPlatformDevices() []string {
	entries, err := os.ReadDir("/dev/snd")
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		// "pcmC0D0p" → card=0 device=0 playback. Trailing 'p' means
		// playback; 'c' would be capture. We only care about the
		// playback side.
		if !strings.HasPrefix(name, "pcmC") || !strings.HasSuffix(name, "p") {
			continue
		}
		core := strings.TrimSuffix(strings.TrimPrefix(name, "pcmC"), "p")
		i := strings.IndexByte(core, 'D')
		if i < 0 {
			continue
		}
		out = append(out, fmt.Sprintf("ioctl:hw:%s,%s", core[:i], core[i+1:]))
	}
	sort.Strings(out)
	return out
}
