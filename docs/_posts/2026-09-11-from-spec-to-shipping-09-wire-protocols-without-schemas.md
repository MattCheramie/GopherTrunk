---
title: "From Spec to Shipping, Part 9: Wire Protocols Without Schemas"
description: "How GopherTrunk shipped a SoapyRemote client that asked for HAS_DC_OFFSET_MODE when it meant SET_ANTENNA — opcode 600 instead of 501 — and the rules that catch schemaless wire drift: pin constants against upstream literals, validate through introspection, read back what you set."
category: deep-dives
keywords: soapyremote rpc protocol, soapysdr set antenna, unconsumed payload bytes, rpc opcode drift, schemaless wire protocol testing, fake server test double, soapyremotedefs.hpp opcodes, read back configuration, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, soapysdr, rpc, drivers, testing, methodology]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 9
---

*Part 9 of **From Spec to Shipping**, a 14-part series on how a protocol
decoder actually gets written — from standards documents and independent
references to code you can trust on air.
[Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
rebuilt a fabricated air interface from decoders proven on air. This part
takes the same discipline off the air: the RPC socket between GopherTrunk
and a remote radio server, where a wrong constant doesn't fail to lock —
it politely invokes a **different function** and reports success. Enums
copied out of a `.hpp` file are constants too, and everything Part 1 said
about sync words applies to them.*

> **TL;DR:** GopherTrunk's pure-Go SoapyRemote client packed `SET_ANTENNA`
> as opcode **600** — which upstream defines as `HAS_DC_OFFSET_MODE`. The
> wire carries no schema, the wrong handler's first two arguments happened
> to match, the leftover antenna-name string produced only a server-side
> `~SoapyRPCUnpacker: Unconsumed payload bytes 9`, and the `bool` reply
> carried no exception — so `rpcVoid` logged "rx antenna set" for a call
> that never ran. The fake server switched on the **same constant**, so
> every test was green — the
> [self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }})
> at the RPC layer. The fixes, all in `internal/sdr/soapyremote/`:
> opcodes pinned against upstream literals
> (`TestOpenSetAntennaUsesUpstreamOpcode`), a fake asserting every request
> byte consumed, and `applyAntennas` validating against `LIST_ANTENNAS`
> and reading back with `GET_ANTENNA`.

**Key takeaways**

- **A schemaless wire cannot tell you you're wrong.** A wrong call id
  dispatches to whichever handler lives at that number — and if the
  leading argument types line up, the call "works." The failure is silent
  by construction.
- **Numeric ids from someone else's enum are extracted constants.**
  Part 1's rule for sync words and CRC polynomials applies verbatim: cite
  the source (`SoapyRemoteDefs.hpp`), pin the value with a literal an
  independent party wrote.
- **A test double is only worth the strictness it enforces.** A fake that
  tolerates leftover bytes hides argument-shape drift; one that switches
  on your own constants can never see opcode drift. The two nets catch
  different bugs; GopherTrunk needed both.
- **Read back what you set.** A success reply proves the server didn't
  throw, not that the device changed. `applyAntennas` now asks the device
  what port it is actually on — the assertion that survives a wrong
  opcode, a permissive driver, or a config moved between rigs.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| RPC wire format | 12-byte `SRPC` header + type-tagged values + `CPRS` trailer | `internal/sdr/soapyremote/rpc.go` (packer/unpacker) |
| Call ids | positions in upstream's enum, cited to `SoapyRemoteDefs.hpp` | `rpc.go` (`callSetAntenna` = 501 and friends) |
| Opcode pin | numeric ids compared to upstream **literals** | `driver_test.go` (`TestOpenSetAntennaUsesUpstreamOpcode`) |
| Full-consumption check | fake server asserts zero leftover request bytes | `driver_test.go` (`newFakeSoapyServer`, `protocolErrs`) |
| Validate + read back | `LIST_ANTENNAS` before the set, `GET_ANTENNA` after | `driver.go` (`applyAntennas`) |
| Wire visibility | per-frame decoded RPC trace, off by default | `rpcdebug.go` (`sdr.soapy_remote[].verbose_debug`) |

## In this post

- **A schema is a shared assumption** — what the SoapyRemote wire
  carries, and what it doesn't check.
- **Opcode 600: anatomy of a silent miss** — a wrong id as a successful
  call to the wrong function.
- **Why every test was green** — the self-consistent trap, RPC edition.
- **The four nets** — literal pins, strict fakes, introspection, read-back.
- **The general rules** — what transfers to any schemaless wire format.

## A schema is a shared assumption you don't get to check

SoapyRemote is how GopherTrunk drives a networked SDR — a USRP X310
behind a `SoapySDRServer` — with a pure-Go client
(`internal/sdr/soapyremote/`) reverse-engineered from upstream's
`SoapyRPCPacket.hpp` and packer/unpacker sources, in the spirit of the
[device-contract]({{ '/blog/deep-dives/rf-front-end-02-the-device-contract/' | relative_url }})
approach the driver stack takes everywhere.

