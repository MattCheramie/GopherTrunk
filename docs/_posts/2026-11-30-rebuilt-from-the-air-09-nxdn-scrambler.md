---
title: "Rebuilt From the Air, Part 9: NXDN III — The Scrambler, and the Key That Ranked First"
description: "GopherTrunk's NXDN voice scrambler was a working model: a guessed shift register, a per-frame restart and a SACCH CRC that verified only all-zero blocks. Two captures made with keys the operator had programmed into their own radios settled all three, and the test of each hypothesis was where the known key ranked among 32,768."
category: solution-postmortem
keywords: nxdn scrambler decode, nxdn 15-bit scrambler key, nxdn sacch vcall cipher type, nxdn sacch crc6, lfsr keystream schedule, ambe pitch continuity test, known key verification sdr, kenwood nxdn scrambler, nxdn encrypted call recording, gophertrunk nxdn scrambler
tags: [rebuilt-from-the-air, nxdn, scrambler, encryption, self-consistent-trap, testing]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 9
---

*Part 9 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each one diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 8]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }})
locked an NXDN control channel and kept voice decoding through a drifting
dongle. This part closes the NXDN trilogy with the radios' built-in voice
scrambler — decoded with keys the operator had programmed into their own
radios — and with a method for testing a cipher model when the key is the
one thing you already know.*

> **TL;DR:** The NXDN scrambler (SACCH VCALL cipher type 1, a 15-bit key)
> lived in `internal/radio/nxdn/scramble.go` as an admitted working model.
> Two captures the reporter made with keys they had programmed — 32767 and
> 12345, Kenwood radios, 447.100 MHz NXDN96
> ([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)) —
> settled three things against it: the register (output = state bit 0,
> feedback bit 0 ⊕ bit 1 into bit 14), the schedule (reseed at every SACCH
> superframe start, then 4 × 49 bits per LICH-valid frame, FACCH frames
> included) and the SACCH CRC-6 (preset `0x3F`, taps `0x27`; the old one
> verified only all-zero blocks). The instrument: dump the FEC-decoded frames
> once, sweep hypotheses, and score each by AMBE pitch continuity *and* by
> where the known key ranks among all 32,768. It ranks first on both
> captures (0.66 / 0.65 against a 0.16 floor). Capture-verified; the live run
> is still the on-air gate.

**Key takeaways**

- **With a known key, the model is the only unknown — so rank the key.**
  Under the right register and schedule the programmed key came first of
  32,768 on both captures; under a near-miss schedule, around 500th.
- **A partial decrypt is a wrong schedule, not a near miss.** A continuity
  of 0.36 sits between ciphertext and speech because part of every
  superframe is descrambled correctly.
- **A checksum that verifies only zeros hides the layer above it.** With the
  reference CRC-6, 48 of 65 real SACCH blocks verify and the VCALL — cipher,
  key ID, superframe phase — becomes readable.
- **Six bits of CRC is not evidence that a call changed cipher.** One corrupt
  block flipped a clear call to "scrambler, key 0" and deleted its
  recording; a change now needs two agreeing superframes.

## Cheat sheet

| | |
|---|---|
| Issue | [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) — NXDN calls using the radios' built-in scrambler |
| Symptom | calls on the operator's own scrambled channel decode FEC-clean and render as noise; the scrambler primitive in the tree, given the right key, decoded nothing |
| Wrong theories | a left-shift x¹⁵ + x¹⁴ + 1 register; a keystream restarted every frame; a reference decoder's literal per-frame order; "0.36 means nearly right" |
| Real cause | fabricated register and schedule, and a SACCH CRC-6 with the wrong preset and taps, so neither the VCALL nor the superframe phase could be read |
| Fix | `Scrambler` (bit 0 out, bit 0 ⊕ bit 1 into bit 14); the schedule in `VoiceCipherTracker`; `SACCHCRC6` preset `0x3F` / taps `0x27`; two-superframe VCALL confirmation |
| Diagnostic | a frame dump (`testdata/scrambler_key*_frames.json`) swept by hypothesis, scored by pitch continuity and known-key rank; `TestNXDNScramblerReplay` |
| Status | capture-verified on two known-key captures; a live daemon run with the key on the scan-list channel is still on-air-gated |

