---
layout: page
title: "Is It Legal to Own a Police Scanner? (2026)"
description: "Is it legal to own a police scanner in 2026? A plain-English guide to federal law (ECPA, Communications Act, 18 U.S.C. §2512), state variation, in-car rules, and recording — not legal advice."
keywords: is it legal to own a police scanner, police scanner laws, scanner legal, listen to police scanner legal, police scanner in car law, is it legal to record police scanner, scanner encryption law
permalink: /police-scanner-legal/
nav_group: Hardware
faq:
  - q: "Is it legal to own and listen to a police scanner?"
    a: "In the United States, owning a scanner and listening to unencrypted public-safety radio is legal at the federal level. The Communications Act protects the airwaves and the ECPA generally exempts readily accessible, non-scrambled radio. A handful of states add restrictions — most commonly on scanner use in a moving vehicle or while committing a crime — so check your own state."
  - q: "Is it legal to use a police scanner in my car?"
    a: "In most states, yes. A few states restrict or prohibit scanner use in a motor vehicle without a permit (often with exceptions for licensed amateur-radio operators, journalists, or first responders). Because this is one of the most common state-level rules, verify your state's statute before mounting a scanner in your vehicle."
  - q: "Is it legal to record what I hear on a scanner?"
    a: "Recording unencrypted radio you lawfully receive is generally treated the same as listening. The legal risk is not the recording itself but divulging or publishing the contents for prohibited purposes, or using it to commit or aid a crime. When in doubt, do not rebroadcast dispatch traffic for commercial gain."
  - q: "Is it legal to decode an encrypted police channel?"
    a: "No. Federal law (18 U.S.C. §2512) prohibits devices whose primary purpose is the surreptitious interception of encrypted or scrambled communications, and defeating encryption without authorization can violate the ECPA. No scanner or SDR can legally decode AES-encrypted P25/DMR, and attempting to break it is a separate legal problem from merely owning a radio."
  - q: "Can I get in trouble just for scanning?"
    a: "Simply listening to clear public-safety radio rarely creates liability by itself. Trouble comes from what you do with it: using intercepted information to interfere with police, evade arrest, aid a crime, or profit from divulging protected communications. Those acts are what statutes target, not the act of tuning a receiver."
  - q: "Do these rules apply outside the United States?"
    a: "No. This page describes U.S. federal and general state concepts. Many countries — including the UK under the Wireless Telegraphy Act — treat scanning far more restrictively, sometimes making it an offense to listen to anything you are not licensed to hear. Always check local law where you are."
---

# Is It Legal to Own a Police Scanner? (2026)

**In the United States, it is legal to own a police scanner and to listen to
unencrypted public-safety radio — the legal lines are about decoding encryption
and about what you *do* with what you hear, not about owning the receiver.**
The rules come from a mix of federal statutes and a patchwork of state laws, so
the short answer is "yes, with caveats," and the caveats depend on where you live.

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Owning and listening** to unencrypted public-safety radio is legal federally.
**Decoding [encryption](/police-scanner-encryption/)** is not — 18 U.S.C. §2512
bars devices meant to intercept scrambled traffic, and no scanner or SDR can do
it anyway. **A few states** restrict scanner use **in a vehicle** or add penalties
for using one **while committing a crime**. **Divulging** the contents of some
communications for prohibited purposes can violate the Communications Act / ECPA.
**This is not legal advice** — check your state and, if it matters, a lawyer.
</div>

## The federal baseline

Three federal ideas do most of the work, and none of them makes owning a scanner
illegal.

- **The Communications Act of 1934 (47 U.S.C. §605).** It protects radio
  communications and restricts *divulging or publishing* certain intercepted
  traffic for the interceptor's benefit. It has long coexisted with a huge
  hobbyist community — the restriction is on misuse and publication, not on the
  radio in your hand.
- **The Electronic Communications Privacy Act (ECPA).** The ECPA generally
  **exempts radio communications that are "readily accessible to the general
  public"** — that is, transmitted in the clear and not scrambled or encrypted.
  Unencrypted conventional and [trunked](/reference/trunked-radio/) public-safety
  radio falls into that readily-accessible category.
- **18 U.S.C. §2512.** This is the encryption line. It prohibits manufacturing,
  selling, or possessing devices **primarily useful for the surreptitious
  interception** of communications — the hook that makes defeating
  [encryption](/reference/encryption/) a legal problem, on top of being a
  cryptographic impossibility for consumer gear.

> **The pattern.** Federal law protects *scrambled* and *private* traffic and
> punishes *misuse*. It does not criminalize the ordinary act of tuning a
> receiver to clear public-safety channels. Encryption is the bright line.

