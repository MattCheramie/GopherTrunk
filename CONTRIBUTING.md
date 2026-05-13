# Contributing to GopherTrunk

Thanks for the interest. GopherTrunk is a pure-Go, zero-CGO digital
trunking scanner; the codebase is set up so that a small change is
easy to land cleanly and a big change is easy to break into
reviewable pieces.

## Quick start

```sh
git clone https://github.com/mattcheramie/gophertrunk.git
cd gophertrunk
make build         # produces ./bin/gophertrunk
make test          # unit tests (fast, < 30 s)
make integration   # daemon end-to-end tests (no SDR required)
make vet           # static analysis
```

You need Go 1.24+ — no C toolchain (`CGO_ENABLED=0` everywhere),
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
| `make build` | Build `bin/gophertrunk` (the daemon + CLI) |
| `make test` | Unit tests, `-race`, `-count=1` |
| `make integration` | Daemon end-to-end test (no SDR) — exercises the full IQ-replay path |
| `make integration-cc-<proto>` | Per-protocol "lights up live trunked reception" check (P25 P1 / P25 P2 / DMR / NXDN / dPMR / EDACS / Motorola / LTR / MPT 1327 / TETRA / YSF) |
| `make test-dvsi` | DVSI hardware backend tests under `-tags dvsi` |
| `make vet` | `go vet ./...` |
| `make cross-build` | Static binaries for Linux / macOS / Windows × amd64 / arm64 |

CI runs `make test`, `make integration`, and `make test-dvsi` on
every PR. A green CI is required before merge.

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
