---
slug: flashing-an-os-image
title: Flashing an OS image
description: Write an operating system image onto an SD card with Raspberry Pi Imager or dd, preconfigure Wi-Fi, SSH, and a hostname before first power-on, and boot a board you'll never plug a monitor into.
keywords: flash SD card, Raspberry Pi Imager, dd command, write OS image, headless setup, enable SSH before boot, preconfigure wifi, burn image
level: beginner
status: full
prereq:
  - operating-systems-for-sbcs
faq:
  - q: How do I install an OS on a Raspberry Pi without a monitor?
    a: Use Raspberry Pi Imager on your PC. Pick the OS (choose the Lite image for headless use), pick the SD card, and before writing open the customisation settings to set a hostname, username and password, Wi-Fi credentials, and — critically — enable SSH. Write the card, put it in the board, apply power, and after a minute or two you can log in over the network. No monitor ever needs to be attached.
  - q: What does "flashing" an image actually do?
    a: It copies a byte-for-byte snapshot of a complete bootable disk — partition table, boot files, and root filesystem — onto the card, destroying whatever was there before. It is not copying a file onto the card in the normal sense; a card that just contains the .img file as a file will not boot. Dedicated tools like Raspberry Pi Imager also verify the write, which catches weak cards early.
  - q: Why won't my freshly flashed board boot?
    a: "The usual suspects, in order: the image was copied as a file instead of written as an image; the write failed or was never verified (bad card or reader); the image doesn't match the board (wrong model or architecture); or the power supply is inadequate, so the board brownouts during boot. Re-flash with a verifying tool and a known-good supply before suspecting anything deeper."
---

# Flashing an OS image

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Flashing** writes a complete bootable-disk **image** byte-for-byte onto an SD card
— not copying a file, but replacing the card's entire contents. **Raspberry Pi
Imager** is the friendly tool, and its customisation screen is the headless
superpower: set the **hostname**, a user and password, Wi-Fi, and **enable SSH**
*before first boot*, so the board comes up on the network ready to log into. The
venerable `dd` command does the same job from any Linux shell. Always **verify** the
write — a silently corrupt flash is the classic mystery no-boot.
</div>

Unit 3 begins the hands-on: no monitor, no keyboard, ever. This lesson takes a blank
SD card to a board that boots straight onto your network, configured and reachable —
the foundation every later lesson stands on.

## What does "flashing" really mean?

An image file (`.img`, often compressed) is a snapshot of an entire disk —
partition table, boot partition, root filesystem, as
[Operating systems for small boards](/learn/embedded/operating-systems-for-sbcs/)
described. Flashing writes those bytes directly over the card's raw device, so the
card *becomes* that disk. Everything previously on the card is destroyed, and the
result is not "a card with an image file on it" but "a bootable system." This
distinction resolves half of all first-timer no-boots.

## How do you flash with Raspberry Pi Imager?

**Raspberry Pi Imager** (free, on Windows/macOS/Linux) wraps the whole job:

1. **Choose device and OS.** Pick your board model, then the OS — for an appliance,
   **Raspberry Pi OS Lite (64-bit)**.
2. **Choose storage.** Select the SD card — and double-check you've selected the
   card, not a backup drive; flashing erases its target absolutely.
3. **Edit settings before writing.** This is the headless magic (next section).
4. **Write and verify.** Imager writes the image, then reads it back to confirm.
   Verification failing means a bad card or reader — deal with it now, not at 2am.

Other good tools exist (balenaEtcher is a popular verifying flasher; your board
vendor may ship its own). The verify step is the feature to insist on.

## What should you preconfigure for headless boot?

In Imager's customisation screen (the gear icon, or offered before writing), set:

- **Hostname** — the board's network name, e.g. `scanner`. You'll reach it as
  `scanner.local` next lesson instead of hunting IP addresses.
- **Username and password** — your own, replacing any default. Default credentials
  on a networked box are how appliances end up in botnets
  ([Users, permissions &amp; updates](/learn/embedded/users-and-updates/) finishes
  this hygiene).
- **Enable SSH** — the whole point: remote login from first boot. Password
  authentication is fine to start; you'll upgrade to keys in
  [Remote administration](/learn/embedded/remote-administration/).
- **Wi-Fi credentials and country** — only if you must use Wi-Fi; wired Ethernet
  needs no configuration at all.

Under the hood these settings become plain files on the FAT **boot partition** — which
is why you can also add or fix them after flashing from any PC, no special tools.

## How does the same job look with dd?

On Linux and macOS the traditional way is `dd`, which copies raw bytes to a device.
It's worth knowing because it works everywhere and demystifies what Imager does:

```bash
# Identify the card FIRST — dd overwrites whatever you point it at
$ lsblk                       # find the card, e.g. /dev/sdX (not your system disk!)

$ xz -dc raspios-lite-arm64.img.xz | sudo dd of=/dev/sdX bs=4M status=progress conv=fsync
```

`of=` is the target device, `bs=4M` copies in efficient blocks, and `conv=fsync`
forces the final bytes onto the card before `dd` exits. The classic hazard is aiming
`of=` at the wrong disk — `dd` will cheerfully destroy it. This is Unit 3's one
genuinely dangerous command; the [Linux CLI module](/learn/linux-cli/sudo-and-root/)
explains the root privileges it runs under.

> Rule of thumb: unplug every external drive you care about before flashing, and
> read the target device name three times. Paranoia here is professionalism.

## What happens on first boot?

Insert the card, connect Ethernet, apply power. No screen will show you anything —
instead, know the choreography: the board resizes the root filesystem to fill the
card, generates its own SSH identity, applies your preconfigured settings, joins the
network, and starts the SSH server. Allow two or three minutes for the first boot.
The status LED settling into steady activity is your only local signal; the real
confirmation is finding it on the network — which is exactly the
[next lesson](/learn/embedded/first-boot-and-ssh/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — preconfiguring SSH, a user, and the hostname is what makes monitor-free setup possible." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes a truly headless first boot possible?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">SBCs automatically broadcast a setup website on first boot</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You must attach a monitor once to enable networking, then remove it</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The flasher writes SSH, user, hostname, and Wi-Fi settings into the image before first power-on</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Flashing** writes a byte-for-byte disk **image** over the card's raw device —
  the card becomes the system; copying the file onto the card does nothing.
- **Raspberry Pi Imager** flashes, **verifies**, and — crucially — preconfigures
  **hostname, user, Wi-Fi, and SSH** before first boot.
- The settings land as plain files on the FAT **boot partition**, editable from any
  PC after flashing too.
- `dd` does the same from any shell — powerful, and pointed at the wrong device,
  destructive. Verify targets obsessively.
- First boot takes a few minutes of invisible setup; the confirmation is the board
  appearing **on the network**.

Next up: [First boot &amp; SSH](/learn/embedded/first-boot-and-ssh/).
