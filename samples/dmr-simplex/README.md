# Single DMR simplex frequency, one RTL-SDR

Monitor one DMR simplex (or hotspot) carrier with a single RTL-SDR on
Linux. The shipped `config.yaml` targets **446.500 MHz, Color Code 1,
Time Slot 1, Talkgroup 99**, but it works for any single DMR Tier II
carrier — just change the frequency.

## The one thing to understand first

You do **not** configure Color Code, Time Slot, or Talkgroup.
GopherTrunk decodes them off the air and reports them per call:

- **Color Code** and **Time Slot** are read from every DMR burst.
- **Talkgroup** (and source Radio ID) come from the call's Link Control.

So the config only needs the **frequency** and that it's **DMR Tier II**.
The `talkgroups-dmr.csv` file is optional cosmetics — it maps the decoded
TG numbers (like `99`) to friendly names in the UI and recordings.

## Setup (Linux)

1. **Plug in the RTL-SDR** and confirm GopherTrunk sees it:

   ```sh
   gophertrunk sdr list
   ```

   Copy the dongle's serial.

2. **Edit `config.yaml`** and replace `REPLACE_WITH_SERIAL` with that
   serial. (If you only have one dongle you can leave it blank — the
   first one found wins — but pinning the serial is more robust.)

3. **Run it**, pointing at this config:

   ```sh
   gophertrunk run -config samples/dmr-simplex/config.yaml
   ```

4. Open **http://127.0.0.1:8080** in a browser for the live UI, or watch
   the terminal: `widebandt2: starting` confirms the engine came up, and
   a `grant` / call line fires each time someone keys up on 446.500.

## Why `role: wideband` for one frequency?

DMR Tier II is *conventional* — there is no trunked control channel, so
the carrier is handed to GopherTrunk's wideband engine, which channelizes
the dongle's IQ and runs a DMR Tier II decoder on the result. With a
single channel that's just one down-converter tap; no extra hardware and
no per-call retuning.

The dongle is intentionally centered at **446.800 MHz**, 300 kHz above
the carrier, so 446.500 MHz lands *off* the RTL-SDR's center-DC spike.
Sitting a digital carrier right on DC corrupts the demod, so this offset
matters. Any offset that keeps the carrier inside `center ± ~1.08 MHz`
(the usable half-band at 2.4 MS/s, minus a 5% guard) is fine.

## If nothing decodes

- **Set the serial** correctly (`gophertrunk sdr list`).
- **Check the antenna / gain.** Try a fixed gain instead of `"auto"`,
  e.g. `gain: "320"` (= 32.0 dB — gains are in *tenths* of a dB here).
- **Set `ppm`** to your dongle's measured frequency error. A few kHz of
  drift on a cheap RTL crystal can keep DMR from locking.
- **Set `log.level: debug`** to see lock/sync detail.
- Confirm the frequency is right and that there's actually traffic
  (key up your own radio on 446.500 / CC1 / TS1 / TG99 to test).
