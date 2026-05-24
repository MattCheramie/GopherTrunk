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
      gain: "auto"             # or a numeric tenths-of-dB string like "496"
      bias_tee: true           # 5V on the SMA — only enable if you want it
```

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