## In this post

- **A cipher model nobody had checked** — the working model and the captures.
- **Three unknowns, one symptom** — register, schedule, phase.
- **The instrument** — one dump, many hypotheses, one rank.
- **What the captures pinned** — the register, the schedule and the CRC-6.
- **Production, and a six-bit CRC** — the tracker, the key config, the false flip.
- **What this overturns, and what we keep** — the rules.

## A cipher model nobody had checked

NXDN radios from Kenwood and Icom ship with a built-in privacy mode the
programming software calls a scrambler: a 15-bit key, 1 to 32767, set per
channel or per key slot. On the air the call's SACCH carries a VCALL message
that names a two-bit cipher type — 0 clear, 1 scrambler, 2 DES, 3 AES — and
a six-bit key ID, and the 49-bit AMBE+2 payload of every voice slot is XORed
with a [keystream]({{ '/reference/keystream/' | relative_url }}) from a
15-bit
[shift register]({{ '/reference/linear-feedback-shift-register/' | relative_url }})
seeded by the key.

GopherTrunk had that primitive in `scramble.go`, labelled honestly as a
working model: a left-shift register for x¹⁵ + x¹⁴ + 1, restarted for each
frame. Its tests were the ones a stream cipher always passes. Scramble then
descramble returns the input, because XOR is its own inverse. Key 0 changes
nothing. The sequence is balanced, with period 32767. All of that is true of
*any* maximal 15-bit register under *any* schedule. The
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
is at its purest in a symmetric cipher: no wrong model fails a round trip.

The reporter on #1261 runs their own NXDN radios. Once
[Part 7]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }})
made their clear calls decode they turned the scrambler on, recorded two
calls on 447.100 MHz — one with the radios programmed to key 32767, one to
key 12345 — and supplied the keys with the captures. That is the premise of
this post. GopherTrunk descrambles only with a key the operator has
configured, for systems they are authorised to monitor; the daemon never
searches for one. Here the key was known from the start. What was unknown
was whether the decoder's *model of the cipher* was right.

It was not. Both captures decoded cleanly at every layer Part 7 had rebuilt
— frame sync, LICH parity, Golay-clean AMBE codewords — and the voice came
out as noise. Applying the tree's scrambler with the correct key changed
nothing.

## Three unknowns, one symptom

Between FEC-clean ciphertext and speech sit three independent things, and a
wrong value in any one of them produces exactly the same noise.

**The register.** Which taps, which shift direction, which bit is the
output, how the key loads.

**The schedule.** When the register is reseeded; how many keystream bits a
frame consumes; which bits each of a frame's four voice slots takes; and
whether a frame with no voice in it — a FACCH frame — consumes keystream
anyway.

**The phase.** If the reseed is tied to the SACCH superframe, the decoder
must know where a superframe starts. That is a field in the SACCH, behind
the SACCH's CRC.

Each had a plausible wrong answer on the table. The register was the
working model's. The schedule was "restart every frame". A reference
decoder's source, read literally in the order its per-frame code runs,
suggested another. And the phase could not be read at all: the package's
`SACCHCRC6` verified none of the captures' SACCH blocks, so no VCALL
assembled and nothing said "cipher 1, key ID 3" in the first place.

Three unknowns multiplied together are a search space, and listening to WAV
files is no way to search one.

## The instrument: one dump, many hypotheses, one rank

The method that settled it has three parts, and each removes a way to fool
yourself.

