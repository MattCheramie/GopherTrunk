---
layout: page
title: Cutting a release
description: How to tag, build, and publish a new GopherTrunk release end-to-end
nav_group: Reference
---

# Cutting a release

This is the maintainer-facing walkthrough for publishing a new
GopherTrunk version. The release workflows
(`.github/workflows/release.yml`, `.github/workflows/installer.yml`)
do almost all of the heavy lifting — this doc covers what humans
need to do before, during, and after.

## Pre-release checklist

Run these on a clean checkout of `main` at the commit you intend to
tag. Anything failing means the tag isn't ready.

- [ ] `git status` is clean and `git pull --ff-only origin main` is
      a no-op (you're at the tip of `main`).
- [ ] `gofmt -l .` prints nothing. CI gates this since v0.1.5; if it
      fails, run `gofmt -w .` and commit before tagging.
- [ ] `make lint` (= `go vet ./...`) clean.
- [ ] `go test -race -count=1 ./...` green. Slow but matches CI.
- [ ] `make integration` green (the dispatch table of per-protocol
      control-channel integration tests; see `Makefile`).
- [ ] `make release-dry-run VERSION=vX.Y.Z` produces a Linux binary
      under `dist/dry-run/gophertrunk` and `gophertrunk version`
      prints the expected version / commit / build time.
- [ ] `CHANGELOG.md` `[Unreleased]` section covers every merged PR
      since the previous tag. Cross-check with:
      ```sh
      git log <prev-tag>..main --oneline
      ```
      Any PR mentioned in that log but not in the CHANGELOG needs a
      bullet before tagging.
- [ ] No open issues blocking release on reference hardware
      (NESDR Smart v5, RTL-SDR Blog v3 / v4) — see
      [hardware.md](hardware.md). If something's broken on the
      reference hardware, document it in the release notes or hold
      the release.

## Tagging

1. **Promote `[Unreleased]` to a versioned section in CHANGELOG.md.**
   Rename `## [Unreleased]` to `## [vX.Y.Z] — YYYY-MM-DD` and insert
   a fresh empty `## [Unreleased]` block above it for the next
   cycle. Commit:
   ```sh
   git add CHANGELOG.md
   git commit -m "release: vX.Y.Z"
   git push origin main
   ```

2. **Create and push the annotated tag.** Annotated (not
   lightweight) so `git describe` produces a useful version string
   inside the binary's ldflags:
   ```sh
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```

3. **Watch the release workflow.** The push triggers
   `.github/workflows/release.yml`, which:
   - builds `gophertrunk.exe` (amd64 + arm64) and the Windows
     installer (Zadig-bundled, Inno Setup),
   - builds Linux + macOS tarballs for amd64 and arm64,
   - builds the web console (`gophertrunk-web/`) and stages it
     alongside every artifact,
   - computes `SHA256SUMS` over every file,
   - attaches everything to a new GitHub Release with notes
     auto-extracted from the matching `CHANGELOG.md` section.

   If any job fails, the release won't publish. Fix and re-tag with
   a fresh patch number — do not delete-and-recreate a tag that
   ever went public, since GitHub's tag deletion is not retroactive
   for clones.

## Post-release

- Update `README.md`'s Quick Start `VERSION=v…` snippet to the new
  tag.
- Bump anything under `docs/downloads.md` that hard-codes a
  version.
- Sanity-check the published `gophertrunk-<ver>-windows-amd64-setup.exe`
  on a real Windows 11 machine — install, run Zadig from the Start
  Menu, run `gophertrunk sdr list`, uninstall and confirm the
  cleanup prompt. The Windows installer has the most moving parts
  (Inno Setup script + Zadig bundle + uninstaller cleanup), so
  smoke-testing it post-release catches regressions the headless
  CI can't.
- Announce in whatever channels the project uses.

## Patch releases

If a fix needs to ship without all of `main`'s unreleased feature
work:

1. Branch off the previous tag: `git checkout -b release/vX.Y vX.Y.0`.
2. Cherry-pick the fix commits onto the release branch.
3. Bump the patch component in `CHANGELOG.md` and tag from the
   release branch (`git tag -a vX.Y.Z release/vX.Y`).
4. Push the branch + tag.

Don't tag patch releases off `main` if `main` contains unreleased
feature work — the patch would silently ship every unreleased
feature too.

## Rolling back

If a published release has to be pulled:

- **Don't delete the tag or the GitHub Release.** Edit the release
  description to mark it superseded and point to the replacement
  version.
- Tag a fresh patch release with the fix.
- Update `README.md` and `docs/downloads.md` to the new patch.
