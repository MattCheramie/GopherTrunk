---
title: "The Operator's Cookbook Vol. 2, Part 5: Tuning the Sound — Voice Profiles, Loudness and FLAC"
description: "A complete GopherTrunk recipe for recordings that sound the way you want: the mbelib and op25 voice profiles and what each one sets, the opt-in enhance chain, EBU R128 loudness normalisation, lossless FLAC, per-channel audio_gain_db on a mixed scan list, the live-stream rates, and an offline A/B with gophertrunk decode."
category: tutorials
keywords: p25 scanner audio quality, imbe ambe vocoder sound, op25 vs dsd audio balance, scanner recording loudness normalization, flac scanner recordings, digital voice louder than analog fm, trunk-recorder audio balance, decode raw vocoder frames, sdr scanner voice profile, gophertrunk voice tuning
tags: [operator-cookbook, cookbook-volume-2, voice-profile, loudness, flac, recordings]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 5
---

*Part 5 of **The Operator's Cookbook, Volume 2**, a 14-part tutorial series
of complete, copy-paste GopherTrunk builds — one working rig per part —
built on what landed after Volume 1.
[Part 4]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
logged who keyed up on an analog channel. This part is about what a call
sounds like once it is decoded: which reference decoder the vocoder is
balanced against, how loud each file ends up, which container it is
written in, and why P25 sits about 10 dB above NFM on the same scan list.
It extends Volume 1's
[archival rig]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }}),
which decided where the audio is kept. This one decides how it sounds.*

> **TL;DR:** `recordings.voice_profile` picks a reference balance for the
> IMBE and AMBE+2 vocoders: the default `mbelib` (the dsd-neo / DSD-FME
> level: `unvoiced_gain` 5.49, enhance tilt 450 Hz) or `op25` (alias
> `trunk-recorder`: `unvoiced_gain` 1 and, with `recordings.enhance.enabled`,
> tilt off and the rumble high-pass at 100 Hz). A profile only fills knobs
> you left unset, so an explicit `tilt_hz` wins. `recordings.normalize`
> rewrites each finished file to −16 LUFS with one linear gain;
> `recordings.format: flac` about halves the bytes; `audio_gain_db` on a
> scan-list channel balances NFM against P25, which is about 10 dB louder
> by default. Digital voice is always 8 kHz, whatever
> `recordings.sample_rate` says. A/B offline with
> `gophertrunk decode -voice-profile` and `-legacy-synthesis`. Measured but
> not changed: a +5 to +7 dB excess at 3–4 kHz against OP25, and whole-LDU
> loss at the head of some P25 replies.

**Key takeaways**

- **A voice profile is a choice between two references that disagree, not a
  quality setting.** mbelib-family decoders synthesise an unvoiced band at
  about 5.5 times a voiced harmonic; OP25 sits at the spec's equal-power
  reading.
- **The profile fills defaults; anything you typed wins.** It touches only
  `unvoiced_gain`, `enhance.tilt_hz` and `enhance.hpf_hz`, and only while
  they are unset.
- **Loudness has three controls in three places.** The vocoder
  [AGC]({{ '/reference/automatic-gain-control/' | relative_url }}) works per
  frame, `recordings.normalize` applies one gain per finished file, and
  `audio_gain_db` moves one scan-list channel.
- **Rule out lost IQ before you tune a sound knob.** A recording made across
  an IQ hole is announced as partial, and no setting repairs samples that
  never arrived.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Reference balance | mbelib (default) or op25 unvoiced level and low-frequency shaping | `recordings.voice_profile` |
