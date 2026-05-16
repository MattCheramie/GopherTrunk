//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// acquireInstanceLock takes a Windows-style exclusive open on a
// sibling of the supplied config path. Unlike the POSIX flock
// version, this uses O_EXCL semantics — the file is created
// exclusively at acquire and removed at release.
//
// Returns a release function that must be called on shutdown.
// Daemons started without a -config file get a no-op lock.
func acquireInstanceLock(cfgPath string) (releaseFn func(), err error) {
	if cfgPath == "" {
		return func() {}, nil
	}
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("instance lock: ensure dir %s: %w", dir, err)
	}
	lockPath := filepath.Join(dir, ".gophertrunk.lock")
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf(
				"another gophertrunk is running against %s (lock file %s exists; if no other instance is running, delete it)",
				cfgPath, lockPath)
		}
		return nil, fmt.Errorf("instance lock: open %s: %w", lockPath, err)
	}
	body := fmt.Sprintf("pid=%d\nstarted=%s\nconfig=%s\n",
		os.Getpid(), time.Now().UTC().Format(time.RFC3339), cfgPath)
	_, _ = f.WriteString(body)

	release := func() {
		_ = f.Close()
		_ = os.Remove(lockPath)
	}
	return release, nil
}