**Dump the FEC-decoded frames once.** The production receiver and
`TrafficChannel` ran over each capture a single time and wrote out what
`TrafficChannel.OnFrame` delivers: every LICH-valid frame's index, LICH and
60 SACCH channel bits, and the 49-bit payload of each voice slot. The dumps
are committed as `testdata/scrambler_key32767_frames.json` (65 frames, 128
voice frames) and `scrambler_key12345_frames.json` (66 and 132). From then
on the demodulator is out of the loop: a hypothesis is a pure function of
bits, and trying one takes microseconds.

**Score by pitch continuity, never by level.** AMBE+2's `b0` is its 7-bit
pitch index — payload bits 0–3 and 37–39. Speech changes pitch slowly;
random bits do not. The score is the fraction of consecutive voice frames
whose `b0` differ by at most 10: speech sits around 0.5 and above,
ciphertext near 0.16. As received, the two captures scored 0.19 and 0.06.
Loudness is useless here, since random AMBE parameters through a vocoder
are loud — the lesson of
[DMR End to End Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }}).

**Then rank the known key.** One continuity number invites argument: is 0.36
a partial success or a coincidence? So for each candidate model — register ×
schedule × phase — the dump is descrambled with *every* key from 1 to 32767
and sorted by continuity, and the question becomes one integer: where does
the key the radios were programmed with land? A correct model must put it
first. Among 32,768 candidates a few wrong keys score well above the floor
by chance, and a nearly-right model lifts the true key off the floor without
making it speech. Rank separates both from the real thing.

The reference's literal per-frame order ranked the known key around 500th.
Reseeding at the superframe start, at the right phase, ranked it **first**,
on both captures, at 0.66 and 0.65.

