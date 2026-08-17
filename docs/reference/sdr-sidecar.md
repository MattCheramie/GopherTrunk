---
title: "Sidecar SDR driver"
layout: default
---

# Sidecar SDR driver

A **sidecar** is any external process that owns a radio and streams raw IQ to
GopherTrunk. GopherTrunk mounts it as an ordinary tuner: it appears in the SDR
pool, takes tuning commands, and feeds the decoders like any other device.

The point is the boundary. GopherTrunk is pure Go with `CGO_ENABLED=0`, which
rules out linking `libuhd`, `libusb`, vendor SDKs, or a VOLK-accelerated DSP
chain directly into the daemon. A sidecar puts all of that in someone else's
process, connected by a pipe:

```
  +-------------------------+                    +---------------------------+
  |      GopherTrunk        |  5-byte commands   |     Your sidecar          |
  |  trunking + decoders    | -----------------> |  libuhd / GNU Radio /     |
  |                         |                    |  vendor SDK / custom DSP  |
  |   internal/sdr/sidecar  | <----------------- |                           |
  +-------------------------+   raw IQ stream    +---------------------------+
```

Anything the sidecar does internally is invisible here — including combining
several antennas into the one stream it hands over.

## Configuration

```yaml
sdr:
  sidecar:
    - transport: "tcp"                 # unix_pipe | tcp | udp (default tcp)
      data_addr: "127.0.0.1:5001"      # FIFO path, or host:port
      control_addr: "127.0.0.1:4001"   # the sidecar's UDP command socket
      format: "cs16"                   # cs16 (default) | complex64
      sample_rate_hz: 2400000          # REQUIRED
      freq_min_hz: 70000000            # tuning range, for the hunt sweep
      freq_max_hz: 6000000000
      serial: "uhd-b210-sidecar"
      role: "auto"
      gain: "auto"
      connect_timeout_ms: 3000
```

`sample_rate_hz` is required and there is no way around it: the stream carries
no metadata, so GopherTrunk cannot discover the rate, and every downstream
filter is sized from it. A wrong value mis-tunes every channel.

Leaving `control_addr` empty is legal and means the sidecar owns tuning —
GopherTrunk's frequency, rate and gain setters become no-ops. That is fine for
a fixed-frequency feed, but a trunked system cannot follow voice grants on such
a device, and the daemon says so once at startup.

## Data stream (sidecar → GopherTrunk)

A bare stream of interleaved complex samples. No framing, no header, no
sequence numbers — byte-for-byte what `gophertrunk replay -format cs16` reads,
so an existing capture file can stand in for a sidecar while you develop.

| `format` | Layout | Bytes/sample |
| --- | --- | --- |
| `cs16` | `int16` I, `int16` Q, little-endian | 4 |
| `complex64` | `float32` I, `float32` Q, little-endian | 8 |

`cs16` is the default and halves the bytes on the wire; `complex64` is the
native Go layout and skips a conversion, which matters only for a local pipe at
a high rate.

Transports:

- **`unix_pipe`** — `data_addr` is a FIFO path. GopherTrunk opens it for
  reading, which blocks until the sidecar attaches as a writer, so
  `connect_timeout_ms` bounds daemon startup against a sidecar that never runs.
  POSIX only.
- **`tcp`** — GopherTrunk dials `data_addr`. The sidecar listens.
- **`udp`** — GopherTrunk binds `data_addr` and the sidecar sends datagrams to
  it. **Every datagram must contain a whole number of samples.** A UDP read
  returns exactly one datagram and discards any excess, so a sample split across
  two datagrams cannot be reassembled; GopherTrunk drops and counts a
  misaligned datagram rather than splicing two packets together.

Back-pressure: GopherTrunk never blocks its reader. If the decode chain falls
behind, the oldest queued chunk is dropped and counted, and a throttled WARN
reports `host_drops`. Blocking the reader instead would stall the socket and
push the overflow back into the sidecar's own buffers, turning one local hiccup
into a stream-wide glitch.

## Control channel (GopherTrunk → sidecar)

One UDP datagram of exactly **5 bytes**:

```
  byte 0      bytes 1..4
+--------+------------------+
| opcode | parameter (BE32) |
+--------+------------------+
```

The parameter is a **big-endian uint32**. The opcodes are `rtl_tcp`'s, not new
ones, so a tool that already speaks `rtl_tcp`'s command protocol works as a
sidecar unmodified:

| Opcode | Name | Parameter |
| --- | --- | --- |
| `0x01` | `SET_CENTER_FREQ` | Hz |
| `0x02` | `SET_SAMPLE_RATE` | Hz |
| `0x03` | `SET_GAIN_MODE` | `1` = manual gain, `0` = the sidecar's own AGC |
| `0x04` | `SET_GAIN` | gain in **tenths of a dB** (`496` = 49.6 dB) |

Setting a manual gain sends `SET_GAIN_MODE 1` followed by `SET_GAIN`. Selecting
AGC sends `SET_GAIN_MODE 0` and no `SET_GAIN` — GopherTrunk's internal AGC
sentinel is a negative gain, and sending that as an unsigned parameter would
arrive as a manual gain of four billion tenths of a dB.

Commands are fire-and-forget: UDP, no reply, no acknowledgement. Keep the
sidecar on loopback or a LAN.

`SetPPM` and `SetBiasTee` have no opcode. The sidecar owns the front end, so its
clock correction and bias-tee are its own business; GopherTrunk accepts those
calls and does nothing.

## Minimal example

A sidecar that replays a capture file over TCP, ignoring tuning — enough to
check the plumbing end to end:

```sh
# terminal 1: the "sidecar"
nc -l 5001 < capture.cs16

# terminal 2: gophertrunk, with
#   sdr.sidecar: [{data_addr: "127.0.0.1:5001", sample_rate_hz: 2400000}]
gophertrunk -config sidecar.yaml
```

## Limitations

- Receive only.
- One stream per endpoint. A sidecar that wants to combine several antennas
  does that internally and hands over one stream.
- No sample-rate or frequency read-back: GopherTrunk reports what you
  configured, because the stream says nothing about itself.
- Plaintext, like `rtl_tcp` and `soapy_remote`. Trusted networks or a tunnel.
