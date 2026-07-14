---
slug: pulse-width-modulation
title: Pulse-width modulation (PWM)
entry_type: concept
category: hw-microcontrollers
description: Pulse-width modulation encodes an analog level as the on/off duty cycle of a fast digital square wave, letting a microcontroller dim LEDs, drive motors, or synthesize voltages with one pin.
keywords: PWM, pulse-width modulation, duty cycle, motor control, LED dimming, servo, analog output, square wave
aka: [PWM]
infobox:
  - { label: Type, value: Signaling technique }
  - { label: Encodes, value: Level as duty cycle }
  - { label: Output, value: Digital square wave }
  - { label: Uses, value: LEDs, motors, servos, DAC }
see_also: [microcontroller, gpio, digital-to-analog-converter, interrupt, sensor, arduino]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Pulse-width_modulation
---

**Pulse-width modulation (PWM)** encodes an analog level as the fraction of time a fast digital square wave spends switched on — its **duty cycle**.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 220" role="img" aria-label="Three digital square waves at 25, 50, and 75 percent duty cycle. Each is fully on or off, but a dashed line marks the average voltage the load actually sees, rising from about a quarter to about three quarters of the supply as the on-time widens." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="end" font-weight="600">
    <text x="46" y="46">25%</text>
    <text x="46" y="106">50%</text>
    <text x="46" y="166">75%</text>
  </g>
  <polyline points="70,40 100,40 100,64 190,64 190,40 220,40 220,64 310,64 310,40 340,40 340,64 430,64" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="70" y1="58" x2="430" y2="58" stroke="currentColor" stroke-width="1.2" stroke-dasharray="5 3" stroke-opacity="0.8"/>
  <text x="438" y="61" font-size="7.5" fill="currentColor" fill-opacity="0.85" text-anchor="start">≈¼</text>
  <polyline points="70,100 130,100 130,124 190,124 190,100 250,100 250,124 310,124 310,100 370,100 370,124 430,124" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="70" y1="112" x2="430" y2="112" stroke="currentColor" stroke-width="1.2" stroke-dasharray="5 3" stroke-opacity="0.8"/>
  <text x="438" y="115" font-size="7.5" fill="currentColor" fill-opacity="0.85" text-anchor="start">≈½</text>
  <polyline points="70,160 160,160 160,184 190,184 190,160 280,160 280,184 310,184 310,160 400,160 400,184 430,184" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="70" y1="166" x2="430" y2="166" stroke="currentColor" stroke-width="1.2" stroke-dasharray="5 3" stroke-opacity="0.8"/>
  <text x="438" y="169" font-size="7.5" fill="currentColor" fill-opacity="0.85" text-anchor="start">≈¾</text>
  <text x="240" y="208" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">wider on-time → higher average — the load is too slow to see the switching (dashed = average)</text>
</svg>
<figcaption>The pin is only ever fully on or fully off, but it switches far faster than the load can respond, so an LED, motor, or heater sees only the average — the dashed line. Widen the on-time (the duty cycle) and that average rises, giving an effectively analog output from a purely digital pin.</figcaption>
</figure>

## Overview

A pin toggled at, say, 20 kHz that is high 25% of the time delivers, on average, a quarter of the supply voltage. Because the switching is far faster than the load can respond, an LED, motor, or heater simply sees the average. A [microcontroller](/reference/microcontroller/) generates PWM in hardware timers, so it costs no CPU once configured, and the duty cycle can be changed on the fly. Servos and many [sensors](/reference/sensor/) also use PWM as their signaling format.

## What it's for

PWM is how an MCU produces an effectively analog output from purely digital [GPIO](/reference/gpio/): dimming LEDs, controlling motor and fan speed, positioning servos, and — with a simple low-pass filter — approximating a [digital-to-analog converter](/reference/digital-to-analog-converter/). On an [Arduino](/reference/arduino/), `analogWrite()` is PWM under the hood. Duty cycle is typically updated in the main loop or from a timer [interrupt](/reference/interrupt/).

## Sources

[^wiki]: [Pulse-width modulation](https://en.wikipedia.org/wiki/Pulse-width_modulation) — Wikipedia, on duty cycle and PWM applications.
