---
title: "Rebuilt From the Air, Part 14: The Audit Checklist — and the Errata for This Blog"
description: "The finale: seven questions that separate a real decoder from a fabricated one, a status ledger of sixteen decoders in the project's four verification words — which are on-air verified, which are capture-verified, which still wait for a recording — and the errata for the seven earlier posts this series corrects."
category: solution-postmortem
keywords: sdr decoder audit checklist, self-consistent test trap, independent encoder test, literal test vectors, real air regression slice, on-air verified decoder status, capture verified vs reference pinned, decoder errata, fabricated framing postmortem, gophertrunk decoder verification
tags: [rebuilt-from-the-air, checklist, verification, testing, errata, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 14
---

*Part 14 — the last — of **Rebuilt From the Air**, a 14-part postmortem
series on the GopherTrunk decoders that passed every test and never matched
a real radio — each diagnosed from a reference implementation or a
reporter's capture, rebuilt against independent decoders, and pinned to real
air where a capture exists.
[Part 13]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
was about reading a capture before theorising about it. This closing part
turns the series into three things you can use: a checklist for telling a
real decoder from a fabricated one, a ledger of what is verified and what
still waits for a recording, and the errata for the earlier posts on this
blog that described the formats these decoders used to implement.*

> **TL;DR:** A fabricated decoder passes every test it has, because its
> test encoder shares its invented layout. Seven questions find one: is
> there an encoder you did not write; do two references agree, and where
> don't they; are there literal vectors in the test file; does the fixture
> transmitter behave like a radio; is a real-air slice in CI; is the PDU
> shape the *live* FEC chain hands out pinned; do the fake servers enforce
> strictness. The ledger: MDC1200, FleetSync, CTCSS/DCS, AM and NXDN
> trunked voice are **on-air verified**; FLEX 1600 bps, NXDN voice, control
> channel and scrambler, and the P25 Phase 2 MAC path are
> **capture-verified**; LoRa, System Fusion, D-STAR and SmartNet are
> **reference-pinned** and wait for a recording. The errata corrects seven
> earlier posts and updates two finales' open lists.

**Key takeaways**

- **Independence is the only test of a format.** Every rebuild in this
  series was settled by something the project did not write — a reference
  encoder, a second decoder, a recording.
- **Verified is four different words.** Synthetic-verified,
  reference-pinned, capture-verified and on-air verified are not
  interchangeable, and the ledger never rounds one up.
- **The open gates are recordings, not work.** LoRa, System Fusion,
  D-STAR, multi-rate FLEX and SmartNet each wait on one capture.
- **A blog that documents a decoder inherits its bugs.** Seven earlier
  posts described formats that never matched the air; this part is the
  index of the corrections.

## Cheat sheet

| | |
|---|---|
| Issue | the series' roll: [#1143](https://github.com/MattCheramie/GopherTrunk/issues/1143), #1184, [#1220](https://github.com/MattCheramie/GopherTrunk/issues/1220), #586, [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261), [#1273](https://github.com/MattCheramie/GopherTrunk/issues/1273), #1364, #1374 |
| Symptom | suites green for months; the first real radio or capture decodes nothing |
| Wrong theories | weak signal, gain, ppm, encryption — and "the tests pass" |
| Real cause | the decoder and its test encoder were written from one layout, and the layout was invented or misread |
| Fix | rebuild against decoders proven on air; pin with literal vectors and an independent encoder; commit a real-air slice where one exists |
| Diagnostic | the seven questions below |
| Status | five rows on-air verified, five capture-verified, six still waiting on a recording — the ledger below |

## In this post

- **The four words** — the vocabulary the ledger is written in.
- **The audit checklist** — seven questions, each with the decoder that failed it.
- **The status ledger** — sixteen rows, no rounding up.
- **The errata** — seven posts corrected, and the open items that moved.
- **What we keep** — the operating principle.

## The four words

Everything below uses four terms, and none implies the next:

- **Synthetic-verified** — it decodes what the project's own encoder
  produced. That is consistency, which is all
  [the self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
  needs.
- **Reference-pinned** — it matches literal output of an independent
  implementation that works on air.
- **Capture-verified** — it decodes a recording of a real transmitter,
  offline, in a test.
- **On-air verified** — a reporter ran a build with the fix against live
  radios and confirmed it.

## The audit checklist

[SmartNet]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
was the first decoder rebuilt from proven references. After this series the
pattern is specific enough to audit for. Ask these of any decoder —
especially one that "works".

1. **Is there an encoder you did not write?** MDC1200 is pinned on the
   reference encoder's own 48 kHz audio, and GopherTrunk's synthesiser was
   read by the reference decoder — both directions
   ([Part 2]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }})).
   D-STAR's header is pinned by the MMDVM firmware's own `txHeader` output.
   The cost of having none is in the ledger: multi-rate FLEX has no
   independent encoder, so it stays synthetic-verified.
