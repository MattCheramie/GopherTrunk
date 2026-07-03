---
title: "Signal Lab, Part 8: Naming the Unknown — Blind Signal-ID & the Wideband Survey"
description: Signal Lab identifies an unknown capture by ranking protocol candidates, names undecodable carriers from a blind symbol-rate and modulation estimate against an offline signal-ID reference database, and surveys a wide capture in the Wideband tab — where the Mercury signal finally gets a best guess but no lock.
category: tutorials
keywords: blind signal identification, signal id, protocol identify, ranked candidates, symbol rate estimate, modulation estimate, wideband survey, sigref database, unknown protocol, 4-level fsk, mercury signal
tags: [siglab, signal-id, wideband, survey, unknown, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Signal Lab"
series_part: 8
charts: true
---

*Part 8 of **Signal Lab**, a 10-part series on GopherTrunk's offline
signal-analysis workbench. Everything so far assumed you knew what the signal
was. This part is for when you don't.*

> **TL;DR:** When a capture doesn't decode as anything named, Signal Lab still
> tries to *name it*. **Identify** ranks protocol candidates rather than
> guessing one; the offline **signal-ID reference database (sigref)** names an
> undecodable carrier from a blind **symbol-rate and modulation** estimate; and
> the **Wideband** tab surveys a wide capture for every carrier in it. This is
> where **Mercury** — Ada's intermittent 453 MHz burst — finally gets a best
> guess (~4800 sym/s, 4-level-FSK-like) but no lock, and gets handed to RF Scope.

**Key takeaways**

- **Identify ranks candidates**, it doesn't guess one — you see the runners-up
  and their scores.
- **Blind estimation names the undecodable.** Symbol rate and modulation class
  are recoverable even when no decoder locks.
- **sigref turns an estimate into a name** by matching it against a reference
  database of known systems.
- **The Wideband survey** finds and characterizes every carrier in a wide
  capture at once.
- **A best guess is not a lock.** "~4800 sym/s, 4-level-FSK-like" is a lead, not
  a decode — which is exactly Mercury's status.

## Cheat sheet

| Surface | What it does |
|---|---|
| Identify tab / `gophertrunk identify` | Rank protocol candidates for a capture |
| Blind symbol-rate estimate | Recover the symbol rate with no lock |
| Blind modulation estimate | Classify modulation family (FSK/PSK levels) |
| sigref reference DB | Name a carrier by matching its blind estimate |
| Wideband tab | Survey a wide capture for all carriers |

## In this post

- **When nothing decodes** — the problem blind ID solves.
- **Ranked candidates** — why a list beats a guess.
- **Blind estimation** — symbol rate and modulation without a lock.
- **The signal-ID reference database** — turning an estimate into a name.
- **The wideband survey** — and Mercury's best guess.

## When nothing decodes

Up to now every capture came with an assumption: *this is P25*, *this is DMR*.
Point the decoder at it, read the metrics, done. But operators constantly hit
carriers that decode as *nothing* — an unfamiliar system, a proprietary waveform,
or a signal too weak or too short to lock. The naive tool gives up: "not locked,"
end of story. That's a dead end exactly when you most want a lead.

Signal Lab's answer is that a signal you can't *decode* is not a signal you can't
*describe*. Even with no lock, the raw IQ still carries recoverable structure —
roughly how fast it's keying (symbol rate) and roughly how it's modulated (the
number of levels, FSK-like vs PSK-like). Extract those and you can *name a
candidate* even when you can't read a single bit. That's the whole premise of the
identify and wideband surfaces.

## Ranked candidates, not a guess

The **Identify** surface (mirrored by `gophertrunk identify` on the CLI) runs a
capture against the decoder's roster and returns **ranked candidates** — a scored
list, best guess on top, with the runners-up visible. That ranking is more honest
and more useful than a single verdict. A close top-two ("62% NXDN, 55% dPMR")
tells you the signal is genuinely ambiguous and worth a human look; a runaway
leader ("94% P25 Phase 1") tells you to just decode it. You get calibrated doubt
instead of false confidence.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="560" height="300" role="img"
        aria-label="Ranked protocol identification candidates by confidence score"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["P25 P1","NXDN","dPMR","DMR","unknown"],
"values":[0.31,0.28,0.22,0.11,0.62],
"pass":[false,false,false,false,true],
"ylabel":"candidate score" }
</script>
<figcaption>Illustrative ranked candidates for a hard capture: no named protocol dominates, and the strongest bar is "unknown" — the signature of a carrier that needs blind identification rather than a decoder.</figcaption>
</figure>

When the ranking's top result is essentially "none of the above," you've left the
land of named decoders and entered blind ID.

## Blind estimation: rate and modulation without a lock

Blind identification estimates two things directly from the IQ, no decoder
required:

- **Symbol rate** — how fast the signal is keying, recovered from the modulation's
  own periodicity (spectral and cyclostationary cues). This is the single most
  discriminating number for narrowband land-mobile radio: 4800 sym/s, 9600
  sym/s, and 2400 sym/s carve the space into families immediately.
- **Modulation class** — the *shape* of the modulation: how many levels, and
  whether it's frequency-like (FSK/C4FM family) or phase-like (PSK/QPSK family),
  read from the constellation and spectral structure you met in Parts 4 and 5.

Neither needs a lock, which is the point — these survive on signals no decoder
will touch. A carrier can be too short, too weak, or too proprietary to decode
and *still* give up "~4800 sym/s, 4-level frequency modulation." That pair is
often enough to name a candidate.

## The signal-ID reference database

