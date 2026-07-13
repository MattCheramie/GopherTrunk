---
slug: key-loader-kfd
title: Key Loader (KFD / KVL)
entry_type: term
category: cryptography
description: "A key loader (KFD or KVL) is the handheld fill device that securely transfers encryption keys into a radio over a wired port, seeding the keys that OTAR later updates over the air."
keywords: key loader, key fill device, KFD, KVL, KVL 3000, KVL 4000, key variable loader, DS-102, key fill, keyfill, keyloader, KFDtool, P25 encryption, TEK, KEK, keyset, crypto ignition key
aka: [KFD, KVL, key fill device, key variable loader, key loader]
autolink: true
infobox:
  - { label: Type, value: Hardware fill device }
  - { label: Loads, value: Encryption keys into radios }
  - { label: Interface, value: Wired fill port (DS-102 / KFD) }
see_also: [cryptographic-key, otar, key-id-algid, project-25, advanced-encryption-standard, data-encryption-standard]
cite_urls:
  - https://en.wikipedia.org/wiki/Fill_device
  - https://tia.org/
---

**A key loader** — commonly a **Key Fill Device (KFD)** or **Key Variable Loader
(KVL)** — is the handheld unit that securely transfers encryption
[keys](/reference/cryptographic-key/) into a radio through a direct wired
connection.[^wiki] It is the trusted starting point of a secure radio system: keys
are generated or held in the loader, then "filled" into each radio's fill port, so
that the sensitive key material never travels over the air in the clear. On
[P25](/reference/project-25/) fleets the loader also seeds the Key Encryption Keys
that later let [OTAR](/reference/otar/) update traffic keys remotely.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 104" role="img" aria-label="A handheld key loader connects by a short wired cable to a radio's fill port and transfers key material into numbered key slots inside the radio." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="26" y="30" width="70" height="52" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="61" y="50">KFD / KVL</text><text x="61" y="62">key loader</text>
    <line x1="96" y1="56" x2="176" y2="56" stroke="currentColor" stroke-width="1.4" marker-end="url(#kfdar)"/><text x="136" y="48">wired fill</text>
    <rect x="178" y="26" width="86" height="60" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="221" y="40">radio</text>
    <rect x="190" y="48" width="62" height="12" fill="currentColor" fill-opacity="0.12" stroke="currentColor"/><text x="221" y="57" font-size="7">key slot 1</text>
    <rect x="190" y="64" width="62" height="12" fill="none" stroke="currentColor"/><text x="221" y="73" font-size="7">key slot 2</text>
    <text x="360" y="44">later:</text>
    <path d="M300 60 q40 -22 80 0" fill="none" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#kfdar)"/><text x="360" y="82">OTAR over air</text>
  </g>
  <defs><marker id="kfdar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The key loader fills key material into a radio over a short wired link; only afterward can OTAR refresh keys wirelessly.</figcaption>
</figure>

## How it works

A key loader stores one or more keys — often organized into keysets and numbered
key slots — and pushes them into a radio over a short serial fill connection.
Classic devices such as Motorola's KVL family use the long-standing **DS-102** fill
interface (a simple clocked serial handshake inherited from earlier crypto gear),
while modern P25 loaders use the standardized **KFD** interface defined by the TIA.
The transfer associates each key with a Key ID and algorithm ID (see
[Key ID & ALGID](/reference/key-id-algid/)) so the radio knows what the loaded key
is called and which cipher it feeds.

Because the fill link is short and physical, it is treated as a trusted channel:
the operator has the radio and the loader in hand, so key bits crossing the cable
are not exposed to the RF environment. The loader itself is the crown jewel of the
system's security and is handled accordingly — PIN- or password-protected, able to
zeroize (erase) its own contents on tamper or command, and stored under access
control when not in use. This is precisely why [OTAR](/reference/otar/) exists as a
complement: a loader can only rekey a radio you physically hold, so the wired fill
is used to plant the long-lived Key Encryption Key once, after which routine
traffic-key rotation happens over the air.

## Relevance to SDR

Key loaders sit entirely on the *authorized* side of a secure system and never
touch the RF path a scanner sees, so they define the boundary GopherTrunk cannot
cross rather than anything it decodes. The keys that make encrypted
[P25](/reference/project-25/) or DMR voice unrecoverable to a monitoring receiver
originate in these devices and are moved by wire specifically so they are never
observable over the air; a receiver only ever sees the resulting ciphertext and the
clear [Key ID and ALGID](/reference/key-id-algid/) labels. Understanding the loader
is nonetheless useful context for a listener: it explains *why* clear-to-encrypted
transitions on a system are deliberate provisioning acts, and why the security of
the entire fleet reduces to the physical control of a few handheld boxes rather than
to the strength of the cipher on the air.

## In practice

Open tooling such as the community **KFDtool** and KFDShield implements the P25 KFD
interface for interoperability testing and for agencies managing their own keys, and
it underscores that the protocol on the fill port is documented while the key values
remain secret. A recurring operational issue is keyset coordination: a radio filled
with the wrong keyset, or one that missed a scheduled changeover, will show correct
signal but fail to decrypt — the same class of fault seen with mismatched
[Key IDs](/reference/key-id-algid/) — which is why disciplined fill records and
loader inventory are as important to a secure system as the cryptography itself.

## Sources

[^wiki]: [Fill device](https://en.wikipedia.org/wiki/Fill_device) — Wikipedia, for the role of keying/fill devices, the DS-102 fill interface, and tamper/zeroize handling of key loaders.
