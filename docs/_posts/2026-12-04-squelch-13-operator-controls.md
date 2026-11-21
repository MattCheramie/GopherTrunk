---
title: "Squelch, Part 13: Operator Controls — Priority, Lockouts and Loudness"
description: "What the operator turns on GopherTrunk's scan list once the gates work: a priority scan that samples marked channels every N picks, lockouts that persist by frequency rather than list index, trunked hold and avoid that deliberately do not persist, pinned talkgroup IDs, per-channel tuner gain, and an audio gain that levels a P25 channel against an NFM one."
category: deep-dives
keywords: conventional scanner priority scan, scanner lockout persist frequency, talkgroup hold avoid scanner, scan list audio gain db, p25 louder than analog fm, per channel tuner gain sdr, synthetic talkgroup id scan list, nxdn scrambler key scan list, gophertrunk scanner controls
tags: [squelch, scanner, priority-scan, lockout, loudness, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Squelch"
series_part: 13
---

*Part 13 of **Squelch**, a 14-part deep dive into GopherTrunk's conventional
scanner — the scan list with no control channel to tell it a call exists —
and every gate that decides when a channel opens.
[Part 12]({{ '/blog/deep-dives/squelch-12-narrowband-front-end/' | relative_url }})
put one channel front end under every narrowband decoder. Twelve parts of
gates decide when a channel opens on its own. This part is about the
decisions the scanner does not make: which channels the operator wants
sampled more often, which they never want to hear again, and how loud each
should be when it does open.*

> **TL;DR:** `scanner.priority_interleave: N` turns the round robin into a
> priority scan: after N ordinary picks `pickNextChannel` visits the next
> channel carrying a `priority`, and the cursor's own visit to one restarts
> the count — with N = 2 over A, P, C, D the rotation is A P C D P.
> Runtime lockouts persist **by frequency** (`storage.ConvLockoutStore`,
> table `conv_lockouts`), never by list index, which shifts when the list
> is edited; `talkgroup_id` pins a channel's synthetic ID for the same
> reason. Trunked hold and avoid (`trunking/holdavoid.go`) are session-only
> on purpose. A channel may carry its own tuner `gain`. And because the
> vocoder AGC targets a peak of 18000 (≈ −5 dBFS) while the analog chain has
> no AGC and peaks near −20 dBFS, P25 is about 10 dB louder than NFM by
> default — `audio_gain_db` rides `Grant.AudioGainDB` to fix it per channel.

**Key takeaways**

- **Priority scan is a second cursor, not a sort.** Any non-zero `priority`
  marks a channel; marked channels take turns, and the rotation never
  goes more than N ordinary channels without one.
- **Durable state is keyed by what the operator means, never by a list
  index.** Lockouts key on frequency, IDs on `talkgroup_id`, channel keys
  on frequency — an index changes meaning the moment the list is edited.
- **Persistence is a choice made per control.** A scan-list lockout
  survives a restart like a hardware scanner's lockout memory; a talkgroup
  hold or avoid clears like a power cycle.
- **Loudness is balanced per channel, not by moving a default.** Digital
  and analog audio are levelled by different mechanisms, and the gain is
  applied in exactly one place for each.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Priority scan | after N ordinary picks, visit the next priority channel; a cursor visit to one restarts the count | `pickNextChannelLocked`, `nextPriorityLocked` in `internal/scanner/conventional/scanner.go`; config `scanner.priority_interleave` |
| Lockout | drops a channel from the rotation and ends a dwell in progress at once | `LockoutChannel`, `UnlockoutChannel` |
| Lockout memory | persisted by frequency, restored before the first pick | `storage.ConvLockoutStore` (table `conv_lockouts`), `cmd/gophertrunk/conv_lockouts.go` |
| Trunked hold and avoid | pin the engine to one talkgroup; a lockout with an expiry; neither persists | `internal/trunking/holdavoid.go` |
| Stable IDs | a fixed synthetic talkgroup ID instead of the positional one | config `talkgroup_id`, `Channel.TalkgroupID` |
| Tuner gain | per-channel gain written before the channel is tuned | `internal/scanner/conventional/gain.go` (`applyChannelGain`), config `gain` |
| Loudness | per-channel dB on recovered audio: analog in the chain, digital in the recorder | `Grant.AudioGainDB`, `internal/voice/audiogain.go`, config `audio_gain_db` |
| Channel keys | scan-list decryption keys resolved by frequency | `cmd/gophertrunk/daemon_conv_keys.go`, `composer.ConventionalKeySystem` |

## In this post

- **Priority scan: a second cursor** — the interleave, the count that resets, and what it does not do.
- **Lockouts that remember, by frequency** — why the key is hertz, and the same rule for `talkgroup_id`.
- **Hold and avoid: session-only on purpose** — where they sit in `HandleGrant`, and why scan-list calls never reach them.
- **Gain and loudness, per channel** — tuner `gain`, the 10 dB gap between P25 and NFM, and `audio_gain_db`.
- **Keys on a scan-list channel** — pseudo systems named by frequency.

## Priority scan: a second cursor

The scan loop of
[Part 1]({{ '/blog/deep-dives/squelch-01-no-control-channel/' | relative_url }})
is a round robin: tune, wait out a scan window, move on. A long list makes
every channel wait for all the others, and a call on the one that matters
is missed if it starts and ends while the scanner is elsewhere. Hardware
scanners answer with a
[priority check]({{ '/reference/priority-scan/' | relative_url }});
GopherTrunk's version is `scanner.priority_interleave`:

```go
// internal/scanner/conventional/scanner.go (shape)
if s.opts.PriorityInterleave > 0 && s.sincePriority >= s.opts.PriorityInterleave {
    if idx, ok := s.nextPriorityLocked(); ok {
        s.sincePriority = 0
        return idx, s.channels[idx], true // the main cursor does not move
    }
}
// ...the ordinary walk, skipping locked-out channels:
if s.channels[idx].Priority > 0 {
    s.sincePriority = 0 // the rotation reached a priority channel on its own
} else {
    s.sincePriority++
}
```

Two counters do the work. `sincePriority` counts ordinary picks since a
priority channel was last sampled; when it reaches the interleave, the next
pick comes from `nextPriorityLocked`, which walks its own cursor
(`priCursor`) over the channels whose `Priority` is non-zero. The main
cursor stays where it was, so the ordinary channels keep their order.

The detail that keeps it from over-sampling: when the main cursor reaches
a priority channel by itself, **that visit counts**, and the clock
restarts. `TestConvScannerPriorityInterleave` pins the result for four
channels with one priority channel and an interleave of 2 — A P C D P A P
C D P. P is never more than two ordinary channels away, and never sampled
twice in a row.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Two rows of ten scan picks over channels A, P, C and D, where P carries a priority. The top row is the plain round robin: A P C D A P C D A P. The bottom row uses a priority interleave of two: A P C D P A P C D P, with the two interleaved visits to P highlighted and the count of ordinary picks since the last priority visit shown under each pick.">
  <text x="10" y="18" fill="var(--fg-muted)" font-size="9" font-weight="bold">priority_interleave: 0 — plain round robin</text>
  <g fill="none" stroke="var(--fg-muted)">
    <rect x="10" y="28" width="56" height="26" rx="4"/><rect x="76" y="28" width="56" height="26" rx="4"/>
    <rect x="142" y="28" width="56" height="26" rx="4"/><rect x="208" y="28" width="56" height="26" rx="4"/>
    <rect x="274" y="28" width="56" height="26" rx="4"/><rect x="340" y="28" width="56" height="26" rx="4"/>
    <rect x="406" y="28" width="56" height="26" rx="4"/><rect x="472" y="28" width="56" height="26" rx="4"/>
    <rect x="538" y="28" width="56" height="26" rx="4"/><rect x="604" y="28" width="56" height="26" rx="4"/>
  </g>
  <g text-anchor="middle" font-size="10" fill="var(--fg-muted)">
    <text x="38" y="45">A</text><text x="104" y="45">P</text><text x="170" y="45">C</text><text x="236" y="45">D</text><text x="302" y="45">A</text>
    <text x="368" y="45">P</text><text x="434" y="45">C</text><text x="500" y="45">D</text><text x="566" y="45">A</text><text x="632" y="45">P</text>
  </g>
  <text x="10" y="92" fill="var(--accent)" font-size="9" font-weight="bold">priority_interleave: 2 — P carries a priority</text>
  <g fill="none" stroke="currentColor">
    <rect x="10" y="102" width="56" height="26" rx="4"/><rect x="76" y="102" width="56" height="26" rx="4"/>
    <rect x="142" y="102" width="56" height="26" rx="4"/><rect x="208" y="102" width="56" height="26" rx="4"/>
    <rect x="340" y="102" width="56" height="26" rx="4"/><rect x="406" y="102" width="56" height="26" rx="4"/>
    <rect x="472" y="102" width="56" height="26" rx="4"/><rect x="538" y="102" width="56" height="26" rx="4"/>
  </g>
  <g fill="none" stroke="var(--accent)" stroke-width="2">
    <rect x="274" y="102" width="56" height="26" rx="4"/><rect x="604" y="102" width="56" height="26" rx="4"/>
  </g>
  <g text-anchor="middle" font-size="10" fill="currentColor">
    <text x="38" y="119">A</text><text x="104" y="119">P</text><text x="170" y="119">C</text><text x="236" y="119">D</text>
    <text x="368" y="119">A</text><text x="434" y="119">P</text><text x="500" y="119">C</text><text x="566" y="119">D</text>
  </g>
  <g text-anchor="middle" font-size="10" fill="var(--accent)" font-weight="bold">
    <text x="302" y="119">P</text><text x="632" y="119">P</text>
  </g>
  <g text-anchor="middle" font-size="8" fill="var(--fg-muted)">
    <text x="38" y="142">1</text><text x="104" y="142">0</text><text x="170" y="142">1</text><text x="236" y="142">2</text><text x="302" y="142">0</text>
    <text x="368" y="142">1</text><text x="434" y="142">0</text><text x="500" y="142">1</text><text x="566" y="142">2</text><text x="632" y="142">0</text>
  </g>
  <text x="10" y="162" fill="var(--fg-muted)" font-size="8">sincePriority after each pick: an ordinary channel adds one, any visit to P resets it</text>
  <text x="10" y="180" fill="var(--accent)" font-size="8">highlighted: picks taken from the priority cursor when the count reached 2 — the main cursor does not move</text>
</svg>
<figcaption>The cursor's own visit to a priority channel restarts the count, so the interleave fills the gaps instead of doubling up.</figcaption>
</figure>

Three edges are pinned by the same test file,
`lockout_persist_test.go`. A locked-out priority channel is skipped by
both cursors. When *every* unlocked channel carries a priority there is
nothing to interleave against, and the rotation stays the plain round
robin (`TestConvScannerPriorityInterleaveOnlyPriority`). And with the key
set but no channel marked, nothing changes.

What it is not: a preemption. The interleave changes which channel the
scanner *picks next*; it does not pull the scanner out of a dwell to check
a priority channel mid-call, as a hardware priority check does. And the
rotation does not rank priority channels against each other — they take
turns.

## Lockouts that remember, by frequency

A lockout is the operator saying "not this one". `LockoutChannel` adds the
index to the scanner's `lockedOut` set, `pickNextChannel` skips it, and if
the channel is dwelling the synthetic call ends immediately
(`EndReasonLockout`) and the visit's stream is cancelled — not after
hangtime. One deliberate exception: a forced dwell (`DwellOn`, the TUI's
Enter on a channel row) beats a lockout for one cycle, because an explicit
"listen to this now" outranks a standing "skip this".

Until the persistence work that set lived in memory, and it was a
`map[int]bool` keyed by list index. Both halves were wrong for anything
that should outlive the process. The fix stores **frequencies**:

```go
// cmd/gophertrunk/conv_lockouts.go (shape)
onChange := func(ch conventional.Channel, locked bool) {
    if locked {
        err = store.Set(ctx, ch.FrequencyHz, ch.Label)
    } else {
        err = store.Clear(ctx, ch.FrequencyHz)
    }
}
return hz, onChange // hz: the frequencies to restore at construction
```

`storage.ConvLockoutStore` is a three-column table, `conv_lockouts`
(`frequency_hz` as the primary key, `label`, `locked_at`). The daemon reads
it once and hands the frequencies to the scanner as `Options.LockedOutHz`;
`New` applies them to whichever channels carry those frequencies *before
the first pick*, so a locked channel is never dwelt on even once after a
restart. Every runtime toggle then reaches `Options.OnLockoutChange` with
the channel. The properties worth stating, all pinned by
`TestConvScannerRestoresPersistedLockouts`:

- **A stale frequency is ignored, and kept.** Remove a channel from the
  config and its lockout matches nothing; put it back and it applies again.
- **Restoring does not re-persist.** Seeding the set fires no callback.
- **A database error never blocks the operator.** The lockout applies in
  memory and the failure is logged — refusing a lockout over a storage
  hiccup is the worse outcome.
- **No storage, no memory.** Without `storage.path` the hooks are nil and
  lockouts stay runtime-only, as before.

The index would have failed silently. Insert one channel at the top of
`scanner.conventional` and every stored index now names its neighbour; the
scanner would lock out the wrong channel and report nothing. Frequency is
what the operator meant.

**The same rule for IDs.** A scan-list channel surfaces as a synthetic
[talkgroup]({{ '/reference/talkgroup/' | relative_url }}) so the call log,
the API and upload consumers have something to key on. The default is
positional — `0x80000000 | index` — and it shifts exactly as a lockout
index would, quietly re-pointing every `talkgroup_file` roster row.
`talkgroup_id`
([#1105](https://github.com/MattCheramie/GopherTrunk/issues/1105)) pins it;
config validation rejects two channels that resolve to the same effective
ID, since the engine would fold their calls into one.
`TestConvScannerUsesExplicitTalkgroupID` is the failing-first pin.

## Hold and avoid: session-only on purpose

The trunked side has the other two front-panel controls, and they were
built to the opposite persistence rule.

**Hold** pins the engine to one talkgroup: every other grant is dropped
until the hold is released. **Avoid** is a lockout with an expiry — the
API's default is 30 minutes — after which scanning resumes on its own.
Both live in `internal/trunking/holdavoid.go`, keyed by (system,
talkgroup), and both are consulted at one point in `HandleGrant`:

```go
// internal/trunking/engine.go (shape)
if tg != nil && tg.Lockout && !g.Emergency { return }  // permanent lockout
// ...patch attribution...
if ok, why := e.holdAvoid.gate(g.System, g.GroupID, g.PatchedGroups); !ok {
    if !g.Emergency { return }                          // hold / avoid
} else if _, isHeld := e.holdAvoid.Held(); isHeld {
    held = true
}
if e.ScanMode() == ScanModeList && !g.Emergency && !held { /* scan-list gate */ }
```

The order is the design. They sit **after** the catalogue's permanent
`Lockout` and **before** the scan-list gate
([Trunking Engine Part 6]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})
covers those two). An emergency grant passes both, as it passes lockout. A
held talkgroup also bypasses the scan-list gate, so "hold on this one"
works in list mode even when its `Scan` flag is off. An avoid on one member
of a patched supergroup does not block the supergroup while another member
is followable. `TestEngineHoldFollowsOnlyTheHeldTalkgroup` and
`TestEngineAvoidIsATimedLockout` pin each clause.

Neither is written anywhere. A restart clears them, like a scanner power
cycle — deliberately. A hold is a statement about *now*; a daemon that came
back from a reboot still pinned to last night's talkgroup, dropping every
other grant, would look broken. A conventional lockout is a statement about
the channel, and should hold until the operator takes it back.

One consequence is easy to miss. A scan-list call does not enter through
`HandleGrant` at all. The scanner announces it with `HandleSyntheticCall`,
which has no grant gate — so a talkgroup hold or avoid never touches a
scan-list channel, and the scanner's own controls (lockout, forced dwell,
and the `Hold` / `Resume` pair that pauses the rotation) never touch a
trunked call. The two halves share an engine and a recorder, not a set of
controls. The recipe for driving both is
[Cookbook Volume 2 Part 9]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }}).

