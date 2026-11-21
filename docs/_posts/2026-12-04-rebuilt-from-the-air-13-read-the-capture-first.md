---
title: "Rebuilt From the Air, Part 13: Read the Capture First — The 'YSF' File That Was an FM Test Tone"
description: "Six captures that were not what the report said — an FM test tone filed as System Fusion, re-recorded audio filed as discriminator taps, a float WAV read as int16, a slice carved at the tuner centre, grabs with holes — and the triage instrument each left behind. Check container, level, spectrum and sync spacing before any DSP theory."
category: solution-postmortem
keywords: iq capture triage, adc rail share clipping, discriminator audio vs iq, system fusion capture analog fm, sdr sharp float wav, iq hole frame sync spacing, capture carved at tuner centre, replay harness verdict, sdr debugging checklist, gophertrunk capture triage
tags: [rebuilt-from-the-air, captures, triage, debugging, testing, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Rebuilt From the Air"
series_part: 13
---

*Part 13 of **Rebuilt From the Air**, a 14-part postmortem series on the
GopherTrunk decoders that passed every test and never matched a real radio —
each diagnosed from a reference implementation or a reporter's capture,
rebuilt against independent decoders, and pinned to real air where a capture
exists.
[Part 12]({{ '/blog/solution-postmortem/rebuilt-from-the-air-12-pn44-gardner-real-carrier/' | relative_url }})
ended on a file with a hole in it. This part is the other side of "pinned to
real air": the captures that were not what the report said they were, and
the instrument each one left in the tree so the next is recognised in a
minute instead of a week.*

> **TL;DR:** A capture is evidence only once you know what it is. The first
> "System Fusion" capture on #1261 was an analog FM carrier with a 1 kHz
> test tone, 1.4 kHz off frequency, 57 % of its samples at the ADC rail.
> `TestYSFReplay` now prints rail share, the discriminator's strongest line,
> the 4-level eye and a `VERDICT`, and `conventional.YSFCallDetector` stops
> a `ysf` scan-list channel opening a call on power alone. Five more: DMR
> "RC4 audio" that was re-recorded audio (20–34 % of its energy below 3 kHz,
> against 92 % for a real tap); an SDR# float WAV read as int16, its carrier
> 424.5 kHz off centre; a Signal Lab slice carved at the tuner centre; nine
> MMR grabs with holes; and P25 voice "loss" that was frame syncs 790–850
> symbols apart instead of 864. Check container, level, spectrum and sync
> spacing before any DSP theory.

**Key takeaways**

- **Ask what the file is before asking why it does not decode.** Four cheap
  measurements — container, rail share, spectrum, sync spacing — settled
  every case here without touching a demodulator.
- **A misread capture should leave an instrument, not a memory.**
  `discriminatorAudioSanity`, the harness `signal:` line,
  `requested_center_hz` and `FrameSlips` each exist because one file fooled
  someone once.
- **The wrong capture can still find a real bug.** The FM tone was not
  System Fusion, and it showed that a `ysf` channel squelched on power alone.
- **Clipped is a fact, not a verdict.** The #836 direct-mode captures decode
  at 36–40 % clipped; the "YSF" file failed because it was not YSF.

## Cheat sheet

| | |
|---|---|
| Issue | [#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261) (11 Nov), with #1187, #1184, #1364 and #1242 as the supporting cases |
| Symptom | a `ysf` scan-list channel opened a call and recorded nothing — "No actual sound was detected" |
| Wrong theories | the rebuilt YSF voice path is still wrong; the gain is too low; the tuner is off frequency |
| Real cause | the capture was an analog FM carrier with a 1 kHz tone, 57 % of samples at the ADC rail; and `ysf` channels squelched on in-channel power alone |
| Fix | `conventional.YSFCallDetector`, a frame-sync gate; `sync_hits` / `adc_rail_pct` and a WARN on the chain's ended line; feed-forward timing in the YSF receiver |
| Diagnostic | `TestYSFReplay`: rail share, carrier offset, strongest discriminator line, 4-level eye, `VERDICT` |
| Status | the gate is pinned shut on the reporter's real-air slice; YSF decode itself is still **unverified on air** — a real DN-mode capture is the gate |

## In this post

- **The report: a call that recorded nothing** — and three theories that fit it.
- **The harness that says what a capture is** — four measurements and a verdict.
- **The bug the wrong capture found** — a power-only squelch, and a frame-sync gate.
- **Five more captures that were something else** — audio, a WAV, a slice, nine grabs, missing samples.
- **The triage order** — container, level, spectrum, spacing, then theory.
- **What we keep** — the rules, including two about reports themselves.

## The report: a call that recorded nothing

On 11 November the reporter on #1261 ran a `ysf` scan-list channel on
448.900 MHz and posted the per-call voice IQ debug capture
(`baseband.voice_iq_debug`: an RTL-SDR Blog V4 at 2.4 MS/s, cs16, tuned on
channel). GopherTrunk had opened a "ysf" call and recorded nothing — "No
actual sound was detected". The chain's log did not say why.

Three theories fit that report, and each would have cost days.
[Part 6]({{ '/blog/solution-postmortem/rebuilt-from-the-air-06-ysf-fich/' | relative_url }})
had replaced a fabricated FICH model with the real air format,
**reference-pinned** and unverified on air; a first real capture that
decodes nothing is exactly what a still-wrong rebuild looks like. The other
two are the perennials: the gain is wrong, or the tuner is off frequency.
All three are DSP theories. None of them asks what is in the file.

## The harness that says what a capture is

`TestYSFReplay` is skip-guarded on `GT_YSF_IQ`. It mirrors the voice chain
— downconvert to 48 kHz, the production YSF receiver with carrier
correction, channel filter and timing acquisition, frames cut at each FSW,
FICH decoded — and then, in the words of its own comment, "says what the
capture IS when nothing framed":

```bash
GT_YSF_IQ=<capture> go test ./cmd/gophertrunk -run 'TestYSFReplay$' -v
```

| Measurement | How | What this capture showed |
|---|---|---|
| ADC rail share | samples with I or Q at ±0.99 or beyond, on the raw file | 57 % — the constellation is the clipping square |
| Carrier offset | mean of the discriminator output | 1.4 kHz off frequency |
| Discriminator spectrum | Hann-windowed 4800-point blocks; strongest line against the median bin | one line at 1000 Hz, 40 dB over the median |
| [4-level eye]({{ '/reference/eye-diagram/' | relative_url }}) | percentiles of the recovered symbols, scaled to ±1 and ±3 | twelve discrete levels — a sine sampled 4.8 times per cycle, not four |

A C4FM data signal has a continuous discriminator spectrum. An analog test
tone is one line far above everything else. The verdicts are in the code:

```go
// cmd/gophertrunk/ysf_replay_test.go (shape)
if db[best]-median > 40 && frames == 0 {
	t.Logf("VERDICT: not a System Fusion transmission — the "+
		"discriminator is a single %.0f Hz tone …", toneHz)
}
switch {
case railPct > 50:
	t.Logf("VERDICT: the front end is overloaded — %.0f%% of "+
		"samples sit at the ADC rail; lower the gain …", railPct)
case frames == 0:
	t.Logf("VERDICT: no YSF frame sync in the capture")
}
```

So the radio was transmitting analog FM with a test tone, with the handheld
next to the dongle. The voice chain decoded nothing from it — correctly.
The rebuilt decoder was not on trial; this file could not put it there.

## The bug the wrong capture found

The capture was not System Fusion, and the scanner had still opened a "ysf"
call on it. That was the real defect. A `ysf` scan-list channel — like NXDN
and D-STAR ones — squelched on in-channel **power** alone, so any carrier
opened a call: analog FM, a test tone, another digital mode.

`conventional.YSFCallDetector` (`ysf_gate.go`) closes it. It runs the
production YSF receiver on the channel's own narrowband IQ and reports
present only while a frame sync — the FSW that opens every 100 ms frame —
was heard within `ysfCallHold`, 500 ms. The scanner used to type-assert
`*DMRCallDetector`; it now asks for a `callGate` interface that both gates
satisfy. `TestScannerYSFAnalogCarrierOpensNoCall` fails without the gate,
and a 1 s slice of the reporter's own file,
`testdata/ysf_issue1261_analog_tone_448900k_48k.cs16`, is committed so
`TestYSFCallDetectorIgnoresRealAirAnalogTone` pins it on real air. The
gate's design belongs to
[Squelch, Part 9]({{ '/blog/deep-dives/squelch-09-call-gates-ysf-nxdn-dstar/' | relative_url }}).

Building the gate exposed the timing lesson Part 12 met on Phase 2. From
three of ten start phases the receiver's Mueller-Müller loop found its
first FSW 0.6–1.2 s in — past the 500 ms scan window. With
`ArmTimingAcquisition` it is 0.11 s at every phase, and the voice chain
arms it too.

And the chain now says what it saw. Its ended line carries `sync_hits` and
`adc_rail_pct`, and a call with no sync at all gets a WARN:

```text
composer: ysf call carried no YSF frame sync at all — the carrier was not
System Fusion C4FM (analog FM, a test tone, another digital mode), or the
front end was overloaded; check adc_rail_pct and lower the gain if it is high
```

What that leaves is exact. The gate is pinned on real air — staying shut.
System Fusion *decode* has still not met a real transmission. The next
capture request names the radio's mode (DN) and a lower gain.

## Five more captures that were something else

The same thing had happened five times already. Each left an instrument.

| Report | What the file was | Instrument it left |
|---|---|---|
| #1187 — DMR "RC4 audio" | decoded audio re-recorded at 96 kHz, not a discriminator tap | `discriminatorAudioSanity` |
| #1184 — FleetSync "decodes nothing" | 32-bit float WAV read as int16; carrier 424.5 kHz off centre | harness `signal:` line, `IQWavInfo.Encoding`, `fleetSyncCarrierCandidates` |
| 15 Sep — a "442.8125 MHz" slice | carved at the tuner centre, 441.7 MHz | `requested_center_hz`, strict `parseCaptureTuning` |
| #1364 — nine MMR files | separate grabs, five with a hole in them | S-ISCH spacing, the `AirFramer` re-anchor |
| #1242 — P25 "weak signal" | 2–3.5 % of every call's samples missing | `FrameSlips`, `short_ldus`, the `voice IQ has holes` WARN |

**Audio that was never a tap.** On
[#1187](https://github.com/MattCheramie/GopherTrunk/issues/1187) the
reporter posted three 96 kHz mono WAVs. The P25 one is a genuine C4FM
discriminator tap: re-modulated through `TestP25ADPReplay` it yields 15
LDU1 and 15 LDU2 FEC-clean, and with the key the operator held the IMBE
pitch track goes from 0.13 to 0.57. (GopherTrunk decrypts only with keys
the operator already holds, for systems they are authorised to monitor.)
The two DMR files were white-noise bursts gated at an exact 30/30 ms slot
cadence with a 15.5 kHz codec cliff — no 4-level eye, no symbol-clock line.
A demodulated 4800-baud signal keeps nearly all its energy below 3 kHz, so
`discriminatorAudioSanity` measures that share: 92 % on the P25 file, 20–34 %
on the DMR ones. Its first version measured the whole file and called a
genuine squelch-off tap "not a tap" at 47 %, because receiver noise between
transmissions is wideband; it now takes the 95th percentile of 20 ms
blocks. `replay -format disc` prints the line. The genuine captures that
followed are in
[DMR End to End, Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }}).

**A WAV that lied about its samples.** The
[#1184](https://github.com/MattCheramie/GopherTrunk/issues/1184) FleetSync
recordings are told in full in
[Beyond Voice, Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }}).
The tell was the harness's level line: a real capture does not sit at
−2.5 dBFS with a 0.16 DC offset. SDR# writes 32-bit float; every reader
assumed 16-bit PCM. Then the carrier sat +424.5 kHz and +323 kHz off
centre, because a baseband recording is the whole tuner span. Then
`dsp.EstimateCarrierCandidatesHz`, which scans only its first few tens of
milliseconds, looked before the radio keyed up — it ranked a −423 kHz spur
first and never saw a 64 dB carrier.

