package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpand(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	t.Setenv("GT_PATHUTIL_VAR", "/opt/gt")

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain relative", "../recordings", "../recordings"},
		{"plain absolute", "/var/lib/gt", "/var/lib/gt"},
		{"tilde only", "~", home},
		{"tilde slash", "~/data", filepath.Join(home, "data")},
		{"posix var", "$GT_PATHUTIL_VAR/x", "/opt/gt/x"},
		{"posix braces", "${GT_PATHUTIL_VAR}/x", "/opt/gt/x"},
		{"windows var", "%GT_PATHUTIL_VAR%/x", "/opt/gt/x"},
		{"unknown windows var preserved", "%GT_NOPE%/x", "%GT_NOPE%/x"},
		{"dangling percent preserved", "50%done", "50%done"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Expand(c.in); got != c.want {
				t.Errorf("Expand(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