## Gain and loudness, per channel

A scan list mixes bands and modes — VHF FM beside air-band AM beside UHF
digital — and one SDR serves all of them. Two different things called
"gain" need balancing, and they are easy to confuse.

**Tuner gain** is RF. `gain` on a channel is the same string
`sdr.devices[].gain` takes — `"auto"` or tenths of a dB — and
`applyChannelGain` writes it to the scanner's SDR before the channel is
tuned. A channel without one runs at the device's configured gain, so a
mixed list returns there after leaving a channel that set its own. The
scanner touches the gain only once *some* channel sets one; from then on
it owns it, and a gain changed from the API is overwritten at the next
channel whose gain differs (`TestConvScannerAppliesPerChannelGain`,
`TestConvScannerLeavesGainAloneWithoutChannelGains`). It arrived with the
in-channel power squelch of
[Part 2]({{ '/blog/deep-dives/squelch-02-channel-power/' | relative_url }}),
and it carries that part's caveat: `squelch_dbfs` is an absolute level, so
a channel's gain and its squelch threshold are set together
([The Analog Edge Part 3]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})).

**Audio gain** is what the listener hears, and here the modes genuinely
differ. Decoded digital voice is levelled by the vocoder's AGC, whose
`TargetPeak` is 18000 — about −5 dBFS; the committed IMBE test vector
`p25p1-tone-bracketed-voice.raw` measures −19.5 dBFS RMS. The analog chain
has **no AGC in production** (the composer's `AudioAGC` option is never
configured); it scales the discriminator by 10 000, so a full ±2.5 kHz NFM
channel peaks near −20 dBFS. By default, then, a P25 channel is about 10 dB
*louder* than the NFM repeater next to it.

