#!/usr/bin/env python3
"""Keep the hand-maintained release-version references in step with the tag.

Three files carry the current release version by hand (everything else —
the binary's ldflags, the release workflow, the installer, the docs site's
live download links — derives it from the git tag):

  * CHANGELOG.md                        newest ``## [vX.Y.Z] — YYYY-MM-DD`` heading
  * README.md                           the Quick Start ``VERSION=vX.Y.Z`` line
  * docs/_includes/latest-version.html  the hardcoded Jekyll fallback tag

They drifted for 41 consecutive tags (v0.7.2 .. v1.1.2 all sat under
``[Unreleased]`` while README pointed at v0.7.1), so this script is now the
single source of truth for where the version lives and CI/the release
workflow enforce it:

  release-refs.py check [--tag vX.Y.Z] [--no-git]
      Exit 1 unless CHANGELOG.md starts with ``## [Unreleased]``, its
      versioned headings are in strictly descending SemVer order, the
      README + docs fallback both name its newest *stable* heading, and
      that heading is not OLDER than the newest stable tag git knows
      about (``git ls-remote --tags origin``, falling back to local
      tags). Running ahead of the tags is fine — that is what preparing
      a release looks like — running behind is the drift this exists to
      catch. With --tag, additionally require that CHANGELOG.md has a
      heading for the tag (and, for a stable tag, that README/fallback
      name it).

  release-refs.py bump vX.Y.Z [--date YYYY-MM-DD]
      Idempotently bring the three files up to date for that tag:
      promote ``[Unreleased]`` to a dated heading (or insert a placeholder
      heading when nothing was recorded) unless one already exists, and —
      for a stable (non-hyphenated) tag — point README + fallback at it.

Run from anywhere; paths resolve relative to the repository root. Only the
standard library is used so it works on a bare CI runner.
"""
from __future__ import annotations

import argparse
import datetime as _dt
import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
CHANGELOG = ROOT / "CHANGELOG.md"
README = ROOT / "README.md"
FALLBACK = ROOT / "docs" / "_includes" / "latest-version.html"

TAG_RE = re.compile(r"^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?$")
HEADING_RE = re.compile(r"^## \[(v[^\]]+)\] — (\d{4}-\d{2}-\d{2})$")
README_RE = re.compile(r"^VERSION=(v\S+)$", re.M)
FALLBACK_RE = re.compile(r'\{%- assign ver = "(v[^"]+)" -%\}')
UNRELEASED = "## [Unreleased]"
PLACEHOLDER = (
    "No changelog entries were recorded for this tag; see the GitHub release notes."
)


def semver_key(tag: str):
    """Sort key: numeric core, then a stable tag ranks above any prerelease."""
    core, _, pre = tag[1:].partition("-")
    nums = tuple(int(x) for x in core.split("."))
    return (nums, 1 if not pre else 0, pre)


def is_stable(tag: str) -> bool:
    return "-" not in tag


def read(path: pathlib.Path) -> str:
    return path.read_text(encoding="utf-8")


def changelog_headings(text: str) -> list[tuple[str, str]]:
    """Every versioned heading as (tag, date), file order (newest first)."""
    out = []
    for line in text.split("\n"):
        m = HEADING_RE.match(line)
        if m:
            out.append((m.group(1), m.group(2)))
    return out


def single(regex: re.Pattern, text: str, what: str, errors: list[str]) -> str | None:
    found = regex.findall(text)
    if len(found) != 1:
        errors.append(f"{what}: expected exactly one version reference, found {len(found)}")
        return None
    return found[0]


def newest_git_stable_tag() -> str | None:
    """Newest stable vX.Y.Z tag on origin (or locally); None if unknowable."""
    names: list[str] = []
    for cmd in (["git", "ls-remote", "--tags", "--refs", "origin"], ["git", "tag", "-l"]):
        try:
            out = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True, timeout=60)
        except (OSError, subprocess.TimeoutExpired):
            continue
        if out.returncode != 0:
            continue
        names = [l.rsplit("refs/tags/", 1)[-1].strip() for l in out.stdout.splitlines() if l.strip()]
        if names:
            break
    stable = [n for n in names if TAG_RE.match(n) and is_stable(n)]
    return max(stable, key=semver_key) if stable else None


