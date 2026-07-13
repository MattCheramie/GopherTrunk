---
slug: edwin-armstrong
title: Edwin Armstrong
entry_type: person
category: people
description: Edwin Armstrong (1890–1954) was an American engineer who invented the regenerative and superheterodyne receivers and wide-band frequency modulation (FM).
keywords: Edwin Armstrong, FM, frequency modulation, superheterodyne, regeneration, radio inventor, vacuum tube
aka: [Edwin Armstrong, Edwin Howard Armstrong]
autolink: true
infobox:
  - { label: Lived, value: "1890–1954" }
  - { label: Field, value: Electrical engineering }
  - { label: Known for, value: FM, superheterodyne receiver }
see_also: [frequency-modulation, superheterodyne-receiver, reginald-fessenden, intermediate-frequency, broadcast-fm, radio-wave]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/rf-sdr/analog-modulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Edwin_Howard_Armstrong
  - https://ethw.org/Edwin_Howard_Armstrong
---

**Edwin Armstrong** (1890–1954) was an American electrical engineer who invented several
cornerstones of radio — regeneration, the [superheterodyne receiver](/reference/superheterodyne-receiver/),
and most notably wide-band **[frequency modulation](/reference/frequency-modulation/)** —
and whose architectures still shape the front end of every receiver, including the SDRs
GopherTrunk runs on.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="An FM waveform whose cycle spacing varies with the message at constant amplitude, contrasted with noisy AM." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="22" font-size="9" fill="currentColor">FM (Armstrong)</text>
  <path d="M20 50 q4 -20 8 0 q4 -20 8 0 q6 -20 12 0 q8 -20 16 0 q8 -20 16 0 q6 -20 12 0 q4 -20 8 0 q4 -20 8 0 q4 -20 8 0 q6 -20 12 0 q8 -20 16 0 q8 -20 16 0 q6 -20 12 0 q4 -20 8 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="20" y="92" font-size="9" fill="currentColor">resists amplitude noise → clear audio</text>
</svg>
<figcaption>Armstrong invented FM (and the superheterodyne and regenerative receivers), giving radio static-free audio.</figcaption>
</figure>

## Life and work

Armstrong was born in New York City in 1890 and grew up in Yonkers, where as a teenager he
built antennas in the back yard and devoured the wireless literature of the day. He studied
electrical engineering at Columbia University under Michael Pupin and spent essentially his
whole career associated with Columbia, later as a professor. His inventive output began
almost at once: while still a student, around 1912, he worked out why the newly invented
triode vacuum tube could amplify so dramatically and devised the **regenerative** (feedback)
receiver, feeding part of the tube's output back to its input to boost sensitivity by orders
of magnitude, and — pushed further — to make the tube oscillate and generate a clean
continuous wave. During the First World War, as a U.S. Army officer in France trying to
detect faint enemy signals, he invented the **superheterodyne** receiver, which mixes an
incoming signal down to a fixed [intermediate frequency](/reference/intermediate-frequency/)
where it can be filtered and amplified with high, stable gain.

Armstrong grew wealthy from his patents but spent much of his life in gruelling litigation
over them, including a long and draining priority fight with Lee de Forest over
regeneration that the U.S. Supreme Court ultimately decided against him — an outcome most
engineers regarded as a miscarriage.

## Contribution

Armstrong's masterpiece was wide-band **frequency modulation**, developed through the 1930s
and demonstrated publicly in 1935. The prevailing wisdom, backed by a narrow-band analysis,
held that FM offered no advantage over AM. Armstrong showed the opposite by going the other
way: deliberately using a *wide* frequency deviation, he traded bandwidth for a dramatic
improvement in [signal-to-noise ratio](/reference/signal-to-noise-ratio/), because most
noise perturbs a signal's amplitude while FM carries information in its instantaneous
frequency. A limiter at the receiver simply clips away the amplitude variation — and with it
the static — leaving audio of a clarity AM could not touch.[^ethw] This was an early and
influential example of the bandwidth-versus-noise trade that
[Claude Shannon](/reference/claude-shannon/) would later formalise. Armstrong built
experimental FM stations, notably at Alpine, New Jersey, to prove the system at scale, and
his continuous-wave work also complemented the voice-radio experiments of
[Reginald Fessenden](/reference/reginald-fessenden/).

## Legacy

FM broadcasting, established over fierce commercial resistance from AM and television
interests, went on to dominate high-fidelity radio and to carry the audio of the
[broadcast FM](/reference/broadcast-fm/) band worldwide. Frequency and phase modulation
underlie a great deal of two-way and digital radio as well: the C4FM and related four-level
FM schemes used in P25 and digital-voice systems are direct descendants of Armstrong's idea
that shifting a carrier's frequency is a robust way to carry information. The superheterodyne
principle he invented is even more pervasive — nearly every receiver built since, from a
pocket transistor set to a modern SDR tuner chip, mixes signals to an intermediate frequency
exactly as he prescribed. Exhausted and impoverished by his patent battles, Armstrong took
his own life in 1954; his widow continued the litigation and eventually won, vindicating his
claims. His inventions remain among the most important in the history of radio, and
GopherTrunk's very ability to tune a distant channel rests on the superheterodyne front end
he conceived in a wartime trench.

## Sources

[^wiki]: [Edwin Howard Armstrong](https://en.wikipedia.org/wiki/Edwin_Howard_Armstrong) — Wikipedia, for biography and his invention of FM and the superheterodyne receiver.
[^ethw]: [Edwin Howard Armstrong](https://ethw.org/Edwin_Howard_Armstrong) — Engineering and Technology History Wiki (IEEE), for regeneration, the superheterodyne, and wide-band FM.
