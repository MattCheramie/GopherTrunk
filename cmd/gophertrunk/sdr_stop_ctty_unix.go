//go:build unix

package main

import "os"

// hasControllingTerminal reports whether the process has a controlling
// terminal: /dev/tty opens only when it does (ENXIO otherwise).
func hasControllingTerminal() bool {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