**A slice carved somewhere else.** On 15 September an operator typed
442.8125 MHz into Signal Lab and got a file whose only carriers sat at
−11, −37, −61, −87 and +213 kHz. Neither repeater was in it. The form
coerced the field with `Number()` and fell back silently on NaN — a decimal
comma or a pasted invisible character is enough — so the slice was carved
at the tuner centre, 441.7 MHz, under the name the operator typed. The
start line now carries `requested_center_hz` beside `tuner_center_hz`,
`parseCaptureTuning` refuses anything that is not a plain decimal, and the
route rejects a centre without a bandwidth
([Field Notebook, Part 9]({{ '/blog/tutorials/field-notebook-09-capture-lines/' | relative_url }})).

**Nine files that are not one recording.** Part 12's `mmr-s9-*.cfile` set:
separate grabs, five with a hole where whole USB buffers were dropped.

**"Weak signal" that was missing samples.** On
[#1242](https://github.com/MattCheramie/GopherTrunk/issues/1242)
consecutive P25 LDU frame syncs landed 790–850 symbols apart instead of the
fixed 864. Between 2 and 3.5 % of every call's samples were missing, in
8–14 ms gaps that hit a third of the LDUs, while the eye measured 12–25 %
EVM — decodable — and the control channel decoded fine. The `iqtap.Broker`
subscriber channels were 16 chunks deep: about 60 ms of RTL-SDR chunks, but
about 2.4 ms of ~360-sample SoapyRemote datagrams. They are now sized in
time (`SubBufferLatency`, 400 ms). The voice chain also measures the
spacing itself: `LDUAssembler` re-anchors on an early sync and counts it,
the `composer: p25p1 decode quality` line carries `short_ldus` and
`missing_symbols`, and a `voice IQ has holes` WARN fires at call end.
Present means IQ was lost upstream; absent means signal or demod
([Field Notebook, Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})).

