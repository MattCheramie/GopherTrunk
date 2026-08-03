#!/usr/bin/env bash
# Synthesize docs/index.md (the Pages landing page) from README.md.
#
# Shared by .github/workflows/pages.yml (the deploy) and docs-check.yml (the PR
# check) so the two builds are byte-identical — a PR must build the same homepage
# production serves, or the internal-link check would see different links on each.
#
# Five transforms:
# 1. docs/assets/foo -> /assets/foo (docs/ is the Jekyll source root, so the
#    asset publishes at /assets/foo).
# 2. ](docs/foo.md) -> ](/foo.html) so README cross-links resolve on Pages.
# 3/4. Remaining repo-relative links (CHANGELOG.md, LICENSE, CONTRIBUTING.md,
#    samples/…, etc.) point at files that live in the repo but NOT on the Pages
#    site, so on the homepage they'd 404. Rewrite them to absolute GitHub URLs —
#    tree/main for directories (trailing slash), blob/main for files. These run
#    AFTER rule 2, so the already-rewritten /foo.html links (leading `/`) and any
#    http(s):// links (the `[^):#]` class stops at the `:`) are left untouched.
# 5. Strip everything from the start of the README through the first `---`
#    horizontal-rule separator. That cuts the README hero (logo + H1 + tagline +
#    badges), already rendered by the home.html layout's hero block — without
#    this, the homepage double-H1s and Google penalizes multi-H1 documents.
set -euo pipefail

readme="${1:-README.md}"
out="${2:-docs/index.md}"

{
  # No `title:` front-matter — jekyll-seo-tag falls back to site.title for the
  # homepage <title>, the SEO-optimal "GopherTrunk · <site.description>".
  printf -- '---\nlayout: home\n---\n\n'
  sed -E \
    -e 's|docs/assets/|/assets/|g' \
    -e 's|\]\(docs/([a-z0-9_-]+)\.md(#[^)]*)?\)|](/\1.html\2)|g' \
    -e 's|\]\(([A-Za-z][^):#]*/)\)|](https://github.com/MattCheramie/GopherTrunk/tree/main/\1)|g' \
    -e 's|\]\(([A-Za-z][^):#]*)(#[^)]*)?\)|](https://github.com/MattCheramie/GopherTrunk/blob/main/\1\2)|g' \
    "$readme" \
    | sed '0,/^---$/d'
} > "$out"
