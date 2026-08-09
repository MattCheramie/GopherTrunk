#!/usr/bin/env python3
"""Guard that every internal link on the docs site actually resolves.

Run against a built Jekyll site:

    python3 .github/scripts/check_internal_links.py _site

``check_site_pages.py`` proves every *built* page is reachable from the site
map. This script proves the complementary invariant: every *outbound* internal
link a built page emits points at something that exists. It exists because the
Google Search Console 404 report (Aug 2026) was entirely broken internal links —
bare relative lesson links that resolved as children of a trailing-slash
permalink (``/learn/<m>/<a>/<b>/``), a ``/launcher/`` link to a page that builds
to ``/launcher.html``, and legacy URLs with no redirect. None of those are
caught by the sitemap guard, because the *target* was simply missing.

What counts as a valid target:

  * any file in the build (``/foo.html``, ``/assets/x.png``), including the
    extensionless (``/foo``) and directory (``/foo/``) forms GitHub Pages serves;
  * a ``jekyll-redirect-from`` stub (those are real files in the build);
  * a **scheduled** post — one dated in the future and therefore excluded from
    this build by ``future: false``. Its URL is computed from ``docs/_posts``
    front matter so the intentional series drip (a published Part N linking the
    not-yet-published Part N+1) does not fail CI, while a genuine typo/wrong
    slug still does.

Ignored: fragments, query strings, ``mailto:``/``tel:``/``javascript:``/``data:``
and any absolute ``http(s)://`` or protocol-relative (``//host``) URL.

Exits non-zero with a per-page report on any broken link.
"""

import os
import re
import sys
from urllib.parse import unquote, urldefrag

# Where the un-built (repo) posts live, relative to repo root. Used only to
# learn the URLs of scheduled future posts that this build intentionally omits.
POSTS_DIR = os.path.join("docs", "_posts")

# Pages we never parse for outbound links.
SKIP_PARSE_URLS = {"/404.html"}

HREF_RE = re.compile(r'(?:href|src)\s*=\s*["\']([^"\']+)["\']', re.IGNORECASE)
REFRESH_RE = re.compile(r'http-equiv=["\']?refresh', re.IGNORECASE)
# Sample markup shown *inside* a code block is not a navigable link — a prose
# example like `<audio src="…/audio/stream">` renders as escaped text within
# <code>/<pre>, keeping its literal src="…" attribute, which HREF_RE would
# otherwise mis-read as a real (and broken) internal link. Strip these regions
# before scanning so example snippets never register as links.
CODE_BLOCK_RE = re.compile(r"<(pre|code)\b[^>]*>.*?</\1>", re.IGNORECASE | re.DOTALL)
POST_FILENAME_RE = re.compile(r"^\d{4}-\d{2}-\d{2}-(?P<slug>.+)\.(?:md|markdown|html)$")


def norm(path):
    """Drop fragment + query and percent-decode to a comparable URL path."""
    path, _frag = urldefrag(path)
    path = path.split("?", 1)[0]
    return unquote(path)


def is_external(href):
    lower = href.strip().lower()
    return (
        lower.startswith(("http://", "https://", "//", "mailto:", "tel:",
                          "javascript:", "data:"))
    )


def file_to_urls(rel):
    """Every URL form GitHub Pages will serve a given built file at."""
    rel = rel.replace(os.sep, "/")
    urls = {"/" + rel}
    if rel == "index.html":
        urls.add("/")
    elif rel.endswith("/index.html"):
        base = "/" + rel[: -len("index.html")]   # /foo/
        urls.add(base)
        urls.add(base.rstrip("/"))               # /foo
    if rel.endswith(".html") and not rel.endswith("/index.html"):
        urls.add("/" + rel[: -len(".html")])     # /foo (extensionless)
    return urls


def collect_servable(site_dir):
    servable = set()
    for root, _dirs, files in os.walk(site_dir):
        for name in files:
            rel = os.path.relpath(os.path.join(root, name), site_dir)
            servable.update(file_to_urls(rel))
    return servable


