---
title: "The Operator's Cookbook Vol. 2, Part 8: Your Own Keys — Decrypting What You're Authorised to Hear"
description: "A copy-paste GopherTrunk recipe for known-key decryption: encryption_keys on a P25, DMR or NXDN system and on a scan-list channel, mute_encrypted versus skip_encrypted, and an offline check with replay -key. Each cipher's verification status is stated plainly, from on-air verified P25 ADP to reference-pinned AES."
category: tutorials
keywords: p25 adp decryption sdr, dmr enhanced privacy rc4 key, nxdn scrambler key scanner, encryption_keys config, decrypt own radio system sdr, p25 aes known key, mute encrypted calls scanner, kfdtool ekc key container, gophertrunk encryption keys
tags: [operator-cookbook, cookbook-volume-2, encryption, p25, dmr, nxdn]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 8
---

*Part 8 of **The Operator's Cookbook, Volume 2**, a 14-part run of complete,
copy-paste GopherTrunk builds — one working rig per part — built on what
landed after Volume 1.
[Part 7]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }})
put a transcript on every call. This part is for the operator who already
holds the keys: a radio fleet you administer, two DMR handhelds you
programmed yourself, an NXDN channel whose scrambler code sits in your own
codeplug. GopherTrunk decrypts those calls in-process with the key you give
it. It performs no key recovery of any kind, and neither does this recipe.*

> **TL;DR:** Decryption is one block, `encryption_keys`, under a trunked
> system or a scan-list channel: `key_id` (the ID the radios transmit),
> `algorithm` (normalised to `rc4`, `des`, `tdes`, `aes` or `scrambler`)
> and `key` — hex, or for the NXDN scrambler the radio's decimal key
> 1..32767. The key's **length** picks the variant (TDES 16 or 24 bytes,
> AES 16 or 32), and the P25 chain refuses a key whose length disagrees
> with the algorithm on air. Calls no key covers go silent with
> `recordings.mute_encrypted`, or leave no file with `skip_encrypted`.
> Check a key offline with `gophertrunk replay -key ALG:KEYID:HEXKEY`.
> Status: P25 ADP is on-air verified
> ([#1187](https://github.com/MattCheramie/GopherTrunk/issues/1187)); DMR
> Enhanced Privacy and the NXDN scrambler are capture-verified; P25
> DES-OFB, TDES and AES are reference-pinned, with no capture yet.

**Key takeaways**

- **One block serves every protocol.** P25, DMR and NXDN keys are all
  `encryption_keys` entries; only the algorithm name and key format change.
- **The key ID is matched, the length decides the variant.** An entry is
  used only when the call names its `key_id`, and a 16-byte AES key is
  never tried against AES-256 traffic.
- **Mute while you commission keys; skip only when you are sure.**
  `mute_encrypted` keeps the file and the raw frames, so a wrong key ID
  costs nothing. `skip_encrypted` deletes the evidence.
- **Verification differs per cipher.** ADP has decoded a real call; AES
  has only matched a reference decoder's constants.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Keys for a trunked system | descrambles calls whose key ID matches an entry | `trunking.systems[].encryption_keys` |
| Keys for a scan-list channel | the NXDN scrambler on a conventional channel | `scanner.conventional[].encryption_keys` |
| One key ID, a key per talkgroup | scopes an entry to calls on one talkgroup | `talkgroup` on the entry |
| Keys encrypted at rest | loads traffic keys from a KFDtool container | `encryption_key_files`, managed with `gophertrunk ekc` |
| Calls no key covers | silent audio; the sidecars keep the ciphertext | `recordings.mute_encrypted` |
| No recording at all | discards the files of a call no key decrypts | `recordings.skip_encrypted` |
| Offline key check | runs a recording through the daemon's voice path | `gophertrunk replay -record-voice -key` |

## In this post

- **What you're building** — one lookup, three protocols, who it is for.
- **The config** — the complete file, then the same block on DMR.
- **First run — what healthy looks like** — log lines, the offline check,
  each cipher's status.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — key containers, ciphertext sidecars, tuner policy.

## What you're building

A rig that records clear speech for the calls your keys cover, and handles
every other encrypted call on purpose rather than by accident.

The mechanism is the same on all three protocols, because each announces
its cipher in the clear. P25 Phase 1 carries an
[Encryption Sync]({{ '/reference/p25-encryption-sync/' | relative_url }})
in every LDU2; DMR sends a Privacy Indicator header after the Voice LC
Header; NXDN names the cipher type and a key ID in its SACCH VCALL. The
voice chain reads that announcement, looks the key up by system,
algorithm, [key ID]({{ '/reference/key-id-algid/' | relative_url }}) and
talkgroup, and — if you supplied one — XORs the keystream off the vocoder
frames before the vocoder sees them. The call is still logged as
encrypted, with its algorithm and key ID.

The cipher internals are other posts' business:
[P25 End to End Part 9]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }}),
[DMR End to End Part 12]({{ '/blog/deep-dives/dmr-end-to-end-12-enhanced-privacy-rc4/' | relative_url }}),
[Rebuilt From the Air Part 9]({{ '/blog/solution-postmortem/rebuilt-from-the-air-09-nxdn-scrambler/' | relative_url }})
for the NXDN scrambler, and
[Trunking Engine Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})
for what a voice tuner does with ciphertext. This part is the operator's
side: where the key goes and how to tell it worked.

