package demod

import "math"

// DQPSK is a differential QPSK demodulator suitable for P25 Phase 2 H-DQPSK.
// At each symbol time, output is the phase difference between the current
// and previous IQ samples, mapped to one of the four dibits {00, 01, 10, 11}.
//
// For Phase 2 H-DQPSK, symbols rotate by π/4-multiples around an offset of
// π/8 — call SetRotation(math.Pi/8) to compensate.
type DQPSK struct {
	last     complex64
	rotation float64
}

func NewDQPSK() *DQPSK {
	return &DQPSK{last: complex(1, 0)}
}

func (d *DQPSK) SetRotation(radians float64) { d.rotation = radians }

// Decode emits one dibit per input sample. Caller should pre-decimate to
// one-sample-per-symbol via a clock-recovery stage.
func (d *DQPSK) Decode(dst []uint8, src []complex64) []uint8 {
	if cap(dst) < len(src) {
		dst = make([]uint8, len(src))
	} else {
		dst = dst[:len(src)]
	}
	for i, s := range src {
		// Phase delta: arg(s * conj(last)) − rotation.
		ar := real(s)*real(d.last) + imag(s)*imag(d.last)
		ai := imag(s)*real(d.last) - real(s)*imag(d.last)
		phi := math.Atan2(float64(ai), float64(ar)) - d.rotation
		// Wrap to [-π, π].
		for phi < -math.Pi {
			phi += 2 * math.Pi
		}
		for phi >= math.Pi {
			phi -= 2 * math.Pi
		}
		// Quadrant: nearest of {0, π/2, π, -π/2}.
		switch {
		case phi >= -math.Pi/4 && phi < math.Pi/4:
			dst[i] = 0b00
		case phi >= math.Pi/4 && phi < 3*math.Pi/4:
			dst[i] = 0b01
		case phi >= -3*math.Pi/4 && phi < -math.Pi/4:
			dst[i] = 0b11
		default:
			dst[i] = 0b10
		}
		d.last = s
	}
	return dst
}
