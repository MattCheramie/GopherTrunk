---
title: "Rebuilt From the Air, Part 1: The Audit — The Decoders That Never Heard a Radio"
description: "Nine GopherTrunk decoders passed every test they had and matched no real radio, because each test encoder was written from the same invented layout as its decoder. This opener maps the roll call — what was wrong, how each one surfaced, what is verified today — and lists the earlier posts this series corrects."
category: solution-postmortem
keywords: self-consistent test trap, fabricated framing, sdr decoder postmortem, round-trip test failure, reference implementation testing, independent encoder, literal test vectors, real-air test fixture, on-air verification, gophertrunk decoder audit
tags: [rebuilt-from-the-air, testing, postmortem, decoders, verification, reference-implementations]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 1
---

*Part 1 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each one diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists. This opener is the audit itself: the pattern, the roll call, how each
case surfaced, what is verified today, and the earlier posts on this blog
that the series corrects.*

> **TL;DR:** Between September and November 2026, nine GopherTrunk decoders
> turned out to implement framing that matched no transmitter: SmartNet,
> MDC1200, the CTCSS and DCS tone gates, FLEX, the LoRa bit layer, System
> Fusion's FICH, NXDN, D-STAR (with dPMR's voice Golay) and P25 Phase 2
> TDMA. In each, the test encoder and the decoder came from one model — an
> invented sync word, bit order, interleaver or line code — so every round
> trip was green and the first real signal decoded nothing. Each was rebuilt
> from implementations proven on air, pinned with literal vectors, and
> pinned again on a real-air slice where one exists. Today MDC1200, the tone
> gates and NXDN trunked voice are **on-air verified**; FLEX at 1600 bps,
> NXDN voice and control and an idle P25 Phase 2 carrier are
> **capture-verified**; LoRa, System Fusion, D-STAR and SmartNet still wait
> for a recording.

**Key takeaways**

- **An encoder you wrote cannot test a decoder you wrote.** Both halves
  came from one reading of the protocol, so the reading itself was never
  under test — only the code's agreement with itself.
- **Fabrication is patchy, and that is its camouflage.** LoRa's chirps were
  right and its six bit layers invented; MDC1200's sync word, interleave and
  CRC were right and its line code wrong.
- **None of the nine was caught by CI.** Each surfaced through a reporter's
  capture or lab bench, or through a diff against a second implementation
  made for some other reason.
- **A caveat in a comment is not a gate.** Several of these decoders said
  "pending real-capture calibration" in their own source for months; the
  status that reaches an operator has to say it too.

## Cheat sheet