Who this is for, once. The recipe assumes keys you already hold for
systems you are authorised to monitor — your own radios, a system you
administer or are contracted to support. Holding a key and being allowed
to use it are separate questions, and the law varies by country and by
state: read the
[legal guide]({{ '/police-scanner-legal/' | relative_url }}) and
[encryption explained]({{ '/police-scanner-encryption/' | relative_url }}),
then check your own jurisdiction.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="How a configured key reaches a call: the call announces its algorithm and key ID through the P25 Encryption Sync, the DMR Privacy Indicator header or the NXDN VCALL; the voice chain looks for a talkgroup-scoped entry, then a system entry; a key of the right length descrambles the frames into a clear recording, while a call with no matching key is either muted with its ciphertext kept in the sidecars or skipped with no file at all">
  <rect x="8" y="66" width="158" height="108" rx="4" fill="none" stroke="currentColor"/>
  <text x="87" y="84" text-anchor="middle" fill="currentColor" font-size="10">the call names its cipher</text>
  <text x="87" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">P25: Encryption Sync (LDU2)</text>
  <text x="87" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="9">DMR: Privacy Indicator header</text>
  <text x="87" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="9">NXDN: SACCH VCALL</text>
  <text x="87" y="160" text-anchor="middle" fill="currentColor" font-size="9">algorithm + key ID, in clear</text>
  <line x1="166" y1="120" x2="194" y2="120" stroke="currentColor"/>
  <rect x="194" y="78" width="158" height="84" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="273" y="98" text-anchor="middle" fill="var(--accent)" font-size="10">encryption_keys lookup</text>
  <text x="273" y="116" text-anchor="middle" fill="var(--fg-muted)" font-size="9">1. key_id + this talkgroup</text>
  <text x="273" y="131" text-anchor="middle" fill="var(--fg-muted)" font-size="9">2. key_id, any talkgroup</text>
  <text x="273" y="149" text-anchor="middle" fill="var(--fg-muted)" font-size="9">length must fit the algorithm</text>
  <line x1="352" y1="100" x2="386" y2="52" stroke="var(--accent)"/>
  <line x1="352" y1="140" x2="386" y2="186" stroke="currentColor"/>
  <rect x="386" y="28" width="132" height="48" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="452" y="47" text-anchor="middle" fill="var(--accent)" font-size="10">key found</text>
  <text x="452" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="9">frames descrambled</text>
  <line x1="518" y1="52" x2="540" y2="52" stroke="var(--accent)"/>
  <rect x="540" y="28" width="132" height="48" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="606" y="47" text-anchor="middle" fill="var(--accent)" font-size="10">clear recording</text>
  <text x="606" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="9">logged with alg + key ID</text>
  <rect x="386" y="162" width="132" height="48" rx="4" fill="none" stroke="currentColor"/>
  <text x="452" y="181" text-anchor="middle" fill="currentColor" font-size="10">no key</text>
  <text x="452" y="197" text-anchor="middle" fill="var(--fg-muted)" font-size="9">frames stay ciphertext</text>
  <line x1="518" y1="176" x2="540" y2="146" stroke="currentColor"/>
  <line x1="518" y1="196" x2="540" y2="212" stroke="currentColor"/>
  <rect x="540" y="122" width="132" height="48" rx="4" fill="none" stroke="currentColor"/>
  <text x="606" y="141" text-anchor="middle" fill="currentColor" font-size="10">mute_encrypted</text>
  <text x="606" y="157" text-anchor="middle" fill="var(--fg-muted)" font-size="9">silent audio, .raw kept</text>
  <rect x="540" y="184" width="132" height="48" rx="4" fill="none" stroke="currentColor"/>
  <text x="606" y="203" text-anchor="middle" fill="currentColor" font-size="10">skip_encrypted</text>
  <text x="606" y="219" text-anchor="middle" fill="var(--fg-muted)" font-size="9">no file, no upload</text>
