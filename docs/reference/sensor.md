---
slug: sensor
title: Sensor
entry_type: hardware
category: hw-microcontrollers
description: A sensor is a device that detects a physical quantity — temperature, light, motion, pressure — and converts it into an electrical signal a microcontroller can read and act on, either as a raw analog voltage or as a digital reading over a bus.
keywords: sensor, transducer, temperature, accelerometer, light, pressure, gas, analog, digital, I2C sensor, ADC
infobox:
  - { label: Type, value: Input transducer }
  - { label: Senses, value: Heat, light, motion, pressure, gas }
  - { label: Output, value: Analog or digital signal }
  - { label: Read via, value: ADC, I²C, SPI, GPIO }
see_also: [microcontroller, analog-to-digital-converter, i2c, spi, gpio, internet-of-things]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Sensor
---

**A sensor** is a device that detects a physical quantity — temperature, light, motion, pressure, gas — and converts it into an electrical signal a computer can read.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 156" role="img" aria-label="Two signal chains from the physical world into a microcontroller. In the top path, a physical quantity drives a transducer to a raw analog voltage, which an analog-to-digital converter turns into numbers the microcontroller reads. In the bottom path, a smart sensor with built-in electronics outputs an already-digital reading over an I-squared-C or SPI bus straight to the microcontroller." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="22" y="30" width="70" height="30" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="132" y="30" width="70" height="30" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="22" y="96" width="120" height="30" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="330" y="58" width="108" height="40" rx="4" fill="currentColor" fill-opacity="0.16"/>
    <line x1="92" y1="45" x2="132" y2="45"/>
    <path d="M132 45 L122 40 M132 45 L122 50" stroke-width="1.1"/>
    <line x1="202" y1="45" x2="330" y2="66"/>
    <path d="M330 66 L319 63 M330 66 L322 72" stroke-width="1.1"/>
    <line x1="142" y1="111" x2="330" y2="90"/>
    <path d="M330 90 L319 89 M330 90 L321 96" stroke-width="1.1"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="57" y="42" font-size="7.5" font-weight="600">Transducer</text>
    <text x="57" y="53" font-size="7" fill-opacity="0.85">analog volts</text>
    <text x="167" y="42" font-size="7.5" font-weight="600">ADC</text>
    <text x="167" y="53" font-size="7" fill-opacity="0.85">to numbers</text>
    <text x="82" y="108" font-size="7.5" font-weight="600">Smart sensor</text>
    <text x="82" y="119" font-size="7" fill-opacity="0.85">digital over I&#178;C / SPI</text>
    <text x="384" y="76" font-size="8.5" font-weight="600">Microcontroller</text>
    <text x="384" y="88" font-size="7" fill-opacity="0.85">reads &amp; acts</text>
    <text x="235" y="146" font-size="7.5" fill-opacity="0.9">analog path digitizes; digital sensors skip the ADC</text>
  </g>
</svg>
<figcaption>Two ways a reading reaches the chip: a plain transducer outputs an analog voltage that the microcontroller's ADC turns into numbers, while a "smart" sensor with its own electronics hands over an already-digital value over I²C or SPI. Either way the sensor is the input side of the sense-decide-act loop.</figcaption>
</figure>

## Overview

Sensors are the input side of nearly every embedded device. Some output a raw analog voltage that a [microcontroller](/reference/microcontroller/) digitizes with its [ADC](/reference/analog-to-digital-converter/); many modern sensors include their own electronics and present a clean digital reading over [I²C](/reference/i2c/) or [SPI](/reference/spi/), or as a simple [GPIO](/reference/gpio/) signal. Common examples include thermistors, accelerometers and gyroscopes, photodiodes, microphones, and pressure and gas sensors.

Technically a sensor is a *transducer* — it converts energy from one form (heat, light, motion) into an electrical signal. Whether that signal needs the extra analog-to-digital step or arrives ready-made as data is the main practical distinction between an analog sensor and a digital one.

## Analog versus digital sensors

The output format shapes how the microcontroller reads it:

| Trait | Analog sensor | Digital sensor |
|-------|---------------|----------------|
| Output | Continuous voltage | Numbers over a bus |
| MCU reads via | On-chip [ADC](/reference/analog-to-digital-converter/) | [I²C](/reference/i2c/) / [SPI](/reference/spi/) / GPIO |
| Extra circuitry | Often needs conditioning | Built into the sensor |
| Example | Thermistor, photocell | BME280, MPU-6050 |

## Where it fits

Sensors give an [embedded system](/reference/embedded-system/) awareness of the physical world; paired with network connectivity they are the data source behind the [Internet of Things](/reference/internet-of-things/). The microcontroller reads the sensor, decides what to do, and drives an output — closing the loop between sensing and action. A weather-station node reporting temperature and humidity over a radio link is a textbook sensor-plus-MCU device — and precisely the kind of small transmitter whose signals GopherTrunk decodes.

## Sources

[^wiki]: [Sensor](https://en.wikipedia.org/wiki/Sensor) — Wikipedia, on sensors as transducers of physical quantities.