| Unvoiced (fricative) level | 0 follows the profile, 1 is equal power, negative is the legacy scaling | `recordings.unvoiced_gain` |
| Enhance chain | tilt, rumble high-pass, shelf, low-pass, optional compressor, louder AGC | `recordings.enhance.*` |
| File loudness | EBU R128, one linear gain per finished file | `recordings.normalize.*` |
| Container | lossless FLAC, about half the size for speech | `recordings.format: flac` |
| Mixed scan-list balance | gain in dB on one channel's audio, within ±30 | `scanner.conventional[].audio_gain_db` |
| Live rate | host speakers and web stream; digital voice is converted from 8 kHz | `audio.sample_rate`, `recordings.sample_rate` |
| Offline A/B | re-synthesise a raw frame sidecar | `gophertrunk decode -voice-profile`, `-legacy-synthesis` |

## In this post

- **What you're building** — the chain from frames to a finished file.
- **The config** — one scan-list rig, every sound setting verified.
- **First run — what healthy looks like** — the summary line and the offline A/B.
- **What is measured, and what was left alone** — the figures behind the defaults; two known gaps.
- **When it doesn't work** — symptom, cause, fix.

## What you're building

A rig whose audio is a set of decisions. The example is a one-dongle scan
list with an NFM, a conventional P25 and an AM channel: the smallest rig on
which every knob is audible. The `recordings:` and `audio:` blocks apply
unchanged to Volume 1's trunked rigs.

Each setting belongs to one stage of a short chain. Digital calls start at
the vocoder: [IMBE]({{ '/reference/imbe/' | relative_url }}) for P25
Phase 1, [AMBE+2]({{ '/reference/ambe-plus-2/' | relative_url }}) for P25
Phase 2, DMR, NXDN and System Fusion. Synthesis is where the profile's
first half acts: the level of the unvoiced bands, the hiss of an "s",
against the voiced harmonics
([Voice Coding Part 3]({{ '/blog/deep-dives/voice-coding-03-mbe-synthesis/' | relative_url }})).
Next is the opt-in enhance chain: a first-order "radio tilt" high-pass, a
rumble high-pass, a warmth shelf, a low-pass and an optional compressor.
Then the AGC drives each frame toward a fixed peak, 18000 in int16 units
by default and 22000 with enhance on
([Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }})).

A scan-list channel's `audio_gain_db` is then applied to the decoded PCM,
reaching the file and the live tap together, and the finished file is
measured whole and rewritten by `recordings.normalize`. Analog channels
skip the vocoder; their FM or AM chain has no AGC of its own, which is the
root of the loudness gap this recipe closes.

Status, plainly: the calibration figures below are measurements against
dsd-neo and trunk-recorder / OP25 decodes of the same frames, and the
live-rate conversion and 8 kHz digital rate are pinned by tests. The
project's engineering notes record no listening panel and no operator
confirmation of the `op25` profile or of `audio_gain_db`.