2. **Do two references agree — and do you know where they don't?** P25
   Phase 2 was rebuilt against OP25 and SDRTrunk, which agree on every
   constant. LoRa was transcribed separately from gr-lora_sdr and LoRaPHY:
   identical on 2916 random cases, different only in padding-codeword
   content. When references do disagree,
   [a capture referees]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }}):
   one reference's literal per-frame order ranked the known NXDN scrambler
   key about 500th of 32768; the reseed schedule the captures settled
   ranked it first.
3. **Are there literal vectors in the test file?** A constant typed from a
   reference fails when yours drifts; one taken from your own encoder never
   does. PN44 is 4320 bits of hex. DCS code 023 is `0x763813`, from
   published parity equations. SmartNet's sync bits, interleave permutation
   and XOR masks are literals
   ([Literal Vectors, Not Round-Trips]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})).
4. **Does the fixture transmitter behave like a radio?** Every synthetic
   DMR fixture laid bursts back to back like a repeater; a simplex handheld
   is off for 32.5 ms of every 60 ms frame, and
   [direct mode never decoded]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }}).
   Real transmitters have noise before the key-up, repeat their headers —
   one radio sends its Voice LC Header ten times over 0.6 s — and carry
   random payload. They also use the right pulse: System Fusion fixtures
   built with the P25 modulator gave about 45 % dibit agreement and looked
   like a receiver bug.
5. **Is a real-air slice in CI?** MDC1200, FleetSync, CTCSS, AM, NXDN
   voice and control channel, the P25 Phase 2 carrier and five FLEX frames
   each have one committed. So does the FM test tone that was filed as
   System Fusion — it pins a gate shut.
6. **Is the PDU shape the live FEC chain hands out pinned?** The
   talker-alias fixtures came from another decoder's 15-byte dumps. A live
   MAC PDU is 144 bits — a 16-byte vendor payload — and the fragment
   parsers sliced to the payload's *end*, so each fragment carried one
   trailing byte that shifted the cipher region: RID parsed, name garbage,
   alias dropped. `TestDispatcherPublishesRealMotorolaAliasAsReliable` now
   carries the real PDUs through the air framer on 144-bit sub-frames.
7. **Do the fake servers enforce strictness?** Not a decoder, the same
   trap. The SoapyRemote driver sent opcode 600 for "set antenna"; the
   upstream value is 501, and the fake server switched on the same wrong
   constant. The fake now fails any request whose body is not fully
   consumed, and `TestOpenSetAntennaUsesUpstreamOpcode` pins the numbers
   against upstream literals
   ([Wire Protocols Without Schemas]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})).

## The status ledger

One row per decoder, in the words above. "P" numbers are parts of this
series; FleetSync's story is
[Beyond Voice 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }}),
AM's is [Squelch 7]({{ '/blog/deep-dives/squelch-07-am/' | relative_url }})
and SmartNet's is From Spec to Shipping 8. Every cell comes from the
project's engineering notes; a dash means they record nothing.

