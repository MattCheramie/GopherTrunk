---
slug: edacs-ltr-mpt1327
title: "EDACS, LTR & MPT-1327"
description: EDACS, LTR, and MPT-1327 explained — three legacy trunking families with a fast EDACS control channel, LTR's distributed sub-audible data, and MPT-1327's global reach.
keywords: EDACS, LTR, MPT-1327, Logic Trunked Radio, AFS, agency fleet subfleet, ProVoice, LTR-Net, Passport, distributed control, legacy trunking, 9600 bps control channel
level: intermediate
status: full
prereq:
  - trunking-flavors
  - analog-trunking-era
faq:
  - q: What is EDACS?
    a: EDACS (Enhanced Digital Access Communications System) is a trunking family from GE, later Ericsson and Harris. It uses a fast dedicated 9600 bps control channel and an AFS — Agency/Fleet/Subfleet — talkgroup numbering scheme. Voice is usually analog FM, though a digital variant called ProVoice exists.
  - q: How is LTR different from other trunking systems?
    a: LTR (Logic Trunked Radio) has no dedicated control channel. Instead, low-speed sub-audible data is carried on every repeater alongside the voice, so the trunking control is distributed across all the channels rather than centralised on one. Variants include LTR-Net and Passport.
  - q: What is MPT-1327?
    a: MPT-1327 is a British signaling standard for analog trunking, widely used internationally outside North America. It uses a 1200 bps control channel and analog FM voice. It was one of the most common trunking standards in Europe, Africa, Australia, and elsewhere for commercial and government fleets.
  - q: Are these systems still in use?
    a: Yes, though they're legacy and shrinking as operators migrate to P25 and DMR. EDACS, LTR, and MPT-1327 systems still run in various regions, and a scanner enthusiast still meets them. Recognising each by its control-channel behaviour is a useful skill.
gophertrunk_links:
  - title: Status
    url: /status.html
    note: which legacy trunking protocols GopherTrunk decodes.
  - title: CC Activity
    url: /cc-activity.html
    note: watch the control-channel data these systems use to coordinate calls.
---

# EDACS, LTR & MPT-1327

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three legacy trunking families you'll still meet. **EDACS** (GE/Ericsson/Harris) uses a
**fast dedicated 9600 bps control channel** and **AFS — Agency/Fleet/Subfleet** numbering,
with a digital **ProVoice** voice option. **LTR** (Logic Trunked Radio) is unusual:
**no dedicated control channel** — low-speed **sub-audible data rides on every repeater**,
so control is **distributed**, with variants **LTR-Net** and **Passport**. **MPT-1327** is
the **British/European** standard with a **1200 bps control channel**, used widely
**internationally**. All three are mostly **analog FM voice** with digital signaling.

</div>

Beyond Motorola's [SmartNet](/learn/digital-trunking/motorola-smartnet/), three other legacy families filled the
analog-trunking world. Each solved the [trunking](/learn/digital-trunking/conventional-vs-trunked/) problem
differently, and each has a recognisable fingerprint on the air.

## EDACS

**EDACS** — Enhanced Digital Access Communications System — came from **General Electric**,
passing through **Ericsson** and then **Harris** as the product line changed hands. Its
hallmarks:

- A **fast, dedicated control channel** running at **9600 bps** — quicker than SmartNet's
  3600 bps or MPT's 1200 bps, which made EDACS feel snappy in assigning channels.
- **AFS** talkgroup numbering — **Agency / Fleet / Subfleet** — a three-level hierarchy that
  organises [talkgroups](/learn/digital-trunking/talkgroups-ids-affiliation/) by agency, then fleet within it, then
  subfleet, rather than a flat number.
- Voice is usually **analog FM**, but a **digital variant, ProVoice**, exists for systems
  that wanted digital audio over the EDACS infrastructure.

EDACS was a serious public-safety and utility contender in its day and you'll still find
systems running.

## LTR

**LTR** — Logic Trunked Radio — takes a genuinely different approach to everything else in
this path. There is **no dedicated control channel at all**.

Instead, **low-speed sub-audible data** is carried on **every repeater**, riding along below
the voice on each channel. The trunking logic is **distributed** across all the channels
rather than centralised on one control frequency: each repeater announces its own status,
and radios use that scattered data to find and follow calls. This makes LTR systems cheaper
and simpler to build — there's no separate control channel to dedicate — but it means the
"control" information is spread out rather than sitting in one place.

Common variants you'll meet:

- **LTR-Net** — a networked, multi-site evolution of basic LTR.
- **Passport** — another LTR variant with its own roaming and numbering features.

