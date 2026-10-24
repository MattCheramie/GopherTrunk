---
title: "From the Issue Tracker, Season 2, Part 8: 'unable to encode sample rate 880029' — FLAC's Frame-Header Table"
description: "Why a FLAC IQ capture of an 880 kHz slice aborted after its first block — the FLAC frame header's 4-bit sample-rate code has no encoding for 880029 Hz or even 880000 Hz, the upstream encoder never falls back to the 'get from STREAMINFO' code, and a 20-bit STREAMINFO field caps any FLAC at 1048575 Hz — and the one shared policy function that now keeps every FLAC writer in the tree honest."
category: solution-postmortem
keywords: flac sample rate 880029, unable to encode sample rate, flac frame header sample rate code, flac streaminfo 20 bit sample rate, flac iq recording sdr, FLACFrameSampleRate, FLACMaxSampleRateHz, mewkiz flac encoder, siglab flac capture, gophertrunk from the issue tracker season 2
tags: [from-the-issue-tracker-s2, flac, capture, recording, siglab, go, postmortem]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From the Issue Tracker, Season 2"
series_part: 8
---

*Part 8 of **From the Issue Tracker, Season 2**, a 14-part run of
postmortems that continues the
[first season]({{ '/blog/series/from-the-issue-tracker/' | relative_url }}):
one bug per part, with receipts.
[Part 7]({{ '/blog/solution-postmortem/issue-tracker-s2-07-capture-at-the-wrong-centre/' | relative_url }})
followed a lost centre frequency through the SigLab capture form. The
operator's *first* attempt at that same grab never produced a file at all:
asked for an 880 kHz slice as FLAC, the daemon waited out the capture and
then aborted with `unable to encode sample rate 880029`. This part reads
the table that error comes from.*

