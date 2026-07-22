---
title: "Recording, Composition & Streaming, Part 12: Live Listening — The Local Audio Path"
description: How GopherTrunk lets you hear a call the instant it decodes — a lock-free AudioPublisher fan-out, an open-ended WAV HTTP stream that even Safari will play, a gRPC StreamAudio path, and a host speaker player that deliberately refuses digital PCM to avoid an echo.
category: deep-dives
keywords: live audio streaming sdr, browser wav stream, safari range request 206, grpc streamaudio, audio publisher fan-out, drop on full backpressure, host speaker player oto, gophertrunk live listening, decoded pcm tap, comb filter echo
tags: [streaming, audio, http, grpc, go, real-time]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 12
---

*Part 12 of **Recording, Composition & Streaming**, following one call — the 3
p.m. dispatch on talkgroup 101 — through GopherTrunk's output half. In
[Part 5]({{ '/blog/deep-dives/recording-streaming-05-wav-on-disk/' | relative_url }})
we watched its PCM land in a crash-safe WAV on disk. But an operator staring at
the scanner doesn't want to open a file thirty seconds later — they want to hear
the dispatcher **now**, while it's still keyed up. This post is the live path:
the same decoded PCM that fills the WAV is teed to a fan-out point that feeds a
browser stream, a gRPC subscriber, and the tone-out detector — and, for analog
FM only, the machine's own speakers.*

> **TL;DR:** The recorder decodes each call exactly once and taps that PCM into
> the **`AudioPublisher`** (`internal/api/audio_publisher.go`), a lock-free
> fan-out that copies frames to every connected subscriber over a **bounded
> channel** and **drops on full** rather than letting one slow client stall the
> composer. Browsers play an **open-ended WAV** whose header lies about its
> length so playback never stops; Safari additionally demands a `Range`/`206`
> handshake, which `handleAudioStream` answers. gRPC clients get the same frames
> via `StreamAudio`. The host-speaker **`Player`** is wired separately and is fed
> **analog FM only** — routing decoded digital PCM there too would play the call
> twice on one box and comb-filter into an echo (issue #598 follow-up).

**Key takeaways**

- **One decode, many ears.** The recorder is the lone decoder; its decoded PCM
  is fanned to the publisher, the tone-out detector, and (for analog) the
  speakers. Nobody decodes the channel twice.
- **Drop-on-full, never block.** Every subscriber owns a 256-frame channel. A
  stalled network write drops frames and counts the loss — it never back-pressures
  the composer goroutine that produced them.
- **The WAV header is a polite lie.** `streamingWAVHeader` declares a ~2 GiB data
  chunk so a browser `<audio>` element keeps reading until the socket closes.
- **The host speaker gets analog only.** Digital PCM is excluded from the local
  `Player` on purpose: it already streams to the WebUI, and playing it locally
  too produces two offset copies that comb-filter into an echo.

## Cheat sheet

| Thing | What it does | Where in code |
|---|---|---|
| `AudioPublisher` | Lock-free fan-out of decoded PCM to N subscribers | `internal/api/audio_publisher.go` |
| `WritePCM` / `WritePCMForCall` | Satisfies `composer.PCMSink`; the second fences cross-call bleed by `CallID` | `audio_publisher.go` |
| `WriteRawFrame` | Fans un-decoded IMBE/AMBE bytes to `IncludeRaw` subscribers | `audio_publisher.go` |
| `Subscribe` / `Unsubscribe` | Registers/removes one wire-side stream + its bounded channel | `audio_publisher.go` |
| `audioSubChanCap` | Per-subscriber channel depth (256 frames) | `audio_publisher.go` |
| `handleAudioStream` | Open-ended WAV over HTTP for `<audio src=…>` | `internal/api/handlers_audio_stream.go` |
| `streamingWAVHeader` / `parseRange` | 44-byte oversized WAV header; Safari `Range` parser | `handlers_audio_stream.go` |
| `StreamAudio` | gRPC server-stream of `AudioFrame`s | `internal/api/grpc.go` |
| `player.Player` | Host-speaker sink (oto → ALSA/CoreAudio/WASAPI) | `internal/voice/player/player.go` |
| `liveSinks` wiring | Assembles the live tap; excludes digital from the speaker | `cmd/gophertrunk/daemon.go` |

## In this post

- **The fan-out point** — why `AudioPublisher` implements the same `WritePCM`
  contract the recorder does, and how drop-on-full keeps a slow client harmless.
- **The browser stream** — an open-ended WAV, and the Safari `Range`/`206` dance
  `handleAudioStream` performs to satisfy WebKit's media element.
- **The gRPC path** — the same frames, structured, for non-browser consumers.
- **The host speaker** — the `Player`, and the deliberate rule that keeps
  **digital PCM off the local speakers** so you don't get an echo.

## The fan-out point

By the time our 3 p.m. dispatch is audible, the
[Voice Coding composer]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})
has turned the voice channel's IQ into 8 kHz PCM, and — as
[Part 4]({{ '/blog/deep-dives/recording-streaming-04-recording-session/' | relative_url }})
described — the recorder is the one component that decodes it. The trick that
makes live listening cheap is that the recorder doesn't keep that PCM to itself.
It exposes a **decoded-PCM tap**, and the daemon points that tap at a fan-out
sink whose members all speak the same tiny interface the recorder itself
satisfies:

