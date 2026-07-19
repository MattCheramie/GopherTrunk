---
title: "Trunking Engine, Part 11: Encrypted-Mode Handling — Follow, Metadata, or Ignore"
description: How GopherTrunk decides what to do with an encrypted trunked call — hold the tuner, grab metadata and release, or ignore it — with a per-system policy, a configured-key exemption, and a metadata-follow window that frees scarce voice SDRs.
category: deep-dives
keywords: encrypted call handling, p25 encryption, algid clear 0x80, metadata follow, tuner starvation, voice sdr allocation, encrypted_calls config, talker alias, gophertrunk trunking, key id exemption
tags: [trunking, go, p25, encryption, event-bus, architecture]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 11
---

*Part 11 of **Trunking Engine**, a 12-part deep dive into the "brain" of
GopherTrunk. This is a current-work post — issue #711 landed the machinery it
describes. The question is deceptively simple: when a call you've tuned turns out
to be encrypted, and you can't decrypt it, should you keep a scarce radio parked
on silence?*

> **TL;DR:** Encrypted calls waste voice SDRs. GopherTrunk gives each system a
> policy: **`EncryptedFollow`** (default, legacy — hold the tuner for the whole
> encrypted call), **`EncryptedMetadata`** (follow briefly to grab talker alias,
> source RID, and encryption sync, then release the voice SDR
> `metadata_follow_ms` after the call is known encrypted — default 1.5 s), or
> **`EncryptedIgnore`** (never tie up a tuner on an encrypted call). A call whose
> `KeyID` the operator has a configured key for is exempt and always followed.
> "Encrypted" means a P25 ALGID other than clear — `algorithmClear = 0x80`.

**Key takeaways**

- The motivation is **tuner starvation**: voice SDRs are scarce, and a radio
  parked on an undecodable encrypted call is a radio not following a clear one.
- Three modes, chosen **per system**, so you can run `metadata` on one system and
  `follow` or `ignore` on another. The zero value is `follow`, so old configs are
  unchanged.
- **Metadata mode** trades a brief hold for identity: it captures the alias / RID /
  enc-sync, then releases — and releases *early* the moment the alias completes.
- The **configured-key exemption** always wins: if you hold a key for the call's
  `KeyID`, it's followed no matter the mode, because you intend to decode it.

## Cheat sheet

| Mode / constant | Where it lives | Behaviour |
|---|---|---|
| `EncryptedFollow` (0) | `encryptedmode.go` | hold the voice SDR for the full call — legacy default |
| `EncryptedMetadata` | `encryptedmode.go` | follow briefly, release after `metadata_follow_ms` |
| `EncryptedIgnore` | `encryptedmode.go` | never allocate / drop the tuner the moment enc is known |
| `algorithmClear = 0x80` | `engine.go` | the P25 ALGID a clear call advertises; anything else is encrypted |
| `keyConfigured(system, keyID)` | `engine.go` | true when the operator holds a key → always follow |
| `applyEncryptedPolicy(...)` | `engine.go` | enforces the policy once a call is known encrypted |

## In this post

- **Why encrypted calls are a resource problem**, not just a listening one.
- **The three modes** and how a per-system policy is configured.
- **When the engine learns a call is encrypted** — up front vs mid-call.
- **The metadata window and the key exemption** — and how they shaped the Go.

## The problem: a radio parked on silence