| | |
|---|---|
| Issues | [#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143), [#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220), [#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184), [#586](https://github.com/MattCheramie/GopherTrunk/issues/586), [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261), [#1273](https://github.com/MattCheramie/GopherTrunk/issues/1273), [#1364](https://github.com/MattCheramie/GopherTrunk/issues/1364), [#1374](https://github.com/MattCheramie/GopherTrunk/issues/1374) and a FLEX feature request |
| Symptom | green CI; a real transmission gives no sync lock, no CRC-valid frame, no call |
| Wrong theories | a green round trip proves the layout; a decoder right in its lower layers is right all the way up; an honest "unverified" comment is enough |
| Real cause | fabricated framing — the test encoder and the decoder shared one invented layout |
| Fix | rebuild from implementations proven on air; literal vectors in the test file; a real-air slice in `testdata/` |
| Diagnostic | an encoder nobody here wrote, or one recording of a real transmitter |
| Status | three on-air verified, three capture-verified, four still gated on a capture (table below) |

## In this post

- **The pattern** — one model, two halves, no radio.
- **The roll call** — nine decoders, what was invented, where each stands.
- **How each one surfaced** — a capture or a second reference, never the test suite.
- **Why it kept happening** — four ordinary habits that add up to fabrication.
- **What this series overturns** — the earlier posts that described these formats.
- **Four words for how sure we are** — the verification vocabulary every part uses.

## The pattern: one model, two halves, no radio

A protocol decoder is rarely written alone. To test it you need frames, so
you write a frame builder, decode what it built, and assert that the payload
survives. It is the natural first test and it is worth having.

[From the Issue Tracker, Part 20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
named what that test cannot see — the
[self-consistent trap]({{ '/reference/self-consistent-test-trap/' | relative_url }}):
anything the builder and the decoder share is invisible to a round trip. That
post collected seven bugs of the kind, each a wrong constant or table inside
a decoder that was otherwise real. This series is about the larger version,
which the project's engineering notes call **fabricated framing**: whole
layers — a line code, a frame layout, an FEC chain, a bit order — that
existed only in this repository. They were not typos against a standard.
They were a working model, written where a standard was vague or out of
reach, with a test encoder written from the same model.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two test loops side by side. On the left, one invented layout feeds both the test encoder and the decoder, their round trip is green, and a real radio's first capture decodes nothing. On the right, two independent references produce literal vectors that pin both the decoder and our own encoder, a real-air slice feeds the decoder directly, and an operator's live run is the final gate.">
  <text x="170" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">what shipped: one model, two halves</text>
  <rect x="90" y="28" width="160" height="30" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="170" y="47" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one layout (guess or misreading)</text>
  <line x1="150" y1="58" x2="90" y2="84" stroke="currentColor"/>
  <line x1="190" y1="58" x2="260" y2="84" stroke="currentColor"/>
  <rect x="30" y="84" width="120" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="90" y="103" text-anchor="middle" fill="currentColor" font-size="9">test encoder</text>
  <rect x="200" y="84" width="120" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="260" y="103" text-anchor="middle" fill="currentColor" font-size="9">decoder</text>
  <line x1="90" y1="114" x2="140" y2="142" stroke="currentColor"/>
  <line x1="250" y1="114" x2="200" y2="142" stroke="currentColor"/>
  <rect x="100" y="142" width="140" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="170" y="161" text-anchor="middle" fill="var(--accent)" font-size="9">round trip: green</text>
  <rect x="30" y="204" width="110" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="85" y="223" text-anchor="middle" fill="var(--fg-muted)" font-size="9">a real radio</text>
  <line x1="140" y1="219" x2="290" y2="219" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <line x1="290" y1="219" x2="290" y2="114" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="215" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="8">first capture: decodes nothing</text>
  <line x1="340" y1="24" x2="340" y2="240" stroke="var(--fg-muted)" stroke-dasharray="2 4"/>
  <text x="510" y="16" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">the rebuilt loop: facts from outside</text>
  <rect x="356" y="28" width="96" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="404" y="47" text-anchor="middle" fill="currentColor" font-size="9">reference A</text>
  <rect x="460" y="28" width="96" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="508" y="47" text-anchor="middle" fill="currentColor" font-size="9">reference B</text>
  <rect x="566" y="28" width="104" height="30" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="618" y="47" text-anchor="middle" fill="var(--accent)" font-size="9">real-air slice</text>
  <line x1="404" y1="58" x2="436" y2="84" stroke="currentColor"/>
  <line x1="508" y1="58" x2="476" y2="84" stroke="currentColor"/>
  <rect x="386" y="84" width="140" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="456" y="103" text-anchor="middle" fill="currentColor" font-size="9">literal vectors</text>
  <line x1="430" y1="114" x2="404" y2="142" stroke="currentColor"/>
  <line x1="482" y1="114" x2="520" y2="142" stroke="currentColor"/>
  <line x1="618" y1="58" x2="570" y2="142" stroke="var(--accent)"/>
  <rect x="356" y="142" width="96" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="404" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="9">our encoder</text>
  <rect x="470" y="142" width="120" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="530" y="161" text-anchor="middle" fill="currentColor" font-size="9">decoder</text>
  <line x1="530" y1="172" x2="530" y2="204" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <rect x="430" y="204" width="200" height="30" rx="5" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="530" y="223" text-anchor="middle" fill="var(--accent)" font-size="9">on-air gate: an operator's live run</text>
  <text x="404" y="188" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pinned too</text>
</svg>
<figcaption>The left loop can only ever agree with itself; every arrow into the right-hand decoder starts somewhere the code under test did not write.</figcaption>
</figure>

Three things were true of every decoder here before its rebuild. The
transmitter side lived next to the receiver side and had the same author. No
value in the tests had been typed in from anywhere else. And no recording of
a real transmitter had ever been through it.

The gaps were often written down. The old FLEX package said its parse was
"validated end-to-end against a synthetic encoder, with exact on-wire bit
order / generator left for real-capture calibration." Posts on this blog
said "synthetic-verified." Neither stopped describing the invented layout as
the protocol's layout — and a described format with a green suite behind it
reads as finished.

## The roll call

One row per decoder. "Invented" means the code and its tests agreed on
something no transmitter sends; the status column uses the vocabulary
defined at the end of this post.

| Decoder | What was invented | How it surfaced | Status now | Told in |
|---|---|---|---|---|
| SmartNet ([#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143)) | 24-bit sync 0xA4D7AA, 32-bit OSW, BCH(64,16,11) — and the modem under them | a reporter whose every hunt ended `cchunt: hunt failed` | reference-pinned; the reporter's capture is still the on-air gate | [From Spec to Shipping 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }}) |
| MDC1200 ([#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220)) | the line code: tone decisions sliced as data bits | a lab run — FleetSync decoded through the identical chain, MDC1200 never locked | on-air verified 28 Sep; real-air slice committed | [Part 2]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }}) |
| CTCSS and DCS ([#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184)) | a threshold calibrated at the wrong sample rate, a rounded Goertzel bin, and the DCS codeword's layout and bit order | the reporter's radios — a gate that never opened, then opened on the wrong tone | on-air verified 24 Sep | [Part 3]({{ '/blog/solution-postmortem/rebuilt-from-the-air-03-ctcss-dcs-tone-gates/' | relative_url }}) |
| FLEX | the mode code hunted after the sync marker, an unprotected frame information word, the page-type table, reversed BCH information bits | a diff against multimon-ng, begun for a 3200 / 6400 bps request | 1600 bps capture-verified on five off-air frames; 3200 / 6400 bps synthetic-verified | [Part 4]({{ '/blog/solution-postmortem/rebuilt-from-the-air-04-flex-four-modes/' | relative_url }}) |
| LoRa ([#586](https://github.com/MattCheramie/GopherTrunk/issues/586)) | whitening, CRC, header, Hamming layout, interleaver, chip mapping | two independent references — the old code decoded 0 of 6 reference frames | reference-pinned on 2,916 cases; no real LoRa IQ decoded yet | [Part 5]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }}) |
| System Fusion ([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)) | a FICH of 100 bits — on air it is 100 dibits | MMDVMHost's encoder; a control channel that locked and never granted | reference-pinned; unverified on air | [Part 6]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }}) |
| NXDN ([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)) | 16-bit frame sync `0xC55A`, 32-dibit SACCH, no scrambler; the control channel's coding; the voice scrambler's register and schedule | the reporter's Kenwood captures — the old code decoded 0 frames | voice and control channel capture-verified; trunked voice on-air verified 21 Oct; scrambler capture-verified, live run open | Parts [7]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }}), [8]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }}), [9]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }}) |
| D-STAR, dPMR voice ([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261), [#1374](https://github.com/MattCheramie/GopherTrunk/issues/1374)) | sync words, header FEC, CRC, the voice bit schedule — and a (23,12) Golay that is not the vocoder's | a diff against the MMDVM firmware's format | reference-pinned; no real D-STAR capture yet | [Part 10]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }}) |
| P25 Phase 2 ([#1273](https://github.com/MattCheramie/GopherTrunk/issues/1273), [#1364](https://github.com/MattCheramie/GopherTrunk/issues/1364)) | one sync per superframe, a Golay ISCH with an invented SlotType, contiguous unscrambled voice; mirrored PN44 taps | a hybrid site that never followed a call; OP25 and SDRTrunk agreeing against it | reference-pinned; an idle real carrier decodes CRC-valid; no live call yet | Parts [11]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }}), [12]({{ '/blog/solution-postmortem/rebuilt-from-the-air-12-pn44-gardner-real-carrier/' | relative_url }}) |

Two things stand out. The first is how long some of these ran: the LoRa
bit layer shipped in June and was replaced in October, and the MDC1200
replay harness says of its own decoder that it "shipped for four months
without ever having seen a real radio."

The second is that the fabrication was never total. LoRa's chirp, preamble
sync and SFD detection matched real LoRa all along, while everything above
the symbol was invented. MDC1200's 40-bit sync word, 16×7 interleave and CRC
were correct; only the step from tone to bit was wrong. FLEX had the right
sync marker and the right block-interleave index inside the wrong frame. A
half-right decoder produces plausible intermediate numbers, which is what
makes the invented half hard to suspect.

## How each one surfaced

There were only two ways.

**A reporter's signal.** SmartNet, MDC1200, the tone gates, NXDN and P25
Phase 2 were found because somebody pointed a radio at GopherTrunk and said
what happened. The most useful reports carried a control experiment. The
MDC1200 reporter's lab decoded Kenwood FleetSync bursts through a front end
identical stage for stage to the MDC1200 one while MDC1200 "never locked",
which clears the DSP chain in one line. The NXDN reporter's two captures
framed nothing under the old layout and thirty-six and fifty-six voice
frames under the new one. A Phase 2 capture had failed its ISCH on 46–62 %
of sub-frames under the old model — a number that reads like a weak signal
until you learn the field being checked does not exist.

**A second implementation.** FLEX, LoRa, the System Fusion FICH and D-STAR
were found by reading a decoder known to work and comparing layouts. FLEX is
the cleanest case: a request for the 3200 and 6400 bps modes meant reading
multimon-ng's frame handling, and the first thing that showed was the
existing 1600 bps path hunting the mode code on the wrong side of the sync
marker. Nobody had reported FLEX broken.

That second route is the uncomfortable one: a fabricated decoder with no
local signal and no reporter stays fabricated until somebody has an
unrelated reason to open the file. It is why
[From Spec to Shipping, Part 2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})
argues for choosing the reference before writing the first line. The
references here are open-source decoders with years on real systems: OP25
and SDRTrunk for P25, trunk-recorder for SmartNet, multimon-ng for FLEX,
MMDVMHost and the MMDVM firmware for System Fusion, NXDN and D-STAR, DSD-FME
for NXDN, gr-lora_sdr and LoRaPHY for LoRa. Several are GPL. They were read
for protocol facts — constants, field positions, bit orders — and nothing
from them is ported; the
[clean-room rules]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})
apply throughout.

## Why it kept happening

Four habits, none of them careless on its own.

**The encoder was written as the decoder's inverse.** The MDC1200 test
helper describes itself as "the transmitter-side inverse of `deinterleave`."
That is a statement about the code, not about a radio; an encoder defined
that way cannot disagree with its decoder even in principle.

**A working model stood in for a standard.** Some of these specifications
are paywalled and some protocols are documented only in other people's
source. Filling the gap with a plausible structure gets a pipeline running.
[Placeholder constants]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }})
are the small form of that habit; a placeholder *frame* is the large one,
and nothing in the build distinguishes it from a real frame.

**There was no capture to argue with.** A decoder with no local signal — a
6400 bps FLEX channel, a D-STAR repeater, an NXDN trunked site — gets its
only feedback from its own fixtures.

**The caveat lived where operators do not read.** "Left for real-capture
calibration" in a package comment is honest and inert.

A fifth, smaller habit recurs through the series: synthetic *transmitters*
that behave like no radio — bursts laid back to back where a handheld leaves
gaps, a deviation that scales with the sample rate. Those give a real format
that only decodes its fixture, and
[DMR End to End, Part 9]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
is the canonical example.

## What this series overturns

Seven earlier posts described the formats listed above as GopherTrunk then
implemented them. Their accounts of the code were accurate when written;
what the code implemented was not the protocol. Each gains an update note
as its correcting part publishes.

| Earlier post | What it described | Corrected in |
|---|---|---|
| [Protocol Decoders, Part 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }}) | a Golay-protected ISCH on every sub-frame naming it voice or MAC, and the MAC path behind it | Parts 11–12 |
| [P25 End to End, Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }}) | twelve sub-frames alternating between two timeslots; MAC PDUs through trellis, RS(24,16,9) and PN44 | Parts 11–12 |
| [Protocol Decoders, Part 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }}) | NXDN's 16-bit frame sync `0xC55A`, 32-dibit SACCH and unscrambled frame | Parts 7–10 |
| [Beyond Voice, Part 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }}) | MDC1200 bits arriving from the slicer as data; a decoder with no air capture | Part 2 |
| [Beyond Voice, Part 6]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }}) | FLEX as marker, then mode code `0x870C`, then frame information word; one mode only (the POCSAG half stands) | Part 4 |
| [Beyond Voice, Part 11]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }}) | the LoRa whitening, CRC, header and interleaver (the chirp receiver stands) | Part 5 |
| [Beyond Voice, Part 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }}) | D-STAR's sync `0xEAA060` and header FEC, the System Fusion FICH codec, dPMR's voice FEC | Parts 6 and 10 |

