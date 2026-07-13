---
slug: reginald-fessenden
title: Reginald Fessenden
entry_type: person
category: people
description: Reginald Fessenden (1866–1932) was a Canadian-American inventor credited with the first audio radio transmissions, pioneering amplitude modulation and continuous-wave radio.
keywords: Reginald Fessenden, AM radio, voice transmission, continuous wave, radio history, heterodyne, alternator
aka: [Reginald Fessenden, Fessenden]
autolink: true
infobox:
  - { label: Lived, value: "1866–1932" }
  - { label: Field, value: Radio engineering }
  - { label: Known for, value: Early voice (AM) transmission }
see_also: [amplitude-modulation, guglielmo-marconi, edwin-armstrong, carrier-wave, continuous-wave, superheterodyne-receiver]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/rf-sdr/analog-modulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Reginald_Fessenden
  - https://ethw.org/Reginald_Fessenden
---

**Reginald Fessenden** (1866–1932) was a Canadian-American inventor credited with some of
the first **audio radio transmissions**, moving radio beyond Morse code to voice and music
using [amplitude modulation](/reference/amplitude-modulation/) on a
[continuous wave](/reference/continuous-wave/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A carrier whose amplitude envelope follows a voice waveform, illustrating the first AM voice transmission." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q4 -16 8 0 q4 -22 8 0 q4 -16 8 0 q4 -8 8 0 q4 -16 8 0 q4 -22 8 0 q4 -16 8 0 q4 -8 8 0 q4 -16 8 0 q4 -22 8 0 q4 -16 8 0 q4 -8 8 0 q4 -16 8 0 q4 -22 8 0 q4 -16 8 0 q4 -8 8 0 q4 -16 8 0 q4 -22 8 0 q4 -16 8 0 q4 -8 8 0 q4 -16 8 0 q4 -22 8 0 q4 -16 8 0 q4 -8 8 0 q4 -16 8 0 q4 -22 8 0" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <path d="M20 60 C 90 20, 150 24, 210 60 S 330 96, 400 60 S 440 44, 440 60" fill="none" stroke="currentColor" stroke-width="1.3" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <text x="230" y="106" text-anchor="middle" font-size="9" fill="currentColor">voice on the amplitude envelope</text>
</svg>
<figcaption>Fessenden pioneered AM voice radio, credited with the first audio (voice and music) broadcast in 1906.</figcaption>
</figure>

## Life and work

Fessenden was born in 1866 in Bolton-Est, Quebec, the son of an Anglican clergyman. Largely
self-taught in the sciences, he moved to New York as a young man and talked his way into a
job with Thomas Edison, working at Edison's laboratory before moving on to Westinghouse. He
then held professorships in electrical engineering at Purdue and at the Western University of
Pennsylvania (now the University of Pittsburgh). From 1900 he worked for the U.S. Weather
Bureau on wireless, and it was in that period, aiming to transmit voice, that he made his key
break with the prevailing approach to radio.

Where [Marconi](/reference/guglielmo-marconi/) and nearly everyone else generated radio with
noisy, damped spark discharges suitable only for on-off Morse keying, Fessenden insisted that
carrying speech required a smooth, undamped **continuous wave** whose amplitude could be
varied continuously by the voice. To produce one he commissioned General Electric — through a
young engineer named Ernst Alexanderson — to build a high-frequency alternator that spun fast
enough to generate radio-frequency current directly. On Christmas Eve 1906, from Brant Rock,
Massachusetts, he is said to have transmitted a program of speech and music, including a
violin performance and a Bible reading, heard by shipboard operators along the Atlantic
coast — often cited as the first audio radio broadcast, though some historical details are
debated.

## Contribution

Fessenden's lasting technical legacy is twofold. First, he championed
[amplitude modulation](/reference/amplitude-modulation/) of a continuous
[carrier wave](/reference/carrier-wave/) as the way to send audio, establishing the basic
scheme that all AM broadcasting would use. Second, and arguably more influential for receiver
design, he invented and patented the **heterodyne principle**: mixing an incoming signal with
a locally generated oscillation to produce a beat at a lower, more tractable frequency.[^ethw]
At the time no oscillator was stable enough to exploit it fully, but once the vacuum tube
matured the idea became the foundation of the superheterodyne receiver perfected by
[Edwin Armstrong](/reference/edwin-armstrong/) — and, through it, of essentially every radio
receiver since, including the tuner front end of a modern SDR. Fessenden was a prolific and
combative inventor, holding hundreds of patents ranging from radio to sonar (his fathometer
depth sounder) to early television concepts, and he fought long legal battles over the credit
and royalties he felt he was owed.

## Legacy

Fessenden is remembered as a pioneer of voice radio, complementing
[Marconi](/reference/guglielmo-marconi/)'s telegraphy and setting the stage for
broadcasting as a mass medium. His continuous-wave, amplitude-modulated transmission is the
conceptual ancestor of AM broadcast radio, and his heterodyne insight lives on every time a
receiver mixes a signal down to an intermediate frequency to be filtered and demodulated.
GopherTrunk does not decode broadcast AM, but the frequency-conversion architecture that lets
it tune and separate trunking channels descends directly from the heterodyne principle
Fessenden patented more than a century ago.

## Sources

[^wiki]: [Reginald Fessenden](https://en.wikipedia.org/wiki/Reginald_Fessenden) — Wikipedia, for biography and his pioneering of AM voice radio.
[^ethw]: [Reginald Fessenden](https://ethw.org/Reginald_Fessenden) — Engineering and Technology History Wiki (IEEE), for the continuous-wave voice transmission and the heterodyne principle.
