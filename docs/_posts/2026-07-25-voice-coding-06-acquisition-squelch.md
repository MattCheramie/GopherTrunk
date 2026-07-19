---
title: "Voice Coding, Part 6: The Call-Startup Acquisition Squelch"
description: A real bug in GopherTrunk — fresh P25 recordings opened with a full-scale noise burst while the receiver acquired lock, because the FEC resolved marginal dibits to valid-but-wrong IMBE frames. Here's the heuristic that mutes it.
category: deep-dives
keywords: p25 startup scratch, acquisition squelch, imbe noise burst, symbol acquisition garbage, voiced frame run mute, p25 recording noise, gophertrunk imbe squelch
tags: [voice, imbe, p25, squelch, debugging, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Voice Coding"
series_part: 6
---

*Part 6 of **Voice Coding**, and the current-work post. Part 5 delivered clean
bits when the signal is good. This one is about the ~half-second at the start of a
call when the signal *isn't* good yet — when the FEC does its job perfectly and
produces exactly the wrong thing.*

> **TL;DR:** Fresh P25 recordings opened with a **full-scale noise burst** — the
> soft-limiter rail, ~24000–26000, for the first ~0.15–0.55 s. The cause: while
> the receiver is still acquiring symbol lock, the FEC resolves the marginal dibit
> stream to **random-but-valid IMBE frames** the vocoder faithfully synthesized as
> a loud scratch, where a reference decoder is silent. There's no error signal to
> key off — the frames FEC-decode *clean*. The fix (v0.7.1) is a **call-startup
> acquisition squelch**: mute output until a sustained run of **stable-pitch voiced
> frames** confirms real speech, with a failsafe max-mute window. Muted frames are
> **zeroed, not dropped**, so the recording keeps its length.*

**Key takeaways**

- Acquisition garbage is a signature, not an error: it's **idle / unvoiced /
  pitch-jumping**, while real speech opens with a run of voiced, stable-pitch
  frames — so the squelch gates on the *speech* signature, not on a nonexistent
  error flag.
- Release requires `acqRunFrames = 4` consecutive candidate frames that are
  voiced (voiced-fraction ≥ 0.1) *and* stable in pitch (`b_0` jump ≤ 20). Any bad,
  silent, idle, or pitch-jumping frame **breaks the run**.
- A failsafe releases after `acqMaxMuteFrames = 100` (~2 s) so an atypical call is
  never fully muted.
- Muted frames are **zeroed, not dropped** — the WAV keeps its exact length. The
  squelch is **opt-in**, enabled only on the recorder/live path; the raw decoder
  is byte-identical to before.

## Cheat sheet

| Constant | Value | Role |
|---|---|---|
| `acqRunFrames` | 4 | consecutive stable-pitch voiced frames to confirm speech |
| `acqVoicedFracMin` | 0.1 | min voiced-harmonic fraction to count as a speech candidate |
| `acqMaxB0Jump` | 20 | max frame-to-frame pitch-index change still "stable" |
| `acqMaxMuteFrames` | 100 | failsafe: never squelch more than ~2 s |
| `EnableStartupSquelch()` | — | opt-in; recorder enables on live/recording path |
| Muted output | zeroed | keeps recording length; not dropped |

## In this post

- **The symptom** — the numbers from the field captures.
- **Why the FEC makes it worse, not better** — the counter-intuitive root cause.
- **The heuristic** — gate on the speech signature, not on errors.
- **The Go** — the run counter, the two gates, and the zero-not-drop rule.

## The symptom: a full-scale scratch on every call

The v0.7.1 changelog entry states it flatly:

> *P25 Phase 1 recordings opened with a full-scale "startup scratch." … Measured
> across field captures, every call blasted the soft-limiter rail (~24000–26000)
> for the first ~0.15–0.55 s.*

Every call. A ~0.15–0.55 second burst of noise pinned to the ±26000 rail (out of
int16's ±32767), right where a reference decoder like Trunk Recorder plays silence.
On playback it's an ugly *scritch* before the voice starts — not a decode failure
you'd catch in a test, because the audio after it is fine. It's a quality bug that
only shows up when you listen to real captures.

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="Before-and-after waveform comparison. Before: the first half second is a full-scale noise burst pinned near the plus and minus 26000 rail, then speech follows. After: the acquisition window is muted to zero, then the same speech follows unchanged.">
  <!-- BEFORE -->
  <text x="20" y="20" fill="currentColor" font-size="12">before — startup scratch</text>
  <line x1="20" y1="60" x2="660" y2="60" stroke="var(--fg-muted)" stroke-dasharray="2 3" opacity="0.5"/>
  <text x="664" y="63" fill="var(--fg-muted)" font-size="8">+26k</text>
  <line x1="20" y1="100" x2="660" y2="100" stroke="var(--fg-muted)" opacity="0.4"/>
  <line x1="20" y1="140" x2="660" y2="140" stroke="var(--fg-muted)" stroke-dasharray="2 3" opacity="0.5"/>
  <text x="664" y="143" fill="var(--fg-muted)" font-size="8">−26k</text>
  <!-- noise burst region -->
  <rect x="20" y="60" width="180" height="80" fill="currentColor" opacity="0.08"/>
  <g stroke="currentColor" stroke-width="1">
    <path d="M20 100 L28 62 L36 138 L44 65 L52 135 L60 61 L68 139 L76 70 L84 132 L92 63 L100 137 L108 66 L116 134 L124 62 L132 138 L140 68 L148 130 L156 64 L164 136 L172 67 L180 133 L188 70 L196 128"/>
  </g>
  <text x="110" y="158" text-anchor="middle" fill="currentColor" font-size="9">~0.15–0.55 s, full-scale</text>
  <path d="M200 100 q10 -18 20 0 q10 18 20 0 q10 -22 20 0 q10 22 20 0 q10 -16 20 0 q10 16 20 0 q10 -20 20 0 q10 20 20 0 q10 -14 20 0 q10 14 20 0 q10 -18 20 0 q10 18 20 0 q10 -20 20 0 q10 20 20 0 q10 -16 20 0 q10 16 20 0 q10 -18 20 0 q10 18 20 0 q10 -15 20 0 q10 15 20 0 q10 -18 20 0 q10 18 20 0" fill="none" stroke="var(--accent)"/>
  <text x="430" y="158" text-anchor="middle" fill="var(--accent)" font-size="9">real speech</text>

  <!-- AFTER -->
  <text x="20" y="188" fill="var(--accent)" font-size="12">after — acquisition squelch (muted, not dropped)</text>
  <line x1="20" y1="205" x2="660" y2="205" stroke="var(--fg-muted)" opacity="0.4"/>
  <line x1="20" y1="205" x2="200" y2="205" stroke="var(--accent)" stroke-width="2"/>
  <text x="110" y="222" text-anchor="middle" fill="var(--accent)" font-size="9">zeroed — same length</text>
  <path d="M200 205 q10 -12 20 0 q10 12 20 0 q10 -15 20 0 q10 15 20 0 q10 -10 20 0 q10 10 20 0 q10 -14 20 0 q10 14 20 0 q10 -9 20 0 q10 9 20 0 q10 -12 20 0 q10 12 20 0 q10 -14 20 0 q10 14 20 0 q10 -11 20 0 q10 11 20 0 q10 -12 20 0 q10 12 20 0 q10 -10 20 0 q10 10 20 0 q10 -12 20 0 q10 12 20 0" fill="none" stroke="var(--accent)"/>
</svg>
<figcaption>Before: the acquisition window renders as a rail-clipping burst. After: those frames are zeroed so the recording keeps its exact length and the speech that follows is untouched. The real numbers — ±26000 rail, 0.15–0.55 s — come straight from the field captures.</figcaption>
</figure>

## Why the FEC makes it worse, not better

This is the counter-intuitive part, and it's why the fix lives in the *vocoder*
rather than the receiver. During the first few hundred milliseconds of a
transmission, the C4FM demodulator hasn't converged on symbol timing yet. The dibit
stream it emits is *marginal* — right on the edge between symbols. You'd expect
that to produce FEC failures the decoder could catch and mute.

It does the opposite. The Golay and Hamming FEC from Part 5 take those marginal
dibits and, doing exactly what they're designed to do, snap them to the **nearest
valid codeword**. The result is a *clean-decoding*, structurally-valid IMBE frame —
`b_0` in range, a plausible `L`, voicing bits set — that just happens to encode
random noise. `UnpackParams` returns no error. The synthesizer faithfully grows a
loud, broadband, phase-aligned burst from it. The better the FEC, the more
convincingly it launders acquisition noise into valid frames.

So there is **no error signal to gate on**. The frames FEC-decode clean; the
corrected-bit count is unremarkable; nothing in the bit layer says "this is
garbage." The comment in the source names the constraint precisely:

> *There is no error signal to key off (the frames FEC-decode clean), so the
> squelch gates on the SPEECH signature instead.*

## The heuristic: gate on speech, not on errors

If you can't detect the garbage, detect its *absence*. Acquisition garbage and real
speech have opposite signatures:

- **Acquisition garbage** is idle, unvoiced, or **pitch-jumping** — successive
  frames land on wildly different `b_0` values because they're random.
- **Real speech** opens with a **sustained run of voiced frames whose pitch is
  stable** — a talker's fundamental glides by a few pitch indices per frame, it
  doesn't jump by ~100.

So the squelch mutes output until it sees that speech signature: a run of frames
that are (a) voiced enough and (b) stable in pitch. The tuning constants encode
exactly those two tests:

```go
// internal/voice/imbe/decoder.go
const (
    acqRunFrames     = 4    // consecutive stable-pitch voiced frames confirm speech
    acqVoicedFracMin = 0.1  // min voiced-harmonic fraction for a speech candidate
    acqMaxB0Jump     = 20   // largest frame-to-frame pitch-index change still "stable"
    acqMaxMuteFrames = 100  // failsafe: never squelch more than ~2 s of a call
)
```

`acqMaxB0Jump = 20` is the crux: real speech pitch glides by a few indices per
frame, acquisition garbage jumps by ~100, so a jump threshold of 20 cleanly
separates them. And `acqVoicedFracMin = 0.1` excludes the fully-unvoiced
acquisition noise (voiced fraction 0) while still catching a genuine — even
breathy — voiced onset, which runs well above 0.1.

## The Go: a run counter and two gates

The squelch runs *before* the frame-disposition switch, on every frame, so that
idle/bad/silent frames break the run just like a pitch jump does:

```go
// internal/voice/imbe/decoder.go (shape) — runs every frame while !acquired
if d.squelchEnabled && !d.acquired {
    voiceCand := err == nil && !p.Silent && !p.IdleTone
    vf := 0.0
    if voiceCand && p.L > 0 {
        vc := 0
        for l := 1; l <= p.L; l++ { if p.Vl[l] == 1 { vc++ } }
        vf = float64(vc) / float64(p.L)          // voiced-harmonic fraction
    }
    if voiceCand && vf >= acqVoicedFracMin {
        if d.acqLastVoicedB0 >= 0 && iabs(b0-d.acqLastVoicedB0) <= acqMaxB0Jump {
            d.acqRun++                            // stable pitch: extend the run
        } else {
            d.acqRun = 1                          // first candidate / pitch jumped
        }
        d.acqLastVoicedB0 = b0
    } else {
        d.acqRun = 0                              // non-candidate breaks the run
        d.acqLastVoicedB0 = -1
    }
    d.acqFrames++
    if d.acqRun >= acqRunFrames || d.acqFrames >= acqMaxMuteFrames {
        d.acquired = true                         // release (run confirmed, or failsafe)
    }
}
```

Two ways out of the muted state: the run reaches 4 stable-pitch voiced frames
(speech confirmed), *or* the failsafe `acqFrames` hits 100 (~2 s). The failsafe is
what makes the heuristic safe to ship — it's a heuristic, and the true acquisition
state lives in the receiver, not the vocoder frames, so a genuinely unusual call
(say, an unvoiced-only opening) that never presents a stable voiced run still
un-mutes after ~2 s rather than staying silent forever.

The actual muting happens in `accumStats`, on **every** return path, so the mute
covers all frame dispositions uniformly:

```go
// internal/voice/imbe/decoder.go (shape) — in accumStats, after AGC
if d.squelchEnabled && !d.acquired {
    for i := range out { out[i] = 0 }   // zero, do NOT drop
    d.stats.sampleCount += len(out)
    return
}
```

Two design choices in those few lines carry the whole fix:

- **Zeroed, not dropped.** The muted samples are set to zero and still counted, so
  the WAV keeps its exact length and stays time-aligned with the call. Dropping
  frames would shorten the recording and slip everything after it.
- **Opt-in.** The squelch only engages when the recorder calls
  `EnableStartupSquelch()` on the live/recording path. The raw decoder — and every
  unit test pinned against reference vectors — is **byte-identical** to before,
  because a heuristic that can mute the opening of an atypical call has no business
  in the faithful-decode path. `ResetStats` re-arms it per call/segment, so every
  split recording re-suppresses its own startup burst.

### Where this sits among its cousins

The acquisition squelch is one of a small family of "don't voice the garbage"
guards in this decoder, and they're worth distinguishing because they solve
adjacent-but-different problems:

| Guard | Trigger | What it suppresses |
|---|---|---|
| **Acquisition squelch** (this post) | call start, until a voiced run | receiver-lock scratch |
| **Idle-tone mute** (`IdleToneRunThreshold`) | run of `b_0 ≤ 7` frames | dead-key ~350 Hz buzz |
| **Adaptive smoothing** (`Smoother`) | high FEC error rate | weak-channel amplitude spikes |
| **Frame-repeat** (`badFrameCount`) | `UnpackParams` error | dropouts on bad frames |

Each keys off a different signature — a voiced-run *absence*, a low-`b_0` *run*, an
error-*rate*, an unpack *error* — because there's no single "this frame is bad"
signal. The acquisition squelch is the one that finally silenced the startup
scratch that every fresh P25 capture opened with.

## Where this goes next

That's the end of the IMBE arc. The next two parts cross to the other codec:
[Part 7]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }})
decodes AMBE+2 2400 (P25 Phase 2 / DMR / NXDN) into the same `mbe.Params` — the
two-stage vector quantization that replaces IMBE's PRBA-and-DCT unpack — and
[Part 8]({{ '/blog/deep-dives/voice-coding-08-ambe-plus-2-fec-knox/' | relative_url }})
covers its Golay-plus-Knox FEC. The synthesis core from Part 3 doesn't change at
all; only the bits feeding it do. For the codec background, the
[AMBE+2]({{ '/reference/ambe-plus-2/' | relative_url }}) Field Guide entry sets up
Part 7.

