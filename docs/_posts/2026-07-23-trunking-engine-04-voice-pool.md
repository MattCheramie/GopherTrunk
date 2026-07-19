---
title: "Trunking Engine, Part 4: The Voice Pool — Allocating Scarce Radios"
description: How GopherTrunk's voice pool allocates a handful of Voice-role SDRs across many talkgroups, binds a call, retunes for handoffs, and detects when a wideband rig's tuning window doesn't cover the granted repeater.
category: deep-dives
keywords: voice pool, sdr allocation, resource pool pattern, voice sdr trunking, tuner interface retune, activecall bind, wideband voice tap coverage, gophertrunk voice pool, first fit allocation
tags: [trunking, go, sdr, resource-pool, voicepool, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 4
---

*Part 4 of **Trunking Engine**. [Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
followed a grant to the point where the engine has decided it needs a radio. This
post is about the thing that hands one out — the voice pool — and the awkward
reality that there are almost always more talkgroups than SDRs to follow them.*

> **TL;DR:** The `VoicePool` owns the set of `VoiceDevice`s (each a `Tuner` +
> serial) and the `ActiveCall` currently bound to each. Allocation is first-fit
> over the device list; `Bind` retunes a device to the grant's frequency and
> records the call; `Retune` follows an already-bound call to a new channel
> without ending it. The pool also knows that not every radio can reach every
> frequency — a wideband rig whose IQ window excludes the repeater is *incapable*,
> not just busy — and exposes that so the engine can tell a coverage gap from an
> all-radios-busy condition and emit the right one-time warning.

**Key takeaways**

- The pool is the classic **resource-pool pattern**: a fixed set of scarce
  devices, allocated on demand, returned on call end.
- A `VoiceDevice` is just a `Tuner` interface plus a serial — the pool retunes
  through `SetCenterFreq` and never touches SDR driver code.
- `Bind` starts a call and stamps a unique `CallID`; `Retune` continues one
  across a handoff, preserving identity.
- **Frequency capability** is first-class: `FindFreeForFrequency` and
  `HasCapableDevice` let the engine distinguish "no free radio" from "no radio
  can tune this" — the `noVoiceCoverageOnce` warning.

## Cheat sheet

| Method | Returns | Role |
|---|---|---|
| `FindFree()` | `*VoiceDevice` | first device with no active call |
| `FindFreeForFrequency(hz)` | `*VoiceDevice` | first free device that can tune `hz` |
| `HasCapableDevice(hz)` | `bool` | any device (busy or free) can tune `hz`? |
| `Bind(d, g, tg, now)` | `*ActiveCall, error` | retune `d`, record the call, stamp `CallID` |
| `Retune(serial, g, now)` | `error` | follow a bound call to a new frequency |
| `Release(serial)` | `*ActiveCall` | free the device, return the ended call |
| `Touch(serial, now)` | — | refresh `LastHeardAt` for the watchdog |
| `LowestPriorityActiveForFrequency(hz)` | `*ActiveCall` | preemption victim (Part 5) |

## In this post

- **What the pool holds** — `VoiceDevice`, `ActiveCall`, and the `Tuner` seam.
- **Allocation** — first-fit, and why order is a preference.
- **Bind and Retune** — starting a call vs following one.
- **Coverage** — the frequency-capable interface and the coverage-gap warning.

## What the pool holds

A voice device is deliberately thin:

```go
// internal/trunking/voicepool.go
type VoiceDevice struct {
    Tuner  Tuner
    Serial string
}

type Tuner interface {
    SetCenterFreq(hz uint32) error
}
```

That's the entire coupling between the engine and the radio: a one-method
interface. The pool retunes a device by calling `SetCenterFreq`; it never opens a
dongle, sets a gain, or reads IQ. A `VoiceDevice` can be a physical SDR or a
*virtual* voice tuner backed by a wideband DDC tap — the pool doesn't care, which
is what lets one wideband rig present several logical voice devices. This is the
same decoupling discipline from
[Part 1]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }}),
applied at the hardware edge.

The pool tracks which device is serving what:

