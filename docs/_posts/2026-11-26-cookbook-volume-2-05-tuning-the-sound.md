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
balanced against, how loud each file ends up, which container it is written
in, why a P25 channel sits about 10 dB above an NFM one on the same scan
list, and how to A/B all of it offline from frames you already have. It
extends Volume 1's
[archival rig]({{ '/blog/tutorials/operator-cookbook-10-archival-rig/' | relative_url }}),
which decided where the audio is kept. This one decides how it sounds.*

> **TL;DR:** `recordings.voice_profile` picks a reference balance for the
> IMBE and AMBE+2 vocoders: the default `mbelib` (the dsd-neo / DSD-FME
> level: `unvoiced_gain` 5.49, enhance tilt 450 Hz) or `op25` (alias
> `trunk-recorder`: `unvoiced_gain` 1 and, with `recordings.enhance.enabled`,
> tilt off and the rumble high-pass at 100 Hz). A profile only fills knobs
> you left unset, so an explicit `tilt_hz` wins. `recordings.normalize`
> rewrites each finished file to −16 LUFS with one linear gain;
> `recordings.format: flac` halves the bytes; `audio_gain_db` on a scan-list
> channel balances NFM against P25, which is about 10 dB louder by default.
> Digital voice is always 8 kHz, whatever `recordings.sample_rate` says. A/B
> offline with `gophertrunk decode -voice-profile` and `-legacy-synthesis`.
> Measured and deliberately not changed: a +5 to +7 dB excess at 3–4 kHz
> against OP25, and whole-LDU loss at the head of some P25 replies.

**Key takeaways**

- **A voice profile is a choice between two references that disagree, not a
  quality setting.** mbelib-family decoders synthesise an unvoiced band at
  about 5.5 times a voiced harmonic of the same amplitude; OP25 sits at the
  spec's equal-power reading. GopherTrunk was measured against both on the
  same frames, and the profile says which one your ears are used to.
- **The profile fills defaults; anything you typed wins.** It touches only
  `unvoiced_gain`, `enhance.tilt_hz` and `enhance.hpf_hz`, only while they
  are still zero, and the tilt and high-pass half exists only when the
  enhance chain is on.
- **Loudness has three controls and they act in three places.** The vocoder
  [AGC]({{ '/reference/automatic-gain-control/' | relative_url }}) works per
  frame, `recordings.normalize` applies one gain per finished file, and
  `audio_gain_db` moves one scan-list channel, live and recorded alike.
- **Rule out lost IQ before you tune a sound knob.** A recording made across
  an IQ hole is announced as partial in the log, and no profile or
  sample-rate change repairs samples that never arrived.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Reference balance | mbelib (default) or op25 unvoiced level and low-frequency shaping | `recordings.voice_profile` |
| Unvoiced (fricative) level | 0 follows the profile (5.49), 1 is equal power, negative is the legacy scaling | `recordings.unvoiced_gain` |
| Enhance chain | tilt, rumble high-pass, shelf, low-pass, optional compressor, louder AGC target | `recordings.enhance.*` |
| File loudness | EBU R128, one linear gain per finished file | `recordings.normalize.*` |
| Container | lossless FLAC, about half the size for speech | `recordings.format: flac` |
| Mixed scan-list balance | gain in dB on one channel's audio, within ±30 | `scanner.conventional[].audio_gain_db` |
| Live rate | host speakers and web stream; digital voice is converted from 8 kHz | `audio.sample_rate`, `recordings.sample_rate` |
| Offline A/B | re-synthesise a `.raw` frame sidecar | `gophertrunk decode -voice-profile`, `-legacy-synthesis` |

## In this post

- **What you're building** — the chain from vocoder frames to a finished file, and where each knob sits.
- **The shopping list** — nothing to buy; a few calls and a pair of headphones.
- **The config** — one scan-list rig with every sound setting verified.
- **First run — what healthy looks like** — the summary line, the recorder lines, and the offline A/B.
- **What is measured, and what was left alone** — the figures behind the defaults, and two known gaps.
- **When it doesn't work** — symptom, cause, fix.

