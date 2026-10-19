package framing

import (
	"encoding/hex"
	"testing"
)

// pn44RefWACN1Sys18NAC293 is the whole 4320-bit outbound superframe
// sequence for WACN 0x00001 / System 0x018 / NAC 0x293, MSB first, as
// OP25's apps/tdma/lfsr.py (p25p2_lfsr) generates it. SDRTrunk's
// LinearFeedbackShiftRegister produces the identical 4320 bits. Both
// decoders descramble live Phase 2 air with it, so this is the literal
// pin a self-consistent round-trip test cannot be (#1273: GopherTrunk's
// register tapped the mirrored polynomial and diverged at bit 49).
const pn44RefWACN1Sys18NAC293 = "0000101829362d3d7a7cc98feb8258b4e2a78e4e118f3f8a813c217dbd056e11e4e468ee912ee6232ba9826888e5678d83218115d599f7b7b4be8115ac7f2442cd3133f52b99ca23596a5cb30157e6aa49e1389d04a728269079630f01c51fb736e127fc50625662fccced4ac47869878b55c54ff4d1a032c6a49f82271b1dbb9f708190f0275e694c51ef86dab30f41dd800ed2daf74c9a5ea8bb0467fb00474e724a2527ed4a0c6b9d21519be46b727d5850dacd78c78e93e0d5a0a05f976237c8c4e0867ad9d7e4171de7a7c50168542556c5c5d46ce9608d8f5d2ca6de071d4f6763972a2dc2a1e420152e0fe15d087d351f98e3470f7dcd63331728870a50bcc50a8f7fcf98d1cb78a6b3542766f61f4424eb24f52ea0d51fd16a9c0743f7c5c6624c82dcade9fe6bee9f7ac34a35d061a93bb9e363851f868356d73003962e471ffffcc1506e64619fbf9f415318a9e3b365ae3bd4d1328a8d2021214fe93694be1084fb829fce2585267c9c9a0cf9141969c57a95a9236d44c13cd425cf1ca05907d28734b3161b41b355338fd6645fcd019bd254c36d453589f49f55b7ddb917c6982c10b9f971474d90eae53d39ca57b95ee9c9028b4bbcef705716cce6048869a3f427e89e61a9d4b16baad6503b9d6338618f9a75842928665210cd3ef4720c2404381102cde33266ad3a1eecad9cc7e705c3462effbbff3ffcb8d39a4f0bff8b9620821ef141f0eefbab140208c9b16083bfccabbd73"

// pn44RefMMR256 is the first 256 bits for the #915 Victorian MMR
// identity (WACN 0xBEE00, System 0x164, NAC 0x161), from the same OP25
// generator.
const pn44RefMMR256 = "bee001641617f79518929cb3425c77de3f853508c31306eb912b4ac7b33aeed9"

func pn44Bits(seed uint64, n int) []byte {
	s := NewPN44Scrambler(seed)
	out := make([]byte, n)
	for i := range out {
		out[i] = s.Next()
	}
	return out
}

func refBits(t *testing.T, h string) []byte {
	t.Helper()
	raw, err := hex.DecodeString(h)
	if err != nil {
		t.Fatal(err)
	}
	bits := make([]byte, 0, 8*len(raw))
	for _, b := range raw {
		for i := 7; i >= 0; i-- {
			bits = append(bits, b>>uint(i)&1)
		}
	}
	return bits
}

func TestPN44MatchesOP25AndSDRTrunk(t *testing.T) {
	for _, tc := range []struct {
		name     string
		wacn     uint32
		sys, nac uint16
		ref      string
	}{
		{"wacn1-sys18-nac293", 0x00001, 0x018, 0x293, pn44RefWACN1Sys18NAC293},
		{"mmr-915", 0xBEE00, 0x164, 0x161, pn44RefMMR256},
	} {
		want := refBits(t, tc.ref)
		got := pn44Bits(PN44SeedFromIdentity(tc.wacn, tc.sys, tc.nac), len(want))
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: bit %d = %d, reference %d", tc.name, i, got[i], want[i])
			}
		}
	}
}