```go
// internal/trunking/voicepool.go (shape)
type VoicePool struct {
    mu      sync.Mutex
    devices []*VoiceDevice
    active  map[string]*ActiveCall // by device serial
    callSeq atomic.Uint64          // hands out Grant.CallID
    // …reacquire callback for USB re-enumerate recovery
}
```

`active` maps a device serial to the `ActiveCall` bound to it. An `ActiveCall`
carries the `Grant`, the resolved `TalkGroup`, `StartedAt` and `LastHeardAt`
timestamps (the watchdog's fuel), and later-stamped quality figures
(`SignalDbFS`, `EVMPct`, `SNRDb`) the composer measures near end-of-call. The
whole struct is guarded by one mutex — the pool is safe for concurrent use
because the decode pipeline `Touch`es it from other goroutines while the engine
loop binds and releases.

## Allocation: first-fit, order is preference

Finding a free radio is a linear scan:

```go
// internal/trunking/voicepool.go
func (p *VoicePool) FindFree() *VoiceDevice {
    p.mu.Lock()
    defer p.mu.Unlock()
    for _, d := range p.devices {
        if _, busy := p.active[d.Serial]; !busy {
            return d
        }
    }
    return nil // every device is busy
}
```

First-fit, and the device *order* is meaningful: the daemon lists physical voice
SDRs before virtual wideband taps, so the pool prefers a dedicated radio and
falls back to a tap. `nil` means every device is busy — the engine's cue to look
at preemption
([Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})).

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A grant enters the voice pool, which scans its ordered device list first-fit: device one busy, device two free and capable is bound to the call, device three left free">
  <rect x="12" y="56" width="110" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="67" y="80" text-anchor="middle" fill="var(--accent)" font-size="12">grant</text>
  <line x1="122" y1="76" x2="168" y2="76" stroke="currentColor"/>
  <polygon points="168,72 178,76 168,80" fill="currentColor"/>
  <text x="245" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="10">ordered device list — first-fit</text>
  <rect x="188" y="34" width="120" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="248" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="10">SDR #1</text>
  <text x="248" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="9">busy</text>
  <rect x="188" y="76" width="120" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="248" y="93" text-anchor="middle" fill="var(--accent)" font-size="10">SDR #2</text>
  <text x="248" y="105" text-anchor="middle" fill="var(--accent)" font-size="9">free → BIND</text>
  <rect x="188" y="118" width="120" height="28" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="248" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="10">tap #3 — free</text>
  <line x1="308" y1="93" x2="360" y2="93" stroke="var(--accent)"/>
  <polygon points="360,89 370,93 360,97" fill="var(--accent)"/>
  <rect x="380" y="70" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="455" y="90" text-anchor="middle" fill="var(--accent)" font-size="11">Bind → ActiveCall</text>
  <text x="455" y="105" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SetCenterFreq + CallID</text>
  <line x1="530" y1="93" x2="576" y2="93" stroke="currentColor"/>
  <polygon points="576,89 586,93 576,97" fill="currentColor"/>
  <rect x="596" y="70" width="72" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="632" y="97" text-anchor="middle" fill="currentColor" font-size="10">tuned</text>
</svg>
<figcaption>First free device wins, in list order — physical SDRs before wideband taps. Binding retunes it and records the call.</figcaption>
</figure>

## Bind and Retune

`Bind` is where a grant becomes a call:

```go
// internal/trunking/voicepool.go (shape)
func (p *VoicePool) Bind(d *VoiceDevice, g Grant, tg *TalkGroup, now time.Time) (*ActiveCall, error) {
    // …reject if d is nil or already busy…
    if err := d.Tuner.SetCenterFreq(g.FrequencyHz); err != nil {
        // one reacquire+retry if the SDR pool wired SetReacquire (USB re-enumerate)
        return nil, err
    }
    g.CallID = p.callSeq.Add(1) // fresh, process-unique — fences audio bleed
    ac := &ActiveCall{Device: d, Grant: g, Talkgroup: tg, StartedAt: now, LastHeardAt: now}
    p.active[d.Serial] = ac
    return ac, nil
}
```

