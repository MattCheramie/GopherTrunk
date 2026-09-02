---
title: "The Operator's Cookbook, Part 5: TETRA Direct Mode — Scanning DMO"
description: A complete GopherTrunk recipe for TETRA DMO — radio-to-radio direct mode with no infrastructure. One dongle camped on a simplex frequency, the tetra-dmo protocol, colour-code auto-recovery, the tetra_mcc/tetra_mnc keys an MNI≠0 network requires, and an honest account of what is still on-air-gated.
category: tutorials
keywords: tetra dmo scanner, tetra direct mode decoder, decode tetra dmo sdr, tetra dmo colour code, tetra mcc mnc config, tetra dmo 430 mhz, tetra walkie talkie decode, sdr tetra decoder, gophertrunk cookbook
tags: [operator-cookbook, tetra, dmo, direct-mode, config, sdr]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook"
series_part: 5
---

*Part 5 of **The Operator's Cookbook**, a 14-part series of complete,
copy-paste GopherTrunk builds — one working rig per part, antenna to browser.
[Part 4]({{ '/blog/tutorials/operator-cookbook-04-tetra-tmo/' | relative_url }})
built a TETRA TMO rig camped on a trunked network's continuous control channel.
This part points the same hardware at the opposite animal:
**Direct Mode Operation** — TETRA radios talking straight to each other, no
base station, no control channel, nothing on the air between transmissions.
It's GopherTrunk's newest decode path, it needs the least hardware of any
recipe here, and it carries the series' most honest caveat list.*

> **TL;DR:** One dongle camped on one simplex UHF frequency decodes
> [TETRA DMO]({{ '/reference/tetra-dmo/' | relative_url }}) end to end:
> `protocol: tetra-dmo` in the system block, the DMO frequency in
> `control_channels`, and **no voice SDR at all** — DMO voice rides the same
> carrier the pipeline decodes, so a single same-carrier tap feeds the ACELP
> vocoder. The traffic descrambler's colour code is **auto-recovered off the
> air** (~20 traffic bursts), but on a network with a non-zero MNI you must
> set `tetra_mcc` / `tetra_mnc` or every colour candidate sits at the chance
> floor. Watch for `tetra dmo cc locked`, then
> `tetra dmo grant (traffic detected)`. Recordings file under **talkgroup 0**
> — DMO grants carry none. Full on-air verification is still open in
> [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003).

**Key takeaways**

- **This is the one-SDR, zero-infrastructure recipe.** No control channel, no
  grants from a network, no voice pool — the pipeline locks the sync bursts,
  detects traffic on a slot grid, and decodes voice off the very same carrier.
- **The colour code is the whole ballgame.** TETRA scrambles traffic with a
  30-bit [extended colour code]({{ '/reference/tetra-extended-colour-code/' | relative_url }});
  GopherTrunk recovers the 6-bit colour by brute force, but the MNI half
  (MCC/MNC) is **not on the air** — on an MNI≠0 network it must come from
  your config.
- **`dnb_qualified` means traffic; `dnb_total` is a noise meter.** The raw
  burst correlator false-fires ~18 times a second on an idle channel by
  design math. A large gap between the two counters is normal, not a fault.
- **Green synthetic ≠ on-air correct.** The DMO path has already produced —
  and retracted — one wrong verdict ("it's encrypted"; it was a descramble
  skip). The remaining gate is operator captures, and this post tells you how
  to contribute one.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Protocol selector | direct-mode pipeline, 144 kHz channel rate | `trunking.systems[].protocol: tetra-dmo` (also accepts `dmo`) |
