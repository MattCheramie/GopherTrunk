---
slug: librtlsdr
title: librtlsdr
entry_type: technology
category: sdr-frameworks
description: "librtlsdr is the Osmocom host library that turns RTL2832U TV dongles into SDR receivers, delivering raw IQ through an asynchronous read-callback API."
keywords: librtlsdr, rtl-sdr driver, Osmocom, rtl_sdr, RTL2832U, USB SDR library, rtlsdr_read_async, rtl_tcp, libusb
aka: [librtlsdr, rtl-sdr library, Osmocom rtl-sdr]
autolink: true
infobox:
  - { label: Type, value: SDR host driver library }
  - { label: Idea, value: Repurpose RTL2832U DVB-T chips as raw IQ receivers }
  - { label: API, value: Async read-callback + synchronous read }
see_also: [rtl-sdr, rtl2832u, gr-osmosdr, callback-vs-stream-api, rtl-tcp, software-defined-radio]
cite_urls:
  - https://osmocom.org/projects/rtl-sdr/wiki
  - https://github.com/osmocom/rtl-sdr
---

**librtlsdr** is the open-source [Osmocom](/reference/gr-osmosdr/) host library that
drives Realtek [RTL2832U](/reference/rtl2832u/) demodulator chips as software-defined
radios, streaming raw 8-bit [IQ](/reference/iq-data/) samples to an application over
USB.[^osmo] It is the C library at the heart of the entire
[RTL-SDR](/reference/rtl-sdr/) phenomenon: the code that discovered the chip's
undocumented "raw sampling" mode and exposed it as a general-purpose receiver API.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An application registers a callback with librtlsdr, which reads IQ over libusb from the RTL2832U dongle and invokes the callback repeatedly with filled sample buffers." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rtlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="20" width="120" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="80" y="39" text-anchor="middle" font-size="9" fill="currentColor">application code</text>
  <rect x="20" y="90" width="120" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/><text x="80" y="105" text-anchor="middle" font-size="9" fill="currentColor">librtlsdr</text>
  <text x="80" y="116" text-anchor="middle" font-size="7" fill="currentColor">(libusb transfers)</text>
  <rect x="200" y="55" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="255" y="74" text-anchor="middle" font-size="9" fill="currentColor">RTL2832U + tuner</text>
  <rect x="360" y="55" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="400" y="74" text-anchor="middle" font-size="9" fill="currentColor">antenna</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="80" y1="50" x2="80" y2="90" marker-end="url(#rtlar)"/>
    <text x="98" y="72" font-size="7" fill="currentColor">register callback</text>
    <line x1="80" y1="90" x2="80" y2="52" marker-end="url(#rtlar)"/>
    <line x1="140" y1="105" x2="200" y2="78" marker-end="url(#rtlar)"/>
    <line x1="360" y1="70" x2="310" y2="70" marker-end="url(#rtlar)"/>
  </g>
  <text x="175" y="100" font-size="7" fill="currentColor">USB</text>
</svg>
<figcaption>librtlsdr hands the application a stream of IQ buffers: the app registers a callback, and the library pumps USB transfers from the dongle and invokes it once per filled buffer.</figcaption>
</figure>

## How it works

An application opens a device by index with `rtlsdr_open`, then configures it through a
uniform set of setters: `rtlsdr_set_center_freq`, `rtlsdr_set_sample_rate`,
`rtlsdr_set_tuner_gain`, and the frequency-correction and bias-tee controls. Under the
hood the library speaks to two chips — the RTL2832U itself and whichever tuner is
fitted (R820T, E4000, FC0013) — over `libusb`, translating high-level calls into the
right register writes for that tuner. The 2832U's ADC captures at up to ~3.2 MS/s
(2.4 MS/s is the reliable ceiling) and delivers interleaved unsigned-8-bit I and Q
bytes.

The defining feature is its **asynchronous read-callback** model. The application calls
`rtlsdr_read_async` with a function pointer; librtlsdr spins a `libusb` transfer loop on
that thread and invokes the callback each time a buffer fills, handing over a pointer and
a length. This is a classic [callback versus stream API](/reference/callback-vs-stream-api/)
design: the library owns the timing and "pushes" data, rather than the app "pulling" it.
A simpler synchronous `rtlsdr_read_sync` also exists for quick captures, but the async
path is what real-time decoders use because it keeps USB transfers in flight and avoids
sample [overruns](/reference/overruns-underruns/).

## Variants

The library ships with `rtl_sdr` (dump raw IQ to a file or pipe), `rtl_fm` (a compact
narrowband FM/AM demodulator), `rtl_power` (a swept power scanner), `rtl_test`, and —
most consequentially — `rtl_tcp`. That server reads from librtlsdr locally and re-serves
the byte stream and control commands over TCP, letting a remote client drive the dongle
as if it were attached; see [rtl_tcp](/reference/rtl-tcp/). Several forks exist (notably
one maintained by the rtl-sdr.com team, and one under librtlsdr's own name) that add
direct-sampling tweaks, newer tuner support, and bias-tee fixes; they share the same
core API.

## In practice

Because the API is small and stable, librtlsdr became the lowest common denominator that
almost everything else builds on. On Windows the WinUSB/Zadig driver has to replace the
DVB-T driver first before librtlsdr can claim the device. The bindings surface it into
higher layers: [gr-osmosdr](/reference/gr-osmosdr/) wraps it as a GNU Radio source, and
[SoapySDR](/reference/soapysdr/) exposes it through a `SoapyRTLSDR` plugin, so most
applications never call librtlsdr directly — they inherit it.

## Relevance to SDR

librtlsdr is arguably the single most important piece of software in hobbyist SDR: by
turning a ~$25 [RTL-SDR](/reference/rtl-sdr/) dongle into a usable IQ source it put a
receiver on millions of desks and seeded the whole open-radio toolchain. For a decoder
author it is a case study in host-library design — a thin, chip-specific driver that
normalizes gain, tuning, and rate control and then gets out of the way of the sample
stream.

GopherTrunk consumes RTL-SDR hardware, but as a pure-Go application it does not link the C
`librtlsdr` directly; it takes IQ over the network via [rtl_tcp](/reference/rtl-tcp/)-style
and SoapySDR-style transports (or from recorded [IQ files](/reference/iq-file-format/) for
replay). Functionally, though, GT sits exactly where an `rtlsdr_read_async` callback would:
it accepts pushed 8-bit IQ buffers, converts them to complex baseband, and feeds its
rate-invariant decode chain — the same producer/consumer contract librtlsdr defined, moved
one hop up the stack.

## Sources

[^osmo]: [rtl-sdr project wiki](https://osmocom.org/projects/rtl-sdr/wiki) — Osmocom, the origin and documentation of the librtlsdr library, its raw-sampling discovery, and the rtl_sdr/rtl_tcp tools.
