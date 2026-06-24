---
slug: avr-atmega
title: AVR / ATmega
entry_type: hardware
category: hw-microcontrollers
description: AVR is an 8-bit RISC microcontroller architecture from Atmel (now Microchip); its ATmega line, especially the ATmega328P, powers the classic Arduino Uno and countless hobby projects.
keywords: AVR, ATmega, ATmega328P, ATtiny, Atmel, Microchip, Arduino Uno, 8-bit RISC, AVR-GCC
aka: [AVR, ATmega]
infobox:
  - { label: Type, value: 8-bit microcontroller architecture }
  - { label: Core, value: AVR RISC }
  - { label: Vendor, value: Atmel (now Microchip) }
  - { label: Famous part, value: ATmega328P (Arduino Uno) }
  - { label: Language, value: C, C++, assembly }
see_also: [arduino, microcontroller, pic-microcontroller, stm32, firmware, in-system-programming]
related_lessons:
  - { title: "Arduino", url: /learn/intro-hardware/arduino/ }
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/AVR_microcontrollers
---

**AVR** is an 8-bit RISC [microcontroller](/reference/microcontroller/) architecture created by Atmel and now owned by Microchip; **ATmega** is its best-known product line.[^wiki]

## Overview

The AVR design pairs a simple, fast 8-bit core with on-chip flash, SRAM, and EEPROM, plus [GPIO](/reference/gpio/), timers with [PWM](/reference/pulse-width-modulation/), an [ADC](/reference/analog-to-digital-converter/), and [UART](/reference/uart/)/[SPI](/reference/spi/)/[I²C](/reference/i2c/) peripherals. Sibling lines include the tiny ATtiny and the larger ATmega. The ATmega328P is iconic because it is the chip on the original [Arduino](/reference/arduino/) Uno, which made AVR the gateway architecture for a generation of makers.

## Where it fits

AVR shines where 8 bits are plenty: low cost, low power, and a friendly toolchain (AVR-GCC, the Arduino IDE). Code is written in [C](/reference/c-language/) or [C++](/reference/cpp-language/) and flashed via [in-system programming](/reference/in-system-programming/) over SPI. When a project needs 32-bit performance, more memory, or built-in radios, designers step up to an [STM32](/reference/stm32/) or [ESP32](/reference/esp32/); the rival 8-bit family is the [PIC](/reference/pic-microcontroller/).

## Sources

[^wiki]: [AVR microcontrollers](https://en.wikipedia.org/wiki/AVR_microcontrollers) — Wikipedia, on the AVR architecture and ATmega line.