| Decoder | Rebuilt against | Capture-verified? | On-air verified? | Open gate |
|---|---|---|---|---|
| MDC1200 · P2 | the reference modem's burst layouts, tone strings and encoder audio | yes — a committed 0.7 s slice; the reporter's clipped scanner recording replays | **yes, 28 Sep** — eleven PTT ID bursts, unit `0x1777`, 447.100 MHz | none; #1220 closed on the reporter's confirmation |
| FleetSync | not a rebuild — the framer was right, the WAV reader was not | yes — 5 of 6 (FS-I) and 8 of 8 (FS-II) bursts CRC-valid; two committed slices | **yes, 16 Sep** — FS-I and FS-II decode live | why two dongles stalled at the same instant in that run is not root-caused |
| CTCSS / DCS · P3 | the EIA tone table; DCS against published parity equations and SDRangel's alias tables | CTCSS yes — real-air slices replayed at 2.4 MS/s | **yes, 24 Sep** — Kenwood NX-300 / NX-5000 and a service monitor; DCS polarity pinned | none recorded; 150.0 / 151.4 Hz and 023 / 047I open together by design |
| AM | a new chain, nothing to rebuild | yes — two slices of a 126.400 MHz capture pin squelch and carrier tracking | **yes, 29 Sep** | none recorded |
| FLEX 1600 bps · P4 | multimon-ng's framing, read for facts | yes — five real-air frames decode to the pages that suite expects | no live confirmation recorded | a live paging run |
| FLEX 3200 / 6400 · P4 | the same reference | no — reference-derived and synthetic-verified only | no | a capture of a 929–932 MHz multi-rate channel |
| LoRa · P5 | gr-lora_sdr and LoRaPHY, identical on 2916 random cases | no | no | any real LoRa IQ capture |
| System Fusion · P6 | MMDVMHost: five FICH vectors from its encoder; YSF2DMR's silence pair | no — the only capture so far was an FM test tone | no | a real DN-mode capture; only V/D mode 2 is decoded |
| NXDN voice · P7 | MMDVMHost and DSD-FME constants | yes — two slices: NXDN96 36 voice frames, NXDN48 56 | the 21 Oct field run reports NXDN48/96 conventional holding through drift | none recorded |
| NXDN control channel · P8 | the same references | yes — 214 of 214 CAC frames pass the CRC | locked live in the 21 Oct trunked run | VCALL_ASSGN is reference-pinned only: the capture is idle |
| NXDN trunked voice · P8 | — | — | **yes, 21 Oct** — lock, grant follow, upload with TG / RID on a 10-channel system | one call ended at the carrier tracker's ±3 kHz bound; cause shown synthetically, unconfirmed on air |
| NXDN scrambler · P9 | DSD-FME's LFSR and the references' CRC-6; the schedule settled by the captures | yes — two known-key captures; the programmed key ranks first on both | no | the reporter's live run with the key on a scan-list channel |
| D-STAR · P10 | the MMDVM firmware's air format: its `txHeader` output and null AMBE frame | no | no | a real D-STAR capture |
| dPMR · P10 | voice Golay corrected to mbelib's code, pinned by MMDVMHost's silence vectors | no | no | a dPMR voice capture; the on-air interleave is still a placeholder |
| P25 Phase 2 · P11–12 | OP25 and SDRTrunk, which agree on every constant | partly — the SACCH MAC path on an idle carrier, valid with the site's seed only | no | a live call: voice, source RID, talker alias |
| SmartNet | OP25 and trunk-recorder | no | no | the #1143 reporter's 854.5625 MHz capture |

The scrambler row needs its one sentence: GopherTrunk decrypts only with
keys the operator already holds, for systems they are authorised to monitor
— there, a key the reporter programmed into their own radios.

