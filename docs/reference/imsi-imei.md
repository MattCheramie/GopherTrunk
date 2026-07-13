---
slug: imsi-imei
title: IMSI & IMEI
entry_type: term
category: cellular
description: The IMSI is the subscriber identity stored on the SIM, while the IMEI is the equipment identity burned into the handset; together they separate who is calling from which device.
keywords: IMSI, IMEI, International Mobile Subscriber Identity, International Mobile Equipment Identity, SIM, MCC, MNC, MSIN, TMSI, GUTI, eSIM, IMEISV, IMSI catcher
aka: [IMSI, IMEI, International Mobile Subscriber Identity, International Mobile Equipment Identity]
autolink: true
infobox:
  - { label: IMSI, value: "Subscriber identity (on the SIM), up to 15 digits" }
  - { label: IMEI, value: "Equipment identity (in the handset), 15 digits" }
  - { label: Structure, value: "IMSI = MCC + MNC + MSIN" }
see_also: [gsm, esim, cellular-modem, registration, lte, radio-id]
cite_urls:
  - https://en.wikipedia.org/wiki/International_mobile_subscriber_identity
  - https://en.wikipedia.org/wiki/International_Mobile_Equipment_Identity
---

The **IMSI (International Mobile Subscriber Identity)** and **IMEI (International Mobile
Equipment Identity)** are the two identifiers that let a cellular network tell *who* is
calling apart from *which device* they are calling on.[^imsi][^imei] The IMSI belongs to
the **subscription** and lives on the SIM; the IMEI belongs to the **phone** and is
burned into the handset at manufacture.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A SIM card carries the IMSI, structured as mobile country code, mobile network code, and subscriber number, while the handset itself carries the IMEI, structured as a type allocation code, serial number, and check digit." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="200" height="90" rx="6" fill="currentColor" fill-opacity="0.1" stroke="currentColor"/>
  <rect x="34" y="52" width="26" height="20" rx="3" fill="currentColor" fill-opacity="0.4" stroke="currentColor"/>
  <text x="120" y="46" text-anchor="middle" font-size="9" fill="currentColor">SIM → IMSI</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="75" y="80" width="34" height="20" fill="none" stroke="currentColor"/><text x="92" y="94">MCC</text>
    <rect x="109" y="80" width="34" height="20" fill="none" stroke="currentColor"/><text x="126" y="94">MNC</text>
    <rect x="143" y="80" width="60" height="20" fill="none" stroke="currentColor"/><text x="173" y="94">MSIN</text>
  </g>
  <rect x="255" y="24" width="90" height="102" rx="10" fill="currentColor" fill-opacity="0.1" stroke="currentColor"/>
  <text x="300" y="46" text-anchor="middle" font-size="9" fill="currentColor">handset → IMEI</text>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <rect x="266" y="60" width="68" height="18" fill="none" stroke="currentColor"/><text x="300" y="72">TAC (type)</text>
    <rect x="266" y="82" width="46" height="18" fill="none" stroke="currentColor"/><text x="289" y="94">serial</text>
    <rect x="312" y="82" width="22" height="18" fill="none" stroke="currentColor"/><text x="323" y="94">chk</text>
  </g>
  <text x="380" y="80" font-size="8" fill="currentColor">who</text>
  <text x="380" y="94" font-size="8" fill="currentColor">vs. which</text>
</svg>
<figcaption>The IMSI (on the SIM) identifies the subscription as MCC + MNC + MSIN; the IMEI (in the handset) identifies the device as a type-allocation code, serial, and check digit.</figcaption>
</figure>

## How it works

The **IMSI** is up to 15 digits, split into three fields: a 3-digit **MCC** (mobile
country code), a 2–3-digit **MNC** (mobile network code) identifying the operator, and
the remaining **MSIN** (mobile subscriber identification number) identifying the
account within that network. When a phone attaches to a network — part of the
[registration](/reference/registration/) process — the network uses the MCC/MNC to know
which operator to authenticate against. To avoid broadcasting the permanent IMSI over
the air, the network quickly assigns a temporary alias (a **TMSI** in
[GSM](/reference/gsm/), a **GUTI** in [LTE](/reference/lte/)) and uses that for most
later signalling.

The **IMEI** is a 15-digit number identifying the physical equipment: an 8-digit
**Type Allocation Code (TAC)** that encodes the make and model, a 6-digit serial
number, and a Luhn check digit. A variant, **IMEISV**, appends a software-version
field. Because the IMEI is device-bound, networks use it to bar stolen handsets via
shared equipment-identity registers, independent of whatever SIM is inserted.

## In practice

Keeping the two identities separate is what makes SIM swapping possible: move the SIM
(and its IMSI) to a new phone and the subscription follows, while the phone's IMEI stays
with the hardware. An [eSIM](/reference/esim/) achieves the same split electronically,
provisioning the IMSI into a soldered secure element instead of a removable card. The
privacy risk in the permanent IMSI is precisely why temporary identifiers exist — and
why a rogue "IMSI catcher" base station that forces phones to reveal their IMSI is a
known surveillance concern.

## Relevance to SDR

IMSI and IMEI are cellular identifiers, and on modern networks the permanent IMSI is
deliberately hidden behind temporary aliases and, in 5G, encrypted, so it is not
casually recoverable from the air. They are relevant to the SDR world as the canonical
example of the **subscriber-vs-equipment** identity split, a pattern echoed in the
land-mobile systems GopherTrunk does decode: a [radio ID](/reference/radio-id/) on a
[P25](/reference/p25-phase-1/) or [DMR](/reference/dmr/) system likewise names a unit on
the network. **GopherTrunk does not recover IMSI or IMEI** — cellular attach signalling
is out of scope for its land-mobile air interfaces, and these identifiers are protected
on contemporary networks. They are documented here for context on how cellular identity
works.

## Sources

[^imsi]: [International mobile subscriber identity](https://en.wikipedia.org/wiki/International_mobile_subscriber_identity) — Wikipedia, for the IMSI's MCC/MNC/MSIN structure and its storage on the SIM.
[^imei]: [International Mobile Equipment Identity](https://en.wikipedia.org/wiki/International_Mobile_Equipment_Identity) — Wikipedia, for the IMEI's TAC/serial/check-digit structure and its role in barring stolen devices.
