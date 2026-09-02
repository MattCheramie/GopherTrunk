---
title: "P25 End to End, Part 9: Encryption Signalling — Flags, Metadata & Policy"
description: Where a P25 call's encryption actually lives on the air — the grant's protected bit, the LDU2 Encryption Sync carrying the Message Indicator, Algorithm ID and Key ID, and Phase 2's MAC_PTT — and how GopherTrunk turns those signals into per-system tuner and recording policy.
category: deep-dives
keywords: p25 encryption sync, p25 algid key id, p25 message indicator, ldu2 encryption sync, encrypted p25 calls scanner, p25 aes 256 adp rc4, skip encrypted recordings, encrypted call policy, gophertrunk p25
tags: [p25-end-to-end, p25, encryption, trunking, policy, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 9
---

*Part 9 of **P25 End to End**, a 14-part deep dive that follows North
America's dominant trunking protocol through GopherTrunk — from a raw C4FM
carrier to recorded, named, multi-site voice.
[Part 8]({{ '/blog/deep-dives/p25-end-to-end-08-imbe-voice/' | relative_url }})
followed an IMBE voice call from LDU superframes to a WAV on disk. This
part is about the calls that should never make that trip: where
"encrypted" actually lives on a P25 carrier — one bit on the grant, and a
96-bit Encryption Sync buried inside the voice stream — and the policy
machinery that decides what a scarce voice tuner, and your disk, should do
about a call no scanner can decode.*

> **TL;DR:** P25 signals encryption in two places GopherTrunk reads. The
> grant's Service Options octet carries a **protected bit**
> (`ServiceOptions.Encrypted`) — one bit, known before voice starts. The
> *metadata* arrives only in-call: the **Encryption Sync** — a 72-bit
> Message Indicator, Algorithm ID and Key ID — rides every LDU2 under a
> Hamming(10,6,3) inner FEC and an outer RS(24,16,9)
> (`phase1.ParseEncryptionSync`). ALGID `0x80` means clear; anything else
> names a cipher (`0x84` = AES-256, `0xAA` = ADP/RC4). On Phase 2 the same
> triple rides the **MAC_PTT** message at key-up (issue #813). Per-system
> `encrypted_calls` policy — **follow / metadata / ignore** (issue #711) —
> decides whether an encrypted call keeps its voice SDR, and
> `recordings.skip_encrypted` (issue #607) keeps undecodable audio off the
> disk. GopherTrunk identifies encryption; it does not break it.

**Key takeaways**

- **Encryption is signalled twice, at different layers.** The grant says
  *that* a call is protected, before allocation; only the traffic channel
  says *how*. Everything downstream is built around that gap: events that
  backfill, policies that re-evaluate, recordings that abort.
- **The metadata is FEC'd like it matters — and gated like it lies.** Two
  FEC layers protect the ES word, and an Algorithm ID that survives both
  but lands outside the TIA-102 registry is *provably* a mis-decode
  (`p25.AlgorithmKnown`, issue #924). A plausible-but-wrong key label is
  worse than none.
- **Policy is per system, and configured keys trump policy.** `follow` /
  `metadata` / `ignore` are set per `trunking.systems[]` entry; a call
  whose Key ID matches a configured `encryption_keys` entry is always
  followed.
- **Identify, never decrypt.** GopherTrunk surfaces ALGID/KID/MI the way
  SDRtrunk and OP25 do. Known-key decode exists only for DMR ARC4; on P25
  there is no decryption and — everywhere — no key recovery of any kind.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Grant protected bit | "this call is encrypted", known at grant time | `phase1.ServiceOptions.Encrypted` (`opcodes.go`) |
| LDU2 Encryption Sync | MI + ALGID + KID; Hamming(10,6,3) + RS(24,16,9) | `internal/radio/p25/phase1/encryption_sync.go` (`ParseEncryptionSync`) |
| Phase 2 key-up sync | the ES triple riding MAC_PTT (slot type, not opcode) | `internal/radio/p25/phase2/mac_standard.go` (`AsPushToTalk`) |
| Algorithm registry | ALGID → mnemonic + the out-of-set gate | `internal/radio/p25/algorithm.go` (`AlgorithmName`, `AlgorithmKnown`) |
| In-call event path | composer → bus → engine backfills the live Grant | `events.KindCallEncryption` (`trunking.CallEncryption`) |
| Tuner policy | follow / metadata / ignore, per system | `internal/trunking/encryptedmode.go` (`EncryptedMode`) |
| Recording guard | never write, or abort-and-delete mid-call | `internal/voice/recorder.go` (`Options.SkipEncrypted`) |

## In this post

- **Two flags, two layers** — the one-bit grant flag vs the 96-bit in-call word.
- **The Encryption Sync word** — MI, ALGID, KID and the two FEC layers under them.
- **Phase 2: four layers between a flag and its metadata** — the postmortem this design came from.
- **From burst to policy** — the event path, and what follow/metadata/ignore actually do.
- **What GopherTrunk won't do** — identification vs decryption vs key recovery.

## Two flags, two layers

A P25 voice grant
([Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }}))
carries an 8-bit Service Options octet, and bit `0x40` of it is the
*protected* flag:

| Signal | Where | When you learn it | What it tells you |
|---|---|---|---|
| Service Options bit `0x40` | grant TSBK / MAC grant, control channel | before the voice SDR is allocated | encrypted: yes/no — nothing else |
| Encryption Sync (Phase 1) | every LDU2 on the traffic channel | mid-call, repeating | MI (72 bits) + ALGID + KID |
| MAC_PTT ES (Phase 2) | key-up message on the traffic channel | at each talk spurt | the same triple (issue #813) |

The protected bit is the early warning: `Grant.Encrypted` is set before the
[voice pool]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }})
ever binds a tuner, which is what lets the `ignore` policy drop a grant
without spending anything on it. But one bit is all it is — no algorithm,
no key, and on compressed Phase 2 grants not even the bit, until the
traffic channel says otherwise.

The spec also front-loads the full MI/ALGID/KID triple into the
[**Header Data Unit**]({{ '/reference/p25-header-data-unit/' | relative_url }})
at key-up. GopherTrunk names the HDU in its DUID zoo (`DUIDHeader`,
[Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }}))
and deliberately leaves it undecoded: a trunked tune-in routinely arrives
*after* key-up — grant, retune, DDC settle and frame lock all cost time — so
a one-shot header at call start is exactly the frame a scanner is most
likely to miss. The same triple repeats in **every LDU2**, one per 360 ms
superframe, and that repeating copy is the one the pipeline is built on.