```go
// internal/composer/sink.go (shape)
type PCMSink interface {
    WritePCM(deviceSerial string, samples []int16) error
}
```

The `AudioPublisher` implements exactly this, which is why the daemon can drop it
into the fan-out beside every other sink without any of them knowing the others
exist. Its whole reason to live is to copy each PCM chunk to any number of
connected listeners:

```go
// internal/api/audio_publisher.go (shape)
func (p *AudioPublisher) WritePCM(deviceSerial string, samples []int16) error {
    p.mu.RLock()
    defer p.mu.RUnlock()
    if len(p.subs) == 0 {
        return nil // no listeners: allocate nothing, copy nothing
    }
    var frame *apiv1.AudioFrame // built lazily on first match
    for sub := range p.subs {
        if !sub.filter.matches(deviceSerial, grant.GroupID) {
            continue
        }
        if frame == nil {
            frame = buildPCMFrame(grant, deviceSerial, samples)
        }
        select {
        case sub.ch <- frame:            // fast consumer: delivered
        default:                          // slow consumer: drop, count, move on
            sub.dropped.Add(uint64(len(samples)))
            p.dropped.Add(uint64(len(samples)))
        }
    }
    return nil
}
```

Two design choices in that loop carry the whole post. First, the frame is built
**lazily** — `buildPCMFrame` runs only once a subscriber actually matches, so the
common "nobody is listening" case copies nothing. Second, the send is a
**non-blocking** `select` with a `default` arm. Each subscriber owns a channel of
`audioSubChanCap` (256) frames — a few seconds of slack at typical chunk sizes.
If a client's network write stalls and its channel backs up, the publisher
**drops the frame and increments a counter** rather than blocking. That is the
single most important property here: the composer goroutine that produced the PCM
is the same one draining the vocoder, and if a dead browser tab could stall it,
one wedged listener would freeze decoding for everyone. Drop-on-full makes a slow
consumer its own problem.

**Subscribers register and deregister** through `Subscribe`/`Unsubscribe`. A
subscriber carries a `filter` (device serials and/or talkgroup IDs; empty matches
everything) and its bounded channel. `Unsubscribe` is idempotent and **closes the
channel**, so the reader side sees a clean end-of-stream. The publisher also
keeps a per-device-serial `Grant` map, fed by its own events-bus subscription, so
each `buildPCMFrame` can stamp the wire frame with talkgroup/system context.

