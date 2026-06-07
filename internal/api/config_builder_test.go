package api

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newCBService(t *testing.T, dir string) *configBuilderService {
	t.Helper()
	svc, err := newConfigBuilderService(ConfigBuilderOptions{Enabled: true, ConfigDir: dir})
	if err != nil {
		t.Fatalf("newConfigBuilderService: %v", err)
	}
	return svc
}

func TestWithinRoot(t *testing.T) {
	root := filepath.Clean("/a/b")
	cases := []struct {
		abs  string
		want bool
	}{
		{"/a/b", true},
		{"/a/b/c.yaml", true},
		{"/a/b/sub/c.yaml", true},
		{"/a/bc.yaml", false}, // sibling prefix, not nested
		{"/a/c.yaml", false},
		{"/etc/passwd", false},
		{"/a/b/../x.yaml", false}, // escapes after clean
	}
	for _, tc := range cases {
		got := withinRoot(root, filepath.Clean(tc.abs))
		if got != tc.want {
			t.Errorf("withinRoot(%q, %q) = %v, want %v", root, tc.abs, got, tc.want)
		}
	}
}

func TestResolvePath_AllowsNestedSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sites")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	svc := newCBService(t, dir)

	got, err := svc.resolvePath(filepath.Join(sub, "metro.yaml"))
	if err != nil {
		t.Fatalf("nested subdir should be allowed: %v", err)
	}
	if !strings.HasPrefix(got, canonicalDir(dir)) {
		t.Fatalf("resolved path %q not under root %q", got, canonicalDir(dir))
	}
}

func TestResolvePath_Rejects(t *testing.T) {
	dir := t.TempDir()
	svc := newCBService(t, dir)

	if _, err := svc.resolvePath(""); err == nil {
		t.Errorf("empty path should error")
	}
	if _, err := svc.resolvePath(filepath.Join(dir, "config.txt")); err == nil {
		t.Errorf("wrong extension should error")
	}
	if _, err := svc.resolvePath("/etc/passwd.yaml"); !errors.Is(err, errPathOutsideRoots) {
		t.Errorf("out-of-root path want errPathOutsideRoots, got %v", err)
	}
	if _, err := svc.resolvePath(filepath.Join(dir, "..", "escape.yaml")); !errors.Is(err, errPathOutsideRoots) {
		t.Errorf("dotdot escape want errPathOutsideRoots, got %v", err)
	}
}

func TestResolvePath_Symlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ on windows")
	}
	dir := t.TempDir()
	outside := t.TempDir()

	// A symlinked subdir inside the root that points outside must be
	// rejected for targets under it.
	link := filepath.Join(dir, "evil")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	svc := newCBService(t, dir)
	if _, err := svc.resolvePath(filepath.Join(link, "x.yaml")); !errors.Is(err, errPathOutsideRoots) {
		t.Errorf("symlink-escape target want errPathOutsideRoots, got %v", err)
	}

	// A root that is itself a symlink to a real dir: files inside resolve
	// and are accepted (roots are canonicalized at construction).
	realRoot := t.TempDir()
	rootLink := filepath.Join(t.TempDir(), "rootlink")
	if err := os.Symlink(realRoot, rootLink); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	svc2 := newCBService(t, rootLink)
	got, err := svc2.resolvePath(filepath.Join(rootLink, "config.yaml"))
	if err != nil {
		t.Fatalf("file under symlinked root should be allowed: %v", err)
	}
	if !strings.HasPrefix(got, canonicalDir(realRoot)) {
		t.Fatalf("resolved %q not under real root %q", got, canonicalDir(realRoot))
	}
}
