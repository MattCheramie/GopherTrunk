---
title: "The Operator's Cockpit, Part 5: The Live Audio Cockpit"
description: How GopherTrunk streams live decoded voice to the browser — a publisher that fans PCM to bounded per-subscriber channels, a continuous open-ended WAV body served over fetch so the auth header rides along, and an AudioWorklet ring buffer that emits silence on underrun for gapless playback.
category: deep-dives
keywords: live audio streaming browser, pcm over http wav, audioworklet ring buffer, gapless playback web audio, fetch audio stream auth, drop on full fan-out, streaming wav header, gophertrunk operator cockpit
tags: [operator-cockpit, audio, web, api, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cockpit"
series_part: 5
---

*Part 5 of **The Operator's Cockpit**. Part 4 pushed events *about* calls to the
browser; this post pushes the calls themselves. A running daemon composes decoded
voice into PCM, and the cockpit turns that into sound coming out of a laptop
speaker over the same HTTP surface — a live-listening feed you can point a phone
at from across the house.*

> **TL;DR:** The daemon's `AudioPublisher` fans composed PCM to any number of
> subscribers, each on a **bounded channel it drops on when full** so a slow
> client never stalls the composer. `handleAudioStream` serves one subscriber as a
> single **open-ended WAV body** — a 44-byte RIFF header with a near-2 GiB
> declared data chunk, then raw int16 samples forever. The browser reads it with
> `fetch()` (so the `Authorization` header rides along, unlike a bare `<audio>`
> element), reframes the byte stream into whole samples with `PcmFramer`, and
> feeds an **AudioWorklet ring buffer** that emits silence on underrun instead of
> glitching. Gapless live voice, over the same API as everything else.

**Key takeaways**

- **The publisher never blocks.** Each subscriber has a 256-frame channel;
  `WritePCM` does a non-blocking send and drops (counting the loss) on full, so one
  stalled network write can't back-pressure the decode path.
- **The wire format is a streaming WAV.** A canonical 44-byte header with an
  oversized data-chunk length, followed by concatenated 16-bit mono samples —
  playable by `<audio>`, `curl`, and Web Audio alike.
- **`fetch()`, not `<audio src>`.** Only `fetch` can send the bearer token, and it
  sidesteps the whole `<audio>`/Range/`Content-Length` class of silent failures on
  an infinite stream (issue #598).
- **The ring buffer emits silence on underrun.** The AudioWorklet writes zeros and
  re-primes when it runs dry, rather than repositioning the playback cursor — no
  clicks, no drift, gapless playback under network jitter.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| PCM fan-out | composed PCM → per-subscriber channels | `internal/api/audio_publisher.go` (`AudioPublisher`) |
| Stream handler | one subscriber → open-ended WAV | `internal/api/handlers_audio_stream.go` |
| WAV header | 44-byte RIFF with oversized data chunk | `internal/api/handlers_audio_stream.go` (`streamingWAVHeader`) |
| Byte reframer | network reads → whole int16 samples | `web/src/audio/streamPlayer.ts` (`PcmFramer`) |
| Playback graph | fetch → sink → gain → destination | `web/src/audio/streamPlayer.ts` (`createStreamPlayer`) |
| Ring buffer | gapless worklet, silence on underrun | `web/src/audio/ringBufferProcessor.js` |

## In this post

- **The publisher** — fanning PCM without blocking the composer.
- **The streaming WAV** — one open-ended file that never ends.
- **Why `fetch`** — the auth header and the `<audio>` bugs it dodges.
- **Reframing bytes into samples** — chunk boundaries fall anywhere.
- **The ring buffer** — how silence-on-underrun beats reschedule-on-underrun.

## The publisher

Upstream of the browser, the per-call composer produces PCM and hands it to a set
of sinks — the disk recorder, the tone-out detector, and the live fan-out point,
`AudioPublisher`. The publisher's whole job is to deliver that PCM to any number of
wire-side subscribers *without ever letting a slow one hurt a fast one or the
composer*. That guarantee lives in one `select`:

```go
// internal/api/audio_publisher.go (shape) — writePCM, per subscriber
if frame == nil {
    frame = buildPCMFrame(grant, deviceSerial, samples)
}
select {
case sub.ch <- frame:
    p.markFanned(sub)      // logs once: "live audio is flowing"
default:
    sub.dropped.Add(uint64(len(samples)))
    p.dropped.Add(uint64(len(samples)))
    p.markChannelFull(sub) // logs once: "slow consumer, dropping"
}
```

Each subscriber's channel is bounded (`audioSubChanCap = 256` frames, a few
seconds of slack). The send is non-blocking: if the channel is full — a stalled
network write, a proxy flushing a buffer — the frame is *dropped*, the loss is
counted for visibility, and the composer moves on. Dropping a few frames of a live
feed is a non-event; blocking the composer would jam the decoder. The design
picks the harmless failure.

One subtlety worth its own mention: live audio does **not** gate on knowing the
call's grant. The publisher keeps a per-device grant cache fed by its own bus
subscription, but that subscription can itself drop under load — and on a busy
control channel it does, often enough that gating audio on it once left the live
stream silent while disk recordings kept working (issue #598). So when the grant is
unknown, the frame still ships with a zero grant; only *talkgroup-filtered*
subscribers are skipped, because their predicate genuinely can't be evaluated
without a known group ID.

## The streaming WAV

A subscriber reaches the wire through `handleAudioStream`, which serves it as a
single WAV file that *never ends*. The trick is a canonical 44-byte RIFF/WAVE
header whose two size fields are set to the largest values a WAV reader will
accept, so the consumer keeps reading until the TCP connection closes:

```go
// internal/api/handlers_audio_stream.go (shape) — streamingWAVHeader
func streamingWAVHeader(sampleRate uint32) []byte {
    // 16-bit little-endian mono PCM. The data-chunk length is baked to
    // ~2 GiB (0x7FFFFFFF) so streaming consumers read until the socket closes.
    buf := make([]byte, 44)
    copy(buf[0:4], "RIFF")
    binary.LittleEndian.PutUint32(buf[4:8], wavMaxDataSize-36)
    copy(buf[8:12], "WAVE"); copy(buf[12:16], "fmt ")
    binary.LittleEndian.PutUint32(buf[24:28], sampleRate)      // 8000 today
    copy(buf[36:40], "data")
    binary.LittleEndian.PutUint32(buf[40:44], wavMaxDataSize)  // ~2 GiB
    return buf
}
```

After the header, the handler just forwards each subscriber frame's raw int16
bytes straight to the wire and flushes. It clears the server's `WriteTimeout` for
this connection (it's long-lived by nature) and — because Safari refuses to play
media unless the server answers a `Range:` request — it honours a byte-range probe
by serving the exact header bytes for a small range, then streaming live for the
open-ended `bytes=N-` that follows. Chrome, Firefox, and `curl` never send a Range
and take the plain `200` streaming path. The sample rate is 8 kHz today (the
vocoder-native rate the recorder writes), carried in the header so the client
learns it from the stream rather than assuming.

## Why `fetch`

The obvious way to play a URL in a browser is `<audio src="…/audio/stream">`. It
does not work here, and the reasons are instructive. An open-ended "infinite WAV"
is unreliable for the media element — Chrome on macOS failed it *silently*, the
error swallowed in a `console.warn` (issue #598) — and, more fundamentally, an
`<audio>` element can't attach an `Authorization: Bearer` header, so it could only
ever work on an unauthenticated bind. Both problems vanish with `fetch()`:

```ts
// web/src/audio/streamPlayer.ts (shape) — run()
res = await fetch(opts.url, {
  headers: opts.token ? { Authorization: `Bearer ${opts.token}` } : {},
  credentials: "include",
  signal: abort.signal,
});
const reader = res.body!.getReader();
const framer = new PcmFramer();
const sink = await getSink();
for (;;) {
  const { value, done } = await reader.read();
  if (done) break;
  framer.feed(value);
  if (!configured && framer.format) { sink.configure(framer.format.sampleRate); configured = true; }
  for (let s = framer.takeSamples(); s; s = framer.takeSamples()) sink.push(s);
}
```

This is the same reason the event stream used a WebSocket in Part 4: the header the
browser can't otherwise send. `fetch` gives a `ReadableStream` reader, sidesteps
the entire `<audio>`/Range/206/`Content-Length` class of bugs, behaves the same
whether or not a reverse proxy sits in front, and works identically across Chrome,
Firefox, and Safari.

## Reframing bytes into samples

A network read boundary can fall *anywhere* — mid-header, or between the two bytes
of a single int16 sample. `PcmFramer` absorbs that: it accumulates bytes, consumes
the 44-byte header exactly once, and only ever yields *whole* samples, carrying a
trailing odd byte forward to the next chunk:

```ts
// web/src/audio/streamPlayer.ts (shape) — PcmFramer.takeSamples
takeSamples(): Float32Array | null {
  if (this.fmt === null) {
    const fmt = parseWavHeader(this.buf); // validates RIFF/WAVE, reads rate
    if (fmt === null) return null;         // header still incomplete
    this.fmt = fmt;
    this.buf = this.buf.subarray(WAV_HEADER_SIZE);
  }
  const whole = this.buf.length - (this.buf.length % 2);
  if (whole < 2) return null;              // no complete sample yet
  const samples = int16ToFloat32(this.buf.subarray(0, whole));
  this.buf = this.buf.subarray(whole);     // keep the trailing odd byte
  return samples;
}
```

`parseWavHeader` *throws* if the magic is wrong, so a corrupt stream surfaces as a
visible error instead of playing noise. Everything downstream sees a clean stream
of Float32 samples with a known rate — the byte-level chaos ends here.

## The ring buffer

The last hop is playback, and this is where an earlier design failed audibly. The
original sink scheduled each chunk as its own `AudioBufferSource` at a computed
start time; when the stream fell behind, that scheduler *repositioned the playback
cursor*, and every reposition was a click. The fix (issue #629) is an
`AudioWorklet` that owns a fixed-size ring buffer and, on every render quantum,
does something almost boringly simple: copy out what's buffered, and write
**silence** when it runs dry rather than moving anything.

```js
// web/src/audio/ringBufferProcessor.js (shape) — process()
if (!this.draining && this.available >= this.prime) this.draining = true; // wait for cushion
if (!this.draining) { channel.fill(0); return true; }                     // still priming
for (let i = 0; i < channel.length; i++) {
  if (this.available > 0) {
    channel[i] = this.ring[this.read];
    this.read = (this.read + 1) % this.capacity;
    this.available--;
  } else {
    channel[i] = 0;        // underrun: emit silence …
    this.draining = false; // … and re-prime before draining again
  }
}
```

Three properties fall out of that. It **primes** — stays silent until a jitter
cushion (about 0.25 s) is buffered — so a bursty start doesn't stutter. It emits
**silence on underrun** and re-primes, so a network hiccup is a brief quiet gap,
not a glitch or a permanent drift. And it **drops the oldest frame on overflow**,
so if the producer briefly outruns the consumer, latency stays bounded instead of
the ring lapping itself. The main thread's `WorkletSink` resamples the 8 kHz stream
up to the AudioContext's native rate before enqueuing — the continuous sinc
resampler that is entirely the subject of Part 6 — and hands the samples across the
thread boundary by *transferring* the buffer, so the audio thread never contends
with the network thread for memory.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="Composed PCM fans through the AudioPublisher's bounded per-subscriber channels to the stream handler which emits an open-ended WAV; the browser fetches it, reframes bytes into whole samples, resamples to the device rate, and feeds an AudioWorklet ring buffer that emits silence on underrun">
  <rect x="6" y="66" width="96" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="54" y="86" text-anchor="middle" fill="currentColor" font-size="11">composer</text>
  <text x="54" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="9">PCM</text>
  <line x1="102" y1="89" x2="126" y2="89" stroke="currentColor"/><polygon points="126,85 136,89 126,93" fill="currentColor"/>
  <rect x="136" y="60" width="110" height="58" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="191" y="82" text-anchor="middle" fill="var(--accent)" font-size="11">publisher</text>
  <text x="191" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bounded chan</text>
  <text x="191" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">drop on full</text>
  <line x1="246" y1="89" x2="270" y2="89" stroke="currentColor"/><polygon points="270,85 280,89 270,93" fill="currentColor"/>
  <rect x="280" y="60" width="110" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="335" y="82" text-anchor="middle" fill="currentColor" font-size="11">WAV body</text>
  <text x="335" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">open-ended</text>
  <text x="335" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">fetch + header</text>
  <line x1="390" y1="89" x2="414" y2="89" stroke="currentColor"/><polygon points="414,85 424,89 414,93" fill="currentColor"/>
  <rect x="424" y="60" width="106" height="58" rx="6" fill="none" stroke="currentColor"/>
  <text x="477" y="82" text-anchor="middle" fill="currentColor" font-size="11">PcmFramer</text>
  <text x="477" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">whole samples</text>
  <text x="477" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ resample</text>
  <line x1="530" y1="89" x2="554" y2="89" stroke="currentColor"/><polygon points="554,85 564,89 554,93" fill="currentColor"/>
  <rect x="564" y="60" width="92" height="58" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="610" y="82" text-anchor="middle" fill="var(--accent)" font-size="11">ring buf</text>
  <text x="610" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="9">worklet</text>
  <text x="610" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">silence≠glitch</text>
  <text x="330" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="10">a slow client drops frames at the publisher and hears silence at the ring — never a stalled composer or a click</text>
</svg>
<figcaption>Live audio, composer to speaker: every stage prefers a harmless drop or a moment of silence over back-pressure or a glitch.</figcaption>
</figure>

## Where this goes next

[Part 6]({{ '/blog/deep-dives/operator-cockpit-06-client-side-resampling/' | relative_url }})
zooms into the one box we waved past — the resampler between the byte stream and
the ring buffer. Playing 8 kHz voice on a 48 kHz AudioContext means bridging rates
band-limited, continuously, across chunk boundaries, and doing it as well as the
browser's own sinc path does for a recorded file. That's a small, testable DSP
problem with a surprising number of ways to get it subtly wrong. For the *other*
end of this feed — how the daemon composes and broadcasts the audio in the first
place — see [Recording, Composition &
Streaming]({{ '/blog/series/recording-streaming/' | relative_url }}) Part 12 on
live listening, and the [Web console]({{ '/web.html' | relative_url }}) operator
docs.

## FAQ

**What happens to my audio if my connection is slow?**
You lose frames, gracefully. The publisher drops on a full subscriber channel
(counting the loss) rather than blocking, and the browser's ring buffer emits
silence when it runs dry. You hear brief gaps, never a stalled scanner or a click.

**Why serve a WAV that claims to be 2 GiB?**
So a streaming consumer keeps reading until the socket closes. The 44-byte header's
data-chunk length is set to the largest value a WAV reader accepts; the real length
is "until you hang up."

**Why not just use an `<audio>` element?**
Two reasons: it can't send the `Authorization: Bearer` header an auth-gated daemon
needs, and an open-ended WAV made the media element fail *silently* on Chrome/macOS
(issue #598). `fetch()` fixes both and behaves consistently across browsers.

**Why does the ring buffer write silence instead of catching up?**
Because repositioning the playback cursor on underrun clicks. Emitting silence and
re-priming turns a network hiccup into a brief quiet gap with no glitch and no
cumulative drift.

**Does live audio need to know the call's talkgroup?**
No. Unfiltered and device-filtered subscribers get every frame even when the grant
is unknown (it ships with a zero grant); only talkgroup-*filtered* subscribers are
skipped without a grant, since their filter can't be evaluated safely.

## Series navigation

**Part 5 of 14** · ←
[Part 4: The Event Stream — SSE to React]({{ '/blog/deep-dives/operator-cockpit-04-sse-to-react/' | relative_url }})
· Next →
[Part 6: Client-Side Resampling]({{ '/blog/deep-dives/operator-cockpit-06-client-side-resampling/' | relative_url }})
</content>
