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

## Overview

A pin toggled at, say, 20 kHz that is high 25% of the time delivers, on average, a quarter of the supply voltage. Because the switching is far faster than the load can respond, an LED, motor, or heater simply sees the average. A [microcontroller](/reference/microcontroller/) generates PWM in hardware timers, so it costs no CPU once configured, and the duty cycle can be changed on the fly. Servos and many [sensors](/reference/sensor/) also use PWM as their signaling format.

## What it's for

PWM is how an MCU produces an effectively analog output from purely digital [GPIO](/reference/gpio/): dimming LEDs, controlling motor and fan speed, positioning servos, and — with a simple low-pass filter — approximating a [digital-to-analog converter](/reference/digital-to-analog-converter/). On an [Arduino](/reference/arduino/), `analogWrite()` is PWM under the hood. Duty cycle is typically updated in the main loop or from a timer [interrupt](/reference/interrupt/).

## Sources

[^wiki]: [Pulse-width modulation](https://en.wikipedia.org/wiki/Pulse-width_modulation) — Wikipedia, on duty cycle and PWM applications.
