---
layout: page
title: NXDN Scrambler (15-bit voice privacy)
description: What GopherTrunk decodes, descrambles and can (and cannot) brute-force for NXDN's optional 15-bit scrambler, and the end-to-end workflow for an operator who owns the radio and knows the key.
nav_group: Reference
---

# NXDN Scrambler

NXDN (IDAS / NEXEDGE) radios have an optional built-in **15-bit scrambler**
(Kenwood / Icom "scrambler" privacy). It is **voice obfuscation, not
encryption**: a 15-bit LFSR generates a keystream that is XORed onto the AMBE+2
voice slots. It is *not* DES/AES and GopherTrunk never treats it as such. (True
NXDN DES/AES encryption is identified and flagged, but not decrypted.)

Key facts that trip people up:

- **Only the voice slots are scrambled.** The SACCH / CAC signalling that
  carries the call's VCALL (parties, cipher type, key id) stays **clear** on
  air — otherwise the system could not route calls.
- **The key space is 2^15 (0..32767).** Small, but see "key recovery" below —
  having a tiny key space is not the same as having a shipped voice attack.
- **GopherTrunk's scrambler model is reference-derived and
  capture-validated** on known-key captures (issue #1261), but it is still
  worth confirming against your own radio.

## If you know the key (you own the radio)

This is the common, fully-supported case. Give GopherTrunk the key and it
descrambles **in-process** — the recording is clear audio, no offline tooling
needed.

Conventional scan-list channel:

```yaml
scanner:
  conventional:
    - label: "Plant ops"
      frequency_hz: 447100000
      mode: nxdn                 # or nxdn48 for 6.25 kHz
      squelch_dbfs: -50
      encryption_keys:
        - key_id: 3              # the key slot the radios were programmed with
          algorithm: scrambler
          key: "32767"           # the radio's DECIMAL 15-bit key, 1..32767
```

The same `encryption_keys` block works on a trunking system. The `key_id` must
match the slot the radios use; it is printed in the daemon debug log on each
VCALL and on the `call.encryption` bus event. If the audio comes out clear,
the model is confirmed for your radio.

### Capturing the ciphertext

When **no** matching key is configured, the voice chain cannot descramble, so
the scrambled AMBE+2 frames fall through to the recording sidecars:

```yaml
recordings:
  path: /your/recordings
  write_raw: true          # .raw sidecar: packed AMBE+2 voice frames
  skip_encrypted: false    # a scrambled call is flagged encrypted; keep false
                           # or the files are deleted
  # mute_encrypted: true   # optional: silences the PCM only; .raw still holds
                           # the ciphertext frames
```

Each `.raw` frame is one NXDN voice frame's 49 payload bits packed MSB-first
into 7 bytes (so 56 bits per line). Hex-dump with `xxd -p -c7 call.raw`.

## Key recovery (offline cryptolab)

The `cryptolab` toolkit is **optional and build-gated**. A default install does
**not** have it — `gophertrunk cryptolab …` will say so. Build it in:

```
make build TAGS=cryptolab
```

`cryptolab nxdn` has two modes:

```
gophertrunk cryptolab nxdn descramble -in <frames.hex> -key <0..32767> [-frame-type sacch|cac|ccitt]
gophertrunk cryptolab nxdn brute      -in <frames.hex> [-oracle crc|known-pt] [-frame-type sacch|cac|ccitt] [-known-pt <crib.hex>]
```

Input is **one scrambled frame per line, as hex (packed bits, MSB-first)** —
the same shape as the `.raw` sidecar.

### What the oracles can and cannot do

`brute` sweeps all 32 768 keys and keeps the ones an **oracle** accepts:

- **`crc` oracle** (`sacch` / `cac` / `ccitt`) — descrambles and re-checks an
  NXDN frame CRC. **This does not recover a voice-call key**, because the voice
  slots are the only thing scrambled and they carry no CRC, while the SACCH/CAC
  that *do* carry a CRC are sent clear. Feeding the voice `.raw` here now
  returns a pointed "these look like AMBE+2 voice frames" diagnostic instead of
  a silent "no key".
  Also note the CRC oracle alone returns a **coset of candidate keys** (≈2^9 for
  SACCH's 6-bit CRC), not a unique key — the LFSR and CRC are both linear.
- **`known-pt` oracle** — confirms a key against a plaintext crib you already
  hold. Works on voice, but you need the plaintext first.
- **AMBE/FEC voice oracle** — the oracle that would recover a voice key from
  ciphertext alone is **not shipped yet** (roadmap). So there is currently **no
  one-command "feed the .raw hex to brute and get the key"** for a voice-only
  scrambled call.

### A validation you *can* run with a key you own

Because you own the radio and know the key, you can confirm the whole pipeline
end to end:

1. Capture the ciphertext (`write_raw`, no key) → `call.raw`.
2. `gophertrunk cryptolab nxdn descramble -in <hex of call.raw> -key <yourkey>`
   and confirm the frames come out as the expected plaintext AMBE (the same
   bytes GopherTrunk produces in-process when you *do* configure the key).
3. Use that plaintext as a crib for `brute -oracle known-pt` and watch it
   recover your key from the 32 768-key sweep — a demonstration that the
   scrambler model and keystream schedule are correct for your hardware.

This known-key round-trip is exactly the capture that settles whether the
polynomial/seed mapping needs any adjustment, and is the most useful thing to
contribute back on a radio you control.
