---
title: "Recording, Composition & Streaming, Part 14: The Aggregator Backends — Broadcastify, RdioScanner, OpenMHz, Icecast & Webhook"
description: A closing tour of every outbound destination's real wire protocol — Broadcastify's two-step metadata-then-PUT upload, the multipart POSTs for RdioScanner and OpenMHz, Icecast's paced silence-topped source connection, and the generic JSON webhook — capped by the pure-Go, zero-CGO testing story that verifies them all.
category: deep-dives
keywords: broadcastify calls api, rdioscanner call upload, openmhz upload multipart, icecast source protocol go, webhook json call feed, two step upload, pure go sdr testing, httptest backend, zero cgo scanner, gophertrunk aggregators
tags: [streaming, broadcast, http, icecast, testing, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Recording, Composition & Streaming"
series_part: 14
---

*Part 14 — the finale — of **Recording, Composition & Streaming**. Thirteen posts
ago our 3 p.m. dispatch on talkgroup 101 was raw IQ; twelve posts ago it was PCM;
last post the broadcast Manager fanned it out to a set of opaque `Backend`s. This
post opens each box — the actual HTTP and TCP conversations that carry a call to
Broadcastify, RdioScanner, OpenMHz, an Icecast server, and a generic webhook —
and then closes the series with the thing that makes all of it trustworthy: a
pure-Go, zero-CGO test suite that stands the whole path up in memory.*

> **TL;DR:** Five backends, five wire protocols behind one `Send(ctx, *Call)
> error` interface. **Broadcastify** is a two-step handshake: POST metadata, get a
> one-time upload URL, PUT the MP3 to it. **RdioScanner** and **OpenMHz** are
> single multipart/form-data POSTs built by a shared `buildMultipart` helper.
> **Icecast** is different in kind — a persistent, paced source socket topped up
> with pre-encoded silence so it never starves. The **webhook** is a stable JSON
> schema, audio opt-in. All five are verified end to end with `httptest` servers,
> in-memory WAVs, and a temp SQLite call log — no CGO, no network, no fixtures.

**Key takeaways**

- **Broadcastify uploads in two steps** because the audio never touches the API
  host: the metadata POST returns a one-time URL (often on object storage) that
  the MP3 is `PUT` to directly.
- **RdioScanner and OpenMHz share the multipart machinery** but not the field
  names — `buildMultipart` writes the body, each backend supplies its own field
  set (RdioScanner's `key`/`system`/`dateTime`, OpenMHz's `source_list`/`patch_list`).
- **Icecast is a live stream, not an upload.** A background goroutine holds a
  source connection open and a ticker writes paced chunks; between calls it cycles
  one second of pre-encoded silence so the server never times the source out.
- **The whole system is pure Go.** MP3 encode, WAV I/O, and SQLite are all
  CGO-free, so the tests run the real encode/decode/persist path in memory against
  `httptest` backends — the strongest possible evidence the wire formats are right.

## Cheat sheet

| Backend | Wire shape | Key functions / fields |
|---|---|---|
| Broadcastify | POST metadata → one-time URL → PUT audio | `requestUploadURL` · `parseBroadcastifyUploadURL` · `putAudio` |
| RdioScanner | one multipart POST to `/api/call-upload` | `buildMultipart` · `key`/`system`/`dateTime` fields |
| OpenMHz | one multipart POST to `/<shortName>/upload` | `source_list` · `patch_list` JSON parts |
| Icecast | persistent paced source socket | `runStream` · `handshake` · `takeChunk` (silence) |
| Webhook | one `application/json` POST | `webhookPayload` · `callType` · `rfc3339In` · `IncludeAudio` |
| Shared | multipart body + filenames + timestamps | `buildMultipart` · `audioFilename` · `rfc3339In` |

## In this post

- **Broadcastify** — why one upload is two HTTP round trips.
- **RdioScanner & OpenMHz** — one multipart helper, two field vocabularies.
- **Icecast** — a paced source connection kept alive with cycled silence.
- **The webhook** — a stable JSON schema and a call-type classifier.
- **The pure-Go coda** — how zero-CGO makes the whole path testable in memory.

## Broadcastify: two round trips for one upload

Every other audio backend POSTs the MP3 in a single request. Broadcastify Calls
splits the upload in two, and the reason is architectural: the API host answers
metadata, but the *audio* goes somewhere else — typically object storage behind a
signed, one-time URL. So `Send` does a metadata POST first, reads back a URL, and
PUTs the audio to it:

```go
// internal/broadcast/broadcastify.go (shape)
func (b *broadcastifyBackend) Send(ctx context.Context, c *Call) error {
    audio, err := c.MP3()
    if err != nil {
        return fmt.Errorf("%s: encode mp3: %w", b.name, err)
    }
    uploadURL, err := b.requestUploadURL(ctx, c) // step 1: metadata POST
    if err != nil {
        return err
    }
    return b.putAudio(ctx, uploadURL, audio) // step 2: PUT to the one-time URL
}
```

Step one, `requestUploadURL`, is a `application/x-www-form-urlencoded` POST
carrying `apiKey`, `systemId`, `callDuration`, `ts` (the Unix start time), `tg`,
`src`, and `freq`. The API answers with a small text body, and
`parseBroadcastifyUploadURL` teases the URL out of it:

```go
// internal/broadcast/broadcastify.go (shape)
func parseBroadcastifyUploadURL(body string) (string, error) {
    fields := strings.Fields(strings.TrimSpace(body))
    if len(fields) == 0 {
        return "", errors.New("broadcastify: empty metadata response")
    }
    if fields[0] == "0" && len(fields) >= 2 {
        return fields[1], nil // "0 <url>" — success + the upload URL
    }
    if strings.HasPrefix(fields[0], "http") {
        return fields[0], nil // bare URL
    }
    return "", fmt.Errorf("broadcastify: metadata response rejected: %s", body)
}
```

The `"0 <url>"` convention — a `0` status token, whitespace, then the URL — is the
Broadcastify Calls protocol's success shape; anything else is treated as a
rejection and surfaced as an error (which the Manager will then retry). Step two,
`putAudio`, is a plain `PUT` of the MP3 bytes with `Content-Type: audio/mpeg` and
an explicit `ContentLength`, to the URL step one handed back. A non-2xx on either
leg returns an error, and the Manager's `sendWithRetry` handles the backoff.

<figure class="lab-figure">
<svg viewBox="0 0 640 210" width="640" height="210" role="img" aria-label="Sequence diagram of the Broadcastify two-step upload. GopherTrunk POSTs form-encoded metadata to the Calls API, which replies with a status zero and a one-time upload URL. GopherTrunk then PUTs the MP3 audio bytes directly to that upload URL, which replies 200 OK.">
  <line x1="70" y1="30" x2="70" y2="190" stroke="var(--fg-muted)"/>
  <text x="70" y="22" text-anchor="middle" fill="var(--accent)" font-size="9">GopherTrunk</text>
  <line x1="330" y1="30" x2="330" y2="190" stroke="var(--fg-muted)"/>
  <text x="330" y="22" text-anchor="middle" fill="currentColor" font-size="9">Calls API</text>
  <line x1="560" y1="30" x2="560" y2="190" stroke="var(--fg-muted)"/>
  <text x="560" y="22" text-anchor="middle" fill="currentColor" font-size="9">upload URL</text>
  <line x1="70" y1="54" x2="330" y2="54" stroke="currentColor"/><polygon points="330,50 340,54 330,58" fill="currentColor"/>
  <text x="200" y="48" text-anchor="middle" fill="currentColor" font-size="8">POST metadata (apiKey, systemId, tg, ts…)</text>
  <line x1="330" y1="86" x2="70" y2="86" stroke="var(--accent)" stroke-dasharray="4 3"/><polygon points="80,82 70,86 80,90" fill="var(--accent)"/>
  <text x="200" y="80" text-anchor="middle" fill="var(--accent)" font-size="8">"0 &lt;upload-url&gt;"</text>
  <line x1="70" y1="130" x2="560" y2="130" stroke="currentColor"/><polygon points="560,126 570,130 560,134" fill="currentColor"/>
  <text x="315" y="124" text-anchor="middle" fill="currentColor" font-size="8">PUT audio/mpeg (the MP3 bytes)</text>
  <line x1="560" y1="162" x2="70" y2="162" stroke="var(--accent)" stroke-dasharray="4 3"/><polygon points="80,158 70,162 80,166" fill="var(--accent)"/>
  <text x="315" y="156" text-anchor="middle" fill="var(--accent)" font-size="8">200 OK</text>
  <text x="315" y="184" text-anchor="middle" fill="var(--fg-muted)" font-size="8">audio never transits the API host — it goes straight to storage</text>
</svg>
<figcaption>Broadcastify Calls: the metadata POST returns a one-time URL, and the MP3 is PUT to that URL directly — two round trips, one logical upload.</figcaption>
</figure>

## RdioScanner and OpenMHz: one multipart helper, two vocabularies

Both of these are single `multipart/form-data` POSTs, and both build the body
with the same tiny helper. `buildMultipart` (`internal/broadcast/multipart.go`)
takes a slice of `multipartField` — a text field carries `Value`, a file part
carries `Filename` + `Data` — and returns the encoded buffer plus the
`Content-Type` header with its generated boundary:

```go
// internal/broadcast/multipart.go (shape)
type multipartField struct {
    Name     string
    Value    string // text field
    Filename string // set → written as a file part
    Data     []byte // the file bytes
}

func buildMultipart(fields []multipartField) (*bytes.Buffer, string, error) {
    // …CreateFormFile for parts with a Filename, WriteField otherwise
}
```

What differs is the **field vocabulary** each service expects. RdioScanner wants
`key`, `system`, `dateTime`, `talkgroup`, `source`, `frequency`, and the audio as
`audio`; OpenMHz wants `api_key`, `freq`, `start_time`, `stop_time`,
`call_length`, `talkgroup_num`, and — the interesting ones —
`source_list` and `patch_list`:

```go
// internal/broadcast/openmhz.go (shape)
sourceList := fmt.Sprintf(`[{"src":%d,"time":%d,"pos":0}]`, c.Source, c.StartedAt.Unix())
patchList := "[]"
if len(c.PatchedGroups) > 0 {
    // "[201,212]" — the patched talkgroups on this call
}
fields := []multipartField{
    {Name: "api_key", Value: b.apiKey},
    {Name: "talkgroup_num", Value: strconv.FormatUint(uint64(c.Talkgroup), 10)},
    {Name: "source_list", Value: sourceList},
    {Name: "patch_list", Value: patchList},
    {Name: "call", Filename: audioFilename(c, "mp3"), Data: audio},
    // …freq, start_time, stop_time, call_length, emergency
}
```

`source_list` is a JSON array OpenMHz expects even when there is only the single
granting unit to report — GopherTrunk sends the one source it knows with `pos:0`.
`patch_list` carries the patched talkgroups when the call is a patch, `[]`
otherwise. Both backends name their audio part with the shared `audioFilename`
helper (`<talkgroup>-<unixstart>.mp3`), and RdioScanner stamps its `dateTime`
with `rfc3339In` so the timestamp renders in the operator's configured display
timezone with an explicit offset — the same helper the webhook uses, so a call
carries one consistent wall-clock time across every feed.

## Icecast: a live stream that never starves

Icecast is the odd one out. The other four backends *upload a finished call*;
Icecast maintains a *continuous live audio stream* that a listener can tune into
at any moment. That changes everything about its shape. There is no per-call
request — instead, `NewIcecast` starts a background goroutine that holds a source
connection open, and `Send` just **appends** the call's MP3 onto a shared queue:

```go
// internal/broadcast/icecast.go (shape)
func (b *icecastBackend) Send(_ context.Context, c *Call) error {
    audio, err := c.MP3()
    if err != nil {
        return fmt.Errorf("%s: encode mp3: %w", b.name, err)
    }
    b.mu.Lock()
    defer b.mu.Unlock()
    if len(b.queue)+len(audio) > b.maxQueue {
        b.log.Warn("broadcast: icecast queue full, dropping call", "tg", c.Talkgroup)
        return nil // best-effort: drop, don't fail (no pointless Manager retry)
    }
    b.queue = append(b.queue, audio...)
    return nil
}
```

Note `Send` returns `nil` even when it drops — a live feed is best-effort, and
returning an error would trigger Manager retries that make no sense for streaming
audio. The real work is in `runStream`, which dials the server, does the Icecast
source `handshake` (a `SOURCE <mount> HTTP/1.0` request with basic auth and
`Ice-*` headers), then loops on a ticker writing paced chunks:

```go
// internal/broadcast/icecast.go (shape)
ticker := time.NewTicker(icecastTick) // 200ms
chunk := b.bytesPerSec * int(icecastTick) / int(time.Second)
silenceOff := 0
for {
    select {
    case <-ctx.Done():
        return nil
    case <-ticker.C:
    }
    payload := b.takeChunk(chunk, &silenceOff)
    conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
    if _, err := conn.Write(payload); err != nil {
        return err // triggers reconnect in run()
    }
}
```

The pacing is the point. Icecast expects a source to feed audio at roughly its
real byte rate; feed too fast and buffers bloat, too slow and the server times
the source out. So each tick writes exactly `bytesPerSec * tick` bytes. When there
is real call audio queued, `takeChunk` serves it; when the queue is empty — the
gap between calls — it **pads the chunk with pre-encoded silence**, cycling
through one second of MP3-encoded digital silence over and over:

```go
// internal/broadcast/icecast.go (shape)
func (b *icecastBackend) takeChunk(n int, silenceOff *int) []byte {
    out := make([]byte, 0, n)
    // …drain up to n bytes of real queued call audio under the lock
    for len(out) < n { // shortfall → top up with cycled silence
        if *silenceOff >= len(b.silence) {
            *silenceOff = 0 // wrap around the one-second silence buffer
        }
        // …append b.silence[silenceOff:end], advance silenceOff
    }
    return out
}
```

The silence is encoded once at construction (`mp3.Encode(make([]int16, rate),
rate)`) and reused forever, so the pacer costs nothing between calls. If the
connection drops, `run` reconnects after a fixed backoff and re-handshakes —
the source stays up across network blips without dropping the listener's stream.

<figure class="lab-figure">
<svg viewBox="0 0 640 210" width="640" height="210" role="img" aria-label="The Icecast pacer. Send appends call MP3 onto a shared queue. A ticker fires every 200 milliseconds; takeChunk pulls a fixed-size chunk, draining real call audio from the queue first and topping any shortfall up from a cycling one-second silence buffer, then writes the chunk to the source socket connected to the Icecast server.">
  <rect x="16" y="30" width="120" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="76" y="49" text-anchor="middle" fill="var(--accent)" font-size="9">Send(call)</text>
  <line x1="76" y1="60" x2="76" y2="86" stroke="currentColor"/><polygon points="72,86 76,96 80,86" fill="currentColor"/>
  <rect x="16" y="96" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="76" y="112" text-anchor="middle" fill="currentColor" font-size="9">queue []byte</text>
  <text x="76" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="8">MP3 of each call</text>
  <rect x="16" y="150" width="120" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="76" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="9">silence buffer</text>
  <text x="76" y="178" text-anchor="middle" fill="var(--fg-muted)" font-size="8">1s, cycled</text>
  <line x1="136" y1="113" x2="250" y2="105" stroke="currentColor"/><polygon points="250,101 260,105 250,109" fill="currentColor"/>
  <line x1="136" y1="167" x2="250" y2="120" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="250,116 260,120 249,124" fill="var(--fg-muted)"/>
  <rect x="260" y="86" width="130" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="325" y="106" text-anchor="middle" fill="var(--accent)" font-size="9">takeChunk(n)</text>
  <text x="325" y="119" text-anchor="middle" fill="var(--fg-muted)" font-size="8">real audio first,</text>
  <text x="325" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="8">pad with silence</text>
  <text x="325" y="72" text-anchor="middle" fill="var(--fg-muted)" font-size="8">ticker · every 200ms</text>
  <line x1="325" y1="80" x2="325" y2="86" stroke="var(--fg-muted)"/>
  <line x1="390" y1="112" x2="440" y2="112" stroke="currentColor"/><polygon points="440,108 450,112 440,116" fill="currentColor"/>
  <rect x="450" y="90" width="80" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="490" y="110" text-anchor="middle" fill="currentColor" font-size="9">source</text>
  <text x="490" y="122" text-anchor="middle" fill="currentColor" font-size="9">socket</text>
  <line x1="530" y1="112" x2="566" y2="112" stroke="currentColor"/><polygon points="566,108 576,112 566,116" fill="currentColor"/>
  <rect x="576" y="90" width="56" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="604" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">Icecast</text>
  <text x="604" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">server</text>
</svg>
<figcaption>The Icecast pacer writes a fixed-size chunk every tick, draining real call audio when there is any and topping up the shortfall with cycled silence so the source connection never starves.</figcaption>
</figure>

## The webhook: a stable schema, audio opt-in

The generic webhook is the escape hatch — the backend for everyone whose sink
isn't one of the four named services. It POSTs one `application/json` object per
call against a documented, stable schema (`webhookPayload`) so a downstream
consumer can rely on it:

```go
// internal/broadcast/webhook.go (shape)
type webhookPayload struct {
    Event       string   `json:"event"` // always "call"
    System      string   `json:"system"`
    Protocol    string   `json:"protocol"`
    CallType    string   `json:"call_type"` // "group" | "unit" | "data"
    Talkgroup   uint32   `json:"talkgroup"`
    Source      uint32   `json:"source,omitempty"`
    FrequencyHz uint32   `json:"frequency_hz"`
    // …P25 site identity (channel_id, rfss_id, site_id, nac), encryption,
    //   emergency, patched_groups — all omitempty so presence means "known"
    StartedAt   string   `json:"started_at"` // rfc3339In
    EndedAt     string   `json:"ended_at"`
    DurationMs  int64    `json:"duration_ms"`
    AudioBase64 string   `json:"audio_base64,omitempty"` // only when opted in
}
```

Two design details are worth calling out. `callType` classifies the call so a
consumer never mistakes a unit-to-unit call's destination radio ID for a
talkgroup — a group call reads `"group"`, an individual call `"unit"`, a data
grant `"data"`. And audio is **opt-in**: unless `IncludeAudio` is set, `Send`
never calls `c.MP3()`, so a metadata-only webhook pays no encode cost and the
payload stays lightweight. When it *is* set, the MP3 is base64-encoded into
`audio_base64` alongside an `audio_format`. Fields that don't apply to a given
call — site identity on non-P25, encryption params on a clear call — are dropped
via `omitempty`, so a consumer can treat presence as "this value is known".

## The coda: pure Go is what makes it testable

The reason this series could describe each wire format with confidence is that
every one of them is exercised by a real test — and those tests run because
GopherTrunk's entire output half is **pure Go, zero CGO**. The MP3 encoder
(Shine), the WAV reader/writer, and the SQLite call log are all CGO-free, so a
test can stand up the real encode → upload → persist path entirely in memory with
nothing mocked but the network endpoint.

The pattern repeats across the suite:

- **In-memory WAV.** `internal/voice/wav_test.go` writes and header-patches a WAV
  through an in-memory `io.WriteSeeker` — no tempfile, no disk. The broadcast
  tests' `writeWAV` helper (`internal/broadcast/broadcast_test.go`) makes a real
  short WAV that the real encoder turns into real MP3 bytes.
- **`httptest` backends.** `internal/broadcast/backends_test.go` stands up
  `httptest.NewServer` handlers that assert on exactly what this post described —
  `TestBroadcastifyTwoStepUpload` checks the metadata form *and* that the second
  leg is a `PUT` whose body begins with the MP3 frame sync `0xFF`;
  `TestRdioScannerUpload` and `TestOpenMHzUpload` parse the multipart body and
  check the field names; `TestIcecastSourceHandshakeAndStream` speaks the raw
  source protocol over a `net.Listener`.
- **Behavioural Manager tests.** `internal/broadcast/manager_test.go` drives the
  full Part 13 machine with a `fakeBackend`: `TestManagerSkipsStreamFalseTalkgroup`
  and `TestManagerDropsShortCalls` pin the two gates,
  `TestManagerRetriesTransientFailure` and `TestManagerGivesUpAfterMaxRetries`
  pin the backoff budget.
- **Timezone and schema.** `internal/broadcast/timezone_test.go` pins `rfc3339In`
  to a fixed offset; `internal/broadcast/webhook_test.go` decodes the posted JSON
  back into a `webhookPayload` and checks the site identity and `call_type`.
- **Temp SQLite.** `internal/storage/calllog_test.go` opens a real database at
  `t.TempDir()/calls.db` and asserts the start/end rows — the same pure-Go SQLite
  the daemon uses, no external server.

That is the quiet payoff of the whole architecture. Because nothing links C,
`go test ./...` builds and runs the real recording, encoding, and streaming code
on any machine with a Go toolchain — no libsndfile, no lame, no sqlite3 dev
package. The wire formats in this post aren't described from a spec; they're
described from tests that send the real bytes and check what a real server
receives.

## The 3 p.m. dispatch, delivered

Fourteen posts ago we picked up one call — the 3 p.m. dispatch on talkgroup 101 —
as it arrived from the vocoders as PCM. The recorder wrote its crash-safe WAV; the
call log indexed it; loudness leveled the distributed copy; the `CallComplete`
event carried its path to the broadcast Manager; the Manager gated it on
`Stream` and `MinDuration`, encoded it to MP3 once, and handed it to a worker.
That worker calls `broadcastifyBackend.Send`. A form POST goes out with the
talkgroup, the source, the frequency, and the start time; Broadcastify answers
`"0 <url>"`; the MP3 bytes `PUT` to that URL; a `200` comes back; `sent["broadcastify"]`
ticks to one. The call is live in the feed. That is the whole output half, end to
end — four independent subsystems, one bus, and a call that made it all the way
out.

## FAQ

**Why does Broadcastify need two requests when everyone else uses one?**
Because the audio doesn't go to the API host. The metadata POST returns a
one-time upload URL — usually pointing at object storage — and the MP3 is `PUT`
directly to it. `parseBroadcastifyUploadURL` reads the `"0 <url>"` success
response; a rejection returns an error the Manager retries.

**Do RdioScanner and OpenMHz share code?**
They share `buildMultipart` and `audioFilename` but not their field sets. Each
backend assembles its own `[]multipartField` — RdioScanner's `key`/`system`/
`dateTime`, OpenMHz's `source_list`/`patch_list`/`talkgroup_num` — and hands it to
the same encoder. The helper writes the body; the backend owns the vocabulary.

**Why does the Icecast backend send silence between calls?**
An Icecast source connection must feed audio continuously or the server times it
out and drops the mount. Between calls there is no audio, so the pacer tops each
timed chunk up from a one-second buffer of pre-encoded silence, cycling it. This
keeps the source alive and the listener's stream unbroken without re-encoding.

**How can the tests be sure the uploads are correct without hitting real servers?**
They stand up `httptest` servers that assert on the exact bytes each backend
sends — the two-step Broadcastify handshake, the multipart field names, the raw
Icecast source protocol — while the encode and WAV paths run for real in memory.
Because the whole stack is pure Go with zero CGO, `go test ./...` exercises the
genuine code path on any machine with a Go toolchain.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Outbound Streaming — The Broadcast Manager]({{ '/blog/deep-dives/recording-streaming-13-broadcast-manager/' | relative_url }})
· This is the finale — back to the
[series index]({{ '/blog/series/recording-streaming/' | relative_url }}).

*Where to next? The pure-Go, zero-CGO story that made this series' tests possible
runs deeper in the SDR Internals series — see
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) ch. 14,
"APIs, Testing & the Pure-Go Story."*
