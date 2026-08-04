---
slug: dmr-bandplan
title: DMR band plan
entry_type: term
category: trunked-radio
description: A DMR band plan maps a Tier III Logical Channel Number to an RF downlink frequency — either by a linear base-plus-spacing formula or by an explicit hand-coded channel table — so a decoder can retune to a granted channel.
keywords: DMR band plan, LCN to frequency, logical channel number, linear band plan, channel table, Tier III retune, LPCN, base spacing
aka: ["band plan", "LCN resolver", "channel plan"]
autolink: true
infobox:
  - { label: Input, value: 12-bit LCN (LPCN) }
  - { label: Output, value: downlink frequency in Hz }
  - { label: Linear form, value: "Base + (LCN − Offset) × Spacing" }
  - { label: Alternative, value: explicit LCN → Hz table }
see_also: [channel-grant, dmr-tier-3, dmr-csbk-payloads, rest-channel, multisite-trunking]
cite_urls:
  - https://en.wikipedia.org/wiki/Digital_mobile_radio
  - https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/
---

A **DMR band plan** is the mapping a Tier III decoder uses to turn a **Logical Channel Number**
(**LCN**) into an actual RF downlink frequency.[^wiki] A [channel grant](/reference/channel-grant/)
CSBK never names a frequency directly — it carries a compact 12-bit LCN (the LPCN, Logical
Physical Channel Number) — so before a receiver can retune to follow a call it must resolve that
number against the system's band plan.[^etsi] Two forms cover real deployments: a linear formula
for sites laid out on a regular grid, and an explicit table for the irregular ones.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A logical channel number entering a resolver that offers two paths: a linear formula computing base frequency plus channel index times spacing, or a lookup table mapping each LCN to a hand-coded frequency; both produce a downlink frequency in hertz." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="52" width="70" height="26" rx="3" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.1"/>
  <text x="51" y="69" text-anchor="middle" font-size="8.5" fill="currentColor">LCN</text>
  <path d="M86 65 L110 65" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <rect x="110" y="20" width="200" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="210" y="39" text-anchor="middle" font-size="8" fill="currentColor">Base + (LCN − Offset) × Spacing</text>
  <rect x="110" y="80" width="200" height="30" rx="3" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="210" y="99" text-anchor="middle" font-size="8" fill="currentColor">table[LCN] → Hz (hand-coded)</text>
  <path d="M110 60 L104 60 L104 35 L110 35" fill="none" stroke="currentColor" stroke-width="1"/>
  <path d="M110 70 L104 70 L104 95 L110 95" fill="none" stroke="currentColor" stroke-width="1"/>
  <path d="M310 35 L340 35 L340 95 L310 95" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <path d="M340 65 L370 65" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <rect x="370" y="52" width="76" height="26" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="408" y="69" text-anchor="middle" font-size="8" fill="currentColor">freq (Hz)</text>
</svg>
<figcaption>A grant's logical channel number is resolved to a downlink frequency either by a base-plus-spacing formula or by a hand-coded lookup table, and the receiver retunes there to follow the call.</figcaption>
</figure>

## The linear plan

Most sites number their channels on a regular grid, so a single formula suffices:

> frequency = Base + (LCN − Offset) × Spacing

`Base` is the downlink frequency of the first channel, `Spacing` is the constant step between
adjacent channels, and `Offset` accommodates the common convention of numbering channels from 1
rather than 0 — with `Offset = 1`, LCN 1 maps to exactly `Base`. This is compact to configure and
covers the majority of Tier III systems: an operator supplies three numbers and every LCN the
system can grant resolves without an entry-by-entry table. The resolver rejects a zero spacing,
guards against an LCN below the offset (which would compute a negative index), and refuses a
result that overflows a 32-bit frequency, returning an "LCN outside band plan" error in each case
rather than retuning to a bogus frequency.

## The table plan

Some systems do not lay channels out linearly — frequencies may be scattered across bands, reused
from an earlier license, or coordinated around incumbents. For these, the band plan is an explicit
LCN → Hz map the operator hand-codes from the license or coordination database. A lookup is a
direct map read: a present key returns its frequency, and a missing key returns the same
"unknown LCN" error the linear plan uses for an out-of-range channel. Configuration validation
guarantees exactly one of the two forms is set per system.

## When resolution fails

Because a grant is useless without a frequency, an unresolvable LCN is treated as a configuration
gap, not a silent no-op. The control channel maps `ErrUnknownLCN` to a `decode.error` event tagged
`stage="no-bandplan"`, so an operator watching metrics can see that grants are arriving for
channels the plan does not cover — the signal to extend the linear range or add table entries. A
system with no band plan at all drops every grant with that same stage, which is how a monitor that
locks the control channel but never follows a call announces that it simply has not been told where
the traffic channels are.

## Relevance to SDR

`internal/radio/dmr/tier3/bandplan.go` defines a `Resolver` interface with two implementations:
`LinearBandPlan` (the base/spacing/offset formula, with the overflow and range guards) and
`TableBandPlan` (the map lookup). `ResolverFromPlan` builds the right one from the operator-supplied
band plan on the trunking system, returning nil when none is configured so the "no band plan ⇒
drop grant" behaviour is preserved. The resolved frequency is what the engine hands to a Voice
device when it follows a grant, so the band plan sits directly between the
[control channel](/reference/dmr-csbk-payloads/) decode and the physical retune — the last step
that turns an abstract channel number into a place on the dial.

## Sources

[^wiki]: [Digital mobile radio](https://en.wikipedia.org/wiki/Digital_mobile_radio) — Wikipedia, on DMR trunking and logical channel numbering.
[^etsi]: [ETSI TS 102 361-4 (DMR Tier III)](https://www.etsi.org/deliver/etsi_ts/102300_102399/10236104/) — ETSI, defining the logical-channel / frequency relationship signalled on a Tier III system.
