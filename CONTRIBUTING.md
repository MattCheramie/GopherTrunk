# Contributing to GopherTrunk

Thanks for the interest. GopherTrunk is a pure-Go, zero-CGO digital
trunking scanner; the codebase is set up so that a small change is
easy to land cleanly and a big change is easy to break into
reviewable pieces.

## Quick start

```sh
git clone https://github.com/mattcheramie/gophertrunk.git
cd gophertrunk
make build         # Go-only — fast iteration; daemon shows a helpful 404 at / until the SPA is bundled
make dist          # SPA + Go — single binary that serves the operator web console at /
make test          # unit tests (fast, < 30 s)
make integration   # daemon end-to-end tests (no SDR required)
make vet           # static analysis
```

Use `make build` while iterating on Go code (no npm needed). Use
`make dist` when you want a daemon binary that serves the SPA at
`/` — it runs `make web-build` first so the `//go:embed all:dist`
snapshot picks up a real bundle.

You need Go 1.25+ — no C toolchain (`CGO_ENABLED=0` everywhere),
no `librtlsdr`, no `libusb`. macOS / Windows / Linux all build the
same way; the platform-specific code paths live behind
`//go:build` guards in `internal/sdr/rtlsdr/usb/` and
`internal/voice/player/`.

## How changes are scoped

Reviewers look for small, single-purpose commits. A useful rule of
thumb: if you can't summarise the change in one sentence in the
imperative mood, the change is doing more than one thing and
should be split.

- **Bug fix**: one commit, narrow diff, regression test that fails
  without the fix and passes with it.
- **New feature**: design in `/root/.claude/plans/` or a GitHub
  issue first if the change touches more than one package. Land
  the design first, then incremental PRs that each close one tier
  of the plan.
- **Refactor**: separate PR, never bundled with a behaviour
  change. The reviewer can confirm "no behaviour change" by
  diffing tests before / after.

## Conventions

The repository's house style is documented implicitly through the
existing code; a few items worth calling out:

- **Comments**: default to writing none. Add a comment when the
  *why* is non-obvious (a hidden constraint, a workaround for a
  specific bug, behaviour that would surprise a reader). Don't
  explain *what* the code does — well-named identifiers handle
  that.
- **Error wrapping**: use `fmt.Errorf("pkg: action: %w", err)` so
  callers can `errors.Is` / `errors.As` against sentinels.
- **Logging**: `log/slog` everywhere. Use structured key/value
  args, not `Sprintf`'d messages. Log at `INFO` for state
  transitions, `WARN` for recoverable surprises, `ERROR` for
  things the operator must act on.
- **Tests**: parallel where it's safe (`t.Parallel()`),
  table-driven for any function with more than two interesting
  inputs, `t.Helper()` on helper functions so failure locations
  surface correctly.
- **No unused imports / variables**: `go vet` and the Go compiler
  enforce this; PR CI fails if it slips through.
- **No `gofmt` violations**: run `gofmt -w` before committing
  (most editors do this automatically). The repo doesn't use
  `goimports`-only flows; default `gofmt` is the canonical
  formatter.

## Building & testing

| Target | Purpose |
| --- | --- |
| `make build` | Build `bin/gophertrunk` (Go-only — fast; daemon serves a helpful 404 at `/` until the SPA is bundled) |
| `make dist` | Build `bin/gophertrunk` with the operator console embedded (`make web-build` then `make build`) |
| `make test` | Unit tests, `-race`, `-count=1` |
| `make integration` | Daemon end-to-end test (no SDR) — exercises the full IQ-replay path |
| `make integration-cc-<proto>` | Per-protocol "lights up live trunked reception" check (P25 P1 / P25 P2 / DMR / NXDN / dPMR / EDACS / Motorola / LTR / MPT 1327 / TETRA / YSF) |
| `make test-dvsi` | DVSI hardware backend tests under `-tags dvsi` |
| `make test-airspy-real` | Opt-in Airspy R2/Mini hardware validation (enumerate → open → tune/sample-rate/gain → first IQ packet) |
| `make test-airspy-real-bias` | Same as `test-airspy-real` plus real-hardware bias-tee on/off validation |
| `make test-airspy-real-diag` | Real Airspy tests plus raw USB control-transfer probe (isolate transport vs driver failures) |
| `make vet` | `go vet ./...` |
| `make cross-build` | Static binaries for Linux / macOS / Windows × amd64 / arm64 |

CI runs `make test`, `make integration`, and `make test-dvsi` on
every PR. A green CI is required before merge.

Real Airspy validation is intentionally opt-in and never runs in CI.
The package-level test skips unless `GOPHERTRUNK_AIRSPY_REAL=1` is
set. Optional overrides:

- `GOPHERTRUNK_AIRSPY_REAL_SERIAL` — match one device by serial.
- `GOPHERTRUNK_AIRSPY_REAL_CENTER_HZ` — center frequency in Hz
  (default `144390000`).
