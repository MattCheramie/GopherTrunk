---
layout: page
title: Hardening & operations
description: API auth, Prometheus, graceful shutdown, Docker, and RTL-SDR pass-through for production deployments
nav_group: Operate
---

# Hardening & operations

Operator playbook for running GopherTrunk in production: API
authentication, Prometheus metrics, graceful shutdown, the Docker
assets, and the RTL-SDR USB pass-through recipe.

## API authentication

> **Default flipped for closed-LAN deployments.** Empty
> `api.auth.mode` now resolves to `disabled` (was `auto`) and an
> empty `api.cors.allowed_origins` permits any origin. The daemon
> logs a loud warning at startup whenever those defaults take effect
> on a non-loopback bind — see the "Hostile networks" section below
> for the recipe to opt back in. Operators on closed networks who
> just want the TUI / SPA to work no longer need any explicit auth
> config.

Every HTTP mutation endpoint (end-call, talkgroup priority/lockout/
scan, retention sweep, tone-detector reset, scanner cockpit, audio
cockpit, manual tune, **settings PATCH**, **import upload/commit**) is
gated by `api.auth` in `config.yaml`.

### Policy modes

- **`disabled` (default — empty `mode` resolves here).** Wide-open
  mutations, no auth. Equivalent to the legacy `allow_mutations: true`
  behaviour; the daemon logs a warning at startup when bound to a
  non-loopback interface. Appropriate for closed-LAN single-host
  setups where the operator already has shell on the box.
- **`auto`.** Require a bearer token on non-loopback binds; bypass
  the check when bound to `127.0.0.1` / `::1`. Reasonable for
  single-host operator boxes — kernel-enforced reachability is a
  peer-cred proxy. The daemon refuses to start in `auto` mode on a
  public bind without a configured token.
- **`required`.** Every mutation request must carry a valid Bearer
  token, even from loopback. Use when the daemon shares a host with
  untrusted users.

### Hostile networks — opt back into the safer posture

```yaml
api:
  http_addr: "0.0.0.0:8080"
  auth:
    mode: "required"             # or "auto" for the loopback bypass
    token_file: "/etc/gophertrunk/api-token"
  cors:
    allowed_origins:
      - "http://laptop.local:5173"   # only your SPA host
  # TLS optional — see docs/tls.md
  tls_cert: "/etc/ssl/gophertrunk.crt"
  tls_key:  "/etc/ssl/gophertrunk.key"
```

### Generating a token

```sh
# 32 bytes of urandom, hex-encoded — 64 ASCII chars.
openssl rand -hex 32 > /etc/gophertrunk/api-token
chmod 600 /etc/gophertrunk/api-token
chown gophertrunk:gophertrunk /etc/gophertrunk/api-token
```

Reference the file from config:

```yaml
api:
  http_addr: "0.0.0.0:8080"
  auth:
    mode: "required"
    token_file: "/etc/gophertrunk/api-token"
```

The daemon re-reads `token_file` on every mutation request, so
rotation is a one-step `openssl rand` + file write — no SIGHUP, no
restart.

### Client usage

```sh
TOKEN=$(cat /etc/gophertrunk/api-token)

# Probe capability first — always open.
curl -sS http://daemon:8080/api/v1/mutations

# Mutation — Authorization header required.
curl -sS -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"reason":"manual"}' \
     http://daemon:8080/api/v1/calls/00000001/end
```

`GET /api/v1/mutations` reports `auth_mode` and `can_mutate` (plus a
`allow_mutations` legacy alias) so TUI / scripts can light up
write-side keybindings without probing real endpoints for 401.

### Trusted networks (LAN bypass)

If the daemon binds to a LAN address and you trust everything on that
segment, list the prefix under `auth.trusted_networks` and run in
`auto` mode:

```yaml
api:
  http_addr: "192.168.1.10:8080"
  auth:
    mode: "auto"
    trusted_networks:
      - "192.168.0.0/16"
```

The middleware honours `RemoteAddr` only — `X-Forwarded-For` is
intentionally ignored so the bypass isn't forgeable by a hostile
upstream proxy. If you front the daemon with an authenticating
reverse proxy (nginx, Caddy, Envoy), point its upstream at the daemon
on loopback and let the proxy handle auth; `auth.mode: auto` then
bypasses on the loopback hop.

### Migrating from `allow_mutations`

Existing configs with `allow_mutations: true` still work: the daemon
logs a deprecation warning at startup and maps the flag to
`auth.mode: disabled` (the wide-open legacy behaviour). Migrate to
explicit `auth.mode` at the next config edit:

```diff
 api:
   http_addr: "127.0.0.1:8080"
-  allow_mutations: true
+  auth:
+    mode: "auto"      # loopback-bypass; switch to required on public binds
```

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

### Connection-drain window

The HTTP server's `Shutdown` ctx is bounded at 30 s (was 5 s before).
The window covers in-flight SSE / WebSocket / per-call audio-stream
subscribers — those connections see a clean close instead of being
torn down mid-frame on a daemon restart. Static REST requests
complete in milliseconds either way; the extra headroom only
matters for streaming endpoints. gRPC uses
`grpc.Server.GracefulStop()`, which waits for every in-flight RPC
(including `AudioService.StreamAudio` subscribers) to finish or
voluntarily close.

