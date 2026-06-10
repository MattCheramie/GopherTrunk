//go:build windows

package main

import (
	"os"
	"os/exec"
)

// execSelf starts a fresh detached daemon process (argv/env already
// rewritten to carry the new -config) and exits the current one. Windows
// has no syscall.Exec image-replace, so the successor is a new process;
// the caller has already released the instance lock + SDRs so the new
// process can claim them. Returns only if the spawn fails.
func execSelf(exe string, argv, env []string) error {
	cmd := exec.Command(exe, argv[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil // unreachable
}