## What the federal rules actually prohibit

It helps to separate the legal receiver from the illegal act.

| You are... | General federal treatment |
|---|---|
| Owning a scanner or SDR | **Legal** |
| Listening to unencrypted police/fire/EMS | **Legal** (readily accessible) |
| Recording clear traffic for personal use | **Generally legal**, like listening |
| Decoding an AES/DES-encrypted channel | **Prohibited** (18 U.S.C. §2512; impossible anyway) |
| Divulging protected contents for gain | **Restricted** (Communications Act / ECPA) |
| Using intercepts to commit or aid a crime | **Illegal**, and usually an enhancement |

The recurring theme: the receiver is fine, the crime is what you do next.

## Where states differ

Federal law sets a floor; states add their own rules on top, and this is where
"is it legal" stops having one answer. Rather than cite statutes that change year
to year, here are the **categories** of state law you should look for — then check
your own state.

- **Mobile-use / in-vehicle restrictions.** The single most common state rule.
  Some states restrict or require a permit to *use* a scanner in a moving motor
  vehicle, frequently with **exceptions** for licensed amateur-radio operators,
  working journalists, or public-safety personnel. This is the one most likely to
  catch a hobbyist off guard.
- **Commit-a-crime enhancements.** Many states make it a separate or aggravated
  offense to *use* a scanner **in the commission of a crime** — to evade police,
  coordinate a burglary, and so on. Listening lawfully is fine; listening to
  further a crime is not.
- **Sensitive-location or context rules.** Occasional rules touch specific
  contexts (for example, use tied to certain offenses). These are narrow but
  real.

> **Do not trust a forum list — including this one — as gospel.** State scanner
> laws are amended, repealed, and reinterpreted. The only reliable move is to read
> your **current** state statute (search your state code for "radio" / "scanner" /
> "police radio" / "interception") or ask a local attorney. GopherTrunk cannot
> tell you what your state says today.

## Listening vs. divulging vs. decoding

Three different verbs, three different legal footings — keeping them straight is
most of understanding this topic.

- **Listening.** Receiving unencrypted, readily-accessible radio is the
  best-protected activity. This is the ordinary scanner hobby.
- **Divulging.** *Publishing or using* the contents of certain communications can
  cross the Communications Act / ECPA lines, especially for commercial benefit or
  when the traffic was not readily accessible. Personal listening is not
  publication.
- **Decoding.** Breaking [encryption](/police-scanner-encryption/) is a different
  category entirely — it implicates 18 U.S.C. §2512 and the ECPA's anti-circumvention
  posture, and it is not something any legitimate scanner or
  [SDR](/reference/software-defined-radio/) does. GopherTrunk decodes *clear* and
  merely-*digital* signals; it does **not** and cannot recover keyed
  [AES](/reference/advanced-encryption-standard/) audio.

## Practical guidance for staying clearly legal

- **Listen, don't interfere.** Never use scanner traffic to evade, obstruct, or
  aid anyone against police, fire, or EMS operations.
- **Don't try to defeat encryption.** If a [talkgroup](/reference/talkgroup/) is
  encrypted, it is off-limits both technically and legally. Move on to the many
  channels still in the clear.
- **Be careful about republishing.** Personal logs are one thing; monetizing or
  broadcasting dispatch content is where divulging rules bite. Many hobbyists feed
  audio to public archives — understand the terms and local norms first.
- **Mind the car rule.** If you plan to scan while driving, confirm your state
  allows it or whether an amateur-radio license or permit changes the answer.
- **Know your country.** Outside the U.S., scanning is often far more restricted.
  The UK, for instance, treats listening to non-broadcast radio you are not
  licensed to hear as an offense.

> **Not legal advice.** This page is general education, not legal advice, and it
> does not create an attorney-client relationship. Laws change and vary by
> jurisdiction. For your situation, read your current state statute or consult a
> licensed attorney.

## Bottom line

Owning a police scanner and listening to **unencrypted** public-safety radio is
legal in the United States at the federal level, and in most states — the real
constraints are the **encryption** line under 18 U.S.C. §2512, the
Communications Act / ECPA rules on **divulging** protected content, and a
**handful of state** restrictions, most notably on **in-vehicle use** and use
**during a crime**. Check your own state before you assume, treat encrypted
channels as off-limits, and remember this page is education, not legal advice.
Curious what you can actually still hear? See
[police scanner encryption explained](/police-scanner-encryption/) and
[scanner vs SDR](/police-scanner-vs-sdr/).
