// Static-binary audio wiring for Linux builds tagged nolibasound (#1230).
//
// The default Linux backend (alsa_linux.go) dlopens libasound2.so.2 through
// purego. purego links the binary against the glibc loader and libdl even
// with cgo off, so a default Linux build cannot start where neither exists —
// Android/Termux (Bionic libc) above all. Built with -tags nolibasound, the
// binary is fully static and every audio device goes to the direct-ioctl
// backend (ioctl_linux.go), which talks to /dev/snd itself. Where /dev/snd is
// not reachable either (Termux without root), opening the backend fails and
// the Player logs it and runs without sound — recordings, the web console and
// the CLI tools are unaffected.

//go:build linux && nolibasound

package player

import "strings"

func init() {
	defaultBackendFactory = func(cfg Config) (Backend, error) {
		spec := ""
		if strings.HasPrefix(cfg.Device, "ioctl:") {
			spec = strings.TrimPrefix(cfg.Device, "ioctl:")
		}
		return newIoctlALSABackend(cfg, spec)
	}
}
