---
slug: glossary
title: Glossary of RF & SDR terms
description: Plain-language definitions of radio and software-defined-radio terms — frequency, dBm, IQ, FSK, control channel, vocoder, and dozens more — each cross-linked to the GopherTrunk learning path.
keywords: SDR glossary, RF terms, radio glossary, dBm definition, IQ data, control channel, talkgroup, FSK, vocoder, P25 DMR terms, SDR dictionary
level: beginner
status: full
lesson_standalone: true
---

# Glossary of RF & SDR terms

Every term used across the [learning path](/learn/), defined in plain language and
linked to the lesson where it's explained in full. Skim it as a refresher, or use
your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are grouped by theme and
ordered roughly from fundamentals to systems.

## Radio wave fundamentals

**Amplitude** — The strength or "height" of a radio wave; more amplitude means a
stronger signal. Your SDR reports it as a power level. See
[What is a radio wave?](/learn/radio-waves/)

**Carrier** — A plain, steady radio wave with no information on it yet. Modulation
changes the carrier to carry a message. See [What is a radio wave?](/learn/radio-waves/)

**Electromagnetic spectrum** — The full range of electromagnetic waves, from radio
through light to X-rays. Radio occupies roughly 3 kHz–300 GHz. See
[What is a radio wave?](/learn/radio-waves/)

**Frequency** — How many times per second a wave cycles, measured in **hertz (Hz)**.
Higher frequency = shorter wavelength. See [What is a radio wave?](/learn/radio-waves/)

**Hertz (Hz)** — The unit of frequency: one cycle per second. Common multiples are
kHz (thousand), MHz (million), and GHz (billion). See
[Frequency, bands & the spectrum](/learn/frequency-and-spectrum/)

**Modulation** — Deliberately varying a carrier's amplitude, frequency, or phase to
encode information. See [Digital modulation & constellations](/learn/digital-modulation/)

**Wavelength (λ)** — The physical length of one wave cycle, in metres. Approximated
for radio by *λ ≈ 300 ÷ frequency in MHz*. See [What is a radio wave?](/learn/radio-waves/)

**Band** — A defined, contiguous range of frequencies assigned to a use (e.g. the FM
broadcast band). See [Frequency, bands & the spectrum](/learn/frequency-and-spectrum/)

**HF / VHF / UHF** — Conventional names for frequency ranges: High Frequency
(3–30 MHz), Very High Frequency (30–300 MHz), Ultra High Frequency (300 MHz–3 GHz).
See [Frequency, bands & the spectrum](/learn/frequency-and-spectrum/)

## Power, noise & decibels

**dB (decibel)** — A logarithmic *ratio* between two powers. +10 dB = ×10 power;
+3 dB ≈ ×2. Used for gain and loss. See [Decibels & signal power](/learn/decibels/)

**dBm** — Absolute power referenced to 1 milliwatt. Received signals are negative
numbers; closer to zero is stronger. See [Decibels & signal power](/learn/decibels/)

**dBFS** — Decibels relative to digital full scale, used inside the SDR. 0 dBFS is
the ADC's ceiling; real samples sit below it. See [Decibels & signal power](/learn/decibels/)

**Gain** — Amplification, expressed in dB. Also the SDR setting that controls how
strong the signal is at the ADC. See [Gain, AGC & avoiding overload](/learn/gain-and-agc/)

**Noise floor** — The ever-present background level of receiver and environmental
noise, in dBm. A signal must rise above it to be usable. See
[Decibels & signal power](/learn/decibels/)

**Path loss** — How much a signal weakens (in dB) as it travels from transmitter to
receiver. See [Decibels & signal power](/learn/decibels/)

**SNR (signal-to-noise ratio)** — The gap in dB between a signal and the noise floor.
Digital modes need a minimum SNR to decode. See [Decibels & signal power](/learn/decibels/)

## Signals & modulation

**Baud (symbol rate)** — Symbols transmitted per second. Different from bit rate when
each symbol carries multiple bits. See [Symbols, baud & bitrate](/learn/symbols-and-baud/)

**Bit rate** — Bits per second = symbol rate × bits per symbol. See
[Symbols, baud & bitrate](/learn/symbols-and-baud/)

**Symbol** — One transmitted state of the carrier, carrying one or more bits. See
[Digital modulation & constellations](/learn/digital-modulation/)

**AM / FM / SSB** — Analog modulations that vary amplitude (AM), frequency (FM), or
send a single sideband (SSB). See [Analog modulation](/learn/analog-modulation/)

**FSK (frequency-shift keying)** — Digital modulation that switches the carrier
between set frequencies. **4FSK** (four levels) underlies P25 C4FM and DMR. See
[Digital modulation & constellations](/learn/digital-modulation/)

**PSK (phase-shift keying)** — Digital modulation that switches the carrier's phase;
QPSK uses four phases. See [Digital modulation & constellations](/learn/digital-modulation/)

**QAM** — Modulation varying both phase and amplitude to pack many bits per symbol;
needs higher SNR. See [Digital modulation & constellations](/learn/digital-modulation/)

**C4FM** — The four-level FSK used by P25 Phase 1. See
[Digital modulation & constellations](/learn/digital-modulation/)

