---
title: "The Operator's Cockpit, Part 6: Client-Side Resampling"
description: How GopherTrunk plays 8 kHz voice on a 48 kHz browser AudioContext — a windowed-sinc polyphase resampler that carries a history tail and fractional cursor across network chunks, per-phase unity-gain normalized, band-limiting the images that linear interpolation leaks, with a split-feed-equals-whole-feed test as its keystone.
category: deep-dives
keywords: sinc resampler javascript, polyphase windowed sinc, 8khz to 48khz audio, band-limited upsampling browser, continuous resampler chunk boundary, blackman window kernel, web audio resampling, gophertrunk operator cockpit
tags: [operator-cockpit, audio, web, dsp, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 6
---

*Part 6 of **The Operator's Cockpit**. Part 5 fed a ring buffer with samples "at
the device rate" and waved past the box that gets them there. This post opens that
box: the client-side resampler that bridges 8 kHz voice to a 48 kHz AudioContext,
continuously, band-limited, and as cleanly as the browser's own path does for a
recorded file.*

> **TL;DR:** The daemon streams 8 kHz PCM; the browser's AudioContext runs at the
> hardware-native rate (~48 kHz) because pinning it to 8 kHz silently rendered
> nothing on Windows/WASAPI. Bridging the two means resampling, and doing it *per
> chunk with no shared state* reintroduces exactly the boundary artifacts we set
> out to remove. So one resampler instance spans the whole stream, carrying a
> **history tail** and a **fractional read cursor** across calls. The cheap
> `LinearResampler` works but leaks audible images; `SincResampler` replaces it
> with a windowed-sinc polyphase kernel — the same class of filter the browser
> uses for a recorded buffer — per-phase unity-gain normalized so DC is exact. The
> keystone test: feeding `[a,b]` then `[c,d]` yields **exactly** what feeding
> `[a,b,c,d]` at once would.

**Key takeaways**

- **One instance spans the stream.** Continuous state — a history tail plus a
  fractional cursor — is what makes consecutive network chunks join seamlessly.
  Per-chunk resampling with no carry re-creates the glitches.
- **Linear interpolation is not good enough.** It's weak anti-imaging: a 3.4 kHz
  in-band tone upsampled to 48 kHz leaks an audible 4.6 kHz image. That's the
  "tinny" character that made live sound worse than the recorded file.
- **Windowed-sinc, precomputed per phase.** A Blackman-windowed sinc kernel,
  quantized to 256 sub-sample phases and normalized to unity DC gain per row, does
  the band-limiting the browser's native sinc path does for a `.wav`.
- **Passthrough is free.** When input and output rates match, both resamplers
  return the input untouched — no work, no copy.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| Linear resampler | cheap continuous linear interp fallback | `web/src/audio/resampler.ts` (`LinearResampler`) |
| Sinc resampler | band-limited polyphase upsampler | `web/src/audio/sincResampler.ts` (`SincResampler`) |
| Kernel build | per-phase windowed-sinc taps, DC-normalized | `sincResampler.ts` (`buildKernel`) |
| History + cursor | continuous state across chunks | `sincResampler.ts` (`history`, `t`) |
| Continuity tests | split feed equals whole feed | `resampler.test.ts`, `sincResampler.test.ts` |
| Worklet sink caller | wires the resampler ahead of the ring | `web/src/audio/streamPlayer.ts` (`WorkletSink`) |

## In this post

- **Why we resample at all** — 8 kHz stream, native-rate context.
- **The continuity problem** — why one instance must span the stream.
- **Linear vs. sinc** — the image linear interpolation leaks.
- **The polyphase kernel** — Blackman-windowed sinc, per-phase, DC-normalized.
- **The keystone test** — split-feed equals whole-feed, bit for bit.

## Why we resample at all

The daemon streams voice at 8 kHz — the vocoder-native rate for digital, and the
analog default. The obvious move is to run the browser's AudioContext at 8 kHz too
and skip resampling entirely. GopherTrunk tried that, and it failed in the worst
way: on Windows/WASAPI an 8 kHz AudioContext is often *accepted without error yet
renders no output* — silent playback in every browser (the follow-up to issue
#598). The reliable choice is to let the context run at the hardware-native rate
(commonly 44.1 or 48 kHz) and resample the 8 kHz stream up to it, exactly the way
the browser resamples a recorded `.wav`. The stream's true rate rides in the WAV
header, so the sink learns it from the data:

```ts
// web/src/audio/streamPlayer.ts (shape) — WorkletSink
configure(streamRate: number): void {
  this.resampler = new SincResampler(streamRate, this.contextRate); // 8k -> ~48k
  this.node.port.postMessage({ type: "reset" });
}
push(samples: Float32Array): void {
  const out = this.resampler ? this.resampler.process(samples) : samples;
  if (out.length === 0) return;
  this.node.port.postMessage({ type: "push", samples: out }, [out.buffer]);
}
```

So resampling sits between the byte reframer and the ring buffer from Part 5, once
per stream, feeding the worklet samples already at the device rate.

## The continuity problem

Here's the trap. Network chunks arrive at arbitrary sizes, and resampling is a
stateful operation: computing an output sample near a chunk's edge needs input
samples that may span *both* sides of the boundary. If you build a fresh resampler
per chunk, every boundary is a discontinuity — a click — which is the exact class
of artifact issue #629 set out to kill. The fix is that **one resampler instance
spans the whole stream**, carrying two pieces of state across every `process`
call: a *fractional read cursor* (where the next output falls, in input-sample
coordinates) and a *history tail* (the last few input samples, so a kernel
centred near the chunk start can still read its left neighbours).

The `LinearResampler` shows the shape in miniature — carry `prev` (the last
sample) and `t` (the fractional position), and shift the cursor by the chunk length
so it stays continuous:

```ts
// web/src/audio/resampler.ts (shape) — LinearResampler.process tail
this.prev = input[len - 1]; // next chunk's index -1 is this chunk's last sample
this.t = t - len;           // keep the cursor continuous across the boundary
```

`SincResampler` does the same thing with a wider window: instead of a single `prev`
it keeps a `2 * HALF_WIDTH`-sample history, and its cursor logic defers any output
whose kernel would read past the chunk end to the next call. Both share the same
contract — same constructor, the same `passthrough` getter, the same
"split-feed equals whole-feed" guarantee — so `SincResampler` is a drop-in for the
`LinearResampler` it replaces.

## Linear vs. sinc

Why replace linear at all, if it's continuous? Because linear interpolation is a
*weak anti-imaging filter*. When you upsample 8 kHz to 48 kHz, an in-band tone
produces spectral images at the input sample rate plus and minus the tone — and
linear interpolation doesn't attenuate them enough, so they fold back as audible
"harsh / tinny" high-frequency junk that the recorded `.wav` (resampled by the
browser's proper sinc path) never had. The test pins it with numbers:

```ts
// web/src/audio/sincResampler.test.ts (shape)
it("band-limits an 8k->48k upsample (suppresses images linear interp leaks)", () => {
  // A 3.4 kHz tone is in-band (below the 4 kHz input Nyquist). Its images sit at
  // 8k ± 3.4k = 4.6 / 11.4 kHz — all above 4 kHz, and must be heavily
  // attenuated. Linear interpolation leaks the 4.6 kHz image badly; sinc does not.
});
```

A 3.4 kHz tone (right in the telephone band) has its first image at 4.6 kHz; the
sinc kernel drives that down to inaudible, linear does not. Same continuity,
categorically better spectrum.

## The polyphase kernel

`SincResampler` precomputes its filter once, at construction, as a table of
**polyphase** rows — one windowed-sinc tap set per fractional sub-sample offset —
so the hot `process` loop is just a table lookup and a dot product, no
trigonometry per sample:

```ts
// web/src/audio/sincResampler.ts (shape) — buildKernel
private static buildKernel(cutoff: number): Float32Array[] {
  const taps = 2 * HALF_WIDTH; // 32-tap effective window
  const rows: Float32Array[] = new Array(PHASES); // 256 sub-sample phases
  for (let p = 0; p < PHASES; p++) {
    const frac = p / PHASES;
    const row = new Float32Array(taps);
    let sum = 0;
    for (let k = 0; k < taps; k++) {
      const x = k - HALF_WIDTH + 1 - frac;
      const w = blackman((x + HALF_WIDTH) / taps);  // ~-58 dB sidelobes
      const h = cutoff * sinc(cutoff * x) * w;
      row[k] = h; sum += h;
    }
    const norm = sum !== 0 ? 1 / sum : 1;           // per-row unity DC gain
    for (let k = 0; k < taps; k++) row[k] *= norm;
    rows[p] = row;
  }
  return rows;
}
```

Three design numbers carry the quality. `HALF_WIDTH = 16` gives a 32-tap window —
a clean stopband for telephone-band speech without being expensive at mono 48 kHz.
`PHASES = 256` quantizes the fractional cursor finely enough that the phase error
sits below the int16 noise floor. And a Blackman window (~-58 dB peak sidelobe)
tames the ringing you'd get from a raw truncated sinc. The `ROLLOFF = 0.9` cutoff
pulls the band edge slightly below Nyquist so the transition band has room; when
*downsampling* the cutoff scales down by the rate ratio so the kernel band-limits
to the *output* Nyquist and doesn't alias.

The per-row normalization is quietly essential. Each phase's taps are scaled so
they sum to 1, which means a constant input reproduces exactly — no amplitude
drift, no DC error. The `preserves a DC level` test asserts precisely that, and it
would fail on any off-by-a-hair kernel.

<figure class="lab-figure">
<svg viewBox="0 0 660 186" width="660" height="186" role="img" aria-label="One resampler instance spans the stream: an 8 kHz input chunk plus a carried history tail feed a windowed-sinc polyphase kernel whose fractional cursor advances by the rate ratio, emitting 48 kHz output samples while any output lacking full kernel support is deferred to the next chunk">
  <rect x="8" y="70" width="120" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="90" text-anchor="middle" fill="currentColor" font-size="11">8 kHz chunk</text>
  <text x="68" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ history tail</text>
  <line x1="128" y1="94" x2="160" y2="94" stroke="currentColor"/><polygon points="160,90 170,94 160,98" fill="currentColor"/>
  <rect x="170" y="58" width="150" height="72" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="245" y="80" text-anchor="middle" fill="var(--accent)" font-size="11">polyphase kernel</text>
  <text x="245" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">256 phases · 32 taps</text>
  <text x="245" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Blackman-windowed sinc</text>
  <text x="245" y="123" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cursor += ratio</text>
  <line x1="320" y1="94" x2="352" y2="94" stroke="currentColor"/><polygon points="352,90 362,94 352,98" fill="currentColor"/>
  <rect x="362" y="70" width="130" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="427" y="90" text-anchor="middle" fill="var(--accent)" font-size="11">48 kHz output</text>
  <text x="427" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">band-limited</text>
  <line x1="492" y1="94" x2="524" y2="94" stroke="currentColor"/><polygon points="524,90 534,94 524,98" fill="currentColor"/>
  <rect x="534" y="70" width="118" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="593" y="90" text-anchor="middle" fill="currentColor" font-size="11">ring buffer</text>
  <text x="593" y="106" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Part 5</text>
  <text x="330" y="160" text-anchor="middle" fill="var(--fg-muted)" font-size="10">outputs whose kernel would read past the chunk end are deferred; the tail folds into history for next time</text>
  <text x="330" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="10">so [a,b] then [c,d] emits exactly what [a,b,c,d] would — the keystone continuity guarantee</text>
</svg>
<figcaption>The resampler is a filter with memory: history plus a fractional cursor make every chunk boundary invisible.</figcaption>
</figure>

## The keystone test

Every property above is nice, but the one that has to be true for live audio not to
click is *continuity across chunk boundaries*, and it gets the strictest test in
the file — a **bit-exact** equality between feeding the stream split into 37-sample
pieces and feeding it whole:

```ts
// web/src/audio/sincResampler.test.ts (shape)
it("joins chunks seamlessly — split feed equals one-shot feed", () => {
  const input = /* 600 samples of a 1 kHz tone at 8 kHz */;
  const whole = once(input, 8000, 16000);           // one process() call

  const r = new SincResampler(8000, 16000);
  const split: number[] = [];
  for (let i = 0; i < input.length; i += 37)         // ragged 37-sample chunks
    split.push(...r.process(Float32Array.from(input.slice(i, i + 37))));

  expect(split.length).toBe(whole.length);
  for (let i = 0; i < whole.length; i++) expect(split[i]).toBe(whole[i]); // ===
});
```

It uses an exact 2× ratio so the comparison can be `toBe` (bit-identical), not "close
enough" — any error in the history/cursor carry would show up as a mismatched
sample. The `handles empty and short chunks` test guards the other edge: a lone
first sample has no right-side kernel support yet, so the resampler correctly emits
*nothing* until enough lookahead arrives, and never throws or emits garbage. Those
two tests together are the guarantee the ring buffer relies on — samples arriving
at a steady rate, joined seamlessly, so the only silence you ever hear is a real
network gap, never a resampler seam.

## Where this goes next

[Part 7]({{ '/blog/deep-dives/operator-cockpit-07-spectrum-waterfall/' | relative_url }})
leaves audio for the other kind of live stream: FFT frames. The daemon streams a
spectrum of the SDR's passband, and the browser paints it as a scrolling waterfall
on a canvas — a different payload, the same "long-lived connection carrying a live
signal" pattern, and the first of the DSP canvases the cockpit renders. For the
finished audio surface, the [Web console]({{ '/web.html' | relative_url }}) and
[live-listening]({{ '/blog/series/recording-streaming/' | relative_url }})
docs show it in use.

## FAQ

**Why not just run the AudioContext at 8 kHz and skip resampling?**
Because it silently renders nothing on Windows/WASAPI — an 8 kHz context is
accepted without error but produces no output. Running at the native rate and
resampling up is reliable across every platform.

**What's wrong with linear interpolation?**
It's a poor anti-imaging filter. Upsampling 8 kHz to 48 kHz, it leaks the spectral
images of in-band tones (e.g. a 4.6 kHz image of a 3.4 kHz tone) as audible harsh
high-frequency content. The sinc kernel attenuates those images properly.

**Why precompute a polyphase table?**
So the hot loop is a table lookup plus a dot product, with no per-sample
trigonometry. 256 phases quantize the fractional position finely enough that the
error is below the int16 noise floor.

**Why must one resampler span the whole stream?**
Because resampling near a chunk edge needs samples from both sides of the boundary.
A fresh resampler per chunk discards the carry and re-creates a click at every
seam. Continuous history plus a fractional cursor make the boundaries invisible —
proven bit-exact by the split-feed test.

**Is resampling always done?**
No. When the input and output rates match (a device whose native rate is already
8 kHz), both resamplers short-circuit to a zero-work passthrough that returns the
input unchanged.

## Series navigation

**Part 6 of 14** · ←
[Part 5: The Live Audio Cockpit]({{ '/blog/deep-dives/operator-cockpit-05-live-audio-cockpit/' | relative_url }})
· Next →
[Part 7: Live Spectrum & Waterfall in the Browser]({{ '/blog/deep-dives/operator-cockpit-07-spectrum-waterfall/' | relative_url }})
</content>