</svg>
<figcaption>The call says which key it needs; the config either has an entry of the right length for that key ID, or the call is handled as ciphertext in the way you chose.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| A rig that already receives the system | $0 | [Volume 1 Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }}) for P25, [Part 3]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }}) for a DMR repeater, [this volume's Part 3]({{ '/blog/tutorials/cookbook-volume-2-03-nxdn/' | relative_url }}) for NXDN |
| Three facts per key | $0 | from your own codeplug or key loader: the key ID, the algorithm, the key |
| A known-key recording (optional) | $0 | IQ or discriminator audio of one of your own radios, for the offline check |

## The config

A P25 system on one wideband dongle, plus an NXDN channel on the scan
list, which runs on its own voice SDR. Every key below is an all-zero
**placeholder**: it parses, it is the right length, and it decrypts
nothing.

```yaml
log:
  level: info

storage:
  path: "../data/calls.db"

recordings:
  dir: "../recordings"
  write_raw: true          # keep the vocoder frames beside every WAV
  mute_encrypted: true     # silence when no key fits
  skip_encrypted: false    # keep the file and the uploads

sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000001"            # from `gophertrunk sdr list`
      role: wideband
      gain: "auto"
      center_freq_hz: 858_000_000
      voice_taps: 2
      channels:
        - frequency_hz: 857_262_500 # your control channel
          system: "Plant-P25"
    - serial: "00000002"
      role: voice                   # the scan list runs on this one
      gain: "auto"

trunking:
  systems:
    - name: "Plant-P25"
      protocol: p25
      control_channels: [857_262_500]
      encrypted_calls:
        mode: metadata              # no key: log it, then release
        metadata_follow_ms: 1500
      encryption_keys:
        - key_id: 1                 # the key ID your radios transmit
          algorithm: adp            # P25 ADP: RC4 with a 40-bit key
          key: "0000000000"         # PLACEHOLDER: 10 hex digits
        - key_id: 2
          algorithm: aes            # 64 hex digits = AES-256, 32 = AES-128
          key: "0000000000000000000000000000000000000000000000000000000000000000"
        - key_id: 2
          talkgroup: 4101           # same key ID, another key on this group
          algorithm: aes
          key: "0000000000000000000000000000000000000000000000000000000000000000"

scanner:
  conventional:
    - label: "Plant ops (NXDN96)"
      frequency_hz: 461_037_500
      mode: nxdn                    # or nxdn48
      squelch_dbfs: -50
      talkgroup_id: 2147500010
      encryption_keys:
        - key_id: 2                 # the radios' key slot, 0..63
          algorithm: scrambler
          key: "1"                  # PLACEHOLDER: DECIMAL key, 1..32767
```

**`key_id`** is the number the radios put on the air: the KID in a P25
Encryption Sync, the key ID in a DMR Privacy Indicator header, the key
slot an NXDN VCALL names. If you do not know it, run once without the key
— the log prints it for every encrypted call.

**`algorithm`** is forgiving about spelling. `rc4`, `arc4` and `adp` are
one family (DMR Enhanced Privacy and P25 ADP are the same
[RC4]({{ '/reference/rc4-cipher/' | relative_url }}) construction); then
`des`, `tdes`, `aes` and `scrambler`. Anything else is rejected at load.

**`key`** is hex, with spaces and a leading `0x` tolerated — except for the
[NXDN scrambler]({{ '/reference/nxdn-scrambler/' | relative_url }}), whose
key is the decimal number the codeplug shows. Lengths are checked: DES 8
bytes, TDES 16 or 24, AES 16 or 32, the RC4 family 1 to 32. A wrong length
stops the daemon at start-up with the entry named:

```text
trunking.systems[0].encryption_keys[0]: a aes key is 16 or 32 bytes (32 or 64 hex digits), got 2 bytes
```

The length also picks the variant: a 16-byte `aes` entry is applied to
AES-128 traffic and refused, with a WARN, on AES-256.

**`talkgroup`** scopes an entry to one talkgroup, for fleets that program
every group with the same key ID and a different key
([#1261](https://github.com/MattCheramie/GopherTrunk/issues/1261)). The
lookup tries the call's talkgroup first, then the entry without one.

A call whose key ID matches an entry is exempt from `encrypted_calls`
and always followed.

**`mute_encrypted` and `skip_encrypted`** both spare a call a configured
key decrypts. For the rest, mute records silence while the `.raw` sidecar
keeps the ciphertext and the call is logged and uploaded as before; skip
deletes the files, so no completed-call event reaches a webhook or an Rdio
Scanner feed. Until a call's first Encryption Sync or PI header arrives
its key ID is unknown, and the recorder then treats it as decryptable if
the system lists any key at all — the opening frames of your own calls
are never muted while the ID is in flight.

Keys in this file are plain text: treat `config.yaml` like the codeplug it
duplicates. The config builder's API returns them as `<redacted>`. The
same block under a DMR system, for the rig from Volume 1 Part 3:

```yaml
trunking:
  systems:
    - name: "Yard-DMR"
      protocol: dmr-tier2
      control_channels: [452_662_500]   # the repeater's output
      encryption_keys:
        - key_id: 5                 # from the Privacy Indicator header
          algorithm: rc4            # DMR Enhanced Privacy
          key: "0000000000"         # PLACEHOLDER
```

## First run — what healthy looks like

```sh
gophertrunk run -config config.yaml
```

Once at start-up, when any system or channel carries a key:

```text
INF daemon: in-process decryption keys configured (DMR enhanced privacy / P25 ADP, DES, AES / NXDN scrambler)
```

Then per call. P25 names the key in its first LDU2; the chain holds that
one superframe (at most 360 ms) until it does, so the first syllable is
decrypted too:

```text
INF composer: p25p1 key resolved — decrypting in-process serial=... system=Plant-P25 alg="0xAA (ADP/RC4)" key_id=1
```

DMR and NXDN print the header they decoded; look for `key_configured`:

```text
INF composer: dmr PI header — call is encrypted system=Yard-DMR ... algorithm=rc4 ... key_id=5 ... key_configured=true ...
INF composer: nxdn VCALL — call is encrypted system=... ... cipher=scrambler cipher_type=1 key_id=2 key_configured=true
```

A call you hold no key for ends like this with the mute on; the last two
fields say the silence is deliberate
([Field Notebook Part 10]({{ '/blog/tutorials/field-notebook-10-recorder-lines/' | relative_url }})
reads the rest):

```text
INF recorder: call ended device=... wav=... duration=... reason=... muted_frames=... encrypted_no_key=true
```

**The offline check.** A log line proves a key was *matched*, not that it
is *right*: the wrong key under the right key ID produces vocoder noise
and no error. `gophertrunk replay` runs a recording of one of your own
transmissions through the daemon's voice path — discriminator audio (a
receiver's raw, unsquelched FM output) first, then an IQ capture:

```sh
KEY_ID=5
KEY_HEX=0000000000

gophertrunk replay -in my-handheld.wav -format disc -protocol dmr-tier2 \
  -freq 1 -record-voice -out-dir rec -key "rc4:${KEY_ID}:${KEY_HEX}"

gophertrunk replay -in my-handheld.cs16 -format cs16 -sample-rate 2400000 \
  -protocol dmr-tier2 -freq 1 -record-voice -out-dir rec \
  -key "rc4:${KEY_ID}:${KEY_HEX}"
```

`-key` is `ALG:KEYID:HEXKEY`, repeatable, and validated like a config
entry; `KEY_HEX` above is the placeholder again. `-format disc` first
prints a sanity check of the audio —

```text
replay: discriminator audio my-handheld.wav at ... Hz, remodulated at ±... Hz full scale; energy below 3 kHz in its most signal-like 20 ms blocks: ...% (consistent with a discriminator tap)
```

— and the call lands under `rec/replay/<talkgroup>/` as a WAV. Listen to
it: intelligible speech is the confirmation, and nothing else is. One
limit: `replay` records only what it can grant on the carrier it decodes.
A conventional DMR recording grants itself; a P25 voice channel recorded
without its control channel grants nothing, so check ADP live.

**What is verified, cipher by cipher.**

| Cipher | Status | What that means |
|---|---|---|
| P25 ADP (RC4), Phase 1 | on-air verified | a reporter's recording of a real call decodes to speech with its key, and the layout is pinned as literal vectors. A live daemon call on that rig is still open |
| P25 DES-OFB, TDES, AES, Phase 1 | reference-pinned | layout and AES IV expansion match a reference decoder's constants; no real DES or AES call has been decoded |
| DMR Enhanced Privacy (RC4) | capture-verified | two known-key captures decode to speech, and the `replay -key` path is pinned on committed real-air frames. A live call is still open |
| NXDN scrambler (15-bit key) | capture-verified | two known-key captures descramble to speech. A live run on a scan-list channel is still open |
| Everything else | not decrypted | P25 Phase 2 voice, P25 DES-XL, DMR DES and AES, Hytera and Kirisun privacy, Motorola Basic Privacy, NXDN DES and AES: identified, logged, recorded as ciphertext |

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| Daemon stops at start-up naming an entry | wrong key length, bad hex, unknown algorithm, or a duplicate | the message names the entry by index, as in `encryption_keys[0]`; fix that line |
| Call logged as encrypted but never decrypted | the ID in the config is not the ID on air, or the fleet uses a key per talkgroup | copy the number from the log into `key_id` (on NXDN the key slot, 0..63), and scope entries with `talkgroup` if needed |
| Key matched, audio is still noise | right key ID, wrong key | run the offline check; an all-zero placeholder left in `key` is the usual culprit |
| Your own calls record as silence | the mute fired because no entry matched | find `encrypted_no_key=true` on the ended line, then fix the ID or the scope |
| Encrypted calls leave no file and no webhook fires | the skip option discards the call before it completes | set `skip_encrypted: false` and use the mute |
| The disc check says the file is not a demodulable discriminator tap | speaker audio or a decoder's playback was recorded | record IQ with `gophertrunk capture`, or the receiver's raw output |

### How this recipe shapes operator practice

- **Read the ID off the air, never from memory.** The log prints the
  algorithm and key ID of every encrypted call whether or not you hold the
  key.
- **Prove a key once, offline.** A matched key and a correct key look the
  same in the log; only audio tells them apart.
- **Keep the evidence while you are learning the system.** The NXDN work
  found a clear call whose one misread header let `skip_encrypted` delete a
  good recording; the chain now wants two agreeing superframes. Mute keeps
  the file either way.

## Variations

**Keys encrypted at rest.** A system can load its keys from a
[KFDtool]({{ '/reference/key-loader-kfd/' | relative_url }}) `.ekc`
container. The daemon decrypts it at start-up, adds its traffic keys to
the system's list (an explicit entry with the same key ID wins) and never
writes them back:

```yaml
trunking:
  systems:
    - name: "Plant-P25"
      protocol: p25
      control_channels: [857_262_500]
      encryption_key_files:
        - path: "../config/keys.ekc"
          password_env: GT_EKC_PASSWORD   # unset = asked at start-up
          keyset: 0                       # 0 = every keyset
```

`gophertrunk ekc -in keys.ekc` lists what would load without printing key
material, and `gophertrunk ekc import` moves a system's plain-text keys
into a container.

- **A tighter tuner policy.** `encrypted_calls.mode: ignore` gives up a
  voice SDR as soon as a call names a key you do not hold; your own keyed
  calls are still followed.
- **NXDN on a trunked system.** Scrambler entries are accepted under an
  `nxdn` system's `encryption_keys` exactly as on a channel.

The full reference is the
[encryption doc page]({{ '/dmr-encryption.html' | relative_url }}).

## Where this goes next

Keys decide what you can hear; the next part is about choosing what you
*do* hear, minute to minute.
[Part 9]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }})
puts the scanner's front panel back: hold a talkgroup through an incident,
avoid a chatty one for an hour, and lock out a noisy frequency in a way
that survives a restart.