## The Encryption Sync word

The LDU2 donates the same six 40-bit slots an LDU1 spends on Link Control
([Part 8]({{ '/blog/deep-dives/p25-end-to-end-08-imbe-voice/' | relative_url }}))
to an Encryption Sync word, under the identical inner FEC — 24 codewords of
shortened Hamming(10,6,3). Above that sits an outer Reed-Solomon layer:

```go
// internal/radio/p25/phase1/encryption_sync.go (shape)
type EncryptionSync struct {
    MessageIndicator [9]byte // 72-bit per-call crypto sync vector (the IV)
    AlgorithmID      uint8   // 0x80 = clear; 0x81 DES-OFB, 0x84 AES-256, …
    KeyID            uint16  // which key in the radio's keyset
}

const AlgorithmClear uint8 = 0x80

func (e EncryptionSync) Encrypted() bool { return e.AlgorithmID != AlgorithmClear }

func ParseEncryptionSync(blocks [LDULCESBlockCount][]byte) (EncryptionSync, int, error) {
    data, innerErrs, err := lcInnerDecode(blocks) // 24 × Hamming(10,6,3)
    /* … */
    info, rsErrs, rerr := framing.DecodeRS24_16(cw[:]) // outer RS(24,16,9), t=4
    if rerr != nil {
        return EncryptionSync{}, innerErrs, ErrEncryptionSyncUncorrectable
    }
    /* … octets 0-8 → MI, octet 9 → ALGID, octets 10-11 → KID … */
}
```