def read_front_matter(text):
    """Return the raw YAML front-matter block (between the first two ---)."""
    if not text.startswith("---"):
        return ""
    end = text.find("\n---", 3)
    return text[3:end] if end != -1 else ""


def scheduled_post_urls(repo_root):
    """URLs of every post in docs/_posts, from the /blog/:categories/:title/
    permalink. Already-built posts are a harmless superset; future-dated posts
    are the ones this adds that the build itself omits."""
    urls = set()
    posts = os.path.join(repo_root, POSTS_DIR)
    if not os.path.isdir(posts):
        return urls
    for name in os.listdir(posts):
        m = POST_FILENAME_RE.match(name)
        if not m:
            continue
        slug = m.group("slug")
        fm = read_front_matter(open(os.path.join(posts, name),
                                     encoding="utf-8", errors="replace").read())
        category = None
        for line in fm.splitlines():
            s = line.strip()
            cm = re.match(r"(?:category|categories):\s*(.+)$", s)
            if cm and category is None:
                val = cm.group(1).strip().strip("[]").strip()
                # first category only (matches :categories path head)
                category = val.split(",")[0].strip().strip("\"'")
            sm = re.match(r"slug:\s*(.+)$", s)
            if sm:
                slug = sm.group(1).strip().strip("\"'")
        cat = (category or "uncategorized").strip()
        urls.add("/blog/%s/%s/" % (cat, slug))
    return urls


def resolve(page_url, href):
    """Resolve a possibly-relative href against the page's URL path."""
    if href.startswith("/"):
        return norm(href)
    base = page_url if page_url.endswith("/") else page_url.rsplit("/", 1)[0] + "/"
    joined = base + href
    # Collapse ./ and ../ segments.
    parts = []
    for seg in joined.split("/"):
        if seg in ("", "."):
            continue
        if seg == "..":
            if parts:
                parts.pop()
            continue
        parts.append(seg)
    trailing = "/" if joined.endswith("/") else ""
    return norm("/" + "/".join(parts) + trailing)


def main():
    site_dir = sys.argv[1] if len(sys.argv) > 1 else "_site"
    if not os.path.isdir(site_dir):
        sys.exit("ERROR: build directory %r not found." % site_dir)

    servable = collect_servable(site_dir)
    servable |= scheduled_post_urls(os.getcwd())

    broken = {}   # source page url -> sorted list of (href, resolved)
    checked = 0

    for root, _dirs, files in os.walk(site_dir):
        for name in files:
            if not name.endswith(".html"):
                continue
            fpath = os.path.join(root, name)
            page_url = norm(next(iter(
                u for u in file_to_urls(os.path.relpath(fpath, site_dir))
                if u.endswith("/") or u.endswith(".html"))))
            if page_url in SKIP_PARSE_URLS:
                continue
            html = open(fpath, encoding="utf-8", errors="replace").read()
            if REFRESH_RE.search(html):   # redirect stub — nothing to check
                continue
            # Ignore href/src that appear inside code samples (see CODE_BLOCK_RE).
            scan_html = CODE_BLOCK_RE.sub("", html)
            for href in HREF_RE.findall(scan_html):
                href = href.strip()
                if not href or href.startswith("#") or is_external(href):
                    continue
                target = resolve(page_url, href)
                checked += 1
                cands = {target}
                if not target.endswith("/"):
                    cands.add(target + "/")
                if not any(c in servable for c in cands):
                    broken.setdefault(page_url, set()).add((href, target))

    if broken:
        total = sum(len(v) for v in broken.values())
        print("Internal link check FAILED: %d broken link(s) on %d page(s).\n"
              % (total, len(broken)))
        for page in sorted(broken):
            print("  %s" % page)
            for href, target in sorted(broken[page]):
                print("      %-45s -> %s" % (href, target))
        sys.exit(1)

    print("Internal link check passed: %d internal links, all resolve." % checked)


if __name__ == "__main__":
    main()
