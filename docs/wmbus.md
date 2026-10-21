---
layout: page
title: Wireless M-Bus meter decoder (gophertrunk wmbus)
description: Decode Wireless M-Bus / OMS utility meter telegrams (heat cost allocators, water, gas, heat and electricity meters) at 868.95 MHz, live or from a capture
nav_group: Reference
---

# Wireless M-Bus meter decoder (`gophertrunk wmbus`)

Wireless M-Bus (EN 13757-4) is the 868 MHz radio protocol of European
utility meters: heat cost allocators on radiators, water, gas, heat and
electricity meters, and the OMS (Open Metering System) devices built on it.
Most meters broadcast a short telegram every few seconds to minutes.
`gophertrunk wmbus` decodes them from a live SDR, an rtl_tcp server, or a
capture file.

## What it decodes

- **Modes T1 and C1**, the two meter-to-reader modes nearly all installed
  meters use, both on **868.95 MHz** at 100 kchip/s. T1 is 3-of-6 coded;
  C1 uses frame format A or B. Mode S1 (868.3 MHz) is not decoded.
- **The sender** of every telegram: manufacturer code (e.g. `TCH` Techem,
  `QDS` Qundis, `KAM` Kamstrup), the 8-digit meter ID printed on the
  device, version and device type.
- **The readings of unencrypted meters**: the EN 13757-3 data records
  (volume, energy, heat cost allocation units, temperatures, dates, …),
  including historic values such as the reading at the last set day
  (`[storage 1]`).

### Encrypted meters

Most OMS meters encrypt their readings with AES-128 (**security mode 5**,
or **mode 7** with an authentication layer), using a per-meter key held by
the metering company. For these, `wmbus` still shows who sent the telegram,
when, and the device type, and prints the security mode, for example:

```
T1-A  -11.7 dBFS  BMT 19131290  water (v13)  encrypted: AES-128-CBC with IV (mode 5), access 121
```

No decryption is attempted, and there is no key option yet.

## Listening live

```
gophertrunk wmbus                            # local RTL-SDR at 868.95 MHz, 1.6 MS/s
gophertrunk wmbus -rtltcp 127.0.0.1:1234     # through an rtl_tcp server, e.g. on a phone
gophertrunk wmbus -gain 400 -ppm 30          # fixed 40 dB gain, 30 ppm correction
```

