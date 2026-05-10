package framing

// Reed-Solomon (12, 9, 4) over GF(2^8) — the inner code DMR uses on
// top of BPTC(196,96) for Voice LC Header, Terminator with LC, and
// Embedded LC bursts. The 96 information bits BPTC reconstructs are
// 9 octets of Full Link Control + 3 octets of RS parity; verifying
// the parity tells the receiver whether the BPTC's correction was
// successful (BPTC reports its own success but doesn't catch
// systematic FEC misses).
//
// Parameters per ETSI TS 102 361-1 Annex B.3.12:
//
//	Field generator: GF(2^8) with primitive polynomial
//	  p(x) = x^8 + x^6 + x^5 + x + 1   (= 0x163, low 8 bits 0x63)
//	Field primitive: α = 2
//	Generator polynomial:
//	  g(x) = (x + α^0)(x + α^1)(x + α^2)
//	       = x^3 ⊕ (1 ⊕ α ⊕ α²)x² ⊕ (α ⊕ α² ⊕ α³)x ⊕ α³
//
// For the three DMR contexts the encoder XORs each parity octet with
// a context-specific seed before transmission, so the verifier has
// to un-XOR before computing syndromes:
//
//	Voice LC Header:       (0x96, 0x96, 0x96)
//	Terminator with LC:    (0x99, 0x99, 0x99)
//	Embedded LC:           (0x6A, 0x6A, 0x6A)
//
// This pass implements verify only — single-error correction (the
// code's nominal capability is t = 1) is a follow-up. BPTC already
// hands Voice LC Header bits a strong correction layer; the RS check
// here is mainly for catching BPTC misdecodes the inner CRC-16 of
// CSBK doesn't apply to.

// RS129 wraps the GF(2^8) tables RS verification needs. A single
// package-level instance is built at init time; callers go through
// the package-level VerifyRS12_9 helper.
var rs129 *rsField

type rsField struct {
	exp [256]byte // exp[i] = α^i in GF(2^8) (i in 0..254 unique, exp[255]=exp[0])
	log [256]byte // log[a] = i such that α^i = a (log[0] is unused)
}

// Generator polynomial coefficients g(x) = x^3 + g[2]x² + g[1]x + g[0].
// Computed once at init from the field tables.
var rs129Gen [3]byte

// DMR-specific seeds applied to the parity octets before transmission.
// VerifyRS12_9 un-XORs the parity bytes before computing syndromes.
var (
	RS129SeedVoiceLCHeader   = [3]byte{0x96, 0x96, 0x96}
	RS129SeedTerminatorLC    = [3]byte{0x99, 0x99, 0x99}
	RS129SeedEmbeddedLC      = [3]byte{0x6A, 0x6A, 0x6A}
	RS129SeedNone            = [3]byte{0x00, 0x00, 0x00}
)

func init() {
	f := &rsField{}
	x := byte(1)
	for i := 0; i < 255; i++ {
		f.exp[i] = x
		f.log[x] = byte(i)
		x = gfMul2(x)
	}
	f.exp[255] = f.exp[0]
	rs129 = f

	// Generator g(x) = (x + α^0)(x + α^1)(x + α^2). For α = 2:
	//   α^0 = 1, α^1 = 2, α^2 = 4, α^3 = 8 (no reduction yet at these powers).
	// Coefficients (low to high):
	//   g[0] = α^3                = 8
	//   g[1] = α + α² + α³        = 2 ⊕ 4 ⊕ 8 = 14
	//   g[2] = 1 ⊕ α ⊕ α²         = 1 ⊕ 2 ⊕ 4 = 7
	rs129Gen = [3]byte{8, 14, 7}
}

// gfMul2 multiplies a GF(2^8) element by α = 2: shift left, then
// reduce modulo the primitive polynomial 0x163 (low 8 bits 0x63) if
// the result overflowed bit 7.
func gfMul2(x byte) byte {
	high := x & 0x80
	out := x << 1
	if high != 0 {
		out ^= 0x63
	}
	return out
}

// gfMul multiplies two GF(2^8) elements via the log/exp tables.
func gfMul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	la := int(rs129.log[a])
	lb := int(rs129.log[b])
	return rs129.exp[(la+lb)%255]
}

// EncodeRS12_9 produces the 12-byte codeword for 9 data bytes:
// [data || parity], systematic, no XOR seed applied. Callers that
// need to match DMR transmission XOR the trailing 3 bytes with the
// appropriate seed after this returns.
//
// Convention: cw[0] holds the highest-degree coefficient (x^11) and
// cw[11] holds the constant term. Data fills cw[0..8] as the
// human-natural reading order (data[0] is most significant), and the
// 3-byte parity fills cw[9..11] with cw[9]=r_2, cw[10]=r_1,
// cw[11]=r_0.
func EncodeRS12_9(data [9]byte) [12]byte {
	// LFSR shift register: p[k] indexed by polynomial degree, so p[0]
	// is the constant term r_0 and p[2] is r_2.
	var p [3]byte
	for i := 0; i < 9; i++ {
		fb := data[i] ^ p[2]
		p[2] = p[1] ^ gfMul(fb, rs129Gen[2])
		p[1] = p[0] ^ gfMul(fb, rs129Gen[1])
		p[0] = gfMul(fb, rs129Gen[0])
	}
	var cw [12]byte
	copy(cw[0:9], data[:])
	cw[9] = p[2]  // r_2 → highest-degree parity slot
	cw[10] = p[1] // r_1
	cw[11] = p[0] // r_0 → constant term slot
	return cw
}

// VerifyRS12_9 takes a 12-byte codeword whose trailing 3 octets were
// XOR'd with seed at transmit time, un-XORs the parity, and reports
// whether the codeword satisfies the RS(12,9,4) parity equations
// (all three syndromes zero). Returns false on any 12-byte slice
// length other than 12.
func VerifyRS12_9(cw []byte, seed [3]byte) bool {
	if len(cw) != 12 {
		return false
	}
	// Un-XOR the parity bytes. Work on a local copy so callers'
	// buffers stay untouched.
	var c [12]byte
	copy(c[:], cw)
	for i := 0; i < 3; i++ {
		c[9+i] ^= seed[i]
	}
	// Syndromes via forward Horner.
	// Convention: c[0] is the leading (x^11) coefficient and c[11] is
	// the constant term, so c(x) = c[0]*x^11 + c[1]*x^10 + ... + c[11].
	// A valid codeword satisfies c(α^j) = 0 for j = 0, 1, 2.
	for j := 0; j < 3; j++ {
		alphaJ := rs129.exp[j] // α^j
		var s byte
		for i := 0; i < 12; i++ {
			s = gfMul(s, alphaJ) ^ c[i]
		}
		if s != 0 {
			return false
		}
	}
	return true
}
