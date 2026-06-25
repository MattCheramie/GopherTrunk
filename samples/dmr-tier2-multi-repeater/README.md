# A fleet of conventional DMR repeaters across a wide band

This sample monitors many conventional **DMR Tier II** repeaters — the
"40 freqs with different Color Codes and slots, just a bunch of repeaters"
case — spread across more RF than a single dongle can cover. It tunes one
`role: wideband` dongle per ~2 MHz cluster of repeaters and decodes every
repeater in parallel.

If all your repeaters cluster inside a single ~2 MHz window, use the simpler
[`dmr-tier2-multichannel`](../dmr-tier2-multichannel/) sample (one dongle). For
a single simplex/hotspot frequency, see [`dmr-simplex`](../dmr-simplex/).

## The one thing to understand first

**Color Code, Time Slot (TS1/TS2), and Talkgroup are not configured.**
GopherTrunk decodes them straight off the air and reports them per call. You
only declare:

- **where** to listen — each repeater's output frequency, and
- **what** it is — `protocol: dmr-tier2`.

So a deployment of 40 repeaters with 40 different Color Codes and both slots in
use is, from a config standpoint, just **40 frequencies**. There is nothing
per-Color-Code or per-slot to type in. The `talkgroups-dmr.csv` file only maps
decoded talkgroup numbers (e.g. `8`) to friendly names.

## Why you may need more than one dongle

DMR is decoded by GopherTrunk's wideband channelizer
(`internal/scanner/widebandt2`): a dongle is pinned to a center frequency and
the daemon splits its IQ stream into one narrow-band DMR decoder per repeater.
There is **no sequential "hop through N frequencies" DMR scanner** — every
repeater must sit inside some dongle's IQ window simultaneously.

At 2.4 MS/s a dongle sees roughly **±1.08 MHz** of usable bandwidth around its
center (≈2.16 MHz total, after a 5% guard at each edge; out-of-band channels
are rejected at startup). So:

- Repeaters that all fall inside one ~2 MHz window → **one dongle**.
- Repeaters spread wider → **one dongle per cluster**.

  > Rule of thumb: 40 repeaters spread over ~8 MHz ≈ **4 dongles**, ~10 each.

## Spanning more than one dongle

Point **every** dongle's channels at the **same** `system:` name. GopherTrunk
unions the wideband channels across all dongles into one system, so calls from
any repeater land under one consistent system in the API, recordings, and call
log. This sample uses three dongles (centered at 453.5 / 460.5 / 465.5 MHz),
all feeding `system: regional-dmr`. To add another cluster, copy a `- serial:`
block, re-center it, and list that cluster's repeaters.

Because Tier II is conventional, the system needs **no `control_channels`** —
GopherTrunk derives the carrier list from the wideband channels assigned to it,
so each repeater frequency is listed exactly once. (An explicit
`control_channels:` list is still accepted and honoured verbatim if you prefer
to be explicit.)

## Channelizer strategy

`tuner_strategy: auto` (the default) picks a single-channel **DDC** tap for ≤ 6
channels and a **polyphase** channelizer above that — so a dongle hosting 8–10
repeaters automatically uses the polyphase path. Force `ddc` or `polyphase`
only if you have a reason to override.

## Tuning

Edit `config.yaml`:

1. **`serial`** — run `gophertrunk sdr list` and paste each dongle's serial.
   The daemon binds each channel list to its device by serial.
2. **`center_freq_hz`** — center each dongle so every one of its
   `channels[].frequency_hz` sits within `center ± ~1.08 MHz`. Keep carriers a
   little off the exact center to dodge the RTL-SDR's center-DC spike. The
   config validator rejects out-of-band channels at startup.
3. **`channels`** — one entry per repeater output frequency, each referencing
   `system: regional-dmr`.

## Running

```sh
go run ./cmd/gophertrunk run -config samples/dmr-tier2-multi-repeater/config.yaml
```

The startup log line `widebandt2: starting` (one per dongle) confirms the
engine came up. `cc.locked` and `grant` events fire per repeater frequency as
calls arrive, each carrying the Color Code and time slot decoded from the air.

## If nothing decodes

- Confirm the repeaters are actually keyed up (DMR is bursty — idle repeaters
  are silent). Bump `log.level` to `debug`.
- Check each carrier is inside its dongle's window (`center ± ~1.08 MHz`); a
  startup error names any out-of-band channel.
- Verify gain — `"auto"` is the safe start; a deafened front end decodes
  nothing.
