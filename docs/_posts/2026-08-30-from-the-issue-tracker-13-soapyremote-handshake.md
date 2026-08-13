---
title: "From the Issue Tracker, Part 13: The SoapyRemote Handshake — Three Wrong Root Causes and a Server That Says Nothing First"
description: A SoapySDRServer fronting a USRP X310 segfaulted the moment GopherTrunk connected. The reporter filed three confident root-cause analyses; all three were disproved by the code. The real bugs were a two-phase, two-socket SETUP_STREAM handshake the client half-implemented — and a TCP flow-control ACK without which the server sends zero samples, forever.
category: solution-postmortem
keywords: soapyremote, usrp x310, twinrx, uhd, rfnoc, setup_stream, flow control ack, segfault, network sdr, stream handshake, agc
tags: [from-the-issue-tracker, soapysdr, usrp, network-sdr, drivers, debugging, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker"
series_part: 13
---

*Part 13 of **From the Issue Tracker**, postmortems of GopherTrunk bugs that fought
back. [Part 12]({{ '/blog/solution-postmortem/from-the-issue-tracker-12-seventy-eight-degrees/' | relative_url }})
diagnosed a driver from three numbers. This part closes the driver cluster with the
opposite situation: a wall of confident diagnosis — three detailed root-cause
analyses from the reporter — every word of it wrong, in instructive ways.*

> **TL;DR:** Pointing GopherTrunk's `soapyremote` driver at a `SoapySDRServer`
> fronting a USRP X310/TwinRX crashed the *server* within seconds
> ([#542](https://github.com/MattCheramie/GopherTrunk/issues/542)). The reporter
> filed three root causes — a concurrent `Open()` race, `setFrequency` ordered
> before `setSampleRate`, and a gain parser blindly enabling AGC. All three were
> disproved by reading the code. The real first bug: SoapyRemote's TCP
> `SETUP_STREAM` is a **two-phase, two-socket** handshake, and the client read one
> reply and opened one socket — the resulting failure/retry storm re-ran setup
> against a still-bound RFNoC graph until libuhd crashed. The real second bug:
> SoapyRemote runs an application-level **flow-control ACK** protocol even over
> TCP, and the server sends *zero samples* until the receiver ACKs first. Plus one
> genuine gain bug the reporter's third theory brushed past: on a TwinRX (no AGC
> hardware), disabling AGC throws, and the old code returned early without ever
> applying manual gain.

## The symptom: a client that kills its server

The setup: GopherTrunk on a Mac, `SoapySDRServer` on the same host, USRP X310 with
a TwinRX daughterboard behind it, UHD 4.7. On connect, GopherTrunk logged
`soapyremote: setup stream port: soapyremote: short rpc response`, retried, and
within three retries the *server* died:

```text
[ERROR] [RFNOC::GRAPH::DETAIL] Attempting to reconnect output port 0/DDC#0:0
SoapyServerListener::handlerLoop() FAIL: SoapyRPCUnpacker::recv(header) FAIL:
zsh: segmentation fault  SoapySDRServer
```

A client that can segfault its server is alarming, and the reporter dug in hard —
producing three separate, detailed, confidently-titled root-cause analyses.

## Three confident theories, three disproofs

Each theory was specific enough to check against the code, and each check failed:

| Theory | Disproof |
|---|---|
| "Concurrent `Open()` calls race inside UHD's graph compiler; serialize with a global mutex singleton" | `sdr.Pool.OpenWith` already opens devices strictly sequentially under a mutex — and this config had exactly one device. The duplicate connections in the server log were *sequential* retry attempts, not parallelism. |
| "`setFrequency` is sent before `setSampleRate`, so the DDC drops the tune" | The driver already set sample rate at open and frequency later at tune time. The `[DDC] Not setting frequency until sampling rate is set` lines were UHD's *own internal block-init debug output*, not responses to any client RPC. |
| "The gain parser blindly calls `setGainMode(true)` for numeric gains" | It calls `setGainMode(false)` — to *disable* AGC before applying manual gain. (There was a real bug adjacent to this one; see below.) |

There's a debugging lesson in the pattern itself: all three theories were built by
reading the *server's log* and inferring what the client must be doing. Logs from
the wrong side of a protocol invite exactly this — every theory was a plausible
story about code the theorist hadn't read. The disproofs each took minutes once the
question became "what does the client actually send, in what order?"

## Real bug one: the handshake has a second phase

The driver's stream setup had never been validated against a live server (its
package doc said as much). Checked against upstream SoapyRemote's
`client/Streaming.cpp` and `server/ClientHandler.cpp`, the TCP `SETUP_STREAM`
choreography is two-phase and two-socket:

1. Client sends `SETUP_STREAM`.
2. Server replies **#1** with just the bound data port — one string.
3. Server calls `listen(2)` and **blocks accepting two** client sockets: a stream
   socket *and* a status socket.
4. Server replies **#2** with the stream id — an **int**.

The old client read one reply expecting two strings, opened one socket, and packed
the stream id as a string. Reading a second string out of a one-string reply ran
off the buffer — the `short rpc response` in the log. Then the layered retry
machinery did its job, which made everything worse: each reacquire re-ran
`SETUP_STREAM` against a graph the server still had bound, provoking
`Attempting to reconnect output port 0/DDC#0:0` inside libuhd until it segfaulted.
The crash was downstream fallout in an external process — a genuine UHD/SoapyRemote
robustness bug worth reporting upstream, since a server should survive any client
RPC — but GopherTrunk was the provocateur, and fixing the handshake stopped the
storm at its source.

One practical footnote that cost a retest: a cold X310 spends several seconds
compiling its RFNoC graph before reply #1 arrives, so setup needs a long read
deadline or it manufactures its own `read rpc header: i/o timeout`.

## Real bug two: the server says nothing until you speak

With the handshake fixed, the segfault vanished — replaced by silence. The stream
socket delivered zero bytes; the hunt failed with `iq_observed=false iq_samples=0`
while the server logged `Received overrun message on port 0` — the radio producing
samples with nowhere to put them.

The missing piece lives in SoapyRemote's `common/SoapyStreamEndpoint.cpp`: an
application-level flow-control ACK protocol on the data socket, used **in TCP mode
too**, not just UDP. The server's sender thread blocks in `waitSend()` while
`not _receiveInitial` — it will not send a single sample until the receiver posts
an initial 24-byte ACK. After that it runs at most `_maxInFlightSeqs` ahead of the
last acknowledged sequence, so the receiver must keep ACKing on a cadence. TCP's
own flow control makes an application-layer window feel redundant, which is exactly
why a from-scratch client omits it: nothing in the connection fails, the RPCs all
succeed, and the stream is simply, permanently empty.

The fix implements the receiver side byte-for-byte: a gratuitous ACK (sequence 0)
right after setup to prime the sender, then periodic ACKs advertising the credit
window. The fake test server now models the `_receiveInitial` gate, so removing the
initial ACK reproduces the reporter's exact zero-IQ timeout in a unit test.

## Real bug three: the TwinRX has no AGC to disable

The reporter's third theory pointed near a genuine bug without landing on it. The
TwinRX daughterboard has no AGC hardware at all, so even `setGainMode(false)` — the
*disable* — throws `NotImplementedError: set_rx_agc() is not supported on this
radio!` remotely. The old code treated that as fatal and returned early, so the
configured manual gain was **never applied**. Disabling AGC is now best-effort:
log it, move on, always run `setGain`. The verification log shows the sequence
working — the AGC complaint demoted to debug, followed by `sdr: gain set gain_db=75`.

The reporter's final report: stream running, P25 decoding with CQPSK on the X310.

## What we keep

- **Disprove theories by reading the accused code, not by patching it.** All three
  proposed fixes (mutex singleton, RPC reorder, gain-parser rework) would have
  shipped real complexity against imaginary bugs — and the symptom would have
  survived all three. Minutes of code-reading per theory beat days of speculative
  engineering.
- **A wire protocol reconstructed from source needs a live-server validation pass
  before it's real.** The handshake bug survived because a fake server faithfully
  implemented the same misreading — the self-consistent-fake trap. The rewritten
  fake now mirrors upstream behavior (two sockets, int id, ACK gate) so the tests
  disagree with wrong clients. Protocol notes live in
  [USRP and SoapyRemote notes]({{ '/reference/usrp-soapyremote-notes/' | relative_url }}).
- **"Connected but zero samples" on SoapyRemote means the flow-control ACK is
  missing.** The server sends nothing until the receiver speaks first — even over
  TCP. That symptom-to-cause pair is in the
  [diagnostic playbook]({{ '/reference/diagnostic-playbook/' | relative_url }}).
- **Capability probes must be best-effort.** "Disable the feature this hardware
  doesn't have" can throw; treating it as fatal turned a no-op into a
  gain-never-applied bug. Apply the setting you actually need, tolerate the
  cleanup you don't.
- **Your retry loop can be someone else's denial-of-service.** Layered recovery
  amplified one malformed handshake into a crash loop in an external process.
  Retries need to distinguish "transient" from "protocol-level wrong," because
  re-running the latter harder only spreads the damage.