## What you're building

A rig whose audio is a set of decisions rather than whatever came out. The
example is a one-dongle scan list with an NFM channel, a conventional P25
channel and an AM channel, because that is the smallest rig on which every
knob in this part is audible. Everything under `recordings:` and `audio:`
applies unchanged to the trunked rigs of Volume 1.

Decoded voice passes through a short chain, and each setting belongs to one
stage of it. For the digital protocols the chain starts at the vocoder:
[IMBE]({{ '/reference/imbe/' | relative_url }}) for P25 Phase 1,
[AMBE+2]({{ '/reference/ambe-plus-2/' | relative_url }}) for P25 Phase 2,
DMR, NXDN and System Fusion. Synthesis is where the profile's first half
acts — the level of the unvoiced bands, the hiss of an "s" or an "f", against
the voiced harmonics
([Voice Coding Part 3]({{ '/blog/deep-dives/voice-coding-03-mbe-synthesis/' | relative_url }})
has the model). Next comes the opt-in enhance chain, in this order: a
first-order "radio tilt" high-pass, a rumble high-pass, a warmth shelf, a
band-limit low-pass and an optional compressor. Then the AGC drives the
frame toward a fixed peak: 18000 in int16 units by default, 22000 when
enhance is on. The stage-by-stage internals are
[Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }}).

Two stages follow that are not the vocoder's. A scan-list channel's
`audio_gain_db` is applied to the decoded PCM, so it reaches the file and
the live tap together. The finished file is then measured as a whole and
rewritten by `recordings.normalize` before the completed-call event is
published, so web playback, uploads and the transcription backend of
[Part 7]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }})
all read the normalised audio.

Analog channels skip the vocoder. Their FM or AM chain band-limits the
audio (the defaults are a 300 Hz high-pass and a 3400 Hz low-pass) and has
no AGC of its own, which is the root of the loudness gap this recipe
closes.

Status, plainly: the calibration figures below are measurements against
dsd-neo and trunk-recorder / OP25 decodes of the same recorded frames (four
sidecar pairs and one seven-reply P25 conversation). The live-rate
conversion and the 8 kHz digital rate are pinned by tests. The project's
engineering notes record measurements, not a listening panel, and no
operator confirmation of the `op25` profile or of `audio_gain_db`.

<figure class="lab-figure">
<svg viewBox="0 0 680 216" width="680" height="216" role="img" aria-label="Signal chain from decoded frames to finished audio: digital voice runs through MBE synthesis, where the voice profile sets the unvoiced level, then the opt-in enhance chain, the AGC and the per-channel audio gain, before splitting to a WAV or FLAC file that is loudness-normalised and to the live stream; the analog FM and AM chain joins at the audio gain with no AGC of its own">
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
| A few recorded calls | $0 | with `write_raw: true`, so the frames exist to re-decode |
| A second decoder (optional) | $0 | dsd-neo or DSD-FME, for an independent decode of the `.imb` / `.amb` sidecars |
| Headphones | $0 | measure with numbers first, then confirm by ear |

## The config

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"
  format: flac               # wav (default) | flac
  sample_rate: 8000          # analog FM/AM files only; digital is always 8 kHz
  write_raw: true            # keep the vocoder frames: the offline A/B needs them
  mbe_files: true            # .imb / .amb for dsd-neo or DSD-FME
  voice_profile: mbelib      # mbelib (default) | op25 (alias trunk-recorder)
  unvoiced_gain: 0           # 0 = follow the profile; 1 = equal power; <0 = legacy
  enhance:
    enabled: true            # opt-in; false = the faithful decode
    lpf_hz: 3400
    shelf_hz: 1500
    shelf_db: 2.0
    agc_target: 22000
    # hpf_hz and tilt_hz are deliberately absent: the profile fills them
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
      role: voice            # the conventional scanner drives a voice SDR
      gain: "auto"

