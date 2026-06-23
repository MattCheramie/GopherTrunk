# GopherTrunk — guidance for Claude

GopherTrunk is a Go SDR trunking scanner/decoder (P25, DMR, NXDN, TETRA, …).
This file is standing guidance for AI-assisted work in this repo. Keep it short.

## Build & test

- `make vet test` — vet + unit tests; must be green before any commit.
- `make integration` — daemon/replay integration tests; run when the daemon,
  DSP, or replay path changed.
- Single package while iterating, e.g. `go test ./internal/scanner/ccdecoder/...`.

## Change scope (mirrors CONTRIBUTING.md)

- **Bug fix**: one narrow commit plus a regression test that **fails without the
  fix and passes with it**. If you can't write a test that fails first, you have
  not yet reproduced the bug — keep digging or ask for a reproduction, don't
  guess at a fix.
- **Feature / refactor**: design first; keep refactors out of behaviour-change PRs.

## Issue-closing policy

Closing an issue is a claim that the reported problem is gone. Do not make that
claim until it is verified. This policy exists because issue #764 was closed
twice on an unverified fix while the symptom was still live (see #771); a
PreToolUse hook (`.claude/hooks/guard-issue-close.py`) now asks for human
confirmation before any close-as-completed.

- **Never close an issue as completed until the fix is verified**: a failing-first
  regression test now passes **and** the reporter has confirmed it, or you have
  reproduced the original symptom and shown this change resolves it.
- **When you can't verify, leave it open.** Post a concise status comment saying
  what you found and what's blocking (e.g. needs the reporter's capture files),
  rather than closing.
- **Address the latest follow-up, not the original report.** Never re-post the
  initial fix description as a close justification — respond to the most recent
  comment specifically.
- **In PRs, prefer `Refs #N` over `Closes #N`** until the fix is verified, so a
  merge doesn't auto-close an unverified issue.
- Closing as `not_planned` or `duplicate` is fine and is not gated.

## DSP / replay notes (so the next investigation starts ahead)

- `gophertrunk replay -tune-hz` uses the single-channel
  `ccdecoder.Downconverter` (`internal/scanner/ccdecoder/ddc.go`), **not** the
  multi-tap wideband `DDCBank` (`internal/dsp/tuner/ddc.go`). They are separate
  paths — a fix to one does not touch the other. (This is what made the #764
  "fix" miss the #771 replay symptom.)
- Both down-converters normalise to the per-protocol channel rate (48 kHz for
  the 4800-baud C4FM family, 144 kHz for TETRA) and the receiver/AGC are sized
  from that output rate, so the decode path is **rate-invariant** to the capture
  rate. A symptom that only appears at a higher capture rate but reproduces in
  offline replay points at the *captured data* (front-end overload / intermod /
  gain staging), not the steady-state DSP — get the raw `.cfile` to reproduce.
  See `internal/scanner/ccdecoder/ddc_highrate_test.go`.
- #764 is now verified against the reporter's own captures and confirms the rule
  above. Mt Anakie (−812.5 kHz) replays at demod SNR ≈19.7 dB / EVM 7.4% from the
  2.5 MS/s capture (locks) but ≈9.5 dB / EVM 22.5% from the 10 MS/s capture (no
  lock). Decimating the 10 MS/s file 4:1 with an *independent* resampler and
  replaying it through the proven 2.5 MS/s path reproduces the SAME ≈9.5 dB — so
  the ~10 dB deficit is baked into the captured samples, not GT's DDC. Neither
  capture clips (both peak ≈−48 dBFS, so it is not overload/IMD), and the wideband
  FFT carrier SNR is actually *higher* at 10 MS/s — carrier-clean but
  modulation-degraded is the signature of front-end phase noise / reciprocal mixing
  at the Airspy's native 10 MS/s clock. `TestDownconverterSNRInvariantAcrossRate`
  in the file above pins this: a noisy channel reaches the receiver at the same
  in-channel SNR whether decoded natively at 10 MS/s or decimated to 2.5 MS/s.