Voice SDRs are the scarcest resource in the whole system. Most rigs have one or
two, and [Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
was entirely about rationing them when more talkgroups are active than radios to
follow. An encrypted call you can't decrypt is the worst possible use of one:
the tuner is bound, the recorder is running, and the output is noise. On a busy
encrypted system — a lot of public-safety traffic is encrypted now — the legacy
behaviour of following every call could leave your only voice SDR camped on
undecodable audio while clear calls you *can* hear go unrecorded. That is **tuner
starvation**, and issue #711 exists to fix it.

The catch is that an encrypted call isn't worthless. Even when the *audio* is
opaque, the traffic channel still carries metadata in the clear — the talker
alias (the radio's display name), the source RID, and the encryption sync
(ALGID / KID / message indicator). That's real intelligence for the affiliation
roster and key-discovery tooling. So the design isn't binary; it's a spectrum from
"keep everything" to "waste nothing," with a middle mode that grabs the metadata
and then lets the tuner go.

<figure class="lab-figure">
<svg viewBox="0 0 680 170" width="680" height="170" role="img" aria-label="A decision flow: a call known encrypted checks the configured-key exemption first, then branches by per-system mode into follow, metadata release after a window, or ignore">
  <rect x="12" y="66" width="120" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="72" y="82" text-anchor="middle" fill="currentColor" font-size="11">call known</text>
  <text x="72" y="97" text-anchor="middle" fill="var(--fg-muted)" font-size="10">encrypted</text>
  <line x1="132" y1="86" x2="180" y2="86" stroke="currentColor"/>
  <polygon points="180,82 190,86 180,90" fill="currentColor"/>
  <rect x="190" y="60" width="120" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="250" y="80" text-anchor="middle" fill="var(--accent)" font-size="11">have key for</text>
  <text x="250" y="94" text-anchor="middle" fill="var(--accent)" font-size="11">this KeyID?</text>
  <text x="250" y="107" text-anchor="middle" fill="var(--fg-muted)" font-size="9">keyConfigured</text>
  <line x1="310" y1="70" x2="360" y2="44" stroke="var(--accent)"/>
  <polygon points="360,40 370,44 359,48" fill="var(--accent)"/>
  <text x="335" y="38" text-anchor="middle" fill="var(--accent)" font-size="9">yes</text>
  <rect x="370" y="26" width="150" height="30" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="445" y="45" text-anchor="middle" fill="var(--accent)" font-size="10">always follow (exempt)</text>
  <line x1="310" y1="100" x2="360" y2="100" stroke="currentColor"/>
  <polygon points="360,96 370,100 360,104" fill="currentColor"/>
  <text x="335" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no → per-system mode</text>
  <rect x="370" y="70" width="150" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="445" y="87" text-anchor="middle" fill="var(--fg-muted)" font-size="10">follow: hold full call</text>
  <rect x="370" y="100" width="150" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="445" y="117" text-anchor="middle" fill="var(--fg-muted)" font-size="10">metadata: arm release</text>
  <rect x="370" y="130" width="150" height="26" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="445" y="147" text-anchor="middle" fill="var(--fg-muted)" font-size="10">ignore: end call now</text>
  <line x1="520" y1="113" x2="566" y2="113" stroke="var(--fg-muted)"/>
  <polygon points="566,109 576,113 566,117" fill="var(--fg-muted)"/>
  <rect x="576" y="98" width="94" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="623" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">release after</text>
  <text x="623" y="123" text-anchor="middle" fill="var(--fg-muted)" font-size="9">follow window</text>
</svg>
<figcaption>The key exemption is checked before the mode: a decryptable call is always followed. Otherwise the per-system policy decides between holding, grabbing metadata, or dropping.</figcaption>
</figure>

## Three modes, per system

The policy is a small enum with a wire form for config, REST, and the TUI:

```go
// internal/trunking/encryptedmode.go (shape)
type EncryptedMode uint8

const (
    EncryptedFollow   EncryptedMode = iota // hold the tuner, full call (legacy default)
    EncryptedMetadata                      // follow briefly, then release
    EncryptedIgnore                        // never tie up a tuner
)

// Empty or unknown config maps to follow, so a typo never silently
// stops following calls.
func ParseEncryptedMode(s string) EncryptedMode { /* "metadata"|"ignore"|else follow */ }
```

Two things about the defaults are deliberate. `EncryptedFollow` is the **zero
value**, so a system absent from the config map behaves exactly as GopherTrunk
always did — pre-existing configs see no change. And `ParseEncryptedMode` maps
*unknown* strings to `follow` too, so a typo in `encrypted_calls` can never
silently stop following calls — the failure mode is "kept recording," never "went
dark." Modes are per system (`trunking.systems[].encrypted_calls`), read-only after
`NewEngine`, so an operator can run `metadata` on one system and `follow` or
`ignore` on another with no locking on the hot path.

## When the engine learns a call is encrypted

Encryption is discovered at two different moments, and the engine handles both.

**Up front**, a P25 Phase 2 grant carries the encryption flag, so `HandleGrant`
can drop an already-encrypted grant *before* allocating a tuner — but only in
`ignore` mode, and only when the operator has no keys for the system:

```go
// internal/trunking/engine.go (shape) — inside HandleGrant
if g.Encrypted && !g.Emergency && e.encModeFor(g.System) == EncryptedIgnore &&
    !e.keyConfigured(g.System, g.KeyID) && !e.systemHasKeys(g.System) {
    return // never tie up a voice SDR on this grant
}
```

Emergency grants bypass the policy entirely, following the existing lockout/scan
precedent. And a system the operator *has* keys for is left to the in-call
handlers, which know the actual `KeyID` and can exempt a decryptable call — dropping
it here would discard a call the operator wants to capture.

**Mid-call** is the P25 Phase 1 case, where the grant looks clear and encryption
only surfaces once the traffic channel is up. The composer publishes a
`KindCallEncryption` (from an in-call Encryption Sync) or a `KindCallSourceUpdate`
carrying the discovered ALGID/KID, and both handlers funnel into one enforcement
point after enriching and republishing the event for the live view:

```go
// internal/trunking/engine.go (shape)
e.applyEncryptedPolicy(c.DeviceSerial, g, c.AlgorithmID != algorithmClear)
```

That `c.AlgorithmID != algorithmClear` is the whole definition of "encrypted":
`algorithmClear` is `0x80`, the ALGID a clear P25 call advertises. Anything else is
an encryption algorithm. The constant is kept local to the package to avoid a
radio-package import, mirroring `p25.AlgorithmClear`.

## The metadata window and the key exemption

`applyEncryptedPolicy` is the single place the decision is made once a call is
known encrypted, and it reads top to bottom as the priority order:

```go
// internal/trunking/engine.go (shape)
func (e *Engine) applyEncryptedPolicy(serial string, g Grant, encrypted bool) {
    if !encrypted {
        return
    }
    if e.keyConfigured(g.System, g.KeyID) {
        e.pool.DisarmEncryptedRelease(serial) // decryptable → always follow
        return
    }
    switch e.encModeFor(g.System) {
    case EncryptedIgnore:
        e.endCall(ac, EndReasonEncrypted) // release the tuner now
    case EncryptedMetadata:
        e.pool.ArmEncryptedRelease(serial, e.now().Add(e.encMetadataFollowFor(g.System)))
    }
    // EncryptedFollow: do nothing — keep holding the tuner.
}
```

The **configured-key exemption** is checked first and unconditionally: if the
operator supplied a key whose ID matches the call's `KeyID`, the call is
decryptable — the operator intends to capture and decode it — so any pending
metadata release is *cancelled* and the call is followed to the end, whatever the
mode says. That's `keyConfigured` consulting the per-system `configuredKeys` set
built at startup.

**Metadata mode** doesn't end the call; it *arms* a release. `ArmEncryptedRelease`
stamps a deadline of `now + metadata_follow_ms` (default 1.5 s — long enough for a
Phase 2 talker-alias reassembly plus a couple of MAC PDU repeats, short enough to
free the tuner fast). The [watchdog]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }})
tick reaps any call whose window has elapsed, ending it with `EndReasonEncrypted`.
There's an early-out too: `handleTalkerAlias` releases an armed call the *instant*
its alias fully reassembles — the reason we held the tuner is already satisfied, so
there's no point waiting out the rest of the window.