The format is simple and completely trusting: a 12-byte header (`"SRPC"`,
version, length), a payload of type-tagged values, a 4-byte trailer. The
first value is a CALL carrying a numeric id; the arguments follow as bare
tagged values — no names, no arity, no signature. The server dispatches
on the id, and the handler unpacks however many arguments *it* believes
the call takes.

So the call ids are not values GopherTrunk gets to choose. They are
**positions in an upstream enum**, and the comment in `rpc.go` says so:

```go
// internal/sdr/soapyremote/rpc.go (shape)
// SoapyRemoteCalls — the subset of RPC call IDs this client issues.
//
// These are POSITIONS IN AN UPSTREAM ENUM, not values GopherTrunk gets to
// choose: the wire carries no schema, so a wrong ID silently invokes a
// different server handler whose argument shape happens to start the same
// way. The authority is pothosware/SoapyRemote common/SoapyRemoteDefs.hpp.
const (
    callListAntennas int32 = 500
    callSetAntenna   int32 = 501
    callGetAntenna   int32 = 502
    /* … callSetGain 703, callSetFrequency 800, callSetSampleRate 900 … */
)
```

Read that next to
[Part 1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})'s
rule — *constants first, every constant cites its source*. An opcode is a
sync word for a TCP stream: one wrong value and you are invoking
something else entirely, with no error message.

## Opcode 600: anatomy of a silent miss

For one release, `callSetAntenna` was **600**. Upstream's enum puts
`SET_ANTENNA` at **501**; 600 is `HAS_DC_OFFSET_MODE` — a capability
query. Walk through what the wire did with that, step by step:

1. GopherTrunk packs `CALL(600), char(direction), i32(channel),
   str("RX2")` — the correct argument shape *for SET_ANTENNA*.
2. The server dispatches to the `HAS_DC_OFFSET_MODE` handler, which
   unpacks a direction and a channel — the first two arguments **happened
   to match** — and returns a `bool`.
3. The antenna-name string is never read. The server's unpacker destructor
   logs `~SoapyRPCUnpacker: Unconsumed payload bytes 9` — a string tag, a
   tagged length, `"RX2"` — once per channel, on the *server's* console,
   without saying which call.
4. The `bool` reply reaches the client. `rpcVoid` checks only for an
   exception value; a `bool` is not one, so GopherTrunk logs
   `rx antenna set` for an antenna that was never set.

The operator-visible symptom was maximally misleading: `antennas: [RX1,
RX2]` did nothing, every radio kept its *driver default* port, and the
same config therefore behaved differently on an X310 and a B210. Nothing
failed; nothing warned client-side. The only witness was a cryptic
destructor message in another room.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two-column RPC ladder comparing the intended SET_ANTENNA call with what opcode 600 actually did. Left column: client packs call 501 with direction, channel and the antenna name; the setAntenna handler consumes all arguments, switches the port, and replies void. Right column: client packs call 600 with the same arguments; the HAS_DC_OFFSET_MODE handler consumes only direction and channel, leaves 9 bytes unconsumed which the server logs, never touches the antenna, and replies with a bool that the client's exception check reads as success.">
  <text x="170" y="20" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">intended: CALL 501 (SET_ANTENNA)</text>
  <text x="510" y="20" text-anchor="middle" fill="currentColor" font-size="11" font-weight="bold">shipped: CALL 600 (HAS_DC_OFFSET_MODE)</text>
  <rect x="40" y="32" width="260" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="170" y="51" text-anchor="middle" fill="currentColor" font-size="10">client packs: dir, channel, "RX2"</text>
  <rect x="380" y="32" width="260" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="510" y="51" text-anchor="middle" fill="currentColor" font-size="10">client packs: dir, channel, "RX2"</text>
  <line x1="170" y1="62" x2="170" y2="80" stroke="var(--fg-muted)"/><polygon points="166,78 170,86 174,78" fill="var(--fg-muted)"/>
  <line x1="510" y1="62" x2="510" y2="80" stroke="var(--fg-muted)"/><polygon points="506,78 510,86 514,78" fill="var(--fg-muted)"/>
  <rect x="40" y="86" width="260" height="44" rx="5" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="170" y="104" text-anchor="middle" fill="var(--accent)" font-size="10">setAntenna handler: consumes ALL args,</text>
  <text x="170" y="119" text-anchor="middle" fill="var(--accent)" font-size="10">switches the RX port on the device</text>
  <rect x="380" y="86" width="260" height="44" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="510" y="104" text-anchor="middle" fill="currentColor" font-size="10">capability handler: consumes dir + channel,</text>
  <text x="510" y="119" text-anchor="middle" fill="currentColor" font-size="10">never reads "RX2" — antenna untouched</text>
  <line x1="170" y1="130" x2="170" y2="148" stroke="var(--fg-muted)"/><polygon points="166,146 170,154 174,146" fill="var(--fg-muted)"/>
  <line x1="510" y1="130" x2="510" y2="148" stroke="var(--fg-muted)"/><polygon points="506,146 510,154 514,146" fill="var(--fg-muted)"/>
  <rect x="40" y="154" width="260" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="170" y="173" text-anchor="middle" fill="currentColor" font-size="10">reply: void → success, and true</text>
  <rect x="380" y="154" width="260" height="30" rx="5" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="510" y="173" text-anchor="middle" fill="currentColor" font-size="10">reply: bool → no exception → "success"</text>
  <rect x="380" y="192" width="260" height="30" rx="5" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="510" y="211" text-anchor="middle" fill="var(--fg-muted)" font-size="10">server log: "Unconsumed payload bytes 9"</text>
  <text x="170" y="211" text-anchor="middle" fill="var(--fg-muted)" font-size="10">device is on RX2</text>
  <text x="340" y="240" text-anchor="middle" fill="currentColor" font-size="10">same bytes from the client, two different functions — and both ladders report success</text>