Note what the clear value is *not*: zero. A clear call advertises ALGID
`0x80`, so "encrypted" means `AlgorithmID != 0x80` — a detail that has
bitten every decoder that treated the field as a boolean. The IDs come from
the TIA-102.AACE-A registry, mirrored in `internal/radio/p25/algorithm.go`:
`0x81` DES-OFB, `0x84` AES-256, `0x85` AES-128, `0xAA` ADP/RC4 (the "cheap
encryption" you meet most often), `0x9F` DES-XL among others — the
[p25-algorithm-id]({{ '/reference/p25-algorithm-id/' | relative_url }})
page keeps the full table.

The two FEC layers earn their keep under marginal SNR, but they hide a
trap: an RS decode can *report success* while "correcting" to garbage when
more errors survive the inner layer than t=4 can honestly fix. A bit-error
smears the Algorithm ID roughly uniformly across 0x00–0xFF with a
near-distinct Key ID per call — surfaced, that's indistinguishable from a
real key downstream. So both phases gate on the registry
(`p25.AlgorithmKnown`, issue #924): an out-of-set ALGID is dropped with a
debug line (`composer: p25p1 dropping out-of-set encryption sync`) rather
than published. The set tracks `AlgorithmName`, so a genuinely new
algorithm is admitted the moment it gets a name.

## Phase 2: four layers between a flag and its metadata

Phase 2
([Part 7]({{ '/blog/deep-dives/p25-end-to-end-07-phase2-tdma/' | relative_url }}))
signals the same two-layer story with MAC PDUs. The protected bit rides the
grant's Service Options exactly as on Phase 1 — and also on the in-call
`GROUP_VOICE_CHANNEL_USER` PDU, which matters because Phase 2 grants often
arrive in a *compressed* form with no source and no service options; the
traffic channel resolves both mid-call, published as a
`KindCallSourceUpdate` the engine folds back onto the live call.

The metadata took longer to find. GopherTrunk's first model put ALGID/KID/MI
in a standalone Encryption Sync MAC opcode (`AsEncryptionSync`) — and on
real systems the triple doesn't ride there. It rides the **MAC_PTT** message
at key-up, identified by its *slot type* rather than any opcode byte
(`AsPushToTalk`, issue #813, field offsets cross-checked against SDRtrunk's
PushToTalk structure). Getting to that answer was the subject of
[From the Issue Tracker Part 3]({{ '/blog/solution-postmortem/from-the-issue-tracker-03-phase2-encryption-metadata/' | relative_url }}):
a system that flagged every encrypted call correctly but never named an
algorithm or key, because a fictional MAC opcode, a missing carrier loop, a
48-bit sync constant in a 40-bit field and a swapped dibit map were stacked
on top of each other — four layers between a flag and its metadata.

Two design rules survive from that postmortem: the working-model layouts
are **confined to one file each**, so a spec correction is a one-file
change; and the protected bit **flags encryption independently** of the
metadata parse, so a wrong layout degrades to "encrypted true, alg/key
zero" instead of failing silently.

## From burst to policy

Here is the full path a discovered-encrypted call takes through the daemon:

<figure class="lab-figure">
<svg viewBox="0 0 680 258" width="680" height="258" role="img" aria-label="Two paths converge on the trunking engine: the grant's protected bit from the control channel, and the LDU2 Encryption Sync parsed and gated by the composer, published as a KindCallEncryption event; the engine backfills the live grant and applies the follow, metadata, or ignore policy that drives the recorder guard and the call log.">
  <text x="130" y="20" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">control channel (early)</text>
  <text x="480" y="20" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">traffic channel (mid-call)</text>
  <rect x="40" y="30" width="180" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="130" y="44" text-anchor="middle" fill="currentColor" font-size="10">grant TSBK / MAC grant</text>
  <text x="130" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SVC options bit 0x40 → Grant.Encrypted</text>
  <rect x="380" y="30" width="200" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="480" y="44" text-anchor="middle" fill="currentColor" font-size="10">LDU2 ES / Phase 2 MAC_PTT</text>
  <text x="480" y="57" text-anchor="middle" fill="var(--fg-muted)" font-size="9">MI + ALGID + KID, Hamming + RS</text>
  <line x1="480" y1="64" x2="480" y2="82" stroke="currentColor"/><polygon points="476,80 480,88 484,80" fill="currentColor"/>
  <rect x="380" y="88" width="200" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="480" y="102" text-anchor="middle" fill="currentColor" font-size="10">composer: ParseEncryptionSync</text>
  <text x="480" y="115" text-anchor="middle" fill="var(--fg-muted)" font-size="9">AlgorithmKnown gate (#924) → bus event</text>
  <line x1="130" y1="64" x2="130" y2="146" stroke="currentColor"/><polygon points="126,144 130,152 134,144" fill="currentColor"/>
  <line x1="480" y1="122" x2="480" y2="134" stroke="currentColor"/>
  <line x1="480" y1="134" x2="240" y2="158" stroke="currentColor"/><polygon points="247,159 232,159 242,151" fill="currentColor"/>
  <text x="500" y="140" fill="var(--accent)" font-size="9">KindCallEncryption</text>
  <rect x="40" y="152" width="290" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="185" y="168" text-anchor="middle" fill="var(--accent)" font-size="10" font-weight="bold">trunking engine</text>
  <text x="185" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="9">backfills ALGID/KID onto the live Grant · applies policy</text>
  <line x1="120" y1="192" x2="120" y2="212" stroke="currentColor"/><polygon points="116,210 120,218 124,210" fill="currentColor"/>
  <line x1="250" y1="192" x2="250" y2="212" stroke="currentColor"/><polygon points="246,210 250,218 254,210" fill="currentColor"/>
  <rect x="30" y="218" width="180" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="120" y="232" text-anchor="middle" fill="currentColor" font-size="10">recorder: skip_encrypted</text>
  <text x="120" y="245" text-anchor="middle" fill="var(--fg-muted)" font-size="9">never open, or abort-and-delete</text>
  <rect x="230" y="218" width="180" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="320" y="232" text-anchor="middle" fill="currentColor" font-size="10">call log + SSE/API</text>
  <text x="320" y="245" text-anchor="middle" fill="var(--fg-muted)" font-size="9">alg/key columns, E(alg=…,key=…) flag</text>
  <rect x="440" y="176" width="220" height="66" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="550" y="192" text-anchor="middle" fill="currentColor" font-size="10">encrypted_calls policy</text>
  <text x="550" y="206" text-anchor="middle" fill="var(--fg-muted)" font-size="9">follow · metadata (1.5 s) · ignore</text>
  <text x="550" y="220" text-anchor="middle" fill="var(--fg-muted)" font-size="9">configured key ⇒ always follow</text>
  <text x="550" y="234" text-anchor="middle" fill="var(--fg-muted)" font-size="9">teardown reason: "encrypted"</text>
</svg>
<figcaption>Two signals — one early and thin, one late and rich — converge on the engine, which owns the single policy decision everything downstream inherits.</figcaption>
</figure>

The Phase 1 composer publishes a `KindCallEncryption` event when an LDU2's
ES survives both FEC layers and the registry gate — deduplicated, since
ALGID/KID rarely change within a call. The engine backfills the bound
`ActiveCall.Grant` and republishes with the call's identity, which is why a
grant log line grows a suffix mid-call: `E(alg=0x84,key=0x01A3)`. ALGID and
KID persist into the call log, so an operator can see *which* key a
recorded call would need. What the engine then does is the per-system
`encrypted_calls` policy (issue #711),
[Trunking Engine Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})'s
subject:

- **`follow`** (the default and the zero value — pre-existing configs see
  no change): hold the voice SDR for the full call, like a clear one.
- **`metadata`**: follow just long enough to harvest what the traffic
  channel gives away — talker alias, source RID, the ES itself — then
  release the tuner. The window is `metadata_follow_ms` (default 1500 ms),
  and on Phase 2 the release short-circuits the moment the talker alias
  completes.
- **`ignore`**: never spend a tuner. A grant already flagged encrypted is
  dropped before allocation (`dropping encrypted grant (encrypted_calls
  mode: ignore)`); encryption discovered mid-call tears the call down with
  the dedicated end reason `encrypted` — a tuner freed by policy, not a
  decode failure. Emergency grants bypass the policy entirely.

Two exemptions cut across all modes. A call whose Key ID matches a
configured `trunking.systems[].encryption_keys` entry is always followed
(`Engine.keyConfigured` disarms any pending release). And the recorder has
its own independent guard: `recordings.skip_encrypted` (issue #607) refuses
to open a session for a flagged grant, and when encryption only surfaces
mid-stream it closes and **deletes** the in-progress WAV/raw files and
suppresses the CallComplete event, so upload feeds and webhooks never see
the partial —
[Recording & Streaming Part 7]({{ '/blog/deep-dives/recording-streaming-07-correctness-guards/' | relative_url }})
walks that abort machinery.

## What GopherTrunk won't do

The honest boundary, stated plainly. GopherTrunk **identifies** encryption
— algorithm, key ID, message indicator — exactly as SDRtrunk does, and
performs **no key recovery of any kind**. Known-key decryption exists today
only for DMR ARC4 "Enhanced Privacy", where an operator authorized to hold
a key configures it under `encryption_keys`; on P25, a configured key
currently buys the policy exemption (the call is followed and recorded),
not decryption.

What the identification path *does* feed is analysis of the signalling
itself: every encrypted LDU2's MI (the IV) and its still-encrypted IMBE
frames can be handed to the crypto-frame capture sink
(`internal/voice/cryptocap`) for offline keystream-reuse analysis — the
workflow the
[Crypto Lab series]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }})
builds on. The bridge captures every encrypted superframe, not one per call
(distinct ciphertexts matter even when ALGID/KID never change), and it is
off by default — the same opt-in discipline as every diagnostic tap in this
series.

### How the gap shaped the Go code

- **Events backfill instead of blocking.** The grant publishes immediately
  with what the control channel knows; `KindCallEncryption` and
  `KindCallSourceUpdate` patch the live call as the traffic channel fills
  the gap — no consumer waits for metadata that may never come.
- **Gates prefer omission to plausible garbage.** `AlgorithmKnown` and
  `ErrEncryptionSyncUncorrectable` drop low-confidence values rather than
  surface them — a wrong key label sends an operator chasing the wrong
  thing.
- **Policy state lives in one enum with a safe zero.** `EncryptedFollow` is
  the zero value *and* the parse fallback, so a config typo can never
  silently stop following calls.

## Where this goes next

Encryption was the last per-call layer. 
[Part 10]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }})
climbs above the call entirely: the WACN / System / RFSS / Site identity
ladder, the status broadcasts that carry it — in both TSBK and multi-block
AMBT forms — and how GopherTrunk votes a system's identity out of noisy
frames and accumulates a neighbour map you can roam by.

