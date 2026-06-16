---
slug: cyclic-redundancy-check
title: Cyclic redundancy check (CRC)
entry_type: algorithm
category: algorithms
description: A cyclic redundancy check is an error-detection code that appends a checksum computed by polynomial division; it detects corrupted frames in AIS, ADS-B, AX.25, and more.
keywords: CRC, cyclic redundancy check, checksum, error detection, FCS, polynomial, CRC-24
aka: [cyclic redundancy check, CRC]
autolink: true
infobox:
  - { label: Type, value: Error-detection code }
  - { label: Method, value: Polynomial division checksum }
  - { label: Used by, value: AIS, ADS-B (CRC-24), AX.25, M17 }
see_also: [forward-error-correction, ads-b, ais, ax25]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Cyclic redundancy check (Wikipedia)", url: https://en.wikipedia.org/wiki/Cyclic_redundancy_check }
---

A **cyclic redundancy check** (**CRC**) is an error-*detection* code that appends a
checksum computed by polynomial division over the data. A mismatch on receive flags a
corrupted frame.

## How it works

Unlike [forward error correction](/reference/forward-error-correction/), a CRC detects
but does not correct errors. Frames failing the CRC are discarded.
[ADS-B](/reference/ads-b/) uses CRC-24; [AIS](/reference/ais/) and
[AX.25](/reference/ax25/) use CRC-16 (FCS).

## Relevance to SDR

CRC validation ensures GopherTrunk only reports data frames that arrived intact.
