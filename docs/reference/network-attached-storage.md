---
slug: network-attached-storage
title: Network-attached storage (NAS)
entry_type: hardware
category: hw-servers
description: Network-attached storage (NAS) is a dedicated file-storage device that connects to a network and lets multiple clients read and write shared files over standard protocols.
keywords: NAS, network-attached storage, file server, SMB, NFS, shared storage, RAID, SAN
aka: [NAS]
infobox:
  - { label: Type, value: Networked file storage }
  - { label: Serves, value: Files to many clients }
  - { label: Protocols, value: SMB, NFS, AFP }
  - { label: Often uses, value: RAID arrays }
  - { label: Contrast, value: SAN (block storage) }
see_also: [data-storage, raid, server, home-server, hard-disk-drive, data-center]
related_lessons:
  - { title: "Home servers", url: /learn/intro-hardware/home-servers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Network-attached_storage
---

**Network-attached storage** (**NAS**) is a dedicated file-storage device that connects to a network and lets multiple clients read and write shared files over standard protocols.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A network-attached storage box in the center holds four disk drives combined into a RAID array. Three clients — a desktop, a laptop, and a phone — connect to it over the network and read and write shared files using the SMB and NFS protocols." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <rect x="185" y="52" width="90" height="76" rx="4" fill-opacity="0.10" stroke-width="1.4"/>
    <text x="230" y="44" text-anchor="middle" font-size="9" font-weight="600" stroke="none">NAS</text>
    <g stroke-width="1.1" fill-opacity="0.18">
      <rect x="194" y="62" width="72" height="12" rx="1"/>
      <rect x="194" y="78" width="72" height="12" rx="1"/>
      <rect x="194" y="94" width="72" height="12" rx="1"/>
      <rect x="194" y="110" width="72" height="12" rx="1"/>
    </g>
    <text x="230" y="140" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.85">4 disks in a RAID array</text>
    <g stroke-width="1.3" fill="none">
      <line x1="185" y1="72" x2="96" y2="46"/>
      <line x1="185" y1="90" x2="96" y2="118"/>
      <line x1="275" y1="90" x2="364" y2="80"/>
    </g>
    <rect x="46" y="32" width="46" height="30" rx="2" fill-opacity="0.14" stroke-width="1.1"/>
    <text x="69" y="51" text-anchor="middle" font-size="7.5" stroke="none">desktop</text>
    <rect x="46" y="104" width="46" height="30" rx="2" fill-opacity="0.14" stroke-width="1.1"/>
    <text x="69" y="123" text-anchor="middle" font-size="7.5" stroke="none">laptop</text>
    <rect x="368" y="64" width="34" height="34" rx="2" fill-opacity="0.14" stroke-width="1.1"/>
    <text x="385" y="84" text-anchor="middle" font-size="7.5" stroke="none">phone</text>
    <text x="140" y="70" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.8">SMB</text>
    <text x="140" y="112" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.8">NFS</text>
    <text x="320" y="76" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.8">SMB</text>
    <text x="230" y="166" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">one shared file store many clients read and write over the network</text>
  </g>
</svg>
<figcaption>A NAS combines several disks into a redundant RAID array and serves the resulting shared files to many clients at once over standard protocols like SMB and NFS — a single storage pool the whole network can reach.</figcaption>
</figure>

## Overview

A NAS is a small, specialized [server](/reference/server/) whose job is storage: a handful of drive bays, a modest CPU, and an operating system tuned for serving files over protocols such as SMB and NFS. Drives are usually combined into a [RAID](/reference/raid/) array for capacity and redundancy, so a single disk failure does not lose data.

This contrasts with *storage-area network* (SAN) hardware, which serves raw block devices rather than files, and with direct-attached storage, which is wired to one machine and cannot be shared. Consumer NAS boxes from vendors like Synology and QNAP have made the form factor common in homes and small offices, while enterprise units scale to dozens of bays.

## How it compares

NAS is one of three ways to attach storage, distinguished by *what* they serve and *who* can reach it:

| Type | Serves | Shared? | Typical use |
|------|--------|---------|-------------|
| NAS | Files (SMB/NFS) | Many clients | Shared documents, media, backups |
| SAN | Raw blocks | Servers (via fabric) | Databases, virtualization |
| Direct-attached | Blocks | One host | A single server's own disks |

For most homes and small offices the file-level, network-shared NAS is the natural fit; SANs earn their complexity only when applications need raw block devices at scale.

## Where it fits

A NAS is the natural shared-storage tier behind a [home server](/reference/home-server/) or in a [data center](/reference/data-center/) rack, built on ordinary [hard disk drives](/reference/hard-disk-drive/). It is a good home for GopherTrunk's bulk output — days of decoded calls and IQ recordings can sit on a redundant NAS volume rather than the capture node's local disk, keeping the node lean and its storage safe from a single drive failure.

## Sources

[^wiki]: [Network-attached storage](https://en.wikipedia.org/wiki/Network-attached_storage) — Wikipedia, on NAS devices, protocols, and RAID.