## FAQ

**How does GopherTrunk know a P25 call is encrypted?**
Twice over: the voice grant's Service Options octet carries a protected bit
(known before the call is even followed), and the traffic channel repeats
the full Encryption Sync — MI, Algorithm ID, Key ID — in every LDU2 on
Phase 1 or in the MAC_PTT key-up message on Phase 2. ALGID `0x80` is the
clear value; anything else names a cipher.

**Can GopherTrunk decrypt encrypted P25 calls?**
No. It surfaces which algorithm and key a call uses — the same
identify-don't-decrypt model as SDRtrunk and OP25 — and performs no key
recovery. Known-key decryption exists for DMR ARC4 only; on P25 a
configured key exempts the call from the encrypted-call policy but the
audio is not decrypted.

**Why do some encrypted calls show no algorithm or key?**
Either the call was torn down before an LDU2/MAC_PTT landed (the grant flag
alone carries no metadata), or the ES word failed its gates. GopherTrunk
deliberately omits ALGID/KID it cannot trust: an RS layer that
"successfully" corrects to an out-of-registry Algorithm ID is a known
mis-decode signature (issue #924).

**What does `encrypted_calls: metadata` actually buy me?**
Alias, source and key intelligence without the tuner cost. The engine
follows an encrypted call just long enough (default 1500 ms, or until the
Phase 2 talker alias completes) to capture the traffic channel's metadata,
then releases the voice SDR with end reason `encrypted`. On a system where
a third of the traffic is encrypted, that's the difference between clear
calls queueing for tuners and not.

**Should I set `recordings.skip_encrypted: true`?**
If you never hold keys, probably: it keeps unplayable audio off the disk
and out of your upload feeds, gating both at grant time and mid-call (the
in-progress file is deleted, not truncated). Leave it `false` if a
downstream consumer wants the call events regardless — the suppressed
CallComplete also suppresses call webhooks (issue #897) — or if you archive
ciphertext deliberately for cryptolab-style analysis.

## Series navigation

**Part 9 of 14** · ←
[Part 8: IMBE Voice — From LDU to WAV]({{ '/blog/deep-dives/p25-end-to-end-08-imbe-voice/' | relative_url }})
· Next →
[Part 10: Sites, WACNs & Roaming a Multi-Site System]({{ '/blog/deep-dives/p25-end-to-end-10-sites-roaming/' | relative_url }})
