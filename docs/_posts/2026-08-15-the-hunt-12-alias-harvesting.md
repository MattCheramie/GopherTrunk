---
title: "The Hunt, Part 12: Alias Harvesting — Following Traffic for Talker Aliases"
description: How GopherTrunk harvests P25 Phase 2 talker aliases off a traffic channel's signalling stream without recording voice — a signalling-only DDC tap per in-window grant, the shared MAC dispatcher the voice composer also uses, RS-parity gating, and harvesting the Motorola alias ciphertext for cryptanalysis.
category: deep-dives
keywords: p25 talker alias, phase 2 signalling decode, facch-s alias, signalling follow, mac pdu dispatch, alias harvesting without voice, motorola alias cipher, rs parity gate, gophertrunk the hunt
tags: [the-hunt, p25, alias, signalling, crypto, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 12
---

*Part 12 of **The Hunt**. Part 11 left us with a system that has named nothing
smaller than itself: talkgroups are decimals, units are decimals. But there is
one genuine, human name that travels over the air — the **talker alias**, the
display name a radio broadcasts for itself. This post is about harvesting those
aliases off our system's traffic channels, and doing it the efficient way: by
decoding the signalling that rides each call, **without** ever recording the
voice. It also runs straight into the series' oldest ghost — the Motorola alias
cipher we still can't read.*

> **TL;DR:** On a busy P25 Phase 2 system most grants never get a voice tuner, and
> encrypted calls are torn down before hangtime — so the talker-alias decode
> wired *inside* the voice composer almost never runs (issue #376). `internal/sigfollow`
> fixes that by **decoupling alias capture from voice following**: a `Manager`
> subscribes to the bus, and for each in-window Phase 2 grant it borrows a
> **signalling-only** DDC tap on the wideband IQ broker, decodes the traffic
> channel's MAC PDUs through the **same** `MACDispatcher` the voice composer uses,
> and publishes any completed talker alias. Two SDRs, no voice pool, aliases
> harvested off the signalling stream — mirroring how SDRTrunk does it. And when
> the alias is the proprietary Motorola FACCH-S kind, it harvests the
> **ciphertext** as cryptanalysis ground truth.

**Key takeaways**

- **Alias capture is decoupled from voice.** A signalling follow needs only to
  decode the MAC signalling on the traffic channel — no vocoder, no recording — so
  it runs on grants the voice pool never covers.
- **One dispatcher, two callers.** `MACDispatcher` is the exact decode the voice
  composer runs; the follower reuses it, so the two paths never diverge on what an
  alias means.
- **RS parity gates trust.** A mis-framed superframe decodes random bytes that
  almost never verify, so only an RS-valid PDU is trusted to set a call's source
  — a wrong RID is worse than an absent one.
- **The Motorola alias is still unread — so harvest the ciphertext.** The
  proprietary alias cipher is unverified, so the decoded name is usually empty;
  the follower logs the reassembled ciphertext (paired with source RID and
  talkgroup) as the corpus the cryptanalysis needs.

## Cheat sheet

| Piece | What it does | Where it lives |
|---|---|---|
| `Manager` | one signalling follow per in-window Phase 2 grant | `internal/sigfollow/manager.go` |
| `follow` | tune a tap, decode MAC signalling, idle out | `internal/sigfollow/manager.go` |
| `MACDispatcher` | decode MAC PDUs, publish talker aliases | `internal/sigfollow/dispatcher.go` |
| `allocTunerLocked` | claim a free signalling tap covering the grant | `internal/sigfollow/manager.go` |
| `onVoiceStart` / `onVoiceEnd` | yield to a voice chain that harvests itself | `internal/sigfollow/manager.go` |
| `completeMotorolaAlias` | log the alias ciphertext for cryptanalysis | `internal/sigfollow/dispatcher.go` |

## In this post

- **The alias that voice-following misses** — why the composer's decode rarely runs.
- **A signalling-only follow** — a tap, a dispatcher, an idle timer.
- **The shared dispatcher** — one decode for the composer and the follower.
- **RS parity as a trust gate** — why a wrong RID is worse than none.
- **The ciphertext we can't read** — harvesting Mercury's corpus.

## The alias that voice-following misses

A P25 talker alias rides the traffic channel, interleaved with the voice as MAC
signalling (FACCH-S). The obvious place to decode it is inside the voice
composer — you're already demodulating that channel to record the call, so read
the alias while you're there. And that works, when it runs. The problem is how
*often* it runs. The package doc is blunt about it:

```go
// internal/sigfollow/dispatcher.go (shape) — package doc
// On a busy multi-site Phase 2 system most grants never get a voice tuner (the
// voice pool can't cover the full voice spread), and encrypted calls are torn
// down before hangtime — so the talker-alias FACCH-S decode wired inside the
// voice composer almost never runs (issue #376). SDRTrunk harvests these aliases
// from the traffic channel's signalling stream with just two SDRs, without
// following the call as voice. This package mirrors that.
```

Two failure modes stack. First, the voice pool is finite: on a busy system there
are more simultaneous calls than voice tuners, so most grants are never followed
as voice at all, and their aliases are never seen. Second — and worse for the
interesting traffic — **encrypted** calls are the ones most likely to be torn
down before hangtime, and hangtime is exactly when the alias burst arrives. The
result is that the aliases you most want ride right past a voice-only decoder.

The fix is to stop coupling the two. You don't need to *record* a call to *read*
its signalling — you only need to demodulate the channel's MAC stream, which is
far cheaper than running a vocoder. So GopherTrunk spins up a decode that does
only that.

## A signalling-only follow

`sigfollow.Manager` subscribes to the event bus and reacts to grants. For each
in-window Phase 2 grant that isn't already being followed and isn't covered by a
voice call, it borrows a **signalling tap** — a standalone wideband DDC tap that
is *not* registered in the voice pool — and starts a follow:

```go
// internal/sigfollow/manager.go (shape) — onGrant
func (m *Manager) onGrant(ctx context.Context, g trunking.Grant) {
    if g.Protocol != "p25-phase2" || g.FrequencyHz == 0 {
        return
    }
    m.mu.Lock()
    if _, following := m.active[g.FrequencyHz]; following {
        m.mu.Unlock(); return
    }
    if m.voiceFreqs[g.FrequencyHz] > 0 {
        m.mu.Unlock(); return // a voice chain already harvests aliases here
    }
    vt := m.allocTunerLocked(g.FrequencyHz) // free tap whose window covers the grant
    if vt == nil {
        m.mu.Unlock(); return // outside every tap's window, or all taps busy
    }
    fctx, cancel := context.WithCancel(ctx)
    m.active[g.FrequencyHz] = cancel
    m.mu.Unlock()
    m.wg.Add(1)
    go m.follow(fctx, vt, g)
}
```

The `follow` goroutine tunes the tap to the grant frequency, opens its IQ stream,
builds a Phase 2 receiver, and pipes decoded superframes into a `MACDispatcher`.
Crucially, it carries the grant's own Phase 2 FEC configuration — trellis, RS,
interleave, scrambler mode, seed, soft-decision — so it decodes the traffic
channel with exactly the same parameters the control channel stamped onto the
grant:

```go
// internal/sigfollow/manager.go (shape) — follow
macCfg := p25p2.MACDecodeConfig{
    Trellis: p25p2.TrellisMode(g.P25Phase2Decode.Trellis),
    RS:      p25p2.RSMode(g.P25Phase2Decode.RS),
    // …Interleave, Scrambler, Seed, SoftDecision, Equalizer — all from the grant
}
dispatcher := NewMACDispatcher(MACDispatcherOptions{
    Bus: m.bus, System: g.System, Serial: vt.Serial(), LogPrefix: "sigfollow",
    // No OnCallSource / OnCallEncryption: a signalling follow has no bound call to
    // backfill, so it harvests talker aliases only.
})
```

The follow ends on one of three bounds: an **idle timeout** (4 s of no decoded
superframe — the call dropped, and aliases arrive within a few seconds of voice
end, so a short idle covers the gap), a **max duration** cap (30 s, so a stuck
decode can't pin a tap forever), or context cancellation. That last one is how the
system yields gracefully: if a voice chain later starts recording the same
frequency, `onVoiceStart` cancels the redundant signalling follow because the
voice chain harvests aliases itself. No frequency is ever decoded twice for the
same reason.

<figure class="lab-figure">
<svg viewBox="0 0 660 210" width="660" height="210" role="img" aria-label="The signalling follow flow. A Phase 2 grant on the event bus triggers the manager to allocate a free signalling DDC tap whose window covers the grant frequency. The follow goroutine tunes the tap, decodes the traffic channel's MAC PDUs through the shared MAC dispatcher, and publishes any completed talker alias back onto the event bus. A voice-call-start event on the same frequency cancels the redundant follow.">
  <rect x="10" y="88" width="110" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="65" y="104" text-anchor="middle" fill="var(--accent)" font-size="10">grant (bus)</text>
  <text x="65" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">p25-phase2</text>
  <line x1="120" y1="108" x2="150" y2="108" stroke="currentColor"/><polygon points="150,104 160,108 150,112" fill="currentColor"/>
  <rect x="160" y="86" width="120" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="220" y="102" text-anchor="middle" fill="currentColor" font-size="10">alloc signalling tap</text>
  <text x="220" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">covers grant freq?</text>
  <line x1="280" y1="108" x2="310" y2="108" stroke="currentColor"/><polygon points="310,104 320,108 310,112" fill="currentColor"/>
  <rect x="320" y="80" width="130" height="56" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="385" y="98" text-anchor="middle" fill="var(--accent)" font-size="10">follow: rx → MAC</text>
  <text x="385" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no vocoder</text>
  <text x="385" y="124" text-anchor="middle" fill="var(--fg-muted)" font-size="9">idle 4s · cap 30s</text>
  <line x1="450" y1="108" x2="482" y2="108" stroke="currentColor"/><polygon points="482,104 492,108 482,112" fill="currentColor"/>
  <rect x="492" y="86" width="150" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="567" y="104" text-anchor="middle" fill="var(--accent)" font-size="10">talker alias (bus)</text>
  <text x="567" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="9">+ alias ciphertext log</text>
  <rect x="320" y="158" width="130" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="385" y="177" text-anchor="middle" fill="var(--fg-muted)" font-size="9">voice start → cancel</text>
  <line x1="385" y1="158" x2="385" y2="136" stroke="var(--fg-muted)" stroke-dasharray="4 3"/><polygon points="381,140 385,132 389,140" fill="var(--fg-muted)"/>
</svg>
<figcaption>A Phase 2 grant borrows a signalling-only tap, decodes just the MAC stream, and publishes any alias — yielding the moment a voice chain takes over the same frequency.</figcaption>
</figure>

## The shared dispatcher

The follow doesn't reimplement alias decoding — it can't be allowed to, because
then the aliases harvested off a traffic channel might disagree with the ones the
voice composer reads on the same channel. Both drive the **same** `MACDispatcher`,
which decodes every MAC PDU in a superframe and routes it: a group-voice PDU
carries the source RID, a MAC-PTT slot carries the encryption sync, and the alias
fragments reassemble into a name. The follower simply leaves the call-bound hooks
(`OnCallSource`, `OnCallEncryption`) nil, because a signalling follow has no bound
call to backfill — it publishes talker aliases and nothing else. One decode, two
callers, guaranteed agreement.

### How that principle shaped the Go code

- **The dispatcher is stateful per channel, so construct one per follow.** The
  alias assemblers buffer fragments per source unit for the life of the channel;
  a fresh dispatcher per follow means a new call never inherits a stale
  half-alias.
- **Hooks, not branches, carry the difference.** The composer wires
  `OnCallSource`/`OnCallEncryption` to its engine-backfill publishers; the
  follower passes nil. The *decode* is identical — only what happens to a
  call-bound PDU differs, and that's a function pointer, not a code path.
- **The follower borrows taps, it doesn't own SDRs.** Signalling taps are DDC taps
  on the wideband IQ broker — the same virtual-tuner mechanism the wideband engine
  uses — so alias harvesting rides existing hardware, not a dedicated radio.

## RS parity as a trust gate

There is a sharp edge here that the code handles carefully. A mis-framed Phase 2
superframe doesn't fail loudly — it decodes a stream of *random* bytes, and an
opcode byte that happens to land on the group-voice value would inject a
plausible-but-wrong source RID. That RID is indistinguishable downstream from a
real one, and it would poison the completed-call metadata. The only signal that
separates a genuine PDU from garbage is the outer Reed–Solomon parity, so the
dispatcher gates on it:

```go
// internal/sigfollow/dispatcher.go (shape) — Dispatch, group-voice PDU
if u, ok := pdu.AsGroupVoiceChannelUser(); ok {
    // The outer RS parity is the only signal that separates a genuine
    // GROUP_VOICE_CHANNEL_USER from garbage (the opcode alone can't), so only an
    // RS-verified PDU is trusted to set the call's source. A wrong RID is worse
    // than an absent one (issue #915).
    if dec.RSValid && d.onCallSource != nil {
        d.onCallSource(u)
    }
    continue
}
```

`Dispatch` returns both a decoded-PDU count and an RS-valid count, and a caller
feeds that ratio into a per-call census: on a correctly framed, descrambled
channel nearly every PDU verifies, whereas `rs_valid=false` across the board is
the fingerprint of a mis-framed superframe rather than an unhandled opcode. It's
the same discipline the [naming post]({{ '/blog/deep-dives/the-hunt-11-naming-the-unknown/' | relative_url }})
drew for talkgroups — refuse to assert what you can't verify — applied to a
single source RID.

## The ciphertext we can't read

And then there is the alias GopherTrunk *reassembles perfectly and still can't
read*. The real Motorola FACCH-S alias runs through a proprietary per-byte
cipher, and that cipher is unverified — `motorola.CipherVerified` is false — so
the decoded name comes out empty on real traffic. Rather than throw the burst
away, the follower harvests the **ciphertext**:

```go
// internal/sigfollow/dispatcher.go (shape) — completeMotorolaAlias
func (d *MACDispatcher) completeMotorolaAlias(res p25p2.MotorolaAliasResult) {
    if len(res.Encoded) > 0 {
        d.log.Info(d.logPrefix+": p25p2 alias ciphertext",
            "system", d.system, "serial", d.serial,
            "rid", res.SourceID, "talkgroup", res.TalkgroupID,
            "encoded_hex", hex.EncodeToString(res.Encoded),
            "crc_ok", res.CRCOK, "reliable", res.Reliable)
    }
    d.publishTalkerAlias(res.SourceID, res.Alias, res.Reliable) // empty until the cipher is cracked
}
```

That log line is the whole point. The reassembled ciphertext — the encoded alias
bytes paired with the known source RID and talkgroup — is exactly the record the
cipher cryptanalysis needs, and the repo's chosen-plaintext capture procedure
names GopherTrunk as a valid capture receiver. Surfacing it here means an operator
can harvest the `rid,talkgroup,encoded_hex,alias` corpus from live air with
GopherTrunk alone, instead of falling back to SDRTrunk. This is the same emitter
the [Protocol Decoders]({{ '/blog/deep-dives/protocol-decoders-01-anatomy-of-a-cc-decoder/' | relative_url }})
series planted as its central mystery — the field that reassembles cleanly and
decodes to noise — and the same one the Crypto Lab series calls *Mercury*. The
hunt doesn't crack it; it feeds the machine that will.

## Where this goes next

We've now pulled everything the air will give us: a mapped, named system, activity
counts, whatever aliases and ciphertext its traffic carried. The find is complete.
[Part 13]({{ '/blog/deep-dives/the-hunt-13-exporting-your-finds/' | relative_url }})
is about getting it *out* — exporting the `DiscoveredSystem` to a RadioReference
submission package, a TrunkRecorder config, GopherTrunk's own import bundle, and a
SigMF-tagged capture bundle — so the work leaves your machine.

## FAQ

**Why not just decode aliases inside the voice recorder?**
Because on a busy system most grants never get a voice tuner, and encrypted calls
— the interesting ones — are torn down before the hangtime alias burst. A
voice-only decode misses the aliases you most want. A signalling follow decodes
the MAC stream without recording, so it runs on grants the voice pool never
covers.

**Does harvesting aliases need extra hardware?**
It needs a signalling tap, which is a DDC tap on an existing wideband IQ broker —
the same virtual-tuner mechanism the wideband engine uses — not a dedicated
radio. Two SDRs can run a whole system's control-plus-signalling harvest, which is
how SDRTrunk does it too.

**Why gate the source RID on RS parity?**
A mis-framed Phase 2 superframe decodes random bytes, and a stray opcode match
would inject a wrong-but-plausible RID that's indistinguishable downstream from a
real one. Only the outer RS parity separates a genuine PDU from garbage, so only
an RS-verified PDU sets a call's source. A wrong RID is worse than none.

**Why does a Motorola alias come out empty?**
Its per-byte cipher is unverified (`motorola.CipherVerified` is false), so the
decode can't be trusted to produce a real name. Rather than emit a wrong one, the
follower publishes an empty alias and logs the reassembled **ciphertext** — the
corpus the cryptanalysis needs. It's the *Mercury* cipher from the decoder and
crypto series.

**What stops two decoders fighting over one frequency?**
The manager tracks active follows and voice-followed frequencies. A grant already
being followed is ignored; a grant on a voice-followed frequency is skipped; and
`onVoiceStart` cancels a signalling follow the moment a voice chain takes the same
channel, because the voice chain harvests aliases itself.

## Series navigation

**Part 12 of 14** · ←
[Part 11: Naming the Unknown]({{ '/blog/deep-dives/the-hunt-11-naming-the-unknown/' | relative_url }})
· Next →
[Part 13: Exporting Your Finds — RadioReference, TrunkRecorder, SigMF]({{ '/blog/deep-dives/the-hunt-13-exporting-your-finds/' | relative_url }})
