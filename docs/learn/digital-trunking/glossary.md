---
slug: glossary
title: Glossary of digital & trunked radio terms
description: Plain-language definitions of digital and trunked radio terms — control channel, talkgroup, vocoder, P25, DMR, TETRA, NXDN, FEC, encryption and more — cross-linked to lessons.
keywords: trunked radio glossary, P25 terms, DMR glossary, control channel, talkgroup, vocoder, NAC, color code, TDMA, FDMA, P25 Phase 2, digital radio dictionary
level: beginner
status: full
---

# Glossary of digital & trunked radio terms

Every term used across the [Digital & Trunked Radio path](/learn/digital-trunking/),
defined in plain language and linked to the lesson where it's explained in full.
Skim it as a refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word.
Terms are grouped by theme and ordered roughly from concepts to specific systems.

<div class="tldr" markdown="1">
<span class="tldr__label">How to use this page</span>
This is a reference, not a reading order. If you're new, start with
[why radio went digital](/learn/digital-trunking/why-radio-went-digital/) and follow
the path; come back here whenever a piece of jargon trips you up. Terms link to the
lesson that explains them, and many also have a live panel in the app you can watch.
</div>

## Trunking concepts

**Trunking** — A system where many user groups share a small pool of frequencies,
with a computer assigning a free channel to each call on demand and reclaiming it
afterward. See [Conventional vs trunked](/learn/digital-trunking/conventional-vs-trunked/)
and [Birth of trunking](/learn/digital-trunking/birth-of-trunking/)

**Conventional radio** — Non-trunked radio where each group uses a fixed, permanent
frequency. Simple to scan, but wasteful of spectrum. See
[Conventional vs trunked](/learn/digital-trunking/conventional-vs-trunked/)

**Control channel** — The data-only frequency that coordinates a trunked system,
announcing call setups and the voice channel each is assigned to. GopherTrunk decodes
it first. See [The control channel](/learn/digital-trunking/the-control-channel/) and
the [CC Activity panel](/cc-activity.html)

**Voice / traffic channel** — A frequency temporarily assigned to carry an actual
call. Released back to the pool when the call ends. See
[The control channel](/learn/digital-trunking/the-control-channel/)

**Talkgroup** — A virtual channel — a number with a label, like "Police Dispatch" —
identifying a group of users. You follow talkgroups, not frequencies. See
[Talkgroups, IDs & affiliation](/learn/digital-trunking/talkgroups-ids-affiliation/)

**Radio ID / Unit ID** — The unique identifier of an individual radio transmitting on
a system, distinct from the talkgroup it's using. See
[Talkgroups, IDs & affiliation](/learn/digital-trunking/talkgroups-ids-affiliation/)
and the [Radio IDs panel](/radio-ids.html)

**Affiliation / Registration** — A radio registering with the trunked system over the
control channel when it powers on or changes talkgroups, so the system knows it's
active and where to route calls. See
[Talkgroups, IDs & affiliation](/learn/digital-trunking/talkgroups-ids-affiliation/)

**Channel grant** — The control-channel message that tells radios "this talkgroup,
go to this voice channel." The event that starts every call. See
[Anatomy of a call](/learn/digital-trunking/anatomy-of-a-call/)

**Hang time** — A short interval after a transmission ends during which the system
keeps the voice channel reserved for the talkgroup, so a quick reply doesn't need a
fresh grant. See [Anatomy of a call](/learn/digital-trunking/anatomy-of-a-call/)

**Message trunking** — A grant strategy where a voice channel stays assigned to a
talkgroup for an entire conversation (including hang time), not just one
transmission. See [Trunking flavors](/learn/digital-trunking/trunking-flavors/)

**Transmission trunking** — A grant strategy where the voice channel is released the
instant each transmission ends, so every push-to-talk needs a new grant. More
efficient, but harder to follow. See
[Trunking flavors](/learn/digital-trunking/trunking-flavors/)

