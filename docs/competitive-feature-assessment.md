---
layout: page
title: Competitive feature assessment
description: Where GopherTrunk stands against SDRTrunk, Trunk Recorder, OP25, DSD-FME / dsd-neo, DSDPlus, Unitrunker, hardware scanners and the call-sharing ecosystem — catalog, gap matrix, what landed, what is still open
nav_group: Reference
---

# Competitive feature assessment (October 2026)

An exhaustive look at the other digital trunking scanner software and
hardware, what each one does in five areas — **scanning**, **trunking /
control-channel decode**, **audio quality**, **recording & playback** and
**encryption / decryption** — and where GopherTrunk had gaps. The gaps
that were tractable without a new on-air capture were closed in the same
series (section 3); the rest are listed with their blocker (section 4).

Sources were the projects' own READMEs, wikis, manuals and release notes
(October 2026); vendor and forum claims are marked. Nothing GPL was
ported — reference code was read for protocol facts only, per the repo's
standing rule.

## 1. The landscape

### Open-source SDR trunking software

| | SDRTrunk 0.6.x (Java, GUI) | Trunk Recorder 5.2 (C++/GNU Radio, headless) | OP25 boatbod (Python/GR, CLI + HTTP) | DSD-FME / dsd-neo (C, CLI/ncurses) | DSDPlus 2.547 (Windows, closed) | Unitrunker 2.1 (Windows, closed) |
|---|---|---|---|---|---|---|
| **Trunking protocols** | P25 P1/P2 (TDMA CC in nightly), DMR Cap+/Con+/CapMax/T3, NXDN, LTR/LTR-Net, MPT-1327, Passport | P25 P1/P2, SmartNet/SmartZone, DMR trunked (Cap+/CapMax/Con+/T3 landing) | P25 P1/P2 + TDMA CC, SmartNet/SmartZone (incl. OBT), Con+ (exp.), DMR T2 | P25 P1/P2, DMR T3 + Cap+/Con+ (channel maps), NXDN48/96 + IDAS, EDACS/ProVoice; dsd-neo adds a real single-tuner **trunk-scan** rotation | P25 P1/P2 + TDMA CC, DMR T3 std + Motorola/Selex non-std, Cap+/Con+, NXDN/IDAS incl. DFA, ProVoice decode, dPMR, D-STAR, YSF | EDACS, Motorola Type I/II/IIi, P25 P1, MPT-1327; no DMR/NXDN |
| **TETRA** | — | — | — | — | — | — |
| **CC data surfaced** | grants, affiliations/registrations, status, GPS (P25 Motorola / DMR LRRP / Fleetsync), talker alias, patches, ESS, neighbours | grants, patches, unit on/join/off/ack/data/answer/location → scripts + MQTT, call-alert pages, OTA aliases | grants, adjacent sites, source RID, ESS at -v 10, Wireshark export | LRRP/GPS, event log, PDU logging, OK-DMRlib output | grants, neighbours, registrations, talker alias, GPS/AVL, raw P25 data window | adjacent sites, registrations, affiliations, patches |
| **Scanning UX** | alias priority 1–99 with preemption, Listen on/off, dup-call detection across sites, auto-start channels; no dwell/hold | priority = recorders required; multi-site preferred-NAC dedupe; conventional CSV with CTCSS + Signal Detector | TGID white/black lists, priority with mid-call preempt, **hold / goto / lockout / skip** keys, NBFM voice-squelch | TG hold/avoid/lockout (session or persistent), encrypted lockout, voice-gated scan, per-target dwell / tone filters | groups priority / lockout, neighbour auto-roaming, FMP24 ScanList conventional | per-VCO roles, hide groups/users |
| **Audio** | JMBE; mono/stereo per call; start/preempt/drop tones; MP3 normalize; NBFM HPF | OP25-derived vocoder; ffmpeg filters + two-pass loudnorm | software IMBE/AMBE, P2 tones, UDP/websocket out | mbelib; per-slot synthesis; HPF/LPF; dsd-neo adaptive EQ / FLL / TED / live SNR | internal IMBE/AMBE/AMBE+2 | P1 IMBE + ProVoice only |
| **Recording / upload** | MP3/WAV per alias; IQ / bitstream / MBE capture; Broadcastify Calls + Feeds, Rdio, OpenMHz, Icecast | WAV + M4A + JSON sidecar; OpenMHz, Broadcastify, Rdio plugin, simplestream, **MQTT plugin**, Prometheus, scripts; status WebSocket | no per-call recorder; Liquidsoap → Icecast with metadata | per-call WAV (`-P`), MBE files, symbol capture; dsd-neo **rdio exporter** + IQ capture/replay | per-call MP3/WAV, event logs | none |
| **Decryption** | detect only | detect + monitorEncrypted metadata; decryption PR unmerged | **ADP / DES-OFB / AES-256 with keys** (crypt_behavior allow/silence/skip) | **the widest**: ADP/RC4, DES, TDEA, AES-128/256, DMR Basic Privacy (+ 256 TG→key overrides), vendor BP/EP (TYT, Retevis, Baofeng, Anytone, Kenwood, Connect Systems), NXDN / dPMR scramblers, M17; muted by default | AlgID/KID logging only | detect only |