scanner:
  conventional:
    - label: "County Fire Dispatch"
      frequency_hz: 154_190_000
      mode: nfm
      squelch_dbfs: -48
      talkgroup_id: 2147500001
      audio_gain_db: 10      # NFM arrives about 10 dB under P25
    - label: "VCALL10"
      frequency_hz: 155_752_500
      mode: p25
      squelch_dbfs: -50
      nac: "F7E"             # any NAC, but only P25 frames open the channel
      talkgroup_id: 2147500002
    - label: "Tower"
      frequency_hz: 118_700_000
      mode: am
      squelch_cn_db: 12
      talkgroup_id: 2147500003

audio:
  enabled: false             # true = also play on the host's speakers
  sample_rate: 8000          # speakers + web stream; 0 = recordings.sample_rate
  live_loudness: true        # AGC on the live network stream, digital voice only
```

Decisions worth understanding.

**The profile line, and the two lines that are missing.**
`config.example.yaml` prints `hpf_hz: 250` and `tilt_hz: 450` in its
`enhance` block. Copy those and a later switch to `voice_profile: op25`
changes only the unvoiced level, because the profile never overrides a
value you typed. Leave them out, as above, and the one word decides all
three: `mbelib` gives unvoiced 5.49, tilt 450 Hz and a 250 Hz rumble
high-pass; `op25` gives unvoiced 1, no tilt and a 100 Hz high-pass. With
`enhance.enabled: false` the profile changes the unvoiced level and nothing
else, since the raw decode carries no tilt to undo.

**Enhance is a trade, and it leans toward one reference.** Off, the output
is the faithful decode. On, it is band-limited, tilted and driven to a
louder AGC target, for the sound the DSD family produces. That chain is
matched to dsd-neo: on the P25 comparison against OP25, recordings made
with the chain on measured 4.4 dB of long-term log-spectral distance,
against 3.6 dB for the plain recorder-default decode of the same frames.
The chain is installed on the IMBE and AMBE+2 decoders; TETRA's ACELP has
neither it nor an unvoiced level.

**Normalisation is one gain per file, not a compressor.** The whole
recording is measured, a single linear gain moves it toward `target_lufs`,
and the gain is backed off if the true peak would pass `true_peak_dbtp`.
`max_boost_db` caps the gain so a near-silent call does not have its hiss
lifted to the target, and a file too short or too quiet to measure is left
untouched. A FLAC recording is rewritten as FLAC. `apply_to: distributed`
leaves the file on disk pristine and normalises only the uploaded MP3 copy.
[Recording, Composition & Streaming Part 8]({{ '/blog/deep-dives/recording-streaming-08-loudness-output-stage/' | relative_url }})
covers the measurement.

**`audio_gain_db` exists because analog and digital are levelled
differently.** The vocoder AGC targets a peak near −5
[dBFS]({{ '/reference/dbfs/' | relative_url }}). The analog chain has no
AGC and a fixed scale, so a full ±2.5 kHz NFM channel peaks near −20 dBFS.
By default P25 is therefore about 10 dB louder than NFM. The fix is a
per-channel knob rather than a new default because the opposite also
happens: 5 kHz-deviation amateur FM and compressed airband AM arrive
hotter. Normalisation evens out the files; `audio_gain_db` is what evens
out live listening, where `live_loudness` only touches digital voice.
Output is clamped at full scale, so a large boost on a loud channel clips.
The scanner side of this knob is
[Squelch Part 13]({{ '/blog/deep-dives/squelch-13-operator-controls/' | relative_url }}).

**Two rates, and only one of them is yours to set.** Decoded digital voice
is 8 kHz by the vocoder's contract. `recordings.sample_rate` sets the rate
of analog FM and AM files only. `audio.sample_rate` is the host speaker and
web stream rate, and live audio of either kind is resampled into it, so any
value from 4000 to 48000 plays at the right speed.

## First run — what healthy looks like

```bash
gophertrunk run -config config.yaml
```

The start-up summary states what the recorder resolved, in one line:

```text
INF daemon: config summary systems=0 sample_rate_hz=2400000 ... rec_dir=../recordings rec_enhance=true rec_normalize=true ... rec_voice_profile=mbelib rec_spec_amplitude_enhance=true
```

`rec_voice_profile` is the canonical name, so an empty key reads `mbelib`
and `trunk-recorder` reads `op25`. A P25 call on the scan list then shows
the container and the vocoder:

```text
INF recorder: call started device=00000001 wav=../recordings/.../2147500002/....flac tg=2147500002 provoice=false vocoder=imbe
INF recorder: call ended device=00000001 wav=... duration=... reason=...
```

Set `log.level: debug` for one session and each digital call also logs
`recorder: voice audio quality`, with `rms`, `crest`, `clip_pct` and
`mean_gain` among its fields. That line turns into a WARN,
`recorder: voice audio quality — output clipping (vocoder gain too hot)`,
when a call clips. The other recorder lines are read in
[The Field Notebook Part 10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }}).

Now the A/B. Each digital call left a `.raw` sidecar beside its audio, and
`decode` re-synthesises it without the radio:

```bash
gophertrunk decode -in call.raw -out a_mbelib.wav -vocoder imbe
gophertrunk decode -in call.raw -out b_op25.wav -vocoder imbe -voice-profile op25
gophertrunk decode -in call.raw -out c_legacy.wav -vocoder imbe -legacy-synthesis
gophertrunk decode -list-vocoders
```

```text
decoded ... frame(s) → a_mbelib.wav (... ms of audio)
voice stats: frames=... (voiced=... unvoiced=... silent=... idle_muted=... bad=... repeated=...)
  pitch: mean_f0=... Hz range=...-... Hz  mean_L=...  voiced_frac=...
  agc:   mean_gain=... range=...-...
  level: peak_preclip=... rms=... crest=... clip=...%