An estimate on its own — "4800 sym/s, 4-level FSK" — is a description, not a name.
The **offline signal-ID reference database** (sigref) closes that gap. It holds
the blind signatures of known systems, and it matches an undecodable carrier's
estimate against them to produce a *best-guess name* rather than dropping the
carrier entirely. So a survey doesn't report "unknown carrier at 453.2125 MHz" —
it reports "unknown carrier, best match: 4-level FSK, ~4800 sym/s, consistent
with [candidate]." It's the difference between a blank and a lead.

The database is offline and bundled, in keeping with the whole workbench — no
lookup service, no network. That matters for the field: you can name the unknown
on an air-gapped laptop from the raw capture alone.

## The wideband survey — and Mercury

The **Wideband** tab scales blind ID from one channel to a whole band. Feed it a
wide capture and it surveys the spectrum, finds every carrier in it, and runs the
blind estimate on each — a table of "here's what's on the air, and here's my best
guess for each," including the ones that don't decode. It's the natural companion
to the spectrogram from Part 5: the waterfall shows you *where* the energy is, and
the survey tries to *name* it.

This is where Ada finally corners **Mercury**. Surveying a wide capture around
453 MHz, the Wideband tab flags an intermittent carrier — present, gone, present
again, exactly its spectrogram stutter — and runs blind ID on it:

| Field | Mercury's survey result |
|---|---|
| Center | ~453 MHz (UHF business band) |
| Channel | ~12.5 kHz |
| Symbol rate (blind) | ~4800 sym/s |
| Modulation (blind) | 4-level-FSK-like |
| Named decode | none — no lock |
| sigref best guess | candidate only |

So Mercury gets a *description* and a *candidate* — ~4800 sym/s, 4-level-FSK-like
— but **no lock**: it does not decode as any named protocol. That's not a failure
of the workbench; it's an honest verdict. Blind ID has done exactly its job —
turned a blank carrier into a characterized lead — and the lead now needs a
different instrument. Ada exports the wideband capture and hands it to
[RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}), whose protocol
hierarchy is built to work an unknown, intermittent emitter.

That handoff is a real seam in the toolchain, not just a narrative one:
[RF Scope Part 3]({{ '/blog/tutorials/rf-scope-03-protocol-hierarchy/' | relative_url }})
reuses this same blind-identify machinery to place an unknown emitter in its
protocol hierarchy. Signal Lab names the candidate; RF Scope works out what the
emitter is *doing*.

## The limits of a best guess

Blind identification is powerful precisely because it's honest about what it
doesn't know, and it's worth being clear about *why* Mercury stops at a candidate
rather than a decode. A symbol-rate and modulation-class estimate constrains the
*family* a signal belongs to, but two systems can share a symbol rate and a
modulation and still be entirely different protocols — the framing, the FEC, the
scrambling, and the higher-layer structure are what actually distinguish them, and
none of that is recoverable without a decoder that locks. "~4800 sym/s,
4-level-FSK-like" narrows Mercury to a neighborhood; it doesn't hand you the
address.

There's also the ceiling that blind ID can't push through by design: if a payload
is scrambled or encrypted, the modulation and rate still read cleanly — that's a
physical-layer property — but nothing above it will ever parse into named fields,
no matter how good the estimate. A carrier can be perfectly characterized at the
symbol level and completely opaque at the content level. That gap is exactly where
Mercury lives, and it's the whole reason the trail has to leave Signal Lab: naming
the modulation was the *last* thing this workbench can do for it. Reading what's
inside the frames is a different discipline, which is why Ada's export becomes RF
Scope's input and, eventually, Crypto Lab's problem.

The healthy mindset, then, is to treat a blind ID result as a **lead with a
confidence attached**, never a conclusion. A tight, well-separated top candidate
is worth trying to decode immediately; a diffuse ranking or a bare "best match"
with no lock is a signal to gather more — a longer capture, a wider survey, a
handoff to the next instrument — rather than to force a decode that isn't there.
Ada's instinct at the start of the series was to keep re-running the decoder on
Mercury hoping it would eventually catch. Reese's correction: the decoder isn't
going to catch, and the survey already told you why — take the characterized lead
and move it down the chain.

## Where this goes next

[Part 9]({{ '/blog/tutorials/signal-lab-09-dissecting-p25-pdus/' | relative_url }})
returns to signals that *do* decode and goes deep — dissecting P25 TSBK PDUs
field by field, watching the receiver-state series, and reading the
sync-landscape heatmap. Meanwhile Mercury's trail leaves Signal Lab and picks up
in [RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}), where an
unknown-protocol, intermittent emitter is exactly the kind of thing the Scene is
built to triage. The [SigLab docs]({{ '/siglab.html' | relative_url }}) describe the
identify and survey surfaces in full.

## FAQ

**How can it identify a signal it can't decode?**
By estimating structure instead of reading content. Symbol rate and modulation
class are recoverable from raw IQ with no lock, and matching that blind estimate
against the signal-ID reference database yields a best-guess name — a lead, not a
decode.

**Why ranked candidates instead of one answer?**
Because confidence is information. A tight top-two means genuine ambiguity worth a
human look; a runaway leader means just decode it. A single verdict throws that
away.

**What is the wideband survey for?**
Characterizing a whole band at once: it finds every carrier in a wide capture and
runs blind ID on each, so you get a named (or best-guessed) inventory of what's on
the air, decodable or not.

**Did Signal Lab decode Mercury?**
No. It produced a best guess — ~4800 sym/s, 4-level-FSK-like — but no lock and no
named protocol. That characterized lead is handed to RF Scope for the next stage.

## Series navigation

**Part 8 of 10** · ←[Part 7]({{ '/blog/tutorials/signal-lab-07-vsa-evm-modulation-quality/' | relative_url }}) · Next →
[Part 9: Dissecting P25 PDUs]({{ '/blog/tutorials/signal-lab-09-dissecting-p25-pdus/' | relative_url }})
