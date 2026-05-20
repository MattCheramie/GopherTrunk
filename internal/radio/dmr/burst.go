package dmr

// Burst is one DMR burst (264 transmitted bits across 132 dibits = 27.5 ms).
//
// Layout per ETSI TS 102 361-1 §6.2 / §6.4.2 for data and control bursts:
//
//   dibits   0..48    (49 dibits, 98 bits)  — info[0]: payload first half
//   dibits  49..53    (5 dibits, 10 bits)   — slot type before sync
//   dibits  54..77    (24 dibits, 48 bits)  — sync / embedded signaling
//   dibits  78..82    (5 dibits, 10 bits)   — slot type after sync
//   dibits  83..131   (49 dibits, 98 bits)  — info[1]: payload second half
//
// The two 98-bit info halves concatenate to form one 196-bit BPTC(196,96)
// codeword for control/data bursts (CSBKs etc.). Voice bursts use a
// different split — 108 + 48 + 108 bits, with no slot-type fields —
// carrying three 72-bit AMBE+2 frames; see internal/radio/dmr/voice.

const (
	BurstDibits       = 132
	BurstBits         = 264
	HalfPayloadDibits = 49
	HalfPayloadBits   = 98
	SlotTypeDibits    = 5
	SlotTypeBits      = 10
	SyncDibits        = 24
	SyncBitCount      = 48
	BPTCPayloadBits   = 196
)

// Burst stores the 132 dibits of one TDMA burst.
type Burst struct {
	Dibits [BurstDibits]uint8
}

// FirstHalf returns the 49 dibits of the first payload half.
func (b *Burst) FirstHalf() []uint8 { return b.Dibits[0:HalfPayloadDibits] }

// SecondHalf returns the 49 dibits of the second payload half.
func (b *Burst) SecondHalf() []uint8 { return b.Dibits[BurstDibits-HalfPayloadDibits : BurstDibits] }

// Sync returns the 24 dibits of the sync / embedded-signaling field.
func (b *Burst) Sync() []uint8 {
	const start = HalfPayloadDibits + SlotTypeDibits
	return b.Dibits[start : start+SyncDibits]
}

// SlotTypeBefore returns the 5 dibits of the slot-type field that precede
// the sync.
func (b *Burst) SlotTypeBefore() []uint8 {
	const start = HalfPayloadDibits
	return b.Dibits[start : start+SlotTypeDibits]
}

// SlotTypeAfter returns the 5 dibits of the slot-type field that follow
// the sync.
func (b *Burst) SlotTypeAfter() []uint8 {
	const start = HalfPayloadDibits + SlotTypeDibits + SyncDibits
	return b.Dibits[start : start+SlotTypeDibits]
}

// SlotTypeBits returns the 20-bit concatenation of the two 10-bit slot-
// type fields surrounding the sync, MSB-first.
func (b *Burst) SlotTypeBitsAll() []byte {
	out := make([]byte, 0, 20)
	for _, d := range b.SlotTypeBefore() {
		out = append(out, (d>>1)&1, d&1)
	}
	for _, d := range b.SlotTypeAfter() {
		out = append(out, (d>>1)&1, d&1)
	}
	return out
}

// PayloadBits concatenates first and second halves into 196 bits, suitable
// for direct hand-off to framing.DecodeBPTC196_96 for control/data bursts.
func (b *Burst) PayloadBits() []byte {
	out := make([]byte, BPTCPayloadBits)
	idx := 0
	for _, d := range b.FirstHalf() {
		out[idx] = (d >> 1) & 1
		idx++
		out[idx] = d & 1
		idx++
	}
	for _, d := range b.SecondHalf() {
		out[idx] = (d >> 1) & 1
		idx++
		out[idx] = d & 1
		idx++
	}
	return out[:BPTCPayloadBits]
}