Reports of the opposite exist, and they are source- and config-specific —
5 kHz-deviation amateur FM, compressed air-band AM, the enhance and
normalize settings of
[Voice Coding Part 10]({{ '/blog/deep-dives/voice-coding-10-enhancement-loudness/' | relative_url }}).
That is why the fix is a per-channel knob rather than a moved default:
`audio_gain_db`, ±30 dB, clamped at full scale
([#1239](https://github.com/MattCheramie/GopherTrunk/issues/1239)).

```yaml
scanner:
  priority_interleave: 2
  conventional:
    - label: "Tower"
      frequency_hz: 118700000
      mode: am
      priority: 1
      gain: "280"          # tenths of a dB: 28.0 dB on this channel only
      audio_gain_db: 6
    - label: "Sheriff Repeater"
      frequency_hz: 155895000
      mode: nfm
      talkgroup_id: 2147500000
    - label: "VCALL10"
      frequency_hz: 155752500
      mode: p25
      audio_gain_db: -10   # digital is the louder side by default
```

The value rides the grant as `Grant.AudioGainDB`, and it is applied in
exactly one place per path. The composer's FM/AM chain multiplies the
demodulated audio just before PCM conversion
(`TestComposerAnalogChainAppliesAudioGain`). For a digital channel PCM
exists only in the recorder, after the vocoder, so `applyAudioGain` scales
it there — covering the WAV and the live tap in one point
(`TestRecorderAppliesConventionalAudioGain`: 9000 becomes 17957 at +6 dB
and clamps at 32767 at +12). Never both: analog audio does not pass
through the recorder's vocoder path, so nothing is scaled twice.

## Keys on a scan-list channel

One more per-channel setting follows the frequency rule. GopherTrunk
decrypts only with keys the operator already holds for systems they are
authorised to monitor, and on a trunked system those keys hang off the
system's `encryption_keys`. A scan-list call has no system — its grant's
`System` is the literal `scanner` — so `conventionalKeySystems`
(`cmd/gophertrunk/daemon_conv_keys.go`) turns each channel's
`encryption_keys` into a pseudo system named
`composer.ConventionalKeySystem(hz)`, which is `conventional:` followed by
the frequency. The composer's `resolveKey` tries the grant's system first
and that name second. Two channels on one frequency pool their keys
(`TestConventionalChannelKeysResolveByFrequency`).

Today that carries an NXDN channel's 15-bit
[scrambler]({{ '/reference/nxdn-scrambler/' | relative_url }}) keys —
`algorithm: scrambler`, a decimal key, `key_id` as the radios' signalling
names it. The cipher itself is capture-verified on the reporter's two
known-key captures
([Rebuilt From the Air Part 9]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }}));
the reporter's live run with the key on a scan-list channel is still the
on-air gate.

