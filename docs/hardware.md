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
| **RTL-SDR** (RTL2832U + R820T / R820T2 / R828D / E4000 / FC0012 / FC0013 / FC2580) | `rtlsdr` | `0x0bda:0x2832` · `0x0bda:0x2838` | Production — on-air-validated across Linux / macOS / Windows. |
| **HackRF One / Jawbreaker / Rad1o** | `hackrf` | `0x1d50:0x6089` · `0x1d50:0x604b` · `0x1d50:0xcc15` | Wire-protocol-complete; on-air validation against attached hardware is the documented follow-up. |
| **Airspy R2 / Airspy Mini** | `airspy` | `0x1d50:0x60a1` | Wire-protocol-complete; on-air validation against attached hardware is the documented follow-up. |
| **Airspy HF+ Discovery / HF+ Dual Port / legacy HF+** | `airspyhf` | `0x03eb:0x800c` | Wire-protocol-complete; HF (9 kHz – 31 MHz) + VHF (60 – 260 MHz). On-air validation against attached hardware is the documented follow-up. |
| **rtl_tcp remote** (any librtlsdr-shipped server) | `rtltcp` | TCP | Remote RTL-SDR mounted over the network. See [Remote rtl_tcp SDRs](#remote-rtl_tcp-sdrs). |

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
`SetSampleRate` then picks the closest available rate, so a `gain:
"auto"` + `sample_rate: 10_000_000` config picks the 10 MSPS slot on
R2 and the 6 MSPS slot on Mini without per-device overrides. The
HF+ driver does the same; it also reads VERSION_STRING_READ to
expose the firmware version in `TunerName`.

```yaml
sdr:
  sample_rate: 768_000             # HF+ Discovery max
  devices:
    - serial: "3652b46d6e6f8867"
      role: control
      gain: "auto"                 # firmware HF AGC handles the HF band well
      bias_tee: false              # set true to power an active HF antenna
```

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
    - name: "neighbour-dmr-t2"
      protocol: dmr-tier2              # Tier II conventional
      control_channels: [852_125_000]
```

The engine picks the right state machine per channel (Tier III's
`ControlChannel` for `protocol: dmr`, Tier II's `ConventionalChannel`
for `protocol: dmr-tier2`).

### P25 trunked control channel on a wideband dongle

A wideband dongle can also host a P25 control channel — Phase 1 (C4FM
or CQPSK / LSM simulcast) or Phase 2 (H-DQPSK TDMA). The configuration
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
      # On a simulcast site whose CC is transmitted as Linear
      # Simulcast Modulation rather than C4FM, opt in to the
      # linear-CQPSK demod path:
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
  CQPSK / LSM simulcast), and `protocol: p25-phase2` (P25 Phase 2
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
  to the physical pool.
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
