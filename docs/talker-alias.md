---
layout: page
title: P25 talker aliases (Motorola)
description: How GopherTrunk decodes the Motorola P25 talker alias — the display name a radio broadcasts for itself — where the name shows up, what to configure, and how to tell a decode problem from a weak-signal problem
nav_group: Reference
---

# P25 talker aliases (Motorola)

A **talker alias** is the human-readable name a Motorola P25 radio broadcasts
for itself — "CW 2419", "ENGINE 12", a unit's callsign — so you see a name
instead of a bare radio ID (RID). GopherTrunk decodes it **end to end**: the
fragment reassembly, the system/unit identity (SUID), the CRC, and the alias
text itself. The proprietary per-byte cipher Motorola runs the name through
was recovered by clean-room reverse engineering and is **enabled by default**
(`CipherVerified = true`; issue
[#773](https://github.com/MattCheramie/GopherTrunk/issues/773), shipped in
v1.0.8). Nothing has to be switched on.

The decode is verified against real over-the-air data: the capture from
[#376](https://github.com/MattCheramie/GopherTrunk/issues/376) (Victorian
MMR, RID 200062) decodes to its radio's actual display name, "CRIO 0062", with
a matching CRC-16/GSM, and the cipher reproduces a held-out reference set
byte-for-byte (1242 of 1242 characters). The whole live path — real alias
PDUs on 144-bit MAC sub-frames through the superframe decoder and the shared
MAC dispatcher onto the event bus — is pinned by an end-to-end test
(`sigfollow.TestDispatcherPublishesRealMotorolaAliasAsReliable`).

## Where the name shows up

| Surface | What you see |
|---|---|
| **Radio IDs** web panel | a **Talker alias** column, and the field on the radio's detail card. A ⚠ next to it means the decode passed the frame check but is not clean printable text — treat it as suspect. |
| `GET /api/v1/rids` and `GET /api/v1/rids/{id}` | `talker_alias`, `talker_alias_at`, and `talker_alias_unreliable` on each radio |
| Live events (`GET /api/v1/events`, gRPC) | a `talker.alias` event with `system`, `protocol`, `source_id`, `alias`, `unreliable` |
| `debug.log` | `composer: p25p2 talker alias` / `sigfollow: p25p2 talker alias` (Phase 2 traffic channel), `p25: cc talker alias` (Phase 1 control channel) |

The talker alias is live state: it is held on the radio's entry while the
radio stays active and refreshed every time the radio announces it. It is
separate from the **alias** column, which is the name *you* assign (from
`rid_alias_file` or `PATCH /api/v1/rids/{id}`). To keep an over-the-air name
permanently, pin it with the PATCH — the operator label wins and persists.

## What has to be true for an alias to decode

The alias is carried on **signalling GopherTrunk has to be receiving**:

- **P25 Phase 2 (most Motorola systems today).** The alias rides FACCH-S MAC
  PDUs on the **traffic channel**, mostly during the hang time after a
  transmission. GopherTrunk decodes it on every path that follows a call to
  voice — a dedicated `role: voice` receiver, a `role: auto` receiver, or a
  wideband receiver's `voice_taps` — no extra configuration.
- **Busy or encrypted Phase 2 systems.** On a multi-site system most grants
  never win a voice tap, and encrypted calls are torn down before hang time,
  so the alias on the voice path rarely runs. Give a wideband receiver
  **`signalling_taps: N`** (2–4 is typical): each tap is a signalling-only DDC
  that follows a granted traffic channel just to decode its MAC PDUs and
  publish the alias, independent of the voice pool. This is how SDRTrunk sees
  every alias on such a system.
- **P25 Phase 1.** The alias also rides LDU1 / TDULC link control on the voice
  channel and a vendor TSBK on the control channel; both are decoded.

A minimal Phase 2 config that decodes aliases:

```yaml
sdr:
  devices:
    - serial: "00000001"
      role: wideband
      center_freq_hz: 851_500_000
      voice_taps: 4
      signalling_taps: 4        # alias harvest off traffic-channel signalling
      channels:
        - frequency_hz: 851_037_500
          system: "metro-p25"

trunking:
  systems:
    - name: "metro-p25"
      protocol: p25
      control_channels: [851_037_500]
```

## Weak Phase 2 signals

An alias surfaces exactly as often as the traffic channel's MAC PDUs pass
their outer Reed-Solomon check. On a weak or multipath Phase 2 channel that
check can fail on most sub-frames (the same limit that keeps the in-call
source RID from landing — issue
[#915](https://github.com/MattCheramie/GopherTrunk/issues/915)). Phase 2
signalling now decodes on the real TDMA air format: every FACCH / SACCH
burst runs a full RS(63,35) errors-and-erasures decode and must pass its
CRC, so a wrong descramble or a noisy burst is dropped rather than turned
into a bogus name. The one per-system knob that still trades CPU for
sensitivity on a marginal site is the equalizer:

```yaml
    - name: "metro-p25"
      protocol: p25
      control_channels: [851_037_500]
      p25_phase2_equalizer: "on"       # blind CMA equalizer for multipath / ISI
```

`p25_phase2_rs_mode` is still accepted but no longer read (the air format's
RS code is always decoded), and `p25_phase2_soft_decision` now affects only
the demodulator. Descrambling needs the site identity: set `wacn` /
`system_id` / `site`, or let a Phase 2 control channel's Network Status
Broadcast supply them.

## Telling a decode problem from a signal problem

`debug.log` has two lines that separate the cases:

- `p25p2 alias ciphertext … rid=… encoded_hex=… crc_ok=… reliable=…` is
  logged for **every** reassembled alias, decoded or not. If you see these
  lines, the radio's alias PDUs are reaching GopherTrunk; the `reliable` flag
  says whether the name decoded cleanly. The same `encoded_hex` repeating
  across a radio's keyups is expected (the ciphertext is deterministic).
- `p25p2 talker alias … src=… alias=…` is the published name.

If a followed Phase 2 call shows neither line, the MAC PDUs are not decoding
on that channel — check the per-call census in the call-end log line
(`mac_pdus`, `mac_rs_valid`): `mac_rs_valid=0` on a locked superframe is a
weak-signal / framing condition, not an alias defect, and the knobs above are
the lever. If you see ciphertext lines with `reliable=false` on a strong,
clean channel, that *would* be an alias-decode defect — please open an issue
with the `encoded_hex` and the radio's known name.

## Further reading

- [P25 talker alias — reference](/reference/p25-talker-alias/): message
  framing, carriers, verification.
- [Motorola talker-alias cipher — reference](/reference/motorola-talker-alias-cipher/):
  the recovered algorithm.
- `research/p25-talker-alias-cleanroom-provenance.md` in the repository: how
  the cipher was recovered without reading any GPL source, and how to
  reproduce the validation.