<figure class="lab-figure">
<svg viewBox="0 0 680 236" width="680" height="236" role="img" aria-label="Two SACCH superframes of four frames each. The structure field counts 3, 2, 1, 0. In the captures the structure 3 and structure 1 frames carry four voice slots and the structure 2 and structure 0 frames are FACCH frames with no voice. Every frame consumes 196 keystream bits, numbered 0 to 195, 196 to 391, 392 to 587 and 588 to 783, and the register is reseeded from the key at each structure 3 frame. Below, one frame's 196 bits split into four 49-bit slots.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">keystream schedule: reseed at the superframe start, 196 bits per LICH-valid frame</text>
  <text x="66" y="38" text-anchor="end" fill="var(--fg-muted)" font-size="8">structure</text>
  <text x="66" y="72" text-anchor="end" fill="var(--fg-muted)" font-size="8">frame</text>
  <text x="66" y="106" text-anchor="end" fill="var(--fg-muted)" font-size="8">keystream</text>
  <text x="107" y="38" text-anchor="middle" fill="var(--accent)" font-size="10">3</text>
  <text x="181" y="38" text-anchor="middle" fill="currentColor" font-size="10">2</text>
  <text x="255" y="38" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="329" y="38" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <text x="403" y="38" text-anchor="middle" fill="var(--accent)" font-size="10">3</text>
  <text x="477" y="38" text-anchor="middle" fill="currentColor" font-size="10">2</text>
  <text x="551" y="38" text-anchor="middle" fill="currentColor" font-size="10">1</text>
  <text x="625" y="38" text-anchor="middle" fill="currentColor" font-size="10">0</text>
  <rect x="70" y="50" width="74" height="36" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="107" y="72" text-anchor="middle" fill="currentColor" font-size="8">4 voice slots</text>
  <rect x="144" y="50" width="74" height="36" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="181" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FACCH</text>
  <rect x="218" y="50" width="74" height="36" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="255" y="72" text-anchor="middle" fill="currentColor" font-size="8">4 voice slots</text>
  <rect x="292" y="50" width="74" height="36" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="329" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FACCH</text>
  <rect x="366" y="50" width="74" height="36" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="403" y="72" text-anchor="middle" fill="currentColor" font-size="8">4 voice slots</text>
  <rect x="440" y="50" width="74" height="36" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="477" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FACCH</text>
  <rect x="514" y="50" width="74" height="36" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="551" y="72" text-anchor="middle" fill="currentColor" font-size="8">4 voice slots</text>
  <rect x="588" y="50" width="74" height="36" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="625" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FACCH</text>
  <text x="107" y="106" text-anchor="middle" fill="currentColor" font-size="8">0–195</text>
  <text x="181" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">196–391</text>
  <text x="255" y="106" text-anchor="middle" fill="currentColor" font-size="8">392–587</text>
  <text x="329" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">588–783</text>
  <text x="403" y="106" text-anchor="middle" fill="currentColor" font-size="8">0–195</text>
  <text x="477" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">196–391</text>
  <text x="551" y="106" text-anchor="middle" fill="currentColor" font-size="8">392–587</text>
  <text x="625" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="8">588–783</text>
  <line x1="70" y1="24" x2="70" y2="116" stroke="var(--accent)" stroke-width="1.5"/>
  <line x1="366" y1="24" x2="366" y2="116" stroke="var(--accent)" stroke-width="1.5"/>
  <text x="74" y="128" fill="var(--accent)" font-size="8">reseed from the key</text>
  <text x="370" y="128" fill="var(--accent)" font-size="8">reseed from the key</text>
  <text x="66" y="172" text-anchor="end" fill="var(--fg-muted)" font-size="8">one frame</text>
  <rect x="70" y="150" width="148" height="34" fill="none" stroke="currentColor"/>
  <text x="144" y="171" text-anchor="middle" fill="currentColor" font-size="8">slot 0 · bits 0–48</text>
  <rect x="218" y="150" width="148" height="34" fill="none" stroke="currentColor"/>
  <text x="292" y="171" text-anchor="middle" fill="currentColor" font-size="8">slot 1 · bits 49–97</text>
  <rect x="366" y="150" width="148" height="34" fill="none" stroke="currentColor"/>
  <text x="440" y="171" text-anchor="middle" fill="currentColor" font-size="8">slot 2 · bits 98–146</text>
  <rect x="514" y="150" width="148" height="34" fill="none" stroke="currentColor"/>
  <text x="588" y="171" text-anchor="middle" fill="currentColor" font-size="8">slot 3 · bits 147–195</text>
  <text x="366" y="204" text-anchor="middle" fill="currentColor" font-size="9">voice slot v of a frame is XORed with that frame's bits [49v, 49v + 49)</text>
  <text x="366" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="8">a FACCH frame carries no voice and still consumes its 196 bits</text>
</svg>
<figcaption>The keystream is tied to the SACCH superframe, not to voice: in these NXDN96 captures every other frame is a FACCH frame, and each one still uses up 196 bits.</figcaption>
</figure>

The figure is also why 0.36 was never "almost". In these captures the frames
alternate: a superframe's first frame (structure field 3) carries four
voice slots, the second is a FACCH frame, the third carries voice, the
fourth is FACCH again. A schedule that restarts the register on every frame
descrambles the first voice frame of each superframe correctly — its bits
really are 0–195 — and the second with bits 0–195 where the air used
392–587. Half right scores between the floor and speech, and is still the
wrong schedule.

## What the captures pinned

**The register.** Output is state bit 0; the feedback is bit 0 ⊕ bit 1,
shifted in at bit 14. It is the construction DSD-FME's NXDN decoder uses —
GPL, read for the protocol fact, nothing ported.

```go
// internal/radio/nxdn/scramble.go (shape)
func (s *Scrambler) Next() byte {
    out := byte(s.state & 1)
    fb := ((s.state >> 1) ^ s.state) & 1
    s.state = ((s.state >> 1) | (fb << 14)) & ScramblerKeyMax
    return out
}
```

`TestScramblerKeystreamLiteral` now holds it to literal bits rather than to
properties: key 1 must open `1000000000000001…`, where the old left-shift
model produced `0000000000000001…`.

