//go:build !windows

package main

import "syscall"

// execSelf replaces the current process image with a fresh invocation of
// the daemon (argv/env already rewritten to carry the new -config). Because
// it keeps the same PID and re-runs from scratch, every config field —
// including restart-required ones like SDR and trunking systems — takes
// effect. Returns only if exec fails.
func execSelf(exe string, argv, env []string) error {
	return syscall.Exec(exe, argv, env)
}
