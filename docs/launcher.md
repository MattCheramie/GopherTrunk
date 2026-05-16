# Daemon launcher

When `gophertrunk` (no subcommand) is invoked on an interactive
terminal, the daemon starts in the background and then prompts you
to pick a UI:

```
──
Daemon ready. How do you want to drive it?
  [1] TUI       (in-process operator console)
  [2] Web       (open the bundled SPA in your browser)
  [3] Headless  (keep running silent)
Choice [1-3, default 3]:
```

The launcher's purpose is to remove the "I started the daemon, now
what do I do?" friction that used to require running a second
process by hand. Whichever option you pick, the daemon keeps running
in the same process — quitting the TUI or closing the browser does
not stop the daemon. Send `SIGINT` / `SIGTERM` (or `Ctrl-C` on a TTY)
to shut down.

## Pre-selecting a mode

Three mutually-exclusive flags skip the prompt:

| Flag | Behaviour |
|------|-----------|
| `-tui` | Bring up the in-process TUI after the HTTP API binds. Requires `stdin`+`stdout` to be a TTY and `api.http_addr` to be configured. |
| `-web` | Locate the bundled `gophertrunk-web/` directory and open `index.html#server=<addr>` in the system browser. Requires `api.http_addr`. |
| `-headless` | Skip the prompt and keep the daemon silent. Same as the auto-default on a non-TTY stdin (systemd, Windows service, Docker, CI). |

```sh
gophertrunk -tui -config config.yaml
gophertrunk -web
gophertrunk -headless                 # explicit
gophertrunk </dev/null                # implicit headless (no TTY)
```

## How the TUI runs in-process

`-tui` and menu option `[1]` spawn `bubbletea` inside the daemon
process, talking to the daemon over the local HTTP API at
`http://127.0.0.1:<port>`. Daemon log output is redirected to a
temp file while the TUI owns the screen, so background log lines
never bleed onto the alt-screen canvas. On TUI exit (`q` /
`Ctrl-C`), `stderr` is restored and the daemon keeps running.

## How the web launcher works

`-web` searches the canonical locations for a bundled SPA:

1. `<dirname of executable>/gophertrunk-web/index.html`
2. `<dirname of executable>/web/dist/index.html` (dev tree)
3. `<dirname of executable>/../share/gophertrunk/web/index.html`
4. `<UserConfigDir>/gophertrunk/web/index.html`
5. `./web/dist/index.html` (running from the repo root)

The first one found is opened with a URL fragment that bootstraps
the SPA against the running daemon:

```
file:///path/to/index.html#server=http://localhost:8080
```

`xdg-open` is used on Linux, `open` on macOS, and `rundll32
url.dll,FileProtocolHandler` on Windows. Headless hosts that don't
have a display (SSH over a Pi without `$DISPLAY`) fall back to
printing the URL + asset path:

```
launcher: could not launch a browser on this host.
          Open this URL on a machine that has one:
            http://192.168.1.42:8080/
          Web SPA assets are at /opt/gophertrunk/gophertrunk-web/index.html
          (open index.html in a browser, then enter the URL above)
```

The daemon keeps running so a remote operator can open the URL from
their laptop / phone.

## Startup warnings

The launcher menu (and the headless path) prints any
non-fatal warnings the daemon collected during startup — for
example, an SDR pool that failed to open or a missing talkgroup CSV
— in yellow above the menu:

```
! SDR pool failed to open (no devices found) — no radios will demodulate
! talkgroup_file "tgs.csv" for system "P25_East" failed to load (open tgs.csv: no such file or directory) — calls on this system will have no alpha tags

──
Daemon ready. How do you want to drive it?
  ...
```

Those same warnings are surfaced on the runtime DTO
(`/api/v1/runtime` → `startup_warnings`) so the TUI's Dashboard and
the web SPA can pin them until dismissed.

## Live edits while a UI is up

The TUI's Settings panel and the SPA's `/settings` route call
`PATCH /api/v1/settings`. Edits land on `config.yaml` with comments
preserved; hot-reloadable knobs apply immediately and
restart-required ones are flagged in the response so the UI can
render a `restart required` badge. See
[`live-edits.md`](live-edits.md) for the full per-field matrix.

The TUI's Import panel and the SPA's `/import` route POST
multipart-encoded PDFs / CSVs to `/api/v1/import`. The daemon
parses each upload, returns a preview, and a follow-up commit
merges the result into `config.yaml` + refreshes the in-memory
talkgroup database. No daemon restart required.

Both PATCH /settings and the import commit serialise through a
single writer mutex so concurrent edits never tear the file. The
mtime guard refuses the write if `config.yaml` was modified
externally (e.g. you opened it in `$EDITOR`) since the daemon last
read it — restart the daemon to pick up your edits before issuing
another mutation.
