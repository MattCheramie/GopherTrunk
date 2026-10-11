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

- `-json` prints one JSON object per telegram instead.
- `-all` also prints frames whose CRC failed.
- `-seconds N` stops after N seconds.

**Tuning.** An RTL-SDR's crystal can be tens of ppm off, which is tens of
kHz at 868 MHz. The decoder tolerates about ±50 kHz, so `-ppm` is rarely
needed. If nothing decodes near a meter, try `-ppm` and a fixed `-gain`.

## Decoding a capture

```
gophertrunk wmbus -in meters.cu8                     # rtl_sdr capture, 1.6 MS/s (the default)
gophertrunk wmbus -in meters.cu8 -rate 2400000       # another rate
gophertrunk wmbus -in meters.flac                    # wav/flac containers carry their own rate
gophertrunk wmbus -in wide.cu8 -rate 2400000 -offset 325000   # capture centred 325 kHz below 868.95 MHz
```

Headerless files default to rtl_sdr's unsigned 8-bit format (`-format u8`);
`cs16` and `f32` work too. To record a capture with GopherTrunk:

```
gophertrunk capture -freq 868950000 -sample-rate 1600000 -seconds 60 -format u8 -out meters.cu8
gophertrunk capture -freq 868950000 -sample-rate 1024000 -seconds 60 -format flac -out meters.flac
```

1.6 MS/s decodes slightly more weak telegrams. FLAC is smaller, but it cannot
store rates above 1,048,575 S/s, hence 1.024 MS/s for it.

## Verification

The decoder was checked against two independent decoders:

- **rtl-wmbus.** Its four real-air sample captures decode with every
  telegram rtl-wmbus finds with a valid CRC, byte for byte. GopherTrunk
  also decodes four more CRC-valid telegrams that rtl-wmbus misses.
- **wmbusmeters.** Unencrypted telegrams from its test set decode to the
  same meter IDs and readings its drivers report.

It has not yet been run on air by a GopherTrunk user (issue #1256). If you
try it near meters, a capture with the decoder's output is welcome on that
issue.
