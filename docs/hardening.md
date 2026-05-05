# Hardening & operations

This page covers the production-readiness items that landed in
Phase 10 of the build plan: Prometheus metrics, graceful shutdown,
the Docker assets, and the USB pass-through recipe.

## Metrics

GopherTrunk exposes Prometheus metrics on the HTTP API at
`GET /metrics` when the daemon is started with metrics enabled. The
collector lives in [`internal/metrics`](../internal/metrics).

Series exposed:

| Series                                                  | Type    | Description                                                       |
| ------------------------------------------------------- | ------- | ----------------------------------------------------------------- |
| `gophertrunk_events_total{kind=...}`                    | counter | Every event observed on the internal bus, by kind                 |
| `gophertrunk_calls_total{reason=...}`                   | counter | Calls completed, labelled by `EndReason` (normal/preempted/...)   |
| `gophertrunk_calls_active`                              | gauge   | Currently-active call count                                       |
| `gophertrunk_control_channel_locked{system=...}`        | gauge   | `1` while the named system has its CC locked, `0` otherwise        |
| `gophertrunk_sdr_iq_underruns_total{driver,serial}`     | counter | IQ pipeline drops because a downstream stage was too slow         |
| `gophertrunk_sdr_usb_reconnects_total{driver,serial}`   | counter | Times the USB driver had to re-open a device                      |
| `gophertrunk_decode_errors_total{protocol,stage}`       | counter | Decode failures by protocol + stage (e.g. `p25`/`tsbk-crc`)       |
| `gophertrunk_sdr_attached{driver,serial,role}`          | gauge   | `1` per currently-attached SDR                                    |
| `gophertrunk_build_info{version}`                       | gauge   | Always `1`; carries the build version as a label                  |

The bus-driven counters (`events_total`, `calls_total`, `calls_active`,
`control_channel_locked`) update automatically. Subsystems push the
others via `Metrics.RecordIQUnderrun`, `RecordUSBReconnect`,
`RecordDecodeError`, and `SetSDRAttached`.

## Graceful shutdown

The daemon (`cmd/gophertrunk run`) installs a `signal.NotifyContext`
for `SIGINT` / `SIGTERM`. Each long-lived component (`api.Server`,
`metrics.Metrics`, `storage.CallLog`, `trunking.Engine`) accepts a
context and a `Close()` method, so the shutdown sequence is:

1. Signal cancels the root context.
2. Each component returns from `Run` / cleans up its bus subscription.
3. The engine drains active calls — every active `ActiveCall` gets a
   final `events.KindCallEnd` with `EndReasonNormal` so the call log
   captures it before the database closes.

The pre-built `engine.Engine.shutdown()` already implements step 3;
the wiring for steps 1-2 lives in `cmd/gophertrunk` (and lands in
the demod-pipeline composer follow-up).

## Docker

The repository ships a multi-stage `Dockerfile`:

```sh
docker build -t gophertrunk:dev .
```

For a single-daemon, single-dongle deployment use the
[`docker-compose.yml`](../docker-compose.yml) at the repo root. It
bind-mounts `config.yaml`, `recordings/`, and `calls.db` from the
host so data persists across container restarts.

### USB pass-through

RTL-SDR dongles need three things to work inside a container:

1. **Host udev rules** that grant a non-root group access to the
   device node. Place this at `/etc/udev/rules.d/20-rtlsdr.rules`:

   ```
   SUBSYSTEM=="usb", ATTRS{idVendor}=="0bda", ATTRS{idProduct}=="2838", \
       MODE="0660", GROUP="plugdev"
   ```

   Reload with `sudo udevadm control --reload && sudo udevadm trigger`,
   then unplug / replug the dongle.

2. **DVB blacklist** so the kernel doesn't claim the device first:

   ```
   # /etc/modprobe.d/blacklist-dvb_usb_rtl28xxu.conf
   blacklist dvb_usb_rtl28xxu
   ```

3. **Container privileges:**
   - Map the device node into the container with `devices:` in compose
     (or `--device` with `docker run`).
   - Use `group_add:` to put the container's user in the same group
     as the host's udev rule (`plugdev` = GID 46 on Debian; check
     `getent group plugdev`).
   - Add `cap_add: DAC_OVERRIDE` so the non-root user inside the
     container can open the device node.

The provided `docker-compose.yml` does all of this; verify the
`devices:` path matches your `lsusb` output.

## Validation

After `docker compose up -d`, smoke-test:

```sh
curl -s http://localhost:8080/api/v1/health
curl -s http://localhost:8080/api/v1/version
curl -s http://localhost:8080/metrics | grep gophertrunk_build_info
docker exec gophertrunk gophertrunk sdr list
```

The last command should list the dongle by index, serial, tuner type,
and supported gains. If it reports no devices, the udev rules and / or
USB pass-through are misconfigured — check `dmesg` on the host for
DVB driver claims, and `ls -l /dev/bus/usb/...` from inside the
container.

## Integration test (deferred)

The plan calls for a `make integration` target that replays 30 minutes
of recorded IQ through the full stack and diffs the resulting event
totals against a golden manifest. That replay path needs the
demod-pipeline composer (gated on the Phase 3/4/5 channel-coding
deferrals plus the per-call demod chain) before it can run end-to-end,
so the integration target is not yet wired up. The metrics, Docker
assets, and graceful-shutdown plumbing here are independent and ready
to use today.
