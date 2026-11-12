---
layout: page
title: "Design: Native ADALM-Pluto / PlutoSky driver (iiod)"
description: A pure-Go driver for AD936x boards that talks to the board's own libiio daemon over TCP, with no SoapySDR or libiio on the host
nav_group: Reference
---

# Design: Native ADALM-Pluto / PlutoSky driver (iiod)

**Status:** implemented in `internal/sdr/iio` (`sdr.iio` config). Reference-
pinned and synthetic-tested; **not yet confirmed on air** (#764/#771).
**Issue:** #1329.
**Scope:** receive only, one RX channel, Pluto-compatible AD936x boards
(ADALM-Pluto, PlutoSky, and other boards that run Analog Devices' Pluto
firmware).

## Problem

GopherTrunk reaches a Pluto today only through `sdr.soapy_remote`, which needs a
host running `SoapySDRServer` with the SoapyPlutoSDR module. That module has to
be built against the host's SoapySDR, libiio and libad9361. The #1329 reporter
cannot build it (pothosware/SoapyPlutoSDR#82), so their PlutoSky cannot reach
GopherTrunk at all, though GopherTrunk handles the board's samples correctly
(#1331 fixed the 12-bit full scale).

The C stack is not actually needed. A Pluto runs **`iiod`**, libiio's network
daemon, on the board itself and serves it on TCP over its Ethernet or
USB-network link (default `192.168.2.1`). The protocol is a short line-based
text protocol. A pure-Go client for it gives GopherTrunk a Pluto driver with
nothing installed on the host, in line with the other pure-Go drivers
(`docs/hardware.md`).

## Protocol facts (pinned to libiio v0.25 source)

These are read from the server and client source, not from memory. The
references are to `analogdevicesinc/libiio` tag `v0.25`.

- **Port 30431** (`dns_sd.h:35`, `#define IIOD_PORT 30431`).
- **Commands are text lines ending in `\r\n`** (the client's
  `iio_snprintf(... "READBUF %s %lu\r\n" ...)`, `network.c:744`). The full set
  is in the server's `HELP` text (`iiod/parser.y`, the `HELP END` rule). This
  driver needs:
  - `VERSION`: replies `major.minor.gittag\n` (`"%u.%u.%-7.7s\n"`).
  - `PRINT`: replies the XML context's length as a decimal line, then the XML,
    then `\n`. The XML lists every device, channel and attribute, including each
    channel's sample format.
  - `READ <dev> INPUT|OUTPUT <chn> <attr>` and `READ <dev> <attr>` (formats in
    `iiod-client.c`, `iiod_client_read_attr`).
  - `WRITE <dev> INPUT|OUTPUT <chn> <attr> <bytes_count>`, followed by exactly
    `bytes_count` bytes of value.
  - `OPEN <dev> <samples_count> <mask>`, `READBUF <dev> <bytes_count>`,
    `CLOSE <dev>`, `TIMEOUT <ms>`, `EXIT`.
- **Replies start with a decimal integer line**: the byte count of what follows,
  or a negative errno (`print_value`, `iiod/ops.c:366`). The client skips leading
  newlines while parsing it (`read_integer`, `network.c:499`). An attribute read
  is that length, then the value, then a trailing `\n`
  (`iiod_client_read_attr`).
- **`READBUF` streams chunks.** Each chunk is a decimal byte-count line, then
  that many raw sample bytes (`send_data`, `iiod/ops.c:439`). On a client's
  **first** chunk the server sends the channel mask (8 hex digits per 32-bit
  word, then `\n`) between the count and the data (`thd->new_client`). A zero
  count ends the transfer. The v0.25 client tries to read a mask on each new
  `READBUF` call (`network_read_mask`, `network.c:528`). **Settled:** the two
  agree. The server sets `thd->new_client = true` in `rw_buffer` on **every**
  `READBUF` (`iiod/ops.c`, just before signalling the reader thread), and
  `send_data` clears it after that call's first chunk; the client
  (`iiod_client_read_unlocked`) reads the mask once per call. So the mask rides
  the first chunk of each `READBUF`, not the first chunk per client. A
  `READBUF` that is fully satisfied ends with no terminator; one the buffer
  could not fill ends with a `0` count (`rw_buffer`: `ret > 0 && ret < nb`).
- **Written values carry their NUL.** `iio_channel_attr_write` sends
  `strlen(src) + 1` bytes (`channel.c`), so `WRITE … frequency 10` is followed
  by `851012500\0`. The driver does the same; the tests assert it literally.
- **The mask bit is libiio's channel number**, not the scan index directly:
  `reorder_channels` (`context.c`) sorts scan elements by index (ties by
  shift) and numbers the device's channels in that order; the mask has
  `(nb_channels + 31) / 32` words over all channels, most significant first.
- **Devices are addressed by name or id.** The server's lexer looks the device
  word up with `iio_context_find_device` (`iiod/lexer.l:96-98`), which matches
  an id (`iio:device1`) or a name (`ad9361-phy`). The driver still resolves names
  from the `PRINT` XML, so a board that numbers its devices differently works.

## AD9361 attributes (pinned to SoapyPlutoSDR)

SoapyPlutoSDR drives the same board through libiio, so its calls are the
reference (`pothosware/SoapyPlutoSDR`, `PlutoSDR_Settings.cpp` /
`PlutoSDR_Streaming.cpp`):

| Purpose | Device | Channel | Attribute |
|---|---|---|---|
| RX LO frequency (Hz) | `ad9361-phy` | `altvoltage0` (output) | `frequency` |
| RX LO power | `ad9361-phy` | `altvoltage0` (output) | `powerdown` (write 0 to turn on) |
| RX sample rate | `ad9361-phy` | `voltage0` (input) | `sampling_frequency` |
| Stream-side sample rate | `cf-ad9361-lpc` | `voltage0` (input) | `sampling_frequency` |
| RX analog bandwidth | `ad9361-phy` | `voltage0` (input) | `rf_bandwidth` |
| Gain mode | `ad9361-phy` | `voltage0` (input) | `gain_control_mode`: `manual` / `slow_attack` |
| Manual gain (dB) | `ad9361-phy` | `voltage0` (input) | `hardwaregain` |
| RX port | `ad9361-phy` | `voltage0` (input) | `rf_port_select` |

- **Samples** come from `cf-ad9361-lpc`, channels `voltage0` (I) and `voltage1`
  (Q), as **12-bit values, LSB-aligned in 16 bits**: SoapyPlutoSDR reports RX
  full scale **2048** (`PlutoSDR_Streaming.cpp`, `getNativeStreamFormat`), the
  value #1331 made GopherTrunk honour. The driver reads each channel's format
  from the `PRINT` XML and refuses to stream if it isn't the expected 12-bit
  little-endian form, rather than assuming it.
- **Sample-rate floor:** below 25 MHz / 12 (≈2.083 MS/s) the AD9361 needs ×8
  decimation, so SoapyPlutoSDR sets the PHY to 8× the rate and the stream device
  to the rate itself. Without a FIR loaded the minimum is 25 MHz / 96
  (≈260 kS/s) (`PlutoSDR_Settings.cpp`, the RX branch of `setSampleRate` and
  `getSampleRateRange`). The driver implements that ×8 path (it is two
  attribute writes), so rates from ≈260.4 kS/s to 61.44 MS/s work; config
  validation rejects a lower `sdr.sample_rate`. FIR loading (libad9361's
  `ad9361_set_bb_rate`) is out of scope.

## Driver shape

- **Package** `internal/sdr/iio` (or `internal/sdr/pluto`):
  - `client.go`: the iiod protocol (connect, `VERSION`, `PRINT`,
    attribute read/write, `OPEN`/`READBUF`/`CLOSE`), with no AD9361 knowledge;
  - `device.go`: an `sdr.Device` mapping `SetCenterFreq` / `SetSampleRate` /
    `SetGain` onto the attributes above, and `StreamIQ` onto `READBUF` chunks
    converted to `complex64` (÷2048).
- **Streaming:** one goroutine reads `READBUF` chunks into a **time-sized**
  channel (400 ms at the stream rate, the `streamBufferLatency` rule every other
  driver now follows; the #1242 and #1321 depth-8 lesson) and drops the newest
  chunk on overrun through `sdr.NotifyIQDrop`. `StreamIQ` must end promptly on
  ctx cancel, not park in a read behind a long deadline (the shutdown-stall
  lesson in CLAUDE.md).
- **Gain:** `SetGain(-1)` selects `slow_attack`; otherwise `manual` plus
  `hardwaregain` in whole dB (tenths rounded).
- **`SetPPM`:** the AD9361 has no ppm trim through these attributes, so return
  nil and let the existing carrier-offset estimator report the error, as with
  any SDR. (`xo_correction` on `ad9361-phy` exists but is board-specific; leave
  it out of v1.)

### Config

A new list beside `rtl_tcp` / `soapy_remote` / `ka9q_radio`:

```yaml
sdr:
  sample_rate: 2_400_000
  iio:
    - addr: "192.168.2.1"     # the board; port 30431 when omitted
      serial: "plutosky"
      role: control
      gain: "400"             # tenths of a dB, or "auto"
      rf_bandwidth_hz: 0      # 0 = follow sample_rate
```

What the config change touches (CLAUDE.md's checklist):
- `config.Config` and its validator;
- `configbuilder/fieldmeta.go` (`TestFieldHelpCoverage`) and `sections.go`;
- `web/configbuilder/src/api/types.ts` (`TestConfigSchemaCoveredByWebBuilder`);
- `config.example.yaml` (the 4 Sep rule);
- **the pool-construction gate in `daemon.go`**, which must name the new list
  or a Pluto-only config registers no driver. This is the
  `TestRemoteOnlySDRConfigsRegisterTheirDriver` lesson; extend that test.

## Testing plan

The SoapyRemote opcode bug (CLAUDE.md) is the warning: a fake server that
shares the driver's constants passes whatever the driver sends. So:

1. **Literal command bytes.** The fake iiod server asserts the exact lines it
   receives (`READ ad9361-phy OUTPUT altvoltage0 frequency\r\n`, and so on),
   written out as literals in the test, not built from the driver's own
   formatters.
2. **Literal replies from a real board.** A recorded `PRINT` XML and attribute
   replies from a real Pluto (the reporter's `iio_info` output, or a capture of
   iiod traffic) are committed as test data, and the driver must parse them.
3. **Framing edge cases.** The fake server splits a `READBUF` chunk across TCP
   reads, sends the mask only on the first chunk, returns negative errnos, and
   leaves leading newlines before an integer, as `read_integer` tolerates.
4. **Scale.** A known 12-bit pattern converts to the expected `complex64`
   values, and the level reads the same as the SoapyRemote path at full scale
   2048.
5. **Teardown.** Cancelling `StreamIQ` mid-chunk returns within a bound, with
   no goroutine leak.

**On-air gate (#764/#771):** the reporter's PlutoSky. Until it runs, the driver
is "reference-pinned, not verified".

## What landed vs. the plan

- Package `internal/sdr/iio`: `client.go` (protocol), `context.go` (PRINT XML,
  channel numbering, masks), `driver.go` (the `sdr.Device`). One deviation from
  the plan: no ppm hint is applied; a non-zero one is logged once.
- Test plan items 1, 3, 4 and 5 are covered (`driver_test.go`,
  `context_test.go`, a fake iiod that records raw bytes). Item 2 landed on
  6 Nov 2026: `testdata/plutosky_rev_c_context.xml` is the reporter's
  PlutoSky's real `PRINT` output (Pluto v0.38 firmware, iiod 0.25), and
  `realboard_test.go` checks every attribute the driver reads or writes
  against it. The reconstructed listing it replaced numbered the devices
  differently (phy `iio:device1` / rx `iio:device4`; the board has
  `iio:device0` / `iio:device3`); the driver binds by name, so nothing in it
  depended on the numbers, but the fake iiod's attribute keys did.
- Config, pool gate (`TestRemoteOnlySDRConfigsRegisterTheirDriver`), config
  builder, web builder, `config.example.yaml` and `docs/hardware.md` are wired.

## What the reporter can check now

`gophertrunk iio probe -addr 192.168.2.1 -out board.xml` (after #1342) does
both checks below in one read-only step and saves the `PRINT` XML. The
reporter ran it on 6–8 Nov 2026 from Linux Mint, Debian 11 without libiio
and Termux over WLAN: version, device list, settings and a 65536-sample read
at −33 to −35 dBFS RMS with no sample at the rail, over the USB link and the
LAN. `gophertrunk power -iio` / `capture -iio` then take the board directly.

Two checks need only the board, not SoapyPlutoSDR, and they settle open points
above:

- `iio_info -u ip:192.168.2.1` (libiio's own tool, if installed). It prints the
  device and channel names and each channel's sample format.
- Without libiio: `nc 192.168.2.1 30431`, then type `VERSION` and `PRINT`. The
  first line confirms iiod is reachable and its version; the XML is what the
  driver will parse.

## Out of scope

- TX, the second RX channel (2R2T boards), FIR loading, and the libiio v1
  binary protocol. The ASCII protocol is what v0.x clients speak. Whether a v1
  `iiod` still accepts it should be confirmed against the reporter's firmware
  `VERSION` before relying on it.
- USB-only access without the board's USB-network link. Pluto firmware exposes
  iiod over that link, so TCP covers USB-connected boards too.
