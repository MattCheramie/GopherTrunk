---
title: "Beyond Voice, Part 5: Two-Tone Paging — Goertzel, Profiles & Tone-Out"
description: "How GopherTrunk's tone-out detector turns decoded PCM into fire-paging alerts: a block Goertzel resonator per configured tone, a per-device sequence matcher with duration, gap and cooldown gates, and the two places where this decoder deliberately skips the eleven-place pattern because it listens to audio, not RF."
category: deep-dives
keywords: two tone paging decoder, goertzel tone detection go, quick call 2 sdr, fire tone out software, tone sequence state machine, kindtonealert events bus, pcm sink fan out, tone_out profiles config, sdr pager tone alerts, gophertrunk tone-out
tags: [beyond-voice, tone-out, goertzel, paging, events, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Beyond Voice"
series_part: 5
---

*Part 5 of **Beyond Voice**, a 14-part deep dive into everything GopherTrunk
decodes that is not a trunked voice call — the AFSK signalling formats, paging,
APRS, ADS-B, AIS, DSC, LoRa, M17 and the amateur digital-voice family — and the
one eleven-place wiring pattern that carries each of them from a burst on the
air to a row in the web console.
[Part 4]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
cloned the MDC1200 template for FleetSync and learned what a mislabelled WAV
can hide. This part turns to the oldest paging format still on the air — the
two-tone sequential page that dispatches volunteer fire stations — and to the
one decoder in this series that never sees RF — which changes where it plugs
in, what gates it, and which of the eleven places it needs.*

> **TL;DR:** `internal/voice/toneout` is three files. `goertzel.go` is a block
> Goertzel resonator — `NewGoertzel` rounds the target to the nearest exact
> bin, runs `s0 = x + coeff·s1 − s2` per sample and emits a normalised squared
> magnitude every `blockSize` samples (800 at 8 kHz = 100 ms, 10 Hz
> resolution). `profile.go` is the `Profile`/`Tone` schema with `Validate`
> defaults (`ToleranceHz` 15, `MagnitudeThreshold` 0.05, `MaxGap` 200 ms,
> `Cooldown` 30 s). `detector.go` is a per-device state machine that satisfies
> `composer.PCMSink`, counts contiguous above-threshold blocks against each
> tone's `MinDuration`/`MaxDuration`, tracks gaps, and fires
> `events.KindToneAlert` (`tone.alert`) plus `toneout: profile matched`. The
> daemon fans PCM to it beside the recorder via `fanoutSink`, so it hears
> analog dwells and digital talkgroups alike — and it skips two of the eleven
> places on purpose: no SQLite table, no REST read.

**Key takeaways**

- **Tone-out is an audio decoder, so it plugs in after the vocoder.** Every
  other Beyond Voice decoder subscribes to an IQ broker; this one implements
  `WritePCM` and receives the 8 kHz stream the recorder writes.
- **A block Goertzel is a DFT bin without the FFT.** One multiply-add per
  sample per tone, a closing step every 100 ms, and a normalisation that makes
  `magnitude_threshold` mean one thing at every block size.
- **The matcher is a duration gate, not a frequency gate.** A tone counts
  when enough consecutive blocks clear the threshold; too short or too long a
  gap resets the sequence, and a cooldown suppresses re-fires.
- **Not every decoder needs all eleven places.** Tone alerts have no
  `tone_log` table and no `GET` route; the panel reads `tone.alert` off SSE.
  Knowing which places a decoder can skip is part of knowing the pattern.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Single-bin resonator | block Goertzel, bin-rounded, normalised to ~[0,1] | `internal/voice/toneout/goertzel.go` (`NewGoertzel`, `Process`) |
| Profile schema + defaults | tones, tolerance, threshold, gap, cooldown, scope | `profile.go` (`Profile.Validate`) |
| Sequence matcher | per-device, per-profile progress over 100 ms blocks | `detector.go` (`advanceProfiles`) |
| PCM entry point | `composer.PCMSink`, fanned beside the recorder | `Detector.WritePCM`; `fanoutSink` in `cmd/gophertrunk/daemon.go` |
| Bus event + log line | `tone.alert` + `INF toneout: profile matched` | `events.KindToneAlert`; `Detector.fire` |
| Reset write side | clear match progress, keep the cooldown clock | `POST /api/v1/devices/{serial}/tone-reset` (`handleToneReset`) |

## In this post

- **Audio in, not IQ in** — where a PCM-side decoder sits in the daemon.
- **One bin, one resonator** — the Goertzel recurrence and its normalisation.
- **Profiles and the sequence matcher** — durations, gaps, cooldown, scope.
- **Which of the eleven places it needs** — and the two it skips.
- **Testing a page without a pager** — synthetic sines and a fake clock.

## Audio in, not IQ in

Every decoder in Parts 2–4 opened the same way: subscribe to an SDR's IQ
broker, FM-demodulate, discriminate two tones, slice, frame. Tone-out starts
where those end. The package doc says it detects "Two-Tone Sequential
(Motorola Quick Call II), single-tone, and DTMF — over the PCM stream produced
by the voice composer" (DTMF is aspirational — concurrent tones are "out of
scope for v1"). The placement is the point: the detector
implements the composer's `PCMSink` interface, the `WritePCM` shape
`voice.Recorder` exposes, so it consumes audio that has already been decoded.

The daemon wires it in twice: into the composer's sink list beside the
recorder, and into the live-audio arms behind
`SetDecodedPCMSink(fanoutSink(liveSinks))`. `fanoutSink` writes one PCM frame
to several sinks, and `Detector.WritePCM` always returns nil "so composer
chains using a fan-out sink don't abort on detector hiccups". The second
wiring exists because of a bug: P25, DMR and NXDN reach the composer as raw
vocoder frames via `WriteRawFrame`, which only the recorder decodes, so a
detector on the first list alone heard analog FM and nothing digital. The
recorder now exposes a decoded-PCM tap fanned to the live arms
([Recording & Streaming Part 12]({{ '/blog/deep-dives/recording-streaming-12-live-listening/' | relative_url }})),
which is what the
[cookbook recipe]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
leans on: a profile hears a page over a trunked talkgroup exactly as over
analog dispatch. `toneout.New` receives `cfg.Recordings.SampleRate` and
defaults `BlockSize` to `SampleRate/10` — 800 samples at 8 kHz.

## One bin, one resonator

The detector's only signal-processing primitive is the
[Goertzel algorithm]({{ '/reference/goertzel-algorithm/' | relative_url }}),
the classic way to evaluate one
[DFT]({{ '/reference/discrete-fourier-transform/' | relative_url }}) bin
without the rest of the spectrum. The constructor quantises the choice: `k = math.Round(N·targetHz/sampleHz)`,
`coeff = 2·cos(2πk/N)`, `normalize = 1/N²`. Rounding `k` puts the target on an
exact bin centre "so the algorithm stays numerically stable", at the cost of a
resolution of `sampleHz/blockSize` — 10 Hz at the defaults. A Quick
Call II tone at 1042.2 Hz is watched by the 1040 Hz bin; the miss costs a
little magnitude, which is why `ToleranceHz` defaults to 15 Hz. One honest
note: `ToleranceHz` is validated and defaulted in `profile.go` but the v1
matcher never consults it — selectivity *is* the bin, and `matchedFreqs`
records the configured target ("live-frequency refinement is a follow-up").

```go
// internal/voice/toneout/goertzel.go
func (g *Goertzel) Process(sample int16) (float64, bool) {
    x := float64(sample) / 32768.0
    s0 := x + g.coeff*g.s1 - g.s2
    g.s2 = g.s1
    g.s1 = s0
    g.count++
    if g.count < g.blockSize {
        return 0, false
    }
    mag2 := g.s1*g.s1 + g.s2*g.s2 - g.coeff*g.s1*g.s2
    g.Reset()
    return mag2 * g.normalize * 4, true // factor of 4 ≈ unit-amplitude sine → 1
}
```

Two decisions here shape everything downstream. The output is **block-aligned,
not sliding**: state resets every 800 samples, so the detector's clock ticks
in 100 ms steps and every profile duration rounds to that grid. And the
**normalisation** (`1/N²`, times 4) makes a full-scale sine at the bin centre
read about 1.0 at any block size, which lets `magnitude_threshold` default to
a fixed 0.05 ("~10 dB above quiet noise").
`TestGoertzelMagnitudePeaksAtTarget` pins the scale: a 0.6-amplitude tone reads
above `0.6²·0.7 ≈ 0.25` on its bin and below 0.05 a kilohertz away — the same
primitive [DTMF]({{ '/reference/dtmf/' | relative_url }}) and
[CTCSS]({{ '/reference/ctcss/' | relative_url }}) decoders use.

## Profiles and the sequence matcher

A `Profile` is an ordered list of `Tone{FrequencyHz, MinDuration,
MaxDuration}` plus policy. `Validate` fills defaults and refuses the unusable
— empty name, no tones, non-positive frequency or `MinDuration`, a
`MaxDuration` below its minimum — and the daemon's `toneProfilesFromConfig`
parses the YAML's Go duration strings (`"250ms"`, `"2.5s"`) first; `System`
and `GroupID` optionally scope a profile. Construction indexes the unique frequencies across all profiles (keyed to
0.1 Hz) so profiles sharing an A tone share one resonator, and state is **per
device**: `stateFor(serial)` lazily creates one `Goertzel` per frequency and
one `matchProgress` per profile, so calls on different SDRs "don't
cross-contaminate match progress". The matcher runs once per completed block:

```go
// internal/voice/toneout/detector.go (shape)
if mag >= profile.MagnitudeThreshold {
    prog.contiguousBlocks++
    prog.gapBlocks = 0
} else if prog.contiguousBlocks > 0 {
    dur := time.Duration(prog.contiguousBlocks) * d.blockDur
    if dur >= expected.MinDuration &&
        (expected.MaxDuration == 0 || dur <= expected.MaxDuration) {
        prog.matchedFreqs = append(prog.matchedFreqs, expected.FrequencyHz)
        prog.toneIdx++
        if prog.toneIdx >= len(profile.Tones) { d.fire(serial, profile, prog, now) /* …stamp lastFiredAt */ }
    } else {
        prog.toneIdx = 0 // too short: restart from tone 0
    }
} else if prog.toneIdx > 0 {
    prog.gapBlocks++
    if time.Duration(prog.gapBlocks)*d.blockDur > profile.MaxGap { prog.toneIdx = 0 }
}
```

While the expected tone's bin is above threshold the block counter climbs. The
tone is judged **at its falling edge** — the first block where the bin drops —
against `MinDuration` and `MaxDuration`. Pass, and the matcher moves to the B
tone's bin. Fail short and the whole sequence resets to tone 0 — a 100 ms blip
at the A frequency is far more likely speech than a page. Between tones the gap
counter runs; silence longer than `MaxGap` (200 ms; Quick Call II gaps are
"typically < 50 ms") also resets. When the last tone passes, `fire`
publishes and stamps `lastFiredAt`, and for the next `Cooldown` (30 s) the
profile is skipped entirely.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Timeline of a two-tone page through the tone-out matcher over 100-millisecond Goertzel blocks. The A-tone bin is above threshold for seven blocks and is judged at its falling edge, so the sequence advances. After a one-block gap the B-tone bin is above threshold for thirty blocks and judged at its falling edge; passing fires the tone alert and starts the cooldown. A note says a short tone or a long gap resets the sequence.">
  <text x="12" y="24" fill="currentColor" font-size="10" font-weight="bold">one Goertzel block = 100 ms (800 samples at 8 kHz)</text>
  <line x1="12" y1="90" x2="668" y2="90" stroke="var(--fg-muted)"/>
  <text x="12" y="82" fill="var(--fg-muted)" font-size="8">A-tone bin</text>
  <rect x="80" y="60" width="140" height="30" fill="none" stroke="var(--accent)"/>
  <text x="150" y="78" text-anchor="middle" fill="var(--accent)" font-size="9">7 blocks ≥ 0.05</text>
  <line x1="220" y1="56" x2="220" y2="96" stroke="currentColor" stroke-dasharray="2 2"/>
  <text x="228" y="70" fill="currentColor" font-size="8">falling edge: 250 ≤ 700 ms ≤ 1500 ✓ → toneIdx 1</text>
  <line x1="12" y1="160" x2="668" y2="160" stroke="var(--fg-muted)"/>
  <text x="12" y="152" fill="var(--fg-muted)" font-size="8">B-tone bin</text>
  <text x="230" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gap 1 block ≤ 200 ms</text>
  <rect x="240" y="130" width="300" height="30" fill="none" stroke="var(--accent)"/>
  <text x="390" y="148" text-anchor="middle" fill="var(--accent)" font-size="9">30 blocks ≥ 0.05 (3.0 s)</text>
  <line x1="540" y1="126" x2="540" y2="166" stroke="currentColor" stroke-dasharray="2 2"/>
  <text x="548" y="146" fill="currentColor" font-size="8" font-weight="bold">2.5 ≤ 3.0 s ≤ 5 ✓ fire → tone.alert</text>
  <text x="12" y="220" fill="var(--fg-muted)" font-size="9">judged at the falling edge in whole blocks; a short tone or long gap resets to tone 0</text>
</svg>
<figcaption>The matcher is a duration gate over 100 ms Goertzel blocks: each tone is judged when its bin drops, the gap between tones is bounded by max_gap, and a completed sequence fires once and then sleeps for the cooldown.</figcaption>
</figure>

## Which of the eleven places it needs — and the two it skips

[Part 1]({{ '/blog/deep-dives/beyond-voice-01-eleven-place-pattern/' | relative_url }})
laid out the pattern: front end, framer, bus event, storage, REST, panel,
config builder, field help, doctor, `config.example.yaml`, tests. Tone-out's
"front end" is the composer and its "framer" is a duration gate.

**Bus event.** `fire` publishes `events.KindToneAlert` — the string
`"tone.alert"` in `internal/events/bus.go` — with a `toneout.Alert` payload
(snake_case JSON tags from birth: `profile`, `alpha_tag`, `device_serial`,
`matched_at`, `frequencies_hz`), and logs:

```
INF toneout: profile matched profile=station-1-engine device=00000001 tones=[1042.2 1297.4]
```

The decoded-message log renders it as a `TONE-ALERT` row; the bus is
[Trunking Engine Part 2]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }})'s.

