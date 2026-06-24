---
slug: network-attached-storage
title: Network-attached storage (NAS)
entry_type: hardware
category: hw-servers
description: Network-attached storage (NAS) is a dedicated file-storage device that connects to a network and lets multiple clients read and write shared files over standard protocols.
keywords: NAS, network-attached storage, file server, SMB, NFS, shared storage, RAID
aka: [NAS]
infobox:
  - { label: Type, value: Networked file storage }
  - { label: Serves, value: Files to many clients }
  - { label: Protocols, value: SMB, NFS, AFP }
  - { label: Often uses, value: RAID arrays }
see_also: [data-storage, raid, server, home-server, hard-disk-drive, data-center]
related_lessons:
  - { title: "Home servers", url: /learn/intro-hardware/home-servers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Network-attached_storage
---

**Network-attached storage** (**NAS**) is a dedicated file-storage device that connects to a network and lets multiple clients read and write shared files over standard protocols.[^wiki]

## Overview

A NAS is a small, specialized [server](/reference/server/) whose job is storage: a handful of drive bays, a modest CPU, and an operating system tuned for serving files over protocols such as SMB and NFS. Drives are usually combined into a [RAID](/reference/raid/) array for capacity and redundancy, so a single disk failure does not lose data. This contrasts with *storage-area network* (SAN) hardware, which serves raw block devices rather than files. Consumer NAS boxes from vendors like Synology and QNAP have made the form factor common in homes and small offices.

## Where it fits

A NAS is the natural shared-storage tier behind a [home server](/reference/home-server/) or in a [data center](/reference/data-center/) rack, built on ordinary [hard disk drives](/reference/hard-disk-drive/). It is a good home for GopherTrunk's bulk output — days of decoded calls and recordings can sit on a redundant NAS volume rather than the capture node's local disk, keeping the node lean.

## Sources

[^wiki]: [Network-attached storage](https://en.wikipedia.org/wiki/Network-attached_storage) — Wikipedia, on NAS devices and protocols.
