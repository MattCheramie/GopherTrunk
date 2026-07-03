---
title: "Crypto Lab, Part 9: The Web Console, Live-Capture Bridge & External Ciphers"
description: Crypto Lab's serve command launches a schema-driven web console that auto-renders a form for every tool, mounts inside the daemon when built with the cryptolab tag, feeds ks and assess from a live decoder bridge, and drives operator-supplied external ciphers as CLI-only subprocesses behind a strict safety boundary.
category: tutorials
keywords: cryptolab serve, web console, recipe builder, live capture bridge, crypto_capture_path, p25 ldu2 superframe, external cipher, tea1 subprocess, safety boundary, gophertrunk cryptolab
tags: [cryptolab, web-console, live-capture, external-cipher, security-testing, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 9
---

*Part 9 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. The surfaces around the tools: a browser console, a live feed from the decoder, and a carefully-fenced door for external ciphers.*

> **TL;DR:** `cryptolab serve` runs a browser console at `127.0.0.1:8096` (`-addr`, `-open`, `-tmp-dir`, `-max-upload`, `-log-level`, `-log-format`) that renders a form for every tool from `GET /api/v1/cryptolab/tools`, plus a Recipe Builder tab. Built into the daemon with `-tags cryptolab`, it mounts at `/cryptolab/` and Signal Lab shows a 🔐 link. The live bridge (`recordings.crypto_capture_path`) appends one JSON record per encrypted P25 LDU2 superframe; point `ks`/`assess` at that JSONL. External ciphers are **not bundled** — you drive an operator-supplied cipher as a subprocess (`-extern-cmd`), and because that runs a host program it is **CLI-only** (the web console hides it; the recipe endpoint returns HTTP 403).

> **Authorized testing only.** The bridge captures encrypted material — enable it only on systems you own or are licensed to assess.

**Key takeaways**

- **The console is schema-driven.** Every tool, mode, and parameter auto-gets a control from the backend schema — new tools appear automatically.
- **The bridge closes the live loop.** A running decoder appends encrypted frames as JSONL; `ks reuse` / `assess crypto` read it directly.
- **External ciphers stay unbundled.** GopherTrunk drives your vetted cipher as a subprocess rather than shipping a licence-incompatible one.
- **The safety boundary is hard.** Anything that runs a host program is CLI-only; the browser can never trigger it.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab serve -open` | Serve the console at `127.0.0.1:8096` and open a browser |
| `-addr host:port` | Change the listen address |
| `-tmp-dir dir` / `-max-upload bytes` | Staging dir / upload cap (0 = 512 MiB default) |
| `recordings.crypto_capture_path:` | Config: where the decoder appends encrypted frames |
| `assess crypto -extern-cmd tea1-tool -extern-algid 0x01 -brute-bits 32` | Drive an external cipher (CLI only) |
| `GET /api/v1/cryptolab/tools` | The schema the console renders from |

## In this post

- **The web console** — schema-driven forms and the Recipe Builder.
- **Daemon mount** — how the console appears inside a running GopherTrunk.
- **The live-capture bridge** — decoder to JSONL to `ks`/`assess`.
- **External ciphers** — driving an unbundled cipher as a subprocess.
- **The safety boundary** — why external ops are CLI-only.

## The web console

`cryptolab serve` is the browser counterpart to the CLI, mirroring the `siglab serve` pattern:

```bash
make cryptolab-web-build           # bundle the SPA into web/cryptolab/dist/
make build TAGS=cryptolab          # link the toolkit + console into gophertrunk
gophertrunk cryptolab serve -open  # serve at 127.0.0.1:8096 and open a browser
```

What makes it maintainable is that it's **schema-driven**. The frontend renders its forms from `GET /api/v1/cryptolab/tools`, the backend's own description of every tool, mode, and parameter. Each parameter — file upload, text, number, checkbox — gets the right control automatically, so when a new tool is added to the toolkit, it appears in the console with no frontend change. A run uploads the inputs, streams the live log, and shows the structured result: summary, fields, ranked findings, notes, and downloadable artifacts (survivor logs, checkpoints, descrambled output). It runs entirely offline against uploaded files — no SDR or daemon required.

The flags mirror the CLI's serve: `-addr` (default `127.0.0.1:8096`), `-open` (launch a browser once it's up), `-tmp-dir` (staging directory for uploads), `-max-upload` (byte cap; `0` = the 512 MiB default), and the usual `-log-level`/`-log-format`. The **Recipe Builder** tab drives the [Part 7]({{ '/blog/tutorials/crypto-lab-07-crc-recovery-recipe/' | relative_url }}) pipeline interactively — pick an input, assemble ops from a palette with move/duplicate/remove and per-op fields, run, and read the per-step report plus the final bytes. It's backed by two endpoints: `GET /api/v1/cryptolab/recipe/ops` (the op catalogue) and `POST /api/v1/cryptolab/recipe` (run a structured recipe over an uploaded file).

## Daemon mount

When the main `gophertrunk` daemon is itself built with `-tags cryptolab`, the same console mounts *inside* it at `/cryptolab/` (its API under `/api/v1/cryptolab/`, alongside the siglab routes), so you can reach it from a running daemon without launching a separate `serve`. Mutating routes share the daemon's mutation gate. The default daemon build links a no-op mount, so the toolkit stays out of the standard binary.

In that build the console is **discoverable from the other web UIs**: the main GopherTrunk console shows a **Crypto Lab** entry in its System nav, and the Signal Lab header shows a **🔐 Crypto Lab** link. Both are gated on `runtime.cryptolab_console` (from `GET /api/v1/runtime`), so the link only appears when the console is actually mounted — a default build or a standalone siglab never shows a dead link. That gating is why Ada, running a default scanner build, simply never sees the Crypto Lab link, while Reese's `-tags cryptolab` daemon surfaces it in both headers.

## The live-capture bridge

The most powerful integration is feeding Crypto Lab straight from live decode. Set one config key:

```yaml
recordings:
  crypto_capture_path: /var/lib/gophertrunk/crypto-frames.jsonl
```

When set, the P25 Phase 1 voice composer appends one JSON record per encrypted **LDU2 superframe** to that file — `{label, iv (Message Indicator), ct (encrypted voice frames), system, protocol, tg, algid, keyid, at}`. That's exactly the frames format `ks` and `assess` read ([Part 5]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }})), so you point them straight at the growing file:

<figure class="lab-figure">
<svg viewBox="0 0 640 120" width="640" height="120" role="img" aria-label="Live bridge: decoder appends encrypted superframes to JSONL, which ks and assess read">
  <g font-size="11.5">
  <rect x="8" y="42" width="130" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="73" y="60" text-anchor="middle" fill="currentColor">P25 decoder</text>
  <text x="73" y="75" text-anchor="middle" fill="var(--fg-muted)" font-size="9">encrypted LDU2</text>
  <rect x="200" y="42" width="180" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="290" y="60" text-anchor="middle" fill="var(--accent)">crypto-frames.jsonl</text>
  <text x="290" y="75" text-anchor="middle" fill="var(--fg-muted)" font-size="9">{label, iv=MI, ct, ...}</text>
  <rect x="446" y="24" width="186" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="539" y="43" text-anchor="middle" fill="currentColor">ks reuse / ks mtp</text>
  <rect x="446" y="70" width="186" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="539" y="89" text-anchor="middle" fill="currentColor">assess crypto</text>
  <g stroke="currentColor" stroke-width="1.3" fill="currentColor">
  <line x1="138" y1="62" x2="198" y2="62"/><polygon points="200,62 192,58 192,66"/>
  <line x1="380" y1="55" x2="444" y2="40"/><polygon points="446,39 437,39 441,47"/>
  <line x1="380" y1="70" x2="444" y2="84"/><polygon points="446,85 437,79 441,87"/>
  </g>
  </g>
</svg>
<figcaption>With <code>crypto_capture_path</code> set, the decoder appends one JSONL record per encrypted superframe; <code>ks</code> and <code>assess</code> read that same file offline.</figcaption>
</figure>

```bash
gophertrunk cryptolab ks reuse -in /var/lib/gophertrunk/crypto-frames.jsonl
gophertrunk cryptolab assess crypto -in /var/lib/gophertrunk/crypto-frames.jsonl
```

The capture records encrypted material and its IV; the decryption *attempts* all run offline in `ks`/`assess`, never on the live path. Crucially, an **empty** `crypto_capture_path` (the default) disables the bridge entirely — no extraction work runs on the voice path, so the standard operator build is completely unaffected. This is the same feed that produced Ada's `mercury-frames.jsonl`, and it's the through-line that connects a live antenna to the analysis bench. It complements [RF Scope Part 7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }})'s `-frames-out`, which produces the identical format from an offline capture.

## External ciphers

The toolkit deliberately does **not** bundle proprietary or reverse-engineered ciphers it can't verify, or whose only implementations are licence-incompatible — the public **TETRA TEA1–4** reference is AGPL, while GopherTrunk is Apache-2.0. Rather than ship or vouch for such a cipher, `assess` can drive an **operator-supplied** cipher program as a subprocess. You point it at a vetted tool that implements a small, shell-free line protocol:

```bash
# Brute the TEA1 32-bit backdoor via your TEA1 tool, then decrypt + grade.
gophertrunk cryptolab assess crypto -in frames.jsonl -protocol tetra \
    -known-label call-3 -known-pt known.bin \
    -extern-cmd "tea1-tool" -extern-algid 0x01 -brute-bits 32
```

The external program exposes two verbs — `keystream <key> <iv> <n>` and `brute <iv> <known_keystream> <bits> <base_key>` — and the harness orchestrates them, verifies the hit against the known-plaintext oracle, and decrypts the corpus. The `brute` verb keeps the heavy key search native in the cipher; GopherTrunk just drives it. Already recovered the key elsewhere? Hand it via `-keys` with no `-brute-bits`, and `assess` verifies it through the external cipher and decrypts every frame. This is how the TEA1 backdoor from [Part 6]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }}) becomes a *runnable* break rather than just an advisory — without GopherTrunk shipping the cipher.

## The safety boundary

Here's the hard rule, and it's a deliberate security design, not an oversight. **External ciphers run a host program**, so `-extern-cmd` and the `extern-decrypt` recipe op are **CLI-only**:

- The web console **hides** the `extern-decrypt` op from its palette.
- `POST /api/v1/cryptolab/recipe` **refuses** an `extern-decrypt` step, returning **HTTP 403**.

So a browser request — even against a daemon-mounted console — can never execute a program on the host. The reasoning is straightforward: a web endpoint that runs arbitrary local commands is a remote-code-execution primitive, and no analysis convenience is worth that. Anything that shells out stays behind the CLI, where the operator running the command already has shell access anyway. Reese's framing: *the browser can analyze bytes all day, but it doesn't get to launch processes.* That boundary is what lets the daemon mount the console with confidence.

## Where this goes next

One tool remains, and it's the one that solves Mercury. [Part 10]({{ '/blog/tutorials/crypto-lab-10-subject-framework-alias/' | relative_url }}) covers the pluggable **subject framework** and the `alias` tool that recovers length-seeded, keyless byte obfuscators — and reveals that Mercury was never encryption at all. It closes the Mercury thread and the whole Lab Bench trilogy. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) cover the console, bridge, and external-cipher protocol in full.

## FAQ

**Does the web console do everything the CLI does?**
Almost — it renders a form for every tool from the backend schema and drives recipes interactively. The one exception is anything that runs a host program (external ciphers), which is CLI-only for safety.

**What does the live bridge capture, and is it on by default?**
It appends one JSONL record per encrypted P25 LDU2 superframe to `crypto_capture_path`. It's off by default (empty path); when off, no extraction runs on the voice path at all. Enable it only for authorized assessment.

**Why doesn't GopherTrunk just bundle TEA1?**
Two reasons: it can't verify a reverse-engineered cipher, and the public TEA1 reference is AGPL while GopherTrunk is Apache-2.0. Instead it drives an operator-supplied, vetted implementation as a subprocess.

**Why is `extern-decrypt` blocked over HTTP?**
Because it runs a host program. A web endpoint that executes local commands is a remote-code-execution risk, so the console hides the op and the recipe API returns HTTP 403. External ciphers stay CLI-only.

## Series navigation

**Part 9 of 10** · ←[Part 8: Analog Voice Descrambling]({{ '/blog/tutorials/crypto-lab-08-analog-voice-descrambling/' | relative_url }}) · Next →[Part 10: The Subject Framework, `alias` & the Mercury Reveal]({{ '/blog/tutorials/crypto-lab-10-subject-framework-alias/' | relative_url }})