<figure class="lab-figure">
<svg viewBox="0 0 680 232" width="680" height="232" role="img" aria-label="Three columns showing the highest verification rung each decoder has reached. Reference-pinned: LoRa, System Fusion, D-STAR, the dPMR voice Golay code, SmartNet, and FLEX 3200 and 6400 marked as reference-derived only. Capture-verified: FLEX 1600, NXDN conventional voice, NXDN control channel, NXDN scrambler and the P25 Phase 2 MAC path. On-air verified: MDC1200, FleetSync, CTCSS and DCS, AM and NXDN trunked voice, each with its date.">
  <text x="340" y="16" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the highest rung each decoder has reached</text>
  <rect x="10" y="30" width="210" height="172" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <rect x="235" y="30" width="210" height="172" rx="6" fill="none" stroke="currentColor"/>
  <rect x="460" y="30" width="210" height="172" rx="6" fill="none" stroke="var(--accent)"/>
  <line x1="220" y1="116" x2="235" y2="116" stroke="currentColor" fill="none"/>
  <line x1="445" y1="116" x2="460" y2="116" stroke="currentColor" fill="none"/>
  <g text-anchor="middle" font-size="10">
    <text x="115" y="50" fill="var(--fg-muted)">reference-pinned</text>
    <text x="340" y="50" fill="currentColor">capture-verified</text>
    <text x="565" y="50" fill="var(--accent)">on-air verified</text>
  </g>
  <g text-anchor="middle" font-size="9" fill="currentColor">
    <text x="115" y="74">LoRa</text><text x="115" y="90">System Fusion (YSF)</text><text x="115" y="106">D-STAR</text>
    <text x="115" y="122">dPMR voice Golay</text><text x="115" y="138">SmartNet</text><text x="115" y="154">FLEX 3200 / 6400 *</text>
    <text x="340" y="74">FLEX 1600 bps</text><text x="340" y="90">NXDN conventional voice</text><text x="340" y="106">NXDN control channel</text>
    <text x="340" y="122">NXDN scrambler</text><text x="340" y="138">P25 Phase 2 MAC path</text>
    <text x="565" y="74">MDC1200 — 28 Sep</text><text x="565" y="90">FleetSync — 16 Sep</text><text x="565" y="106">CTCSS / DCS — 24 Sep</text>
    <text x="565" y="122">AM — 29 Sep</text><text x="565" y="138">NXDN trunked voice — 21 Oct</text>
  </g>
  <text x="115" y="180" text-anchor="middle" font-size="8" fill="var(--fg-muted)">* reference-derived, synthetic-verified only</text>
  <text x="340" y="180" text-anchor="middle" font-size="8" fill="var(--fg-muted)">needs a recording to get here</text>
  <text x="565" y="180" text-anchor="middle" font-size="8" fill="var(--fg-muted)">needs a reporter's live run</text>
  <text x="340" y="222" text-anchor="middle" fill="var(--accent)" font-size="9">nothing moves a column to the right without evidence from outside the repository</text>
</svg>
<figcaption>Sixteen ledger rows by the highest rung reached: five on air, five on a capture, six still on references alone.</figcaption>
</figure>

## The errata

This series is also a correction. These posts described the formats the
decoders implemented at the time; each now carries a dated update, and
this is the index.

