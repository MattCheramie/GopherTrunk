//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// acquireInstanceLock takes an advisory POSIX flock on a sibling of
// the supplied config path so two daemons aimed at the same config
// can't both try to claim the same RTL-SDR dongles via libusb. The
// lock file holds the running PID + start timestamp so operators
// who hit the contention case can immediately see who's holding it.
//
// Returns a release function that must be called on shutdown (defer
// in main). releaseFn closes the file descriptor, which drops the
// lock; the file itself is intentionally left in place so subsequent
// starts can still inspect "who held this last?" forensically.
//
// Daemons started without a -config file (cfgPath == "") get a no-op
// lock — sharing built-in defaults across multiple instances is fine
// because no two of them can have configured the same SDR pool.
func acquireInstanceLock(cfgPath string) (releaseFn func(), err error) {
	if cfgPath == "" {
		return func() {}, nil
	}
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("instance lock: ensure dir %s: %w", dir, err)
	}
	lockPath := filepath.Join(dir, ".gophertrunk.lock")
	f, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("instance lock: open %s: %w", lockPath, err)
	}
	// Non-blocking exclusive flock — we want the second daemon to
	// fail fast, not hang waiting for the first to exit.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		// Read the existing file for forensic context — the previous
		// holder wrote its PID + start timestamp on acquisition.
		info := readLockInfo(lockPath)
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf(
				"another gophertrunk is running against %s%s; use a different -config or stop the other process",
				cfgPath, info)
		}
		return nil, fmt.Errorf("instance lock: flock %s: %w", lockPath, err)
	}

	// Truncate + write our identity so operators can see who owns
	// the lock.
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	body := fmt.Sprintf("pid=%d\nstarted=%s\nconfig=%s\n",
		os.Getpid(), time.Now().UTC().Format(time.RFC3339), cfgPath)
	_, _ = f.WriteString(body)

	release := func() {
		// Drop the lock; flock is released on FD close.
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
	return release, nil
}

// readLockInfo returns " (pid=42, started=…)" or "" if the lock
// file can't be parsed. Best-effort — surfaced to the operator
// purely for context.
func readLockInfo(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pid, started string
	for _, line := range strings.Split(string(data), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "pid":
			pid = strings.TrimSpace(v)
		case "started":
			started = strings.TrimSpace(v)
		}
	}
	if pid == "" && started == "" {
		return ""
	}
	parts := []string{}
	if pid != "" {
		// Verify the PID is still alive — if not, the holder
		// crashed without releasing and the lock should never have
		// been contended in the first place. Surface that case so
		// the operator knows to delete the file.
		if pidNum, err := strconv.Atoi(pid); err == nil {
			if !pidAlive(pidNum) {
				parts = append(parts, fmt.Sprintf("pid=%s (stale, process gone)", pid))
			} else {
				parts = append(parts, "pid="+pid)
			}
		} else {
			parts = append(parts, "pid="+pid)
		}
	}
	if started != "" {
		parts = append(parts, "started="+started)
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// pidAlive reports whether the process with the given PID is still
// running. POSIX-specific (kill 0).
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