**Storage and REST read — skipped.** There is no `tone_log` table in
`internal/storage/sqlite.go`, no provider, no `GET` route, and nothing for the
retention sweeper's `decoderLogTables` to sweep. An alert is a moment, not a
record: `web/src/panels/Tones.tsx` filters the shared event store for
`e.kind === "tone.alert"` and renders profile, alpha tag, device and
frequencies newest-first over the SSE-fed store from
[Operator Cockpit Part 4]({{ '/blog/deep-dives/operator-cockpit-04-sse-to-react/' | relative_url }});
`internal/tui/panels/tones.go` is the terminal mirror, `KnownUITabs` lists
`"tones"` and `web/src/App.panels.test.tsx` mounts `/tones`.

**A write side instead of a read side.** What tone-out has that the message
decoders lack is a mutation: `POST /api/v1/devices/{serial}/tone-reset` calls
`ResetDevice`, clearing one device's match progress while keeping
`lastFiredAt` — "without throwing away the cooldown clock" — behind write mode
([Operator Cockpit Part 10]({{ '/blog/deep-dives/operator-cockpit-10-write-mode/' | relative_url }})).

**Config, builder, example, doctor.** `config.ToneOutConfig` holds
`Profiles []ToneProfileConfig`; `internal/configbuilder/sections.go` registers
a `tone_out` section rendered by `web/configbuilder/src/sections/ToneOut.tsx`;
`config.example.yaml` carries the per-profile schema and the
`station-1-engine` example. Doctor does *not* list tone-out among the decoders
that "need storage.path" — correctly. Nine places wired and two knowingly
absent is the honest shape of an alerting decoder, and in the
[opt-in feature matrix]({{ '/blog/deep-dives/running-it-for-real-08-opt-in-feature-matrix/' | relative_url }})
an empty `profiles` list builds no detector at all.