| Earlier post | What it described | Corrected in |
|---|---|---|
| [Beyond Voice 3]({{ '/blog/deep-dives/beyond-voice-03-mdc1200-template-decoder/' | relative_url }}) | an MDC1200 receiver that fed the sliced tones to the sync hunt as data | [Part 2]({{ '/blog/solution-postmortem/rebuilt-from-the-air-02-mdc1200-tones-as-data/' | relative_url }}) |
| [Beyond Voice 6]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }}), FLEX half | a mode code hunted after the sync marker, and one mode only | [Part 4]({{ '/blog/solution-postmortem/rebuilt-from-the-air-04-flex-four-modes/' | relative_url }}) |
| [Beyond Voice 11]({{ '/blog/deep-dives/beyond-voice-11-lora-chirp-receiver/' | relative_url }}), bit layer | invented whitening, CRC, header, Hamming layout, interleaver and chip mapping | [Part 5]({{ '/blog/solution-postmortem/rebuilt-from-the-air-05-lora-bit-layer/' | relative_url }}) |
| [Beyond Voice 13]({{ '/blog/deep-dives/beyond-voice-13-dstar-ysf-dpmr/' | relative_url }}) | the YSF FICH codec; D-STAR's sync words, header FEC and voice tables; dPMR's voice Golay | [Part 6]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }}), [Part 10]({{ '/blog/solution-postmortem/rebuilt-from-the-air-10-dstar/' | relative_url }}) |
| [Protocol Decoders 6]({{ '/blog/deep-dives/protocol-decoders-06-nxdn-dpmr/' | relative_url }}), NXDN half | a 16-bit frame sync, a 32-dibit SACCH, no scrambler | [Part 7]({{ '/blog/solution-postmortem/rebuilt-from-the-air-07-nxdn-frame-layer/' | relative_url }}), [8]({{ '/blog/solution-postmortem/rebuilt-from-the-air-08-nxdn-control-channel/' | relative_url }}), [9]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }}) |
| [Protocol Decoders 4]({{ '/blog/deep-dives/protocol-decoders-04-p25-phase-2-tdma-mac/' | relative_url }}), [P25 End to End 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }}) | a Golay-protected ISCH, alternating slots, a trellis → RS(24,16,9) MAC chain | [Part 11]({{ '/blog/solution-postmortem/rebuilt-from-the-air-11-p25-phase2-airframe/' | relative_url }}), [Part 12]({{ '/blog/solution-postmortem/rebuilt-from-the-air-12-pn44-gardner-real-carrier/' | relative_url }}) |

Two earlier finales listed open items that have since moved. From
[Beyond Voice, Part 14]({{ '/blog/deep-dives/beyond-voice-14-adding-a-decoder-checklist/' | relative_url }}):

- **"FleetSync on a live Kenwood fleet"** — on-air verified on 16 Sep: FS-I
  and FS-II both decode live in the reporter's Kenwood lab. The list was
  out of date the day it ran.
- **"YSF's FICH caller"** — the functions it names no longer exist. The
  FICH model was fabricated; `fich.go` and `fich_trellis.go` are deleted,
  and the control channel grants from on-air FICH through an `AirFramer`,
  pinned failing-first by `TestControlChannelGrantsFromOnAirFrames`. Still
  unverified on air.
- **"D-STAR and dPMR voice tables"** — D-STAR was rebuilt end to end to the
  MMDVM firmware's air format, its header pinned by
  `dstar_header_mmdvm.txt`; a real capture is still the gate. dPMR's voice
  Golay was corrected; its on-air interleave is still a placeholder.
- The rest of that list — M17's panel, dPMR's CSBK coding, the untested
  checklist rows — is outside this series, and no change is claimed.

From
[DMR End to End, Part 14]({{ '/blog/deep-dives/dmr-end-to-end-14-playbook/' | relative_url }}),
one item moved and one did not. **"Enhanced Privacy through the daemon"** —
the daemon's decrypt path can now be checked offline with the release
binary, from a discriminator recording and an operator-held key:

```bash
gophertrunk replay -in ep-call.wav -format disc -protocol dmr-tier2 -freq 1 \
                  -record-voice -out-dir rec -key rc4:11:4E77AD0B51
```

`TestReplayDecryptsRealAirEnhancedPrivacyFromDiscAudio` pins it on committed
real-air ciphertext: 54 of 54 frames clear with the key, 54 of 54
ciphertext without. The reporter's own run, or a live call, is still open
on #1187
([Cookbook Volume 2, Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }})
is the recipe). **"Direct mode on a live build"** is unchanged:
capture-verified on the reporter's 15 Sep captures, with the live run on a
fixed build still the gate on #836.

