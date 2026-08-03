---
title: "Running It For Real, Part 8: The Opt-In Feature Matrix"
description: How GopherTrunk decides what runs by default versus what stays off until you ask — the three kinds of feature gate, why headless-safe defaults matter, and the one config rule that keeps optional subsystems from surprising a 24/7 daemon.
category: deep-dives
keywords: sdr feature flags, opt-in features, config gates, headless defaults, daemon subsystems, gophertrunk config yaml, safe defaults, feature matrix, gophertrunk running it for real
tags: [running-it-for-real, config, hardening, operations, sdr, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Running It For Real"
series_part: 8
---

*Part 8 of **Running It For Real**, the series about taking GopherTrunk from a
laptop demo to a hardened, always-on service. On the laptop you flip everything
on to see what it does. On a 24/7 daemon you do the opposite: you run the
smallest surface that meets your goal and turn features on one at a time, on
purpose. This post is the map of that decision — which subsystems are on by
default, which stay off until you ask, and why the difference is a deployment
property, not a preference.*

> **TL;DR:** GopherTrunk sorts every optional subsystem into three buckets:
> **on by default, opt-out** (spec-correct FEC, clock recovery, metrics —
> correct for almost everyone); **off by default, opt-in** (audio playback,
> broadcast feeds, loudness, equalizer — each needs a credential, costs CPU, or
> changes output, so no zero-config default is right); and **opt-in by nature**
> (patent-encumbered vocoders, CI-only test tags — never candidates to flip on).
> The rule that makes it safe is that a headless daemon's default is whatever
> won't crash or warn loudly in a distroless container, and every gate is a plain
> config knob you can audit before it leaves your LAN.

**Key takeaways**

- **Three buckets, one question each.** "Is the default correct for a headless
  server?" decides which bucket a feature lands in — not whether the feature is
  good.
- **A missing credential is not a default.** Broadcast feeds, Icecast, and tone-out
  all stay off because there is no useful zero-config value — the operator must
  supply one.
- **On-by-default means opt-out per system, not global.** Spec-correct FEC runs
  everywhere; you disable it per-system only for pre-stripped capture files, and
  the connector maps an empty string to the on-default.
- **Everything is auditable before it runs.** `/api/v1/runtime`, the config
  loader's startup log, and the TUI Settings panel all report the effective value
  of every gate, so you confirm posture without probing endpoints.

## Cheat sheet

| Bucket | Example gates | Where it lives |
|---|---|---|
| On by default, opt-out per protocol | `tetra_channel_coding`, `nxdn_viterbi_mode`, `p25_phase2_clock_mode` | `internal/scanner/ccdecoder/pipelines.go` (`Parse*Mode`) |
| On by default, daemon-wide | `metrics.enabled`, `retention.call_log_days`, `recordings.write_raw` | `internal/config/config.go` |
| Off by default, opt-in | `audio.enabled`, `broadcast.*`, `recordings.normalize.enabled`, `recordings.equalizer.enabled` | `internal/config/config.go` |
| Off by default, auto-detect | `scanner.manual_tune_enabled` | `internal/config/config.go` (`ScannerConfig`) |
| Opt-in by nature (build tag) | `-tags dvsi`, `-tags integration` | build flags |
| Verify effective state | `GET /api/v1/runtime`, `GET /api/v1/mutations` | `internal/api/handlers_runtime.go` |

## In this post

- **The three buckets** — and the single question that assigns a feature to one.
- **On by default** — why FEC and metrics run everywhere, and what "opt-out" means.
- **Off by default** — the credential rule, the CPU rule, the changes-output rule.
- **Auto-detect** — the one gate that reads your hardware instead of a flag.
- **Verifying posture** — how to know what's actually on before it matters.

## The three buckets

Point GopherTrunk at a control channel on your laptop and you want maximum
signal: every decoder aggressive, audio on, the dashboard busy. Run it as a
service and the calculus inverts — every subsystem you don't need is CPU you're
paying for, a credential you're storing, or a way the process can die in a
container that has no sound card. So the [opt-in features
reference]({{ '/opt-in-features.html' | relative_url }}) sorts every optional
piece by a single question: **is the default correct for a headless server?**

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="Three feature-gate buckets. The first, on by default with per-protocol opt-out, holds spec-correct FEC, clock recovery, and metrics. The second, off by default and opt-in, holds audio, broadcast feeds, loudness, and the equalizer. The third, opt-in by nature, holds patent-encumbered vocoders and CI-only build tags. An arrow shows a feature moving from opt-in to on-by-default once its headless default becomes correct.">
  <rect x="6" y="40" width="204" height="90" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="108" y="60" text-anchor="middle" fill="var(--accent)" font-size="12">on by default</text>
  <text x="108" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">opt-out per protocol</text>
  <text x="108" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">FEC · clock recovery</text>
  <text x="108" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">metrics · retention</text>
  <rect x="238" y="40" width="204" height="90" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="60" text-anchor="middle" fill="currentColor" font-size="12">off by default</text>
  <text x="340" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">opt-in</text>
  <text x="340" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">audio · broadcast</text>
  <text x="340" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">loudness · equalizer</text>
  <rect x="470" y="40" width="204" height="90" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="572" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="12">opt-in by nature</text>
  <text x="572" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">never flips on</text>
  <text x="572" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DVSI vocoder</text>
  <text x="572" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CI build tags</text>
  <line x1="340" y1="36" x2="150" y2="36" stroke="var(--accent)" stroke-dasharray="4 3"/><polygon points="150,32 140,36 150,40" fill="var(--accent)"/>
  <text x="245" y="26" text-anchor="middle" fill="var(--fg-muted)" font-size="9">features migrate left once the headless default is correct</text>
</svg>
<figcaption>Every optional subsystem answers one question — is the default right for a headless server? — and lands in one of three buckets accordingly.</figcaption>
</figure>

The buckets aren't fixed forever. Spec-correct FEC used to be opt-in; once the
chain was validated against reference decoders it was **flipped on by default**
for every protocol, moving from the middle bucket to the left. What can never
move is the third bucket: a patent-encumbered vocoder or a CI-only test tag isn't
waiting to become a good default — it's structurally opt-in.

## On by default: correct for everyone, opt-out per system

The left bucket is the features where the spec-correct behaviour is right for
almost every deployment. Per-protocol forward error correction is the headline
case: the `ccdecoder` connector runs each protocol's `Parse*Mode` function over
the configured YAML value, and an **empty string maps to the on-default**. You
get the full TETRA §8.3.1 chain, NXDN's spec Viterbi, EDACS BCH, and the rest
without configuring anything.

```go
// internal/scanner/ccdecoder/pipelines.go (shape)
// The connector always parses the operator's YAML string through the
// per-protocol mode function; "" is the spec-correct on-default, not "off".
mode := tetra.ParseChannelCodingMode(opts.System.TETRAChannelCoding) // "" → ChannelCodingOn
cc.SetChannelCodingMode(mode)
```

The only reason to opt out is a **pre-stripped capture file** — a DSD-FME dump or
an OP25 fixture that already had its FEC removed — where running the chain again
would fail on data that was never encoded. That's a per-system `<key>: off`, not
a global switch, because it's a property of *that* capture, not your deployment.
The same shape covers Gardner clock recovery (`p25_phase2_clock_mode`,
`tetra_clock_mode`), which defaults to `ClockGardner` and opts back to a naive
decimator only for sample-aligned synthetic fixtures.

Daemon-wide on-defaults follow the same logic from the other direction:
`metrics.enabled` is `true`, `retention.call_log_days` is `30`, and
`recordings.write_raw` ships `true` in the example config — each is a sensible,
already-safe default that a headless daemon wants, listed in the matrix only for
completeness so an operator auditing the surface finds nothing hidden.

### How that principle shaped the Go code

- **The zero value is legacy `Off`; the connector installs the on-default.** In
  package `ControlChannel` constructors still zero-value to `Off` so direct unit-test
  callers see historical behaviour without setup. The operator-facing on-default
  lives one layer up, in the connector's `Parse*Mode(opts.System.X)` call — so the
  fixtures' expectations and the operator's default don't fight.
- **Opt-out is per system, keyed in the `System` struct.** Each protocol carries
  its own YAML field (`tetra_channel_coding`, `ltr_fcs_mode`, …), so one
  pre-stripped system's opt-out never touches the system on the next tuner.
- **The parse is total.** There's no "unset" path that silently skips the chain —
  an empty string is a value that means on, so a config that names a system at all
  gets spec-correct decoding.

## Off by default: three reasons a feature stays dark

The middle bucket is where the daemon's headless posture actually lives, and each
gate is off for one of three concrete reasons.

**It needs a credential.** Outbound streaming (`broadcast.*`) is the clearest
case. Every Broadcastify, RdioScanner, OpenMHz, or Icecast feed requires an
operator-supplied API key, system ID, or source password — there is no
zero-config default because the value is yours, not ours. A feed with
`enabled: false` is parsed but skipped, so you can stage a feed's config and flip
it live later. Tone-out paging detection is the same: two-tone profiles are
per-agency, so the default is an empty list.

**It costs CPU without a universal payoff.** The CMA blind equalizer
(`recordings.equalizer.enabled`) mitigates simulcast distortion, but on a
clean-RF site it's pure overhead — so it stays a global opt-in until a per-call
auto-tune heuristic can decide site-by-site.

**It changes your output.** Loudness normalization (`recordings.normalize.enabled`)
and the "sound-good" enhancement chain (`recordings.enhance.enabled`) both alter
recorded or streamed audio. Faithful output is byte-identical to the vocoder;
turning these on trades a little faithfulness for a louder, cleaner sound, and not
every operator wants that — so the default is off and honest.

```go
// internal/config/config.go (shape) — the credential-gated feeds
type BroadcastifyFeedConfig struct {
    Enabled  bool     `yaml:"enabled"`  // parsed-but-skipped when false
    Name     string   `yaml:"name"`
    APIKey   string   `yaml:"api_key"`  // operator-supplied; no default exists
    SystemID int      `yaml:"system_id"`
    Systems  []string `yaml:"systems"`  // empty = every system
}

// AudioConfig gates live speaker playback. Off by default so distroless /
// container deployments stay silent; the recorder is unaffected either way.
type AudioConfig struct {
    Enabled bool `yaml:"enabled"` // default false — WAVs still land on disk
    // …Device, SampleRate, BufferMs, Volume
}
```

Audio playback (`audio.enabled`) deserves its own note because its default
protects against a *crash*, not just a preference. Audio-on-by-default would warn
loudly — or fail outright — in a distroless or Alpine container with no
PulseAudio/ALSA and no sound device. So it's off by design, and critically, WAV
recording is completely independent of it: recordings land on disk whether
playback is on or off. The headless daemon is silent and still fully functional.

## Auto-detect: the gate that reads your hardware

One gate doesn't take a boolean at all. Manual VFO tune
(`scanner.manual_tune_enabled` / `scanner.manual_tune_disabled`) **auto-enables
when two or more Voice SDRs are present** — the daemon builds the conventional
scanner off the spare tuner. You can force it on with `manual_tune_enabled: true`
even with a single Voice SDR, or veto the auto-detect with
`manual_tune_disabled: true` if you want every Voice SDR reserved for trunking.

That three-state shape (unset = auto, or one of two explicit overrides) is the
right pattern whenever the correct default depends on hardware the operator
hasn't told us about yet. It shows up again in `dmr_interleaved_voice` — unset
picks the protocol default, true/false forces it — and it's the reason the
matrix has an "auto-detect" row at all: sometimes the honest default is "look and
decide," not a fixed yes or no.

## Verifying what's actually on

A hardened deployment's worst failure is a feature you *think* is off. So every
gate is auditable, and the read is safe to scrape. `GET /api/v1/runtime` returns
a sanitised snapshot — paths and modes, never secrets or tokens — that the TUI and
web Settings panel render directly:

```go
// internal/api/handlers_runtime.go (shape) — read-only, no credentials
type RuntimeDTO struct {
    MetricsEnabled           bool   `json:"metrics_enabled"`
    RecordingWriteRaw        bool   `json:"recording_write_raw"`
    RecordingSkipEncrypted   bool   `json:"recording_skip_encrypted"`
    RecordingEQEnabled       bool   `json:"recording_eq_enabled"`
    AudioEnabled             bool   `json:"audio_enabled"`
    ScannerManualTuneEnabled bool   `json:"scanner_manual_tune_enabled"`
    // …every other effective gate, omitempty for what doesn't apply
}
```

The DTO comment is a contract: **keep it strictly read-only — no secrets, no
credentials, no auth tokens** — because operators expect `/api/v1/runtime` to be
safe to point a dashboard at. Beyond it, three more doors report posture: the
config loader logs the effective value of every section as it parses; the TUI
Settings panel's FEC tab lists each system's one-line summary
(`channel coding: on`, `viterbi: spec`); and `GET /api/v1/mutations` is always
open and reports the auth mode plus `can_mutate` for the current request, so a
script can light up write-side controls without probing real endpoints. You never
have to guess what's enabled — you read it.

## Where this goes next

The biggest opt-in in the matrix is outbound streaming, and it's the next three
posts. [Part 9]({{ '/blog/deep-dives/running-it-for-real-09-broadcast-backends-i/' | relative_url }})
starts operating the Broadcastify Calls feed for real — the credentials it
needs, how it retries, what a rate limit looks like from the daemon's side, and
how you notice when it silently stops. From there we take RdioScanner, OpenMHz,
and Icecast (Part 10), then the grant webhook that fires per decoded grant
(Part 11).

## FAQ

**Why isn't spec-correct FEC just always on with no knob at all?**
It effectively is — the default runs the full chain for every protocol. The opt-out
exists only for pre-stripped capture files (DSD-FME `-r` dumps, OP25 fixtures) whose
FEC was already removed offline; running the decoder over them would fail. That's a
property of the capture, so the opt-out is per system, not global.

**Why does audio ship off by default when it's the obvious thing a scanner does?**
Because a headless daemon is the primary deployment, and audio-on-by-default would
warn or crash in a distroless/Alpine container with no sound device. WAV recording
is independent of playback, so the daemon still records everything with `audio.enabled`
off. Set it `true` on a desktop where you actually want speakers.

**A feed is in my config but nothing streams — is it broken?**
Check `enabled`. A feed with `enabled: false` is parsed but skipped, by design, so you
can stage credentials before going live. Also confirm the talkgroup isn't set
`stream: false`, which opts it out of every feed at once.

**How do I confirm a feature is really off without turning it on to test?**
Read `GET /api/v1/runtime` — it reports the effective value of every gate and is
sanitised of secrets so it's safe to scrape. The config loader also logs effective
values at startup, and the TUI Settings panel shows per-system FEC state. You never
need to probe a live endpoint to learn posture.

**What's the difference between opt-out, opt-in, and opt-in by nature?**
Opt-out means on by default with a per-system disable (FEC, clock recovery).
Opt-in means off by default because no zero-config value is correct (credentials,
CPU-cost, output-changing features). Opt-in by nature means the feature will never
be a default — patent-encumbered vocoders behind `-tags dvsi`, CI-only test tags —
regardless of how good it is.

## Series navigation

**Part 8 of 14** · ←
[Part 7: SDR Doctor & Preflight]({{ '/blog/deep-dives/running-it-for-real-07-sdr-doctor-preflight/' | relative_url }})
· Next →
[Part 9: Broadcast Backends I — Broadcastify]({{ '/blog/deep-dives/running-it-for-real-09-broadcast-backends-i/' | relative_url }})
