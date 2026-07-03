---
title: "Crypto Lab, Part 1: Breaking It Is the Test"
description: Crypto Lab is GopherTrunk's byte-oriented cryptographic-research toolkit for security-testing RF encryption — shipped inside the binary but excluded from the default install behind a build tag, it attempts decryption by every applicable method and grades how far each one got.
category: tutorials
keywords: crypto lab, gophertrunk, rf encryption testing, cryptographic research toolkit, p25 encryption, keystream reuse, iv reuse, security testing, build tag, cryptanalysis, resistant partial broken
tags: [cryptolab, security-testing, encryption, getting-started, p25, rf]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 1
---

*Part 1 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit — the third leg of the **Lab Bench trilogy** ([Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}) finds and names signals, [RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}) dissects them like Wireshark, and Crypto Lab tries to break their encryption). It is also where a recurring mystery — a signal we call **Mercury** — finally gets an answer.*

> **TL;DR:** Crypto Lab is a byte-oriented cryptographic-research toolkit that lives *inside* the `gophertrunk` binary but is **excluded from the default build** behind a `cryptolab` build tag. Its `assess` harness attempts to decrypt captured frames by every applicable method and grades each one — because attempting decryption *is* the test. It cannot brute a strong cipher with a good key and rotated IVs (that grades <span class="lab-verdict lab-verdict--ok">RESISTANT</span>); what it breaks is what fails in the field: reused IVs, default/test keys, keyless obfuscation, and weak keystreams.

**Key takeaways**

- **Attempting decryption is the security test.** A complete decryption means the deployment failed; recovering nothing means the encryption held.
- **Opt-in by design.** The toolkit is behind a build tag, so a default operator install never ships an attack toolkit it doesn't need.
- **Three verdicts.** Every assessment lands on <span class="lab-verdict lab-verdict--ok">RESISTANT</span>, <span class="lab-verdict lab-verdict--warn">PARTIAL</span>, or <span class="lab-verdict lab-verdict--bad">BROKEN</span>.
- **Honest about limits.** Crypto Lab does not pretend to break AES. It finds the misconfigurations and weak constructions that break real deployments.

> **Responsible use — read this first.** Crypto Lab is for **authorized security testing only**: assessing systems you own or operate, systems you are explicitly licensed to test, and security research, CTF, and classroom use on your own material. Radio encryption exists to protect real people — decrypting traffic you are not authorized to touch is illegal in most jurisdictions, and nothing in this series is a licence to do it. Every workflow here assumes you are testing *your own* deployment or one you have written permission to assess. If that is not you, stop here.

## In this post

- **What Crypto Lab is** — a research toolkit, not a magic decryptor, and why "breaking it is the test."
- **Why it is a build-tag opt-in** — how the toolkit ships inside the binary yet stays out of the default install.
- **The CLI shape** — `gophertrunk cryptolab <tool> [<mode>] [flags]`, plus `list` and `serve`.
- **The verdict scale** — RESISTANT / PARTIAL / BROKEN, and the escalation ladder of methods behind it.
- **A map of the series** so you can jump to the tool you care about.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `make build TAGS=cryptolab` | Build `gophertrunk` with the toolkit linked in |
| `go build -tags cryptolab ./cmd/gophertrunk` | Equivalent raw `go build` |
| `make test-cryptolab` | Run the toolkit's tests (including the tagged CLI) |
| `gophertrunk cryptolab list` | List every tool and its modes |
| `gophertrunk cryptolab serve -open` | Launch the web console at `127.0.0.1:8096` |
| `gophertrunk cryptolab <tool> [<mode>] [flags]` | Run one tool; global flags precede the tool name |
| `gophertrunk cryptolab assess crypto -in frames.jsonl` | The headline: attempt every attack, grade each |

## What Crypto Lab actually is

Crypto Lab is a byte-oriented cryptographic-research toolkit that collects the analyses you reach for when you are staring at an unfamiliar RF payload and asking two questions: *what am I looking at, and can I break it?* Statistical triage, autocorrelation period detection, a NIST SP 800-22 randomness battery, an obfuscation-class classifier, keyspace brute force, LFSR and keystream analysis, keystream-reuse / many-time-pad recovery, CRC parameter recovery, analog voice descrambling, and a pluggable "subject" framework for studying specific byte obfuscators — all in one subcommand.