## FAQ

**What caused the P25 startup scratch?**
During the first ~0.15–0.55 s of a call the receiver hasn't acquired symbol lock,
so the demodulator emits marginal dibits. The FEC snaps them to valid IMBE
codewords, producing clean-decoding but meaningless frames the vocoder synthesized
as a full-scale (±26000) noise burst.

**Why not just fix it in the receiver or with the FEC error count?**
Because the frames FEC-decode *clean* — there's no error signal. The FEC is doing
its job correctly; it just can't tell noise-snapped-to-a-codeword from real data.
The garbage is only distinguishable by its *speech* signature, which is a
vocoder-level property.

**How does the squelch know when real speech has started?**
It waits for a run of 4 consecutive frames that are voiced (voiced-harmonic
fraction ≥ 0.1) and stable in pitch (frame-to-frame `b_0` change ≤ 20). Acquisition
garbage is unvoiced or pitch-jumping, so it can't sustain that run.

**What if a call never presents a stable voiced run?**
A failsafe releases the squelch after 100 frames (~2 s), so an atypical call is
never fully muted. The heuristic degrades to "mute at most the first ~2 s," which
is far better than a permanently silent call.

**Why zero the muted frames instead of dropping them?**
To preserve the recording's length and time alignment. Zeroed frames still count
toward the sample total, so the WAV stays the right duration; dropping them would
shorten the file and shift everything after the mute.

## Series navigation

**Part 6 of 12** · ←
[Part 5: IMBE FEC & De-Interleave]({{ '/blog/deep-dives/voice-coding-05-imbe-fec-deinterleave/' | relative_url }})
· Next →
[Part 7: AMBE+2 2400]({{ '/blog/deep-dives/voice-coding-07-ambe-plus-2/' | relative_url }})