**Constellation diagram** — A plot of symbols in the IQ plane; tight clusters mean a
clean signal. See [Digital modulation & constellations](/learn/digital-modulation/)

**Eye diagram** — Overlaid symbol waveforms vs. time; a wide-open "eye" means good
decoding margin. See [Digital modulation & constellations](/learn/digital-modulation/)

## Software-defined radio

**SDR (software-defined radio)** — A radio whose tuning, filtering, and demodulation
happen in software. See [What is software-defined radio?](/learn/what-is-sdr/)

**ADC (analog-to-digital converter)** — The chip that samples the analog signal into
digital numbers. Clipping it causes distortion. See [What is SDR?](/learn/what-is-sdr/)

**IQ samples** — Pairs of numbers (In-phase and Quadrature) that capture a signal's
amplitude and phase together. The SDR's output. See [IQ data & complex signals](/learn/iq-data/)

**Sample rate** — How many samples per second the SDR produces; sets how much
spectrum (bandwidth) you can see at once. See
[Sample rate, bandwidth & Nyquist](/learn/sample-rate-nyquist/)

**Nyquist limit** — The rule that you must sample at least twice the bandwidth you
want to capture. See [Sample rate, bandwidth & Nyquist](/learn/sample-rate-nyquist/)

**Bandwidth** — The width of spectrum, in Hz, occupied by a signal or captured by a
receiver. See [Sample rate, bandwidth & Nyquist](/learn/sample-rate-nyquist/)

**AGC (automatic gain control)** — Circuitry or software that adjusts gain on its own.
See [Gain, AGC & avoiding overload](/learn/gain-and-agc/)

**PPM (parts per million)** — A measure of an SDR's frequency error; correcting it
aligns the radio to true frequency. See [Calibration & troubleshooting](/learn/calibration-troubleshooting/)

**RTL-SDR / HackRF / Airspy** — Common SDR hardware. RTL-SDR is the cheap entry
point; HackRF and Airspy add bandwidth, sensitivity, or transmit. See
[SDR hardware](/learn/sdr-hardware/) and the [Hardware guide](/hardware.html)

## DSP

**DSP (digital signal processing)** — Manipulating sampled signals with math:
filtering, tuning, demodulating. See [The demodulation pipeline](/learn/demodulation-pipeline/)

**FFT (Fast Fourier Transform)** — The algorithm that turns IQ samples into a
spectrum view; the basis of the waterfall. See [The FFT & reading a waterfall](/learn/fft-and-waterfall/)

**Waterfall** — A scrolling spectrum display showing signal strength across frequency
over time. See [The FFT & reading a waterfall](/learn/fft-and-waterfall/)

**Filtering** — Keeping a chosen slice of spectrum and discarding the rest. See
[Filtering & decimation](/learn/filtering-decimation/)

**Decimation** — Reducing the sample rate after filtering to save CPU. See
[Filtering & decimation](/learn/filtering-decimation/)

**Demodulation** — Recovering the original message from a modulated signal. See
[The demodulation pipeline](/learn/demodulation-pipeline/)

**Clock recovery** — Finding the timing of a digital signal so each symbol is sampled
at the right instant. See [Clock recovery & symbol timing](/learn/clock-recovery/)

## Digital voice & trunked radio

**Vocoder** — A speech codec that compresses voice into tiny data frames; IMBE (P25)
and AMBE+2 (DMR) are common. See [Vocoders](/learn/vocoders/) and the
[Vocoders reference](/vocoders.html)

**Trunked radio** — A system that shares a pool of frequencies, assigning a channel
per call. See [What is trunked radio?](/learn/what-is-trunking/)

**Conventional radio** — Non-trunked radio where each group uses a fixed frequency.
See [What is trunked radio?](/learn/what-is-trunking/)

**Control channel** — The data-only frequency that coordinates a trunked system and
announces channel assignments. GopherTrunk decodes it first. See
[What is trunked radio?](/learn/what-is-trunking/)

**Voice channel** — A frequency temporarily assigned to carry a call. See
[What is trunked radio?](/learn/what-is-trunking/)

**Talkgroup** — A virtual channel identifying a group of users; you follow talkgroups,
not frequencies. See [What is trunked radio?](/learn/what-is-trunking/)

**Affiliation** — A radio registering with the trunked system over the control
channel. See [What is trunked radio?](/learn/what-is-trunking/)

**Radio ID** — The unique identifier of an individual radio transmitting on a system.
See the [Radio IDs panel](/radio-ids.html)

**P25 / DMR / NXDN / TETRA** — Digital land-mobile-radio standards GopherTrunk
decodes. See [The digital protocol landscape](/learn/protocol-landscape/)

**Encryption** — Scrambling that prevents voice from being decoded; encrypted
talkgroups stay silent. See [Encryption & what you can decode](/learn/encryption/)

**APRS / AIS / ADS-B** — Non-trunked data signals (amateur position, ship tracking,
aircraft tracking) GopherTrunk can also decode. See [Other signals you'll meet](/learn/other-signals/)

---

Didn't find a term? It's probably explained in one of the
[learning-path lessons](/learn/) — or open an
[issue on GitHub](https://github.com/MattCheramie/GopherTrunk/issues) and we'll add it.
