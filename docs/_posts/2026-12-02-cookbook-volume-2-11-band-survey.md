---
title: "The Operator's Cookbook Vol. 2, Part 11: The Band Survey — gophertrunk power"
description: "A complete recipe for gophertrunk power, the rtl_power-compatible sweep logger: the command line, the CSV and its honest samples column, sweeping through rtl_tcp or a Pluto's iiod, and the DC line at every hop centre, removed by what it is rather than by overwriting bins. Ends by turning a survey into frequencies to hunt."
category: tutorials
keywords: rtl_power alternative go, sdr band survey csv, rtl_power csv heatmap, spectrum sweep rtl_tcp, plutosdr power sweep iio, dc spike rtl_power hop centre, sdr occupancy logging, find trunking control channel sweep, gophertrunk power
tags: [operator-cookbook, cookbook-volume-2, power-sweep, spectrum, rtl-power, survey]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Operator's Cookbook, Volume 2"
series_part: 11
---

*Part 11 of **The Operator's Cookbook, Volume 2**, a 14-part run of
complete, copy-paste GopherTrunk builds — one working rig per part — built
on what landed after Volume 1.
[Part 10]({{ '/blog/tutorials/cookbook-volume-2-10-phone-rig/' | relative_url }})
ran a sweep from a phone and moved on. This part stays with it:
`gophertrunk power`, the logger that answers the question every rig in
both volumes starts from — what is actually on the air here, and when?
No config file, no decoder, one CSV.*

> **TL;DR:** `gophertrunk power -f lower:upper:bin` steps an SDR across a
> range and writes one CSV line per step in rtl_power's layout:
> `date, time, Hz low, Hz high, Hz step, samples, dB, …`. The bin size is
> a maximum (`10k` at 2.4 MS/s gives 192 values of 9375 Hz). Each hop gets
> a wall-clock share of `-i`, so a slow CPU averages fewer samples and
> the `samples` column says how many. `-rtltcp` and `-iio` name network
> radios, because the tool reads no config file. The tuner's DC line at
> every hop centre is removed by reporting only what *changed* between
> FFT frames; `-keep-dc` is the A/B, and when an AGC moves the offset
> `power` says so — the remedy is a fixed `-gain`. Status: a Galaxy S5
> keeps up at 2.4 MS/s; the DC removal was retested once on a PlutoSky,
> and the follow-up has not been run on that board.

**Key takeaways**

- **It writes rtl_power's CSV, so rtl_power's tools read it.** The
  differences are the default crop, gain in tenths of a dB, and a fixed
  Hann window.
- **The `samples` column is honest.** A hop stops at its time budget with
  whatever it averaged, so sweeps stay on schedule and the CSV shows the
  cost.
- **The DC line is removed by what it is.** A tuner's offset is the same
  complex value in every FFT frame; a received signal is not. Nothing is
  painted over by default.
- **Automatic gain makes hops incomparable.** With `auto`, every hop is
  measured at the gain the AGC chose for it. Fix the gain first.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Range and resolution | lower edge, upper edge, largest bin width | `-f 88M:108M:10k` |
| Cadence | one sweep per interval; stop after a duration, or after one | `-i 30s`, `-e 1h`, `-1` |
| Which radio | a local SDR, an rtl_tcp server, a Pluto's iiod | `-serial`, `-rtltcp host:port`, `-iio addr` |
| Honest averaging | each hop ends at its share of the interval | the CSV's `samples` column |
| The spike at each hop centre | removed; kept for comparison; filled; reported | `-keep-dc`, `-fill-dc`, `-dc-report` |
| Comparable hops | one gain for the whole sweep, in tenths of a dB | `-gain 280` |
| A slow CPU | less work per second, more hops | `-rate 1024000` |
| From survey to system | sweep or probe what the survey found | `gophertrunk hunt` with `-band` or `-candidates` |

## In this post

- **What you're building** — a spectrum logger, and what a hop is.
- **The config** — four command lines and the flags.
- **First run — what healthy looks like** — the banner, the CSV, the
  spike that is not there, and the hand-off to `hunt`.
- **When it doesn't work** — symptom → cause → fix.
- **Variations** — phone, Pluto, fine bins, heatmaps.

## What you're building

