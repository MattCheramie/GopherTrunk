---
slug: operating-systems-for-sbcs
title: Operating systems for small boards
description: Most single-board computers run real Linux. Learn what a Raspberry Pi OS image contains, how SBC Linux differs from server or desktop Linux, and what "Lite" vs desktop images mean.
keywords: Raspberry Pi OS, SBC operating system, Linux on Raspberry Pi, Debian, OS image, lite vs desktop image, boot partition, device tree, embedded Linux
level: beginner
status: full
prereq:
  - arm-and-socs
---

# Operating systems for small boards

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Nearly every SBC runs **real Linux** — the same kernel and tools as a server, usually
packaged as a **Debian** derivative like **Raspberry Pi OS**. You install it by
writing a complete **image** (a ready-made snapshot of a bootable disk) onto an SD
card. SBC Linux differs from laptop Linux mainly in the **boot process** (firmware
and config files on a visible boot partition instead of a PC BIOS) and in
board-specific drivers. For appliances, pick the **Lite** image — no desktop, fewer
resources used, nothing you'll miss on a headless machine.
</div>

Unit 1 closes with the software half of the foundation. The good news is enormous:
everything you know (or will learn) about Linux transfers to a small board almost
unchanged. This lesson covers the differences that remain — and the choices you'll
face when you download your first image in Unit 3.

## Is it really the same Linux?

Yes — and this is worth pausing on. Raspberry Pi OS is a build of **Debian**, one of
the most established Linux distributions. The kernel is the mainline Linux kernel with
board patches. `bash`, `ssh`, `apt`, `systemd`, users, permissions, cron — every tool
from the [Linux &amp; CLI module](/learn/linux-cli/what-is-linux/) works identically.
When you SSH into a Pi you cannot tell, from the shell prompt, that you're not on a
big server.

The differences are at the edges:

- **Architecture.** Packages are built for ARM ([last lesson](/learn/embedded/arm-and-socs/)),
  so a rare package that only exists for x86 won't be installable — but the standard
  repositories carry many thousands of ARM packages, including everything this module uses.
- **Booting.** No PC-style BIOS; the SoC's firmware loads the OS a different way (below).
- **Drivers.** GPIO, the camera connector, and the SoC's video blocks have
  board-specific drivers baked into the image.

## What is an OS image?

On a PC you typically run an *installer*. On an SBC you skip the installer entirely:
you download an **image** — a byte-for-byte snapshot of a complete, already-installed
bootable disk — and write it directly onto an SD card. First boot then expands the
filesystem to fill your card and generates fresh identity (hostname, SSH keys).

An image contains, in one file:

| Piece | What it is |
|-------|-----------|
| **Boot partition** | A small FAT partition with firmware, the kernel, and plain-text config files |
| **Root filesystem** | The whole Linux system: `/etc`, `/home`, `/usr`, packages, defaults |
| **Partition table** | The map that tells the SoC where those live on the card |

The boot partition is an SBC peculiarity worth knowing early: because it's plain FAT,
**you can open it from any PC** after flashing and edit text files to preconfigure the
system — enable SSH, set Wi-Fi credentials, set the hostname — before the board has
ever powered on. That trick is what makes fully headless setup possible, and
[Flashing an OS image](/learn/embedded/flashing-an-os-image/) uses it heavily.

## Desktop or Lite — which image do you want?

Most SBC distributions ship two main flavours:

- **Desktop** images include a graphical environment — windows, browser, menus — for
  use with a monitor and keyboard. Fine for a first play, or if the board *is* a desktop.
- **Lite** (or "server") images boot to a text console and include no graphics at all.

For everything this module builds, choose **Lite**. A headless appliance never renders
a desktop, and leaving one installed just costs RAM, storage, background CPU, and
update time. GopherTrunk in particular needs no local display: its interface is a
**web console** you open from another machine's browser — the appliance itself can be
a bare shell.

> Rule of thumb: if you'll reach the board by SSH and browser, install Lite. You can
> always add packages later; you'll rarely wish you'd installed more up front.

## How does an SBC boot differently from a PC?

You don't need the details yet, but the shape helps when troubleshooting: a PC has a
BIOS/UEFI in motherboard flash that finds and boots an OS from any disk. An SBC's SoC
instead runs a small **boot firmware** that reads the boot partition of the SD card
(or, on newer boards, USB or network) and follows its plain-text config files —
on a Pi, `config.txt` (hardware settings) and `cmdline.txt` (kernel options). Two
practical consequences:

- **Board settings live in text files** you can read and version-control — no
  pressing DEL at the right millisecond.
- **A board that won't boot is very often a card/image problem**, because the
  card *is* the whole system. Re-flashing is the standard first fix, which is also why
  [Backups &amp; images](/learn/embedded/backups-and-images/) makes re-flashing painless.

## What about non-Linux options?

For completeness: some SBC-shaped systems run other software — retro-gaming or media
images (still Linux underneath), Android builds, and real-time OSes on
microcontroller-class boards. And special-purpose appliance distributions exist that
boot straight into one application. All out of scope here: plain **Raspberry Pi OS
Lite** (or your board's Debian/Ubuntu equivalent) is the foundation this module — and
most of the SBC world — builds on.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an image is a complete bootable-disk snapshot written straight onto the card." markdown="0">
  <p class="knowledge-check__q">Quick check: how does an OS normally get onto a single-board computer?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">You run an installer wizard on the board itself, like on a PC</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">You write a complete pre-built image of a bootable disk onto the SD card</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The OS is permanently built into the board at the factory</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- SBCs run **real Linux** — Raspberry Pi OS is Debian, and every standard tool and
  skill transfers unchanged.
- The OS arrives as an **image**: a complete bootable-disk snapshot written straight
  to the SD card; first boot expands and personalises it.
- The FAT **boot partition** is editable from any PC — the key to preconfiguring SSH
  and Wi-Fi for headless setup.
- Choose the **Lite** image for appliances: no desktop, fewer resources, and
  GopherTrunk's interface is a web console anyway.
- SBC boot settings are **plain text files** on the card, and re-flashing the card is
  the standard fix for a board that won't boot.

Next up: [Picking a board](/learn/embedded/picking-a-board/).
