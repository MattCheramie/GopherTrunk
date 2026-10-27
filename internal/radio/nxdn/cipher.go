package nxdn

// VoiceCipherTracker follows a traffic channel's SACCH superframe to learn
// the call's VCALL (parties, cipher type, key ID) and, for a scrambled call
// whose key the operator holds, descrambles the voice slots with the
// keystream schedule the issue #1261 known-key captures pinned:
//
//   - the register (Scrambler) is reseeded from the key at the first frame
//     of each SACCH superframe (structure field 3);
//   - it then runs continuously, ScramblerBitsPerFrame (4 × 49) bits per
//     LICH-valid air frame, voice frame or not — a FACCH frame between two
//     voice frames consumes its 196 bits too;
//   - voice slot v of a frame is XORed with that frame's bits [49v, 49v+49).
//
// Frames arrive one at a time (TrafficChannel.OnFrame). Alignment comes
// from any CRC-valid SACCH block's structure field and flywheels by frame
// count across blocks that fail their CRC; a frame the framer missed
// entirely desynchronises the flywheel until the next CRC-valid block.
//
// Before the first VCALL decodes the cipher is unknown, so with Hold set
// the tracker buffers voice (up to HoldLimit frames) and releases it once
// the VCALL says clear (as is) or scrambler (descrambled when a key is
// set); without Hold voice passes straight through and only the VCALL
// bookkeeping runs. Everything happens on the caller's goroutine.
type VoiceCipherTracker struct {
	// Hold buffers voice until the first VCALL names the cipher.
	Hold bool
	// HoldLimit caps the hold in LICH-valid frames (0 = 8, two
	// superframes); past it voice is released as received.
	HoldLimit int
	// OnVCALL runs when a VCALL superframe decodes with fields that differ
	// from the previous one (so once per call normally). A caller that
	// resolves the key sets it here, before held voice is released.
	OnVCALL func(VCALLMessage)

	// VCALL is the latest decoded VCALL; HaveVCALL whether one decoded.
	VCALL     VCALLMessage
	HaveVCALL bool

	key     uint16
	haveKey bool
	sf      SACCHSuperframe
	aligned bool
	sfStart int64 // frame index of the current superframe's first frame

	pending   []pendingVoice
	held      int
	holdOver  bool
	scrambler Scrambler

	// Counters for the chain's log lines.
	SACCHOK, SACCHFail uint64
	// Superframes counts completed SACCH superframes; VCALLs those that
	// parsed as VCALL.
	Superframes, VCALLs uint64
	// Descrambled counts voice frames XORed with the keystream; Ciphertext
	// those released as received from a scrambled call (no key, or not
	// aligned); Clear those from a call with no cipher.
	Descrambled, Ciphertext, Clear uint64
	// Unaligned counts scrambled voice frames released raw because no
	// SACCH block had fixed the superframe phase yet.
	Unaligned uint64
	// Unknown counts voice frames released before any VCALL named the
	// cipher (a pass-through tracker, or a hold that ran out).
	Unknown uint64
}

type pendingVoice struct {
	index  uint64
	frames []VCHFrame
}

// SetKey installs the scrambler key (1..ScramblerKeyMax).
func (t *VoiceCipherTracker) SetKey(key uint16) {
	t.key, t.haveKey = key&ScramblerKeyMax, true
}

// ClearKey forgets the key; scrambled voice is then released as ciphertext.
func (t *VoiceCipherTracker) ClearKey() { t.haveKey = false }

// HasKey reports whether a key is installed.
func (t *VoiceCipherTracker) HasKey() bool { return t.haveKey }

// Aligned reports whether the superframe phase (and so the keystream
// position) is known.
func (t *VoiceCipherTracker) Aligned() bool { return t.aligned }