A survey: hours of a band's occupancy in a file. `power` tunes the SDR to
a frequency, averages an FFT
[power spectrum]({{ '/reference/power-spectral-density/' | relative_url }})
there, writes a line, retunes one step up, and repeats to the top of the
range — then starts again on the next interval. Each step is a **hop**.
At the default 2.4 MS/s a hop keeps the centre three quarters of the
spectrum (`-crop 0.25` drops an eighth at each edge, where the tuner's
filter rolls off), and hops abut exactly, so a sweep's lines concatenate
into one continuous spectrum.

It is the slow, patient complement to
[The Hunt]({{ '/blog/deep-dives/the-hunt-02-wideband-sweep-engine/' | relative_url }})'s
own sweep. `hunt` crosses a band once,
[finds carriers]({{ '/blog/deep-dives/the-hunt-03-peak-occupancy-detection/' | relative_url }})
and tries to identify them. `power` identifies nothing; it records levels over time,
which is what tells you a control channel (always there) from a voice
channel (there in bursts) from a neighbour's weather station (there every
few minutes).

Every retune has a side effect. A tuner leaks a little of its own
[local oscillator]({{ '/reference/local-oscillator/' | relative_url }})
into its output — a [DC offset]({{ '/reference/dc-offset/' | relative_url }})
— which a spectrum shows as a narrow carrier at exactly the tuned
frequency. A sweep retunes for every hop, so that false carrier used to
sit in the middle of every line.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Top: the 88 to 108 megahertz sweep at 2.4 megasamples per second laid out as twelve abutting hops, eleven of 192 bins and a last one of 22, with the tuner's DC line falling at the centre of each full hop. Bottom: the three centre bins of a hop with the DC line kept, the centre eleven decibels up and its neighbours six decibels lower, and the same bins by default, flat, after what stood still between FFT frames is removed">
  <text x="340" y="18" text-anchor="middle" fill="var(--accent)" font-size="9">every hop retunes, so the tuner's DC line lands at every hop centre</text>
  <rect x="40" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="94" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="148" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="202" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="256" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="310" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="364" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="418" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="472" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="526" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="580" y="30" width="54" height="40" fill="none" stroke="currentColor"/>
  <rect x="634" y="30" width="6" height="40" fill="none" stroke="currentColor"/>
  <line x1="67" y1="30" x2="67" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="121" y1="30" x2="121" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="175" y1="30" x2="175" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="229" y1="30" x2="229" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="283" y1="30" x2="283" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="337" y1="30" x2="337" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="391" y1="30" x2="391" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="445" y1="30" x2="445" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="499" y1="30" x2="499" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="553" y1="30" x2="553" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <line x1="607" y1="30" x2="607" y2="70" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <text x="40" y="86" fill="var(--fg-muted)" font-size="9">88 MHz</text>
  <text x="640" y="86" text-anchor="end" fill="var(--fg-muted)" font-size="9">108 MHz</text>
  <text x="340" y="86" text-anchor="middle" fill="currentColor" font-size="9">12 hops: 11 × 192 bins of 9375 Hz (1.8 MHz), then 22 bins</text>
  <text x="116" y="112" text-anchor="middle" fill="currentColor" font-size="10">-keep-dc</text>
  <text x="116" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">centre +11 dB, neighbours 6 dB lower</text>
  <rect x="60" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="76" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="92" y="168" width="14" height="32" fill="none" stroke="var(--accent)"/>
  <rect x="108" y="144" width="14" height="56" fill="none" stroke="var(--accent)"/>
  <rect x="124" y="168" width="14" height="32" fill="none" stroke="var(--accent)"/>
  <rect x="140" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="156" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <line x1="200" y1="176" x2="388" y2="176" stroke="var(--accent)"/>
  <text x="294" y="168" text-anchor="middle" fill="var(--accent)" font-size="9">bins −1, 0, +1: Σ|X|² − |ΣX|²/m per 16 frames</text>
  <text x="456" y="112" text-anchor="middle" fill="currentColor" font-size="10">default</text>
  <text x="456" y="126" text-anchor="middle" fill="var(--fg-muted)" font-size="9">what stood still between FFTs is removed</text>
  <rect x="400" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="416" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="432" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="448" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="464" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="480" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <rect x="496" y="188" width="14" height="12" fill="none" stroke="currentColor"/>
  <line x1="40" y1="200" x2="640" y2="200" stroke="var(--fg-muted)"/>
  <text x="340" y="214" text-anchor="middle" fill="var(--fg-muted)" font-size="8">bar heights are the shape measured on a PlutoSky sweep of the FM band (issue #1230), not a scale</text>
</svg>
<figcaption>A sweep is a row of abutting hops, and each hop's centre carries the tuner's own DC line unless it is taken out; by default it is, by subtracting what does not change from frame to frame.</figcaption>
</figure>

## The shopping list

| Item | Price (rough) | Notes |
|---|---|---|
| Any SDR GopherTrunk drives | $0 if you have one | a local dongle, one behind an rtl_tcp server, or a Pluto or PlutoSky |
| An antenna for the band surveyed | $0–$30 | a survey with the wrong antenna is a survey of the antenna |
| Something to read a CSV | $0 | a spreadsheet, `awk`, or the rtl_power heatmap script |

Stop the daemon first: only one program can use a dongle.

## The config

`power` reads no config file. Everything is on the command line, and a
radio the daemon knows from `config.yaml` has to be named again here.

```sh
gophertrunk power -f 88M:108M:10k -i 10 -1 -out fm.csv
gophertrunk power -f 851M:869M:12.5k -i 30s -e 1h -gain 280 -out 800.csv
gophertrunk power -rtltcp 192.168.1.50:1234 -f 851M:869M:12.5k -i 30s -e 1h -out 800.csv
gophertrunk power -iio 192.168.2.1 -f 88M:108M:10k -gain 400 -out fm.csv
```

The first is a ten-second smoke test on a band that is never empty. The
second is a real survey: the 800 MHz band in bins no wider than 12.5 kHz,
a sweep every 30 seconds for an hour, at a fixed 28 dB. The third is the
same from a remote dongle, the fourth from a Pluto over its USB network
link.

| Flag | Default | Meaning |
|---|---|---|
| `-f lower:upper:bin` | required | range and largest bin width, with k, M or G suffixes |
| `-i` | 10s | one sweep starts every interval; a bare number is seconds |
| `-e`, `-1` | run until Ctrl-C | stop after a duration, checked between sweeps, or after one sweep |
| `-out` | stdout | CSV path; a trailing file name also works |
| `-rtltcp`, `-iio`, `-serial` | the only local SDR | an rtl_tcp server, a Pluto's iiod, or one of several local SDRs; mutually exclusive |
| `-gain` | auto | tenths of a dB: 280 is 28 dB, and 28.0 also reads as 28 dB |
| `-rate` | 2400000 | sample rate; each hop covers the rate times one minus the crop |
| `-crop` | 0.25 | fraction of each FFT dropped at the edges |
| `-settle` | 50 ms local, 250 ms network | samples discarded after each retune |
| `-keep-dc`, `-fill-dc`, `-dc-report` | off | the DC line: keep it, fill what is left of it, print each hop's measurements |

Two of these surprise people. The **bin size is a maximum**, as in
rtl_power: the FFT is the smallest power of two whose bins are no wider
than you asked. At 2.4 MS/s, 128 points would give 18,750 Hz — too wide
for `10k` — and 256 give 9375 Hz, of which 192 are kept: 1.8 MHz per hop.
And **`-i` is per sweep, not per hop**: each hop integrates the interval
divided by the number of hops, less the settle. `-ppm` and `-bias-tee`
exist too.

## First run — what healthy looks like

On stderr, the plan — and, when bins come out narrower than requested, the
arithmetic:

```text
power: rtlsdr[00000001], 88000000–108000000 Hz in 12 hops of 192 × 9375.0 Hz bins (256-point FFT), 10s per sweep
power: bins are 2400000 S/s ÷ 256 = 9375.0 Hz (the largest width not over the 10000 Hz asked for); each hop keeps the centre 192 of 256 bins (1800.0 kHz, -crop 0.25), so a CSV line has 192 values
```

In the file, one line per hop. This is the documentation's example row:

```text
date, time, Hz low, Hz high, Hz step, samples, dB, dB, ...
2026-10-03, 21:30:05, 88000000, 89800000, 9375.00, 4000000, -61.20, -60.95, ...
```

Value *k* of a line is at `Hz low + k × Hz step`. Levels are uncalibrated
[dBFS]({{ '/reference/dbfs/' | relative_url }}), and a carrier reads 1.8
to 3.2 dB low through the Hann window — compare levels with each other,
not with an absolute reference.

**Read `samples` before you read dB.** A CPU that keeps up averages about
the interval divided by the hops, less the settle, times the rate. See
what each hop actually got with `awk -F', ' '{print $3, $6}' fm.csv`. If a
sweep averages under 90 % of that, `power` says why, once: either
`power: note — the FFT can't keep up with 2400000 S/s on this CPU …`,
with a count of IQ chunks dropped — lower `-rate` — or a note that *no IQ
was dropped on the way to the FFT, so the CPU kept up*, which means the
SDR delivered too few samples and a lower rate will not help. Either way
the sweeps still start on time. Measured on phones in
[#1230](https://github.com/MattCheramie/GopherTrunk/issues/1230): a Galaxy
S5 kept up at 2.4 MS/s from its fourth hop on; a Galaxy S5 Neo managed
about 38 %.

**The spike that is not there.** Look at value *k* = 96 of a 192-value
line, the hop's centre. Before the fix, three bins there stood up on
every line (rtl_power hides its own by filling its DC bin in from the
bins beside it). A PlutoSky sweep of the FM band in #1230 showed one every
3.6 MHz: the centre bin 11 dB over the noise and its neighbours 6 dB
under that — exactly a constant offset seen through a Hann window.

`power` removes it by what it is. The offset is the same complex value
in every FFT frame of a hop; anything off the air turns from frame to
frame. So for the three centre bins the accumulator keeps the frames'
complex sum beside their power sum, in blocks of 16 frames, and reports
only the part that varied. Noise and real signals there read as before —
there is no notch. On real RTL-SDR noise with a 4096-point FFT the centre
went from −58.8 dBFS to −67.4 against a −67.8 floor. To see the
difference on your own tuner:

```sh
gophertrunk power -f 88M:108M:10k -i 10 -1 -out with.csv -keep-dc
gophertrunk power -f 88M:108M:10k -i 10 -1 -out without.csv
```

The removal takes out what holds *still*. If the tuner's offset moves
while a hop is being measured, part of the line stays, and `power` tells
you once per run:

```text
power: note — sweep 1: after the tuner's DC line was removed, a line of its shape is still at the centre of 1 of 6 hops (100.6000 MHz; up to 7.3 dB over the bins beside it). …
```

That is the reporter's retest on the PlutoSky: clean in hops with one
dominant station, the whole line left in a hop with two nearly equal
ones. An AGC hunting between them fits, but that cause is inferred from
the sweep, not confirmed on the board. The remedy that costs nothing is a
fixed gain, `-gain 400`, which holds the offset still and makes the hops
comparable. `-fill-dc` fills the three bins in from their neighbours, as
rtl_power does — and is not the default, because three bins cannot tell
a leftover offset from a narrowband signal exactly on a hop's centre; a
recording in the repository has one, and the fill erases it. `-dc-report`
prints a line per hop for a bug report:

```text
power: dc 98.8000–102.4000 MHz (centre 100.6000): ... dBFS stood still and was removed; centre ... dB over the bins beside it, shape ..., alike ..., turn ...° a frame (... Hz): clean
```

**From survey to hunt.** Average each bin over the whole log and list the
strongest:

```sh
awk -F', ' '{for(i=7;i<=NF;i++){f=$3+(i-7)*$5; s[f]+=$i; n[f]++}}
  END{for(f in s) printf "%.4f MHz %.1f dB\n", f/1e6, s[f]/n[f]}' 800.csv \
  | sort -k3 -n -r | head -20
```

A bin that is strong in *every* sweep is a candidate control channel; one
that is strong in some is traffic. Neither is an identification — hand
them to the tool that identifies
([The Hunt Part 1]({{ '/blog/deep-dives/the-hunt-01-what-discovery-means/' | relative_url }})
starts that story):

```sh
gophertrunk hunt -serial 00000001 -band 851:869
gophertrunk hunt -serial 00000001 -candidates 857.2625,858.4875 -no-sweep
```

`-band` is in MHz and sweeps the range; `-candidates` probes the listed
frequencies directly.

## When it doesn't work

| Symptom | Likely cause | Fix |
|---|---|---|
| 192 values of 9375 Hz when 10 kHz was asked for | the bin size is a maximum and the FFT is a power of two | for exact 10 kHz bins use `-rate 2560000`, if the dongle runs it without drops |
| The error says no SDR was found, though the config lists one | the one-shot tools read no config file | name the radio: `-rtltcp` or `-iio`; the error prints the flag to use |
| A strong carrier appears one hop away from where it belongs | over rtl_tcp, samples from the previous frequency were still in flight | raise `-settle` |
| A spike is still at the centre of some hops, and a note says so | the offset moved during those hops, or a narrowband signal sits exactly there | fix the gain with `-gain`; failing that `-fill-dc`; post the `-dc-report` lines |
| A steady carrier you know is there is missing | an unmodulated carrier within about a sixteenth of a bin of a hop centre, or exactly one bin either side, is removed with the line | compare with `-keep-dc`, or move the lower edge of `-f` so the centres shift |
| Levels step up and down from hop to hop | automatic gain: each hop was measured at a different gain | use a fixed `-gain` |
| The samples column is low | a slow CPU, or an SDR that delivered too few samples; the note says which | lower `-rate` for the first; check the USB link for the second |

### How this recipe shapes operator practice

- **Fix the gain before you compare anything.** A survey taken on `auto`
  is a record of the AGC's opinions;
  [gain staging]({{ '/blog/tutorials/analog-edge-03-gain-staging/' | relative_url }})
  is how to pick the number.
- **Survey long, hunt short.** An hour of `power` costs nothing and tells
  you which three frequencies deserve `hunt`'s attention.
- **Keep the stderr with the CSV.** The plan and the notes are the
  survey's metadata.

## Variations

- **From a Pluto.** `-iio` with the board's address;
  [Part 13]({{ '/blog/tutorials/cookbook-volume-2-13-pluto-native/' | relative_url }})
  is the driver's own recipe.
- **Fine bins on a narrow band.** `-f 446.0M:446.2M:1k` is a single hop
  with a 4096-point FFT and bins of about 586 Hz.
- **A heatmap.** The heatmap script written for rtl_power takes the CSV
  as is: `python3 heatmap.py fm.csv fm.png`.

The reference is the
[power sweep doc page]({{ '/power-sweep.html' | relative_url }}); the
discovery tool it feeds is in the [hunt doc]({{ '/hunt.html' | relative_url }}).

## Where this goes next

A sweep looks at everything and decodes nothing.
[Part 12]({{ '/blog/tutorials/cookbook-volume-2-12-meter-walk/' | relative_url }})
does the opposite on a single frequency: 868.95 MHz, where heat
allocators and water meters send a few milliseconds of Wireless M-Bus
every few minutes.

## FAQ

**Is gophertrunk power compatible with rtl_power?**
Its CSV is: the same fields in the same order, one line per hop, so tools
written for rtl_power read it unchanged. The command line differs: the
default crop is 0.25 (rtl_power's is 0), gain is in tenths of a dB, and
there is no window or downsampling option.

**Why does a 10 kHz bin size give 9375 Hz bins?**
The bin size is a maximum. The FFT is the smallest power of two whose
bins are no wider than requested, and at 2.4 MS/s that is 256 points of
9375 Hz.

**What is the spike in the middle of every hop of a spectrum sweep?**
The tuner's own DC offset: a little of its local oscillator leaking into
its output, which appears as a narrow carrier at the tuned frequency.
`gophertrunk power` subtracts what stays constant from one FFT frame to
the next; `-keep-dc` leaves it in.

**Is the DC line removal verified on real hardware?**
On an RTL-SDR's own noise, yes: the centre fell to within half a dB of
the floor. On a PlutoSky the reporter's retest came out clean in some
hops and not in others; the note, `-fill-dc` and `-dc-report` that
answered it have not yet been run on the board. The rtl_tcp and iio
sources are synthetic-verified, against fake servers
(`TestPowerSweepOverRTLTCP`, `TestPowerSweepOverIIO`).

## Series navigation

**Part 11 of 14** · ←
[Part 10: The Phone Rig — GopherTrunk in Termux]({{ '/blog/tutorials/cookbook-volume-2-10-phone-rig/' | relative_url }})
· Next →
[Part 12: The Meter Walk — Wireless M-Bus at 868 MHz]({{ '/blog/tutorials/cookbook-volume-2-12-meter-walk/' | relative_url }})