## The triage order

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="Five boxes in a row joined by arrows: container, level, spectrum, sync spacing, and only then DSP theory. Under each of the first four is the capture that step caught: a float WAV read as int16, 57 percent of samples at the rail, a single 1000 hertz line and a carrier 424.5 kilohertz off centre, and frame syncs 790 to 850 symbols apart instead of 864.">
  <text x="340" y="16" text-anchor="middle" fill="var(--accent)" font-size="9">ask what the file is, in this order — each step is cheaper than the next</text>
  <g fill="none" stroke="currentColor">
    <rect x="10" y="30" width="120" height="34" rx="6"/><rect x="145" y="30" width="120" height="34" rx="6"/>
    <rect x="280" y="30" width="120" height="34" rx="6"/><rect x="415" y="30" width="120" height="34" rx="6"/>
    <line x1="130" y1="47" x2="145" y2="47"/><line x1="265" y1="47" x2="280" y2="47"/>
    <line x1="400" y1="47" x2="415" y2="47"/><line x1="535" y1="47" x2="550" y2="47"/>
  </g>
  <rect x="550" y="30" width="120" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <g text-anchor="middle" font-size="9" fill="currentColor">
    <text x="70" y="51">1 · container</text><text x="205" y="51">2 · level</text>
    <text x="340" y="51">3 · spectrum</text><text x="475" y="51">4 · sync spacing</text>
  </g>
  <text x="610" y="51" text-anchor="middle" font-size="9" fill="var(--accent)">5 · DSP theory</text>
  <g text-anchor="middle" font-size="8" fill="var(--fg-muted)">
    <text x="70" y="84">float32 WAV read</text><text x="70" y="96">as int16 (#1184)</text>
    <text x="205" y="84">57 % at the rail (#1261);</text><text x="205" y="96">−2.5 dBFS, DC 0.16 (#1184)</text>
    <text x="340" y="84">one 1000 Hz line (#1261);</text><text x="340" y="96">carrier +424.5 kHz off</text>
    <text x="475" y="84">LDU syncs 790–850</text><text x="475" y="96">apart, not 864 (#1242)</text>
    <text x="610" y="84">only when 1–4 say</text><text x="610" y="96">the file is sound</text>
  </g>
  <text x="340" y="130" text-anchor="middle" fill="currentColor" font-size="9">every wrong diagnosis in this post started at box 5</text>
</svg>
<figcaption>Four cheap questions about the file come before the first question about the receiver, and each one caught a real capture.</figcaption>
</figure>

1. **Container.** Sniff the content and read the fmt chunk; never trust an
   extension or a label. A [WAV]({{ '/reference/wav-iq-recording/' | relative_url }})
   can hold 8-bit, 16-bit or float samples, IQ or discriminator audio.
2. **Level.** Rail share, RMS and DC. The harnesses print these first
   because a dead or absurd level answers nothing
   ([gain staging]({{ '/reference/sdr-gain-overload/' | relative_url }})).
3. **Spectrum.** Is the carrier in the file, and where? What does the
   discriminator carry — a continuous spectrum, or one line?
4. **Sync spacing.** Are frame syncs the fixed distance apart? Early syncs
   mean missing samples, whatever the eye looks like.
5. **Then theory.**

One caution keeps step 2 honest. Rail share is a measurement to report, not
a verdict to act on alone. The
[#836 direct-mode captures]({{ '/blog/deep-dives/dmr-end-to-end-09-direct-mode-carrier-gate/' | relative_url }})
decode with 36–40 % of samples clipped, and an MDC1200 recording clipped at
both rails replays CRC-valid. The "YSF" file earned an *overloaded* verdict
at 57 % and a *not System Fusion* verdict from its spectrum; only the
second explains the silence. An earlier postmortem,
[Nineteen Dibits]({{ '/blog/solution-postmortem/from-the-issue-tracker-08-nineteen-dibits/' | relative_url }}),
is the case where the rail *was* the answer.

## What we keep

- **Read the capture before any DSP theory.** The most expensive mistake
  available is tuning a receiver against a file that never held the signal.
  [Capture discipline]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }})
  is the operator's half of this; triage is the developer's.
- **Leave an instrument in the tree.** A lesson in someone's head is lost
  with the next capture. A verdict line in a harness, or a field on a log
  line, is there for whoever opens the next one
  ([Instruments, Not Logs]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})).