## Testing a page without a pager

`toneout_test.go` never touches a radio, a capture or a pager. `genTone`
synthesises an int16 sine at 0.6 of full scale ("to leave headroom"), the
tests drive `WritePCM` on a fake serial and collect bus events.
`TestDetectorMatchesTwoTone` feeds 700 ms of 1000 Hz, a 30 ms gap and 2 s of
1500 Hz and expects one alert with `FrequenciesHz == [1000 1500]`; the
`IgnoresWrongFrequency` and `IgnoresTooShortTone` tests expect none;
`TestDetectorPerDeviceIsolation` keeps `VOICE-1` and `VOICE-2` apart.

The cooldown test is the one to notice. `Options.Now` is injectable, so
`TestDetectorCooldownSuppressesRefires` advances a `fakeClock` two hours
between pages instead of sleeping — a second page inside a one-hour cooldown
is silent, the third fires. What the suite deliberately lacks is a recording
of a real dispatch page: vocoder and composer are tested against references
elsewhere
([Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})),
and tone-out's contract — "given PCM with these tones, fire" — a synthetic
sine exercises completely. Whether *your* agency's encoder lands above 0.05 in
the 10 Hz bins is answered by the log line, not a fixture.

### How Goertzel shaped the Go code

- **Block, not sliding, and the clock follows.** Because `Process` resets
  every `blockSize` samples, `blockDur` is the unit of time; durations and
  gaps are counted in blocks, never measured.
