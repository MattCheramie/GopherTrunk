---
slug: crc-16-ccitt
title: CRC-16/CCITT
entry_type: algorithm
category: error-correction
description: CRC-16/CCITT is the family of 16-bit cyclic redundancy checks built on polynomial 0x1021 — the same polynomial that, by changing the initial value, bit reflection and final XOR, yields the distinct CCITT/FALSE, XMODEM and augmented variants GopherTrunk reuses across P25, YSF, M17, MDC1200 and LoRa.
keywords: CRC-16 CCITT, polynomial 0x1021, CCITT FALSE, XMODEM CRC, augmented CRC, init value, bit reflection, final XOR, P25 TSBK, YSF FICH, MDC1200 CRC
aka: [CRC-16/CCITT, "CRC-CCITT", "CRC-16 0x1021"]
autolink: true
infobox:
  - { label: Polynomial, value: "0x1021" }
  - { label: Width, value: 16 bits }
  - { label: Varies by, value: "init, reflection, final XOR" }
  - { label: Used by, value: "P25, YSF, M17, MDC1200, LoRa" }
see_also: [cyclic-redundancy-check, forward-error-correction, mdc1200-frame, lora-whitening]
cite_urls:
  - https://en.wikipedia.org/wiki/Cyclic_redundancy_check
  - https://reveng.sourceforge.io/crc-catalogue/16.htm
---

**CRC-16/CCITT** is not one checksum but a *family* of them — every member built on the same
16-bit generator polynomial `0x1021` (`x¹⁶ + x¹² + x⁵ + 1`).[^crc] What makes two members
produce different check values on the same bytes is not the polynomial but three surrounding
parameters: the register's **initial value**, whether input and output bits are **reflected**,
and a **final XOR** applied to the result. GopherTrunk reuses this one polynomial under several
parameter sets across its protocols, which is why getting the parameters right matters more than
the polynomial ever does.[^cat]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A 16-bit shift register seeded with an initial value takes message bits in, XORing the polynomial 0x1021 back on overflow; the same register produces different final check values depending on the initial value, whether bits are reflected, and a final XOR applied at the end." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="40" width="70" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
  <text x="53" y="57" text-anchor="middle" font-size="8" fill="currentColor">init value</text>
  <path d="M88 53 L120 53" stroke="currentColor" stroke-width="1.1" fill="none" marker-end="url(#car)"/>
  <defs><marker id="car" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="120" y="38" width="150" height="30" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="195" y="50" text-anchor="middle" font-size="8" fill="currentColor">16-bit register</text>
  <text x="195" y="62" text-anchor="middle" font-size="7.5" fill="currentColor">XOR 0x1021 on overflow</text>
  <path d="M195 68 L195 92 L120 92 L120 74" stroke="currentColor" stroke-width="1" fill="none" stroke-dasharray="2 2"/>
  <text x="150" y="106" font-size="7" fill="currentColor">feedback</text>
  <path d="M270 53 L302 53" stroke="currentColor" stroke-width="1.1" fill="none" marker-end="url(#car)"/>
  <rect x="302" y="40" width="70" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/>
  <text x="337" y="57" text-anchor="middle" font-size="8" fill="currentColor">final XOR</text>
  <path d="M372 53 L404 53" stroke="currentColor" stroke-width="1.1" fill="none" marker-end="url(#car)"/>
  <text x="430" y="57" text-anchor="middle" font-size="8" fill="currentColor">CRC</text>
  <text x="18" y="132" font-size="7.5" fill="currentColor">same polynomial 0x1021 · different init / reflection / final XOR → different check value</text>
</svg>
<figcaption>Every CRC-16/CCITT variant runs the same 0x1021 feedback register; the initial value, bit reflection and final XOR are what separate CCITT/FALSE from XMODEM from the augmented form.</figcaption>
</figure>

## How it works

A CRC treats the message as a big binary polynomial and returns the remainder after dividing it
by the generator — computed with a 16-bit shift register that, on each step, shifts a message bit
in and XORs `0x1021` back whenever the register overflows the top bit. It is a
[cyclic redundancy check](/reference/cyclic-redundancy-check/) in the textbook sense; the
"CCITT" label just fixes the polynomial. GopherTrunk's `CRCCCITTWithInit` is the parametric core
— MSB-first, no reflection — with the initial value passed in, and the reflected and augmented
forms are separate functions with their own bit order.

## The variants GopherTrunk uses

The same `0x1021` appears under these parameter sets. The check value is the CRC of the ASCII
string `"123456789"`, the standard reference vector:

| Variant | Init | Reflect | Final XOR | Check | Used by |
| --- | --- | --- | --- | --- | --- |
| CCITT/FALSE | `0xFFFF` | no | `0x0000` | `0x29B1` | P25 TSBK (stored complemented) |
| XMODEM | `0x0000` | no | `0x0000` | `0x31C3` | YSF FICH, M17, LoRa payload |
| Reflected | `0x0000` | in/out | `0xFFFF` | — | [MDC1200](/reference/mdc1200-frame/) |
| Augmented | `0x0000` | no | `0xFFFF` | — | P25 TSBK trailer |

The [MDC1200](/reference/mdc1200-frame/) reflected form runs the polynomial in its bit-reversed
representation `0x8408`; the [LoRa](/reference/lora-whitening/) payload CRC is the plain XMODEM
member.

## Init, reflection and final XOR

Each parameter changes the result independently:

- **Initial value.** Seeding the register with `0xFFFF` instead of `0x0000` makes the CRC
  sensitive to leading zero bytes — a run of zeros at the start of a message no longer leaves the
  register at zero, so prepended zeros are detected.
- **Reflection.** Reflecting input and output bits processes each byte LSB-first, matching how
  UART/serial hardware clocks bits out. It changes the value but not the error-detection strength;
  it is a convention, chosen to match the wire.
- **Final XOR.** XORing the register with `0xFFFF` at the end guards against trailing zero bytes
  the same way a non-zero init guards against leading ones.

Two implementations sharing the polynomial but differing in any of these will disagree on every
input — which is exactly the failure mode below.

## The augmented variant and issue #275

The P25 TSBK trailer uses the **augmented-message** form, which is a genuinely different
algorithm, not just different parameters: message bits are shifted *into* the low end of the
register and the polynomial is XORed when the register grows past 16 bits, with a final XOR of
`0xFFFF`. The encoder computes the trailer as the CRC of `info ‖ 16 zero bits`; a receiver checks
that the CRC of `info ‖ trailer` is zero. GopherTrunk's original P25 code used the CCITT/FALSE
function here, and every on-air TSBK on the Mt Anakie capture failed verification even when the
trellis decoder reported a clean path — issue #275. Switching to `CRCCCITTAugmented` made the
on-air trailers verify. The lesson is the recurring one with this family: the polynomial is the
easy part; the init, shift direction and final XOR are where a decoder silently gets it wrong.

## Sources

[^crc]: [Cyclic redundancy check](https://en.wikipedia.org/wiki/Cyclic_redundancy_check) — Wikipedia, on CRC computation, generator polynomials and the init/reflect/xorout parameters.
[^cat]: [CRC catalogue (16-bit)](https://reveng.sourceforge.io/crc-catalogue/16.htm) — Greg Cook's catalogue, the reference for CRC-16 parameter sets and their check values.