- `GOPHERTRUNK_AIRSPY_REAL_RATE_HZ` — sample rate in Hz
  (default `2500000`).
- `GOPHERTRUNK_AIRSPY_REAL_GAIN_TENTH_DB` — gain in 0.1 dB, use
  `-1` for AGC (default `-1`).
- `GOPHERTRUNK_AIRSPY_REAL_BIAS_TEE` — set to `1` to also run the
  real-hardware bias-tee on/off test (leaves bias-tee off on exit).
- `GOPHERTRUNK_AIRSPY_REAL_DIAG` — set to `1` to also run a lower-level
  raw USB control-transfer probe (`receiver off` + `set sample type`).

### Capturing Windows USB traces with Wireshark (Airspy)

When a device works in other SDR apps but fails in GopherTrunk, we need
packet-level USB evidence to diff request order and setup fields.

1. Install **Wireshark** and **USBPcap** (USBPcap is offered during the
   Wireshark installer). Reboot if prompted.

1. Close SDR apps so only one app talks to the Airspy during each capture.

1. Open Wireshark **as Administrator**.

1. Start capture on the USBPcap interface for the controller/hub where the
   Airspy is attached (usually `\\.\USBPcap1`, `\\.\USBPcap2`, etc.).

1. In Wireshark, apply a display filter to focus on Airspy traffic:

  ```text
  usb.idVendor == 0x1d50 && usb.idProduct == 0x60a1
  ```

1. Record a short known-good capture (about 10-20 seconds): start capture,
   launch a working SDR app, perform open/init (and one start/stop if
   possible), then stop capture.

1. Record a short GopherTrunk capture (about 10-20 seconds): start capture,
   then run either:

    ```powershell
    $env:RTLSDR_DEBUG_USB='1'
    $env:RTLSDR_DEBUG_USB_CSV='1'
    $env:GOPHERTRUNK_AIRSPY_REAL='1'
    $env:GOPHERTRUNK_AIRSPY_REAL_DIAG='1'
    go test -count=1 -run TestRealHardware_USBControlTransferProbe ./internal/sdr/airspy -v
    ```

   or your normal daemon start path that reproduces the failure.

1. Save captures as `.pcapng` files with clear names, for example
   `airspy-good-app-init.pcapng` and `airspy-gophertrunk-init.pcapng`.

1. Export matching GopherTrunk USB trace lines from terminal output:
   human-readable lines prefixed `rtlsdr-usb [` and CSV lines prefixed
   `rtlsdr-usb-csv,`.

1. Attach all artifacts to the issue/PR: both `.pcapng` files, the
   `rtlsdr-usb-csv` log excerpt, the exact command used, and device serial
   plus Windows version.

Notes:

USB captures can include other peripherals on the same controller; review
before sharing and redact unrelated traffic if needed.

Keep captures short and focused on device open/init to simplify
request-by-request diffing.

## Sending a pull request

1. Fork the repository or create a branch named
   `<topic>/<short-description>` from `main` if you have push
   access.
2. Make the change. Include a regression test for any bug fix and
   a representative test for any new behaviour.
3. Update `README.md` if user-visible behaviour changed; update
   `docs/` for operational changes. Add a line to
   [`CHANGELOG.md`](CHANGELOG.md) under `## [Unreleased]`.
4. Run `make vet test integration` locally — they must all pass.
5. Open a PR. The body should explain *why* the change is needed
   (what's broken or missing), not just *what* it does (the diff
   covers that).
6. Respond to review comments by updating the branch (force-push
   is fine on feature branches; the maintainer squashes on merge
   to keep `main` history clean).

## Cutting a release

Releases are produced by [`.github/workflows/release.yml`](.github/workflows/release.yml),
triggered by pushing a SemVer tag (`vX.Y.Z`) or via the workflow
dispatch button in the GitHub Actions UI.

Before tagging a new release, rehearse the build locally so any
ldflags / packaging breakage surfaces before a tag is cut:

```sh
make release-dry-run VERSION=v0.99.0
```

…which produces `dist/dry-run/gophertrunk` with the supplied
`VERSION`, `COMMIT` (short SHA), and `BUILD_TIME` (UTC ISO-8601)
injected via `-ldflags`. The script then runs `./gophertrunk
version` against the built binary and writes a `SHA256SUMS` file —
match those values against what you expect the production release
to print before pushing the real tag.

The first production release should be a prerelease (e.g.
`v0.99.0`) so the full release workflow runs end-to-end against
the live GitHub Actions infrastructure before a v1.0.0 tag goes
out. Trigger via the **Actions → Release → Run workflow** button
with the version field set.

## Security issues

If you've found a vulnerability, please follow the disclosure
process in [`SECURITY.md`](SECURITY.md) instead of opening a
public issue.

## License

By contributing you agree that your contribution will be released
under the project's existing license (see
[`LICENSE`](LICENSE)). All commits should be authored under a
name you're comfortable having publicly attached to the project's
git history.