- **Normalise once so a threshold means one thing.** The `1/N²·4` scaling
  lets `MagnitudeThreshold` default to a constant across sample rates.
- **Share resonators, isolate progress.** Frequencies are indexed globally so
  profiles share filters; match state is per device serial.
- **Never fail the fan-out.** `WritePCM` returns nil unconditionally — an
  alerting sidecar must never cost a recording.

## Where this goes next

Tone-out told a station *that* it was paged; the next format tells it *why*.
[Part 6]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }})
returns to the IQ side for the paging networks proper — POCSAG's
`0x7CD215D8` sync codeword, BCH(31,21) with its trailing parity bit, the
frame-slot arithmetic that rebuilds a 21-bit RIC, and the FLEX decoder that
reuses the same BCH primitive bit-reversed for the one mode it decodes.

## FAQ

**How does GopherTrunk detect two-tone fire pager tones?**
A Goertzel resonator per configured frequency runs over the decoded 8 kHz PCM
in 100 ms blocks. A per-device matcher counts consecutive blocks above
`magnitude_threshold` (default 0.05), judges each tone at its falling edge
against `min_duration`/`max_duration`, bounds the gap with `max_gap`, and fires
`tone.alert` when the last tone passes.

**Why does the tone-out detector run on audio instead of IQ?**
Because a page can arrive over any voice path — analog FM dispatch or a
trunked digital talkgroup. Implementing `composer.PCMSink` and taking the
recorder's decoded PCM means one detector covers both, scoped per profile by
`system` and `group_id`, with no extra demodulator.