// Frame processes one LICH-valid frame and returns the voice frames now
// ready for the vocoder, in order (possibly from earlier, held frames).
func (t *VoiceCipherTracker) Frame(f TrafficFrame) []VCHFrame {
	t.observeSACCH(f)
	var out []VCHFrame
	if t.HaveVCALL || !t.Hold || t.holdOver {
		out = t.release(nil)
		if len(f.Voice) > 0 {
			out = append(out, t.apply(f.Index, f.Voice)...)
		}
		return out
	}
	if len(f.Voice) > 0 {
		t.pending = append(t.pending, pendingVoice{index: f.Index, frames: f.Voice})
	}
	t.held++
	if t.held > t.holdLimit() {
		// No VCALL in two superframes: stop holding, release what we have
		// as received. A later VCALL resumes the normal policy for the
		// frames after it.
		t.holdOver = true
		return t.release(nil)
	}
	return nil
}

// Flush returns any held voice as received; call at end of call.
func (t *VoiceCipherTracker) Flush() []VCHFrame { return t.release(nil) }

func (t *VoiceCipherTracker) holdLimit() int {
	if t.HoldLimit > 0 {
		return t.HoldLimit
	}
	return 8
}

// release empties the hold through apply (now that the policy is known,
// or because the hold is over).
func (t *VoiceCipherTracker) release(out []VCHFrame) []VCHFrame {
	for _, p := range t.pending {
		out = append(out, t.apply(p.index, p.frames)...)
	}
	t.pending = t.pending[:0]
	t.held = 0
	return out
}

// apply descrambles frames (in place) when the call is scrambled, the key
// is known and the superframe phase is known; otherwise it counts why not
// and returns them as received.
func (t *VoiceCipherTracker) apply(index uint64, frames []VCHFrame) []VCHFrame {
	if !t.HaveVCALL {
		t.Unknown += uint64(len(frames))
		return frames
	}
	if t.VCALL.Cipher != CipherScrambler {
		t.Clear += uint64(len(frames))
		return frames
	}
	if !t.haveKey {
		t.Ciphertext += uint64(len(frames))
		return frames
	}
	if !t.aligned {
		t.Ciphertext += uint64(len(frames))
		t.Unaligned += uint64(len(frames))
		return frames
	}
	pos := int64(index) - t.sfStart
	pos = ((pos % 4) + 4) % 4
	t.scrambler.Reset(t.key)
	t.scrambler.Skip(int(pos) * ScramblerBitsPerFrame)
	ks := t.scrambler.Generate(ScramblerBitsPerFrame)
	for _, vf := range frames {
		if vf.Slot < 0 || vf.Slot > 3 || len(vf.Payload) != VoiceSlotBits {
			continue
		}
		k := ks[vf.Slot*VoiceSlotBits : (vf.Slot+1)*VoiceSlotBits]
		for i := range vf.Payload {
			vf.Payload[i] ^= k[i]
		}
		t.Descrambled++
	}
	return frames
}

// observeSACCH decodes the frame's SACCH block, keeps the superframe phase
// and assembles the layer-3 message.
func (t *VoiceCipherTracker) observeSACCH(f TrafficFrame) {
	if len(f.SACCH) != SACCHChannelBits {
		return
	}
	info, _, err := DecodeSACCH(f.SACCH)
	if err != nil {
		t.SACCHFail++
		return
	}
	t.SACCHOK++
	h, msg := ParseSACCHInfo(info)
	start := int64(f.Index) - int64(SACCHStructureFirst-h.Structure)
	if !t.aligned || start != t.sfStart {
		t.aligned, t.sfStart = true, start
	}
	sf, ok := t.sf.Push(h, msg)
	if !ok {
		return
	}
	t.Superframes++
	v, ok := ParseVCALLSuperframe(sf)
	if !ok {
		return
	}
	t.VCALLs++
	changed := !t.HaveVCALL || v != t.VCALL
	t.VCALL, t.HaveVCALL = v, true
	if changed && t.OnVCALL != nil {
		t.OnVCALL(v)
	}
}