### How operator controls shaped the Go code

- **Callbacks, not a storage dependency.** The scanner knows
  `LockedOutHz` and `OnLockoutChange`; only the daemon knows SQLite.
- **One lock for pick and publish.** `pickVisit` picks the channel and
  publishes the visit in one critical section, so a lockout landing
  mid-tune finds something to abort.
- **A separate mutex for hold and avoid.** `holdAvoid` has its own lock so
  the API can read it without contending with decode.
- **Gain travels on the grant.** `AudioGainDB` is 0 on every trunked
  grant, so the trunked path is untouched.

Everything in this part is pinned by unit tests rather than by air: these
are control paths, and none of them measures a signal. The numbers behind
the loudness gap come from the code and one committed test vector, not
from a field survey.

## Where this goes next

Thirteen parts, one gate or control at a time.
[Part 14]({{ '/blog/deep-dives/squelch-14-playbook/' | relative_url }})
folds them into a single gate map — channel mode to gate to statistic to
config key — with the failure signatures operators actually report and the
honest list of what is still open.

## FAQ

**How does priority scan work on GopherTrunk's conventional scan list?**
Set `scanner.priority_interleave` to N and give the important channels a
non-zero `priority`. After N ordinary channels the rotation visits the
next priority channel without moving the main cursor, and the cursor's own
visit to a priority channel restarts the count. It changes the pick order;
it does not interrupt a call in progress.

