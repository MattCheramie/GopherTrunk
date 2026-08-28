---
slug: ka9q-radio
title: ka9q-radio
entry_type: technology
category: sdr-data-streaming
description: "ka9q-radio is Phil Karn's multicast SDR system: one radiod process channelizes a front end once and publishes every channel as IP multicast streams that any number of consumers on the LAN can receive independently."
keywords: ka9q-radio, radiod, Phil Karn, KA9Q, multicast SDR, IP multicast IQ, RTP streams, channelizer, SSRC, mDNS discovery, network SDR
aka: [ka9q-radio, radiod, KA9Q radio]
autolink: true
infobox:
  - { label: Type, value: Multicast SDR server }
  - { label: Author, value: Phil Karn, KA9Q }
  - { label: Transport, value: RTP over IP multicast }
  - { label: Model, value: "Channelize once, consume many times" }
see_also: [soapyremote, rtl-tcp, network-iq-streaming, channelizer, phil-karn, spyserver-protocol]
external:
  - { title: "ka9q-radio on GitHub", url: "https://github.com/ka9q/ka9q-radio" }
cite_urls:
  - https://github.com/ka9q/ka9q-radio
  - https://en.wikipedia.org/wiki/Phil_Karn
---

**ka9q-radio** is [Phil Karn](/reference/phil-karn/)'s answer to a structural waste in
networked SDR: most protocols ship the *whole* digitized passband to each client, which
then throws away everything but one channel.[^gh] ka9q-radio inverts that. A single
`radiod` process owns the front end, channelizes the entire passband **once** with an
efficient FFT-based [channelizer](/reference/channelizer/), and publishes each channel —
hundreds at a time, if asked — as its own RTP stream on an **IP multicast** group. Any
number of consumers on the LAN then subscribe to just the channels they want, and adding a
listener costs the network nothing it was not already carrying.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="One SDR front end feeds a radiod process that fans out many channel streams onto a multicast network, where several independent consumers each subscribe to the channels they want." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="62" width="64" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="52" y="81" font-size="9" fill="currentColor" text-anchor="middle">SDR</text>
  <line x1="84" y1="77" x2="126" y2="77" stroke="currentColor" stroke-width="1.2" marker-end="url(#kqar)"/>
  <rect x="130" y="52" width="86" height="50" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <text x="173" y="73" font-size="9" fill="currentColor" text-anchor="middle">radiod</text>
  <text x="173" y="88" font-size="7.5" fill="currentColor" text-anchor="middle">channelize once</text>
  <line x1="216" y1="62" x2="286" y2="34" stroke="currentColor" stroke-width="1.1" marker-end="url(#kqar)"/>
  <line x1="216" y1="77" x2="286" y2="77" stroke="currentColor" stroke-width="1.1" marker-end="url(#kqar)"/>
  <line x1="216" y1="92" x2="286" y2="122" stroke="currentColor" stroke-width="1.1" marker-end="url(#kqar)"/>
  <text x="251" y="48" font-size="7.5" fill="currentColor" text-anchor="middle">multicast groups</text>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="290" y="20" width="140" height="26" rx="4"/><rect x="290" y="64" width="140" height="26" rx="4"/><rect x="290" y="108" width="140" height="26" rx="4"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="360" y="36">decoder A (one channel)</text><text x="360" y="80">decoder B (another)</text><text x="360" y="124">recorder (several)</text>
  </g>
  <defs><marker id="kqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The front end is digitized and channelized exactly once; every consumer subscribes to ready-made channels instead of re-tuning its own copy of the passband.</figcaption>
</figure>

## How it works

`radiod` reads the hardware (RX888, Airspy, RTL-SDR, and others), runs the passband through
a fast-convolution filter bank, and maintains one demodulator instance per configured
channel — each with its own frequency, bandwidth, and mode (IQ, USB, FM, …). Each channel's
output goes out as RTP on a multicast group, identified by an **SSRC** (by convention, the
channel's frequency in Hz), alongside a *status* multicast group carrying metadata —
sample rate, output socket, signal levels — as typed key/value pairs. Control uses the same
status group: a consumer can create or re-tune channels remotely by sending commands to it.
Service discovery rides mDNS, so streams have names like `hf.local` rather than bare
multicast addresses. The consumer-side toolbox (`monitor`, `pcmrecord`, digital-decoder
front ends) is deliberately small and composable in the Unix style.

Contrast the models: [rtl_tcp](/reference/rtl-tcp/) ships one client the raw passband;
[SoapyRemote](/reference/soapyremote/) gives one client remote *control* of a device plus
its stream; [SpyServer](/reference/spyserver-protocol/) multiplexes clients but demodulates
per client on the server. ka9q-radio alone makes the channel — not the device — the shared
network product, which is what lets one antenna on a rooftop serve a whole shack (or a
whole club) of independent decoders.

## Relevance to SDR

For a trunking scanner the model is a natural fit: a trunked system *is* a set of known
channels, and a radiod instance can carry the control channel and every voice channel of a
site simultaneously from one front end. GopherTrunk consumes ka9q-radio natively
(`internal/sdr/ka9qradio/`, config key `ka9q_radio`): each configured `{addr, ssrc}` pair
appears to the scanner as a virtual tuner — the driver resolves the status group via mDNS,
learns the channel's data socket, sample rate, and encoding from a status poll, and can
re-tune the channel through radiod's command interface. The practical wins are physical:
`radiod` runs on a small computer at the antenna, the samples cross the LAN instead of
lossy coax, and several GopherTrunk instances (or GopherTrunk plus other ka9q consumers)
share one front end without contending for USB.

## Sources

[^gh]: [ka9q-radio](https://github.com/ka9q/ka9q-radio) — Phil Karn, KA9Q, on the multicast channelize-once architecture, radiod, and the RTP/status stream design.