<figure class="lab-figure">
<svg viewBox="0 0 680 216" width="680" height="216" role="img" aria-label="Signal chain from decoded frames to finished audio: digital voice passes MBE synthesis, where the voice profile sets the unvoiced level, then the opt-in enhance chain, the AGC and the per-channel audio gain, and splits to a loudness-normalised WAV or FLAC file and to the live stream; the analog chain joins at the audio gain with no AGC">
  <text x="8" y="22" fill="var(--fg-muted)" font-size="9">digital voice — always 8 kHz</text>
  <rect x="8" y="32" width="82" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="49" y="49" text-anchor="middle" fill="currentColor" font-size="10">frames</text>
  <text x="49" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">.raw sidecar</text>
  <line x1="90" y1="52" x2="104" y2="52" stroke="currentColor"/>
  <rect x="104" y="32" width="118" height="40" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="163" y="49" text-anchor="middle" fill="var(--accent)" font-size="10">MBE synthesis</text>
  <text x="163" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">unvoiced_gain ← profile</text>
  <line x1="222" y1="52" x2="236" y2="52" stroke="currentColor"/>
  <rect x="236" y="32" width="126" height="40" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="299" y="49" text-anchor="middle" fill="var(--accent)" font-size="10">enhance (opt-in)</text>
  <text x="299" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">tilt · HPF · shelf · LPF</text>
  <line x1="362" y1="52" x2="376" y2="52" stroke="currentColor"/>
  <rect x="376" y="32" width="86" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="419" y="49" text-anchor="middle" fill="currentColor" font-size="10">AGC</text>
  <text x="419" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">18000 / 22000</text>
  <line x1="462" y1="52" x2="476" y2="52" stroke="currentColor"/>
  <rect x="476" y="32" width="88" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="520" y="49" text-anchor="middle" fill="currentColor" font-size="10">audio_gain_db</text>
  <text x="520" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">per channel</text>
  <line x1="564" y1="52" x2="580" y2="30" stroke="currentColor"/>
  <line x1="564" y1="52" x2="580" y2="88" stroke="currentColor"/>
  <rect x="580" y="10" width="94" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="627" y="27" text-anchor="middle" fill="currentColor" font-size="10">wav or flac</text>
  <text x="627" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="9">normalize: 1 gain</text>
  <rect x="580" y="68" width="94" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="627" y="85" text-anchor="middle" fill="currentColor" font-size="10">live stream</text>
  <text x="627" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">audio.sample_rate</text>
  <text x="104" y="130" fill="var(--fg-muted)" font-size="9">analog FM / AM — recordings.sample_rate</text>
  <rect x="104" y="138" width="258" height="40" rx="4" fill="none" stroke="currentColor"/>
  <text x="233" y="155" text-anchor="middle" fill="currentColor" font-size="10">FM / AM audio chain</text>
  <text x="233" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="9">band-limited voice · no AGC</text>
  <line x1="362" y1="158" x2="520" y2="158" stroke="currentColor"/>
  <line x1="520" y1="158" x2="520" y2="72" stroke="currentColor"/>
  <text x="530" y="150" fill="var(--fg-muted)" font-size="9">NFM: about 10 dB under P25</text>
  <text x="340" y="204" text-anchor="middle" fill="var(--fg-muted)" font-size="9">profile fills only unset knobs — mbelib: unvoiced 5.49, tilt 450 Hz · op25: unvoiced 1, tilt off, HPF 100 Hz</text>
</svg>
<figcaption>Every sound setting belongs to one stage: the profile acts at synthesis and in the enhance chain, the AGC per frame, audio_gain_db per channel, and normalisation once per finished file.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| A working rig | $0 | any rig from Volume 1 or Parts 1–4; the config below needs one `role: voice` dongle |
| A second decoder (optional) | $0 | dsd-neo or DSD-FME, for an independent decode of the `.imb` / `.amb` sidecars |

## The config

```yaml
storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"
  format: flac               # wav (default) | flac
  sample_rate: 8000          # analog files only; digital is always 8 kHz
  write_raw: true            # keep the frames: the offline A/B needs them
  mbe_files: true            # .imb / .amb for dsd-neo or DSD-FME
  voice_profile: mbelib      # mbelib (default) | op25
  unvoiced_gain: 0           # 0 = follow the profile
  enhance:
    enabled: true            # false = the faithful decode
    agc_target: 22000
    # hpf_hz and tilt_hz deliberately absent: the profile fills them
  normalize:
    enabled: true
    apply_to: recording      # recording | distributed | both
    target_lufs: -16.0
    true_peak_dbtp: -1.5
    max_boost_db: 12.0

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"     # from `gophertrunk sdr list`
      role: voice            # the scan list drives a voice SDR
      gain: "auto"

scanner:
  conventional:
    - label: "County Fire Dispatch"
      frequency_hz: 154_190_000
      mode: nfm
      squelch_dbfs: -48
      talkgroup_id: 2147500001
      audio_gain_db: 10      # NFM sits about 10 dB under P25
    - label: "VCALL10"
      frequency_hz: 155_752_500
      mode: p25
      squelch_dbfs: -50
      nac: "F7E"             # any NAC, P25 frames only
      talkgroup_id: 2147500002
    - label: "Tower"
      frequency_hz: 118_700_000
      mode: am
      squelch_cn_db: 12
      talkgroup_id: 2147500003

audio:
  enabled: false             # true = also play on the host's speakers
  sample_rate: 8000          # speakers + web stream rate
  live_loudness: true        # AGC on the live stream, digital voice only
```