> **TL;DR:** FLAC stores a stream's sample rate twice: in the 20-bit
> STREAMINFO field (ceiling 1 048 575 Hz) and in every frame header's
> 4-bit code, which carries a rate verbatim only if it is one of eleven
> fixed audio rates, ≤ 65 535 Hz, ≤ 655 350 Hz in tens of Hz, or ≤ 255 kHz
> in whole kHz; code `0000` means "get it from STREAMINFO". A siglab slice
> of a 6.25 MS/s X310 at 880 kHz lands at 880 029 Hz through the DDC's
> capped L/M — odd, above 65 535, not whole kHz — and `mewkiz/flac` v1.0.14
> never picks `0000` on its own, so the first block failed; a clean
> 880 000 Hz would have failed too. `baseband.FLACFrameSampleRate` now
> returns the rate when a frame can carry it and 0 otherwise, shared by
> `FLACIQEncoder` and `voice.FlacWriter`; `FLACMaxSampleRateHz` = 1<<20 − 1
> is refused at construction instead of truncated; and the capture route
> 400s a full-band FLAC over the ceiling before pinning the tuner. Pinned
> by `TestFLACIQWriterEncodesFrameHeaderUnrepresentableRates`,
> `TestFLACFrameSampleRate`, `TestSiglabCaptureFLACSliceAtOddRate` (the
> operator's request verbatim) and
> `TestSiglabCaptureFLACFullBandOverCeilingIs400`.

**Key takeaways**

- **A format that stores one number in two places has two failure modes.**
  The frame header's code table and STREAMINFO's 20-bit field have
  different ranges; an IQ rate can be legal in one and not the other.
- **An audio container at radio sample rates is off-label.** Every rate the
  frame header encodes verbatim is one an audio engineer would choose;
  880 029 Hz is not, and neither is a round 880 000.
- **Refuse before the wait, not after it.** The ceiling is known from the
  request; the route now 400s before pinning the tuner instead of aborting
  after 60 s.
- **One policy function for every writer.** Two FLAC writers with their own
  frame-header logic would drift; `FLACFrameSampleRate` is the single place
  the rule lives.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Frame-header rate policy | verbatim when the 4-bit code can carry it, else 0 (= STREAMINFO) | `internal/sdr/baseband/flac.go` (`FLACFrameSampleRate`) |
| STREAMINFO ceiling | `1<<20 − 1` = 1 048 575 Hz, refused at construction | `FLACMaxSampleRateHz`, `NewFLACIQEncoder`, `voice.NewFlacWriter` |
| Shared encode core | stereo 16-bit I/Q, 4096-sample blocks, used by `FLACIQWriter` and `siglab.IQContainer` | `baseband.FLACIQEncoder` (`flushBlock`) |
| Mono voice twin | same policy, `ChannelsMono` | `internal/voice/flac.go` (`FlacWriter.flushBlock`) |
| Route guard | 400 for a full-band FLAC over the ceiling, before the tuner is pinned | `internal/api/capture.go` |
| Diversity fallback | FLAC branch capture above 1 MS/s falls back to cs16 with a WARN | `soapyremote/branchcapture.go` (`flacCaptureMaxRateHz`) |
| Failing-first pins | 880 029 / 880 000 / 1 000 000 / max round-trip; the operator's request verbatim | `flac_rate_test.go`, `api/capture_flac_rate_test.go` |

## In this post

- **The error and the number** — how an 880 kHz request becomes 880 029 Hz.
- **Two places for one rate** — STREAMINFO's 20 bits and the frame header's 4.
- **The code the encoder never picks** — `0000` and the upstream switch.
- **One policy, two writers** — `FLACFrameSampleRate` and where it is called.
- **The ceiling, refused early** — the 400 and the diversity fallback.
- **What the pins cover** — the operator's request as a test.

## The error and the number

The 15 Sep request was a 60 s slice: an X310 streaming 6.25 MS/s, centre
442.8125 MHz, bandwidth 880 kHz, format `flac`. The capture route builds
one `siglab.NewStreamDownconverter` per slice and takes its output rate
as `uint32(ddc.OutRateHz() + 0.5)`. The rational resampler inside it caps
its L/M, so 6.25 MS/s to "880 kHz" lands at **880 029 Hz**, not 880 000 —
a perfectly good rate for a cs16 body, which carries no rate at all, and
for a WAV header, whose rate field is a plain 32-bit integer. The FLAC
container is different. The capture ran, the first block was handed to the encoder, and it
returned `unable to encode sample rate 880029`; the capture aborted after
3908 samples, and the operator had waited out the grab for nothing. Their second attempt, as
cs16, is the one Part 7 took apart.

It is the next layer down from a bug in the FLAC rollout itself: the
streaming `EncodeCapture` had no container case and fell through to u8, so
a staged "flac" capture once got a mislabelled body, fixed by routing
wav/flac through `siglab.IQContainer`. Here the container is right, and
the format cannot describe the rate.

## Two places for one rate

A FLAC stream stores the sample rate in **two** places. STREAMINFO — the
mandatory first metadata block — holds it in a 20-bit field, so the
largest rate any FLAC can declare is 1 048 575 Hz. Then every frame header
carries a 4-bit sample-rate code, so that a decoder which has lost the
metadata (a stream cut mid-file, a seek into the middle) can still decode
a frame on its own. The table is small:

<figure class="lab-figure">
<svg viewBox="0 0 680 230" width="680" height="230" role="img" aria-label="A table of the FLAC frame header's 4-bit sample-rate codes. Code 0000 means get the rate from STREAMINFO. Codes 0001 to 1011 are eleven fixed audio rates from 8 kilohertz to 192 kilohertz. Code 1100 carries an 8-bit rate in whole kilohertz up to 255 kilohertz. Code 1101 carries a 16-bit rate in hertz up to 65535. Code 1110 carries a 16-bit rate in tens of hertz up to 655350. Code 1111 is invalid. Three rates are plotted against the table: 880029 hertz matches no code, 880000 matches no code, and 1000000 matches no code, so all three need code 0000, while STREAMINFO's 20-bit field caps everything at 1048575.">
  <text x="340" y="14" text-anchor="middle" fill="currentColor" font-size="10" font-weight="bold">FLAC frame header: 4-bit sample-rate code</text>
  <g font-size="9" fill="currentColor">
    <text x="30" y="40" font-family="monospace">0000</text><text x="80" y="40">get from STREAMINFO (20-bit field, ≤ 1 048 575 Hz)</text>
    <text x="30" y="58" font-family="monospace">0001–1011</text><text x="110" y="58">eleven fixed audio rates: 8 k … 48 k, 88.2 k, 96 k, 176.4 k, 192 k</text>
    <text x="30" y="76" font-family="monospace">1100</text><text x="80" y="76">8-bit suffix, whole kHz — ≤ 255 000 and rate % 1000 == 0</text>
    <text x="30" y="94" font-family="monospace">1101</text><text x="80" y="94">16-bit suffix, Hz — ≤ 65 535</text>
    <text x="30" y="112" font-family="monospace">1110</text><text x="80" y="112">16-bit suffix, tens of Hz — ≤ 655 350 and rate % 10 == 0</text>
    <text x="30" y="130" font-family="monospace">1111</text><text x="80" y="130" fill="var(--fg-muted)">invalid</text>
  </g>
  <line x1="30" y1="142" x2="650" y2="142" stroke="var(--fg-muted)"/>
  <g font-size="9">
    <text x="30" y="162" fill="var(--accent)" font-family="monospace">880 029</text><text x="110" y="162" fill="currentColor">odd → not 1110; &gt; 65 535 → not 1101; not whole kHz → not 1100</text><text x="640" y="162" fill="var(--accent)" text-anchor="end" font-weight="bold">0000</text>
    <text x="30" y="180" fill="var(--accent)" font-family="monospace">880 000</text><text x="110" y="180" fill="currentColor">&gt; 655 350 → not 1110; &gt; 255 000 → not 1100</text><text x="640" y="180" fill="var(--accent)" text-anchor="end" font-weight="bold">0000</text>
    <text x="30" y="198" fill="var(--accent)" font-family="monospace">1 000 000</text><text x="110" y="198" fill="currentColor">same; still under the STREAMINFO ceiling</text><text x="640" y="198" fill="var(--accent)" text-anchor="end" font-weight="bold">0000</text>
    <text x="30" y="216" fill="var(--fg-muted)" font-family="monospace">2 400 000</text><text x="110" y="216" fill="var(--fg-muted)">over 1 048 575: no FLAC can carry it — refused at construction</text>
  </g>
</svg>
<figcaption>The frame header can carry an audio rate verbatim; every SDR slice rate above 65 535 Hz that is not a round multiple needs code 0000, and the upstream encoder never chose it.</figcaption>
</figure>

Read against that table, 880 029 fails three ways: it is odd, so the
tens-of-Hz code cannot carry it; it is above 65 535, so the plain-Hz code
cannot; and it is not a whole kilohertz. A clean 880 000 Hz — had the
resampler produced one — fails too, because 880 000 is above 655 350. So
does 1 000 000. Every one of them is comfortably under the STREAMINFO
ceiling, and every one of them needs the frame header to say `0000`.

## The code the encoder never picks

GopherTrunk encodes through `github.com/mewkiz/flac` v1.0.14. Its
`encode_frame.go` writes the sample-rate code with a `switch` over the
fixed rates and then a `default` that tries the three suffix codes in
turn — whole kHz, then Hz, then tens of Hz — and, finding none that fits,
returns `errutil.Newf("unable to encode sample rate %v", sampleRate)`.
A header whose `SampleRate` field is 0 is written as code `0000`, but
**nothing selects it automatically**: the encoder writes whatever rate the
caller put in the frame header, and GopherTrunk's writers put the stream
rate there, because for every rate a
recorder had ever used — 8 kHz voice, 48 kHz channelised IQ, 144 kHz
TETRA, 200 and 250 kS/s MRC captures — that was representable and
produced a stream a third-party decoder could seek without reading
STREAMINFO. The first unrepresentable rate was the first FLAC slice off a
multi-MS/s tuner.

The comment on the fix records the two cases that matter:

```go
// internal/sdr/baseband/flac.go (shape)
// FLACFrameSampleRate returns the sample rate to stamp in each FLAC FRAME
// header for a stream whose STREAMINFO carries rate: the rate itself when the
// frame header can encode it verbatim, else 0 — the spec's "get from
// STREAMINFO" code. … (15 Sep: a siglab slice carved at 880029 Hz, and a clean
// 880000 Hz would have failed the same way). Shared by the IQ encode core and
// the mono voice twin so the two writers cannot drift.
func FLACFrameSampleRate(rate uint32) uint32 {
    switch rate {
    case 88200, 176400, 192000, 8000, 16000, 22050, 24000, 32000, 44100, 48000, 96000:
        return rate
    }
    switch {
    case rate <= 255000 && rate%1000 == 0,
        rate <= 65535,
        rate <= 655350 && rate%10 == 0:
        return rate
    }
    return 0
}
```

The function mirrors the encoder's own switch exactly, which is the point:
it answers "will the upstream switch succeed?" before the upstream switch
runs, and substitutes the one answer the encoder will not supply for
itself.

## One policy, two writers

There are two FLAC writers in the tree and they share one encode policy
by construction. `baseband.FLACIQEncoder` is the stereo 16-bit I/Q core
(I left, Q right, 4096-sample blocks) that both the file-owning
`FLACIQWriter` — `baseband.record[].format: flac`, the replay driver's
content-sniffed `.flac` mount — and `siglab.IQContainer` feed; the two
differ only in their float→int16 scale (×32767 vs ×32768) and both call
the same `flushBlock`. `voice.FlacWriter` is the mono twin for per-call
recordings under `recordings.format: flac`, with the same
`WriteSamples` / `DataBytes` / `Close` surface as `WavWriter` so the
recorder treats them interchangeably. Both `flushBlock`s now stamp the
header the same way:

```go
// internal/voice/flac.go (shape) — FlacWriter.flushBlock
hdr := frame.Header{
    HasFixedBlockSize: true,
    BlockSize:         uint16(n),
    // The frame header cannot carry every rate STREAMINFO can; the
    // shared policy defers to STREAMINFO where it can't.
    SampleRate:    baseband.FLACFrameSampleRate(w.sampleRate),
    Channels:      frame.ChannelsMono,
    BitsPerSample: 16,
}
```

The reader side needs nothing: `ReadIQFLACSamples` returns the rate from
STREAMINFO, and `IsFLACIQFile` sniffs the `fLaC` marker rather than the
extension, so a stream whose frames say `0000` decodes exactly as one
whose frames say 48 000. `TestFLACIQWriterEncodesFrameHeaderUnrepresentableRates`
writes 10 000 samples — more than two blocks, so several frame headers are
emitted — at 880 029, 880 000, 1 000 000 and `FLACMaxSampleRateHz`, reads
each file back, and checks the STREAMINFO rate and every int16 pair; it
fails against the old encoder at every rate. `TestFLACFrameSampleRate`
pins the policy table directly: 8 000, 48 000, 144 000, 250 000, 65 535 and
655 350 stay verbatim; 65 537, 880 029, 880 000 and 1 000 000 return 0.

## The ceiling, refused early

The STREAMINFO field has its own failure mode, and it is worse than an
error: the upstream encoder writes the low 20 bits of whatever rate it is
handed, so a 2.4 MS/s full-band grab would have produced a FLAC that
*decodes* — labelled with a wrong rate. Every FLAC writer now refuses a
rate above `FLACMaxSampleRateHz` at construction with an error that names
1 048 575 (`TestNewFLACIQEncoderRejectsOverStreamInfoCeiling`), and the
capture route checks the same bound **before pinning the tuner**:

```go
// internal/api/capture.go (shape)
if format == siglab.FormatFLAC && outRate > baseband.FLACMaxSampleRateHz {
    s.writeError(w, http.StatusBadRequest, fmt.Sprintf(
        "siglab: flac cannot carry a %.3f MS/s stream (FLAC's STREAMINFO ceiling is %d Hz) — "+
            "request a narrowband slice (center_hz + bandwidth_hz) under that rate, or use cs16/wav for the full band",
        float64(outRate)/1e6, baseband.FLACMaxSampleRateHz))
    return
}
```

`outRate` is the slice's decimated rate for a narrowband request and the
full band otherwise, so a legitimate slice is never rejected on its
full-band footprint. `TestSiglabCaptureFLACFullBandOverCeilingIs400`
posts a full-band FLAC against a 2.4 MS/s tuner and wants a 400 naming
1048575. The one recorder that cannot 400 — the pre-combine
`diversity_capture` inside the SoapyRemote driver, which starts on its own
schedule — applies the bound the other way: `flacCaptureMaxRateHz` =
1 000 000, above which `diversity_capture_format: flac` logs
`diversity_capture_format flac not usable at this rate — falling back to
cs16` and records the cs16 twin, whose alignment invariant and every
downstream conclusion are container-independent.

## What the pins cover

The failing-first test for the route is the operator's request verbatim.
`TestSiglabCaptureFLACSliceAtOddRate` fakes an X310 at 6.25 MS/s centred
441.7 MHz and posts `{"seconds":60,"format":"flac","center_hz":442812500,
"bandwidth_hz":880000}`. It then asserts the staged capture's
`SampleRateHz` is **880 029** — the capped-L/M rate the operator saw, not
a rounded 880 000 — that its `CenterHz` is the requested 442 812 500, that
the downloaded bytes start with `fLaC`, and that
`baseband.ReadIQFLACSamples` decodes a non-empty slice at 880 029 Hz.
Against the old encoder the route returned a 500 from `encode flac frame`.
Two things about that test are deliberate. It pins the *odd* rate rather
than normalising it away, because the odd rate is what the DDC produces
and a future "fix" that rounds it would silently change every slice's
clock. And it pins the centre, which is the Part 7 bug — both failures
came from one request, and one test now covers both.

What this does not change: FLAC remains a 16-bit container, and
`hunt -survey-capture` deliberately stays f32. The practical rule is
unchanged from
[Analog Edge Part 10]({{ '/blog/tutorials/analog-edge-10-capture-discipline/' | relative_url }}):
a narrowband slice as FLAC is a good archive; a multi-MS/s full band is
cs16 or wav, and the route will now say so instead of letting the grab
run.

## Where this goes next

Two capture-tooling bugs are enough; the next part returns to the decode
pump. A dual-TETRA wideband rig reported `host overruns` at a "tiny"
200 kS/s, with no `decode can't keep up` WARN anywhere — and the profile
put 65 % of the pump inside a 30-bit block decoder that re-encoded all
16 384 codewords on every call.
[Part 9]({{ '/blog/solution-postmortem/issue-tracker-s2-09-sixteen-thousand-codewords/' | relative_url }})
is that search, its once-built codebook, and the FIR that was next in line.

## FAQ

**Why does GopherTrunk's FLAC capture fail with "unable to encode sample rate 880029"?**
It no longer does. The FLAC frame header's 4-bit code can carry a rate
verbatim only if it is a fixed audio rate, ≤ 65 535 Hz, ≤ 655 350 Hz in
tens of Hz, or ≤ 255 kHz in whole kHz; 880 029 fits none, and
`mewkiz/flac` never picks the "get from STREAMINFO" code itself.
`baseband.FLACFrameSampleRate` now stamps 0 for such rates, and the stream
decodes from STREAMINFO.

**What is the maximum sample rate a FLAC IQ recording can have?**
1 048 575 Hz — `FLACMaxSampleRateHz`, the 20-bit STREAMINFO field. Every
FLAC writer in GopherTrunk refuses a higher rate at construction, the
siglab capture route returns 400 for a full-band FLAC over it before
pinning the tuner, and `diversity_capture_format: flac` falls back to cs16
above `flacCaptureMaxRateHz` = 1 MS/s with a WARN.

**Would a clean 880000 Hz slice have worked?**
No. 880 000 is above 655 350, the largest rate the tens-of-Hz code carries,
and above 255 000, the whole-kHz code's limit, so it also needs code 0000.
`TestFLACFrameSampleRate` pins 880 000 and 1 000 000 to 0 alongside
880 029, and `TestFLACIQWriterEncodesFrameHeaderUnrepresentableRates`
round-trips all three.

**Why does the slice rate come out as 880029 and not 880000?**
The streaming down-converter's rational resampler caps its L/M, so an
880 kHz request from a 6.25 MS/s stream lands at 880 029 Hz; the route
takes `ddc.OutRateHz()` as the file's rate.
`TestSiglabCaptureFLACSliceAtOddRate` pins that exact value rather than
rounding it, because the odd rate is the slice's true sample clock.

## Series navigation

**Part 8 of 14** · ←
[Part 7: The Capture Carved at the Wrong Centre — A NaN That Fell Back Silently]({{ '/blog/solution-postmortem/issue-tracker-s2-07-capture-at-the-wrong-centre/' | relative_url }})
· Next →
[Part 9: Sixteen Thousand Codewords per Slot — The AACH Decoder Behind 'Overruns at 200 kS/s']({{ '/blog/solution-postmortem/issue-tracker-s2-09-sixteen-thousand-codewords/' | relative_url }})
