---
layout: page
title: Hardware setup
description: RTL-SDR, HackRF and Airspy dongles, udev rules, DVB blacklist, and supported USB chipsets
nav_group: Install
---

# Hardware Setup

GopherTrunk ships with pure-Go drivers for three SDR families — no
`librtlsdr`, `libhackrf`, `libairspy`, `libusb`, or C toolchain on
the build host. All three drivers share the same pure-Go USB
transport (USBDEVFS on Linux, IOKit on macOS, WinUSB on Windows).

## Supported devices

| Family | Driver | USB IDs | Status |
| --- | --- | --- | --- |
| **RTL-SDR** (RTL2832U / RTL2838U + R820T / R820T2 / R828D / E4000 / FC0012 / FC0013 / FC2580) | `rtlsdr` | `0x0bda:0x2832` · `0x0bda:0x2838` | Production — on-air-validated across Linux / macOS / Windows. |
| **HackRF One / Jawbreaker / Rad1o / Pro** | `hackrf` | `0x1d50:0x6089` · `0x1d50:0x604b` · `0x1d50:0xcc15` | Wire-protocol-complete; exercised and fixed against attached hardware in the field (Pro board-ID, `fpga_dc_block`, `dc_avoid`, `rf_amp`). No hardware-gated test harness yet. |
| **Airspy R2 / Airspy Mini** | `airspy` | `0x1d50:0x60a1` | Wire-protocol-complete; exercised against attached hardware in the field (macOS async bulk-IN, native-rate behaviour), with a hardware-gated harness (`internal/sdr/airspy/airspy_real_test.go`, `GOPHERTRUNK_AIRSPY_REAL`). |
| **Airspy HF+ Discovery / HF+ Dual Port / legacy HF+** | `airspyhf` | `0x03eb:0x800c` | Wire-protocol-complete; HF (9 kHz – 31 MHz) + VHF (60 – 260 MHz). On-air validation against attached hardware is the documented follow-up. |
| **rtl_tcp remote** (any librtlsdr-shipped server) | `rtltcp` | TCP | Remote RTL-SDR mounted over the network. See [Remote rtl_tcp SDRs](#remote-rtl_tcp-sdrs). |
| **SoapySDRServer remote** (USRP / LimeSDR / bladeRF / HackRF / Airspy / RTL-SDR / SDRplay …) | `soapyremote` | TCP | Any SoapySDR-supported radio mounted over the network with 16/32-bit IQ + control. Field-tested on air against USRP X310 and B210 rigs (single-branch; MRC diversity combining is still experimental). See [Remote SoapySDRServer SDRs](#remote-soapysdrserver-sdrs). |

The HackRF and Airspy / Airspy HF+ drivers speak the documented
libhackrf, libairspy, and libairspyhf USB vendor protocols directly
(transceiver / receiver mode, frequency, sample rate, LNA / VGA /
mixer / amp / attenuator / bias-tee, bulk-IN sample reaper with
real-time decode of HackRF int8 IQ and Airspy / HF+ INT16_IQ into
`complex64`). Their wire protocols are exercised by unit tests
against `usb.MockTransport`. SDRPlay, USRP and BladeRF require
vendor C libraries and are out of scope for the zero-CGO build.

At enumeration time each driver reports the canonical model name
rather than echoing whatever the USB descriptor happens to carry:
HackRF maps the PID to `HackRF One` / `HackRF Jawbreaker` / `Rad1o`,
Airspy R2/Mini detects the `MINI` substring in the descriptor to
emit `R820T (Airspy R2)` or `R820T (Airspy Mini)`, and the HF+
driver distinguishes Discovery / Dual Port / legacy units the same
way. On Open, the HackRF driver also reads the firmware's
BOARD_ID_READ + VERSION_STRING_READ control transfers, so the
operator-visible `TunerName` field carries the running firmware
version and a `+ PortaPack` tag when a PortaPack / Mayhem build is
detected. The HF+ driver appends its firmware version the same way.

### RTL-SDR tested combinations

| Device | Tuner | Notes |
| --- | --- | --- |
| **NooElec NESDR Smart v5** | R820T2 | 0.5 ppm TCXO, software-controllable bias-tee. Use `bias_tee: true` in config to power an external LNA via the SMA. |
| NooElec NESDR Smart (v4 and earlier) | R820T2 | TCXO; no bias-tee on early units. |
| Generic RTL-SDR Blog v3 / v4 | R820T2 / R828D | Bias-tee on most units. |
| Plain RTL2832U DVB-T sticks | R820T | No TCXO; expect a few ppm offset — set `ppm:` in config after measuring. |

> **"RTL2838U" dongles are already supported.** Many economy SDRs are
> marketed or labelled by their demodulator/USB-bridge chip — the
> **RTL2838U** — which is a variant of the RTL2832U, *not* a tuner.
> These units enumerate as `0x0bda:0x2838` (and usually report
> `RTL2838UHIDIR` in `gophertrunk sdr list`); the actual tuner inside is
> an R820T2 or R828D, both supported above. If your dongle says "RTL2838",
> it works out of the box — no extra driver needed.

If you have a v5 (or any modern dongle with a bias-tee) and want to
power an LNA, the config snippet looks like:

```yaml
sdr:
  devices:
    - serial: "00000001"      # whatever `gophertrunk sdr list` shows
      role: control            # or voice / auto
      ppm: 0                   # 0 is fine for TCXO-equipped units
      gain: "auto"             # TENTHS of a dB, not dB — "496" = 49.6 dB
      bias_tee: true           # 5V on the SMA — only enable if you want it
```

> **Always set `gain:` explicitly.** A device listed in
> `sdr.devices[]` with no `gain:` key opens at whatever the librtlsdr
> default chose for the tuner — typically a middle-range fixed value
> that's too low for many LNA + antenna combinations. The field
> symptom is "voice grants land on a Voice SDR but every call ends
> `reason=timeout` with an empty WAV" (issue #356 follow-up). The
> daemon now surfaces this at startup with `sdr: no gain configured
> for device ...`; if you see that line, set `gain: "auto"` for AGC
> or pick a tenth-dB value that matches your front-end.

> **`gain:` is in TENTHS of a dB, not whole dB.** This is the most
> common first-run footgun for operators coming from SDRTrunk / OP25 /
> gqrx, which all take whole dB. In GopherTrunk `"320"` = 32 dB and
> `"496"` = 49.6 dB; a bare `"32"` is parsed as **3.2 dB**, which the
> driver then snaps to the bottom of the tuner's gain ladder, leaving
> the radio effectively deaf (no control-channel lock, no decodes).
> Multiply your usual dB figure by 10, or use `"auto"`. The daemon now
> warns at startup (`gain looks like dB, not tenths-of-dB ...`) when a
> bare integer gain parses to ≤ 5.0 dB, and logs the applied gain in dB
> on every device (`sdr: gain set ... gain_db=...`). A decimal form like
> `"32.0"` is taken as whole dB, so that works too.

> **Multi-site `role: wideband` dongles: prefer `gain: "auto"`.** Every
> tap on a wideband dongle shares one antenna, one centre and one gain.
> A single *fixed* gain can't serve sites of differing strength: a value
> chosen so the strongest site doesn't clip leaves weaker co-tenants
> under-amplified, sitting dead and flat at the ADC noise floor — and the
> receiver's own post-discriminator AGC can't recover SNR that was lost
> at the converter. AGC (`gain: "auto"`) lets the front end run the band
> hot enough for weak sites to clear the floor; in field testing it took
> a co-tenant control channel from zero decoded TSBKs to hundreds (issue
> #749). The daemon warns at startup when a multi-tap wideband dongle is
> pinned to a fixed gain. A genuinely weak/distant site may still not
> survive a shared capture even under AGC — give it its own dongle.

### HackRF tested combinations

| Device | PID | Coverage | Gain chain | Bias-tee | Notes |
| --- | --- | --- | --- | --- | --- |
| **HackRF One** | `0x6089` | 1 MHz – 6 GHz, half-duplex 8/10/20 MSPS | RF amp (on/off, +14 dB) + LNA (0–40 dB / 8 dB steps) + VGA (0–62 dB / 2 dB steps) | +3.3 V on `ANT` (HW rev 6+) | Single SMA antenna port. PortaPack add-on (Mayhem firmware) is auto-detected via the VERSION_STRING_READ control transfer — `gophertrunk sdr list` then shows `HackRF One + PortaPack`. |
| HackRF Jawbreaker | `0x604b` | 30 MHz – 6 GHz prototype | Same as One (MAX2837 + MAX5864) | None | Pre-production batch; functional but rarely seen in the field. |
| Rad1o | `0xcc15` | 50 MHz – 4 GHz | Same as One | None | Chaos Communication Camp 2015 badge; same firmware family as HackRF One, identical wire protocol. |

The HackRF has no hardware AGC. Passing `gain: "auto"` selects a
safe fixed split (LNA = 16 dB, VGA = 20 dB, RF amp off); positive
tenth-dB values are distributed across the three stages. The
firmware-reported board ID (BOARD_ID_READ) is the canonical model
identifier and takes precedence over the USB descriptor when the
two disagree.

```yaml
sdr:
  sample_rate: 8_000_000          # HackRF baseband; filter follows automatically
  devices:
    - serial: "0000000000000000a06064c8333819cf"
      role: control
      gain: "400"                  # 40 dB target distributed across LNA + VGA
      bias_tee: false              # set true to power an external LNA
```

### Airspy tested combinations

| Device | PID | Coverage | Max sample rate | Gain chain | Bias-tee | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| **Airspy R2** | `0x60a1` (`airspy`) | 24 – 1700 MHz | 10 MSPS | R820T LNA + Mixer + VGA (each 0–15) with per-stage AGC | +4.5 V on SMA | Most common variant. Identified by the USB Product string `Airspy R2`. |
| **Airspy Mini** | `0x60a1` (`airspy`) | 24 – 1700 MHz | 6 MSPS | Same as R2 | +4.5 V on SMA | Identified by the `MINI` substring in the USB Product string. Same R820T tuner; smaller form factor and lower max rate. |
| **Airspy HF+ Discovery** | `0x800c` (`airspyhf`) | 9 kHz – 31 MHz HF + 60 – 260 MHz VHF | 768 kSPS | HF AGC (firmware-managed) + HF attenuator 0–48 dB (6 dB steps) + +6 dB LNA preamp | +4.5 V on HF SMA | Most popular HF receiver. `gain: "auto"` enables firmware HF AGC; numeric values map to attenuator step + LNA preamp. |
| Airspy HF+ Dual Port | `0x800c` (`airspyhf`) | Same as Discovery | 768 kSPS | Same as Discovery | +4.5 V on the HF SMA only | Identified by the `DUAL` substring. The VHF SMA does not carry bias voltage. |
| Legacy Airspy HF+ | `0x800c` (`airspyhf`) | Same as Discovery | 768 kSPS | Same as Discovery | Hardware-revision dependent | Pre-Discovery hardware; uncommon. |

The R2 / Mini driver pins the device to `INT16_IQ` sample mode at
open-time and reads the firmware's advertised sample-rate table —
`SetSampleRate` then picks the closest available rate, so a
`sample_rate: 10_000_000` config selects the 10 MSPS slot on R2 and
a `sample_rate: 6_000_000` config selects the 6 MSPS slot on Mini
without per-device overrides. **For a single narrowband control
channel, don't reach for the top native rate** — see "Choosing a
sample rate" below. The HF+ driver does the same; it also reads
VERSION_STRING_READ to expose the firmware version in `TunerName`.

```yaml
sdr:
  sample_rate: 768_000             # HF+ Discovery max
  devices:
    - serial: "3652b46d6e6f8867"
      role: control
      gain: "auto"                 # firmware HF AGC handles the HF band well
      bias_tee: false              # set true to power an active HF antenna
```

### Choosing a sample rate

A wideband front-end like the Airspy R2 advertises native rates up to
10 MSPS, but a trunking **control channel is only ~12.5 kHz wide** and the
down-converter normalises it to 48 kHz before decode. Capturing at the top
native rate buys nothing for a single narrowband channel — and on some
units it actively *hurts*.

On an R2 captured at its native 10 MSPS, the same control channel — same
antenna, gain, and centre frequency — carried roughly **16 dB worse
in-channel SNR** than the identical signal captured at 2.5 MSPS (issue
#771: EVM 22.5% / demod SNR 9.5 dB versus 7.4% / 19.7 dB). The penalty
sits *inside* the channel, co-band with the signal, so no downstream
filtering removes it, and it is independent of gain (the same deficit shows
at 60 dB and 30 dB). The cause is front-end clock phase noise / reciprocal
mixing at the higher native clock — not clipping, and not GopherTrunk's
DSP, which is rate-invariant. The 2.5 MSPS capture locked; the 10 MSPS one
did not.

> **For a control-channel role, prefer a moderate native rate
> (~2.4–2.5 MSPS).** The `gophertrunk capture` default is `2_400_000` for
> this reason, and the tool prints a one-line hint if you ask for a high
> rate on a narrowband capture. Reach for a high native rate only when you
> genuinely need the wide IQ window — a wideband survey or several
> repeaters at once — and even then verify lock, because the same
> phase-noise penalty can still bite a narrowband channel sitting inside a
> wide capture. (TETRA's 25 kHz channels still decode fine at a moderate
> rate; the down-converter normalises them to 144 kHz.)

```yaml
sdr:
  sample_rate: 2_400_000           # moderate native rate for narrowband CC decode
  devices:
    - serial: "..."                # whatever `gophertrunk sdr list` shows
      role: control
      gain: "auto"
```

#### Will a different radio (e.g. Airspy Mini at 6 MSPS) avoid the deficit?

There is no way to promise this in advance — it has to be measured on the
radio in hand — but the physics narrows it down. The #771 deficit has two
possible sources and the two radios share one of them:

- The **Airspy Mini uses the same R820T2 tuner** as the R2, so the tuner's
  LO phase noise — the reciprocal-mixing term — is common to both. If that
  term dominates the deficit, a Mini will inherit it.
- The two radios **sample on different clocks** (the Mini's top native rate
  is 6 MSPS, the R2's is 10 MSPS). If the deficit is instead dominated by
  phase noise on the *sampling* clock, the Mini's lower top rate could be
  cleaner. That part is genuinely untested.

The one thing the #771 data does settle: **the deficit only appeared at the
R2's *top* native rate.** The same R2's decimated 2.5 MSPS slot was clean
and locked; only the native 10 MSPS was ~16 dB worse. 6 MSPS is the Mini's
*top* native rate, i.e. the same regime that bit the R2 — so it is exactly
the case you cannot assume is clean without checking. For a **single**
control channel on any of these R820T2 radios the safe answer is unchanged:
capture at a moderate/decimated rate (~2.4–2.5 MSPS) and it behaves like the
R2 did at 2.5 MSPS. If the goal is a **wide window covering several sites at
once** (the reason to reach for 6 MSPS at all), measure the top-rate capture
before relying on it — an RTL-SDR already covering that span in wideband
mode is a proven fallback that needs no new hardware.

#### Measuring the in-channel deficit on your own hardware

You don't have to guess whether a given radio/rate carries the deficit —
`replay -diag` reports the in-channel demod quality, so you can measure it
directly. Capture the *same* control channel at a moderate rate and at the
rate under test (same antenna, gain, and centre frequency — only
`-sample-rate` differs), then replay each with `-diag`:

```
# Same CC, two rates, on the radio under test (example: an Airspy Mini)
gophertrunk capture -freq 420900000 -sample-rate 2400000 -gain 600 -seconds 10 \
  -format f32 -out cc_lo.cfile -serial ""
gophertrunk capture -freq 420900000 -sample-rate 6000000 -gain 600 -seconds 10 \
  -format f32 -out cc_hi.cfile -serial ""

gophertrunk replay -in cc_lo.cfile -format f32 -sample-rate 2400000 \
  -protocol p25p1 -demod c4fm -tune-hz -812500 -diag
gophertrunk replay -in cc_hi.cfile -format f32 -sample-rate 6000000 \
  -protocol p25p1 -demod c4fm -tune-hz -812500 -diag
```

Compare the `demod` line each run prints:

```
siglab: demod (c4fm): EVM=7.4%  SNR≈19.7 dB  (n=… symbols)
```

A radio that is *clean at the higher rate* reports near-identical EVM/SNR at
both rates. A radio that is phase-noise-limited at its top rate reports the
higher rate several dB worse — on the R2, EVM went 7.4%→22.5% and demod SNR
19.7→9.5 dB (issue #771). The `-diag` line is emitted whether or not the
channel locks, so it works even when the higher rate no longer decodes.

## Linux

No package install is needed for the build itself; the driver only
needs USB-device permissions at runtime.

Add a udev rule so non-root processes can claim each dongle. One
file per family is fine:

```
# /etc/udev/rules.d/20-rtlsdr.rules
SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2832", MODE="0666"
SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", MODE="0666"

# /etc/udev/rules.d/21-hackrf.rules
SUBSYSTEM=="usb", ATTRS{idVendor}=="1d50", ATTRS{idProduct}=="6089", MODE="0666"
SUBSYSTEM=="usb", ATTRS{idVendor}=="1d50", ATTRS{idProduct}=="604b", MODE="0666"
SUBSYSTEM=="usb", ATTRS{idVendor}=="1d50", ATTRS{idProduct}=="cc15", MODE="0666"

# /etc/udev/rules.d/22-airspy.rules
SUBSYSTEM=="usb", ATTRS{idVendor}=="1d50", ATTRS{idProduct}=="60a1", MODE="0666"

# /etc/udev/rules.d/23-airspyhf.rules
SUBSYSTEM=="usb", ATTRS{idVendor}=="03eb", ATTRS{idProduct}=="800c", MODE="0666"
```

Reload udev (`sudo udevadm control --reload && sudo udevadm trigger`) and
unplug/replug the dongle.

Blacklist the kernel's DVB driver — only matters for RTL-SDR, but
the file is harmless to ship even on HackRF / Airspy-only hosts:

```
# /etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf
blacklist dvb_usb_rtl28xxu
```

See [`install-linux.md`]({{ '/install-linux.html' | relative_url }})
for the full step-by-step including systemd service setup.

## macOS

IOKit lets user-space claim USB devices without rebinding the kernel
driver, so there's no kext to install and no driver swap to perform.
Plug in the dongle and run `gophertrunk sdr list`. See
[`install-macos.md`]({{ '/install-macos.html' | relative_url }}) for
the full step-by-step including the Gatekeeper bypass and launchd
service setup.

## Windows

Bind each dongle to **WinUSB** with Zadig (bundled in the Windows
installer — Start Menu → GopherTrunk → "Install RTL-SDR driver
(Zadig)") once per device. The same Zadig walkthrough also works
for HackRF, Airspy and Airspy HF+ — pick the device in the
dropdown, choose the WinUSB driver, click Replace. See
[`install-windows.md`]({{ '/install-windows.html' | relative_url }})
for the click-by-click walkthrough.

Airspy R2 / Mini in particular: the official Airspy installer
typically binds **libusbK**, which is not the in-box WinUSB.sys
GopherTrunk talks to. If `sdr list` shows the device but
`gophertrunk -config …` fails on open with `winusb: device rejected
request (ERROR_GEN_FAILURE …)`, re-bind to WinUSB via Zadig — the
hardware is reachable but the function driver is mismatched.

## Verifying the build

```sh
make build
./bin/gophertrunk sdr list
```

You should see one row per attached dongle with index, serial,
tuner type, and the supported gain values:

- **RTL-SDR** dongles report a `R820T2` / `R828D` / `E4000` / `FC0012` /
  `FC0013` / `FC2580` `TunerName` depending on what the driver detects
  on the RTL2832U.
- **HackRF** dongles report `MAX2839+MAX5864 (fw <version>)` —
  `<version>` is what the firmware returns via `VERSION_STRING_READ`,
  e.g. `git-2024.02.1`. A `+ PortaPack` suffix appears in `Product`
  when a Mayhem build is detected.
- **Airspy R2 / Mini** report `R820T (Airspy R2)` or
  `R820T (Airspy Mini)`; the variant is inferred from the USB
  descriptor's Product string.
- **Airspy HF+** reports `Airspy HF+ Discovery` /
  `Airspy HF+ Dual Port` / `Airspy HF+` (with a firmware suffix when
  available) for both `Product` and `TunerName` — there's no
  conventional tuner chip to name.

The `Driver` column reads `rtlsdr`, `hackrf`, `airspy`, or
`airspyhf` for each row.

## Sharing one dongle across multiple repeaters

A single SDR can monitor several conventional DMR Tier II repeaters as
long as every carrier falls inside the dongle's IQ bandwidth. The
dongle is pinned to a centre frequency; an internal channelizer
extracts one narrow-band IQ stream per repeater and feeds an
independent T2 decoder for each. No extra hardware is needed beyond
the one dongle, and there is no per-repeater hardware re-tune.

Add a `role: wideband` entry to `sdr.devices` in your config:

```yaml
sdr:
  sample_rate: 2_400_000
  devices:
    - serial: "00000003"
      role: wideband
      center_freq_hz: 453_500_000
      channels:
        - frequency_hz: 453_125_000
          system: "regional-dmr-t2"
        - frequency_hz: 453_275_000
          system: "regional-dmr-t2"
        - frequency_hz: 453_775_000
          system: "regional-dmr-t2"
        - frequency_hz: 454_100_000
          system: "regional-dmr-t2"

trunking:
  systems:
    - name: "regional-dmr-t2"
      protocol: dmr-tier2
      # Tier II is conventional, but trunking.System.Validate()
      # requires a non-empty control_channels list. List the same
      # repeater carriers - the wideband engine ignores them when
      # choosing the state machine.
      control_channels:
        - 453_125_000
        - 453_275_000
        - 453_775_000
        - 454_100_000
      talkgroup_file: "/etc/gophertrunk/talkgroups-dmr.csv"
```

The daemon programs the dongle's tuner to `center_freq_hz` once, opens
a single IQ stream, and routes one decimated 48 kHz IQ stream per
configured `channels[].frequency_hz` into a separate DMR state
machine. Grants and `cc.locked` events fire per repeater frequency
just like they do for a dedicated dongle.

### Mixing DMR Tier II and Tier III on one dongle

A wideband dongle can host a DMR Tier III control-channel tap
alongside Tier II conventional carriers. Use `protocol: dmr` for the
T3 system, list the T3 control frequency under `control_channels`,
and have one of the dongle's `channels[]` entries point at that
frequency:

```yaml
sdr:
  devices:
    - serial: "00000003"
      role: wideband
      center_freq_hz: 851_500_000
      channels:
        - frequency_hz: 851_037_500    # T3 CC
          system: "regional-dmr-t3"
        - frequency_hz: 852_125_000    # T2 conventional carrier
          system: "neighbour-dmr-t2"

trunking:
  systems:
    - name: "regional-dmr-t3"
      protocol: dmr                    # Tier III trunked
      control_channels: [851_037_500]  # MUST include the wideband channel above
      dmr_band_plan:                   # REQUIRED for T3 voice (LCN → Hz)
        linear: { base_hz: 851_012_500, spacing_hz: 25_000, offset: 1 }
    - name: "neighbour-dmr-t2"
      protocol: dmr-tier2              # Tier II conventional
      control_channels: [852_125_000]
```

The engine picks the right state machine per channel (Tier III's
`ControlChannel` for `protocol: dmr`, Tier II's `ConventionalChannel`
for `protocol: dmr-tier2`).

A DMR Tier III voice-grant CSBK identifies its traffic channel by a
12-bit Logical Channel Number (LCN), not an absolute frequency, so the
T3 system **must** carry a `dmr_band_plan` to resolve LCN → downlink
Hz. Provide exactly one of:

- `linear` — a regular grid: `freq = base_hz + (lcn - offset) × spacing_hz`.
  Set `offset: 1` for sites that number LCNs from 1 (the common case).
- `table` — an explicit list of `{ lcn, freq_hz }` pairs for irregular
  sites.

Without it, T3 control-channel monitoring still works but every voice
grant is dropped with `decode.error stage=no-bandplan` (and the daemon
logs a warning at startup). Tier II carries its repeater frequency
directly and needs no band plan.

### P25 trunked control channel on a wideband dongle

A wideband dongle can also host a P25 control channel — Phase 1 (C4FM
or CQPSK / LSM) or Phase 2 (H-DQPSK TDMA). The configuration
mirrors the DMR Tier III case: the wideband channel sits on the
system's declared `control_channels`, and the engine decodes the TSBK
(Phase 1) or MAC PDU (Phase 2) chain inline. Voice grants ride on the
existing physical voice pool — see the "Limits" section below.

```yaml
sdr:
  devices:
    - serial: "00000003"
      role: wideband
      center_freq_hz: 851_500_000
      channels:
        - frequency_hz: 851_037_500       # P25 Phase 1 CC
          system: "regional-p25"

trunking:
  systems:
    - name: "regional-p25"
      protocol: p25                       # Phase 1
      control_channels: [851_037_500]     # MUST include the wideband channel above
      # For a system that actually transmits a linear/CQPSK waveform
      # (Linear Simulcast Modulation) rather than C4FM, opt in to the
      # linear-CQPSK demod path. NOTE: simulcast alone does NOT require
      # this — most simulcast systems are C4FM; set cqpsk only when a
      # strong, clean signal won't lock in C4FM (issue #935):
      # p25_phase1_demod_mode: cqpsk
```

For P25 Phase 2 use `protocol: p25-phase2`; the same per-system
`p25_phase2_*` knobs (trellis / RS / interleave / scrambler / clock
mode) that apply to a dedicated CC SDR apply to the wideband channel
too — see the trunking systems section of `config.example.yaml`.

### One SDR for control + voice (virtual voice pool)

Voice grants can also be decoded on the same wideband dongle that's
hosting the CC tap, as long as the grant frequency falls inside the
dongle's IQ window. Set `voice_taps: N` on the wideband entry; the
daemon spins up `N` per-grant DDC tuners that subscribe to the
dongle's IQ stream on demand and emit 48 kHz IQ centred on the grant
frequency — exactly what the existing P25 / DMR voice composer
chains expect. The result is a single SDR doing both jobs, with no
physical `role: voice` dongle required (for grants that fit in the
window).

```yaml
sdr:
  devices:
    - serial: "00000003"
      role: wideband
      center_freq_hz: 851_500_000
      voice_taps: 4               # allow up to 4 concurrent voice calls
      channels:
        - frequency_hz: 851_037_500
          system: "regional-p25"

trunking:
  systems:
    - name: "regional-p25"
      protocol: p25
      control_channels: [851_037_500]
```

This works identically for **DMR Tier III** — point the channel at a
`protocol: dmr` system instead. The only extra requirement is that the
T3 system carry a `dmr_band_plan` (see "Mixing DMR Tier II and Tier III
on one dongle" above): the voice composer can only follow a grant once
the decoder has resolved its LCN to a frequency, so a T3 system without
a band plan emits no voice grants for the taps to serve.

**Timeslots count as separate calls.** A DMR carrier is 2-slot TDMA, so
one 12.5 kHz repeater can run *two* independent voice calls at once —
one on TS1, one on TS2. GopherTrunk treats `(frequency, timeslot)` as
the call identity: each slot's grant binds its own voice tap (or
`role: voice` SDR) and is tracked, recorded, and logged as a distinct
call. To follow both slots of a carrier simultaneously you therefore
need at least **two** voice taps/devices that cover the frequency; with
only one, the engine serves whichever slot it bound first and the other
slot waits for a free tap (or preempts by priority). Per-slot recordings
are disambiguated by a `_ts1` / `_ts2` suffix on the WAV filename, and
the slot is carried through the call log (`timeslot` column) and the
REST/SSE/gRPC call APIs.

The 2-slot interleaved voice decoder is the **default** for DMR
Tier II conventional and Tier III trunked systems: rather than relying
on the tap to isolate one slot, each call decodes its own timeslot from
the carrier's interleaved burst stream and is routed by the embedded
Link Control's talkgroup. DMR Tier I direct-mode is genuinely
single-slot and stays on the single-slot decoder. The per-system
`dmr_interleaved_voice` key is a tri-state override — leave it unset
for the per-protocol default, or set `true` / `false` to force it. A
few embedded-signalling constants (de-interleave order, EMB FEC, CRC
polynomial) still await a real-capture cross-check (see
[docs/status.md](status.md)).

How spillover works: when a grant's frequency lands *outside* the
wideband IQ window (more common on geographically spread P25 systems
than on a single-site DMR T3 cluster), the virtual tuner returns
out-of-band and the engine binds a physical `role: voice` SDR
instead — if one is configured. So a typical mixed setup keeps a
single physical voice SDR around as the spillover fallback while the
wideband dongle handles the in-window majority. With no physical
voice SDR present, out-of-window grants are dropped and the daemon
logs a one-shot warning that the grant frequency falls outside every
voice device's tuning window — widen `sample_rate` or move
`center_freq_hz` so the repeaters fit, or add a `role: voice` SDR to
cover the spillover (issues #379, #422).

CPU is roughly linear in `voice_taps`: each tap runs one NCO mixer +
polyphase resampler at the SDR rate during its call's lifetime;
between calls the tap consumes no CPU. The validator caps the value
at 8 to keep one wideband dongle bounded.

### Picking a centre frequency and bandwidth

The usable IQ band is `center_freq_hz ± sample_rate/2` with a 5 %
guard at each edge. At `sample_rate: 2_400_000` that's ±1.08 MHz of
usable spectrum either side of the centre. Put the centre frequency
such that every repeater you care about fits inside that window. The
config validator rejects out-of-band channels at load time with a
message that names the offending entry.

### Tuner strategy

`tuner_strategy` chooses how the dongle's wide IQ stream is sliced:

- `auto` (default) — picks `ddc` for ≤ 6 channels, `polyphase` above.
- `ddc` — one independent NCO mixer + rational resampler per channel.
  Linear cost in channel count; no constraint on the spacing between
  repeaters. Best for a handful (≤ 6) of repeaters.
- `polyphase` — one shared M-channel polyphase channelizer amortises
  the wide-band filter across all channels; a per-channel fine-tune
  DDC cleans up the residual. Wins on CPU once you have 7+ channels.

### Limits

- **DMR + P25.** Wideband supports `protocol: dmr-tier2` (Tier II
  conventional), `protocol: dmr` (Tier III trunked control channel),
  `protocol: p25` (P25 Phase 1 trunked control channel — C4FM and
  CQPSK / LSM), and `protocol: p25-phase2` (P25 Phase 2
  H-DQPSK trunked control channel). Other protocols (NXDN, TETRA,
  …) are not in scope yet.
- **Trunked voice on the same wideband dongle.** With `voice_taps`
  set, the daemon allocates per-grant DDC tuners from the dongle's
  IQ stream so DMR T3 / P25 Phase 1 / P25 Phase 2 voice grants
  decode inline without a physical `role: voice` SDR — see the
  "One SDR for control + voice" section above. Voice grants whose
  frequency falls **outside** the wideband IQ window still spill
  over to a physical `role: voice` SDR (when configured), so a
  single backup voice dongle covers the edge cases on
  geographically spread systems. Setups with `voice_taps: 0` (or
  unset) keep the legacy behaviour where every voice grant routes
  to the physical pool. DMR Tier III additionally requires the
  system's `dmr_band_plan` (LCN → frequency) before any voice grant
  — virtual or physical — can be followed.
- **DDC-with-real-signal RX limits.** The wideband engine's per-tap
  DDC (when the SDR sample rate is higher than 48 kHz) uses the same
  Kaiser anti-alias prototype as the single-channel ccdecoder path,
  so live captures with the same SNR characteristics that lock on a
  dedicated dongle also lock here. The in-package end-to-end test
  exercises the engine wiring at the bank's native per-tap rate
  (48 kHz) where the resampler is a no-op; full validation at a
  decimating wideband rate against a TX-side filter cascade that
  mirrors the RX matched filter is a planned follow-up.
- **CPU scales with channel count.** Eight DDC taps at 2.4 MS/s is a
  few percent of one modern x86 core; the polyphase mode lands lower
  at the same count.

## Remote rtl_tcp SDRs

`rtl_tcp` ships with librtlsdr and exposes a single RTL-SDR dongle
over a TCP socket. GopherTrunk's `rtltcp` driver consumes the same
wire protocol SDR++, Gqrx, and OpenWebRX speak, so any host with a
USB-attached RTL-SDR can publish its radio to the daemon.

Typical layout:

- **Antenna host** (Raspberry Pi / Mac mini at the antenna, USB
  range from the antenna minimised): run `rtl_tcp -a 0.0.0.0 -p 1234`
  against the local dongle.
- **Daemon host** (the box with CPU + storage for decode): list the
  endpoint under `sdr.rtl_tcp` in `config.yaml`.

```yaml
sdr:
  sample_rate: 2_400_000
  rtl_tcp:
    - addr: "192.168.1.50:1234"
      serial: "antenna-pi"   # generator fills this from addr when blank
      role: control           # control | voice | auto
      ppm: 0
      gain: "auto"            # "auto" or tenths-of-dB ("496" = 49.6 dB)
      bias_tee: false
      connect_timeout_ms: 3000
```

Each entry becomes a pool device alongside any local USB dongles —
the engine roles them via the same hint matcher (`control` /
`voice` / `auto`) and the broker / fan-out path is the same as
local SDRs, so the live spectrum panel, baseband recorder, and CC
decoder all work against remote sources.

**Limitations:**

- One client per `rtl_tcp` endpoint (the upstream protocol is
  single-tuner).
- Plaintext over TCP. Keep it on a trusted network, or tunnel it
  through SSH / WireGuard / Tailscale before exposing it.
- Bias-tee + advanced rtlsdr-only knobs (direct sampling, offset
  tuning, IF gain) are wired through but rely on the remote
  running librtlsdr ≥ 0.7. Servers that ignore those commands
  silently no-op them.

**Diagnostics:** the daemon logs `rtltcp: connected addr=... tuner=...`
on each successful Open, `dial: connection refused` if the remote
isn't listening, and `header magic = "..."` if the address points
at something that isn't an `rtl_tcp` server.

## Remote SoapySDRServer SDRs

`rtl_tcp` is hardcoded to 8-bit unsigned IQ, so it throws away the
dynamic range of professional hardware. For high-bit-depth radios —
Ettus USRP, LimeSDR, bladeRF, HackRF, Airspy, RTL-SDR, SDRplay, and
anything else with a SoapySDR driver — GopherTrunk's `soapyremote`
driver speaks the [SoapyRemote](https://github.com/pothosware/SoapyRemote)
wire protocol directly, in pure Go with no CGO and no SoapySDR C
libraries. It carries 16-bit (`CS16`) or 32-bit float (`CF32`) IQ and
controls frequency, sample rate, and gain over SoapyRemote's RPC.

Typical layout:

- **Antenna host**: install SoapySDR + the device's Soapy module and
  run `SoapySDRServer --bind` against the local radio (default port
  55132).
- **Daemon host**: list the endpoint under `sdr.soapy_remote` in
  `config.yaml`.

```yaml
sdr:
  sample_rate: 2_400_000
  soapy_remote:
    - addr: "192.168.1.60:55132"  # bare host gets :55132 appended
      driver: "uhd"               # SoapySDR device key (blank = first device)
      args: ""                    # extra make() kwargs "k=v,k=v" (see below)
      master_clock_rate: 0        # USRP master clock in Hz; 0 = device default
      serial: "usrp-roof"         # generator fills this from addr when blank
      role: control               # control | voice | auto
      format: "CS16"              # CS16 (16-bit, default) or CF32 (float)
      stream_protocol: "tcp"      # tcp (default/only for now)
      stream_mtu: 0               # stream endpoint MTU in bytes; 0 = default 1500
      ppm: 0                      # best-effort (driver-dependent)
      gain: "auto"                # "auto" or tenths-of-dB ("300" = 30.0 dB)
      bias_tee: false             # best-effort (driver-dependent)
      connect_timeout_ms: 3000
```

Each entry becomes a pool device alongside any local USB dongles and
`rtl_tcp` endpoints, roled through the same hint matcher, with the same
broker / fan-out path.

### USRP B210 (UHD) — local setup

There is no native/local UHD backend — USRP support goes through the
`soapyremote` driver above, because UHD needs vendor C libraries that
are out of scope for the zero-CGO build. That path works fine on a
**single machine**: run `SoapySDRServer` locally and point GopherTrunk
at `127.0.0.1`, so the daemon still talks pure-Go SoapyRemote over a
loopback socket while UHD drives the B210.

On Ubuntu 24.04:

```sh
# 1. Install SoapySDR + the UHD module + UHD itself.
sudo apt install soapysdr-tools soapysdr-module-uhd uhd-host
sudo uhd_images_downloader           # one-time FPGA/firmware images for the B210

# 2. Confirm UHD sees the B210 through SoapySDR.
SoapySDRUtil --find="driver=uhd"     # should list your B210 by serial

# 3. Serve it on loopback (leave running; default port 55132).
SoapySDRServer --bind=127.0.0.1:55132
```

Then in `config.yaml`:

```yaml
sdr:
  sample_rate: 6_144_000            # exact ÷10 of the 61.44 MHz master clock below
  soapy_remote:
    - addr: "127.0.0.1:55132"
      driver: "uhd"
      master_clock_rate: 61_440_000 # B210: makes sample_rate an exact divisor
      role: control                 # control | voice | auto
      format: "CS16"                # 16-bit IQ (keeps the B210's dynamic range)
      gain: "auto"                  # or TENTHS of a dB — "300" = 30.0 dB, not "30"
      # args: "antenna=RX2"         # non-default port / subdev via make() kwargs
```

B210 specifics worth knowing before the first run:

- **Master clock / sample rate.** A USRP only delivers rates that are
  integer decimations of its master clock; UHD silently *coerces* a
  non-divisor request to the nearest achievable rate. GopherTrunk reads
  the delivered rate back over the RPC and clocks every down-converter
  and symbol loop from it, so a coerced rate stays decode-correct — but
  for a clean exact-divisor stream, set `master_clock_rate: 61_440_000`
  and pick a `sample_rate` that divides it (e.g. `6_144_000` = ÷10,
  `2_048_000` = ÷30). Issues #402 / #550.
- **Gain is in TENTHS of a dB** (like every device here) — `"300"` is
  30 dB, a bare `"30"` is parsed as 3.0 dB. Use `"auto"` for AGC where
  the front end supports it.
- **Antenna / subdev** on a multi-port B210 go through the free-form
  `args` string as SoapySDR `make()` kwargs (e.g.
  `args: "rx_subdev_spec=A:0,antenna=RX2"`), since there is no dedicated
  `antenna:` key.
- The SoapyRemote path is wire-protocol-complete, unit-tested against a
  mock server, and **field-tested on air against real USRP X310 and B210
  rigs** (single-branch reception; MRC diversity combining is still
  experimental pending a broader on-air A/B). Field reports from other
  SoapySDR-supported radios are welcome.

> **macOS loopback caps the stream MTU — bind to your NIC IP, not
> `127.0.0.1`.** macOS limits the MTU on the loopback interface (`lo0`),
> so a local `SoapySDRServer` bound to `127.0.0.1` can't negotiate a jumbo
> `stream_mtu` — the stream falls back to the small loopback frame size and
> a high-rate USRP can overrun. The workaround, even when the server and
> GopherTrunk run on the **same** Mac, is to bind `SoapySDRServer` to the
> host's real NIC address and point `addr:` at that same IP:
> `SoapySDRServer --bind=192.168.1.50:55132` with
> `addr: "192.168.1.50:55132"`. Linux loopback has no such cap, so
> `127.0.0.1` is fine there. This is only relevant when you raise
> `stream_mtu` above the ~1500-byte default (issue #876).

For a B210 the `args` string is where the UHD device selection lives, e.g.
`args: "type=b200,num_recv_frames=512,recv_frame_size=16384"` alongside
`master_clock_rate: 61_440_000` and `sample_rate: 6_144_000`.

### Field-tested example: networked USRP X310

The B210 recipe above runs `SoapySDRServer` on loopback, but the same driver
works over the network — the antenna host runs `SoapySDRServer` bound to the
radio's NIC and the daemon connects across the LAN. This config, contributed
from real field use of a networked X310 (issue #876), shows a full UHD
`args` string driving subdev/antenna selection, a GPSDO clock/time reference,
and the bias tee — all passed through to the remote `make()`:

```yaml
sdr:
  sample_rate: 6_250_000              # ÷32 of the X310's 200 MHz default clock
  soapy_remote:
    - addr: "10.110.162.1:23313"      # antenna host running SoapySDRServer
      driver: "uhd"
      # UHD device-selection kwargs go here (num_recv_frames/recv_frame_size
      # are fine too). But remote:mtu / remote:window are SoapyRemote STREAM
      # args — set the MTU under stream_mtu below, not in args (config load
      # rejects remote:* here).
      args: "addr=10.110.162.17,rx_subdev_spec=A:0,antenna=RX1,clock_source=gpsdo,time_source=gpsdo"
      format: "CS16"
      role: "auto"
      gain: "750"                     # 75.0 dB, set manually — see gain note below
      bias_tee: true                  # do NOT also put enable_bias_tee=1 in args (duplicate)
      stream_mtu: 8192                 # jumbo-frame link; sizes SETUP_STREAM + window
      connect_timeout_ms: 20000
```

**Per-channel antenna selection (`antenna:`).** Set the RX antenna port with the
dedicated `antenna:` list, applied in channel order via SoapySDR `setAntenna`
after the device opens — **not** with `antenna=` in `args`. An `antenna=` in the
flat args string only reaches the device `make()`, and on drivers like SoapyUHD
`antenna` is a per-channel *runtime* setting, so the make kwarg is silently
dropped and the port never changes (an X310 stays on its default RX port
regardless of what you write). GopherTrunk now rejects `antenna=` in `args` and
points you here.

Use `antenna:` for both cases — a single receiver and MRC diversity:

```yaml
  soapy_remote:
    # Single receiver (non-diversity): pin the one RX channel's port. Pair with
    # rx_subdev_spec in args to pick the daughterboard (SoapyUHD maps channel 0
    # onto it); antenna: sets that channel's port.
    - addr: "10.110.162.1:23313"
      driver: "uhd"
      args: "addr=10.110.162.17,rx_subdev_spec=A:0,clock_source=gpsdo,time_source=gpsdo"
      antenna: [RX1]                  # RX1 on the single RX channel
      format: "CS16"
      gain: "750"

    # MRC diversity: two RX channels A:0 B:0 (a space inside a value is fine —
    # args only split on commas). RX1 on channel 0, RX2 on channel 1.
    - addr: "10.110.162.1:23314"
      driver: "uhd"
      args: "addr=10.110.162.17,rx_subdev_spec=A:0 B:0,clock_source=gpsdo,time_source=gpsdo"
      diversity: mrc                  # opens RX0+RX1, phase-coherent combine
      antenna: [RX1, RX2]             # RX1→channel 0, RX2→channel 1
      format: "CS16"
      gain: "750"
```

`antenna:` accepts at most two entries; more than one requires `diversity: mrc`
(only one RX channel is opened otherwise). `antennas:` (plural) is the legacy
spelling and still works, but setting both — or putting `antenna=` in `args` — is
rejected. Pick one.

Port names are device-specific and do **not** transfer between radios: a B210
offers `TX/RX` and `RX2`, a TwinRX offers `RX1` and `RX2`. GopherTrunk checks the
requested name against the device's own `listAntennas` and fails to open naming
the ports that exist, then reads the antenna back after setting it — so the log
line reports what the radio did, not what was asked for.

**Checking that both diversity branches are alive.** Under `diversity: mrc`
GopherTrunk logs a per-branch line when the stream starts and every 30 s
afterwards:

```
soapyremote: MRC diversity branches addr=… branch_dbfs="ch0=-31.2 ch1=-33.8" \
  reference_branch=0 calibrated=true coherence=0.93 branch_gain_db=-1.8 \
  branch_phase_deg=41.2 mode=tracking updates=812 holds=3
```

Read it in this order:

- **`coherence`** is the one that matters. It is |rho| between the two branches,
  measured on the whole wideband stream, and it answers "are these two receivers
  seeing the same signal?". Roughly, |rho| = SNR/(1+SNR) for the whole span, so
  it is diluted by every kHz of noise-only bandwidth around your carrier — a
  narrow strong channel in a wide quiet span reads low even when it decodes
  perfectly. That is fine: the lock gate scales with the estimation window (the
  WARN prints the actual `lock_gate`), so the combiner calibrates whenever the
  estimate is trustworthy, not when the number looks impressive. If coherence
  sits at the gate's floor, the antennas are seeing different things (wrong band,
  wrong polarisation, far enough apart that each carrier arrives with its own
  phase — see the limitation below) — or one branch is buried under its own
  front-end noise floor: check `branch_dbfs`, and if one branch sits far below
  the other, fix that branch's gain staging (raising ITS gain genuinely helps in
  that case, because a converter floor does not scale with RF gain).
- **`branch_dbfs`** — both branches should sit within a few dB of each other. If
  one is far down the line becomes a WARN naming the dead branch: an antenna,
  connector or per-channel-gain problem on that receiver, not a decode problem.
  The second receiver is gained automatically (`gain:` is applied to every open
  RX channel), so a branch is never left at the driver default.
- **`branch_phase_deg`** tells you which kind of front end you have, with no
  capture needed. On a shared-LO single-chip radio (B210/AD9361) it is a
  constant. On two receivers with independent PLLs — an X310 with two TwinRX
  cards, `rx_subdev_spec=A:0 B:0` — it walks, because they are frequency-locked
  to a common reference but their relative phase is random at each lock. That
  walk is why `mrc` re-estimates and smooths rather than freezing a constant.
- **`holds`** climbing while `updates` stays flat means windows are being
  rejected by the coherence gate — the same story as a low `coherence`.

A WARN about receiving fewer channels than requested means the remote streamed
only one channel despite the 2-channel request: check the device really has a
second RX channel and that the `SoapySDRServer` accepted the request (its debug
log is in the diagnostics note below).

**`diversity: mrc-static`** is an escape hatch that estimates the branch gain
once and freezes it — the pre-tracking behaviour. Use it to A/B the update policy
in isolation on hardware where the phase really is constant.

**Capturing the branches for offline work (`diversity_capture`).** Every other IQ
tap in GopherTrunk sits *downstream* of the combiner, so an ordinary capture has
already had one combiner applied and cannot be replayed through a different one.
`diversity_capture` is the only tap that sees the branches before they are
combined:

```yaml
  soapy_remote:
    - addr: "10.110.162.1:23313"
      diversity: mrc
      diversity_capture: "iq/mrc/x310"   # path PREFIX
      diversity_capture_seconds: 5       # 1..60; two CS16 branches are tens of MB/s
```

It writes `x310.br0.cs16`, `x310.br1.cs16` and `x310.diversity.json` once per
stream. Each branch file is a plain headerless cs16 capture, so it plays through
`gophertrunk replay -format cs16` on its own. A datagram that did not carry both
branches is dropped from *both* files and counted in the sidecar, so sample *i*
of one file is always simultaneous with sample *i* of the other.

Replay it to compare combiners and measure the phase drift directly:

```
GT_DIVERSITY_CAPTURE=iq/mrc/x310.diversity.json \
GT_DIVERSITY_TUNE_HZ=-87500 \
  go test ./cmd/gophertrunk -run TestDiversityCombinerReplay -v
```

That prints two coherence traces and decodes eight arms, all scored by
CRC-clean BSCH count. Decode yield is the verdict; do not conclude anything from
EVM or from how the constellation looks.

**The two coherence traces are the important part.** The first is measured on
the wideband stream — what the driver's combiner actually sees. The second is
measured after each branch goes through the per-channel DDC. If narrowband
coherence is high where wideband coherence is low, the branches *do* agree on
the target channel and the single wideband gain is the limitation, exactly as
the note under Limitations predicts for widely separated antennas; no amount of
tracking will rescue it, and the `nb-*` arms are the ones to look at. If
narrowband coherence is also low, the two receivers are not seeing the same
signal at all and no combiner will help.

The arms:

| Arm | What it is |
| --- | --- |
| `branch0-only`, `branch1-only` | each receiver alone — the baseline any combiner has to beat |
| `wb-static`, `wb-tracking` | today's driver behaviour: one complex gain for the whole span, frozen or tracked |
| `wb-irc-blind` | MMSE-IRC on the wideband stream. Expected to match `wb-tracking`: without a training sequence the channel estimate is a power-weighted blend of every signal in the span, so the null is steered at a mixture. Carried so that claim is checked against real air rather than assumed |
| `nb-static`, `nb-tracking` | one gain per narrowband channel, combined **after** the DDC — the per-channel combining the wideband limitation calls for, not built into the daemon |
| `nb-branch0-only` | the narrowband baseline the `nb-*` arms have to beat |

Nothing here changes what the daemon does. These are measurements; a combiner
becomes a default only after a capture says it earns it.

Two operational notes from that deployment:

- **Prefer a manually chosen gain over AGC.** On UHD front ends, dial in the
  gain with a familiar SDR app first (e.g. SDR++), then set that value here in
  tenths of a dB — a bare integer is ALWAYS tenths, so `gain: "750"` is 75.0 dB
  and `gain: "820"` is 82.0 dB (write it as `"82.0"` if you prefer whole dB).
  `gain: "auto"` requests the device's
  AGC where it exists, but a fixed, known-good gain is more predictable for
  survey work where you want comparable signal levels run to run.
- **`SoapySDRServer` can be unstable.** Independent of GopherTrunk, the server
  has been observed to crash under load or on some devices; run it under a
  supervisor that restarts it, and see the diagnostics note below for turning on
  its debug log.

Once it is streaming, the whole toolset works against it — trunked
decode, plus the user-defined frequency-range survey in
[`hunt.md`]({{ '/hunt.html' | relative_url }}): `gophertrunk hunt
-survey -band 144:148 -band 420:470 …` sweeps the bands you name,
classifies every carrier, and streams the inventory to
`-survey-ndjson`. See that page for the SDR-selection and sweep flags.

**Limitations:**

- Receive only. One RX channel (channel 0), or channels 0 and 1 combined into
  one stream with `diversity: mrc`.
- MRC combines the **wideband** stream, so the single complex gain it estimates
  maximises SNR across the whole span rather than for one channel. That is exact
  only when the branches differ by a frequency-flat constant. Two antennas metres
  apart give each carrier its own phase difference, which one scalar cannot align
  — the symptom is a `coherence` stuck around 0.3-0.5 that no amount of tracking
  improves. Co-located, co-polarised antennas are the intended case.
- The IQ stream uses SoapyRemote's in-order **TCP** transport
  (`stream_protocol: tcp`), which opens two sockets to the server (a stream
  socket and a status socket, matching `SoapySDRServer`'s setup). UDP
  streaming with the windowed flow-control is a planned follow-up.
- `stream_mtu` sets the SoapyRemote stream endpoint MTU in bytes. It is sent
  to the server as the `remote:mtu` *stream* argument (the equivalent of
  `SoapySDR`'s `remote:mtu` knob) and used to size the client's flow-control
  window so both ends agree. Because it's a stream argument, not a `make()`
  kwarg, it cannot be expressed via `args`. Leave it `0` for SoapyRemote's
  default of 1500; raise it (e.g. `8192`) on jumbo-frame or high-throughput
  links.
- `stream_window` is worth setting explicitly on macOS servers. Left at `0`,
  GopherTrunk does not send `remote:window` and the **server** picks its own
  default socket-buffer size — which is host-dependent. `SoapySDRUtil --probe`
  reports it: one operator's Linux host defaulted to ~44 MB while their macOS
  host defaulted to **16 KiB**, and `SoapySDRServer` duly logged
  `Configured sender endpoint: … window=16 KiB`. At 1 MS/s CS16 over two
  diversity channels that is about 2 ms of buffer, so any scheduling hiccup on
  the server stalls its sender. If the probe shows a small window, set
  `stream_window` (e.g. `8388608`) rather than relying on the default.
- `args` passes extra SoapySDR device kwargs to the remote `make()` as a
  `"key=value,key2=value2"` string, merged with `driver` (an explicit
  `driver=` in `args` wins). Use it for server-side device selection and
  configuration that `driver` alone can't express — e.g. a USRP TwinRX needs
  `args: "rx_subdev_spec=A:0,antenna=RX1"`, and a UHD device can take
  `rx_subdev_spec`, `antenna`, `clock_source`/`time_source` (e.g. `gpsdo`),
  `enable_bias_tee`, and receive-buffer sizing like
  `num_recv_frames`/`recv_frame_size`. This is distinct from the top-level
  `serial`, which only names the virtual pool device locally.
  **Stream arguments do not belong here.** `remote:mtu`, `remote:window`, and
  `remote:prot` configure SoapyRemote's *stream* setup, which GopherTrunk builds
  itself — anything `remote:*` placed in `args` reaches `make()` and is silently
  dropped for streaming (you stay on the 1500-byte default). Config load now
  rejects those keys in `args` and points you at the dedicated `stream_mtu`,
  `stream_window`, and `stream_protocol` keys that actually take effect
  (issue #876).
- `ppm` (frequency correction) and `bias_tee` map to SoapySDR's
  `setFrequencyCorrection` / `writeSetting` and silently no-op on
  drivers that don't implement them.
- **USRP sample rates and the master clock.** A USRP only delivers rates that
  are integer decimations of its master clock, so UHD silently *coerces* a
  request that isn't an exact divisor to the nearest achievable rate. GopherTrunk
  reads the delivered rate back over the RPC (`getSampleRate`) and builds every
  per-channel down-converter / symbol clock from it — not from the requested
  value — so a coerced rate stays decode-correct instead of drifting the symbol
  clock (issues #402, #550). For a clean exact-divisor stream, pick a
  `sample_rate` that divides the master clock, and set `master_clock_rate` when
  the default clock doesn't divide your target: an X310's 200 MHz default already
  divides `6_250_000` (÷32), while a B210 needs `master_clock_rate: 61_440_000`
  to stream `6_144_000` (÷10) exactly. `master_clock_rate` is a convenience for
  putting `master_clock_rate=…` in `args`; an explicit value in `args` wins.
- `gain` is applied as a manual overall gain. AGC is disabled first on a
  best-effort basis, so a numeric gain still applies on front-ends that have
  no AGC at all (e.g. a USRP TwinRX, which rejects `set_rx_agc()`); use
  `gain: "auto"` to request AGC where the radio supports it.
- Plaintext over TCP. Keep it on a trusted network, or tunnel it
  through SSH / WireGuard / Tailscale.
- The RPC, stream framing, and TCP stream-setup choreography are byte-matched
  to SoapyRemote's source and exercised by the driver's tests; validate
  against a live `SoapySDRServer` before production use.

**Diagnostics:** the daemon logs `soapyremote: connected addr=...
format=... proto=...` on each successful Open, `make device:` with the
remote exception text when the server can't open the requested device,
and `dial: connection refused` if the server isn't listening.

On the server side, start `SoapySDRServer` with SoapySDR's debug log enabled to
see what UHD is doing with the device and stream setup:

```sh
SOAPY_SDR_LOG_LEVEL=DEBUG SoapySDRServer --bind=10.110.162.17:23313
```

(bind to the radio host's LAN address for a networked SDR, or `127.0.0.1` for
the loopback B210 recipe above). This is the first place to look when a device
opens but never streams, or when the server crashes on a client connect.

> **Note — upstream server crashes.** Some radios were observed crashing
> `SoapySDRServer` itself (a `zsh: segmentation fault` inside `libuhd`) when the
> client mis-spoke the stream protocol. A server should never crash on a client
> RPC, so that is an upstream UHD / `SoapySDRServer` robustness bug. GopherTrunk
> no longer provokes it now that the TCP `SETUP_STREAM` handshake and stream
> flow-control ACKs are byte-matched to SoapyRemote (issue #542). If you can
> still reproduce a server crash, please report it upstream at
> <https://github.com/pothosware/SoapyRemote/issues> with the server-side log.

### Tracing the RPC conversation from GopherTrunk's side

The SoapyRemote wire carries no schema: a message is a call id followed by bare
tagged values, and the server dispatches on the id alone. When it logs

```text
[ERROR] ~SoapyRPCUnpacker: Unconsumed payload bytes 9
```

it is saying "the handler I dispatched to wanted fewer arguments than arrived" —
and it does not say which call, so from the server log alone you are guessing.
That guessing is how `SET_ANTENNA` sat on opcode 600 (`HAS_DC_OFFSET_MODE`)
through a release: the server logged unconsumed bytes, replied with a bool, and
the bool passed as success while no antenna was ever set.

`verbose_debug` on a `soapy_remote` source logs the other half of the
conversation — every request and response, decoded, plus the raw frame:

```yaml
log:
  level: debug              # the trace is emitted at DEBUG
sdr:
  soapy_remote:
    - addr: 10.110.162.1:23313
      verbose_debug: true   # diagnostic only; leave off for normal operation
```

```text
level=DEBUG msg="soapyremote: rpc >" addr=10.110.162.1:23313 seq=7
  call=SET_ANTENNA(501) args="call SET_ANTENNA(501), char 0x01, i32 0, str \"RX1\""
  bytes=41 hex="53 52 50 43 ..."
level=DEBUG msg="soapyremote: rpc <" addr=10.110.162.1:23313 seq=7
  call=SET_ANTENNA(501) args="void" bytes=1 hex="0e"
```

The decoded `args` are the useful part: line them up against the matching
handler in upstream's `common/ClientHandler.cpp` and an argument-shape or
opcode mismatch is visible directly. The `seq` pairs a request with its reply.
It is per endpoint, so a multi-radio config can trace one server at a time.

## USB disconnect recovery

A dongle that physically disconnects from the USB bus mid-run (flaky
cable, marginal hub power, EMI burst, a brief brown-out on a laptop
running on battery) no longer takes the daemon down: the disconnect is
recoverable in-process when the device re-enumerates under the same
serial. Three independent paths cover the three states a device can
be in when the disconnect lands.

### 1. Control SDR — in-stream IQ death

The ccdecoder's `Decoder.Run` surfaces the stream death as
`ErrIQStreamClosed`. The retry loop backs off, calls `Pool.Reacquire`,
swaps the fresh `Device` into `ccDecoderOpts.IQ`/`Tuner`, and tells
the cchunt supervisor to swap its tuner via `SwapTuner` so the next
hunt round picks up the new handle:

```text
WARN  daemon: ccdecoder: IQ stream died; retrying attempt=1 max_attempts=4 backoff=1s
       err="ccdecoder: IQ stream closed unexpectedly: rtl2832u: write block=1 ... usb: device disconnected"
INFO  daemon: ccdecoder: control SDR reacquired serial=76361606
INFO  sdr: reacquired driver=rtlsdr serial=76361606 role=control old_index=0 new_index=1
INFO  cchunt: control tuner swapped (reacquired)
```

Bounded by the existing 1 s / 2 s / 5 s / 10 s retry budget and the
60 s "healthy run" window that resets the attempt counter.

### 2. Voice SDR — stale at next call

A voice dongle that disconnected while idle leaves the trunking
engine holding a stale `Tuner` handle. The next time the engine
calls `VoicePool.Bind`, `SetCenterFreq` fails. The pool's reacquire
hook (wired by the daemon to `sdr.Pool.Reacquire`) opens a fresh
handle for the same serial, swaps it into the `VoiceDevice`, and
retries the tune once before the call drops:

```text
INFO  sdr: reacquired driver=rtlsdr serial=00000002 role=voice old_index=1 new_index=2
```

If the reacquire fails (device truly gone), Bind returns the
original `SetCenterFreq` error joined with the reacquire failure so
the operator gets the full story. The call drops and the next grant
on that talkgroup will retry.

### 3. Periodic watchdog — idle devices

A background watchdog ticks every `sdr.watchdog_interval_ms` (default
30 s, opt-out via `-1`), re-enumerates every registered driver, and
acts on serial-level state transitions:

- A serial the pool expects but the enumerate doesn't see transitions
  to "missing"; one `KindSDRDetached` event surfaces the gap so the
  API / TUI / web snapshot reflect it.
- A serial that was missing in the previous tick and is now back
  triggers `Pool.Reacquire` so the next consumer (a voice call or a
  ccdecoder retry) touches a live handle instead of paying the
  reacquire round-trip mid-use.

```text
WARN  sdr: watchdog: device missing from USB enumerate serial=76361606
INFO  sdr: watchdog: device reappeared; reacquiring serial=76361606
INFO  sdr: reacquired driver=rtlsdr serial=76361606 role=control old_index=0 new_index=1
```

The watchdog never reacquires a device that's been continuously
present (no spurious churn on healthy hardware) and never proactively
closes an in-use stream (the IQ-death path owns the in-use case).

### Common contract

What happens inside every `Pool.Reacquire` call, regardless of the
caller:

1. The original `Device` handle is `Close`d best-effort — a dead
   handle's Close return is logged and ignored.
2. The driver's `Enumerate()` re-scans the USB bus and finds the
   matching serial at its new index.
3. `Driver.Open(idx)` returns a fresh `Device` against the
   re-enumerated USB transport.
4. The configured sample rate (`sdr.sample_rate` in YAML) and the
   original hint state (PPM, gain, bias-tee) are re-applied to the
   fresh handle — the operator's tuning survives the disconnect.
5. The new `Device` is swapped into the `PoolEntry` in place, so any
   later `Pool.FindBySerial` / API snapshot returns the live handle.
6. `KindSDRDetached` then `KindSDRAttached` are published so the API
   (`GET /api/v1/devices`), the TUI device line, and the web console
   reflect the gap and the recovery.

### Unrecoverable cases

If the device stays gone after re-enumerate, or if `Driver.Open()`
fails (kernel hasn't finished re-binding the USB descriptor, a
permissions race, the dongle is genuinely dead), the in-stream paths
exhaust their retry budget and the daemon escalates to a clean fatal
exit. A process supervisor (`systemd`, `docker`, the GopherTrunk
launcher's `-headless` mode) then restarts the daemon, which
re-discovers the SDR by serial on next `pool.Open()`. The watchdog
keeps logging "missing from USB enumerate" until the device returns
but never escalates on its own — it's the safety net, not the
authority.

### Operator note

If you see these log shapes on a hardware unit, the daemon is doing
the right thing — but the cause is almost always physical: a
marginal USB-A cable, an unpowered hub, EMI from a nearby switching
supply, or a thermal trip on a poorly-ventilated dongle. The on-board
recovery keeps the stream live across one or two events per hour, but
it isn't a substitute for fixing the underlying USB-link instability.

## Capturing IQ for replay

GopherTrunk has two replay paths.

**Wideband baseband replay** — record the full IQ stream of any
attached tuner with a `baseband.record` config entry (see
[`opt-in-features.md`]({{ '/opt-in-features.html' | relative_url }})),
then mount the resulting two-channel 16-bit WAV (or one of
SDRtrunk's baseband recordings — same layout) as a virtual tuner via
`baseband.replay`. The replay loops on EOF, so a short capture
becomes a continuous source.

**Raw cfile mock driver** — the legacy mock driver replays raw u8-IQ
files (`.cfile` format) generated with `gqrx`, `csdr`, or any tool
that produces interleaved unsigned-8-bit samples. Drop `.cfile`
files under `testdata/iq/` to use them through the mock driver. The
baseband replay path above is the preferred option for new work;
the cfile mock stays around for the existing integration tests.

**P25 Phase 1 offline decoder** — `gophertrunk replay` runs a raw
IQ capture through the production receiver + control-channel chain
and prints every lock / grant / decode-error event the daemon would
emit, plus the per-frame NID-decoder diagnostics. Reuses the real
`internal/radio/p25/phase1/receiver` and `phase1.ControlChannel`,
so what it decodes is what the daemon decodes — a replay-lock
implies an on-air lock, and a replay-fail makes the capture a
reproducible test fixture for the protocol path. Built for
[issue #275](https://github.com/MattCheramie/GopherTrunk/issues/275)
investigations where on-site retests round-trip too slowly.

```
gophertrunk replay -in capture.iq -sample-rate 960000 -demod c4fm
gophertrunk replay -in cbd.cfile -format f32 -sample-rate 960000 -demod cqpsk
```

Flags:

- `-format u8|f32` — `u8` is the rtl_sdr default (interleaved 8-bit
  unsigned IQ); `f32` is GNU Radio's interleaved-float32 cfile.
- `-sample-rate Hz` — the capture's sample rate. The decoder prints
  the effective baud at EOF; if it deviates >2% from 4800 the
  capture's true sample rate doesn't match this flag.
- `-demod c4fm|cqpsk` — modulation. C4FM restricts the FSW + NID
  rotation search to physically-meaningful {0,2} rotations; CQPSK
  / π/4-DQPSK keeps the all-rotation default.
- `-freq Hz` — informational; reported alongside the events.
- `-nid-search-span N` — NID-alignment grid radius in dibits
  (default 6, matching the production ccdecoder). Widen to 12 / 18
  / 36 on a stubborn capture to bisect a span-bounded failure (errs
  drop at the new optimum) from a demod-quality-bounded one (errs
  stay at the BCH(63,16,11) correction ceiling regardless of
  alignment).
- `-diag` — after EOF, dump the dibit-value histogram, the
  pre-slicer soft-sample magnitude distribution, the FSW
  correlation landscape per rotation (Hamming-distance histogram +
  hit count), the FSW positions and inter-hit deltas, the raw
  NID + first 24 TSBK dibits at the first 5 perfect-distance
  FSWs, and the trellis-decoded 12-byte TSBK info block with its
  augmented-CRC check (a clean TSBK shows `crc=0x0000`). The
  off-path diagnostic that pinpoints which stage of the C4FM
  demod chain produces wrong dibits — see the
  `cmd/gophertrunk/iqdiag.go` package comment for the failure-mode
  interpretation table.
