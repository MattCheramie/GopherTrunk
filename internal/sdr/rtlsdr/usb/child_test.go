package usb

import "testing"

func TestParseInstanceID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		vid   uint16
		pid   uint16
		mi    int
		hasMI bool
	}{
		{"composite child interface 0", `USB\VID_0BDA&PID_2838&MI_00\6&1234abcd&0&0000`, 0x0bda, 0x2838, 0, true},
		{"composite child interface 1", `USB\VID_0BDA&PID_2838&MI_01\6&1234abcd&0&0001`, 0x0bda, 0x2838, 1, true},
		{"single-interface whole device (no MI)", `USB\VID_0BDA&PID_2832\77771111153705700`, 0x0bda, 0x2832, 0, false},
		{"lower-case tokens", `usb\vid_1209&pid_2832&mi_00\x`, 0x1209, 0x2832, 0, true},
		{"garbage", `nonsense`, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vid, pid, mi, hasMI := parseInstanceID(tt.id)
			if vid != tt.vid || pid != tt.pid || mi != tt.mi || hasMI != tt.hasMI {
				t.Errorf("parseInstanceID(%q) = (%#04x,%#04x,%d,%v), want (%#04x,%#04x,%d,%v)",
					tt.id, vid, pid, mi, hasMI, tt.vid, tt.pid, tt.mi, tt.hasMI)
			}
		})
	}
}

func TestIsInterfaceZero(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{`USB\VID_0BDA&PID_2838&MI_00\x`, true},
		{`USB\VID_0BDA&PID_2838&MI_01\x`, false},
		{`USB\VID_0BDA&PID_2832\x`, false}, // single-interface: no MI token
		{`garbage`, false},
	}
	for _, tt := range tests {
		if got := isInterfaceZero(tt.id); got != tt.want {
			t.Errorf("isInterfaceZero(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestEffectiveCompositeService(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		found  bool
		want   string
	}{
		{"composite parent + winusb child → child wins (OK)", "usbccgp", "WinUSB", true, "WinUSB"},
		{"composite parent + libusbK child → child wins (accurate hint)", "usbccgp", "libusbK", true, "libusbK"},
		{"composite parent, no child found → keep parent (BAD)", "usbccgp", "", false, "usbccgp"},
		{"non-composite parent is never overridden", "WinUSB", "anything", true, "WinUSB"},
		{"case-insensitive parent match", "USBCCGP", "WinUSB", true, "WinUSB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := effectiveCompositeService(tt.parent, tt.child, tt.found); got != tt.want {
				t.Errorf("effectiveCompositeService(%q,%q,%v) = %q, want %q",
					tt.parent, tt.child, tt.found, got, tt.want)
			}
		})
	}
}

func TestFirstInterfaceGUID(t *testing.T) {
	tests := []struct {
		name   string
		in     []string
		want   string
		wantOK bool
	}{
		{"single", []string{"{abc}"}, "{abc}", true},
		{"skips blank padding", []string{"", "  ", "{def}"}, "{def}", true},
		{"trims whitespace", []string{"  {ghi}  "}, "{ghi}", true},
		{"empty list", nil, "", false},
		{"all blank", []string{"", " "}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := firstInterfaceGUID(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("firstInterfaceGUID(%q) = (%q,%v), want (%q,%v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