<figure class="lab-figure">
<svg viewBox="0 0 680 150" width="680" height="150" role="img" aria-label="A timeline of a metadata-mode encrypted call: the call starts clear-looking, encryption is discovered, a follow window is armed, metadata is captured, and the tuner is released either when the alias completes or when the window elapses">
  <line x1="30" y1="60" x2="650" y2="60" stroke="currentColor"/>
  <line x1="90" y1="54" x2="90" y2="66" stroke="currentColor"/>
  <text x="90" y="44" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grant / call start</text>
  <line x1="230" y1="54" x2="230" y2="66" stroke="var(--accent)"/>
  <text x="230" y="44" text-anchor="middle" fill="var(--accent)" font-size="9">enc discovered</text>
  <text x="230" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="9">arm release: +metadata_follow_ms</text>
  <rect x="230" y="94" width="220" height="20" rx="4" fill="none" stroke="var(--fg-muted)"/>
  <text x="340" y="108" text-anchor="middle" fill="var(--fg-muted)" font-size="9">capture alias · RID · enc sync</text>
  <line x1="360" y1="54" x2="360" y2="66" stroke="var(--accent)"/>
  <text x="360" y="44" text-anchor="middle" fill="var(--accent)" font-size="9">alias done → release early</text>
  <line x1="450" y1="54" x2="450" y2="66" stroke="currentColor"/>
  <text x="450" y="128" text-anchor="middle" fill="var(--fg-muted)" font-size="9">or window elapses → watchdog reaps</text>
  <line x1="450" y1="66" x2="450" y2="120" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <line x1="450" y1="60" x2="520" y2="60" stroke="var(--accent)" stroke-width="2"/>
  <polygon points="520,56 530,60 520,64" fill="var(--accent)"/>
  <text x="580" y="56" text-anchor="middle" fill="var(--accent)" font-size="10">tuner freed</text>
  <text x="580" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">EndReasonEncrypted</text>