def check(tag: str | None, use_git: bool = True) -> int:
    errors: list[str] = []
    changelog = read(CHANGELOG)
    first_heading = next((l for l in changelog.split("\n") if l.startswith("## ")), "")
    if first_heading != UNRELEASED:
        errors.append(f"CHANGELOG.md: first section must be '{UNRELEASED}', got '{first_heading}'")
    headings = changelog_headings(changelog)
    if not headings:
        errors.append("CHANGELOG.md: no '## [vX.Y.Z] — YYYY-MM-DD' heading found")
    for (newer, _), (older, _) in zip(headings, headings[1:]):
        if not semver_key(newer) > semver_key(older):
            errors.append(f"CHANGELOG.md: headings out of order: {newer} is listed above {older}")
            break
    for t, _ in headings:
        if not TAG_RE.match(t):
            errors.append(f"CHANGELOG.md: heading '{t}' is not a vX.Y.Z tag")
    latest_stable = next((t for t, _ in headings if is_stable(t)), None)

    readme_ver = single(README_RE, read(README), "README.md", errors)
    fallback_ver = single(FALLBACK_RE, read(FALLBACK), str(FALLBACK.relative_to(ROOT)), errors)

    if latest_stable and readme_ver and readme_ver != latest_stable:
        errors.append(f"README.md VERSION={readme_ver} but CHANGELOG.md's newest stable release is {latest_stable}")
    if latest_stable and fallback_ver and fallback_ver != latest_stable:
        errors.append(f"docs fallback names {fallback_ver} but CHANGELOG.md's newest stable release is {latest_stable}")

    git_tag = newest_git_stable_tag() if (use_git and latest_stable) else None
    if use_git and latest_stable:
        if git_tag is None:
            print("release-refs: warning: could not list git tags; skipping the staleness check", file=sys.stderr)
        elif semver_key(latest_stable) < semver_key(git_tag):
            errors.append(
                f"stale: git's newest stable tag is {git_tag} but CHANGELOG.md/README/docs still document {latest_stable}"
            )

    if tag:
        if not TAG_RE.match(tag):
            errors.append(f"--tag {tag!r} is not a vX.Y.Z tag")
        elif tag not in {t for t, _ in headings}:
            errors.append(f"CHANGELOG.md has no '## [{tag}]' heading")
        elif is_stable(tag) and latest_stable != tag:
            errors.append(f"CHANGELOG.md's newest stable heading is {latest_stable}, not {tag}")

    if errors:
        for e in errors:
            print(f"release-refs: {e}", file=sys.stderr)
        hint = tag or git_tag or latest_stable or "vX.Y.Z"
        print(f"release-refs: fix with: python3 scripts/release-refs.py bump {hint}", file=sys.stderr)
        return 1
    print(f"release-refs: consistent — newest stable release {latest_stable}, README/docs fallback agree")
    return 0


def bump_changelog(text: str, tag: str, date: str) -> tuple[str, str]:
    lines = text.split("\n")
    if any(HEADING_RE.match(l) and HEADING_RE.match(l).group(1) == tag for l in lines):
        return text, f"CHANGELOG.md: '## [{tag}]' heading already present"
    try:
        u = lines.index(UNRELEASED)
    except ValueError:
        raise SystemExit(f"release-refs: CHANGELOG.md has no '{UNRELEASED}' section to promote")
    nxt = next((i for i in range(u + 1, len(lines)) if lines[i].startswith("## ")), len(lines))
    body = lines[u + 1:nxt]
    heading = f"## [{tag}] — {date}"
    if any(l.strip() for l in body):
        # Promote: the heading slides in above the accumulated bullets and a
        # fresh, empty [Unreleased] stays on top.
        new = lines[:u + 1] + ["", heading] + lines[u + 1:]
        note = f"CHANGELOG.md: promoted [Unreleased] to '{heading}'"
    else:
        new = lines[:nxt] + [heading, "", PLACEHOLDER, ""] + lines[nxt:]
        note = f"CHANGELOG.md: [Unreleased] was empty; inserted placeholder '{heading}'"
    return "\n".join(new), note


def bump(tag: str, date: str | None) -> int:
    if not TAG_RE.match(tag):
        print(f"release-refs: {tag!r} is not a vX.Y.Z tag", file=sys.stderr)
        return 2
    date = date or _dt.datetime.now(_dt.timezone.utc).strftime("%Y-%m-%d")
    notes = []

    text = read(CHANGELOG)
    new, note = bump_changelog(text, tag, date)
    notes.append(note)
    if new != text:
        CHANGELOG.write_text(new, encoding="utf-8")

    if is_stable(tag):
        for path, regex, render in (
            (README, README_RE, lambda t: f"VERSION={t}"),
            (FALLBACK, FALLBACK_RE, lambda t: f'{{%- assign ver = "{t}" -%}}'),
        ):
            text = read(path)
            if len(regex.findall(text)) != 1:
                print(f"release-refs: {path.relative_to(ROOT)}: expected exactly one version reference", file=sys.stderr)
                return 1
            new = regex.sub(render(tag), text)
            if new != text:
                path.write_text(new, encoding="utf-8")
                notes.append(f"{path.relative_to(ROOT)}: now {tag}")
            else:
                notes.append(f"{path.relative_to(ROOT)}: already {tag}")
    else:
        notes.append(f"{tag} is a prerelease; README/docs fallback keep pointing at the newest stable tag")

    for n in notes:
        print(f"release-refs: {n}")
    return check(tag)


def main(argv: list[str]) -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    c = sub.add_parser("check", help="verify the references agree (exit 1 if not)")
    c.add_argument("--tag", help="additionally require this tag to be the documented release")
    c.add_argument("--no-git", action="store_true", help="skip comparing against git's tags (offline/file-only check)")
    b = sub.add_parser("bump", help="bring the references up to date for a tag")
    b.add_argument("tag")
    b.add_argument("--date", help="release date for the changelog heading (default: today, UTC)")
    a = p.parse_args(argv)
    return check(a.tag, not a.no_git) if a.cmd == "check" else bump(a.tag, a.date)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