**The schedule.** `VoiceCipherTracker.apply` is the figure in four lines: a
frame's position in its superframe is `(index − sfStart) mod 4`; the register
is reset to the key, skipped forward `position × ScramblerBitsPerFrame`
(4 × 49 = 196) bits, and the next 196 bits are that frame's; slot `v` takes
`[49v, 49v+49)`. Every LICH-valid frame counts, voice or not.

**The CRC-6.** The reference register — MMDVMHost and DSD-FME agree on it —
is preset to `0x3F` with taps `0x27`. The one in the tree was preset to zero
with a different polynomial, and verified exactly one kind of block: all
zeros. With the reference CRC, 48 of the 65 SACCH blocks in the first
capture verify; the other 17 are RF errors. `TestSACCHCRC6RealAir` requires
that yield *and* requires that an all-zero block no longer passes.

That last fix is what made the others findable. A CRC-valid SACCH block
carries a two-bit structure field that counts a superframe down — 3, 2, 1,
0 — and four blocks' 18-bit messages concatenate into the 72-bit VCALL. Both
captures now read the same: source unit 8814, cipher type 1, key ID 3 on the
first and 1 on the second. The structure field is the reseed phase.

`TestVoiceCipherTrackerDecodesRealAirScrambledCalls` runs the whole path on
both dumps, installing the key when the VCALL names it, as the voice chain
does. With the programmed key the continuity is 0.66 and 0.65; with the
neighbouring key, 0.24 and 0.17. Every assertion fails on the previous code.

## Production, and a six-bit CRC

In the daemon the NXDN voice chain feeds every LICH-valid frame to an
`nxdn.VoiceCipherTracker`. It takes its superframe phase from any CRC-valid
block and flywheels by frame count across the ones that fail. When the
operator has keys configured it holds voice until the first VCALL names the
cipher — capped at eight frames — then releases it clear, descrambled, or as
ciphertext if no key matches. A scrambled call is published as
`call.encryption` with its cipher type and key ID, so call logs and
[encrypted-call policy]({{ '/reference/encrypted-call-handling/' | relative_url }})
treat it like any other. DES and AES, cipher types 2 and 3, are identified
and not decrypted.

The key is configured the way the radio shows it — decimal — under the key
slot the VCALL names, on a trunking system or directly on a scan-list
channel:

```yaml
scanner:
  conventional:
    - label: "Plant ops"
      frequency_hz: 447100000
      mode: nxdn
      squelch_dbfs: -50
      encryption_keys:
        - key_id: 3
          algorithm: scrambler
          key: "32767"
```

A scanner call's grant names the scanner, not a trunking system, so a
channel's keys are registered under a pseudo-system for its frequency
(`ConventionalKeySystem`). Entries can also carry a `talkgroup`: on the
reporter's trunked site key ID 0 is a real slot that two talkgroups use with
different keys. The recipe is
[Cookbook Volume 2, Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }});
the [encryption page]({{ '/dmr-encryption.html' | relative_url }}) is the
reference.

Then a live log came back with a new complaint: after a call that failed to
descramble, a *clear* call failed too. It had not failed. Its 104 voice
frames were all counted clear — and then, late in the call, one VCALL
superframe decoded as "scrambler, key 0". A SACCH block's CRC is six bits,
so one corrupt block in 64 passes, and this call failed 12 of its 52. That
single misread flipped the call to encrypted, and with
`recordings.skip_encrypted` set the recorder deleted a good clear recording.

A VCALL does not change inside one transmission, so `takeVCALL` now trusts
the first VCALL of a call at once — a scrambled call must start descrambling
as soon as it can — and applies a *different* later one only when the next
superframe repeats it. `VCALLUnconfirmed` counts the ones that never were,
and `TestVoiceCipherTrackerIgnoresOneOffVCALLChange` replays the sequence.
Rejecting key ID 0 outright was not an option; it is a real slot.