The philosophy behind it is worth stating plainly, because it reframes what the tool is *for*. **Attempting decryption is the test.** When you point Crypto Lab at a captured encrypted system, a complete decryption means the deployment failed — its encryption did not do its job. Recovering nothing means the encryption held. You are not "hacking" the system; you are measuring it, the same way a penetration tester measures a network by trying to get in and reporting exactly how far they got.

This is why the toolkit is deliberately honest about its ceiling. It **cannot brute-force a strong key out of a strong cipher.** Point it at AES-256 with a non-default key and rotated IVs and it will tell you, plainly, that the keyspace is infeasible and the encryption is <span class="lab-verdict lab-verdict--ok">RESISTANT</span>. That result is not a failure of the tool — it *is* the security finding: the encryption is doing its job. What Crypto Lab breaks is what actually fails in the field: **reused IVs, default and test keys, keyless "obfuscation" dressed up as encryption, and structurally weak keystreams** (short LFSRs, backdoored small keyspaces). Those are the findings that matter, because those are the deployments that leak.

<aside class="lab-cast">
  <span class="lab-cast__badge">A</span>
  <div class="lab-cast__body">
    <p class="lab-cast__who">Meet the cast.</p>
    <p><strong>Ada</strong> just unboxed her first SDR and is working her first real captures — she's our getting-started point of view, learning each tool as she meets it. <strong>Reese</strong> has been chasing signals for twenty years and supplies the "why": the field war stories, the reason a default IV is a five-alarm fire, and the discipline of grading honestly. They'll turn up lightly throughout the series.</p>
  </div>
</aside>

## The Mercury thread

Running through all three Lab Bench series is a single mystery. **Mercury** is an intermittent, apparently-encrypted burst Ada first captured near **453 MHz** in the UHF business band — short transmissions on a ~12.5 kHz channel that come and go with no obvious schedule. In [Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}) Part 8, blind signal-ID gave a best guess (~4800 sym/s, 4-level FSK-like) but no protocol lock. In [RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}) Part 7, the entropy analyzer triaged Mercury's payload as *not obviously strong* and exported the raw frames. Those frames land here, in Crypto Lab. We'll pick Mercury back up in Part 5, and in Part 10 we finally reveal what it was — a reveal that turns out to be the whole moral of the series. For now, just know the thread is there.

## Why it's a build-tag opt-in

Here is the design decision that shapes everything: **the toolkit ships inside the `gophertrunk` binary but is excluded from the default install.** It sits behind the `cryptolab` build tag — the same mechanism the DVSI vocoder uses. A standard build does not link it in:

```bash
make build                 # default: `gophertrunk cryptolab` just prints how to opt in
make build TAGS=cryptolab  # opt in: the full toolkit is linked
go build -tags cryptolab ./cmd/gophertrunk   # equivalent raw go build
make test-cryptolab        # run the toolkit's tests, including the tagged CLI
```

In a default build, the `cryptolab` subcommand exists but does nothing except tell you how to opt in. Rebuild with `TAGS=cryptolab` and the same subcommand becomes the whole toolkit.

Why bother? Two reasons. First, **the standard operator install stays lean** — someone running GopherTrunk as an unattended scanner at an antenna has no need for a cryptanalysis toolkit in their binary, and now it isn't there. Second, and more importantly, **it's a statement of intent.** Attack tooling is opt-in. You have to deliberately build the security-testing version. That friction is a feature: it keeps the default distribution firmly a *scanner*, not an *interception tool*.

There's a subtlety worth calling out. The toolkit's engine and subject packages under `internal/cryptolab/` carry **no** build tag — they always compile and are covered by the normal `make test`. Only the binary's `cryptolab` *subcommand* is gated. So the cryptographic code is continuously tested; what the build tag controls is merely whether the CLI surface is wired into the shipped binary.

## The CLI shape

Every invocation follows one grammar:

```text
gophertrunk cryptolab [global flags] <tool> [<mode>] [tool flags]
gophertrunk cryptolab list      # list tools and modes
gophertrunk cryptolab serve     # launch the web console
```

