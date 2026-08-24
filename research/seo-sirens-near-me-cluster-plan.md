# SEO/AISO plan: the "sirens near me" moment-of-need cluster

**Goal:** reach people at the exact moment they hear sirens or see emergency
lights and search for an explanation — an audience gophertrunk.org previously
had zero content for — and funnel a fraction of them into the existing scanner
education → hardware → GopherTrunk pipeline.

**Status:** phase 1 implemented (8 pages, nav, llms.txt, internal links).
This doc records the intent research, the architecture decisions, and the
measurement/expansion plan so future content work starts ahead.

## The search moment

When people hear sirens or see lights, they search *situationally and
urgently*, on a phone, in natural-language question form. Representative
queries (moment-of-need head + long tail):

| Intent | Queries | Page |
|---|---|---|
| Sirens right now | "sirens near me", "why are there sirens near me right now", "what's happening near me" | `/sirens-near-me/` (pillar) |
| Police presence | "police activity near me", "why are police on my street", "police in my neighborhood" | `/police-activity-near-me/` |
| Fire trucks, no fire | "fire trucks near me", "fire truck at neighbors house no fire", "why do fire trucks come for medical calls" | `/fire-trucks-near-me/` |
| Helicopter | "helicopter circling my neighborhood", "why is a helicopter over my house", "police helicopter near me" | `/helicopter-circling-my-neighborhood/` |
| Siren sound ID | "what do different sirens mean", "hi-lo siren meaning", "police vs ambulance siren" | `/what-do-siren-sounds-mean/` |
| Warning sirens | "why are tornado sirens going off", "siren going off right now no storm", "tornado siren test times" | `/tornado-sirens-going-off/` |
| Listen online | "listen to police scanner online free", "live police scanner near me", "Broadcastify" | `/listen-to-police-scanner-online/` |
| After the fact | "what happened near me last night", "sirens last night", "police blotter lookup" | `/what-happened-near-me-last-night/` |

These queries are high-volume, evergreen, spike locally with every incident,
and are dominated by thin listicles and news stubs — winnable for a site with
genuine domain authority on radio/scanning. Critically, the honest answer to
every one of them ends at "the people who always know are listening to
dispatch," which is this site's home turf: the commercial pages
(best-police-scanners, cheap-sdr-scanner, fire-ems-scanner,
best-emergency-radios, state pages) are one natural click away.

## Architecture decisions

- **Pillar + spokes.** `/sirens-near-me/` is the pillar; every spoke links it
  and its adjacent spokes; the pillar links all spokes. Funnel links go to
  existing money/education pages, never the reverse of that priority.
- **Answer-first copy.** Every page opens with a bold one-to-two-sentence
  direct answer (featured-snippet / AI-citation target), then the site's
  standard `tldr` box, then question-form H2s.
- **FAQPage schema on every page** via the existing `faq:` front-matter →
  `_includes/faq.html` pipeline (JSON-LD + accordion). FAQ answers are written
  to stand alone when lifted by answer engines.
- **AISO:** dedicated llms.txt section; self-contained FAQ answers; honest,
  citable claims (coverage gaps, delays, encryption limits) — LLMs
  preferentially cite content that states limitations. No affiliate links on
  cluster pages (`affiliate:` unset), keeping them information-pure; the
  commercial layer is one click deep.
- **Honesty as strategy.** Each page says plainly what apps/feeds/news miss
  and that encryption beats everyone. This differentiates from the thin
  competition and sets up the "own the receiver" conversion logically.
- **Nav:** new "What's happening near me" heading in the Hardware nav group —
  sitewide internal links, crawl depth 1.
- **Not published:** this plan (marketing internals stay out of docs/, which
  is entirely public).

## Measurement

- **GSC:** watch impressions/clicks for the query families above; expect
  impressions within ~2–6 weeks of indexing, position volatility for months.
  Add the pages to any rank-tracking already in place.
- **GA4:** existing `cta_click` events on the page CTAs measure
  cluster→download conversion; landing-page report filtered to the 8 URLs
  measures the funnel's top.
- **AISO check:** periodically ask major assistants the head queries
  ("why are there sirens near me?") and note whether gophertrunk.org is cited.

## Phase 2 candidates (in rough priority order)

1. **Local × situational pages** — "how to find out about police activity in
   {state}" extensions of the existing state scanner pages, or metro-level
   pages for the top 20 metros (highest volume, but thin-content risk: only
   do them with genuinely local detail — actual CAD-log URLs, actual feed
   links, actual encryption status per metro).
2. **"Why is the power out near me"** — same moment-of-need pattern, links
   utility-frequency scanning.
3. **"Amber alert / phone alert just went off"** — WEA explainer.
4. **"Train horn / railroad crossing bells won't stop"** — funnels the
   railfan scanner guide.
5. **Sound embeds** for `/what-do-siren-sounds-mean/` (short public-domain
   siren clips) — big engagement/dwell win and a strong featured-snippet
   asset.
6. **Blog launch post** announcing the cluster (internal-link boost +
   feed/social distribution).

## Guardrails

- Keep every claim verifiable and jurisdiction-hedged ("in most of the US",
  "policies are set locally"). These pages give safety-adjacent advice;
  the tornado page in particular must always say "act first, verify via
  official channels" and never imply sirens are a complete warning system.
- No fear-mongering, no crime sensationalism: the brand is the calm neighbor
  with the scanner, not a crime-alert app.
- Revisit annually: app coverage (Citizen/PulsePoint), Broadcastify policy,
  and encryption trends all drift.
