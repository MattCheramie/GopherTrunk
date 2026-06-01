package main

import "testing"

func TestLooksDriveRootedOnWindows(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Unix-style absolute paths lose their drive on Windows.
		{"/var/lib/gophertrunk/recordings", true},
		{"\\var\\lib\\gophertrunk\\recordings", true},
		{"/calls.db", true},
		// Drive-qualified and UNC paths are fine.
		{"C:\\Users\\me\\GopherTrunk", false},
		{"\\\\server\\share\\recordings", false},
		{"//server/share", false},
		// Relative paths are fine.
		{"recordings", false},
		{"data/recordings", false},
		{".\\recordings", false},
		{"", false},
	}
	for _, c := range cases {
		if got := looksDriveRootedOnWindows(c.path); got != c.want {
			t.Errorf("looksDriveRootedOnWindows(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