Part 14 is the full errata, including the items earlier series finales
listed as open that have since closed.

## Four words for how sure we are

Every part of this series ends with a status, and the status is one of four
phrases. They extend the ladder in
[From Spec to Shipping, Part 14]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }});
no decoder is ever described above the rung its evidence reaches.

- **Synthetic-verified** — it decodes signals built by GopherTrunk's own
  encoder. This licenses "the two halves agree" and nothing else. Every
  decoder in the roll call had it on the day it was found to be wrong.
- **Reference-pinned** — constants, layouts and outputs are held to literal
  vectors that came from an independent implementation proven on air: typed
  in from it, or generated by it, never by our encoder
  ([Part 3 of that series]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})
  is the method). This licenses "it matches the reference."
- **Capture-verified** — a recording of a real transmitter decodes through
  the production path offline, and a slice of it sits in `testdata/` as a
  regression. This licenses "it works on this capture."
- **On-air verified** — an operator ran the live daemon against their own
  radios or system, it decoded, and they said so. Only this licenses "the
  problem is gone," and only this
  [closes an issue]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }}).

A decoder waiting on the next rung is **on-air-gated**, and each part says
exactly what recording or live run would move it.

## What we keep

- **Ask where the encoder came from.** If the answer is "the same author,
  the same reading," the suite tests arithmetic, not the protocol.
