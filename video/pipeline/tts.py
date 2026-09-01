#!/usr/bin/env python3
"""VO-first narration builder (pipeline doc §5–§6).

Parses a segment script (video/<PILLAR>/scripts/*.md), synthesizes each
sentence with Piper (en-us-ryan-high), measures durations, and emits:

  <out>/<seg>.timeline.json   — the single source of truth (audio/video/captions)
  <out>/<seg>.narration.wav   — 48 kHz mono narration with scripted gaps
  <out>/<seg>.srt             — one cue per VO line, wrapped ≤42 chars, ≤2 lines
  <out>/<seg>.transcript.md   — display-text transcript

Script grammar (§5): `**[V: name variant "arg"]**` visual lines; VO paragraphs;
`[CLIP cN start]` / `[CLIP cN end]` span marks; `{display|speech}` alternates.

Usage: tts.py <script.md> <voice.onnx> <outdir>
"""
import json, re, sys, wave
from pathlib import Path

import numpy as np

GAP_SENT = 0.12      # between sentences within a paragraph
GAP_PARA = 0.35      # between paragraphs / blocks
GAP_RECAP = 0.8      # extra beat before the recap card
TITLE_DUR = 2.0      # silent title card
ENDSLATE_DUR = 20.0  # pillar end slate (music only)
TAIL_RECAP = 2.5     # hold after last recap line
TAIL_PLAIN = 1.0

VRE = re.compile(r'^\*\*\[V:\s*([\w-]+)(?:\s+([\w-]+))?(?:\s+"([^"]*)")?\s*\]\*\*\s*(.*)$')
CLIP_RE = re.compile(r'\[CLIP\s+(c\d+)\s+(start|end)\]')
ALT_RE = re.compile(r'\{([^|{}]+)\|([^|{}]+)\}')


def parse(path: Path):
    seg, title = None, None
    blocks = []  # {visual, variant, arg, paras: [[(disp, speech, clipmarks)]]}
    clips = {}
    cur = None
    for raw in path.read_text().splitlines():
        line = re.sub(r'<!--.*?-->', '', raw).strip()
        if not line:
            continue
        m = re.match(r'^#\s+(\S+)\s+—\s+(.*)$', line)
        if m:
            seg, title = m.group(1), m.group(2).strip()
            continue
        if re.match(r'^(slug|also_slugs):', line):
            continue
        vm = VRE.match(line)
        if vm:
            cur = {"visual": vm.group(1), "variant": vm.group(2) or "",
                   "arg": vm.group(3) or "", "paras": []}
            blocks.append(cur)
            line = vm.group(4).strip()
            if not line or not CLIP_RE.search(line):
                if line:
                    raise SystemExit(f"unexpected trailing text on [V:] line: {line!r}")
                # clip marks handled below via empty remainder
        marks = [(c, w) for c, w in CLIP_RE.findall(line)]
        text = CLIP_RE.sub('', line).strip()
        if vm and not text:
            for c, w in marks:
                clips.setdefault(c, {})[w] = ('block', len(blocks) - 1)
            continue
        if cur is None:
            raise SystemExit(f"VO text before any [V:] block: {text!r}")
        if text:
            cur["paras"].append((text, marks))
        else:
            for c, w in marks:
                clips.setdefault(c, {})[w] = ('block', len(blocks) - 1)
    return seg, title, blocks, clips


def sentences(text):
    parts = re.split(r'(?<=[.!?…])\s+', text.strip())
    return [p for p in (s.strip() for s in parts) if p]


def disp(text):
    return ALT_RE.sub(lambda m: m.group(1), text)


def speech(text):
    t = ALT_RE.sub(lambda m: m.group(2), text)
    t = t.replace('—', ', ').replace('·', ', ').replace('→', ' to ')
    t = re.sub(r'["“”]', '', t)
    return t