Two corrections fall outside the rebuilds. Protocol Decoders
[10]({{ '/blog/deep-dives/protocol-decoders-10-alias-hunt-framing/' | relative_url }})
and
[11]({{ '/blog/deep-dives/protocol-decoders-11-alias-hunt-cryptanalysis/' | relative_url }})
describe the Motorola talker-alias cipher as unsolved and gated. It is
solved and enabled — `motorola.CipherVerified = true` — verified on 1242 of
1242 held-out characters and on a real capture that decodes to "CRIO 0062";
on-air confirmation on the reporter's rig is still open
([operator guide]({{ '/talker-alias.html' | relative_url }})). And the
[PN44 glossary entry]({{ '/reference/pn44-scrambler/' | relative_url }})'s
figure still draws the mirrored taps Part 12 corrected.

## What we keep

The series ends where
[From Spec to Shipping]({{ '/blog/deep-dives/from-spec-to-shipping-14-definition-of-verified/' | relative_url }})
did, because it is the project's operating principle: green synthetic is
not on-air correct.

- **A decoder is a claim about someone else's transmitter.** Only
  something from outside the repository can test it — a reference's
  literal output, an encoder you did not write, a recording.
- **Say which of the four words you have earned.** "Works" is not one of
  them. Six ledger rows say *no* twice, and that is their accurate state.
- **A fabricated layer hides under a correct one.** LoRa's chirp was right
  over six invented layers; the rebuilt Phase 2 frame sat over a mirrored
  scrambler. Audit each layer on its own evidence.
- **Correct the prose when you correct the code.** A post that explains an
  invented format teaches it.
- **Every open gate is a named recording.** None is blocked on effort.

## Where to go from here

The [series index]({{ '/blog/series/rebuilt-from-the-air/' | relative_url }})
lists all fourteen parts. Two series ran beside this one.
[Squelch]({{ '/blog/series/squelch/' | relative_url }}) is the design half
of stories told here as investigations — how a scan list decides a call
exists, from the CTCSS and DCS gates to the System Fusion, NXDN and D-STAR
call gates.
[The Operator's Cookbook, Volume 2]({{ '/blog/series/operator-cookbook-volume-2/' | relative_url }})
is the bench half: recipes for the rigs these decoders run on — NXDN,
fleet signalling, paging, your own keys. And
[From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }})
is the method this series kept having to apply after the fact.

## FAQ

**How can I tell whether a decoder has ever decoded a real signal?**
Look at where its fixtures come from. If every test vector is produced by
the project's own encoder, nothing in the suite can disagree with the
decoder's idea of the format. Look for a committed real-air slice, or a
literal vector attributed to an independent decoder.

**Which GopherTrunk decoders in this series are verified on air?**
MDC1200 (28 Sep), FleetSync (16 Sep), the CTCSS and DCS tone gates
(24 Sep), AM (29 Sep) and NXDN trunked voice (21 Oct). FLEX 1600 bps, the
NXDN control channel and scrambler and the P25 Phase 2 MAC path are
capture-verified; LoRa, System Fusion, D-STAR and SmartNet have not yet
met a recording.

**What capture would help most right now?**
Any open row: a LoRa IQ capture, a System Fusion DN-mode transmission at
a gain that does not clip, a D-STAR or dPMR voice transmission, a
929–932 MHz multi-rate FLEX channel, a P25 Phase 2 voice call, or a
SmartNet control channel. Record IQ, not audio, and say what the radio was
doing.

**Were the earlier posts wrong when they were published?**
They described the code accurately; the code was wrong about the air. Each
now carries an update naming the part that corrects it. The exception is
one finale's open list, which had FleetSync waiting on a live test two
weeks after it passed.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Read the Capture First — The 'YSF' File That Was an FM Test Tone]({{ '/blog/solution-postmortem/rebuilt-from-the-air-13-read-the-capture-first/' | relative_url }})
· [Back to the series index]({{ '/blog/series/rebuilt-from-the-air/' | relative_url }})