- **Write the verdict in words.** "Energy below 3 kHz: 27 %" needs an
  expert; "NOT a demodulable 4FSK discriminator tap" does not.
- **A slice of what the signal was *not* is a fixture too.** The analog-tone
  slice pins a gate shut. It is as much a real-air regression as a slice
  that decodes.
- **Check the build the report ran.** One "the fix didn't work" report
  tested a release cut two days before the fix merged;
  `git merge-base --is-ancestor <fix> <tag>` answers that in a second.
- **Check a config "fix" against the log.** A setting credited with curing
  sped-up audio could not have changed it; the same log showed
  `host_drops=3873` just before the bad audio.

## Where this goes next

[Part 14]({{ '/blog/solution-postmortem/rebuilt-from-the-air-14-audit-checklist-errata/' | relative_url }})
closes the series: the checklist for telling a real decoder from a
fabricated one, the ledger of what is on-air verified and what still waits
for a recording, and the errata for the earlier posts this series corrects.

## FAQ

**How do I tell whether an IQ capture is clipped?**
Count the samples whose I or Q sits at the converter's rail. `TestYSFReplay`
counts values at ±0.99 or beyond on the raw file and prints the share; the
voice chain reports the same idea as `adc_rail_pct`. A high share means
lower the gain or move the antenna — but a clipped capture can still
decode, so read it alongside the spectrum.