- **One literal from outside beats a thousand round trips.** The tone
  string a reference encoder emits for one MDC1200 burst is enough to fail
  the old receiver.
- **Check layers independently.** A correct sync word says nothing about
  the line code under it or the FEC above it.
- **Diff before you extend.** FLEX was caught because a feature request
  began with reading the reference; make that the first step of any change
  to a decoder with no capture behind it.
- **Put the status where the operator is.** A decoder is synthetic-verified
  in its documentation, not only in a source comment.

## Where this goes next

The series takes the roll call in roughly the order the cases were opened,
starting with the one that was closest to right.
[Part 2]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }})
is MDC1200: a correct sync word, a correct interleave, a correct CRC, and a
receiver that for four months handed the framer the tones instead of the
bits.

## FAQ

**What is fabricated framing in a protocol decoder?**
It is a frame layout, line code, FEC chain or bit order that exists only in
the decoder and its own test encoder — a working model written where the
real format was unknown or misread. Because both halves share it, every
round-trip test passes while no real transmission can decode.

**How can a decoder pass every test and still never decode a real radio?**
A round-trip test encodes with your encoder and decodes with your decoder,
so it proves only that they agree. If both were written from one wrong
reading of the protocol, they agree perfectly. Only a fact from outside — a
vector from another implementation, or a recording of a real transmitter —
can disagree with it.

**Which of these GopherTrunk decoders are verified on air today?**
MDC1200 (28 Sep 2026), the CTCSS and DCS tone gates (24 Sep) and NXDN
trunked voice (21 Oct) are on-air verified. FLEX at 1600 bps, NXDN voice and
control and an idle P25 Phase 2 carrier are capture-verified. LoRa, System
Fusion, D-STAR, FLEX at 3200 and 6400 bps and SmartNet are reference-pinned
or synthetic-verified and still wait for a recording.

**Do the earlier blog posts about these protocols still apply?**
Their descriptions of GopherTrunk's wiring, bus events, storage and panels
still hold. Their descriptions of the on-air formats listed in the table
above do not; each of those posts points to the part of this series that
replaces it, and Part 14 collects the errata.

**How do I check whether a decoder I rely on is real?**
Look for three things in its tests: an encoder the authors did not write,
literal expected values copied from a second implementation, and a recording
of a real transmitter in the test data. A suite with none of the three is
synthetic-verified, however green it is.

## Series navigation

**Part 1 of 14** · Next →
[Part 2: MDC1200 — The Tones Were Sliced as the Data]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }})