</svg>
<figcaption>The wrong opcode doesn't fail — it dispatches to a handler whose leading argument types happen to fit, and the only evidence is a destructor complaint on the far machine.</figcaption>
</figure>

## Why every test was green

The unit tests could not catch this — the series' villain in a new
costume. The package tests against `newFakeSoapyServer`, a real TCP
listener speaking the real framing — but the fake's dispatch `switch` was
written against **the same Go constants the client packs**. Change
`callSetAntenna` to any value and both sides move together: the client
sends 600, the fake's `case callSetAntenna` matches 600, the test passes.
The test verified that GopherTrunk agrees with itself — which it always
does.

That is the round-trip failure of
[Part 3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})'s
SCCB parser and
[Part 8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})'s
fabricated SmartNet framing, transplanted to a TCP socket. A shared
constant is a shared assumption, and no test living entirely inside the
assumption can falsify it. (Not this protocol's first postmortem, either —
the stream-handshake saga in
[From the Issue Tracker Part 13]({{ '/blog/solution-postmortem/from-the-issue-tracker-13-soapyremote-handshake/' | relative_url }})
is the same wire biting a different way.)

## The four nets, and what each one catches

The fix was not one test but four distinct nets, because the failure has
four distinct faces — and any one net alone leaves a hole:

| Failure face | Net that catches it | Pinned by |
|---|---|---|
| Opcode drift — wrong call id | literals transcribed from upstream | `TestOpenSetAntennaUsesUpstreamOpcode` |
| Argument-shape drift | fake asserts full byte consumption | `newFakeSoapyServer` cleanup |
| Port name not on this device | validate via `LIST_ANTENNAS` | `TestOpenRejectsAntennaNotOnDevice` |
| Setter silently ignored | `GET_ANTENNA` read-back | `applyAntennas` |
| "Which call was that?" | decoded per-frame RPC trace | `verbose_debug` (`rpcdebug.go`) |

**Net 1: pin opcodes against upstream literals.** The only test that
catches *opcode* drift is one whose expected value came from outside the
codebase:

```go
// internal/sdr/soapyremote/driver_test.go (shape) — TestOpenSetAntennaUsesUpstreamOpcode
// Values from pothosware/SoapyRemote common/SoapyRemoteDefs.hpp. Written
// as literals on purpose: comparing the constant to itself proves nothing.
for _, tc := range []struct {
    name string
    got  int32
    want int32
}{
    {"LIST_ANTENNAS", callListAntennas, 500},
    {"SET_ANTENNA", callSetAntenna, 501},
    {"GET_ANTENNA", callGetAntenna, 502},
    /* … SET_GAIN 703, SET_FREQUENCY 800, SETUP_STREAM 300 … */
} {
    if tc.got != tc.want {
        t.Errorf("%s opcode = %d, want %d (upstream SoapyRemoteDefs.hpp)",
            tc.name, tc.got, tc.want)
    }
}
```

Sixteen calls, sixteen literals transcribed from the `.hpp`. It looks too
dumb to be a test — `501 == 501` — and that is the point: the literal is
a fact from an independent source, the role the OP25 CRC spans and the
ETSI reference codec play elsewhere in this series. Reference-literal
tests are the only tests that catch constant drift, on the air or off it.

