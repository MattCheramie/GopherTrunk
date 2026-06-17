#!/usr/bin/env python3
"""Re-date a blog post series onto consecutive days.

GopherTrunk's docs site (docs/) dates posts by filename
(`YYYY-MM-DD-slug.md`) and hides future-dated posts until a build runs on or
after their date (`future: false` in docs/_config.yml). A daily cron in
.github/workflows/pages.yml then rebuilds the site every day, so a series
committed all at once drips out one post per day, including weekends.

This helper assigns those dates for you. Give it a start date and the posts in
reading order; it rewrites each filename's `YYYY-MM-DD-` prefix to consecutive
calendar days. It prints a dry-run table by default; pass --apply to perform the
renames with `git mv`.

Examples:
    # Preview: schedule a series starting 2026-07-06, in part order
    scripts/schedule-series.py 2026-07-06 docs/_posts/*sdr-internals*.md

    # Apply the renames (uses `git mv` so history follows)
    scripts/schedule-series.py --apply 2026-07-06 docs/_posts/*sdr-internals*.md

Files are processed in the order given on the command line, so glob/sort them
into reading order first (the default shell glob sorts lexically, which matches
zero-padded part numbers like -01-, -02-, ...).
"""

from __future__ import annotations

import argparse
import datetime as dt
import os
import re
import subprocess
import sys

DATE_PREFIX = re.compile(r"^(\d{4})-(\d{2})-(\d{2})-(.+)$")


def consecutive_dates(start: dt.date, count: int) -> list[dt.date]:
    """`count` consecutive calendar days starting at `start`."""
    return [start + dt.timedelta(days=i) for i in range(count)]


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(
        description="Re-date a post series onto consecutive calendar days.",
    )
    parser.add_argument("start", help="first publish date, YYYY-MM-DD")
    parser.add_argument("files", nargs="+", help="post files in reading order")
    parser.add_argument(
        "--apply",
        action="store_true",
        help="perform the renames (default is a dry run)",
    )
    args = parser.parse_args(argv)

    try:
        start = dt.date.fromisoformat(args.start)
    except ValueError:
        parser.error(f"start date {args.start!r} is not valid YYYY-MM-DD")

    dates = consecutive_dates(start, len(args.files))

    renames: list[tuple[str, str]] = []
    for path, date in zip(args.files, dates):
        directory, name = os.path.split(path)
        m = DATE_PREFIX.match(name)
        slug = m.group(4) if m else name  # tolerate files without a date prefix
        new_name = f"{date.isoformat()}-{slug}"
        renames.append((path, os.path.join(directory, new_name)))

    width = max((len(os.path.basename(src)) for src, _ in renames), default=0)
    print(f"{'date':<10}  {'weekday':<9}  {'file':<{width}}  ->  new name")
    for (src, dst), date in zip(renames, dates):
        print(
            f"{date.isoformat():<10}  {date.strftime('%A'):<9}  "
            f"{os.path.basename(src):<{width}}  ->  {os.path.basename(dst)}"
        )

    if not args.apply:
        print("\n(dry run — re-run with --apply to perform the renames)")
        return 0

    for src, dst in renames:
        if src == dst:
            continue
        if os.path.exists(dst):
            print(f"error: target already exists: {dst}", file=sys.stderr)
            return 1
        try:
            subprocess.run(["git", "mv", src, dst], check=True)
        except (subprocess.CalledProcessError, FileNotFoundError):
            # Fall back to a plain rename if git isn't available / file untracked.
            os.rename(src, dst)
    print("\nDone. Review with `git status` and commit when ready.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