**The profile line, and the two lines that are missing.**
`config.example.yaml` prints `hpf_hz: 250` and `tilt_hz: 450` in its
`enhance` block. Copy those and a later switch to `voice_profile: op25`
changes only the unvoiced level, because a profile never overrides a value
you typed. Leave them out and the one word decides all three: `mbelib`
gives unvoiced 5.49, tilt 450 Hz and a 250 Hz rumble high-pass; `op25`
(alias `trunk-recorder`) gives unvoiced 1, no tilt and a 100 Hz high-pass.
With enhance off, a profile changes the unvoiced level and nothing else.

**Enhance is a trade, and it leans toward one reference.** Off, the output
is the faithful decode. On, it is band-limited, tilted and louder, matched
to dsd-neo: against OP25, recordings made with it on measured 4.4 dB of
log-spectral distance, against 3.6 dB for the plain decode of the same
frames. It exists on the IMBE and AMBE+2 decoders only; TETRA's ACELP has
neither it nor an unvoiced level.

**Normalisation is one gain per file, not a compressor.** The recording is
measured, one linear gain moves it toward `target_lufs`, and the gain
backs off if the true peak would pass `true_peak_dbtp`. `max_boost_db`
caps it so a near-silent call does not have its hiss lifted. FLAC is
rewritten as FLAC, and `apply_to: distributed` normalises only the
uploaded copy
([Recording, Composition & Streaming Part 8]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }})).

**`audio_gain_db` exists because analog and digital are levelled
differently.** The vocoder AGC targets a peak near −5
[dBFS]({{ '/reference/dbfs/' | relative_url }}); the analog chain has a
fixed scale, so a full ±2.5 kHz NFM channel peaks near −20 dBFS. It is a
per-channel knob rather than a new default because the opposite also
happens: 5 kHz-deviation amateur FM and compressed airband AM arrive
hotter. Normalisation evens out the files; this knob evens out live
listening too. Output clamps at full scale, so a large boost clips
([Squelch Part 13]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }})).

**FLAC costs nothing downstream.** It is lossless and about half the size
for speech. Playback, normalisation, uploads, the retention sweep and
transcription identify the container by its content, not its extension, so
an archive can mix both
([Volume 1 Part 10]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }})
has the disk budget).

**Two rates, and only one is yours to set.** Decoded digital voice is
8 kHz by the vocoder's contract; `recordings.sample_rate` sets analog
files only. `audio.sample_rate` is the speaker and web stream rate,
and live audio is resampled into it, so any value from 4000 to 48000 plays
at the right speed.

## First run — what healthy looks like

```bash
gophertrunk run -config config.yaml
```

The start-up summary states what the recorder resolved:

```text
INF daemon: config summary systems=0 sample_rate_hz=2400000 ... rec_enhance=true rec_normalize=true ... rec_voice_profile=mbelib rec_spec_amplitude_enhance=true
```

`rec_voice_profile` is the canonical name: an empty key reads `mbelib`,
`trunk-recorder` reads `op25`. A P25 call then shows the container and the
vocoder:

```text
INF recorder: call started device=00000001 wav=.../2147500002/....flac tg=2147500002 provoice=false vocoder=imbe
INF recorder: call ended device=00000001 wav=... duration=... reason=...
```

Now the A/B. Each digital call left a `.raw` sidecar, and `decode`
re-synthesises it:

```bash
gophertrunk decode -in call.raw -out a_mbelib.wav -vocoder imbe
gophertrunk decode -in call.raw -out b_op25.wav -vocoder imbe -voice-profile op25
gophertrunk decode -in call.raw -out c_legacy.wav -vocoder imbe -legacy-synthesis
```

