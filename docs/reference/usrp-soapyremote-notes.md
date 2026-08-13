---
slug: usrp-soapyremote-notes
title: USRP & SoapyRemote field notes
entry_type: term
category: fn-hardware
description: "Field-tested facts for running USRP hardware through GopherTrunk's pure-Go SoapyRemote driver: silently-ignored stream args, the macOS loopback MTU cap, UHD rate coercion, and the wire-protocol behaviors behind zero-sample streams."
keywords: usrp, b210, x310, twinrx, soapyremote, soapysdr, soapysdrserver, stream_mtu, mtu, uhd, master clock rate, rate coercion, flow control, rfnoc, agc, gain
see_also: [soapyremote, soapysdr, usrp-b210, usrp-ettus, sdr-gain-overload, sample-rate, automatic-gain-control, dbfs, airspy-rate-selection, diagnostic-playbook]
---

**USRP & SoapyRemote field notes** cover the traps found while field-testing Ettus
hardware ([B210](/reference/usrp-b210/), X310/TwinRX, N310) with GopherTrunk.
GopherTrunk is built with CGO disabled, so there is no native UHD backend: all
[USRP](/reference/usrp-ettus/) devices are driven through a pure-Go
[SoapyRemote](/reference/soapyremote/) wire-protocol client talking to a
`SoapySDRServer` process — over loopback for a locally attached device, or across the
network. The B210 is a USB device (`type=b200`); only the N- and X-series are
themselves networked.

## Configuration traps

| Symptom | Looks like | Actually | Fix/Check |
|---|---|---|---|
| Tuned `remote:mtu=8000` (or `remote:window`, `remote:prot`) has no effect | Server ignoring the client | These keys inside `args` are **silently ignored** — GopherTrunk builds the SETUP_STREAM frame itself and never forwards `args` into stream setup; the stream ran at the 1500-byte default | Use the dedicated `stream_mtu` / `stream_window` keys; config load now hard-rejects the `remote:*` forms ([#876](https://github.com/MattCheramie/GopherTrunk/issues/876)) |
| High-rate USRP overruns even though server and client are the same Mac | Server too slow | macOS caps the MTU on `lo0`, so a server bound to 127.0.0.1 can never negotiate a jumbo `stream_mtu` | Bind `SoapySDRServer` to the host's **real NIC IP**, even for local use; Linux loopback has no such cap ([#876](https://github.com/MattCheramie/GopherTrunk/issues/876)) |
| Decode works but the rate isn't what you configured | GopherTrunk bug | UHD silently coerces any [sample rate](/reference/sample-rate/) that isn't an integer decimation of the master clock; GopherTrunk reads the delivered rate back over RPC and clocks its DSP from it, so decode survives | For exactness set `master_clock_rate: 61_440_000` and a dividing `sample_rate` — 6,144,000 (÷10) ran overrun-free on a B210 ([#876](https://github.com/MattCheramie/GopherTrunk/issues/876)) |
| Manual gain apparently ignored on a TwinRX | Gain bug | The TwinRX has **no AGC hardware**; `setGainMode(false)` throws, and an early driver returned on that error before ever applying manual gain | Disabling [AGC](/reference/automatic-gain-control/) must be best-effort; fixed in [#542](https://github.com/MattCheramie/GopherTrunk/issues/542) |
| Gain value behaves 10× off | Driver bug | Gain is in **tenths of a dB**: `"300"` = 30 dB, `"30"` = 3 dB | See [SDR gain & front-end overload](/reference/sdr-gain-overload/) |

## Wire-protocol behaviors that look like bugs

Both of these were misdiagnosed repeatedly in
[#542](https://github.com/MattCheramie/GopherTrunk/issues/542) — all three of the
originally proposed root causes (an `Open()` race, rate-after-frequency ordering, and
`setGainMode(true)`) turned out to be wrong.

- **SETUP_STREAM is two-phase and two-socket.** The server first replies with just the
  bound port string, then blocks accepting **two** client connections (stream + status),
  and only then sends a second reply with the stream id. A client that reads one reply
  and opens one socket gets a short-response error — and retrying SETUP_STREAM on the
  server's still-bound graph can crash libuhd in the server process
  (`Attempting to reconnect output port ...` followed by a segfault). The segfault is
  downstream fallout, not the bug.
- **TCP mode still uses application-level flow control.** The server's sender thread
  transmits **zero samples** until the client sends an initial 24-byte ACK, then stays
  at most a window of sequences ahead. A missing ACK looks like a healthy connection
  that delivers no IQ (`iq_observed=false iq_samples=0` read timeouts, with the server
  logging overruns). It is not a UDP-only mechanism.
- **A cold X310 takes seconds before reply #1** while it compiles its RFNoC graph, so
  stream setup needs a long read deadline — otherwise the client fails with an RPC
  header read timeout against a perfectly healthy server.
- **UHD's own debug output misleads.** Lines like
  `[DDC] Not setting frequency until sampling rate is set` are internal UHD block-init
  chatter, not evidence about the client's call ordering — treating them as a smoking
  gun sent #542 down a false path. The same pattern (UHD CIC-rolloff messages read as
  resampler failures) derailed
  [#550](https://github.com/MattCheramie/GopherTrunk/issues/550).

## Operational notes

- **Run `SoapySDRServer` supervised.** It crashes on its own; a process supervisor
  (systemd, runit) that restarts it is part of any unattended deployment
  ([#876](https://github.com/MattCheramie/GopherTrunk/issues/876)).
- Debugging incantation:
  `SOAPY_SDR_LOG_LEVEL=DEBUG SOAPY_REMOTE_IP=<gt-host-ip> SoapySDRServer --bind=<usrp-host-ip>:23313`.
- Throughput reference point: an N310 ran 20 MS/s overrun-free through the SoapyRemote
  flow control ([#876](https://github.com/MattCheramie/GopherTrunk/issues/876)).
- **`signal_dbfs` on a recorded call is channel power in [dBFS](/reference/dbfs/)** —
  not calibrated RSSI, and not SNR or EVM. Do not compare it across devices with
  different gain settings.

For gain selection on these front ends, see
[SDR gain & front-end overload](/reference/sdr-gain-overload/); for the general
capture-and-replay approach to qualifying a device/rate combination, see
[Airspy sample-rate selection](/reference/airspy-rate-selection/) and the
[diagnostic playbook](/reference/diagnostic-playbook/).

## Provenance

- [#876](https://github.com/MattCheramie/GopherTrunk/issues/876) — USRP B210 field testing: ignored `remote:*` stream args, the macOS `lo0` MTU cap, UHD rate coercion and the master-clock divisor rule, gain units, supervision advice.
- [#542](https://github.com/MattCheramie/GopherTrunk/issues/542) — SoapyRemote "segfault on connect": the two-phase two-socket SETUP_STREAM handshake, TCP flow-control ACK, TwinRX best-effort AGC, cold-X310 RFNoC compile delay — after three confidently wrong root causes.