</svg>
<figcaption>Metadata mode: hold just long enough to capture identity, then release — early if the talker alias completes first, otherwise when the follow window elapses.</figcaption>
</figure>

### How that principle shaped the Go code

The design principle is **separate discovery from policy**. Encryption can be
learned from three different places — a grant flag, an in-call encryption sync, a
source update — but every one of them reduces to a single boolean handed to one
`applyEncryptedPolicy`. The handlers don't each re-implement "should I drop this?";
they enrich, republish for the live view, and call the one enforcement point. And
enforcement itself is *lazy*: metadata mode doesn't spin a timer per call, it
stamps a deadline the watchdog already sweeps once a tick, so the encrypted-release
path costs nothing on the hot loop. The single-writer rule holds throughout —
arming, disarming, and reaping all happen on the engine's one goroutine, so there's
no race between "arm a release" and "the alias just completed."

## Where this goes next

[Part 12]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }})
closes the series on the machinery this post leaned on: control-channel hunting and
backoff, the 500 ms watchdog that reaps both the encrypted-release deadlines here
and ordinary silent calls, and how the whole engine — encrypted-mode policy
included — is tested by publishing synthetic grants to a fake bus with no radio in
sight. For the crypto background, see the
[encryption reference]({{ '/reference/encryption/' | relative_url }}).

## FAQ

**What does "encrypted" mean to the engine?**
A P25 call advertises an Algorithm ID; the clear (unencrypted) value is
`algorithmClear = 0x80`. Any other ALGID means the call is encrypted. DMR and other
protocols surface an equivalent encrypted flag on the grant.

**What's the difference between metadata mode and ignore mode?**
`ignore` never ties up a voice SDR on an encrypted call — it drops the grant up
front when encryption is known, or releases the tuner the instant it's discovered
mid-call. `metadata` follows *briefly* first, long enough to capture the talker
alias, source RID, and encryption sync, then releases after the follow window
(default 1.5 s).

**I have the key for a system — will it still get dropped?**
No. The configured-key exemption is checked before the mode: a call whose `KeyID`
matches a key you supplied is always followed to the end, regardless of whether the
system's mode is `metadata` or `ignore`, because you intend to decode it.

**Will enabling this change my existing setup?**
No. `EncryptedFollow` is the default and the zero value, and an unknown config
string also maps to `follow`. A system you don't configure for encrypted-call
handling behaves exactly as before, holding the tuner for the full call.

## Series navigation

**Part 11 of 12** · ←
[Part 10: Sites, Topology & Roaming]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
· Next →
[Part 12: CC Hunting, the Watchdog & Testing]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }})