```text
decoded ... frame(s) → a_mbelib.wav (... ms of audio)
voice stats: frames=... (voiced=... unvoiced=... silent=... idle_muted=... bad=... repeated=...)
  level: peak_preclip=... rms=... crest=... clip=...%
```

Use `-vocoder imbe` for P25 Phase 1, `ambe2-dmr` for P25 Phase 2, DMR, NXDN
and System Fusion, `ambe2` for D-STAR only, `tetra-acelp` for TETRA;
`-list-vocoders` prints them. The two AMBE table sets are not
interchangeable: a Phase 2, DMR or NXDN file decoded with `ambe2` comes out
as speech-shaped noise. The command applies the recorder's
synthesis defaults — the spec-faithful amplitude enhancement and the
unvoiced level the profile names — so files A and B differ in exactly one
thing, and `-legacy-synthesis` turns both off. It does not run the enhance
chain or `audio_gain_db`, and normalises only with `-normalize`; to hear a
profile's tilt and high-pass half, change the config and restart. In the
stats, natural speech has a crest factor around 8 to 15; a crest near 3 or
4 with non-zero `clip` is an over-hot AGC.

## What is measured, and what was left alone

Every default came from decoding the same frames with GopherTrunk and with
a reference decoder, then comparing spectra
([Voice Coding Part 12]({{ '/blog/deep-dives/voice-coding-12-calibration-testing/' | relative_url }})).
The `.imb` and `.amb` sidecars go straight into dsd-neo or DSD-FME, as
the [vocoders guide]({{ '/vocoders.html' | relative_url }}) and
[voice calibration]({{ '/voice-calibration.html' | relative_url }}) pages
show.

| Comparison on the same frames | Figure | What it settled |
|---|---|---|
| Unvoiced bands before calibration, against dsd-neo | 6 dB low (IMBE) to 13–15 dB low (AMBE+2 2450) | the default `unvoiced_gain` of 5.49 |
| Low-frequency excess on voiced harmonics | +4 dB at 500 Hz rising to +16 dB at 125 Hz | the 450 Hz `tilt_hz` stage |
| Log-spectral distance to dsd-neo, 300–3400 Hz | 8.3 down to 1.5 dB (P25 male), 4.9 down to 2.1 dB (DMR male) | the unvoiced calibration works |
| Distance to OP25 as the unvoiced gain falls | 5.49 gives 3.57 dB, 2 gives 2.96, 1 gives 2.69, 0.5 gives 2.53 | OP25 sits at equal power: the `op25` profile |
| 3–4 kHz against OP25, relative to 1 kHz | +5 to +7 dB, at every gain | not changed |
| Length of seven P25 replies | 27.18 s against trunk-recorder's 30.42 s | not changed |

The fourth row is why `op25` is a profile and not the default: the
references disagree by several dB, and one conversation is not grounds for
moving everyone. The last two rows are the honest part. The 3–4 kHz excess
does not move with the unvoiced gain, so it sits in the high-harmonic
amplitude reconstruction; `enhance.lpf_hz` and the shelf shape that band
but do not fix it. The missing length is whole LDUs at the head of a
reply — two (0.36 s) on two of the replies, one on another. That is
receiver acquisition, and no setting here recovers a first syllable.

## When it doesn't work

Two log lines to search for before touching a sound setting:

```text
WRN recorder: recording is partial — IQ was lost during it, so the audio has gaps
WRN recorder: voice audio quality — output clipping (vocoder gain too hot)
```

