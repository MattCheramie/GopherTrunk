package wmbus

// maxFramesInFlight bounds how many frames one polarity frames at once.
// Two would cover the case seen on air (one telegram starting inside
// another); the third absorbs a chance access-code match inside a frame's
// data without costing a real telegram that follows it.
const maxFramesInFlight = 3

// framerBank frames one polarity's chip stream with several Framers, so a
// telegram that starts while another is being framed is still found.
//
// A single Framer stops hunting once it has an access code and does not
// look again until that frame's last byte. On a busy channel that loses
// whole telegrams (issue #1256, the reporter's 23 Nov run from a car):
// a meter keys up during another meter's telegram, the stronger signal
// takes over the receiver (an FM-style capture: the chip clock and the
// tone correlators follow whichever transmitter dominates), and the
// framer, still counting the first frame's bytes, never sees the second
// one's access code. The first frame then fails its CRC from the step on,
// and the second, stronger one is not heard at all. Several of the run's
// CRC failures show exactly that: the channel power steps up by 6-15 dB
// part-way through the frame and stays up past its end.
//
// So one Framer always hunts. When it finds an access code it joins the
// frames in flight and a fresh one takes over the hunt. A frame whose
// access code was found while another frame was in flight is "nested":
// most such access codes are chance matches inside the other frame's data
// (16 chips match at random once in 65536), so a nested frame is
// delivered only when its header block passes its CRC, which a chance
// match does once in 65536 again. A real telegram that loses its header
// to the collision would have been reported "CRC FAILED, sender
// unreliable" anyway.
type framerBank struct {
	all    []*bankFramer
	hunter *bankFramer
	active []*bankFramer
	free   []*bankFramer
	// deliver receives every frame the bank lets through.
	deliver func(Frame)
}

type bankFramer struct {
	*Framer
	nested bool
}

func newFramerBank(deliver func(Frame)) *framerBank {
	b := &framerBank{deliver: deliver}
	b.hunter = b.newFramer()
	return b
}

func (b *framerBank) newFramer() *bankFramer {
	bf := &bankFramer{}
	bf.Framer = NewFramer(func(f Frame) {
		if bf.nested && !f.HeaderCRCOK() {
			return
		}
		b.deliver(f)
	})
	b.all = append(b.all, bf)
	return bf
}

// busy reports that a frame is in flight.
func (b *framerBank) busy() bool { return len(b.active) > 0 }

// push feeds one chip to every frame in flight and to the hunter.
func (b *framerBank) push(chip uint8, rel float32) {
	kept := b.active[:0]
	for _, bf := range b.active {
		bf.PushSoft(chip, rel)
		if bf.Busy() {
			kept = append(kept, bf)
		} else {
			b.free = append(b.free, bf)
		}
	}
	for i := len(kept); i < len(b.active); i++ {
		b.active[i] = nil
	}
	b.active = kept

	b.hunter.PushSoft(chip, rel)
	if !b.hunter.Busy() {
		return
	}
	b.hunter.nested = len(b.active) > 0
	if len(b.active) == maxFramesInFlight {
		// Give up the oldest frame: by now the channel has carried two
		// more access codes since it began.
		old := b.active[0]
		old.Reset()
		copy(b.active, b.active[1:])
		b.active = b.active[:len(b.active)-1]
		b.free = append(b.free, old)
	}
	b.active = append(b.active, b.hunter)
	if n := len(b.free); n > 0 {
		b.hunter = b.free[n-1]
		b.free = b.free[:n-1]
	} else {
		b.hunter = b.newFramer()
	}
}

// stats sums the counters of every framer the bank has used.
func (b *framerBank) stats(s *FramerStats) {
	for _, bf := range b.all {
		fs := bf.Stats()
		s.Syncs += fs.Syncs
		s.Frames += fs.Frames
		s.CRCOK += fs.CRCOK
		s.CodingError += fs.CodingError
		s.BadLength += fs.BadLength
		s.Repaired += fs.Repaired
	}
}
