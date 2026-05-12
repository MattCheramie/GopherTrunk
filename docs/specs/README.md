# Reference specifications

Air-interface specs and reference documents that the on-air FEC /
channel-coding implementations in this repo derive from. Files
land here so a future contributor can cross-check the code against
the original source without re-hunting the PDFs from upstream
archives. None of these documents are redistributed under a licence
that conflicts with the rest of the repo's MIT licence — they are
either openly published (ETSI) or widely-mirrored open references
(NXDN Forum, M/A-COM LBI).

| File | What it covers | Used by |
| --- | --- | --- |
| [`nxdn-ts-1-a-v1.3.pdf`](nxdn-ts-1-a-v1.3.pdf) | **NXDN Forum Technical Specification Part 1-A, Common Air Interface, rev 1.3** — §4.5 channel coding (CAC outbound + Long CAC + Short CAC inbound + SACCH); §4.6 frame structure (RCCH outbound = FSW 20 + LICH 16 + CAC 300 + E 24 + Post 24); §6 RCCH message types. | `internal/radio/nxdn/cac_channel.go` (155-bit info ‖ CRC-16 ‖ tail → K=5 → puncture(50/350) → 25×12 interleave → 300 bits); `internal/radio/nxdn/sacch.go` (K=5 + 60-position interleaver + 12-bit puncture + CRC-6); `internal/radio/nxdn/process.go` (`ViterbiSpec` mode wires the §4.5.1.1 chain end-to-end). |
| [`etsi-en-300-392-2-v3.8.1.pdf`](etsi-en-300-392-2-v3.8.1.pdf) | **ETSI EN 300 392-2 v3.8.1 — TETRA V+D Air Interface** — §8.2 RCPC + Reed-Muller + scrambler + (K,a) interleaver primitives; §8.3.1 per-channel coding (BSCH / SCH/HD / SCH/HU / SCH/F / AACH); §9 burst layout. | `internal/radio/framing/{rcpc_tetra,rcpc_tetra_sig,rm_30_14_tetra,scramble_tetra,interleave_tetra}.go`; `internal/radio/tetra/channel_coding.go` (per-channel encode/decode helpers); `internal/radio/tetra/process.go` (`SetChannelCoding(ChannelCodingOn)` wires the §8.3.1 chain). |
| [`lbi-38463c-edacs-system-manager.pdf`](lbi-38463c-edacs-system-manager.pdf) | **M/A-COM LBI-38463C — EDACS System Manager Supervisor's Guide, v3.0 (Group 11)** — workstation UI for site reconfiguration, agency partitions, user accounts, dynamic regroup, alarm controls. Kept here as a **negative reference**: it does *not* document the air interface, CCW bit layout, BCH parameters, or any channel coding. Future contributors looking for an EDACS air-interface spec should skip this and look for LBI-39031 / LBI-39154 / LBI-38894 instead. | None directly — see `internal/radio/edacs/edacs.go` package doc for the canonical open reference (`lwvmobile/edacs-fm`'s `bch3.h`) the BCH(40, 28, 2) implementation in `internal/radio/framing/bch_edacs.go` mirrors. |

## Provenance + licensing

- **NXDN-TS-1-A** is published openly by the NXDN Forum (a JVCKENWOOD
  + Icom industry consortium). Document marked
  "Copyright © 2007–2012 JVC KENWOOD Corporation and Icom Incorporated".
  Distribution for reference / interoperability is the document's
  stated purpose.
- **ETSI EN 300 392-2** is published openly by the European
  Telecommunications Standards Institute under their standard
  IPR policy. ETSI standards are free to download from
  `etsi.org` and are explicitly licensed for reference + research
  use.
- **LBI-38463C** is a GE / Ericsson Mobile Communications field
  service manual from 1990. The "LBI" series was distributed
  with EDACS infrastructure as user-operator documentation;
  modern survivals appear on radio-archaeology sites
  (`repeater-builder.com`, archive.org). This copy is
  preserved for historical reference; the document is
  ~35 years out of any commercial distribution.

## Air-interface specs *not* in this directory

These would be valuable additions if anyone tracks them down,
but were not located at the time of this commit:

- **LBI-39031** / **LBI-39154** — EDACS Communications Protocol
  Specification. The closest GE/Ericsson document to a formal
  air-interface spec. Not publicly distributed.
- **LBI-38894** — EDACS Trunked Radio System Description /
  Theory of Operation. Frame layout + CCW timing.
- **LBI-38540** — EDACS RF Subsystem Description (GMSK
  modulation parameters).
- **TIA-102.AABF** / **TIA-102.BAAA** — Project 25 Phase 2
  + Phase 1 air interface. Currently the `internal/radio/p25/`
  implementations cite TIA-102 sections in code comments but
  no PDF is in-tree.
- **ETSI TS 102 361-2** — DMR Tier III air interface.

Pulls welcome.
