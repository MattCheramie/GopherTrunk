# GopherTrunk Terms of Service

**Version 1 (4 September 2026)**

GopherTrunk is free, open-source software for receiving and decoding
trunked and conventional radio systems with software-defined radio
hardware. Like any radio receiver, what you may lawfully do with it
depends on where you are and how you use it.

You must acknowledge these terms when you install GopherTrunk, or on
its first run. By installing, running, or using GopherTrunk you accept
them. If you do not accept them, do not install or use the software.

## 1. License

GopherTrunk's source code is licensed under the Apache License 2.0
(see `LICENSE`); bundled third-party components are listed in
`THIRD_PARTY_LICENSES.md`. These terms govern your *use* of the
software. They supplement, and do not replace or restrict, the rights
the Apache License grants you to copy, modify, and redistribute the
code.

## 2. You are responsible for using it lawfully

Laws on radio monitoring differ by country, state, and province.
Depending on your jurisdiction, some or all of the following may be
restricted or illegal: listening to particular services or
frequencies, recording transmissions, using a scanner in a vehicle or
in the commission of a crime, and disclosing or acting on what you
hear. Examples include 47 U.S.C. 605 and the Electronic Communications
Privacy Act in the United States; comparable laws exist in most other
countries.

**You are solely responsible for knowing and complying with every law
that applies to you.** The GopherTrunk contributors do not and cannot
advise you on legality, and nothing in the software or its
documentation is legal advice. If monitoring a system would be
unlawful where you are, do not use GopherTrunk to monitor it.

## 3. Encrypted and protected communications

GopherTrunk does not decrypt communications, and you must not use it
to attempt to defeat, circumvent, or unlawfully intercept encryption
or any other protection. The optional `cryptolab` research toolkit is
provided for education and for the analysis of systems you own or are
explicitly authorized to test, and for nothing else.

## 4. What you do with received content

Where the law protects the content of communications, do not divulge,
publish, sell, or otherwise use it for your benefit or anyone else's.
Never use GopherTrunk to facilitate a crime, to interfere with
public-safety operations, or to stalk, harass, or track any person.

## 5. Recordings and privacy

Voice recordings, call logs, radio IDs, talkgroup activity, and
similar metadata that GopherTrunk stores may constitute personal data
about identifiable people. If you record, retain, publish, or share
them, privacy and data-protection law (such as the GDPR) may apply to
you as the responsible party. Handle them accordingly.

## 6. Not for safety-of-life use

GopherTrunk is a hobbyist and research tool. It is not a certified
receiver; it can and does mis-decode, drop, delay, or entirely miss
traffic. Never rely on it for emergency response, life safety,
navigation, or any other purpose where a missed or incorrect message
could cause harm.

## 7. Receive only

GopherTrunk is designed as a receiver. Its signal-generation features
(such as `gen`, `replay`, and Signal Lab) produce IQ data for offline
testing; feeding that data to transmit-capable hardware is a radio
transmission, which in nearly every jurisdiction requires an
appropriate license and proper shielding or a dummy load. You are
solely responsible for any RF you emit.

## 8. Patents (voice codecs)

GopherTrunk includes clean-room implementations of digital voice
decoders (IMBE, AMBE+2, TETRA ACELP). Some of the techniques they use
are patent-encumbered in some jurisdictions. It is your responsibility
to determine whether your use requires a patent license where you are.
See `docs/vocoders.md`.

## 9. Export control

You must comply with all export, re-export, and sanctions laws that
apply to obtaining and using this software.

## 10. No warranty; limitation of liability

The software is provided **"AS IS", without warranty of any kind**,
and the contributors are not liable for damages arising from its use,
as set out in Sections 7 and 8 of the Apache License 2.0. This
includes any consequence of unlawful use, of missed or mis-decoded
traffic, or of reliance on decoded content.

## 11. Changes

These terms carry a version number. When they change, you will be
asked to acknowledge the new version on the next install, or on the
first run after upgrading.

---

Acknowledgment is recorded locally only; nothing is sent anywhere.
The Windows installer records it after its Terms of Service page, and
on every platform the CLI records it under your user configuration
directory the first time you accept: interactively on a terminal, via
`gophertrunk terms accept`, or with `GOPHERTRUNK_ACCEPT_TERMS=1` for
unattended or containerized installs.