### Hardware scanners

**Uniden SDS100/200, BCD436/536HP**: Favorites Lists + 100 Quick Keys,
location-based scanning (zip / GPS / range), Service Types, Priority
Scan / Priority ID / DND, **Hold**, **temporary (power-cycle) and permanent
Avoid**, Close Call RF capture, Discovery mode (auto-record new TG/freq),
Fire Tone-Out, Number Tags, custom per-channel **alerts (9 tones, LED
colours)**, Instant Replay 30–240 s, record to SD, **encryption muting**;
DMR/NXDN/ProVoice as paid upgrades; no decryption. **Whistler TRX-1/2**:
object-oriented scan lists (200), Spectrum Sweeper, Skywarn, V-Scanner
configs, auto CTCSS/DCS detect, alert LED, 50 h recording, IF output.

### Call-sharing ecosystem

**rdio-scanner**: LIVE FEED queue, HOLD SYS / HOLD TG, SKIP, **timed AVOID
(30/60/120 min)**, REPLAY LAST, archive search by date/system/TG/group/tag
with download, ON/OFF/PARTIAL selection; `POST /api/call-upload`.
**OpenMHz**: live + calendar archive, group filters, star, URL-shareable
filters, **playback speed**, delay. **Broadcastify Calls**: per-call
archive with node voting/dedupe, playlists, client API. **Scanner Radio**:
live **transcriptions**, clips, alerts. **Trunking Recorder / ProScan**:
uploads for hardware scanners, web server, no-loss recorder.

### General SDR applications

SDRangel (DSD demod DMR/dPMR/D-Star/YSF/NXDN, **stereo per slot**,
Frequency Scanner with VAD / linger), SDR++ (FFT range scanner with linger,
Frequency Manager, rigctl server; TETRA demodulator plugin), Gqrx
(bookmarks, rigctl), SDR# (Frequency Manager + Scanner with metrics DB,
**TETRA plugin + Trunk Tracker** — the only other TETRA trunk follower),
CubicSDR (bookmarks). None decrypt; none record per call with metadata.

## 2. GopherTrunk before this series — strengths and gaps

**Strengths no competitor matches**: TETRA TMO + DMO end-to-end
(bit-exact ACELP, neighbour cells, SDS-less but complete call following),
fifteen trunking protocols in one daemon including dPMR / EDACS / LTR /
MPT1327 / D-STAR / YSF, a first-party web console with call history and
playback, wideband multi-site decode, MRC diversity combining, offline
replay / siglab / cryptolab instruments, RadioReference import, hunt /
survey discovery, FLAC everywhere, per-call JSON + `.raw` / `.imb` / `.amb`
sidecars, Broadcastify / Rdio / OpenMHz / Icecast / webhooks, rigctld.

**Gaps found** (✔ = closed in this series, ○ = open, see section 4):

