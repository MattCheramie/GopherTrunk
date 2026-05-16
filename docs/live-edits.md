---
layout: page
title: Live config editing
description: Per-field matrix for PATCH /api/v1/settings — what hot-reloads, what needs a daemon restart
nav_group: Operate
---

# Live config editing

`PATCH /api/v1/settings` writes operator edits straight to
`config.yaml` (preserving comments and formatting) and hot-applies
the in-memory subset the daemon knows how to change without a
restart. The response carries an `applied` list (took effect now)
and a `restart_required` list (written to disk but waiting for a
daemon bounce).

Both the TUI's Settings panel and the web SPA's `/settings` route
drive this endpoint. The matrix below reflects what each field
does today; expect the `applied` column to grow as more subsystems
expose hot-reload surfaces.

## Field matrix

| YAML field                     | Hot-reload? | Notes                                                         |
|--------------------------------|-------------|---------------------------------------------------------------|
| `log.level`                    | yes         | Logger level switch.                                          |
| `log.format`                   | restart     | Handler is constructed at startup.                            |
| `api.http_addr`                | restart     | Listener is bound once.                                       |
| `api.grpc_addr`                | restart     | Listener is bound once.                                       |
| `api.auth.mode`                | restart     | Middleware constructed at startup.                            |
| `api.tls_cert` / `api.tls_key` | restart     | Read at bind time. Pre-flight validates parse cleanly.        |
| `audio.enabled`                | restart     | Backend handle is opened once.                                |
| `audio.device`                 | restart     | Backend handle is opened once.                                |
| `audio.volume`                 | yes         | Software gain — instant.                                      |
| `audio.muted`                  | yes         | Software gain bypass — instant.                               |
| `audio.buffer_ms`              | restart     | Buffer allocated at startup.                                  |
| `recordings.dir`               | restart     | Recorder constructed at startup.                              |
| `recordings.sample_rate`       | restart     | Recorder constructed at startup.                              |
| `recordings.write_raw`         | yes         | Routes through the recorder gate.                             |
| `retention.call_log_days`      | restart     | Sweeper constructed at startup.                               |
| `retention.files_days`         | restart     | Sweeper constructed at startup.                               |
| `retention.interval`           | restart     | Ticker constructed at startup.                                |
| `sdr.sample_rate`              | restart     | SDR pool opened once.                                         |
| `scanner.scan_mode`            | yes         | Engine atomically swaps the mode.                             |
| `scanner.manual_tune_enabled`  | restart     | Conventional scanner decision baked at startup.               |
| `scanner.cc_hunt.*`            | restart     | Supervisor constructed once.                                  |
| `storage.path`                 | restart     | SQLite handle opened once.                                    |
| `storage.cc_cache_file`        | restart     | Cache opened once.                                            |
| `metrics.enabled`              | restart     | Collector wired at startup.                                   |

## Wire format

```
PATCH /api/v1/settings
Content-Type: application/json

{
  "audio_volume": 0.5,
  "scanner_scan_mode": "list"
}
```

Response:

```json
{
  "applied": ["audio.volume", "scanner.scan_mode"],
  "restart_required": [],
  "config_path": "/etc/gophertrunk/config.yaml",
  "runtime": { ...full RuntimeDTO... }
}
```

## Concurrency + mtime guard

- A single `sync.Mutex` on the daemon's config writer serialises
  every `PATCH /api/v1/settings` and `POST /api/v1/import/.../commit`
  call so concurrent writes never tear the file.
- The writer stat's `config.yaml` before every write and refuses to
  clobber it when its `mtime` doesn't match what the writer last
  observed. If you edit `config.yaml` in `$EDITOR` while the daemon
  is running, the next live edit returns:
  ```
  config <path>: config.yaml was modified externally; reload the
  daemon to pick up the new file before editing again
  ```
  Send `SIGHUP` (POSIX) to re-load + apply your edits, or restart
  the daemon.

## SIGHUP

Sending `SIGHUP` to the daemon process reloads `config.yaml` and
applies the diff against the in-memory config. The hot-reloadable
fields above take effect immediately; restart-required fields are
logged so operators see what's still pending. Windows has no
SIGHUP — restart the daemon instead.

```sh
# Edit config.yaml in $EDITOR, then:
kill -HUP "$(pidof gophertrunk)"
# → "config reloaded: applied=2 restart_required=1 (applied:
#    audio.volume, scanner.scan_mode) (restart needed: sdr.sample_rate)"
```

## See also

- [Launcher overview]({{ '/launcher.html' | relative_url }}) —
  `gophertrunk -tui` / `-web` / `-headless` and the in-process UI
  flow.
- [Hardening]({{ '/hardening.html' | relative_url }}) — re-enable
  auth + CORS for hostile networks.
