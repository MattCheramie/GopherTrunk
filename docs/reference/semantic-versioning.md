---
slug: semantic-versioning
title: Semantic versioning
entry_type: concept
category: testing-delivery
description: Semantic versioning (SemVer) is a MAJOR.MINOR.PATCH numbering convention in which the version number communicates the nature of each change — breaking, backward-compatible feature, or bug fix.
keywords: semantic versioning, SemVer, MAJOR MINOR PATCH, version number, breaking change, backward compatible, release versioning, dependency ranges, tags
aka: [semantic versioning "SemVer"]
autolink: true
infobox:
  - { label: Category, value: "Versioning convention" }
  - { label: Format, value: "MAJOR.MINOR.PATCH (e.g. 2.4.1)" }
  - { label: MAJOR, value: "Breaking change" }
  - { label: MINOR, value: "Backward-compatible feature" }
  - { label: PATCH, value: "Backward-compatible bug fix" }
see_also: [package-manager, api, build-systems, ci-cd, version-control]
related_lessons:
  - { title: "Releases & tags", url: /learn/git/releases-and-tags/ }
related_reading:
  - { title: "Build in the Open, Part 11: Releases, pre-release, SemVer & changelogs", url: /blog/tutorials/build-in-the-open-11-releases-prerelease-semver-changelogs/ }
cite_urls:
  - https://semver.org/
  - https://en.wikipedia.org/wiki/Software_versioning
---

**Semantic versioning** (SemVer) is a `MAJOR.MINOR.PATCH` numbering convention in which
the version number itself communicates the nature of each change — breaking,
backward-compatible feature, or bug fix.[^semver]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A version number 2.4.1 broken into MAJOR for breaking changes, MINOR for features, and PATCH for fixes." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="230" y="38" font-size="24" fill="currentColor">2 . 4 . 1</text>
    <line x1="150" y1="46" x2="150" y2="62" stroke="currentColor" stroke-width="1"/><text x="150" y="78">MAJOR</text><text x="150" y="92" font-size="8">breaking</text>
    <line x1="230" y1="46" x2="230" y2="62" stroke="currentColor" stroke-width="1"/><text x="230" y="78">MINOR</text><text x="230" y="92" font-size="8">feature</text>
    <line x1="310" y1="46" x2="310" y2="62" stroke="currentColor" stroke-width="1"/><text x="310" y="78">PATCH</text><text x="310" y="92" font-size="8">bug fix</text>
  </g>
</svg>
<figcaption>Each position in MAJOR.MINOR.PATCH signals a different kind of change.</figcaption>
</figure>

## The rules

Under SemVer you bump exactly one part of the version per release, by the nature of the
change:

| Bump | When | Example |
| --- | --- | --- |
| PATCH | Backward-compatible bug fix | 1.4.2 → 1.4.3 |
| MINOR | Backward-compatible new feature | 1.4.3 → 1.5.0 |
| MAJOR | Breaking, backward-incompatible change | 1.5.0 → 2.0.0 |

A user reading `1.5.0 → 1.5.1` knows it is a safe update; `1.x → 2.0` warns them to read
the release notes. Pre-`1.0.0` versions are treated as unstable, and suffixes like
`-rc.1` or `-beta` mark pre-releases. That single convention prevents a great deal of
confusion about whether an upgrade is safe.[^semver]

## Why it matters

SemVer turns a version string into a *contract* about compatibility, which is what makes
automated [dependency management](/reference/package-manager/) possible: a manifest can
safely request "compatible with 1.4" and let the resolver accept newer 1.x releases but
not a breaking 2.0. The "breaking change" that triggers a MAJOR bump is precisely a change
to a published [API](/reference/api/) — its operations, types, or behavior — so SemVer and
good API design go hand in hand.

## In practice

A release is usually a tagged commit in [version control](/reference/version-control/),
and a [CI/CD](/reference/ci-cd/) pipeline often builds and publishes the versioned
artifacts automatically when a tag is pushed. The version baked into the
[build](/reference/build-systems/) and printed by the program lets users and maintainers
identify exactly which release they are running.[^wiki]

## Sources

[^semver]: [Semantic Versioning 2.0.0](https://semver.org/) — the authoritative SemVer specification defining MAJOR.MINOR.PATCH and pre-release suffixes.
[^wiki]: [Software versioning](https://en.wikipedia.org/wiki/Software_versioning) — Wikipedia, for versioning schemes and release-tagging practice.