**What is the frequency resolution of the tone detector?**
`sample_rate / block_size` — 10 Hz at the 8 kHz, 800-sample defaults. Targets
round to the nearest exact bin, so a 1042.2 Hz tone is watched by the 1040 Hz
bin. `tolerance_hz` defaults to 15 Hz but is not consulted by the v1 matcher;
selectivity comes from the bin itself.

**Are tone alerts stored in the database or exposed over REST?**
No. Tone-out publishes `events.KindToneAlert` and logs `toneout: profile
matched`; the web and TUI Tones panels render the live event stream. There is
no `tone_log` table and no `GET` route — the only HTTP surface is the write
side, `POST /api/v1/devices/{serial}/tone-reset`.

**Why did my alert fire only once for two pages a few seconds apart?**
The profile's `cooldown` (default 30 s) suppresses re-fires after a match so a
long B tone or a repeated page does not raise duplicates. Shorten it for
agencies that page stations back-to-back — cooldowns are tracked per profile,
per device.

## Series navigation

**Part 5 of 14** · ←
[Part 4: FleetSync — Cloning the Template & the WAV That Lied]({{ '/blog/deep-dives/beyond-voice-04-fleetsync-wav-that-lied/' | relative_url }})
· Next →
[Part 6: POCSAG & FLEX — Paging Networks]({{ '/blog/deep-dives/beyond-voice-06-pocsag-flex-paging/' | relative_url }})
