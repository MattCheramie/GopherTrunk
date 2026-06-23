#!/usr/bin/env python3
"""Guard that every page on the docs site stays in the site map.

Run against a built Jekyll site:

    python3 .github/scripts/check_site_pages.py _site

It enforces two invariants so a newly added page can never silently fall out
of the site map:

  1. Every rendered content page in the build is present in the crawl
     sitemap (``sitemap.xml``). Catches an accidental ``sitemap: false`` or a
     page that Jekyll didn't render (e.g. missing front matter).

  2. Every URL in ``sitemap.xml`` is linked from the human-readable HTML site
     map (``/sitemap/``). Catches an orphan page that exists and is crawlable
     but isn't reachable from the on-site map.

Intentional exclusions (the search page, 404, redirect stubs, and non-HTML
assets) are skipped. Exits non-zero with a report on any violation.
"""

import os
import re
import sys
from xml.etree import ElementTree

# Paths that are deliberately not in the crawl sitemap or the HTML site map.
# Keep this list tiny and explicit — it is the only escape hatch.
EXCLUDED_PATHS = {
    "/404.html",
    "/search/",
    "/sitemap/",  # the HTML site map itself; it need not list itself
}


def normalize(path):
    """Strip query/fragment and collapse to a comparable URL path."""
    path = path.split("#", 1)[0].split("?", 1)[0]
    return path or "/"


def file_to_url(site_dir, file_path):
    """Map a built file to the URL path jekyll-sitemap would emit."""
    rel = os.path.relpath(file_path, site_dir).replace(os.sep, "/")
    if rel == "index.html":
        return "/"
    if rel.endswith("/index.html"):
        return "/" + rel[: -len("index.html")]
    return "/" + rel


def is_redirect_stub(html):
    """jekyll-redirect-from emits a tiny page with a refresh meta tag."""
    return bool(re.search(r'http-equiv=["\']?refresh', html, re.IGNORECASE))


def collect_content_pages(site_dir):
    """All rendered HTML content pages, keyed by URL path."""
    pages = {}
    for root, _dirs, files in os.walk(site_dir):
        for name in files:
            if not name.endswith(".html"):
                continue
            fpath = os.path.join(root, name)
            url = normalize(file_to_url(site_dir, fpath))
            if url in EXCLUDED_PATHS:
                continue
            with open(fpath, "r", encoding="utf-8", errors="replace") as fh:
                html = fh.read()
            if is_redirect_stub(html):
                continue
            pages[url] = fpath
    return pages


def parse_sitemap_xml(site_dir):
    """URL paths listed in sitemap.xml (host stripped)."""
    xml_path = os.path.join(site_dir, "sitemap.xml")
    if not os.path.exists(xml_path):
        sys.exit("ERROR: sitemap.xml not found — is jekyll-sitemap enabled?")
    tree = ElementTree.parse(xml_path)
    ns = {"sm": "http://www.sitemaps.org/schemas/sitemap/0.9"}
    paths = set()
    for loc in tree.iterfind(".//sm:url/sm:loc", ns):
        url = (loc.text or "").strip()
        # Strip scheme + host -> path.
        path = re.sub(r"^https?://[^/]+", "", url)
        paths.add(normalize(path))
    return paths


def parse_html_sitemap_links(site_dir):
    """Internal hrefs linked from the HTML site map page."""
    html_path = os.path.join(site_dir, "sitemap", "index.html")
    if not os.path.exists(html_path):
        sys.exit("ERROR: /sitemap/ HTML page not found — expected "
                 "docs/sitemap-page.md to build to sitemap/index.html.")
    with open(html_path, "r", encoding="utf-8", errors="replace") as fh:
        html = fh.read()
    links = set()
    for href in re.findall(r'href=["\']([^"\']+)["\']', html):
        if href.startswith("/"):
            links.add(normalize(href))
    return links


def main():
    site_dir = sys.argv[1] if len(sys.argv) > 1 else "_site"
    if not os.path.isdir(site_dir):
        sys.exit("ERROR: build directory %r not found." % site_dir)

    content = collect_content_pages(site_dir)
    sitemap_urls = parse_sitemap_xml(site_dir)
    html_links = parse_html_sitemap_links(site_dir)

    errors = []

    # 1. Every content page must be in the crawl sitemap.
    missing_from_xml = sorted(set(content) - sitemap_urls)
    if missing_from_xml:
        errors.append(
            "Pages built but missing from sitemap.xml (accidental "
            "`sitemap: false` or unrendered page?):\n  "
            + "\n  ".join(missing_from_xml)
        )

    # 2. Every crawlable URL must be linked from the HTML site map.
    orphan = sorted(
        u for u in sitemap_urls
        if u not in EXCLUDED_PATHS and u not in html_links
    )
    if orphan:
        errors.append(
            "Pages in sitemap.xml not linked from the HTML site map "
            "(/sitemap/) — add them to docs/sitemap-page.md or its data "
            "source:\n  " + "\n  ".join(orphan)
        )

    if errors:
        print("Site map check FAILED:\n")
        print("\n\n".join(errors))
        sys.exit(1)

    print("Site map check passed: %d content pages, %d sitemap.xml URLs, "
          "%d links on /sitemap/." % (len(content), len(sitemap_urls),
                                      len(html_links)))


if __name__ == "__main__":
    main()