**Global flags precede the tool name** — flag parsing stops at the first non-flag argument, then the tool and mode parse their own flags. The globals are `-out` (artifact directory for survivor logs and checkpoints), `-resume` (resume a resumable mode), `-format` (`text|json|jsonl|yaml|csv`), `-log-level`, and `-log-format`. So a full command reads left to right as *globals, then tool, then mode, then that mode's flags*:

```bash
gophertrunk cryptolab -out ./out -format json alias structure -csv ground_truth.csv
```

`cryptolab list` prints the whole catalogue — eleven tools, each with its modes and a one-line synopsis. It's the fastest way to remember what exists. And `cryptolab serve` (covered in Part 9) launches a browser console at `127.0.0.1:8096` that renders a form for every tool automatically. Everything the CLI can do, the console can do — except the handful of operations that run a host program, which are CLI-only for safety.

## The verdict scale and the escalation ladder

The unifying idea across the toolkit is the three-way verdict. Every `assess` run ends on one of three chips:

<figure class="lab-figure">
<svg viewBox="0 0 640 96" width="640" height="96" role="img" aria-label="The verdict scale: RESISTANT to PARTIAL to BROKEN">
  <rect x="8" y="30" width="180" height="40" rx="8" fill="none" stroke="var(--accent)"/>
  <text x="98" y="55" text-anchor="middle" fill="var(--accent)" font-size="14">RESISTANT</text>
  <text x="98" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="11">nothing recovered</text>
  <rect x="230" y="30" width="180" height="40" rx="8" fill="none" stroke="currentColor"/>
  <text x="320" y="55" text-anchor="middle" fill="currentColor" font-size="14">PARTIAL</text>
  <text x="320" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="11">information leaked</text>
  <rect x="452" y="30" width="180" height="40" rx="8" fill="none" stroke="currentColor"/>
  <text x="542" y="55" text-anchor="middle" fill="currentColor" font-size="14">BROKEN</text>
  <text x="542" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="11">verified decryption</text>
  <line x1="188" y1="50" x2="230" y2="50" stroke="currentColor" stroke-width="1.5"/>
  <polygon points="230,50 222,46 222,54" fill="currentColor"/>
  <line x1="410" y1="50" x2="452" y2="50" stroke="currentColor" stroke-width="1.5"/>
  <polygon points="452,50 444,46 444,54" fill="currentColor"/>
</svg>
<figcaption>The three verdicts. Left is the encryption doing its job; right is the encryption failing the test.</figcaption>
</figure>

- <span class="lab-verdict lab-verdict--ok">RESISTANT</span> — no applicable method recovered anything. The encryption held.
- <span class="lab-verdict lab-verdict--warn">PARTIAL</span> — information leaked: a reused IV, structured (non-random) ciphertext, a recovered keystream segment, or an algorithm with a *published* break in use.
- <span class="lab-verdict lab-verdict--bad">BROKEN</span> — a method achieved verified, complete decryption. A fail.

Behind that verdict is a ladder of methods that escalate in capability, from cheap statistical checks to full key recovery:

<figure class="lab-figure">
<svg viewBox="0 0 640 210" width="640" height="210" role="img" aria-label="Method escalation ladder from cipher-strength to key-brute">
  <g font-size="12" fill="currentColor">
  <rect x="20" y="8" width="380" height="26" rx="5" fill="none" stroke="currentColor"/>
  <text x="30" y="25">cipher-strength — is it distinguishable from random?</text>
  <text x="470" y="25" fill="var(--fg-muted)">PARTIAL</text>
  <rect x="40" y="42" width="380" height="26" rx="5" fill="none" stroke="currentColor"/>
  <text x="50" y="59">known-weakness — published break in the algorithm?</text>
  <text x="470" y="59" fill="var(--fg-muted)">PARTIAL</text>
  <rect x="60" y="76" width="380" height="26" rx="5" fill="none" stroke="currentColor"/>
  <text x="70" y="93">iv-reuse — two frames share an IV</text>
  <text x="470" y="93" fill="var(--fg-muted)">PARTIAL</text>
  <rect x="80" y="110" width="380" height="26" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="90" y="127">known-plaintext — recover keystream, decrypt group</text>
  <text x="470" y="127" fill="var(--accent)">BROKEN</text>
  <rect x="100" y="144" width="380" height="26" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="110" y="161">weak-key — default/test keys against the real cipher</text>
  <text x="470" y="161" fill="var(--accent)">BROKEN</text>
  <rect x="120" y="178" width="380" height="26" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="130" y="195">key-brute — reduced keyspace against an oracle</text>
  <text x="470" y="195" fill="var(--accent)">BROKEN</text>
  </g>