One subtlety worth pinning: the grant cache is best-effort. The bus can drop the
`CallStart` event under load, and gating audio on a known grant once left the
WebUI silent while disk recordings kept working (issue #598). So `WritePCM` emits
the PCM **even when the grant is unknown**, with a zero `Grant`; only
talkgroup-*filtered* subscribers are skipped in that case, because their
predicate can't be proven without a `GroupID` and we must never leak another
talkgroup's audio to them. There is also a stricter sibling, `WritePCMForCall`,
that carries the `CallID`: when a voice-tap serial has already been reused by a
newer call, PCM still draining from the older one is **dropped rather than
mislabelled** with the new call's talkgroup — closing a cross-call audio-bleed
window. A parallel pair, `WriteRawFrame`/`WriteRawFrameForCall`, does the same for
verbatim un-decoded vocoder bytes (IMBE/AMBE+2), fanned only to subscribers that
set `IncludeRaw`.

<figure class="lab-figure">
<svg viewBox="0 0 660 220" width="660" height="220" role="img" aria-label="The recorder decodes a call once and taps its PCM into the AudioPublisher, which fans copies to the HTTP WAV stream, the gRPC StreamAudio path, and the tone-out detector, each over its own bounded drop-on-full channel; the host speaker player is fed on a separate path by analog FM only, not by the digital decoded PCM.">
  <rect x="8" y="86" width="120" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="68" y="106" text-anchor="middle" fill="var(--accent)" font-size="11">recorder</text>
  <text x="68" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">lone decoder</text>
  <line x1="128" y1="109" x2="162" y2="109" stroke="currentColor"/><polygon points="162,105 172,109 162,113" fill="currentColor"/>
  <text x="150" y="101" text-anchor="middle" fill="var(--fg-muted)" font-size="8">decoded PCM tap</text>
  <rect x="172" y="88" width="100" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="222" y="108" text-anchor="middle" fill="var(--accent)" font-size="11">AudioPublisher</text>
  <text x="222" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">fan-out · drop-on-full</text>
  <line x1="272" y1="98" x2="312" y2="34" stroke="currentColor"/><polygon points="308,30 318,32 312,41" fill="currentColor"/>
  <line x1="272" y1="106" x2="312" y2="82" stroke="currentColor"/><polygon points="308,78 318,82 311,90" fill="currentColor"/>
  <line x1="272" y1="118" x2="312" y2="130" stroke="currentColor"/><polygon points="311,125 318,132 308,134" fill="currentColor"/>
  <rect x="318" y="18" width="170" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="403" y="33" text-anchor="middle" fill="currentColor" font-size="10">HTTP WAV stream</text>
  <text x="403" y="44" text-anchor="middle" fill="var(--fg-muted)" font-size="8">&lt;audio src=…&gt; · Safari-safe</text>
  <rect x="318" y="66" width="170" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="403" y="81" text-anchor="middle" fill="currentColor" font-size="10">gRPC StreamAudio</text>
  <text x="403" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="8">structured AudioFrame</text>
  <rect x="318" y="114" width="170" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="403" y="129" text-anchor="middle" fill="currentColor" font-size="10">tone-out detector</text>
  <text x="403" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Goertzel on raw PCM</text>
  <line x1="128" y1="180" x2="312" y2="180" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="312,176 322,180 312,184" fill="var(--fg-muted)"/>
  <text x="215" y="173" text-anchor="middle" fill="var(--fg-muted)" font-size="8">analog FM only (composer main fanout)</text>
  <rect x="322" y="166" width="170" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="407" y="181" text-anchor="middle" fill="var(--fg-muted)" font-size="10">host speaker Player</text>
  <text x="407" y="192" text-anchor="middle" fill="var(--fg-muted)" font-size="8">digital excluded (no echo)</text>
</svg>
<figcaption>One decode, several ears. The publisher fans decoded PCM to the network stream and tone-out; the host speaker is on a separate arm fed by analog FM only, so a digital call is never played locally and streamed at the same time.</figcaption>
</figure>

## The browser stream: an open-ended WAV

The most convenient live sink needs no client software at all — an
`<audio src="/api/v1/audio/stream">` element in the WebUI. `handleAudioStream`
serves that URL as a **single, never-ending WAV file**. The mechanism is delightfully
blunt: a WAV file begins with a 44-byte RIFF/WAVE header whose `data` chunk
declares how many sample bytes follow. `streamingWAVHeader` fills that field with
`0x7FFFFFFF` — almost 2 GiB — so a browser believes it has an enormous file ahead
of it and keeps reading PCM off the socket until the connection closes:

```go
// internal/api/handlers_audio_stream.go (shape)
const (
    wavHeaderSize  = 44
    wavMaxDataSize = uint32(0x7FFFFFFF) // declared data-chunk size (a lie)
    wavTotalSize   = int64(wavHeaderSize) + int64(wavMaxDataSize)
)
```

After emitting the header the handler subscribes to the publisher, then loops:
every matching `AudioFrame` is already 16-bit little-endian mono per
`buildPCMFrame`, so it writes the sample bytes straight through and `Flush`es.
Because the response is long-lived, the handler first clears the server's
`WriteTimeout` via `http.NewResponseController` so a quiet channel between overs
doesn't get the socket torn down mid-call. For Chrome, Firefox, and `curl`, that
plain `200` chunked stream is the whole story.

### The Safari Range dance

Safari (macOS and iOS) is the exception, and it's the reason `parseRange` exists.
WebKit's media element **refuses to play from a server that doesn't advertise
byte-range support.** It will not accept a bare `200`; instead it opens with a
tiny probe — typically `Range: bytes=0-1` — to learn the resource size, and only
then issues the real `Range: bytes=0-` for the body. `handleAudioStream` answers
both:

- **The probe** (`bytes=0-1`, entirely inside the 44-byte header) is served
  without ever subscribing to live audio. The handler slices those exact header
  bytes, sets `Accept-Ranges: bytes` and a `Content-Range` citing the ~2 GiB
  total, and returns `206 Partial Content`. Safari now knows the "size."
- **The bulk request** (`bytes=0-`, open-ended) subscribes for real, replies
  `206` with a `Content-Range` of `bytes 0-<total-1>/<total>`, writes the header
  from the requested offset, and then streams live frames exactly as the `200`
  path does.

`parseRange` is deliberately narrow: it accepts a single `bytes=start-end` or
`bytes=start-`, and returns `ok=false` for suffix ranges (`bytes=-N`), multi-range
requests, or anything malformed — in which case the handler falls back to the
plain `200` stream, which is safe for every non-Safari client. The upshot is that
a **live** stream, which has no real seekable length, satisfies a media element
that was designed around seekable files.

<figure class="lab-figure">
<svg viewBox="0 0 640 210" width="640" height="210" role="img" aria-label="A sequence diagram of Safari's handshake with the audio stream endpoint: Safari first sends a Range request for bytes 0 to 1, the server replies 206 Partial Content with the WAV header bytes and a Content-Range citing the near-2-gigabyte total; Safari then sends an open-ended Range request for bytes 0 onward, and the server replies 206 and streams the live WAV body until the connection closes.">
  <text x="150" y="22" text-anchor="middle" fill="var(--accent)" font-size="11">Safari media element</text>
  <text x="490" y="22" text-anchor="middle" fill="var(--accent)" font-size="11">/api/v1/audio/stream</text>
  <line x1="150" y1="32" x2="150" y2="196" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="490" y1="32" x2="490" y2="196" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="150" y1="52" x2="486" y2="52" stroke="currentColor"/><polygon points="486,48 496,52 486,56" fill="currentColor"/>
  <text x="318" y="46" text-anchor="middle" fill="currentColor" font-size="9">GET · Range: bytes=0-1  (size probe)</text>
  <line x1="490" y1="82" x2="154" y2="82" stroke="var(--accent)"/><polygon points="154,78 144,82 154,86" fill="var(--accent)"/>
  <text x="318" y="76" text-anchor="middle" fill="var(--accent)" font-size="9">206 · Content-Range bytes 0-1/2147483691 · header bytes</text>
  <line x1="150" y1="122" x2="486" y2="122" stroke="currentColor"/><polygon points="486,118 496,122 486,126" fill="currentColor"/>
  <text x="318" y="116" text-anchor="middle" fill="currentColor" font-size="9">GET · Range: bytes=0-  (open-ended body)</text>
  <line x1="490" y1="152" x2="154" y2="152" stroke="var(--accent)"/><polygon points="154,148 144,152 154,156" fill="var(--accent)"/>
  <text x="318" y="146" text-anchor="middle" fill="var(--accent)" font-size="9">206 · WAV header, then live PCM frames…</text>
  <line x1="490" y1="180" x2="154" y2="180" stroke="var(--fg-muted)"/><polygon points="154,176 144,180 154,184" fill="var(--fg-muted)"/>
  <text x="318" y="174" text-anchor="middle" fill="var(--fg-muted)" font-size="9">…flushed per over until the socket closes</text>
</svg>
<figcaption>Safari won't play a bare 200. It probes for a size, then requests the body — and <code>handleAudioStream</code> answers both with a 206, serving the header from cache for the probe and streaming live for the body.</figcaption>
</figure>

## The gRPC path: the same frames, structured

Not every consumer is a browser. A companion app, a transcription bridge, or a
CLI wants **structured** frames, not a raw byte river. That's `StreamAudio`, a
server-streaming RPC that reuses the *identical* publisher plumbing:

```go
// internal/api/grpc.go (shape)
func (g *GRPCServer) StreamAudio(req *apiv1.StreamAudioRequest,
    srv apiv1.AudioService_StreamAudioServer) error {
    sub := g.audio.Subscribe(AudioSubFilter{
        DeviceSerials: req.GetDeviceSerials(),
        TalkgroupIDs:  req.GetTalkgroupIds(),
        IncludeRaw:    req.GetIncludeRaw(),
    })
    defer g.audio.Unsubscribe(sub)
    for {
        select {
        case <-srv.Context().Done():
            return nil
        case frame, ok := <-sub.ch:
            if !ok {
                return nil
            }
            if err := srv.Send(frame); err != nil {
                return err
            }
        }
    }
}
```

It is the HTTP loop with the framing swapped: `Subscribe`, drain the bounded
channel, `Send`, and clean up on `Unsubscribe`. Because a gRPC subscriber can set
`IncludeRaw`, this is the path that carries **un-decoded vocoder frames** — the
verbatim IMBE/AMBE bytes the recorder also writes to the `.raw` sidecar from
[Part 6]({{ '/blog/deep-dives/recording-streaming-06-segmentation-naming-sidecars/' | relative_url }})
— to a client that wants to run its own decoder. HTTP browser clients never opt
into raw, so they only ever see PCM.

## The host speaker, and why digital PCM stays off it

The last live sink is the machine's own speakers: `player.Player`. It, too,
implements `WritePCM`, so it drops into the fan-out like everything else. Its
backend is `github.com/ebitengine/oto/v3` — ALSA on Linux, CoreAudio on macOS,
WASAPI on Windows — with a **software gain/mute stage** applied at `WritePCM`
time so volume changes are instant, and the same **drop-on-full** discipline: a
bounded queue and a non-blocking send, so a stalled audio device never
back-pressures the composer. When `audio.enabled` is false, the device is
`"null"`, or the backend won't open on a headless box, `New` returns a **no-op
player** and the rest of the daemon runs identically — a dead speaker surfaces as
`Stats().BackendError`, not as a crash.

Here is the rule that trips people up. The host speaker is **not** on the same
tap as the network stream. Look at the daemon's live wiring:

```go
// cmd/gophertrunk/daemon.go (shape)
// The host speaker player is deliberately EXCLUDED from this tap:
// routing decoded digital PCM there plays the call on the local
// speaker while the WebUI streams the same PCM, and on one machine
// the two offset playbacks echo / comb-filter (issue #598 follow-up).
var liveSinks []composer.PCMSink
if d.toneout != nil {
    liveSinks = append(liveSinks, d.toneout)
}
liveSinks = append(liveSinks, liveStreamSink) // the AudioPublisher
d.voiceDecoder.SetDecodedPCMSink(fanoutSink(liveSinks))
```

The `liveSinks` fan-out — the decoded-digital tap — contains the publisher and
the tone-out detector, but **not** the `Player`. Digital protocols (P25, DMR,
NXDN, …) reach the WebUI live, and if the same box is running the browser, the
operator hears the call there. Feeding the speaker the same decoded PCM would
play the call **twice on one machine**, a few tens of milliseconds apart, and the
two offset copies comb-filter into a hollow echo. Analog FM, by contrast, still
reaches the speaker — but through the composer's *main* sink fanout, wired via a
`playerSink` adapter that also honours the per-talkgroup `Mute` flag. So the
`Player` gets analog only; digital host-speaker playback is intentionally out of
scope.

> **⚠ If your live browser stream is silent but recordings are fine**, the usual
> cause is grant-gating, not the audio path — which is exactly issue #598. The
> publisher now emits PCM with a zero grant when `CallStart` was dropped, so
> unfiltered and device-only subscribers still hear audio; only talkgroup-filtered
> subscribers are skipped without a known grant.

The `TestDaemonWiresLiveAudioWithoutRecordings` integration test
(`cmd/gophertrunk/daemon_live_audio_test.go`) pins the other half of this:
configuring **no** recordings directory must still build a decode-only voice
decoder, a composer, and a live `audioPub`, so a DMR user who never wanted files
still hears calls in the browser.

## Where this goes next

Live listening is the ephemeral output — heard once, kept nowhere. The durable,
shared output is next.
[Part 13]({{ '/blog/deep-dives/recording-streaming-13-broadcast-manager/' | relative_url }})
opens the **broadcast manager**: the subsystem that waits on `KindCallComplete`,
reads the finished WAV, encodes it to MP3 once, and pushes it to aggregator feeds
with bounded retry — the step that finally turns our 3 p.m. dispatch into a row
in a Broadcastify feed.

## FAQ

**Why does the browser stream never end?** `streamingWAVHeader` writes a 44-byte
WAV header whose `data` chunk claims ~2 GiB of samples follow. A browser `<audio>`
element trusts that length and keeps reading PCM off the socket until the
connection closes, so a single request plays live audio indefinitely rather than
stopping at a real file boundary.

**Why the special handling for Safari?** WebKit's media element refuses to play
from a server that doesn't support byte ranges. Safari sends a small `Range:
bytes=0-1` probe to learn the size, then an open-ended `bytes=0-` for the body;
`handleAudioStream` answers both with `206 Partial Content` and an `Accept-Ranges`
header. Chrome, Firefox, and curl send no `Range` and take the plain `200`
streaming path.

**What happens if a listener is too slow?** Each subscriber owns a bounded
256-frame channel (`audioSubChanCap`). When it fills, the publisher drops the
frame and increments both a per-subscriber and a global counter instead of
blocking. This keeps one wedged client from back-pressuring the composer goroutine
that decodes audio for everyone.

**Why can't I hear digital calls on the host's speakers?** By design. Digital PCM
is excluded from the local `Player` because it already streams to the WebUI;
playing it locally too would produce two offset copies on one machine that
comb-filter into an echo (issue #598 follow-up). Analog FM still reaches the
speakers through the composer's main fan-out.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Retention & Housekeeping]({{ '/blog/deep-dives/recording-streaming-11-retention-housekeeping/' | relative_url }})
· Next →
[Part 13: The Broadcast Manager]({{ '/blog/deep-dives/recording-streaming-13-broadcast-manager/' | relative_url }})