What is verified: both known-key captures descramble to speech through the
production tracker, offline. What is not: the reporter's live run with the
key on the scan-list channel. Until that recording exists this is
**capture-verified** and on-air-gated. The offline check is one command:

```bash
GT_NXDN_SCR_IQ=447.100MHz.cs16 GT_NXDN_SCR_RATE=2400000 GT_NXDN_SCR_KEY=32767 \
  go test ./cmd/gophertrunk -run TestNXDNScramblerReplay -v
```

It prints the VCALL, the SACCH CRC yield and the continuity as received and
descrambled; `GT_NXDN_SCR_BRUTE=1` adds the rank of the key you gave it.

## What this overturns

No earlier post on this blog described the scrambler; the model it replaces
lived in `scramble.go` under a "working model" label — the right label, on
a wrong model.
[Protocol Decoders Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }})
did describe the NXDN SACCH, as a 32-dibit field, without its CRC or its
superframe. On air the SACCH is 30 dibits, its 32 information bits end in
the CRC-6 above, and four blocks make the message that says who is talking
and under which cipher.

## What we keep

- **Dump once, then test hypotheses on bits.** Take the receiver out of the
  loop before searching a model space — an instrument in the sense of
  [From Spec to Shipping Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }}).
- **When you hold the key, rank it.** A correct cipher model puts the known
  key first among all candidates. 500th, or 0.36, is a wrong model, however
  encouraging it sounds.
- **Property tests cannot pin a symmetric cipher.** Round trips, balance and
  period hold for every wrong register; only literal keystream bits and a
  known-key capture can disagree with you.
- **Fix the checksum first.** A CRC that verifies nothing real hides every
  field behind it, including the ones that explain the rest.
- **Weigh a field by the strength of its check.** Six bits of CRC is enough
  to start a call's policy, not to reverse it.

## Where this goes next

That is NXDN: frame, control channel, scrambler — three layers, each
invented, each green, each settled by one reporter's captures. The next mode
had no capture at all, only the firmware its hotspots run.
[Part 10]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }})
rebuilds D-STAR, where the sync words, the header FEC, the CRC, the voice
bit schedule and the Golay code were all wrong at once — a whole framing,
as in the
[SmartNet rebuild]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }}).

## FAQ

**Can GopherTrunk decode NXDN calls that use the built-in scrambler?**
Yes, when you configure the key your radios are programmed with: an
`encryption_keys` entry with `algorithm: scrambler`, the decimal key and the
key ID the call's VCALL names. The voice chain then descrambles in-process
and records clear speech. Without a matching key the call is recorded as
ciphertext and flagged encrypted.

**Does GopherTrunk search for an NXDN scrambler key on its own?**
No. The daemon descrambles only with keys the operator has configured for
systems they are authorised to monitor. The all-keys sweep described here
was a test instrument run offline against captures whose key was already
known, to check the decoder's model of the cipher.

**Why was a clear NXDN call flagged as encrypted and its recording skipped?**
A SACCH block is protected by a 6-bit CRC, so on a weak call a corrupt block
occasionally passes and can misread the VCALL as scrambled. Before the fix
one such block flipped the call and `recordings.skip_encrypted` deleted the
file. A VCALL change now takes effect only when the next superframe repeats
it.

**Is the NXDN scrambler decode verified on air?**
It is capture-verified: two captures made with known keys descramble to
speech through the production tracker, and the known key ranks first of
32,768 on both. A live daemon run with the key configured on the scan-list
channel has not been confirmed yet, so the on-air gate is still open.

## Series navigation

**Part 9 of 14** · ←
[Part 8: NXDN II — The Control Channel on Real Air, 214 of 214]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }})
· Next →
[Part 10: D-STAR — Wrong Sync, Wrong FEC, Wrong Golay]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }})
