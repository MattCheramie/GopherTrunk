---
slug: what-is-open-source
title: What open source really means
description: 'Open source is a precise term defined by the OSI''s Open Source Definition, not just "the code is visible." Learn the criteria, the FSF''s four freedoms, free-as-in-freedom vs free-as-in-beer, and why source-available is not the same thing.'
keywords: open source, Open Source Definition, OSI, free software, FSF, four freedoms, FOSS, FLOSS, free as in freedom, source available, software license, OSD criteria, copyleft, permissive
level: beginner
status: full
faq:
  - q: "Does 'open source' just mean I can see the source code?"
    a: "No — that's the most common misconception. **Open source** is a precise term defined by the OSI's [Open Source Definition](/learn/software-licensing/what-is-open-source/): the license must also let you **use, modify, and redistribute** the code, including for any purpose and by anyone. Code that is merely visible but carries usage restrictions is **source-available**, not open source."
  - q: "What's the difference between 'open source' and 'free software'?"
    a: "They describe almost the same set of licenses but emphasize different things. The Free Software Foundation's **free software** frames it around four **freedoms** for the user (run, study, modify, distribute); the Open Source Initiative's **open source** frames it around a ten-point definition aimed at developers and businesses. In practice the approved license lists overlap heavily, which is why people say **FOSS** or **FLOSS** to cover both."
  - q: "Is free software always free of charge?"
    a: "No. **Free** here means freedom, not price — \"free as in freedom, not free as in beer.\" You are allowed to sell free or open-source software, and many companies do. What the license guarantees is the user's freedom to run, study, change, and share it, not that it costs nothing."
---
# What open source really means

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Open source is a precise term** — defined by the OSI's ten-point Open Source Definition, not by whether you can read the code. **Free software and the four freedoms** — the FSF's parallel idea, framed as freedoms to run, study, modify, and distribute. **Free means freedom, not price** — "free as in freedom, not free as in beer"; you can sell it. **Source-available is not open source** — visible code with usage limits fails the definition.
</div>

"Open source" is one of the most used and most misunderstood phrases in software. Plenty of people assume it just means "I can see the code," but it is actually a precise, agreed-upon term with a published definition — and a lot of code that looks open turns out not to qualify. By the end of this lesson you'll know what the Open Source Initiative's definition actually requires, how the Free Software Foundation's "four freedoms" frame the same idea, why "free" here means freedom rather than zero cost, what the FOSS and FLOSS acronyms mean, and the crucial distinction between open source and merely *source-available* code.

## "Open source" is not just "the source is visible"

The single biggest misconception is that open source means you can look at the code. You can read the source of plenty of programs — leaked code, code published "for reference," code with a "no commercial use" sticker on it — and none of that makes them open source.

Open source is a term of art. It refers to software released under a license that meets a specific, published standard. That standard grants you far more than the ability to read: the right to **run** the program for any purpose, to **modify** it, and to **redistribute** it and your changes. Visibility is necessary but nowhere near sufficient.

This matters because the word carries real expectations. When a developer says a library is open source, others assume they can build on it, ship it, and fork it if the project goes sideways. If the license quietly forbids commercial use, those expectations are wrong — and the project shouldn't be calling itself open source. We pick this thread up directly in [Source-available & "fair source"](/learn/software-licensing/source-available-licenses/).

## The Open Source Definition (OSI)

The authority most of the industry points to is the **Open Source Initiative (OSI)** and its **Open Source Definition (OSD)** — a checklist of ten criteria a license must satisfy to be called open source. The OSI also maintains a list of *approved* licenses that have been reviewed against it, which is how you avoid arguing the criteria from scratch every time.

You don't need all ten memorized, but the key ones are worth knowing:

- **Free redistribution.** The license can't stop anyone from giving the software away or selling it as part of a larger distribution, and can't demand a royalty for that.
- **Source code available.** The program must include source, or make it readily available at no more than a reasonable reproduction cost. Deliberately obfuscated source doesn't count.
- **Derived works allowed.** The license must permit modifications and derived works, and allow them to be distributed under the same terms.
- **No discrimination against persons or groups.** The license can't say "everyone except company X" or exclude any group of people.
- **No discrimination against fields of endeavor.** It can't forbid use in a particular field — no "not for commercial use," no "not for military use," no "not for genetic research." This one trips up a lot of would-be open licenses.
- **License must not be specific to a product, and must not restrict other software.** The rights travel with the code on their own, and the license can't, for example, require that everything on the same disk also be open source.

The field-of-use rule is the one to remember, because it rules out a whole category of "almost open" licenses. A "free for non-commercial use" license is, by definition, not open source, no matter how visible the code is.

## Free software and the four freedoms (FSF)

