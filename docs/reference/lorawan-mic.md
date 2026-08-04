---
slug: lorawan-mic
title: LoRaWAN MIC
entry_type: algorithm
category: cryptography
description: The LoRaWAN MIC is a 4-byte Message Integrity Code computed as an AES-CMAC (RFC 4493) over a B0 header plus the frame, keyed under the network session key — it authenticates every data frame, while a separate AES-CTR scheme encrypts the payload.
keywords: LoRaWAN MIC, message integrity code, AES-CMAC, RFC 4493, subkey Rb 0x87, B0 block, NwkSKey, AppSKey, AES-CTR payload encryption, frame authentication
aka: [LoRaWAN MIC, "message integrity code", "AES-CMAC MIC"]
autolink: true
infobox:
  - { label: Algorithm, value: "AES-CMAC (RFC 4493)" }
  - { label: MIC length, value: 4 bytes }
  - { label: Rb constant, value: "0x87" }
  - { label: Keyed under, value: NwkSKey }
see_also: [lorawan, lora, advanced-encryption-standard]
cite_urls:
  - https://datatracker.ietf.org/doc/html/rfc4493
  - https://en.wikipedia.org/wiki/LoRa
---

The **LoRaWAN MIC** (Message Integrity Code) is the 4-byte tag that authenticates every
[LoRaWAN](/reference/lorawan/) data frame: a receiver that shares the network session key can
confirm the frame is genuine and unaltered, and reject anything that fails.[^rfc] It is an
**AES-CMAC** (RFC 4493) computed over a synthetic B0 header block prepended to the frame, using
[AES](/reference/advanced-encryption-standard/) as its underlying block cipher — the same
primitive that, in a different mode, encrypts the payload.[^lora]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="AES-CMAC over a B0 block followed by the frame: the key encrypts a zero block to L, from which subkeys K1 and K2 are derived by a left-shift and conditional XOR with 0x87; the message blocks are CBC-chained and the last block is XORed with K1 or K2, then encrypted, and the first four output bytes are the MIC." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="24" width="70" height="24" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1"/>
  <text x="53" y="40" text-anchor="middle" font-size="8" fill="currentColor">B0 block</text>
  <rect x="92" y="24" width="60" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="122" y="40" text-anchor="middle" font-size="8" fill="currentColor">block 1</text>
  <rect x="156" y="24" width="60" height="24" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1"/>
  <text x="186" y="40" text-anchor="middle" font-size="8" fill="currentColor">…</text>
  <rect x="220" y="24" width="72" height="24" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/>
  <text x="256" y="40" text-anchor="middle" font-size="8" fill="currentColor">last ⊕ K1/K2</text>
  <path d="M53 48 L53 66 M122 48 L122 66 M186 48 L186 66 M256 48 L256 66" stroke="currentColor" stroke-width="1" fill="none"/>
  <rect x="18" y="66" width="274" height="22" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="155" y="81" text-anchor="middle" font-size="8" fill="currentColor">CBC chain under AES_K</text>
  <path d="M292 77 L330 77" stroke="currentColor" stroke-width="1.1" fill="none" marker-end="url(#lmar)"/>
  <defs><marker id="lmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="330" y="66" width="110" height="22" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="385" y="81" text-anchor="middle" font-size="8" fill="currentColor">MIC = first 4 bytes</text>
  <text x="18" y="118" font-size="7.5" fill="currentColor">L = AES_K(0¹²⁸) · K1 = shift(L) ⊕ 0x87 (if MSB set) · K2 = shift(K1) ⊕ 0x87</text>
  <text x="18" y="136" font-size="7.5" fill="currentColor">last block complete → ⊕ K1 · incomplete → append 0x80 pad, ⊕ K2</text>
</svg>
<figcaption>The MIC is an AES-CMAC: derive subkeys K1/K2 from the key via a left-shift and conditional XOR with 0x87, CBC-chain the B0 block and message under AES, XOR the final block with the right subkey, encrypt, and take the first four output bytes.</figcaption>
</figure>

## How it works

AES-CMAC first derives two subkeys. The cipher encrypts an all-zero block to get `L`, and each
subkey is a one-bit left shift of the previous value, conditionally XORed with the constant
`Rb = 0x87` when the shifted-out high bit was set — the spec's generator for a 128-bit block:

```go
// shiftLeftXorRb: one-bit left shift, XOR Rb (0x87) when the high bit was set.
func shiftLeftXorRb(in [16]byte) [16]byte {
    var out [16]byte
    var carry byte
    for i := 15; i >= 0; i-- {
        out[i] = in[i]<<1 | carry
        carry = in[i] >> 7
    }
    if in[0]&0x80 != 0 {
        out[15] ^= 0x87
    }
    return out
}
```

`K1` comes from shifting `L`, `K2` from shifting `K1`. The message is then split into 16-byte
blocks and CBC-chained under the key. The final block gets special handling: if the message is a
whole number of blocks it is XORed with `K1`; otherwise it is padded with a `0x80` byte (then
zeros) and XORed with `K2`. Encrypting that last block yields a 16-byte value whose **first four
bytes are the MIC**. GopherTrunk's `VerifyMIC` recomputes this under the network session key and
compares against the received tag in constant time.

## The B0 block

CMAC is not run over the frame alone — LoRaWAN prepends a 16-byte **B0** block that binds the MIC
to the frame's context, so a valid tag cannot be replayed on a different address, direction or
frame counter:

| Byte(s) | Contents |
| --- | --- |
| 0 | `0x49` (block-type marker) |
| 5 | direction — 0 uplink, 1 downlink |
| 6–9 | device address (little-endian) |
| 10–13 | 32-bit frame counter (little-endian) |
| 15 | length of the MAC payload |

The MIC is computed over `B0 ‖ frame`, so tampering with the address, the counter, or the
direction changes the input and invalidates the tag.

## Payload encryption

Authentication and confidentiality are separate mechanisms. The frame payload is encrypted with
an **AES-CTR-style** scheme (LoRaWAN 1.0 §4.3.3): AES encrypts a per-block `A` counter block
(marker `0x01`, direction, device address, frame counter, and a block index in its last byte)
to produce a keystream, which is XORed with the payload. Because XOR is symmetric, the same
routine encrypts and decrypts. The key differs by port — the application session key for
`FPort > 0`, the network session key for `FPort == 0` MAC-command payloads — while the MIC is
always keyed under the network session key. Key material itself is held in the keystore and
never logged.

## Sources

[^rfc]: [RFC 4493 — The AES-CMAC Algorithm](https://datatracker.ietf.org/doc/html/rfc4493) — IETF, the subkey generation (Rb = 0x87), padding rules and CMAC construction.
[^lora]: [LoRa](https://en.wikipedia.org/wiki/LoRa) — Wikipedia, on LoRaWAN's security, session keys and frame format.
