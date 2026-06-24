---
slug: sensor
title: Sensor
entry_type: hardware
category: hw-microcontrollers
description: A sensor is a device that detects a physical quantity — temperature, light, motion, pressure — and converts it into an electrical signal a microcontroller can read and act on.
keywords: sensor, transducer, temperature, accelerometer, light, pressure, analog, digital, I2C sensor
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

## Overview

Sensors are the input side of nearly every embedded device. Some output a raw analog voltage that a [microcontroller](/reference/microcontroller/) digitizes with its [ADC](/reference/analog-to-digital-converter/); many modern sensors include their own electronics and present a clean digital reading over [I²C](/reference/i2c/) or [SPI](/reference/spi/), or as a simple [GPIO](/reference/gpio/) signal. Common examples include thermistors, accelerometers and gyroscopes, photodiodes, microphones, and pressure and gas sensors.

## Where it fits

Sensors give an [embedded system](/reference/embedded-system/) awareness of the physical world; paired with network connectivity they are the data source behind the [Internet of Things](/reference/internet-of-things/). The microcontroller reads the sensor, decides what to do, and drives an output — closing the loop between sensing and action. A weather station node reporting temperature and humidity over a radio link is a textbook sensor-plus-MCU device.

## Sources

[^wiki]: [Sensor](https://en.wikipedia.org/wiki/Sensor) — Wikipedia, on sensors as transducers of physical quantities.