Before "open source" was coined in 1998, the **Free Software Foundation (FSF)**, founded by Richard Stallman, had been promoting **free software** since the 1980s. It frames the same territory differently — as a matter of user freedom rather than a developer/business checklist. A program is free software if it grants its users **four freedoms**:

| Freedom | What it means |
|---|---|
| **Freedom 0** | Run the program as you wish, for any purpose |
| **Freedom 1** | Study how it works and change it (requires source access) |
| **Freedom 2** | Redistribute copies so you can help others |
| **Freedom 3** | Distribute copies of your modified versions |

The freedoms are numbered from zero partly as a programmer's joke and partly to signal that the right to *run* the software is the most basic. Freedoms 1 and 3 both depend on having the source code, which is why source availability is built into the idea rather than tacked on.

The OSD and the four freedoms were developed somewhat independently and use different language, but they describe nearly the same set of licenses. A license that satisfies the four freedoms almost always meets the OSD, and vice versa. The disagreements are at the edges and are usually about emphasis and philosophy, not about which licenses are acceptable.

## "Free as in freedom," not "free as in beer"

English overloads the word "free," and that overload causes endless confusion. The FSF's stock clarification is the one to internalize: **"free as in freedom, not free as in beer."**

- *Free as in beer* means zero cost — someone bought it for you.
- *Free as in freedom* means liberty — you are free to do certain things with it.

Free and open-source software is about the second kind. The license guarantees your freedom to run, study, modify, and share the software. It does **not** guarantee that the software costs nothing, and it explicitly permits selling it. Red Hat built a large business selling a supported Linux distribution that is free software; that's entirely consistent with the definition. We'll return to the business side in [Commercial license models](/learn/software-licensing/commercial-license-models/) and [Can you sell software built on open source?](/learn/software-licensing/using-oss-in-products/).

Because of the ambiguity, many people prefer "open source" specifically to sidestep the price confusion — "open" doesn't sound like "no charge" the way "free" does.

## FOSS and FLOSS

Two communities, two preferred words, one overlapping reality — so people invented neutral umbrella terms:

- **FOSS** — Free and Open Source Software.
- **FLOSS** — Free/Libre and Open Source Software. The "Libre" (the Spanish/French word for free-as-in-freedom) is added to make the freedom sense unmistakable to anyone confused by English "free."

Both terms mean roughly "software that is free *and/or* open source," covering the whole family without taking sides in the philosophical debate between the FSF and the OSI. When you see FOSS or FLOSS, read it as a deliberate attempt to be inclusive of both camps.

## Crucially: source-available is not open source

This is the distinction that most often catches people out, so it's worth stating plainly. **Source-available** means the source code is published and readable, but the license imposes restrictions that an open-source license is not allowed to have — most commonly a ban on commercial use, a ban on competing with the vendor, or a delay before the rights kick in.

Source-available code can be a perfectly reasonable choice for a business, and some well-known projects use it. But it is **not open source**, because it fails the OSD — usually on the no-discrimination-against-fields-of-endeavor or derived-works criteria. Calling it open source is inaccurate and, increasingly, a thing the community pushes back on hard.

The mental test: *can anyone use, modify, and redistribute this for any purpose, including commercially?* If the answer is "no, not for X," it's source-available, not open source. We give this its own full treatment in [Source-available & "fair source"](/learn/software-licensing/source-available-licenses/) — for now, just hold the line between "I can read it" and "I'm free to use it."

<div class="knowledge-check" data-quiz data-correct-msg="Right — the field-of-use rule means a 'no commercial use' license fails the Open Source Definition, so it's source-available, not open source." markdown="0">
  <p class="knowledge-check__q">Quick check: a library's full source is on GitHub, but its license forbids commercial use. Is it open source?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Yes — the source code is publicly visible, so it's open source</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">No — banning a field of use (commercial) fails the Open Source Definition; it's source-available</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Yes, as long as the project calls itself open source in its README</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Open source is a precise term** — it means a license meeting the OSI's Open Source Definition, not simply that the code is visible.
- **The OSD's key criteria** — free redistribution, available source, allowed modifications and derived works, and no discrimination against people or fields of use.
- **Free software's four freedoms** — run, study, modify, and distribute; the FSF's parallel framing of nearly the same set of licenses.
- **Free means freedom, not price** — "free as in freedom, not free as in beer"; selling open-source software is allowed.
- **FOSS / FLOSS** — umbrella acronyms covering both the free-software and open-source camps.
- **Source-available is not open source** — visible code with usage restrictions fails the definition.

Next up: the single biggest divide within open-source licenses — whether you must share your changes back. See [Permissive vs copyleft](/learn/software-licensing/permissive-vs-copyleft/).