| The frequency | simplex channel to camp on | `control_channels: [438_900_000]` |
| Network identity | MNI folded into every colour candidate | `tetra_mcc`, `tetra_mnc` ([MNI]({{ '/reference/tetra-mobile-network-identity/' | relative_url }})) |
| Manual colour override | skip auto-recovery entirely | `tetra_colour_code` (leave 0 to auto-recover) |
| Voice decode | same-carrier tap → TCH/S → ACELP | automatic — no `role: voice` device needed |
| Where TETRA even is | region/legality background | [TETRA scanner guide]({{ '/tetra-scanner/' | relative_url }}) |
| The full story | how this path was built, bug by bug | [TETRA End to End 11–13]({{ '/blog/deep-dives/tetra-end-to-end-11-dmo-direct-mode/' | relative_url }}) |

## In this post

- **What you're building** — a camped ear on a radio-to-radio channel.
- **The shopping list** — Part 4's hardware, unchanged.
- **The config** — three blocks, one protocol string, two identity keys.
- **First run — what healthy looks like** — camping, lock, grant, colour, voice.
- **When it doesn't work** — symptom → cause → fix, with the MNI trap up top.
- **Variations & the open gate** — wideband hosting, and how to close #1003.

## What you're building

Everything in Parts 1–4 decoded *infrastructure*: a tower transmitting
continuously, voice channels handed out by grants. DMO has none of that.
Two TETRA handhelds on a construction site or an event crew — one radio keys
up, transmits directly on a simplex frequency, and stops. Between
transmissions the channel is **pure noise floor**.