| Area | Gap | Who has it | |
|---|---|---|---|
| Encryption | P25 DES-OFB / TDES / AES not decrypted (keys rejected at load) | OP25, DSD-FME | ✔ |
| Encryption | undecryptable encrypted audio played / recorded as vocoder noise | every scanner, SDRTrunk, TR, DSD-FME mute | ✔ |
| Encryption | DMR Basic Privacy, DMR DES/AES, Hytera EP, NXDN/dPMR scramblers, vendor BP | DSD-FME | ○ |
| Encryption | P25 Phase 2 (TDMA) decryption | OP25 (experimental) | ○ |
| Scanning | no alert rules / notifications (tone.alert never left the bus) | Uniden/Whistler alerts, SDRTrunk alias actions, TR MQTT | ✔ |
| Scanning | no talkgroup **Hold**, no **timed Avoid** | Uniden, OP25, dsd-neo, rdio-scanner | ✔ |
| Scanning | no priority-channel sampling on the conventional list | Uniden | ✔ (`scanner.priority_interleave`) |
| Scanning | favourites / quick keys, Close Call, Discovery mode, location-based scanning | Uniden | ○ |
| Scanning | conventional lockouts not persisted | Uniden, Whistler | ✔ (`conv_lockouts` table) |
| Scanning | no site roaming / neighbour following | DSDPlus roaming | ○ |
| Trunking | P25 status / message / call-alert / deny / queue / ack / extended-function / radio-monitor TSBKs unparsed | SDRTrunk, TR (MQTT) | ✔ |
| Trunking | no SMS / SDS / LRRP / P25 GPS / data-channel decode; TETRA SDS | SDRTrunk, DSD-FME, DSDPlus, SDR# TETRA plugin | ○ |
| Trunking | DMR Connect Plus / Hytera XPT recognised, not decoded; NXDN48 trunked control channel (scan-list NXDN48 voice decodes); ProVoice / YSF audio | DSDPlus | ○ |
| Audio | no transcription | Scanner Radio, TR plugins | ✔ |
| Audio | no stereo / per-slot output, no start/drop tones, no noise reduction | SDRangel, SDRTrunk, SDR++ | ○ |
| Recording | History had no free-text search, date range, flag filter or CSV export | rdio-scanner, OpenMHz | ✔ |
| Recording | no playback speed | OpenMHz | ✔ |
| Recording | retention left `.json` / `.imb` / `.amb` / `.mp3` sidecars; JSON sidecar lacked ALGID / KID | — | ✔ |
| Recording | patches table not exposed over REST | SDRTrunk / Unitrunker show patches | ✔ |
| Recording | no MQTT; no exec hooks (`uploadScript` / `unitScript`) | TR | ✔ (via alerts channels) |
| Recording | no instant-replay / live delay queue; no OGG/Opus/M4A | Uniden, OpenMHz, TR | ○ |

## 3. What landed in this series

Each item shipped with failing-first tests and, where it decodes air, a
reference pin rather than a capture pin (the #764/#771 rule — a synthetic
pass is not an on-air pass; the capture A/B recipe is named).