Each telegram prints one line, with one indented line per data record.
This is an unencrypted Qundis heat cost allocator (a test telegram from
wmbusmeters' simulation set, modulated and decoded by GopherTrunk):

```
2026-10-10 14:03:12.418  C1-A   -6.9 dBFS  QDS 78563412  heat cost allocator (v35)
    heat cost allocation 127
    heat cost allocation [storage 1] 145
    date [storage 1] 2018-12-31
    heat cost allocation [storage 17] 79
    date [storage 17] 2019-01-31
    date (error) invalid date FFFF
    date time 2019-02-20 11:32
```

Storage 0 is the current value, and higher storage numbers are historic
values. Here, storage 1 is the reading at the last set day. A capture file
prints the time into the capture (`t=   2.802s`) instead of the wall clock.

- `-json` prints one JSON object per telegram instead. Its `frame` field is
  the telegram's raw bytes in hex.
- `-all` also prints frames whose CRC failed, with the CRC block that
  failed:

  ```
  C1-A   -7.9 dBFS  0x097D 98A18249  electricity (v01)  CRC FAILED (block 1 of 4; block 1 is the header, so the sender shown is unreliable)
  C1-A  -12.0 dBFS  DWZ 11130881  heat cost allocator (v80)  CRC FAILED (blocks 3,4 of 4; header OK)
  ```

  Block 1 carries the sender (manufacturer, ID, device type). When it fails,
  those fields are whatever the damaged bytes decode to; a manufacturer
  shown as a hex number such as `0x097D` is not a real code. When only later
  blocks fail, the sender is right and the damage is in the payload. With
  `-json`, a failed frame carries `crc_blocks`, `crc_failed_blocks`,
  `header_crc_ok` and `raw` (the frame as received, CRC bytes included), so
  it can be checked offline.
- `-seconds N` stops after N seconds.
- `-save-frames DIR` keeps the radio signal (IQ) around each telegram; see
  [Saving telegrams to share](#saving-telegrams-to-share).
- `-log FILE` appends every telegram line and progress line to FILE as it
  happens. It survives Ctrl-C and a closed terminal; prefer it to `| tee`,
  which Ctrl-C kills along with GopherTrunk (or use `| tee -i`).
- Ctrl-C, `kill` / `pkill gophertrunk` (SIGTERM) and a closed terminal all
  stop a live run the same way, with its summary. If the SDR does not close
  within 5 s (a USB transfer stuck on a phone) it says so and exits anyway.
  On Termux under `su`, the terminal's Ctrl-C can end `su` without reaching
  GopherTrunk; stop it from a root shell with `pkill gophertrunk`, or give
  `-seconds`.
- `-status 30s` (the default) prints a progress line that often; `0` turns
  it off. `gophertrunk help wmbus` lists every flag.

**Progress line.** Meters send every couple of minutes to every few hours, so
a live run can be quiet for a long time. The progress line shows it is alive:

```
wmbus: listening 5m0s: 3 frames (1 CRC-valid, 2 CRC-failed), 412 access codes, 0 dropped chunks
```

- *Access codes* are sync words. They also appear by chance in noise, so a
  steady trickle with no frames is normal.
- *CRC-failed* frames are telegrams that were heard but damaged: the
  signal is there but weak, clipped or interfered with. Add `-all` to see them.
- *Dropped chunks* must stay at 0. Any drop means the computer did not keep
  up with the SDR and a telegram in the gap is lost. A warning is printed the
  first time it happens. On a slow phone, lower `-rate`, or record with
  `capture` and decode the file afterwards, which has no real-time limit.

**Tuning.** An RTL-SDR's crystal can be tens of ppm off, which is tens of
kHz at 868 MHz. The decoder tolerates about ±50 kHz, so `-ppm` is rarely
needed. If nothing decodes near a meter, try `-ppm` and a fixed `-gain`.

**Gain.** `-gain` (on `wmbus` and `capture`) takes tenths of a dB: `-gain 200`
is 20 dB, `-gain 0` the tuner's minimum, and `auto` lets the tuner's AGC
choose, which a strong nearby transmitter can drive into clipping. Start
around `-gain 200` and step down while `capture` reports clipping; step up
only if telegrams are weak (the `dBFS` level on each line). With an external
preamplifier in front of the dongle, start low (`-gain 0` to `-gain 100`):
the preamplifier already supplies the gain, and too much clips the ADC.

## Decoding a capture

```
gophertrunk wmbus -in meters.cu8                     # rate/centre from meters.metadata.json
gophertrunk wmbus -in meters.flac                    # wav/flac containers carry their own rate
gophertrunk wmbus -in rtl.cu8 -rate 2400000          # rtl_sdr capture: no sidecar, so give the rate
gophertrunk wmbus -in wide.cu8 -rate 2400000 -offset 325000   # capture centred 325 kHz below 868.95 MHz
```

A headerless file (`.cu8`, `.raw`, `.bin`) carries no sample rate. When
`gophertrunk capture` wrote it, the `.metadata.json` beside it does, and
`wmbus -in` reads the rate, format and centre frequency from there (it says
so on the first lines). Without a sidecar, `-rate` (or `-sample-rate`)
defaults to 1.6 MS/s. Decoding at the wrong rate finds access codes but no
frames, and stretches the reported duration: a 30 s capture at 2.8 MS/s read
at 1.6 MS/s reports 52.5 s. Flags you give override the sidecar.

Headerless files default to rtl_sdr's unsigned 8-bit format (`-format u8`);
`cs16` and `f32` work too. A u8 file holds two bytes (I and Q) per sample, so
30 s at 2.8 MS/s is 168 MB. To record a capture with GopherTrunk:

```
gophertrunk capture -freq 868950000 -sample-rate 1600000 -seconds 60 -format u8 -out meters.cu8
gophertrunk capture -freq 868950000 -sample-rate 1024000 -seconds 60 -format flac -out meters.flac
```

1.6 MS/s decodes slightly more weak telegrams. FLAC is smaller, but it cannot
store rates above 1,048,575 S/s, hence 1.024 MS/s for it.

To look at a capture rather than decode it, `gophertrunk spectrum -in
meters.cu8` prints its averaged spectrum and the carriers in it. It reads the
format, sample rate and centre from the capture's `.metadata.json` (flags you
give win); without one, give `-format u8 -sample-rate 1600000 -freq
868950000`. A file read in the wrong format is refused rather than printed as
NaN. Each carrier line is its offset from the centre (and its frequency, when
the centre is known), its peak level in dBFS and how far it stands above the
noise floor (SNR). A wM-Bus capture is mostly noise between telegrams, so
expect few carriers in the average; meters show up only if they sent often.

GopherTrunk has no waterfall for a whole capture file;
[inspectrum](https://github.com/miek/inspectrum) shows each telegram as a
burst. inspectrum picks the sample format from the file **extension**, and
reads `.raw` / `.bin` as 32-bit float: a u8 capture with those names shows as
one solid red block. Name (or link) it `.cu8`, then set the sample rate in its
side panel.

## Saving telegrams to share

A meter sends a telegram every few seconds to few minutes, and each one lasts
only tens of milliseconds, so a whole capture is mostly empty air: 30 s at
1.6 MS/s is about 96 MB even in rtl_sdr's 8-bit format. `-save-frames` keeps just the part that matters:

```
gophertrunk wmbus -rtltcp 127.0.0.1:1234 -save-frames meters/   # live, phone + rtl_tcp
gophertrunk wmbus -save-frames meters/                         # live, local dongle
gophertrunk wmbus -in street.cu8 -save-frames meters/          # cut a big capture down
```

For every telegram it prints, it writes two files to the directory:

- `<time>_<mode>_<manufacturer>_<id>.wav`: the IQ from just before the
  telegram to just after it, typically 100–400 KB. The WAV carries its own
  sample rate, so `gophertrunk wmbus -in <file>.wav` replays it (the offset
  comes from the snippet's `.metadata.json`).
- `<time>_<mode>_<manufacturer>_<id>.metadata.json`: the sample rate, the
  centre frequency and offset, and the decoded telegram (manufacturer, ID,
  device type, CRC result, level, and the raw frame in hex).

This works live, including through rtl_tcp, and on a capture you already
have. (`gophertrunk capture -rtltcp host:port` records a whole capture through
rtl_tcp too, but most of it is empty air between telegrams.)

Only CRC-valid telegrams are saved unless `-all` is given; with `-all`,
telegrams whose CRC failed are saved too, which is the useful case when a
meter does not decode. A live run prints `wmbus: saved <file>` as each one is
written, and says at start-up when it is keeping CRC-valid telegrams only.

To report results, the printed lines (or `-json` lines) can be pasted as text
on its own. Zip the `meters/` directory when the signal itself needs looking
at.

## Verification

The decoder was checked against two independent decoders:

- **rtl-wmbus.** Its four real-air sample captures decode with every
  telegram rtl-wmbus finds with a valid CRC, byte for byte. GopherTrunk
  also decodes five more CRC-valid telegrams that rtl-wmbus misses.
- **wmbusmeters.** Unencrypted telegrams from its test set decode to the
  same meter IDs and readings its drivers report.

It has not yet been run on air by a GopherTrunk user (issue #1256). If you
try it near meters, a capture with the decoder's output is welcome on that
issue.