```

Use `-vocoder imbe` for P25 Phase 1, `ambe2` for P25 Phase 2, `ambe2-dmr`
for DMR, NXDN and System Fusion, and `tetra-acelp` for TETRA. Know what the
command does and does not apply. It applies the recorder's synthesis
defaults — the spec-faithful amplitude enhancement and the unvoiced level
the profile names — so files A and B differ in exactly one thing.
`-legacy-synthesis` turns both off, for the raw decoder. It does not run
the enhance chain or `audio_gain_db`, and it normalises only with
`-normalize`. To hear the tilt and high-pass half of a profile, change the
config and restart. In the stats, natural speech has a crest factor around
8 to 15; a crest near 3 or 4 with a non-zero `clip` is an over-hot AGC.

For a decode that shares no code with GopherTrunk, give the `.imb` or
`.amb` sidecar to dsd-neo or DSD-FME; the
[vocoders guide]({{ '/vocoders.html' | relative_url }}) has the commands,
and [voice calibration]({{ '/voice-calibration.html' | relative_url }})
the comparison harness.

## What is measured, and what was left alone

Every default above came from decoding the same frames with GopherTrunk and
with a reference decoder, then comparing spectra. The method is
[Voice Coding Part 12]({{ '/blog/deep-dives/voice-coding-12-calibration-testing/' | relative_url }}).

| Comparison on the same frames | Figure | What it settled |
|---|---|---|
| Unvoiced bands, before calibration, against dsd-neo | 6 dB low (IMBE) to 13–15 dB low (AMBE+2 2450) | the default `unvoiced_gain` of 5.49 |
| Low-frequency excess on voiced harmonics | +4 dB at 500 Hz rising to +16 dB at 125 Hz | the 450 Hz `tilt_hz` stage |
| Log-spectral distance to dsd-neo, 300–3400 Hz, P25 male | 8.3 dB down to 1.5 dB | the unvoiced calibration works |
| The same, DMR male | 4.9 dB down to 2.1 dB | and for AMBE+2 |
| Distance to OP25 as `unvoiced_gain` falls | 5.49 gives 3.57 dB, 2 gives 2.96, 1 gives 2.69, 0.5 gives 2.53 | OP25 sits at equal power: the `op25` profile |
| 3–4 kHz against OP25, relative to 1 kHz | +5 to +7 dB, at every gain | not changed |
| Length of seven P25 replies | 27.18 s against trunk-recorder's 30.42 s | not changed |

The first five rows explain the defaults. The sixth is why `op25` is a
profile and not the default: the two references disagree with each other by
several dB, and one conversation is not grounds for moving everyone.

The last two rows are the honest part. The 3–4 kHz excess does not move
with the unvoiced gain, so it sits in the high-harmonic amplitude
reconstruction, and the next instrument is a per-harmonic comparison
against OP25's vocoder. The knobs that reach that band, `enhance.lpf_hz`
and the shelf, shape it; they do not fix it. The missing length is whole
LDUs at the head of a reply, two (0.36 s) on two of the replies and one on
another, where OP25 catches the first frame more often. That is receiver
acquisition, not audio, and it needs a voice IQ capture to tune. No setting
in this post recovers a first syllable.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Switched to `voice_profile: op25` and the low end did not change | `tilt_hz` / `hpf_hz` are set explicitly, or `enhance.enabled` is false | Delete the two keys so the profile can fill them; confirm `rec_voice_profile=op25` in the summary line |
| Toggled enhance from the web Settings panel and `op25` sounds like `mbelib` | the live toggle switches the chain on with the values in the file; the profile's tilt and high-pass are resolved at start-up, and only when `enhance.enabled` is already true | Set both in the config and restart |
| Audio is sped up, or gapped, and a `recordings.sample_rate` change seemed to cure it | it cannot have: digital files are 8 kHz at any setting. Look for `recorder: recording is partial — IQ was lost during it, so the audio has gaps` | Fix the IQ loss upstream, then judge the sound |
| `recorder: forcing WAV rate to vocoder-native 8000; recordings.sample_rate applies to analog/NBFM only` | `recordings.sample_rate` is not 8000 and a digital call arrived | Expected. The file is right; the WARN is the reminder |
| NFM is much quieter than P25 on the scan list | analog has no AGC; the vocoder targets a fixed peak | `audio_gain_db` on the NFM channel, starting near 10 |
| WARN `output clipping (vocoder gain too hot)`, or harsh loud passages | `agc_target` or a channel boost is too high; output clamps at full scale | Lower `enhance.agc_target` toward 18000, or reduce `audio_gain_db` |
| Quiet calls stay quiet with normalise on | `max_boost_db` caps the gain; very short or very quiet files are skipped | Raise the cap with care: it lifts noise too |
| The A/B file sounds unlike the daemon's recording | `decode` runs no enhance chain, channel gain or normalisation | Compare against a recording made with enhance off, or add `-normalize` |
| An encrypted call records as loud noise | the vocoder is rendering ciphertext | `recordings.mute_encrypted: true`, or a key — [Part 8]({{ '/blog/tutorials/cookbook-volume-2-08-your-own-keys/' | relative_url }}) |

The third row is worth a paragraph, because it is the trap this whole area
sets. An operator reported that changing `recordings.sample_rate` from 8000
to 0 fixed sped-up FLAC audio. Digital FLAC is 8 kHz in its header and in
every frame at any setting, and a test pins that. Their own log had a
driver overrun with `host_drops=3873` immediately before the bad audio. On
P25 Phase 1 the voice chain says so itself, with a WARN that begins
`composer: p25p1 voice IQ has holes`. Read those lines first
([The Field Notebook Part 7]({{ '/blog/tutorials/field-notebook-07-overruns-host-drops/' | relative_url }})):
if IQ was lost, the audio is damaged before any knob in this post is
reached.

### How this recipe shapes operator practice

- **Change one stage at a time.** Profile, enhance, channel gain and
  normalisation stack; an A/B that moves two of them tells you nothing.
- **Keep the frames.** `write_raw: true` costs a few hundred bytes per
  second and makes every later sound decision reversible.
- **Measure, then listen.** Band energy, crest factor and loudness are
  numbers; "muffled" and "harsh" are where the search starts, not the
  result.

## Variations

- **The trunk-recorder balance.** `voice_profile: op25`, restart, done —
  provided `tilt_hz` and `hpf_hz` are unset as above.
- **Faithful archive, pleasant feed.** `enhance.enabled: false` and
  `normalize.apply_to: distributed`: the files on disk stay untouched and
  only the uploaded copies of
  [Volume 1 Part 9]({{ '/blog/tutorials/operator-cookbook-09-sharing-the-feed/' | relative_url }})
  are levelled.
- **A noisy room.** `enhance.compress.enabled: true` adds a gentle 2:1
  compressor above −18 dB (`threshold_db`, `ratio`, `attack_ms`,
  `release_ms`, `makeup_db`).
- **European analog.** `fm_deemphasis: eu` selects the 50 µs curve, and
  `fm_channel_bandwidth_hz: 12500` narrows the IF filter to a 12.5 kHz
  channel. Both shape analog channels only, such as the airband rig of
  [Part 1]({{ '/blog/tutorials/cookbook-volume-2-01-airband-rig/' | relative_url }}).
- **Re-master an archive.** `gophertrunk decode -normalize` turns old
  `.raw` sidecars into levelled WAVs with today's calibration.
- **DMR that still sounds synthetic.** Some of that is the codec itself;
  the [DMR voice quality]({{ '/dmr-voice-quality.html' | relative_url }})
  page records what was ruled out.

## Where this goes next

The calls now sound the way you chose. Most of them still do not need your
attention.
[Part 6]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }})
builds the `alerts:` section: rules that watch for an emergency talkgroup,
a tone-out match or one particular radio, and channels that push to a
phone, an MQTT broker or a script — behind a queue that a dead destination
cannot turn into a stalled decoder.

## FAQ

**Why is P25 louder than analog FM on my scan list?**
The vocoder's AGC drives decoded voice to a fixed peak near −5 dBFS, while
the analog chain has no AGC and a full-deviation NFM channel peaks near
−20 dBFS. By default P25 is about 10 dB louder. Set `audio_gain_db` on the
quieter channel; it applies to the recording, the live stream and uploads.

**What is the difference between the mbelib and op25 voice profiles?**
`mbelib`, the default, matches the dsd-neo / DSD-FME balance: unvoiced
bands at 5.49 and, with enhance on, a 450 Hz tilt. `op25` (alias
`trunk-recorder`) uses the equal-power unvoiced level of 1 and, with
enhance on, no tilt and a 100 Hz rumble high-pass. Either one only fills
settings you left unset.

**Does changing recordings.sample_rate fix digital audio that plays at the wrong speed?**
No. Decoded digital voice is always 8 kHz and the recorder forces that
rate, so the setting cannot change a digital file. Audio that sounds sped
up or gapped usually lost IQ upstream; look for the partial-recording and
dropped-IQ WARNs first.

**How do I re-decode a call with different vocoder settings?**
Record with `recordings.write_raw: true`, then run `gophertrunk decode -in
call.raw -out out.wav -vocoder imbe`, adding `-voice-profile op25` or
`-legacy-synthesis` for the alternatives. The command applies the vocoder
defaults only, not the enhance chain or channel gain.

**Is the voice calibration verified by ear?**
It is verified by measurement: the same frames decoded by GopherTrunk and
by dsd-neo or OP25, compared by spectrum and log-spectral distance. Two
gaps are documented and unfixed — a 3–4 kHz excess against OP25 and lost
frames at the head of some P25 replies — and neither is a profile setting.

## Series navigation

**Part 5 of 14** · ←
[Part 4: Fleet Signalling — MDC1200, FleetSync and Tone Gates]({{ '/blog/tutorials/cookbook-volume-2-04-fleet-signalling/' | relative_url }})
· Next →
[Part 6: Alerts — Tell Me When It Matters]({{ '/blog/tutorials/cookbook-volume-2-06-alerts/' | relative_url }})
