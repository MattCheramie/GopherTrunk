---
slug: file-system
title: File system
entry_type: concept
category: hw-storage
description: A file system is the structure an operating system uses to organise data on storage into named files and directories, tracking where each file's blocks live on the device.
keywords: file system, filesystem, ext4, NTFS, FAT, directory, inode, block, journaling, mount
infobox:
  - { label: Type, value: Storage organisation layer }
  - { label: Provides, value: Files and directories }
  - { label: Managed by, value: Operating system }
  - { label: Examples, value: ext4, NTFS, FAT32, APFS }
  - { label: Tracks, value: Blocks, metadata, free space }
see_also: [data-storage, operating-system, solid-state-drive, hard-disk-drive, sd-card, flash-memory]
related_lessons:
  - { title: "Building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/File_system
---

A **file system** is the structure an [operating system](/reference/operating-system/) uses to organise raw storage into named files and directories, and to track where each file's data physically lives.[^wiki]

## Overview

A storage device on its own offers only a flat array of numbered blocks. The file system imposes order on that array: it groups blocks into files, arranges files into a directory tree, records metadata such as names, sizes, timestamps, and permissions, and keeps track of which blocks are free. Common file systems include ext4 on Linux, NTFS on Windows, APFS on macOS, and the simple FAT family used on most [SD cards](/reference/sd-card/) and USB sticks. Many modern file systems are *journaling*, logging intended changes first so the volume can recover cleanly after a crash or power loss.

## Where it fits

The file system is the bridge between bare [data storage](/reference/data-storage/) hardware — an [SSD](/reference/solid-state-drive/), [HDD](/reference/hard-disk-drive/), or [flash](/reference/flash-memory/) card — and the files programs actually read and write. Before storage can be used it is *formatted* with a file system and *mounted* into the OS. GopherTrunk relies on it transparently: every decoded-call recording and IQ capture it writes becomes a file, and a journaling file system helps those logs survive an unexpected power cut at an unattended capture node.

## Sources

[^wiki]: [File system](https://en.wikipedia.org/wiki/File_system) — Wikipedia, on how operating systems organise storage into files and directories.