</svg>
<figcaption>Methods escalate from cheap exposure checks (top, PARTIAL) to verified full recovery (bottom, BROKEN). Part 6 walks the whole ladder.</figcaption>
</figure>

Reese's rule of thumb: the top of the ladder tells you *this could be weak*; the bottom tells you *this is broken, here's the plaintext*. A responsible assessment reports both, and never claims a break it did not verify.

## The series map

| Part | Topic | What you'll do |
|---|---|---|
| 1 | Breaking It Is the Test (this post) | Understand the philosophy, build tag, and verdicts |
| [2]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }}) | First triage: `classify` & `stats` | Triage an unknown payload and get the next command |
| [3]({{ '/blog/tutorials/crypto-lab-03-randomness-nist-lfsr/' | relative_url }}) | Randomness: NIST SP 800-22 & LFSRs | Tell strong keystreams from scramblers |
| [4]({{ '/blog/tutorials/crypto-lab-04-classical-ciphers-brute/' | relative_url }}) | Classical ciphers: `brute` | Break XOR, Caesar, Vigenère, substitution |
| [5]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }}) | Keystream reuse & many-time-pad | Exploit IV/MI reuse with `ks` |
| [6]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }}) | The `assess` battery | Grade P25/DMR/TETRA end to end |
| [7]({{ '/blog/tutorials/crypto-lab-07-crc-recovery-recipe/' | relative_url }}) | CRC recovery & the recipe pipeline | Recover a CRC and chain transforms |
| [8]({{ '/blog/tutorials/crypto-lab-08-analog-voice-descrambling/' | relative_url }}) | Analog voice descrambling | Undo inversion, split-band, rolling code |
| [9]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }}) | Web console, live bridge, external ciphers | Drive it from a browser; feed it live captures |
| [10]({{ '/blog/tutorials/crypto-lab-10-subject-framework-alias/' | relative_url }}) | Subject framework, `alias` & the Mercury reveal | Recover a keyless obfuscator; solve Mercury |

## Where this goes next

[Part 2]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }}) starts where every investigation should: triage. Before you brute anything, you run `classify auto` and `stats scan` to find out what class of thing you're holding — plaintext, a shift cipher, repeating XOR, a periodic scrambler, an LFSR, or strong encryption — and let the tool hand you the exact next command. If you want the full picture of where these payloads come from, the [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) cover the tool surface, and the sibling series ([Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}), [RF Scope]({{ '/blog/series/rf-scope/' | relative_url }})) cover finding and dissecting the signals that feed it.

## FAQ

**Is Crypto Lab a tool for decrypting other people's radio traffic?**
No. It is a security-testing toolkit for encryption you are authorized to assess — your own deployment, a system you're licensed to test, or research and teaching material. Decrypting traffic you have no authorization for is illegal, and the toolkit's whole framing is defensive: measure how well *your* encryption holds up.

**Why hide it behind a build tag instead of just shipping it?**
Two reasons: it keeps the default operator binary lean, and it makes attack tooling a deliberate opt-in rather than something every install carries by default. The default build's `cryptolab` command just prints how to enable it.

**Can it break AES or a properly-keyed P25 system?**
No, and it says so — it reports <span class="lab-verdict lab-verdict--ok">RESISTANT</span>. It cannot brute a strong key out of a strong cipher. What it finds is misconfiguration: reused IVs, default keys, keyless obfuscation, and weak keystreams — the things that actually break deployments.

**What does "breaking it is the test" mean in practice?**
The `assess` harness tries every applicable attack and grades each. A full decryption means the deployment's encryption failed the test; recovering nothing means it passed. The attempt is the measurement.

## Series navigation

**Part 1 of 10** · Next →
[Part 2: First Triage — `classify auto` and `stats scan`]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }})