**Dedicated control channel** — A trunking design where one frequency is permanently
set aside to carry only control data. Most large P25 and Motorola systems work this
way. See [Trunking flavors](/learn/digital-trunking/trunking-flavors/)

**Distributed control channel** — A design where the control role rotates among
channels or shares time with voice, rather than living on one fixed frequency. Common
in DMR Tier III and some LTR-style systems. See
[Trunking flavors](/learn/digital-trunking/trunking-flavors/)

## Sites, coverage & roaming

**Site** — One physical transmitter location (one set of towers and channels) within
a larger system. A system can have many sites. See
[Sites, simulcast & roaming](/learn/digital-trunking/sites-simulcast-roaming/)

**Simulcast** — Several transmitters broadcasting the *same* signal on the *same*
frequency at once, time-aligned to cover a wide area as if it were one site. Powerful,
but the overlapping signals can stress a decoder. See
[Sites, simulcast & roaming](/learn/digital-trunking/sites-simulcast-roaming/) and
[Multisite & simulcast in practice](/learn/digital-trunking/multisite-and-simulcast-in-practice/)

**Roaming** — A radio automatically moving from one site to another as the user
travels, re-affiliating at each new site. See
[Sites, simulcast & roaming](/learn/digital-trunking/sites-simulcast-roaming/)

**WACN (Wide Area Communications Network)** — In P25, a top-level identifier for a
network operator's whole system-of-systems. Combined with the System ID it uniquely
names a network. See
[Identifying the system](/learn/digital-trunking/identifying-the-system/)

**System ID** — A number identifying a particular trunked system, used (with the WACN
in P25) to tell one system apart from another on the air. See
[Identifying the system](/learn/digital-trunking/identifying-the-system/)

**RFSS (RF Sub-System)** — A P25 grouping of one or more sites under common control,
sitting between the whole system and an individual site. See
[Sites, simulcast & roaming](/learn/digital-trunking/sites-simulcast-roaming/)

**NAC (Network Access Code)** — A 12-bit code carried in every P25 frame, acting like
a "squelch code" so receivers ignore signals from other systems on the same
frequency. See
[Identifying the system](/learn/digital-trunking/identifying-the-system/)

**Color code** — DMR's equivalent of the NAC: a small number (0–15) that distinguishes
co-channel systems so radios only act on their own. See
[Identifying the system](/learn/digital-trunking/identifying-the-system/)

## Digital voice & vocoders

**Vocoder** — A speech codec that analyzes voice and compresses it into tiny data
frames for transmission, then reconstructs it at the other end. See
[Voice to bits: vocoders](/learn/digital-trunking/voice-to-bits-vocoders/) and the
[Vocoders panel](/vocoders.html)

**IMBE (Improved Multi-Band Excitation)** — The vocoder used by P25 Phase 1,
encoding voice at 4.4 kbit/s plus error correction. See
[Voice to bits: vocoders](/learn/digital-trunking/voice-to-bits-vocoders/)

**AMBE+2** — A later, more efficient vocoder used by DMR, NXDN, and P25 Phase 2,
fitting voice into a smaller data rate. See
[Voice to bits: vocoders](/learn/digital-trunking/voice-to-bits-vocoders/)

**ACELP (Algebraic Code-Excited Linear Prediction)** — The vocoder family used by
TETRA, a different lineage from the MBE codecs above. See
[Voice to bits: vocoders](/learn/digital-trunking/voice-to-bits-vocoders/)

**Analog voice** — Voice sent as a continuous FM signal, which degrades gradually with
distance and noise. Contrast with digital voice, which is crisp until it fails. See
[Analog vs digital voice](/learn/digital-trunking/analog-vs-digital-voice/)

## Modulation for trunked radio

**C4FM (Compatible 4-level FM)** — The four-level FSK used by P25 Phase 1 and the
basis of DMR's modulation. See
[Digital modulation for trunking](/learn/digital-trunking/digital-modulation-for-trunking/)