## FAQ

**Can GopherTrunk decrypt encrypted P25 or DMR calls?**
Only with a key you supply for a system you are authorised to monitor:
P25 Phase 1 ADP, DES-OFB, TDES and AES, DMR Enhanced Privacy and the NXDN
scrambler, when an `encryption_keys` entry matches the call's key ID. It
performs no key recovery; without the key a call stays ciphertext.

**What is the difference between mute_encrypted and skip_encrypted?**
`mute_encrypted` records silence for a call no key covers but keeps the
file, the History row and the ciphertext in the raw sidecar.
`skip_encrypted` deletes the recording, which also suppresses the
completed-call backends. Neither touches a call your key decrypts.

**Is P25 AES decryption verified on air?**
No. It is reference-pinned: the frame layout and the AES IV expansion
match a reference decoder's constants and a synthetic test, but no real
AES call has been decoded. P25 ADP is the on-air verified one.

**How do I check a decryption key without a live radio?**
Record one of your own transmissions as IQ or discriminator audio, then
run `gophertrunk replay` with `-record-voice` and `-key ALG:KEYID:HEXKEY`.
It uses the daemon's voice path, so intelligible audio in the output WAV
means the key is right.

## Series navigation

**Part 8 of 14** · ←
[Part 7: Transcription — Whisper on Every Call]({{ '/blog/tutorials/cookbook-volume-2-07-transcription/' | relative_url }})
· Next →
[Part 9: Hold, Avoid and Priority — Driving It Like a Scanner]({{ '/blog/tutorials/cookbook-volume-2-09-hold-avoid-priority/' | relative_url }})
