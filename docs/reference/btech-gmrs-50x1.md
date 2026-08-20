---
slug: btech-gmrs-50x1
title: BTECH GMRS-50X1
entry_type: hardware
category: two-way-radios
description: "The BTECH GMRS-50X1 was the first cheap 50-watt Part 95E GMRS mobile — now discontinued and superseded twice (GMRS-50V2, then GMRS-50PRO). A used-market buy at roughly $80–160, with known settings-byte and power-variance gotchas to test for."
keywords: BTECH GMRS-50X1, GMRS-50X1 review, used GMRS mobile radio, 50 watt GMRS radio, GMRS-50X1 CHIRP, GMRS-50X1 vs GMRS-50V2, gmrs license, cheap 50 watt GMRS, GMRS repeater radio, used GMRS radio buying guide, BTECH GMRS mobile
aka: [GMRS-50X1, BTech GMRS-50X1]
autolink: true
affiliate: true
product:
  name: "BTECH GMRS-50X1"
  brand: BTECH
  category: GMRS mobile radio (discontinued)
  itemCondition: used
  lowPrice: "80"
  highPrice: "160"
  url: https://www.amazon.com/dp/B0932TH6T9?tag=gophertrunk-20
infobox:
  - { label: Type, value: "GMRS mobile radio (discontinued, superseded twice)" }
  - { label: Service, value: "GMRS TX; RX/scan 136–174 / 400–520 MHz" }
  - { label: Power, value: "50 W nominal (real output varies by channel)" }
  - { label: Repeater, value: "Yes — split tones" }
  - { label: License, value: "GMRS — $35 FCC, no test, covers family" }
  - { label: Price, value: "around $80–160 used (thin data)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0932TH6T9?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">Renewed on Amazon &rarr;</a>" }
see_also: [btech-gmrs-50pro, midland-mxt400, wouxun-kg-1000g-plus, radioddity-db20-g, baofeng-uv-5r, rtl-sdr]
related_lessons:
  - { title: "Learn RF & SDR", url: /learn/rf-sdr/ }
related_reading:
  - { title: "Best GMRS mobile & base radios", url: /best-gmrs-mobile-radios/ }
cite_urls:
  - https://baofengtech.com/product/gmrs-50x1/
  - https://shop.mygmrs.com/products/btech-gmrs-50v2-50w-gmrs-radio
faq:
  - q: "Is the BTECH GMRS-50X1 discontinued?"
    a: "Effectively, yes — BTECH still hosts the product page, but retail channels sell its successors: the GMRS-50V2 (redesigned board, constant 50 W output, better receive filtering), and now the GMRS-50PRO. The 50X1 is two generations old and lives on the used market."
  - q: "How much is a used BTECH GMRS-50X1 worth?"
    a: "Roughly $80–160, with wide variance — one used unit with mic sold around $80, other listings run $150–160, and full packages with antennas ask more in forum classifieds. The data is thin, so check current sold listings. Amazon carries it only as a Renewed listing."
  - q: "Do I need a license for the BTECH GMRS-50X1?"
    a: "To transmit, yes — a GMRS license: $35 to the FCC, good for 10 years, no test, and it covers your immediate family. Its wide 136–174/400–520 MHz coverage is receive-only, and listening needs no license."
  - q: "Does the GMRS-50X1 really put out 50 watts?"
    a: "Not always. Real-world output famously varies by channel — expect around 40 W on some. The successor GMRS-50V2's 'constant 50W on GMRS' marketing is an implicit admission. Measure output if you can before buying used."
  - q: "What should I check before buying a used GMRS-50X1?"
    a: "Verify it programs cleanly with CHIRP (early units shipped with out-of-range settings bytes — CHIRP bug #7761 — causing failed reads/writes), test transmit on GMRS channels 15–30 with a second radio (some units stop keying on GMRS after reprogramming while FRS still works), check the mic, and prefer packages with the PC04 cable."
---
**The BTECH GMRS-50X1** was a genuinely important radio: the first cheap
50-watt GMRS mobile with a real Part 95E grant, huge 136–174/400–520 MHz
receive coverage, and CHIRP support.[^btech] BTECH has since superseded it
**twice** — the GMRS-50V2 fixed its power consistency and receive filtering,
and the [GMRS-50PRO](/reference/btech-gmrs-50pro/) is the current platform —
so today the 50X1 is a used-market buy at roughly **$80–160**, with wide
variance and thin sold-price data.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0932TH6T9?tag=gophertrunk-20" rel="nofollow sponsored noopener">Renewed on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Our best used/classic GMRS buy — the most capability per used dollar.**
50 W nominal (real output varies by channel — measure it), split tones, wide
receive/scan plus FM broadcast and NOAA, CHIRP support, genuine Part 95E
grant. **Amazon carries it Renewed-only; eBay is the other half of the
market.** Test before buying: the early settings-byte bug (CHIRP #7761) and
the stops-keying-on-GMRS-after-reprogramming failure are both documented.
**GMRS license to transmit: $35, 10 years, no test, covers family.**
Rankings: [best GMRS mobile radios](/best-gmrs-mobile-radios/).
</div>

## Overview

When the 50X1 launched, "50 watts, certified, cheap" was a category of one,
and it built the template BTECH still follows: maximum legal GMRS power, a
wide receive-only scan range (that receive-only lock on transmit is exactly
what keeps an import's Part 95E grant legal), split
[CTCSS](/reference/ctcss/)/[DCS](/reference/dcs/) tones, and open programming
via CHIRP. The 5-row display and fixed (non-detachable) face are dated now,
the menus were always clunky, and the speaker-mic was the era's weak point —
but the fundamentals still cover everything a repeater user needs.

Why it was discontinued is worth knowing because it's your used-buying map:
the GMRS-50V2's redesigned board exists specifically to fix the 50X1's
**inconsistent power output** and audio filtering, and the marketing line
"constant 50 W on GMRS" is an implicit admission that the 50X1 wasn't. Expect
~40 W on some channels from a good unit; measure if you can.

Where to buy: **Amazon lists it only as a "Renewed" unit** — the button above
goes there — and **eBay is the other, often cheaper, half of the used
market** (~$80 sold for a used unit with mic; $150–160 typical asks;
new-old-stock occasionally surfaces around $99–115 at liquidation sellers,
though those listings can be stale).

**License note:** GMRS transmit requires a license — $35 to the
[FCC](/reference/fcc/), valid 10 years, no test, one license covering your
immediate family. Listening across all that receive coverage requires none.

## Buying used: what to check

The 50X1's known failure modes are specific, documented, and testable:

1. **Verify it programs cleanly before buying.** Early units shipped with
   out-of-range settings bytes — CHIRP bug #7761 — whose symptom is failed
   programming reads/writes. It's fixable via CHIRP, but a radio that won't
   read cleanly is telling you something.
2. **Test transmit on GMRS 15–30 with a second radio.** There are documented
   cases of 50X1s that stop keying up on GMRS channels after CHIRP
   reprogramming while FRS channels still work. Don't just watch the TX
   indicator — confirm with a receiver.
3. **Measure power output if possible.** Some units run well under 50 W on
   some channels.
4. **Check the microphone.** BTECH mics of that era were the weak point;
   confirm the included one works.
5. **Prefer packages with the PC04 programming cable.**

Used-market viability, plainly: parts and factory support for a
two-generations-old import are thin, but CHIRP keeps it programmable forever,
the failure modes are known, and at $80–120 a tested unit is the most radio
per used dollar in GMRS — which is why it edges the
[Midland MXT400](/reference/midland-mxt400/) (narrowband-only, no NOAA, no
out-of-box split tones) as our used pick.

## GopherTrunk alternative

FRS and GMRS are analog narrowband [FM](/reference/frequency-modulation/) at
462/467 MHz — trivial for a ~$30 [RTL-SDR](/reference/rtl-sdr/) running free
GopherTrunk, which monitors and records all local FRS/GMRS simplex and
repeater activity. Before hunting used listings, let a dongle tell you
whether your local repeaters are active enough to justify 50 used watts —
and afterward, it's the second radio you use to verify a 50X1 actually keys
up on GMRS 15–30. GopherTrunk cannot transmit; it complements the radio.
[Download GopherTrunk](/downloads.html) and listen first.

## Who it's for

- **Buy a used GMRS-50X1** if you want maximum certified wattage and receive
  coverage for double-digit money, and you can test-drive the unit (or buy
  Renewed with a return window) against the checklist above.
  <a class="btn btn--buy" href="https://www.amazon.com/dp/B0932TH6T9?tag=gophertrunk-20" rel="nofollow sponsored noopener">Renewed on Amazon &rarr;</a>
- **Buy the [GMRS-50PRO](/reference/btech-gmrs-50pro/)** instead if you want
  the current platform — GPS, Bluetooth, constant output — with a warranty,
  for ~$285 new.
- **Skip it** for the [Wouxun KG-1000G Plus](/reference/wouxun-kg-1000g-plus/)
  if you want the proven enthusiast 50-watter new, or a
  [Radioddity DB20-G](/reference/radioddity-db20-g/) if 20 new watts beat 50
  used ones for you. Full rankings:
  [best GMRS mobile radios](/best-gmrs-mobile-radios/).

## Sources

[^btech]: [BTECH GMRS-50X1 product page](https://baofengtech.com/product/gmrs-50x1/) — BTECH, on the 50 W nominal output, receive coverage, split tones, and Part 95E certification; successor: [BTECH GMRS-50V2 — myGMRS shop](https://shop.mygmrs.com/products/btech-gmrs-50v2-50w-gmrs-radio), whose constant-output redesign frames the 50X1's known power variance.