That changes the decoder's whole posture. So GopherTrunk **camps**: the hunt
supervisor parks on the frequency without demanding a lock, the pipeline
holds its lock stickily across silence, and traffic detection is deliberately
skeptical — a transmission is declared only when bursts line up on the
255-symbol TETRA slot grid, because the raw correlator alone false-fires on
noise ~18 times per second. The
[three]({{ '/blog/deep-dives/tetra-end-to-end-11-dmo-direct-mode/' | relative_url }})
[DMO]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }})
[deep dives]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }})
tell that story properly; this recipe cooks with the result.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Signal chain of the TETRA DMO rig: two handhelds share one simplex carrier; one SDR feeds a 144 kilohertz down-converter, the DMO pipeline locks sync bursts and grid-qualifies traffic into a grant, and a same-carrier voice tap recovers the colour code and decodes ACELP voice into recordings filed under talkgroup zero">
  <rect x="8" y="30" width="54" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="35" y="49" text-anchor="middle" fill="currentColor" font-size="10">radio A</text>
  <rect x="8" y="76" width="54" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="35" y="95" text-anchor="middle" fill="currentColor" font-size="10">radio B</text>
  <line x1="62" y1="45" x2="100" y2="64" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="62" y1="91" x2="100" y2="72" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="82" y="120" fill="var(--fg-muted)" font-size="9">one simplex carrier,</text>
  <text x="82" y="132" fill="var(--fg-muted)" font-size="9">silent between PTTs</text>
  <rect x="104" y="52" width="74" height="32" rx="4" fill="none" stroke="currentColor"/>
  <text x="141" y="72" text-anchor="middle" fill="currentColor" font-size="10">SDR dongle</text>
  <line x1="178" y1="68" x2="212" y2="68" stroke="currentColor"/>
  <rect x="212" y="14" width="230" height="216" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="327" y="30" text-anchor="middle" fill="var(--fg-muted)" font-size="10">GopherTrunk (protocol: tetra-dmo)</text>
  <rect x="226" y="42" width="202" height="30" rx="4" fill="none" stroke="currentColor"/>
  <text x="327" y="61" text-anchor="middle" fill="currentColor" font-size="10">DDC → 144 kHz channel stream</text>
  <rect x="226" y="86" width="202" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="327" y="100" text-anchor="middle" fill="var(--accent)" font-size="10">DSB sync decode → sticky lock</text>
  <text x="327" y="113" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SCH/S + SYNC PDU, frame counter</text>
  <rect x="226" y="134" width="202" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="327" y="148" text-anchor="middle" fill="var(--accent)" font-size="10">DNB slot-grid vote → grant</text>
  <text x="327" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dnb_qualified, not dnb_total</text>
  <rect x="226" y="182" width="202" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="327" y="196" text-anchor="middle" fill="currentColor" font-size="10">same-carrier voice tap</text>
  <text x="327" y="209" text-anchor="middle" fill="var(--fg-muted)" font-size="9">colour recovery → TCH/S → ACELP</text>
  <line x1="327" y1="72" x2="327" y2="86" stroke="currentColor"/>
  <line x1="327" y1="120" x2="327" y2="134" stroke="currentColor"/>
  <line x1="327" y1="168" x2="327" y2="182" stroke="currentColor"/>
  <line x1="428" y1="199" x2="480" y2="199" stroke="currentColor"/>
  <rect x="480" y="150" width="190" height="34" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="575" y="164" text-anchor="middle" fill="var(--accent)" font-size="10">web console: Active / History</text>
  <text x="575" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="9">calls appear under group 0</text>
  <rect x="480" y="196" width="190" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="575" y="210" text-anchor="middle" fill="currentColor" font-size="10">recordings/&lt;system&gt;/0/*.wav</text>
  <text x="575" y="223" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DMO grants carry no talkgroup</text>
  <line x1="442" y1="160" x2="480" y2="162" stroke="var(--accent)"/>
</svg>
<figcaption>One dongle, one carrier, no infrastructure: the DMO pipeline locks the sync bursts, votes traffic onto the slot grid, and decodes voice from the same tap — recordings land under talkgroup 0 because direct mode never announces one.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| RTL-SDR Blog V3/V4 (or your Part 4 dongle) | ~$35 | a [TCXO]({{ '/reference/tcxo/' | relative_url }}) matters more here — DMO channels are 25 kHz and often up at 430–470 MHz |
| UHF-capable antenna | $0–$30 | the kit whip extended for 70 cm works; a tuned [whip]({{ '/reference/whip-antenna/' | relative_url }}) buys margin |
| Computer | $0 | anything that ran Parts 1–4 |

Nothing new — the cheapest recipe in the series. You also need one fact you
can't buy: the **DMO frequency** your target radios use. There's no control
channel to hunt, so it comes from a codeplug, a licence record, or a
band-scope sweep (the captures this path was built on live at 438.9 MHz).

## The config

```yaml
log:
  level: info        # set debug to see the DMO decode-status counters

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"        # from `gophertrunk sdr list`
      role: control
      gain: "auto"

trunking:
  systems:
    - name: "Site-DMO"
      protocol: tetra-dmo       # "dmo" is accepted too
      control_channels:
        - 438_900_000           # the simplex DMO frequency
      tetra_mcc: 250            # your network's MCC — see below
      tetra_mnc: 1              # your network's MNC — see below
```

Three things to understand before running it.

**There is no voice device — on purpose.** DMO voice is transmitted *on the
carrier you're already decoding* (there is no separate traffic channel), so
GopherTrunk allocates a same-carrier tap automatically and the composer runs
its DMO voice chain on it. Adding a voice SDR here does nothing.

**`tetra_mcc` / `tetra_mnc` are load-bearing on real networks.** TETRA seeds
its traffic scrambler with the full 30-bit extended colour code —
MCC + MNC + 6-bit colour. The DMO sync burst is *always* colour-0 scrambled
and carries MNI 0 on the air, so the network's real
[MNI]({{ '/reference/tetra-mobile-network-identity/' | relative_url }}) is
not recoverable by listening — it has to come from you. Learned the hard
way: a reporter's Motorola MTP8500Ex radios ran MCC 250 / MNC 1, the colour
search assumed MNI 0, and all 64 candidates sat at the chance floor while
signalling decoded perfectly. True out-of-the-box MNI-0 direct mode keeps
the zero defaults; anything with a codeplug, get its MCC/MNC.

**Leave `tetra_colour_code` alone.** The 6-bit DM colour is recovered
automatically by decoding traffic under all 64 candidates and requiring a
dominant winner — on a real capture the correct colour won ~35 CRC-valid
frames against a runner-up of ≤3. Set the key only to skip the ~20-burst
recovery delay on a colour you already know.

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

A silent DMO channel is the *normal* startup state, and the log says so
explicitly instead of alarming:

```
INF cchunt: camped on conventional channel — idle, waiting for traffic system=Site-DMO
```

That line fires once (issue #1036 made it transition-only, so a quiet night
doesn't spam the log). Now wait for someone to key up. The first PTT produces
a burst of activity in order:

```
INF tetra dmo cc locked freq=438900000 system=Site-DMO
INF tetra dmo grant (traffic detected) freq=438900000 colour=0 system=Site-DMO
INF composer: tetra DMO voice follow started — DNB TCH/S decode + ACELP vocoder serial=cc:same-carrier:1 colour_hint=0 rate_hz=18000
INF composer: tetra DMO colour code recovered serial=cc:same-carrier:1 colour=3 attempt=1
INF recorder: call started device=cc:same-carrier:1 wav=../recordings/Site-DMO/0/... tg=0 provoice=false vocoder=tetra-acelp
```

Read the timing honestly: the grant lands **about half a second into the
transmission** (the slot-grid latch plus four qualified bursts), and colour
recovery needs roughly **20 traffic bursts** — but the voice chain buffers
everything from the grant onward and decodes it retroactively once the
colour is known, so leading speech isn't lost. The call ends on
`voice_hangtime_ms` after the last decoded voice — DMO carries no release
message GopherTrunk decodes, so a timeout-flavoured ending is normal.

Note the `tg=0`: **every DMO recording files under talkgroup 0**. A DMO
grant carries no talkgroup, so History and the `recordings/Site-DMO/0/`
folder collect everything under group zero. Name it in a talkgroup file if
the raw `0` bothers you.

With `log.level: debug`, a periodic status line shows the pipeline's
internals:

```
DBG tetra dmo: decode status system=Site-DMO locked=true carrier_off_hz=-412.5 dsb_total=54 dsb_schs_crc=46 dnb_total=4541 dnb_qualified=837 tch_crc=203 distinct_fn=17 colour=3 colour_known=true grant_active=true
```

The counter pair to internalize: **`dnb_qualified` is traffic,
`dnb_total` is a noise meter.** The raw correlator is loose enough that
noise trips it ~18 times a second on a dead-silent channel — arithmetic, not
a bug; only bursts landing on the learned slot grid qualify. Thousands of
`dnb_total` next to hundreds of `dnb_qualified` is a healthy channel. An
early version that trusted raw detections granted 230 ms of noise on an
empty channel — which is exactly why the qualified counter exists.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Locks, `dsb_schs_crc` climbs, but `tch_crc` stays near zero and `colour_known=false` forever | **wrong MNI** — the colour search can never reach the real scrambler seed | Set `tetra_mcc`/`tetra_mnc` from the codeplug. This signature (several colours rising modestly, none dominant) cost weeks before the MNI-0 blind spot was found — [the descramble saga]({{ '/blog/deep-dives/tetra-end-to-end-12-dmo-descramble-colour/' | relative_url }}) has the story |
| `colour_known=false` on short PTTs only | recovery needs ~20 qualified bursts; a 2-second PTT ends first | Nothing is wrong — the next longer transmission recovers it, and recovery re-arms per transmission |
| `dnb_total` climbing constantly, no grants, no lock | noise doing what noise does | Expected on an idle channel. If radios ARE transmitting and there's no `tetra dmo cc locked`, treat it as RF: gain, antenna, exact frequency |
| Decodes, but audio garbled or thin | marginal signal — handhelds at range are a weak-signal regime | The blind equalizer is already on. Better antenna first ([The Analog Edge]({{ '/blog/tutorials/analog-edge-07-antennas/' | relative_url }})), then contribute a capture (below) |
| "It must be encrypted" | maybe — but this path was burned by that verdict once | GopherTrunk once called a clear TEA0 capture "encrypted"; the real bug was a skipped colour-0 descramble. Confirm from the codeplug before concluding encryption — a chance-floor decode on a known-clear channel is a defect worth reporting |
| Calls end at odd times / `reason=timeout` | DMO decodes no release PDU | Normal — hangtime is the only end signal; tune `voice_hangtime_ms` |

## Variations & the open gate

- **Host it on a wideband dongle.** A `role: wideband` device can carry the
  DMO channel as a `channels:` entry pointing at the `tetra-dmo` system,
  alongside other protocols' taps — same pipeline, shared hardware
  (see [Part 7]({{ '/blog/tutorials/operator-cookbook-07-multi-system-pool/' | relative_url }})).
- **Pin the colour.** On your own known radios, set `tetra_colour_code` and
  skip recovery — first-PTT decode with no 20-burst warm-up.
- **Capture for posterity (and for #1003).** The honest part: the
  full-daemon DMO path is offline-verified, **not yet on-air-verified** —
  and per
  [the self-consistent-trap discipline]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}),
  a green synthetic is not a verified decode. What closes
  [#1003](https://github.com/MattCheramie/GopherTrunk/issues/1003): run
  `protocol: tetra-dmo` against live radios of **known** colour and MNI,
  with someone actually *talking* (a silent keyed carrier is a poor test
  vector), and report whether an intelligible recording lands. A raw IQ
  capture of the session (`gophertrunk capture`, MCC/MNC noted) makes any
  failure reproducible.

### How this recipe shapes operator practice

- **Silence is a state, not an error.** `camped … waiting for traffic` is the
  healthy idle line; don't chase it.
- **Trust the qualified counter.** A DMO question that starts from
  `dnb_total` starts from a noise meter.
- **Identity comes from config, not the air.** The MNI cannot be sniffed —
  write it down whenever you get codeplug access.

## Where this goes next

Direct mode is the simplest *digital* recipe; [Part
6]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
goes simpler still — plain analog FM. Fire dispatch, marine VHF, GMRS: a
conventional scan list with real squelch, CTCSS/DCS tone gating, and two-tone
fire paging that fires an alert the moment your station's tones hit the air.

## FAQ

**What is TETRA DMO and can an SDR decode it?**
Direct Mode Operation is TETRA's radio-to-radio mode — handhelds on a
simplex channel with no network. GopherTrunk decodes it with one cheap SDR:
sync, traffic detection, colour recovery and clear-voice ACELP. Encrypted
DMO (TEA ciphers) yields metadata only.

**Why do my DMO recordings all show talkgroup 0?**
Because a DMO transmission genuinely announces none to a listener at this
layer — the grant GopherTrunk publishes carries `GroupID 0` by design, and
recordings file under `<system>/0/`. It's a property of what's decodable,
not a config mistake.

**What are tetra_mcc and tetra_mnc for in GopherTrunk?**
They supply the network's Mobile Network Identity, which TETRA folds into
the voice-traffic scrambler seed but never transmits in DMO sync bursts.
With the wrong MNI, voice sits at the chance floor while signalling decodes
fine — the diagnostic tell.

**How fast does GopherTrunk catch a DMO transmission?**
Roughly half a second after PTT: the pipeline requires a sync lock plus four
slot-grid-qualified traffic bursts before declaring a call, so channel noise
can't open recordings on a silent frequency. Buffering means speech from
before the grant still gets decoded.

**Is DMO decoding in GopherTrunk finished?**
The decoders are conformance-tested and capture-verified offline; the
full-daemon on-air loop is the last open gate (#1003). Run it and report what
you see — especially a known-clear channel that *doesn't* decode.

## Series navigation

**Part 5 of 14** · ←
[Part 4: A TETRA TMO Rig]({{ '/blog/tutorials/operator-cookbook-04-tetra-tmo/' | relative_url }})
· Next →
[Part 6: Analog FM & Tone-Out Paging]({{ '/blog/tutorials/operator-cookbook-06-analog-fm-tone-out/' | relative_url }})