Three things happen: the device is retuned via the one-method `Tuner`, a fresh
`CallID` is stamped so the voice chain and recorder can reject stale frames from
the previous call on a reused tap serial
([Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})),
and the `ActiveCall` is recorded in `active`. If the initial `SetCenterFreq`
fails and the daemon wired a `SetReacquire` callback, the pool asks the SDR pool
to re-open the serial (recovering a device whose USB handle died while idle) and
retries the tune once — issue #345.

`Retune` is the *handoff* path — following a live call to a new channel without
ending it:

```go
// internal/trunking/voicepool.go (shape)
func (p *VoicePool) Retune(serial string, g Grant, now time.Time) error {
    ac := p.active[serial]           // must already be bound
    d := ac.Device
    if err := d.Tuner.SetCenterFreq(g.FrequencyHz); err != nil { /* reacquire+retry */ }
    g.CallID = ac.Grant.CallID       // preserve identity across the handoff
    ac.Grant = g                     // replace the grant, keep StartedAt
    ac.LastHeardAt = now
    return nil
}
```

The distinction matters: `Bind` starts a new `StartedAt` and a new `CallID`;
`Retune` preserves both, so a call that moves across channels stays *one* call
with one duration and one identity. The engine reaches for `Retune` from the
duplicate-grant guard's "same call, new frequency" branch. The returned pointer
is the *same* `ActiveCall` instance the engine already mirrors, so the updated
grant is visible with no extra bookkeeping.

`Release(serial)` deletes the entry and returns the freed `ActiveCall`;
`Touch(serial, now)` just advances `LastHeardAt` and is called from the decode
pipeline every time a frame lands, which is how the watchdog in
[Part 12]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }})
knows a call is still alive.

## Coverage: not every radio can reach every frequency

Here is the subtlety that makes this pool more than a free-list. A physical SDR
can tune anywhere in its range, but a *virtual* voice tuner backed by a wideband
DDC tap can only follow grants that fall inside the wideband dongle's IQ window.
A grant for a repeater at the band edge might land outside every tap's window —
the radios exist, they're even free, but none of them can *reach* the frequency.

The pool models this with an optional interface:

```go
// internal/trunking/voicepool.go
type FrequencyChecker interface {
    CanTune(hz uint32) bool
}
```

A device whose `Tuner` implements `FrequencyChecker` is consulted; a physical SDR
that doesn't is treated as universally tunable. Allocation then has a
frequency-aware form:

```go
// internal/trunking/voicepool.go
func (p *VoicePool) FindFreeForFrequency(hz uint32) *VoiceDevice {
    for _, d := range p.devices {
        if _, busy := p.active[d.Serial]; busy { continue }
        if fc, ok := d.Tuner.(FrequencyChecker); ok && !fc.CanTune(hz) { continue }
        return d
    }
    return nil
}
```

And a separate `HasCapableDevice(hz)` reports whether *any* device — busy or free
— could tune `hz` at all. Those two let the engine tell three situations apart
when it can't place a grant, and give the operator an actionable message for
each:

| Situation | Detected by | Message |
|---|---|---|
| No voice devices configured | `len(Devices()) == 0` | add a `role: voice` SDR / wideband tap |
| Devices exist, none covers `hz` | `!HasCapableDevice(hz)` | **coverage gap** — widen sample rate / adjust center |
| Devices cover `hz`, all busy | preemption falls through | no free device (Part 5) |

### The problem we hit

The middle case used to masquerade as an engine bug. A wideband-only rig whose IQ
window excluded a repeater would drop every grant for that repeater, and the log
read like the pool was broken. It isn't — it's a *coverage gap*, an RF-planning
problem, and it deserves its own message exactly once:

```go
// internal/trunking/engine.go (shape) — after allocation + preemption both fail
if !e.pool.HasCapableDevice(g.FrequencyHz) {
    e.noVoiceCoverageOnce.Do(func() {
        e.log.Warn("voice grant frequency outside every voice device's tuning " +
            "window; widen sdr.sample_rate / adjust center_freq_hz, or add a " +
            "role: voice SDR", "grant", g.String())
    })
    return
}
```

