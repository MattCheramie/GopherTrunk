---
layout: page
title: Power sweep logger (gophertrunk power)
description: Sweep a frequency range with a local or rtl_tcp SDR and log the averaged power spectrum as rtl_power-compatible CSV
nav_group: Reference
---

# Power sweep logger (`gophertrunk power`)

`gophertrunk power` steps an SDR across a frequency range, averages an FFT
power spectrum at each step, and writes the result as CSV in **rtl_power's
layout**. Tools built for rtl_power, such as `heatmap.py` and `flatten.py`,
read it unchanged.

Unlike rtl_power, it can sweep a dongle on **another machine** through an
`rtl_tcp` server, as well as a local USB SDR. That includes an rtl_tcp driver
app on an Android phone ([Termux guide](termux.html)).

## Quick start

```sh
# FM broadcast band in 10 kHz bins, one sweep every 10 s, until Ctrl-C
gophertrunk power -f 88M:108M:10k -out fm.csv

# The same from a remote dongle behind rtl_tcp
gophertrunk power -rtltcp 192.168.1.50:1234 -f 88M:108M:10k -out fm.csv

# One sweep of the 2 m band to the terminal, integrating for 1 s
gophertrunk power -f 144M:148M:5k -i 1 -1

# Log 446 MHz PMR for an hour, a sweep every 30 s
gophertrunk power -f 446.0M:446.2M:1k -i 30s -e 1h -out pmr.csv
```

## Output

One CSV line per tuning step (a "hop"), with the same fields as rtl_power:

```
date, time, Hz low, Hz high, Hz step, samples, dB, dB, ...
2026-10-03, 21:30:05, 88000000, 89800000, 9375.00, 4000000, -61.20, -60.95, ...
```

- **date, time:** local time when the sweep started. Every hop of one sweep
  carries the same stamp.
- **Hz low:** the frequency of the line's first value. Value *k* is at
  `Hz low + k × Hz step`.
- **Hz high:** `Hz low + (number of values) × Hz step`.
- **Hz step:** the bin width. It is the requested bin size rounded down to
  `sample rate / FFT size`, where the FFT size is a power of two.
- **samples:** IQ samples averaged for this hop. On a CPU too slow to FFT
  every sample (see below) it is less than `interval ÷ hops × rate`.
- **dB:** averaged power per bin in **uncalibrated dBFS**. A carrier's
  power reads 1.8–3.2 dB low here (Hann window), so compare levels with
  each other, not against an absolute reference.

Hops abut exactly, so concatenating a sweep's lines gives one continuous
spectrum from the lower edge to the upper edge. A signal that sits right on
a hop boundary is therefore split: half of it ends one line and the rest
opens the next.

### Why "10k" gives 192 values of 9375 Hz

As with rtl_power, the bin size in `-f` is a **maximum**. For
`-f 88M:108M:10k` at the default 2.4 MS/s:

1. **Bin width.** The FFT is the smallest power of two whose bins are no
   wider than 10 kHz. 2,400,000 ÷ 128 = 18,750 Hz is too wide, and
   2,400,000 ÷ 256 = **9375 Hz** fits, so the FFT has 256 points.
2. **Bins per line.** `-crop 0.25` drops a quarter of the FFT (an eighth at
   each edge, where the tuner's filter rolls off), leaving
   256 × 0.75 = **192 bins**, or 192 × 9375 Hz = 1.8 MHz per hop.
3. **Hops.** 20 MHz is 2134 bins of 9375 Hz, which takes **12 hops**, so a
   sweep is 12 lines. The first 11 have 192 values each and the last has the
   remaining 22, ending at 108 MHz.

`power` prints this breakdown on stderr whenever the bins come out narrower
than you asked for. For exactly 10 kHz bins, pick a rate that divides into
them, for example `-rate 2560000` (2,560,000 ÷ 256 = 10,000 Hz), provided
your dongle runs that rate without dropping samples.

## Flags

| Flag | Default | Meaning |
|---|---|---|
| `-f lower:upper:bin` | (required) | Range and bin size, with `k` / `M` / `G` suffixes. |
| `-i` | `10s` | Integration interval. One sweep starts every interval, and each hop integrates `interval ÷ hops` of samples. A bare number is seconds. |
| `-e` | (none) | Stop after this long. It is checked between sweeps, so a sweep is never cut short: `-i 30 -e 5m` gives exactly 10 complete sweeps. Without it, sweeps run until Ctrl-C. |
| `-1` | off | Run one sweep, then exit. |
| `-out` | stdout | CSV path. A trailing positional file name also works, as with rtl_power. |
| `-rtltcp host:port` | (none) | Sweep the dongle behind an rtl_tcp server instead of a local SDR. |
| `-serial` | (none) | Pick a local SDR when several are attached. |
| `-gain` | `auto` | `auto`, or tenths of a dB as everywhere in GopherTrunk: `280` is 28 dB, and `28.0` also reads as 28 dB. |
| `-ppm` | `0` | Frequency correction. |
| `-rate` | `2400000` | Sample rate. Each hop covers `rate × (1 − crop)` Hz. |
| `-crop` | `0.25` | Fraction of each FFT dropped at the edges, where the tuner's filter rolls off. |
| `-settle` | 50 ms local, 250 ms rtl_tcp | Samples discarded after each retune. |
| `-bias-tee` | off | Power an LNA through the antenna port. |

Each sweep's lines are flushed as soon as the sweep finishes, so you can
`tail -f` a long log, and Ctrl-C loses at most the sweep in progress.

## Things to know

- **Only one program can use the dongle.** Stop the daemon, or any other
  rtl_tcp client, before sweeping the same SDR.
- **rtl_tcp needs a longer settle.** The server keeps streaming while the
  retune command crosses the network, so samples from the previous frequency
  are still in flight. If a strong carrier shows up one hop away from where
  it belongs, raise `-settle`.
- **The centre of each hop carries the tuner's DC spike.** rtl_power has the
  same artefact. A larger `-crop` narrows each hop but does not move the spike.
- **A slow CPU averages fewer samples; it doesn't slow the sweeps.** Each hop
  gets its share of the interval. If the FFT can't keep up with the sample
  rate, as on older phones, the hop stops at that time limit with whatever
  it averaged, so sweeps still start every `-i`. `power` says so once on
  stderr, and the `samples` column shows how many each hop used. A lower
  `-rate` (for example `1024000`) needs less CPU but takes more hops.
- **Differences from rtl_power:** the default crop is 0.25 (rtl_power's is
  0), gain is in tenths of a dB, and there is no `-w` window choice (Hann is
  used) or `-F` downsampling filter.