**Do conventional scanner lockouts survive a restart?**
Yes, when `storage.path` is set. Runtime lockouts are stored by frequency
in the `conv_lockouts` table and re-applied before the first scan pick, so
re-ordering the scan list cannot lock out the wrong channel. Without
storage they are runtime-only.

**Why is my P25 channel louder than my analog FM channel?**
Digital voice is levelled by the vocoder's AGC to a peak near −5 dBFS; the
analog chain has no AGC and a full ±2.5 kHz NFM signal peaks near
−20 dBFS, so P25 is about 10 dB louder by default. Set `audio_gain_db` on
either channel (±30 dB) to balance them.

**Does a talkgroup hold or avoid affect scan-list channels?**
No. Hold and avoid gate trunked grants inside the engine's `HandleGrant`;
a scan-list call enters through `HandleSyntheticCall`, which has no grant
gate. Use the scanner's own lockout, dwell and hold for scan-list
channels. Hold and avoid are also session-only: a restart clears them.

**Why should I set talkgroup_id on a scan-list channel?**
Without it the channel's synthetic ID is `0x80000000` plus its position in
the list, which shifts when channels are inserted, removed or re-ordered
and silently re-points roster rows and upload mappings. A pinned
`talkgroup_id` survives list edits.

## Series navigation

**Part 13 of 14** · ←
[Part 12: One Front End for Every Narrowband Decoder]({{ '/blog/deep-dives/squelch-12-narrowband-front-end/' | relative_url }})
· Next →
[Part 14: The Squelch Playbook]({{ '/blog/deep-dives/squelch-14-playbook/' | relative_url }})