def main():
    script, model, outdir = Path(sys.argv[1]), sys.argv[2], Path(sys.argv[3])
    outdir.mkdir(parents=True, exist_ok=True)
    seg, title, blocks, clipmarks = parse(script)

    from piper import PiperVoice, SynthesisConfig
    voice = PiperVoice.load(model)
    syn = SynthesisConfig(length_scale=1.04, volume=0.85)
    sr = voice.config.sample_rate

    events, pieces = [], []  # pieces: np arrays at sr
    cursor = 0.0
    clip_bounds = {}

    def add_silence(d):
        nonlocal cursor
        pieces.append(np.zeros(int(round(d * sr)), dtype=np.int16))
        cursor += d

    line_idx = 0
    for b, blk in enumerate(blocks):
        vis, var = blk["visual"], blk["variant"]
        if vis == "recap":
            add_silence(GAP_RECAP)
        if not blk["paras"]:
            dur = TITLE_DUR if vis == "title" else (ENDSLATE_DUR if vis == "endslate" else 3.0)
            events.append({"b": b, "t": round(cursor, 3), "dur": dur, "visual": vis,
                           "variant": var, "arg": blk["arg"]})
            add_silence(dur)
            continue
        first = True
        for text, marks in blk["paras"]:
            if not first or b > 0:
                add_silence(GAP_PARA if first else GAP_PARA)
            first = False
            para_start = cursor
            sents = sentences(text)
            for si, s in enumerate(sents):
                if si:
                    add_silence(GAP_SENT)
                wav_path = outdir / f"{seg}.line{line_idx:03d}.wav"
                line_idx += 1
                with wave.open(str(wav_path), "wb") as wf:
                    voice.synthesize_wav(speech(s), wf, syn_config=syn)
                with wave.open(str(wav_path), "rb") as wf:
                    n = wf.getnframes()
                    data = np.frombuffer(wf.readframes(n), dtype=np.int16)
                dur = n / sr
                events.append({"b": b, "t": round(cursor, 3), "dur": round(dur, 3),
                               "visual": vis, "variant": var, "arg": blk["arg"],
                               "text": disp(s)})
                pieces.append(data)
                cursor += dur
            for c, w in marks:
                if w == "start":
                    clip_bounds.setdefault(c, {})["start"] = para_start
                else:
                    clip_bounds.setdefault(c, {})["end"] = cursor
    # block-index clip marks (from [V:] lines)
    for c, ends in clipmarks.items():
        for w, (_, bi) in ends.items():
            evs = [e for e in events if e["b"] == bi]
            if evs:
                clip_bounds.setdefault(c, {})[w] = evs[0]["t"] if w == "start" else evs[-1]["t"] + evs[-1]["dur"]

    add_silence(TAIL_RECAP if blocks and blocks[-1]["visual"] == "recap" else TAIL_PLAIN)

    audio = np.concatenate(pieces) if pieces else np.zeros(0, dtype=np.int16)
    raw = outdir / f"{seg}.narration.{sr}.wav"
    with wave.open(str(raw), "wb") as wf:
        wf.setnchannels(1); wf.setsampwidth(2); wf.setframerate(sr)
        wf.writeframes(audio.tobytes())
    # resample to 48k via ffmpeg (soxr)
    import subprocess, imageio_ffmpeg
    ff = imageio_ffmpeg.get_ffmpeg_exe()
    nar = outdir / f"{seg}.narration.wav"
    subprocess.run([ff, "-y", "-loglevel", "error", "-i", str(raw),
                    "-ar", "48000", "-af", "aresample=resampler=soxr", str(nar)], check=True)
    raw.unlink()

    tl = {"seg": seg, "title": title, "fps": 30, "dur": round(cursor, 3),
          "clips": {c: {k: round(v, 3) for k, v in b.items()} for c, b in clip_bounds.items()},
          "events": events}
    (outdir / f"{seg}.timeline.json").write_text(json.dumps(tl, indent=1))

    # SRT
    def fmt(t):
        ms = int(round(t * 1000))
        return f"{ms//3600000:02d}:{ms//60000%60:02d}:{ms//1000%60:02d},{ms%1000:03d}"
    def wrap(text, width=42):
        words, lines, cur = text.split(), [], ""
        for w in words:
            if len((cur + " " + w).strip()) > width:
                lines.append(cur.strip()); cur = w
            else:
                cur += " " + w
        if cur.strip():
            lines.append(cur.strip())
        return lines
    srt, n = [], 1
    for e in events:
        if "text" not in e:
            continue
        lines = wrap(e["text"])
        groups = [lines[i:i+2] for i in range(0, len(lines), 2)]
        t0 = e["t"]
        per = e["dur"] / len(groups)
        for g in groups:
            srt.append(f"{n}\n{fmt(t0)} --> {fmt(t0+per)}\n" + "\n".join(g) + "\n")
            t0 += per; n += 1
    (outdir / f"{seg}.srt").write_text("\n".join(srt))

    # transcript
    lines = [f"# {seg} — {title}", ""]
    for blk in blocks:
        for text, _ in blk["paras"]:
            lines.append(disp(text)); lines.append("")
    (outdir / f"{seg}.transcript.md").write_text("\n".join(lines))
    print(f"{seg}: {cursor:.1f}s, {line_idx} lines, clips={tl['clips']}")


if __name__ == "__main__":
    main()