**CQPSK / LSM (Linear Simulcast Modulation)** — A phase-and-amplitude (linear) way of
sending the same P25 Phase 1 symbols as C4FM, designed to behave better on simulcast
systems. See
[Digital modulation for trunking](/learn/digital-trunking/digital-modulation-for-trunking/)

**π/4-DQPSK** — The differential phase-shift keying used by P25 Phase 2 and TETRA,
sending two bits per symbol by shifting phase. See
[Digital modulation for trunking](/learn/digital-trunking/digital-modulation-for-trunking/)

**4FSK (four-level FSK)** — Frequency-shift keying with four tone levels, carrying two
bits per symbol; underlies C4FM, DMR, and NXDN. See
[Digital modulation for trunking](/learn/digital-trunking/digital-modulation-for-trunking/)
and the [Constellation panel](/constellation.html)

**GMSK (Gaussian Minimum-Shift Keying)** — A smoothed frequency-shift modulation used
by D-STAR (and many other digital systems) that keeps the signal spectrally tidy. See
[Digital modulation for trunking](/learn/digital-trunking/digital-modulation-for-trunking/)

**Symbol rate** — How many symbols per second a signal sends; with bits-per-symbol it
sets the bit rate. P25 and DMR run at 4800 symbols/s. See
[Digital modulation for trunking](/learn/digital-trunking/digital-modulation-for-trunking/)
and the [Symbol scope](/symbol-scope.html)

**FDMA (Frequency-Division Multiple Access)** — Giving each call its own narrow
frequency channel. P25 Phase 1 and NXDN are FDMA. See
[TDMA vs FDMA](/learn/digital-trunking/tdma-vs-fdma/)

**TDMA (Time-Division Multiple Access)** — Splitting one frequency into repeating time
slots so several calls share it. DMR and P25 Phase 2 are TDMA. See
[TDMA vs FDMA](/learn/digital-trunking/tdma-vs-fdma/)

**Time slot** — One of the alternating time windows on a TDMA channel; DMR has two
(slot 1 and slot 2), each able to carry a separate call. See
[TDMA vs FDMA](/learn/digital-trunking/tdma-vs-fdma/)

## Framing, error control & decoding

**Frame / burst** — A fixed-size package of bits sent over the air, containing voice
or data plus sync and signaling. DMR calls its slot-sized unit a burst. See
[Framing, FEC & interleaving](/learn/digital-trunking/framing-fec-interleaving/)

**Forward error correction (FEC)** — Extra redundant bits added before transmission so
the receiver can fix errors without asking for a resend — what lets digital voice
survive noise. See [Framing, FEC & interleaving](/learn/digital-trunking/framing-fec-interleaving/)

**Interleaving** — Scrambling the order of bits before sending so that a short burst of
interference damages many separate, recoverable bits instead of one unrecoverable
chunk. See [Framing, FEC & interleaving](/learn/digital-trunking/framing-fec-interleaving/)

**CRC (Cyclic Redundancy Check)** — A checksum attached to a frame so the receiver can
tell whether the bits arrived intact. See
[Framing, FEC & interleaving](/learn/digital-trunking/framing-fec-interleaving/)

**BER (Bit Error Rate)** — The fraction of received bits that are wrong; a key health
metric for a decode, climbing as the signal degrades. See
[Troubleshooting a decode](/learn/digital-trunking/troubleshooting-a-decode/)

**Digital cliff** — The point where a digital signal stops gracefully degrading and
suddenly fails completely: clear audio one moment, silence the next. See
[Analog vs digital voice](/learn/digital-trunking/analog-vs-digital-voice/)

## Security & control

**Encryption** — Scrambling voice or data so only authorized radios can recover it;
encrypted talkgroups can be detected but not decoded. See
[Encryption & authentication](/learn/digital-trunking/encryption-and-authentication/)
and the [DMR encryption panel](/dmr-encryption.html)

