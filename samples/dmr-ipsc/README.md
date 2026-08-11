# DMR IPSC / linked-conventional ("pseudo-trunk") repeater profile

A ready-to-edit config for the **linked conventional DMR** systems that are
common on a budget: several DMR Tier II repeaters joined over an IP back-haul
(Motorola calls the link IPSC), all on the **same Colour Code**, with radios
programmed to a **list of repeater frequencies** that they roam by best RSSI.
There is **no control channel** — it is conventional DMR, just spread across a
few linked carriers — so GopherTrunk decodes it as `protocol: dmr-tier2`.

See [`config.yaml`](config.yaml) in this folder; copy it, set your dongle
serial and the repeater frequencies, and run
`gophertrunk --config samples/dmr-ipsc/config.yaml`.

## Why a dedicated profile

Pointing GopherTrunk at a linked conventional system as a plain trunked system
produced two annoyances this profile fixes:

- **Quiet logs.** A conventional repeater drops to *no carrier* between calls —
  an idle carrier is normal, not a fault. GopherTrunk no longer floods the log
  with `iq power very low …` every few seconds for a quiet or unused listed
  frequency. A genuinely dead / mistuned carrier (one that never decodes) is
  still called out — but only **once**, not on every diagnostics window.

- **One Colour Code.** Set `color_code:` on the system and GopherTrunk pins the
  decoder to that Colour Code: any DMR burst on a *different* Colour Code (a
  co-channel system bleeding into the wideband passband) is dropped before it
  can log a call. Leave it unset to accept every Colour Code (the historical
  default — GT still reads and reports the Colour Code per call).

## What you do and don't configure

| You set | GopherTrunk reads off the air |
| --- | --- |
| Repeater output frequencies (as `role: wideband` channels) | Talkgroup numbers |
| The system's Colour Code (`color_code:`, optional filter) | Time Slot (TS1 / TS2) |
| Friendly TG names (the talkgroups CSV) | Source radio IDs, aliases, GPS |

Both timeslots are followed automatically: each transmission begins with its
own Voice LC Header, which becomes its own call, so a busy repeater's two slots
record as two calls. Size `voice_taps` for the number of simultaneous calls you
want to follow.

## Fitting the repeaters into the dongle

DMR is decoded by GopherTrunk's wideband channelizer: one `role: wideband`
dongle is tuned to a centre frequency and split into a per-repeater decoder.
At 2.4 MS/s one dongle sees ~±1.08 MHz around its centre — list only repeaters
inside that window. For carriers spread wider, add a second `role: wideband`
dongle pointing at the **same** system name (channels are unioned across
dongles); see [`../dmr-tier2-multi-repeater`](../dmr-tier2-multi-repeater) for
the multi-dongle layout, and [`../dmr-simplex`](../dmr-simplex) for a single
hotspot carrier.

## If nothing decodes

A single DMR carrier is only ~12.5 kHz wide and the 4-level slicer tolerates
only ~±100 Hz of residual carrier offset. A cheap RTL-SDR is often tens of ppm
off — enough to stop decoding while SDR++ still makes the warbling "helicopter"
FM sound (FM audio is offset-tolerant; a symbol slicer is not). If the daemon
logs `strong in-channel signal but no sync`, measure your dongle's error and
set `ppm:` on the device. This is the single most common cause of "I hear it in
SDR++ but GopherTrunk decodes nothing."
