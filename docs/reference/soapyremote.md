---
slug: soapyremote
title: SoapyRemote
entry_type: technology
category: sdr-data-streaming
description: "SoapyRemote is a SoapySDR module that exposes any local SDR over the network as a transparent remote device, tunnelling the full device API and IQ stream over TCP."
keywords: SoapyRemote, SoapySDR remote, network SDR, remote SDR device, SoapySDR module, IQ over network, SoapySDRServer, transparent remote radio, Pothos
aka: [SoapyRemote, remote.soapy]
autolink: true
infobox:
  - { label: Type, value: Network transport for SoapySDR }
  - { label: Exposes, value: Any SoapySDR device as a remote device }
  - { label: Server, value: SoapySDRServer daemon }
see_also: [soapysdr, network-iq-streaming, rtl-tcp, ka9q-radio, spyserver-protocol, gnuradio]
cite_urls:
  - https://github.com/pothosware/SoapyRemote/wiki
  - https://github.com/pothosware/SoapySDR/wiki
---

**SoapyRemote** is a [SoapySDR](/reference/soapysdr/) module that makes any locally-attached SDR
usable over a network as if it were plugged into the remote machine — it forwards the *entire*
SoapySDR device API and the [IQ](/reference/iq-data/) sample stream across the link.[^soapy] Where
[rtl_tcp](/reference/rtl-tcp/) is a fixed protocol for one family of dongles, SoapyRemote is a
general transparent bridge: whatever radio SoapySDR supports locally, SoapyRemote exports remotely
with the same interface applications already use.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="On the server, SoapySDRServer wraps a real SDR behind SoapySDR; over the network the SoapyRemote client module presents it to an application as an ordinary SoapySDR device." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <text x="118" y="24" font-size="8" fill="currentColor">server host</text>
    <path d="M34 78 v-18 m-6 0 l6 -8 l6 8" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="54" y="62" width="54" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="81" y="80">real SDR</text>
    <rect x="112" y="62" width="86" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="155" y="76">SoapySDR-</text><text x="155" y="86">Server</text>
    <text x="352" y="24" font-size="8" fill="currentColor">client host</text>
    <rect x="262" y="62" width="86" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="305" y="76">SoapyRemote</text><text x="305" y="86" font-size="7">client module</text>
    <rect x="352" y="62" width="86" height="30" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="395" y="80">application</text>
    <line x1="198" y1="77" x2="261" y2="77" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3" marker-end="url(#srar)"/><text x="230" y="70" font-size="7">TCP</text>
    <line x1="348" y1="77" x2="351" y2="77" stroke="currentColor" stroke-width="1.1"/>
  </g>
  <defs><marker id="srar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SoapySDRServer exports a real radio; the SoapyRemote client module re-presents it to the application as an ordinary local SoapySDR device.</figcaption>
</figure>

## How it works

On the machine with the radio you run **`SoapySDRServer`**, which advertises the device on the LAN
(via mDNS/Avahi) and listens for clients. On the application side, SoapySDR is told to open a device
with a `driver=remote` argument naming the server; the SoapyRemote client module connects, discovers
the remote device's channels, gain elements, sample rates and tuning ranges, and presents them
upward exactly as a local driver would. Every API call — `setFrequency`, `setSampleRate`, `setGain`,
`readStream` — is marshalled across the link, so the application cannot tell it is talking to a radio
in another room.

Two channels carry the traffic: a reliable control connection for the API calls and a separate,
latency-optimised path for the bulk sample stream, which can use UDP so a late packet is dropped
rather than stalling the pipeline. SoapyRemote can apply a lightweight sample-format conversion to
trim bandwidth — for example forwarding CS8 or CS16 instead of CF32 — and lets the client pick the
[sample format](/reference/sample-format/) it wants. Because the abstraction is complete, the same
GNU Radio flowgraph or SDR application runs unchanged against a local or a remote device; only the
device string changes.

## Relevance to SDR

SoapyRemote is the modern, device-agnostic answer to remote SDR. It is the natural choice when the
radio is *not* an RTL dongle — a HackRF, LimeSDR, [Airspy](/reference/airspy/), or SDRplay — because
[rtl_tcp](/reference/rtl-tcp/) only speaks RTL and [SpyServer](/reference/spyserver-protocol/) only
serves Airspy, whereas SoapyRemote inherits SoapySDR's broad hardware support. In the GNU Radio and
Pothos world it is the standard way to keep the radio at the antenna and the flowgraph on a
workstation, a common [network IQ streaming](/reference/network-iq-streaming/) topology.

GopherTrunk has its own hardware backends rather than a SoapySDR dependency, so it does not consume a
SoapyRemote device directly; its remote-radio story is the raw-IQ network sources it supports and its
offline replay of captured files. SoapyRemote matters to a GopherTrunk user as the way to relay a
non-RTL radio's stream to a host that a GopherTrunk-supported input can read, and as the clearest
example of the abstraction SoapySDR provides — a single device interface that a network transport can
slot underneath without the application noticing.

## In practice

A typical setup runs `SoapySDRServer --bind` on the antenna-side machine and opens
`driver=remote,remote=192.168.1.50,remote:driver=hackrf` on the client. Discovery over mDNS means the
server often appears automatically in SoapySDR device enumeration on the same LAN, so tools list it
without a hand-typed address. The two failure modes to watch are bandwidth — SoapyRemote forwards the
full negotiated sample rate, so a high-rate radio still needs a gigabit link unless you decimate first —
and clock domain: the samples are timed by the server's radio, so a saturated network manifests as
overruns at the source rather than smooth degradation. Keeping the bulk stream on UDP and the same
subnet, as with any [network IQ streaming](/reference/network-iq-streaming/), is the reliable
configuration.

## Sources

[^soapy]: [SoapyRemote wiki](https://github.com/pothosware/SoapyRemote/wiki) — the Pothos project documentation for the SoapyRemote module and SoapySDRServer, describing transparent network access to any SoapySDR device.