1. **P25 DES-OFB, Triple-DES and AES-128/256 decryption with operator keys**
   — `algorithm: des | tdes | aes` in `encryption_keys`; OP25's on-air
   frame layout (one cipher block discarded, then ADP's mapping) and the
   TIA LFSR MI→IV expansion, pinned by literal offsets and a transcription
   of OP25's expansion. [DMR & P25 encryption](dmr-encryption.html).
2. **`recordings.mute_encrypted`** — silence instead of vocoder noise for a
   call no key decrypts; sidecars keep ciphertext; `muted_frames` logged.
3. **Alerts** — `alerts:` rules (events × system / talkgroup / radio /
   emergency / encrypted / tone profile / duration / keywords, cooldown,
   templates) → Discord, Slack, ntfy, Pushover, Telegram, Gotify, webhook,
   exec, **MQTT** (QoS 0, optional full event mirror). Audio attachments on
   Discord / Telegram / webhook. `GET /api/v1/alerts`, channel test.
   [Alerts](alerts.html).
4. **Talkgroup hold + timed avoid** — engine, `POST/DELETE
   /api/v1/scanner/hold`, `POST/DELETE /api/v1/talkgroups/{id}/avoid`,
   Scanner + Talkgroups panels.
5. **P25 unit signalling events** — `unit.status`, `unit.message`,
   `call.alert`, `unit.ack`, `unit.queued`, `unit.deny`, `unit.function`
   (radio check / inhibit …), `unit.monitor`, with reason / function tables;
   CC Activity rows; alertable. [Events](api-events.html).
6. **Transcription** — `transcription:` posts recordings to any
   OpenAI-compatible Whisper server (OpenAI, whisper.cpp, faster-whisper,
   LocalAI); transcript on the call, searchable, exported, `call.transcript`
   event, keyword alerts. [Transcription](transcription.html).
7. **History search / date range / flag filters / CSV export**, **playback
   speed**, **sidecar retention**, **ALGID / KID in the JSON sidecar**,
   **`GET /api/v1/patches`**.

## 4. Still open, with the blocker

Ordered by value to a scanner operator.

| Gap | Blocker / why not now |
|---|---|
| **DMR Basic Privacy (16-bit) and vendor BP/EP (TYT, Retevis, Baofeng, Anytone, Kenwood, Connect Systems), DMR DES/AES** | Reference implementations are GPL (DSD-FME); the key schedules are not spec-published. Needs an independent description of each keystream construction plus a known-key capture, or the result is the self-consistent-synthetic trap. DMR Enhanced Privacy (RC4) is already in and capture-verified. |
| **P25 Phase 2 decryption** | OP25's TDMA path is marked experimental (7-byte frames, `PCW[6] &= 0x80`); the composer's Phase 2 chain also still fails the real-air MAC descramble (#915), so there is no verified place to hang it. |
| **TETRA SDS (text, LIP location) and D-STATUS** | Layouts exist in tetra-kit / osmo-tetra; worth doing, but GopherTrunk's SDS path needs a capture with known message content to pin — none is on hand. |
| **DMR LRRP / P25 Motorola & Harris unit GPS / SMS / data-channel decode** | The P25 PDU / SNDCP / IPv4 parsers exist but nothing feeds them; LRRP needs a data-call follow (T3 data grants are observed, not followed). Capture-gated. |
| **Site roaming / neighbour following** | Neighbours are decoded and displayed; following them needs an RSSI/decode-quality policy and a multi-site test rig. |
| **Close Call / Discovery mode, favourites & quick keys, location-based scanning** | Scanner-UX features with no decode risk; Discovery mode (auto-record unknown TGs) is the most requested — a design item for the next series. Priority-channel sampling landed as `scanner.priority_interleave`. |
| **Stereo / per-slot audio, start / drop tones, noise reduction** | Audio-path work; per-slot panning conflicts with the mono 8 kHz live stream contract and needs a web-player change too. |
| **Instant replay / live delay queue; Opus / M4A recordings** | M4A needs an AAC encoder (none pure-Go); Opus is a dependency decision. |
| **Talkgroup hold / avoid surviving a restart** | Deliberately session-only (a scanner power cycle clears them); conventional lockouts now persist by frequency in the `conv_lockouts` table. |
| **DMR Connect Plus / Hytera XPT, NXDN48 control channel, ProVoice & YSF audio** | All capture- or licence-gated (`docs/status.md`). |

## 5. Method note

Three research passes (open-source trunking suites; DSD lineage +
hardware scanners + PC control + call-sharing + general SDR apps; the long
tail incl. TETRA tooling) were cross-checked against two codebase
inventories (scanning + trunking; audio + recording + encryption) that
cite the implementing file for every feature, so each gap above is a
confirmed absence, not a doc oversight. The inventories also surfaced
stale package docs (`internal/radio/ltr`, `mpt1327`, `tetra/tetra.go`)
that describe gaps now shipped; those are a docs follow-up.