**AES-256** — A strong, standardized encryption cipher (256-bit key) used by P25 and
others for secure traffic; not breakable by monitoring. See
[Encryption & authentication](/learn/digital-trunking/encryption-and-authentication/)

**DES-OFB** — An older 56-bit DES cipher in output-feedback mode, once common on P25;
weak by modern standards but still seen in the field. See
[Encryption & authentication](/learn/digital-trunking/encryption-and-authentication/)

**ADP (Advanced Digital Privacy)** — Motorola's RC4-based 40-bit encryption option,
lighter than AES and intended as basic privacy rather than strong security. See
[Encryption & authentication](/learn/digital-trunking/encryption-and-authentication/)

**Authentication** — A system verifying that a radio is a legitimate, authorized unit
before granting it service, separate from encrypting the traffic itself. See
[Encryption & authentication](/learn/digital-trunking/encryption-and-authentication/)

**Stun / Kill** — Over-the-air commands a system can send to disable a radio
temporarily (stun) or permanently (kill), used when a unit is lost or stolen. See
[Encryption & authentication](/learn/digital-trunking/encryption-and-authentication/)

## P25 (Project 25)

**P25 (Project 25)** — A North American suite of standards for public-safety digital
radio, defining both conventional and trunked operation. See
[P25 Phase 1](/learn/digital-trunking/p25-phase-1/)

**P25 Phase 1** — The first generation of P25: FDMA, 12.5 kHz channels, C4FM
modulation, and the IMBE vocoder. See
[P25 Phase 1](/learn/digital-trunking/p25-phase-1/)

**P25 Phase 2** — The newer generation: two-slot TDMA on a 12.5 kHz channel
(effectively 6.25 kHz per call) using π/4-DQPSK and the AMBE+2 vocoder. See
[P25 Phase 2](/learn/digital-trunking/p25-phase-2/)

**TSBK (Trunking Signaling Block)** — The control-channel message format that carries
P25 trunking signaling — grants, affiliations, and system data. See
[Control-channel signaling](/learn/digital-trunking/control-channel-signaling/)

**LDU (Logical Data Unit)** — A P25 Phase 1 voice frame structure (LDU1/LDU2) that
packages IMBE voice along with embedded signaling and link control. See
[P25 Phase 1](/learn/digital-trunking/p25-phase-1/)

## DMR

**DMR (Digital Mobile Radio)** — An ETSI standard for digital land-mobile radio using
two-slot TDMA on 12.5 kHz channels, popular for business and amateur use. See
[DMR Tier II & III](/learn/digital-trunking/dmr-tier-2-3/)

**Tier II** — Conventional (non-trunked) DMR: licensed two-slot TDMA repeaters with no
central trunking controller. See
[DMR Tier II & III](/learn/digital-trunking/dmr-tier-2-3/)

**Tier III** — Trunked DMR, adding a control-channel-driven trunking layer on top of
DMR's TDMA. See [DMR Tier II & III](/learn/digital-trunking/dmr-tier-2-3/)

**CSBK (Control Signaling Block)** — DMR's signaling message unit, used for trunking
control and other system messages — DMR's counterpart to P25's TSBK. See
[Control-channel signaling](/learn/digital-trunking/control-channel-signaling/)

## TETRA

**TETRA (Terrestrial Trunked Radio)** — A European ETSI standard for trunked digital
radio, using four-slot TDMA and π/4-DQPSK, common for public safety and transit
outside North America. See [TETRA](/learn/digital-trunking/tetra/)

**TMO (Trunked Mode Operation)** — TETRA radios operating through the infrastructure
(sites and controllers), the normal trunked mode. See
[TETRA](/learn/digital-trunking/tetra/)

**DMO (Direct Mode Operation)** — TETRA (and DMR) radios talking directly to each
other without infrastructure, like a walkie-talkie. See
[TETRA](/learn/digital-trunking/tetra/)

## Other digital standards