| Symptom | Likely cause | Fix |
|---|---|---|
| Switched the profile to op25 and the low end did not change | the tilt and high-pass keys are set explicitly, enhance is off, or the key is misspelled (the loader ignores a key it does not know and names it in a start-up WARN) | Delete `tilt_hz` and `hpf_hz` so the profile can fill them; confirm `rec_voice_profile=op25` in the summary line |
| Enhance was switched on live and op25 sounds like mbelib | the live toggle uses the file's values; a profile's tilt and high-pass are resolved at start-up, and only if enhance is already enabled there | Set `enhance.enabled: true` in the config and restart, instead of `recordings_enhance_enabled` on `PATCH /api/v1/settings` |
| Audio is sped up or gapped, and a sample-rate change seemed to cure it | it cannot have: digital files are 8 kHz at any setting. The first WARN above is the usual cause | Fix the IQ loss upstream, then judge the sound |
| The second WARN above, or harsh loud passages | the AGC target or a channel boost is too high | Lower `enhance.agc_target` toward 18000, or reduce `audio_gain_db` |
| An encrypted call records as loud noise | the vocoder is rendering ciphertext | `recordings.mute_encrypted: true`, or a key ([Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }})) |

The third row is the trap this area sets. An operator reported that
changing `recordings.sample_rate` from 8000 to 0 fixed sped-up FLAC audio.
Digital FLAC is 8 kHz at any setting, and a test pins that; their own log
had a driver overrun with `host_drops=3873` just before the bad audio. On
P25 Phase 1 the voice chain says so itself, in a WARN that begins
`composer: p25p1 voice IQ has holes`
([The Field Notebook Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})).

### How this recipe shapes operator practice

- **Change one stage at a time.** Profile, enhance, channel gain and
  normalisation stack; an A/B that moves two tells you nothing.
- **Keep the frames.** `write_raw: true` costs a few hundred bytes per
  second and makes every sound decision reversible.
- **Measure, then listen.** Band energy, crest factor and loudness are
  numbers; "muffled" is where a search starts.

## Variations

- **Faithful archive, pleasant feed.** `enhance.enabled: false` and
  `normalize.apply_to: distributed`: files on disk stay untouched and only
  the uploads of
  [Volume 1 Part 9]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
  are levelled. `enhance.compress.enabled: true` adds a 2:1 compressor for
  a noisy room.
- **European analog FM.** `fm_deemphasis: eu` selects the 50 µs curve and
  `fm_channel_bandwidth_hz: 12500` a 12.5 kHz IF filter. Digital voice and
  AM (the airband rig of
  [Part 1]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }}))
  are unaffected.
- **DMR that still sounds synthetic.** The
  [DMR voice quality]({{ '/dmr-voice-quality.html' | relative_url }}) page
  records what was ruled out.

## Where this goes next

The calls now sound the way you chose; most still do not need your
attention.
[Part 6]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }})
builds the `alerts:` section: rules for an emergency talkgroup, a tone-out
match or one particular radio, pushed to a phone, an MQTT broker or a
script.

## FAQ

**Why is P25 louder than analog FM on my scan list?**
The vocoder's AGC drives decoded voice to a peak near −5 dBFS, while the
analog chain has no AGC and a full-deviation NFM channel peaks near
−20 dBFS. Set `audio_gain_db` on the quieter channel; it applies to the
recording, the live stream and uploads.

**What is the difference between the mbelib and op25 voice profiles?**
`mbelib`, the default, matches the dsd-neo / DSD-FME balance: unvoiced
bands at 5.49 and, with enhance on, a 450 Hz tilt. `op25` uses the
equal-power level of 1 and, with enhance on, no tilt and a 100 Hz
high-pass. Either only fills settings you left unset.

**Does changing recordings.sample_rate fix digital audio that plays at the wrong speed?**
No. Decoded digital voice is always 8 kHz and the recorder forces that
rate. Audio that sounds sped up or gapped usually lost IQ upstream; look
for the partial-recording WARN.

**Is the voice calibration verified by ear?**
By measurement: the same frames decoded by GopherTrunk and by dsd-neo or
OP25, compared by spectrum. Two gaps are documented and unfixed: a
3–4 kHz excess against OP25 and lost frames at the head of some P25
replies.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Fleet Signalling — MDC1200, FleetSync and Tone Gates]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
· Next →
[Part 6: Alerts — Tell Me When It Matters]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }})