<figure class="figure" markdown="0">
<svg viewBox="0 0 540 170" role="img" aria-label="A comparison: a centralised system with one dedicated control channel feeding voice channels, versus LTR where each repeater carries its own sub-audible control data." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor">
    <text x="135" y="18" text-anchor="middle" font-weight="600">Dedicated control (EDACS, MPT)</text>
    <rect x="40" y="30" width="190" height="22" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="135" y="45" text-anchor="middle" font-size="9">one control channel</text>
    <rect x="40" y="70" width="58" height="20" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="69" y="84" text-anchor="middle" font-size="8">voice</text>
    <rect x="106" y="70" width="58" height="20" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="135" y="84" text-anchor="middle" font-size="8">voice</text>
    <rect x="172" y="70" width="58" height="20" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="201" y="84" text-anchor="middle" font-size="8">voice</text>
    <line x1="135" y1="52" x2="135" y2="68" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
    <text x="405" y="18" text-anchor="middle" font-weight="600">Distributed (LTR)</text>
    <rect x="310" y="70" width="58" height="34" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="339" y="86" text-anchor="middle" font-size="8">voice</text><text x="339" y="98" text-anchor="middle" font-size="7">+ sub-audible</text>
    <rect x="376" y="70" width="58" height="34" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="405" y="86" text-anchor="middle" font-size="8">voice</text><text x="405" y="98" text-anchor="middle" font-size="7">+ sub-audible</text>
    <rect x="442" y="70" width="58" height="34" rx="3" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1"/><text x="471" y="86" text-anchor="middle" font-size="8">voice</text><text x="471" y="98" text-anchor="middle" font-size="7">+ sub-audible</text>
    <text x="405" y="128" text-anchor="middle" font-size="9">every repeater carries its own control data</text>
  </g>
</svg>
<figcaption>EDACS and MPT-1327 use one dedicated control channel; LTR distributes the control as sub-audible data on every repeater, with no separate control frequency.</figcaption>
</figure>

## MPT-1327

**MPT-1327** is a **British signaling standard** (the "MPT" prefix comes from the UK Ministry
of Posts and Telecommunications) that became the **dominant trunking standard outside North
America**. It uses a **1200 bps control channel** and **analog FM** voice.

For decades MPT-1327 was the go-to for commercial and government fleets across **Europe,
Africa, Australia, Asia, and Latin America** — taxi companies, utilities, transport, and
public services. Its international ubiquity is roughly the role SmartNet played in the US.
You're most likely to meet it if you monitor outside North America.

## Telling them apart

| System | Control channel | Talkgroup numbering | Origin | Where common |
|--------|-----------------|---------------------|--------|--------------|
| **EDACS** | Dedicated, **9600 bps** (fast) | **AFS** (Agency/Fleet/Subfleet) | GE → Ericsson → Harris | US public safety, utilities |
| **LTR** | **None** — distributed sub-audible on every repeater | Home repeater + ID | Specialised mobile radio | Business, light commercial |
| **MPT-1327** | Dedicated, **1200 bps** | Prefix + ident | UK (MPT) | International, outside North America |

The quickest fingerprints: EDACS has a *fast* dedicated control channel; **LTR has no
dedicated control channel** to find at all (look for sub-audible data on the voice
repeaters); and MPT-1327's slower 1200 bps control channel turns up mostly outside the US.
As always, the surest identification is letting a decoder read the [control-channel
signaling](/learn/digital-trunking/control-channel-signaling/) once it's locked.

<div class="knowledge-check" data-quiz data-correct-msg="Right — LTR has no dedicated control channel; the data is distributed across every repeater." markdown="0">
  <p class="knowledge-check__q">Quick check: which of these systems has no dedicated control channel?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">EDACS</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">MPT-1327</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">LTR</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **EDACS** (GE/Ericsson/Harris) uses a **fast 9600 bps dedicated control channel** and
  **AFS** numbering, with a digital **ProVoice** option.
- **LTR** has **no dedicated control channel** — control is **distributed** as sub-audible
  data on every repeater; variants include **LTR-Net** and **Passport**.
- **MPT-1327** is the **British/European** standard, **1200 bps** control channel, used
  **widely internationally**.
- All three carry mostly **analog FM voice** with digital signaling.
- Tell them apart by control-channel behaviour — or let the decoder identify the signaling.

Next, we lighten up with the smaller digital modes you'll meet on the bands: [dPMR, D-STAR
& System Fusion](/learn/digital-trunking/dpmr-dstar-ysf/).