The `sync.Once` guard (`noVoiceCoverageOnce`) logs the warning loudly the first
time and then drops subsequent out-of-window grants at DEBUG, so a band-edge
talkgroup that keys up every few seconds doesn't flood the log with the same
diagnosis. Preemption is also coverage-aware —
`LowestPriorityActiveForFrequency(hz)` only considers victims *on devices that can
tune `hz`*, because ending a call to free a radio that then can't bind the
incoming grant would be pure loss. That method is the handoff into
[Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 170" width="680" height="170" role="img" aria-label="A wideband SDR's IQ window covers a band of channels; a grant for a repeater inside the window can be tuned by the virtual tap, but a grant for a repeater at the band edge outside the window is a coverage gap that triggers the noVoiceCoverageOnce warning">
  <rect x="60" y="40" width="360" height="44" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="240" y="30" text-anchor="middle" fill="var(--accent)" font-size="11">wideband IQ window (one dongle)</text>
  <line x1="120" y1="40" x2="120" y2="84" stroke="var(--fg-muted)"/>
  <line x1="200" y1="40" x2="200" y2="84" stroke="var(--fg-muted)"/>
  <line x1="280" y1="40" x2="280" y2="84" stroke="var(--fg-muted)"/>
  <line x1="360" y1="40" x2="360" y2="84" stroke="var(--fg-muted)"/>
  <text x="240" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">channels inside window — CanTune = true</text>
  <circle cx="200" cy="106" r="6" fill="var(--accent)"/>
  <text x="200" y="128" text-anchor="middle" fill="var(--accent)" font-size="10">grant inside → tap binds</text>
  <circle cx="540" cy="106" r="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="540" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="10">repeater at band edge</text>
  <line x1="420" y1="62" x2="540" y2="100" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="560" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="10">grant outside → coverage gap</text>
  <text x="560" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="9">noVoiceCoverageOnce warns once</text>
</svg>
<figcaption>A virtual voice tap can only follow grants inside its dongle's IQ window. A grant outside every window is a coverage gap — an RF-planning problem, warned about once, not an engine bug.</figcaption>
</figure>

## Where this goes next

The pool hands out radios until it runs out. When `FindFreeForFrequency` returns
`nil` and there *are* capable-but-busy devices, the engine has to decide whether
the new grant is worth kicking an active call off its radio.
[Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
is that policy — priority ranking, strict-higher preemption, and the thrash it's
designed to avoid. Later,
[Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})
returns to the pool for the encrypted-release machinery
(`ArmEncryptedRelease`/`EncryptedReleasesDue`) that frees a tuner from a call
that turns out to be encrypted.

## FAQ

**What is a `VoiceDevice`?**
A `Tuner` interface (one method, `SetCenterFreq`) plus a serial string. It can be
a physical SDR or a virtual voice tuner backed by a wideband DDC tap — the pool
treats them uniformly and never touches driver code.

**What's the difference between `Bind` and `Retune`?**
`Bind` starts a *new* call: it stamps a fresh `StartedAt` and a process-unique
`CallID` and records a new `ActiveCall`. `Retune` follows an *existing* call to a
new frequency, preserving `StartedAt` and `CallID` so the call keeps one identity
and one duration across a handoff.

**How does the pool handle more talkgroups than radios?**
The pool itself just reports "no free device" (`FindFree`/`FindFreeForFrequency`
return `nil`). The engine then consults `LowestPriorityActiveForFrequency` and
decides whether to preempt — the subject of Part 5. The pool provides the
capability-aware victim search; the priority policy lives above it.

**Why can a free radio still fail to take a grant?**
Because a wideband voice tap can only tune frequencies inside its dongle's IQ
window. `FindFreeForFrequency` skips a free-but-incapable tap, and if no capable
device exists at all, `HasCapableDevice` is false and the engine logs the
`noVoiceCoverageOnce` coverage-gap warning rather than mislabelling it a bug.

**What does `CallID` protect against?**
Cross-call audio bleed. When a tap serial is immediately reused for the next
call, a straggling frame from the previous call could otherwise land in the new
recording. `Bind` stamps a unique `CallID` the voice chain tags frames with, and
the recorder rejects any frame whose `CallID` doesn't match the open session.

## Series navigation

**Part 4 of 12** · ←
[Part 3: Grants]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
· Next →
[Part 5: Priority & Preemption]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