## Transport encryption (TLS)

Both the HTTP API (REST + SSE + WebSocket) and the gRPC server
support optional TLS. Plain TCP stays the default — bearer-token
auth in `api.auth` is sufficient for loopback / trusted-LAN
deployments and TLS adds operational overhead (cert rotation, CA
choice). Public-bind operators should enable TLS.

Wire up by adding two paths to `api` in `config.yaml`:

```yaml
api:
  http_addr: ":8080"
  grpc_addr: ":50051"
  tls_cert: "/etc/gophertrunk/tls/cert.pem"
  tls_key:  "/etc/gophertrunk/tls/key.pem"
```

Both keys must be set together — setting one without the other is
a config error the daemon refuses to start with rather than
silently falling back to plain HTTP. The same cert / key pair
is used for both the HTTP and gRPC listeners. Generate a real
cert with `certbot`, your CA's tooling, or for testing:

```sh
openssl req -x509 -newkey rsa:2048 -nodes -days 365 \
    -keyout /etc/gophertrunk/tls/key.pem \
    -out    /etc/gophertrunk/tls/cert.pem \
    -subj "/CN=gophertrunk.example.com" \
    -addext "subjectAltName=DNS:gophertrunk.example.com,IP:10.0.0.1"
```

Cert rotation requires a daemon restart — `internal/api/server.go`'s
`ServeTLS` reads both files at start-up. SIGHUP / auto-reload is a
follow-up.

Once TLS is up, clients use `https://` and `grpc+tls://`:

```sh
curl --cacert /etc/gophertrunk/tls/cert.pem \
     -H "Authorization: Bearer $TOKEN" \
     https://gophertrunk.example.com:8080/api/v1/health

grpcurl -cacert /etc/gophertrunk/tls/cert.pem \
        gophertrunk.example.com:50051 \
        gophertrunk.v1.AudioService/StreamAudio
```

## Health endpoint diagnostics

`GET /api/v1/health` returns a JSON body shaped like:

```json
{
  "status":              "ok",
  "now":                 "2026-05-13T19:00:00Z",
  "version":             "v1.2.3",
  "pool_attached_count": 2,
  "active_calls":        1,
  "db_connected":        true,
  "metrics_enabled":     true,
  "auth_mode":           "auto"
}
```

The endpoint is always open (no token required) so liveness /
readiness probes can hit it from outside the auth boundary. Every
field beyond `status` and `now` is best-effort — a daemon
configured without a particular collaborator (no SDR pool, no
engine, no history DB) just reports the corresponding field at its
zero value rather than failing the request.

Suggested probe semantics:

| Probe type | Condition |
| --- | --- |
| Liveness | HTTP 200 + body decodes |
| Readiness | `status == "ok"` AND `pool_attached_count >= 1` AND `db_connected == true` |

The two-field readiness check distinguishes "the daemon process
is up" from "the daemon process is actually doing work" so
orchestrators (k8s, Nomad) can roll deployments cleanly.

## Timeouts and keep-alive

| Knob | Value | What it bounds |
| --- | --- | --- |
| `http.Server.ReadHeaderTimeout` | 10 s | Slowloris-style header send delays |
| `http.Server.ReadTimeout` | 30 s | Total request-read time (bounds slow uploads) |
| `http.Server.WriteTimeout` | 30 s | Per-request write window for non-streaming handlers; **disabled per-request** for SSE (`/api/v1/events`) and audio-streaming endpoints via `http.ResponseController.SetWriteDeadline(time.Time{})`; not enforced after WebSocket Upgrade hijacks the connection |
| `http.Server.IdleTimeout` | 120 s | Idle keep-alive socket lifetime |
| `grpc.KeepaliveParams.Time` | 30 s | Server pings idle clients after this much inactivity |
| `grpc.KeepaliveParams.Timeout` | 10 s | Server waits this long for a ping ack before closing the connection |
| `grpc.KeepaliveEnforcementPolicy.MinTime` | 5 s | Floors how often clients are allowed to ping the server (anti-flood) |
| `grpc.KeepaliveEnforcementPolicy.PermitWithoutStream` | `true` | Long-lived idle `StreamAudio` subscribers stay open through their keep-alive pings |

The bounds are conservative — every standard REST endpoint
completes in well under 30 s, and live streaming endpoints (SSE,
WebSocket, gRPC `StreamAudio`) opt out of HTTP `WriteTimeout`
explicitly so they aren't torn down mid-frame.

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

## Integration test

`make integration` boots the wired daemon end-to-end (no real SDR),
publishes a synthetic call grant on the events bus, and asserts every
component agrees: the call appears in the SQLite history, a `.wav`
lands under `<recordings>/<system>/<talkgroup>/`, `/metrics` reflects
`gophertrunk_calls_active 1`, and `CallEnd` ticks
`gophertrunk_calls_total{reason="normal"}`. The target runs on every
CI build alongside the unit tests.

A live-IQ replay variant (replay a recorded `.cfile` through the
real SDR mock + protocol decoders + composer chain and diff the
resulting event totals against a golden manifest) is future work —
it needs live grant publication from the protocol decoders, which is
gated on the channel-coding pieces still in flight (P25 trellis +
TSBK interleaver, DMR slot-type Hamming(20,8), NXDN SACCH FEC).
