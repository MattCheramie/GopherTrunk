---
slug: encrypted-call-handling
title: Encrypted-call handling
entry_type: term
category: fn-config
description: "Encrypted-call handling is GopherTrunk's per-system encrypted_calls policy for spending scarce voice SDRs on encrypted traffic, shaped by when encryption is actually detectable and by the skip_encrypted and metadata-mode traps worth knowing in advance."
keywords: encrypted_calls, skip_encrypted, metadata_follow_ms, encryption policy, algorithm id, key id, talker alias, webhook, p25 encryption
aka: [encrypted_calls policy, encryption policy, metadata follow mode]
infobox:
  - { label: Type, value: Config policy }
  - { label: Modes, value: follow / metadata / ignore }
  - { label: Scope, value: "Per system, on each trunking.systems entry" }
  - { label: Key fact, value: Encryption is often only revealed mid-call }
see_also: [p25-encryption, p25-algorithm-id, key-id-algid, encryption, p25-talker-alias, p25-encryption-sync, p25-phase-2, wideband-voice-taps]
related_reading:
  - { title: "From the Issue Tracker, Part 3: Encrypted, Says Who — Four Layers Between a Flag and Its Metadata", url: /blog/solution-postmortem/from-the-issue-tracker-03-phase2-encryption-metadata/ }
cite_urls:
  - https://github.com/MattCheramie/GopherTrunk/issues/711
  - https://github.com/MattCheramie/GopherTrunk/issues/897
  - https://github.com/MattCheramie/GopherTrunk/issues/813
---

**Encrypted-call handling** is the per-system policy for how scarce voice SDRs are spent on
[encrypted](/reference/encryption/) traffic, configured in the `encrypted_calls` block on
each `trunking.systems[]` entry. Without a policy, a heavily encrypted system ties up the
whole voice pool recording noise while clear calls on other systems starve
([#711](https://github.com/MattCheramie/GopherTrunk/issues/711)). The block is deliberately
per-system — it moved there from a global key as a breaking change — so one system can run
`metadata` while another runs `follow` or `ignore`.

- `mode: follow` (default) — hold a voice SDR for the full encrypted call, like a clear call.
- `mode: metadata` — follow briefly so traffic-channel metadata (talker alias, source RID,
  [encryption sync](/reference/p25-encryption-sync/)) is captured, then release the tuner.
  The release fires when the alias completes or after `metadata_follow_ms`, whichever comes
  first (0 uses the 1500 ms engine default).
- `mode: ignore` — never tie up a voice SDR on an encrypted call.

A system with a configured decryption key is exempt and always followed. Note GopherTrunk
does not decrypt P25 in-process; only DMR RC4 (Enhanced Privacy) keys are accepted, so the
exemption is effectively DMR-only.

## Encryption is often not known at grant time

The policy cannot be a grant-time-only check. Where the encrypted flag actually appears:

- **P25 Phase 2 full grants** signal encryption in the grant's service options — known up front.
- **P25 Phase 1** reveals it mid-call, in the LDU2 encryption sync.
- **P25 Phase 2 compressed grants** arrive with no source and `encrypted: false`; the real
  state lands mid-call via a source update ([#897](https://github.com/MattCheramie/GopherTrunk/issues/897)).

So `ignore` and `metadata` must be able to release a tuner that is *already bound* when the
mid-call update arrives — and any consumer of call records must expect the encrypted flag to
change after the call starts. That is exactly what bit the webhook: it was built from a
grant-time snapshot while the history API used the backfilled end-of-call state, so the
webhook reported `encrypted: false` forever until the recorder learned to mirror mid-call
facts onto its session ([#897](https://github.com/MattCheramie/GopherTrunk/issues/897)).

## Traps

| Symptom | Looks like | Actually | Fix / check |
| --- | --- | --- | --- |
| No completed-call webhook for encrypted calls | Webhook or policy bug | `recordings.skip_encrypted: true` discards the call **without publishing CallComplete** — it silences the webhook and Rdio Scanner backends too ([#897](https://github.com/MattCheramie/GopherTrunk/issues/897)) | Keep `skip_encrypted: false` if downstream consumers need encrypted-call events |
| Webhook says `encrypted: false`, history says `true` | Data corruption | Two snapshots of one call: stale grant-time state vs backfilled mid-call state ([#897](https://github.com/MattCheramie/GopherTrunk/issues/897)) | Fixed; a reminder that the flag legitimately changes mid-call |
| `metadata` mode still starves the tuner pool | Policy not applied | On some systems the talker alias **never completes**, so nothing triggers the early release and the call is followed to timeout — reproducing the starvation the mode was built to solve ([#711](https://github.com/MattCheramie/GopherTrunk/issues/711)) | `metadata_follow_ms` is the backstop; keep it modest (1500–2000 ms) |
| Aliases truncated / failing CRC in metadata mode | Decode problem | Releasing too early: on Phase 2 the alias arrives during call *hangtime*, not at call start ([#711](https://github.com/MattCheramie/GopherTrunk/issues/711)) | Allow 1500–2000 ms so the full alias block sequence lands |
| `encrypted: true` but no `algorithm_id`/`key_id` | Missing feature | Normal on some traffic: the encrypted flag and the ALGID/KID travel separately, and the IDs are gated to known TIA-102 algorithm values so bit-error garbage is not published ([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)) | Absent IDs with a true flag is a valid, honest state |

## Metadata that survives encryption

The [talker alias](/reference/p25-talker-alias/) rides link control *outside* the encrypted
voice payload, which is why `metadata` mode can harvest it from encrypted calls at all
([#711](https://github.com/MattCheramie/GopherTrunk/issues/711)). Two caveats:

- Timing: on Phase 2 it arrives as FACCH-S blocks during hangtime, hence the follow window above.
- Reliability: the alias passes through a vendor cipher that can launder bit errors into a
  plausible-but-wrong name; decodes that trip the plausibility checks are flagged
  (`talker_alias_unreliable` on the RID API) rather than silently published
  ([#711](https://github.com/MattCheramie/GopherTrunk/issues/711)).

The [algorithm ID](/reference/p25-algorithm-id/) and [key ID](/reference/key-id-algid/) have
their own hard-won path: on [P25 Phase 2](/reference/p25-phase-2/) they ride the MAC_PTT
message, and after a stretch where field decodes produced uniformly smeared garbage values,
the composer now validates the ALGID against the TIA-102 algorithm registry before
publishing — an unknown ID is withheld, not guessed
([#813](https://github.com/MattCheramie/GopherTrunk/issues/813)). So a populated
`algorithm_id` is a claim with evidence behind it, and its absence on an encrypted call is
information, not a bug.

## Provenance

- [#711](https://github.com/MattCheramie/GopherTrunk/issues/711) — the `encrypted_calls` policy, mid-call detection requirement, hangtime alias timing, the never-completing-alias failure, and the alias reliability flag.
- [#897](https://github.com/MattCheramie/GopherTrunk/issues/897) — the stale-snapshot webhook bug and the `skip_encrypted` webhook-suppression trap.
- [#813](https://github.com/MattCheramie/GopherTrunk/issues/813) — where Phase 2 ALGID/KID really live and the validity gate that keeps garbage IDs out of the API.
