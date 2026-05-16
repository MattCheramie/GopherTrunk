//go:build !windows

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstanceLock_EmptyPathIsNoop(t *testing.T) {
	release, err := acquireInstanceLock("")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if release == nil {
		t.Fatal("release func is nil")
	}
	release()
}

func TestInstanceLock_AcquireAndRelease(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("log:\n  level: info\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	release, err := acquireInstanceLock(cfgPath)
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// Second acquire on same path should fail with a clear message.
	_, err2 := acquireInstanceLock(cfgPath)
	if err2 == nil {
		t.Fatal("expected second acquire to fail")
	}
	if !strings.Contains(err2.Error(), "another gophertrunk is running") {
		t.Errorf("want 'another gophertrunk is running' in error, got %q", err2.Error())
	}

	// After release, a fresh acquire succeeds.
	release()
	release2, err := acquireInstanceLock(cfgPath)
	if err != nil {
		t.Fatalf("re-acquire after release: %v", err)
	}
	release2()
}

func TestInstanceLock_LockFileContents(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("log:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	release, err := acquireInstanceLock(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	body, err := os.ReadFile(filepath.Join(dir, ".gophertrunk.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "pid=") || !strings.Contains(string(body), "started=") {
		t.Errorf("lock file should record pid + started timestamp, got %q", string(body))
	}
}