**Net 2: make the fake as strict as the real server.** The fake now turns
the real `~SoapyRPCUnpacker`'s log line into a failure: every handler
parses its arguments, a request with bytes left over (or an unknown id)
lands in `protocolErrs`, and `newFakeSoapyServer` registers an
`assertCleanProtocol` cleanup so **every** test in the package gets the
check for free. This catches *argument-shape* drift — and adding it
immediately found **two more calls** the fake had silently not been
parsing. It cannot catch opcode drift (the fake still switches on the
client's constants), which is why Net 1 exists separately. The general
rule — test doubles ranked by how much reality they enforce — got its own
treatment in
[Part 7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }}).

**Net 3: introspect, then read back.** The deepest fix is in the
production path:

```go
// internal/sdr/soapyremote/driver.go (shape) — applyAntennas
avail, err := d.listAntennas(ch) // LIST_ANTENNAS (500)
/* … */
if len(avail) > 0 && !slices.Contains(avail, name) {
    return fmt.Errorf("channel %d antenna %q is not a port on this device (available: %s)",
        ch, name, strings.Join(avail, ", "))
}
/* … SET_ANTENNA (501) … */
got, err := d.getAntenna(ch) // GET_ANTENNA (502)
/* … */
if got != name {
    return fmt.Errorf("channel %d antenna set to %q but device reports %q", ch, name, got)
}
d.log.Info("soapyremote: rx antenna set", "addr", d.addr, "channel", ch, "antenna", got)
```

Three RPCs where one used to be. `LIST_ANTENNAS` matters because port
names do not transfer between radios — a TwinRX offers `RX1`/`RX2`, a
B210 `TX/RX`/`RX2` — so a config moved between rigs must fail loudly with
the names that *do* exist (`TestOpenRejectsAntennaNotOnDevice` pins the
error message). The `GET_ANTENNA` read-back survives everything: a wrong
opcode, a driver that ignores the setter, a typo'd port. And the log line
now reports what the **device** said, not what GopherTrunk had asked —
the entire difference between a log and an instrument.

**Net 4: make the wire visible.** `sdr.soapy_remote[].verbose_debug`
enables a per-frame RPC tracer rendering each request and response as a
*decoded* argument list — the form that lines up against upstream's
handler source, which a raw hex dump does not. When the server next says
"Unconsumed payload bytes N," the question "which call?" has a
client-side answer.

## The rules, genericized

Nothing above is SoapySDR-specific. Any schemaless wire — a binary RPC, a
vendor USB control protocol, a register map, a serial command set — has
the same failure faces, and earns the same nets:

- **Treat foreign enum values as extracted constants** (Part 1's rule):
  cite the defining file, pin every value with a literal transcribed from
  it — never from your own header.
- **Give your test double the real endpoint's strictness, mechanically** —
  wired into the constructor, so no test can forget it.
- **Prefer the protocol's introspection to your beliefs.** A peer that
  enumerates its own capabilities is an independent reference shipping
  inside the protocol.
- **Read back every write that matters.** A non-exception reply means
  "the server didn't throw," nothing more — a rule the
  [testing module]({{ '/learn/testing/' | relative_url }}) generalizes
  beyond wire protocols.

## Where this goes next

Every net here defends a claim made *before* the hardware is in the
loop — and passing all four still proves nothing on a real system.
[Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
makes that boundary explicit: the on-air gate, the ladder of evidence
every decoder claim climbs, and the DMO "encrypted" verdict overturned
capture by capture.

## FAQ

**How does a wrong RPC opcode go unnoticed when the arguments don't match?**
Because nothing compares them. The handler unpacks what *it* expects; if
the leading types coincide — a direction byte and a channel int are
near-universal prefixes in SoapySDR calls — it runs happily on the
prefix, and the surplus bytes are at most a log line in a destructor.

**Why not just generate the Go constants from SoapyRemoteDefs.hpp?**
Generation moves the trust to the generator and its input snapshot, which
can drift just as silently. The literal test is smaller and cleaner:
sixteen numbers transcribed from the authority, and any refactor changing
one must argue with a named upstream file in the failure message.

**What does "Unconsumed payload bytes N" from SoapySDRServer actually mean?**
The dispatched handler finished unpacking with N bytes of your request
still unread — your call id and your argument list belong to two
different calls. It doesn't say which RPC; the client-side tracer
(`verbose_debug`) exists to answer that.

**Is a fake server worth having if it can't catch opcode drift?**
Yes — it catches everything else: framing, argument shapes, sequencing,
error paths. The discipline is knowing what a double *cannot* see and
covering that face with a different net, not retiring the double.

**Do these rules apply to protocols that have schemas, like gRPC?**
Less force, same direction. A schema checks names and types, but semantic
drift — wrong units, a deprecated endpoint that still answers — passes
any schema. Read-back and introspection stay worth their cost wherever a
success reply is cheaper to send than the write is to perform.

## Series navigation

**Part 9 of 14** · ←
[Part 8: Case Study — Rebuilding SmartNet From Proven Decoders]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }})
· Next →
[Part 10: The On-Air Gate — Green Synthetics Prove Nothing]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