**Why did GopherTrunk open a YSF call on an analog FM signal?**
Because a `ysf` scan-list channel used to squelch on in-channel power alone,
and any carrier has power. `conventional.YSFCallDetector` now opens the
channel only when the production YSF receiver hears a frame sync within
500 ms. It is pinned on a slice of the reporter's own analog-tone capture.

**Can GopherTrunk decode DSD-style discriminator audio instead of IQ?**
Yes: `gophertrunk replay -format disc` re-modulates a discriminator-audio
WAV or FLAC into IQ and runs the normal decode. It first prints a sanity
line — the share of energy below 3 kHz in the most signal-like 20 ms blocks
— and says plainly when the file is not a tap at all.

**How do I know whether garbled P25 voice is weak signal or lost samples?**
Read the call's log first. If a `voice IQ has holes` WARN is present, frame
syncs arrived early and samples were lost upstream of the decoder — look
for overrun and `host_drops` lines and CPU load. If it is absent, the
problem is the signal or the demodulator.

**Is System Fusion decoding verified on air?**
No. The FICH and V/D mode 2 voice are reference-pinned against MMDVMHost,
and the scan-list gate is pinned on a real capture of something that was
not System Fusion. A real DN-mode transmission, at a gain that does not
clip, is still the gate.

## Series navigation

**Part 13 of 14** · ←
[Part 12: P25 Phase 2 II — Mirrored Taps, a Sign Error, and the Carrier Already in the Repo]({{ '/blog/solution-postmortem/rebuilt-from-the-air-12-pn44-gardner-real-carrier/' | relative_url }})
· Next →
[Part 14: The Audit Checklist — and the Errata for This Blog]({{ '/blog/solution-postmortem/rebuilt-from-the-air-14-audit-checklist-errata/' | relative_url }})