**NXDN** — A narrowband FDMA digital standard (6.25 kHz) developed by Icom and Kenwood,
available in conventional and trunked forms. See [NXDN](/learn/digital-trunking/nxdn/)

**dPMR (digital Private Mobile Radio)** — An ETSI 6.25 kHz FDMA standard closely
related to NXDN, aimed at light commercial use. See
[dPMR, D-STAR & YSF](/learn/digital-trunking/dpmr-dstar-ysf/)

**D-STAR** — An amateur-radio digital voice and data standard (developed with Icom)
using GMSK and the AMBE vocoder. See
[dPMR, D-STAR & YSF](/learn/digital-trunking/dpmr-dstar-ysf/)

**System Fusion / YSF (Yaesu System Fusion)** — Yaesu's amateur C4FM digital voice
system, also known as YSF. See
[dPMR, D-STAR & YSF](/learn/digital-trunking/dpmr-dstar-ysf/)

## Legacy & proprietary systems

**SmartNet / SmartZone** — Motorola's earlier trunking systems: SmartNet for a single
site, SmartZone adding multi-site networking, both using an analog or digital voice
payload. See [Motorola SmartNet](/learn/digital-trunking/motorola-smartnet/)

**Type I / Type II** — Two Motorola trunking fleet-management schemes. Type I uses
fleet/subfleet addressing; Type II uses flat talkgroup IDs and is far more common. See
[Motorola SmartNet](/learn/digital-trunking/motorola-smartnet/)

**EDACS (Enhanced Digital Access Communications System)** — A GE/Ericsson/Harris
trunking system known for fast channel access and a distinctive control channel. See
[EDACS, LTR & MPT-1327](/learn/digital-trunking/edacs-ltr-mpt1327/)

**ProVoice** — The digital voice option layered onto EDACS systems, using the AMBE
vocoder. See [EDACS, LTR & MPT-1327](/learn/digital-trunking/edacs-ltr-mpt1327/)

**LTR (Logic Trunked Radio)** — An EF Johnson trunking scheme with *distributed*
control — signaling is embedded on each channel rather than a dedicated control
channel. See [EDACS, LTR & MPT-1327](/learn/digital-trunking/edacs-ltr-mpt1327/)

**Passport** — A proprietary trunking extension of LTR developed by Trident/SmarTrunk,
adding features on top of the LTR scheme. See
[EDACS, LTR & MPT-1327](/learn/digital-trunking/edacs-ltr-mpt1327/)

**MPT-1327** — A British signaling standard for analog trunked radio, widely used
internationally before digital systems took over. See
[EDACS, LTR & MPT-1327](/learn/digital-trunking/edacs-ltr-mpt1327/)

## Standards bodies

**APCO (Association of Public-Safety Communications Officials)** — The North American
body that drove the creation of Project 25 (the "APCO-25" name). See
[Standards & bodies](/learn/digital-trunking/standards-and-bodies/)

**ETSI (European Telecommunications Standards Institute)** — The standards body behind
DMR, TETRA, and dPMR. See [Standards & bodies](/learn/digital-trunking/standards-and-bodies/)

**TIA (Telecommunications Industry Association)** — The U.S. standards organization
that publishes the P25 (TIA-102) document suite. See
[Standards & bodies](/learn/digital-trunking/standards-and-bodies/)

**ITU (International Telecommunication Union)** — The UN agency that allocates spectrum
and sets worldwide radio regulations within which all these systems operate. See
[Standards & bodies](/learn/digital-trunking/standards-and-bodies/)

**DMR Association** — The industry group that promotes DMR interoperability and
maintains shared identifier resources for the standard. See
[Standards & bodies](/learn/digital-trunking/standards-and-bodies/)

---

Didn't find a term? It's probably explained in one of the
[Digital & Trunked Radio lessons](/learn/digital-trunking/) — or open an
[issue on GitHub](https://github.com/MattCheramie/GopherTrunk/issues) and we'll add it.
