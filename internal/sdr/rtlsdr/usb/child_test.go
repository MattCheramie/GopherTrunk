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

func TestSerialFromInstanceID(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{`USB\VID_0BDA&PID_2838\90000002`, "90000002"},
		{`USB\VID_0BDA&PID_2838\7&227E3EE8&0&2`, "7&227E3EE8&0&2"}, // no programmed serial
		{`no-backslash`, ""},
		{``, ""},
	}
	for _, tt := range tests {
		if got := serialFromInstanceID(tt.id); got != tt.want {
			t.Errorf("serialFromInstanceID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

// TestPickInterfaceZeroChildTwoIdenticalDongles is the issue #1131 regression:
// two RTL2832U composite dongles share VID/PID, so a first-match child lookup
// hands every Open the SAME &MI_00 child. The daemon opens dongle A, then
// Open(serial=B) resolves to A's child again — WinUsb_Initialize on the
// already-owned handle fails with ERROR_NOT_ENOUGH_MEMORY and the second
// device never opens, even though both pass `sdr list --probe` (sequential
// opens never collide). Selection must follow the serial on the parent node.
func TestPickInterfaceZeroChildTwoIdenticalDongles(t *testing.T) {
	cands := []compositeChildCandidate{
		{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&AAAA&0&0000`, ParentInstanceID: `USB\VID_0BDA&PID_2838\90000002`},
		{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&BBBB&0&0000`, ParentInstanceID: `USB\VID_0BDA&PID_2838\00000001`},
	}
	if got := pickInterfaceZeroChild(cands, "00000001"); got != 1 {
		t.Fatalf("pickInterfaceZeroChild(serial=00000001) = %d, want 1 (issue #1131: first-match opens the wrong dongle)", got)
	}
	if got := pickInterfaceZeroChild(cands, "90000002"); got != 0 {
		t.Fatalf("pickInterfaceZeroChild(serial=90000002) = %d, want 0", got)
	}
}

func TestPickInterfaceZeroChild(t *testing.T) {
	twoDongles := []compositeChildCandidate{
		{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&AAAA&0&0000`, ParentInstanceID: `USB\VID_0BDA&PID_2838\90000002`},
		{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&BBBB&0&0000`, ParentInstanceID: `USB\VID_0BDA&PID_2838\00000001`},
	}
	tests := []struct {
		name   string
		cands  []compositeChildCandidate
		serial string
		want   int
	}{
		{"no candidates", nil, "00000001", -1},
		{"no serial keeps first-match behaviour", twoDongles, "", 0},
		{"case-insensitive serial match", []compositeChildCandidate{
			{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&AAAA&0&0000`, ParentInstanceID: `USB\VID_0BDA&PID_2838\ABCDEF01`},
		}, "abcdef01", 0},
		{"synthesized id (serial-less dongle) matches path serial", []compositeChildCandidate{
			{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&AAAA&0&0000`, ParentInstanceID: `USB\VID_0BDA&PID_2838\7&227E3EE8&0&2`},
		}, "7&227e3ee8&0&2", 0},
		{"no parent resolved anywhere → first (no information)", []compositeChildCandidate{
			{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&AAAA&0&0000`},
			{InstanceID: `USB\VID_0BDA&PID_2838&MI_00\7&BBBB&0&0000`},
		}, "00000001", 0},
		{"parents resolved but none match → -1, never the wrong dongle", twoDongles, "55555555", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pickInterfaceZeroChild(tt.cands, tt.serial); got != tt.want {
				t.Errorf("pickInterfaceZeroChild(serial=%q) = %d, want %d", tt.serial, got, tt.want)
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
