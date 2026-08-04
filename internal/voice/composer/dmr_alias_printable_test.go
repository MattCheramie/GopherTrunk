package composer

import "testing"

func TestAliasPrintable(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"Truck 349", true},
		{"Café Unit", true}, // accented UTF-8 is legitimate
		{"", false},
		{"   ", false},         // whitespace only, no printable glyph
		{"Bad\x01Name", false}, // C0 control char → corrupted decode
		{"weird�", false},      // replacement rune → bad decode
	}
	for _, tc := range cases {
		if got := aliasPrintable(tc.in); got != tc.want {
			t.Errorf("aliasPrintable(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
