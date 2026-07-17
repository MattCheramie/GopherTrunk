---
slug: file-system
title: File system
entry_type: concept
category: hw-storage
description: A file system is the structure an operating system uses to organise data on storage into named files and directories, tracking where each file's blocks live on the device.
keywords: file system, filesystem, ext4, NTFS, FAT, directory, inode, block, journaling, mount, metadata
infobox:
  - { label: Type, value: Storage organisation layer }
  - { label: Provides, value: Files and directories }
  - { label: Managed by, value: Operating system }
  - { label: Examples, value: ext4, NTFS, FAT32, APFS }
  - { label: Tracks, value: Blocks, metadata, free space }
see_also: [data-storage, operating-system, solid-state-drive, hard-disk-drive, sd-card, flash-memory, nvme]
related_lessons:
  - { title: "Building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/File_system
---

A **file system** is the structure an [operating system](/reference/operating-system/) uses to organise raw storage into named files and directories, and to track where each file's data physically lives.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="How a file system maps a name to data. A directory tree points to a metadata record (an inode) that stores a file's size, permissions, and timestamps, and that record holds a list of pointers to the scattered numbered blocks on the device that actually contain the file's bytes." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" stroke="currentColor">
    <g stroke-width="1.2" fill="none">
      <path d="M40 30 V50 H70 M40 50 V78 H70 M70 78 V96 H100"/>
    </g>
    <g stroke="none">
      <text x="20" y="26" font-size="9" font-weight="600">/</text>
      <text x="74" y="42" font-size="8.5">logs/</text>
      <text x="74" y="70" font-size="8.5">captures/</text>
      <text x="104" y="90" font-size="8.5">iq.cfile</text>
    </g>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="196" y="48" width="86" height="80" rx="3" fill="currentColor" fill-opacity="0.06"/>
    <line x1="196" y1="64" x2="282" y2="64"/>
  </g>
  <g fill="currentColor" stroke="none">
    <text x="239" y="60" font-size="8.5" text-anchor="middle" font-weight="600">inode</text>
    <text x="203" y="78" font-size="7.5">size, owner</text>
    <text x="203" y="90" font-size="7.5">permissions</text>
    <text x="203" y="102" font-size="7.5">timestamps</text>
    <text x="203" y="120" font-size="7.5">block ptrs &#8595;</text>
  </g>
  <line x1="140" y1="88" x2="196" y2="88" stroke="currentColor" stroke-width="1.2"/>
  <g stroke="currentColor" fill="currentColor">
    <rect x="330" y="34" width="30" height="18" rx="2" fill-opacity="0.18" stroke="none"/>
    <rect x="386" y="60" width="30" height="18" rx="2" fill-opacity="0.18" stroke="none"/>
    <rect x="330" y="92" width="30" height="18" rx="2" fill-opacity="0.18" stroke="none"/>
    <rect x="392" y="120" width="30" height="18" rx="2" fill-opacity="0.18" stroke="none"/>
    <g stroke-width="1" fill="none">
      <path d="M282 96 L330 43"/>
      <path d="M282 100 L386 69"/>
      <path d="M282 104 L330 101"/>
      <path d="M282 108 L392 129"/>
    </g>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="345" y="46" text-anchor="middle">#12</text>
    <text x="401" y="72" text-anchor="middle">#57</text>
    <text x="345" y="104" text-anchor="middle">#13</text>
    <text x="407" y="132" text-anchor="middle">#88</text>
    <text x="373" y="162" text-anchor="middle" font-size="8">data blocks scattered across the device</text>
  </g>
</svg>
<figcaption>A path name walks the directory tree to an inode — the record of a file's size, permissions, and timestamps — whose block pointers gather the scattered numbered blocks that hold the actual bytes.</figcaption>
</figure>

## Overview

A storage device on its own offers only a flat array of numbered blocks. The file system imposes order on that array: it groups blocks into files, arranges files into a directory tree, records metadata such as names, sizes, timestamps, and permissions, and keeps track of which blocks are free. On Unix-like systems that metadata lives in a per-file *inode* that also holds the list of data blocks, while directories are just files that map names to inodes.

Common file systems include ext4 on Linux, NTFS on Windows, APFS on macOS, and the simple FAT family used on most [SD cards](/reference/sd-card/) and USB sticks. They differ in maximum file size, metadata richness, and crash behaviour. Many modern file systems are *journaling*: they log intended changes first, so after a crash or power loss the volume can be replayed to a consistent state instead of needing a slow full scan.

## Inside it

The layers stack from the raw device up to the names a program uses:

| Layer | Holds | Example |
|-------|-------|---------|
| Block | Smallest allocation unit | 4 KB block |
| Inode / record | One file's metadata + block list | size, mode, mtime |
| Directory | Names mapped to inodes | `captures/` |
| Journal | Pending changes for crash recovery | ext4 journal |
| Free-space map | Which blocks are unused | bitmap / extent tree |

## Where it fits

The file system is the bridge between bare [data storage](/reference/data-storage/) hardware — an [SSD](/reference/solid-state-drive/), [HDD](/reference/hard-disk-drive/), or [flash](/reference/flash-memory/) card — and the files programs actually read and write. Before storage can be used it is *formatted* with a file system and *mounted* into the OS. GopherTrunk relies on it transparently: every decoded-call recording and IQ capture it writes becomes a file, and a journaling file system helps those logs survive an unexpected power cut at an unattended capture node.

## Sources

[^wiki]: [File system](https://en.wikipedia.org/wiki/File_system) — Wikipedia, on how operating systems organise storage into files and directories.
