"""Synthesize narration for one segment script (VO-first workflow).

Outputs per segment under video/_render/<SEG>/:
  narration.wav      — full VO track, 22050 mono PCM16 (gaps included; title-card lead-in)
  timeline.json      — events with absolute t0/t1 per visual, per line, clip spans
  <SEG>.srt          — captions from the same timing
Usage: python3 tts.py ../GT-RF-01/scripts/GT-RF-01.05-decibel.md [--lead 2.0] [--tail 3.0]
"""
import io
import json
import os
import re
import sys
import wave

import numpy as np
from piper import PiperVoice
from piper.config import SynthesisConfig

from common import PIPER_MODEL, RENDER
from script_parse import parse_script, split_sentences

RATE = 22050
GAP_SENT = 0.22
GAP_PARA = 0.45
GAP_RECAP = 0.8   # extra beat before recap
TITLE_S = 2.0

# pronunciation dictionary: applied to TTS input only (captions keep original text)
PRON = [
    (r"\bAM\b", "A-M"), (r"\bIQ\b", "I-Q"), (r"\bUHF\b", "U-H-F"),
    (r"\bFEC\b", "F-E-C"), (r"\bEVM\b", "E-V-M"), (r"\bQAM\b", "kwam"),
    (r"\bLED\b", "L-E-D"), (r"\bd-B-m\b", "dee bee em"),
    (r"\bd-B-F-S\b", "dee bee eff ess"), (r"\bGopherTrunk\b", "Gopher Trunk"),
]

_voice = None


def voice():
    global _voice
    if _voice is None:
        _voice = PiperVoice.load(PIPER_MODEL)
    return _voice


def synth(text):
    for pat, rep in PRON:
        text = re.sub(pat, rep, text)
    buf = io.BytesIO()
    with wave.open(buf, "wb") as w:
        voice().synthesize_wav(text, w, SynthesisConfig(length_scale=1.12))
    buf.seek(0)
    with wave.open(buf, "rb") as w:
        assert w.getframerate() == RATE and w.getnchannels() == 1
        data = np.frombuffer(w.readframes(w.getnframes()), dtype=np.int16)
    # trim leading/trailing near-silence to keep gaps deterministic
    amp = np.abs(data.astype(np.int32))
    idx = np.where(amp > 120)[0]
    if len(idx):
        lo = max(0, idx[0] - int(0.05 * RATE))
        hi = min(len(data), idx[-1] + int(0.09 * RATE))
        data = data[lo:hi]
    return data


def srt_ts(t):
    ms = int(round(t * 1000))
    return f"{ms//3600000:02d}:{ms//60000%60:02d}:{ms//1000%60:02d},{ms%1000:03d}"


def wrap_caption(text, width=42):
    words, lines, cur = text.split(), [], ""
    for w in words:
        if len(cur) + len(w) + 1 > width and cur:
            lines.append(cur)
            cur = w
        else:
            cur = (cur + " " + w).strip()
    lines.append(cur)
    if len(lines) > 2:  # rebalance into 2 lines
        mid = len(text) // 2
        sp = min(range(len(text)), key=lambda i: abs(i - mid) if text[i] == " " else 10**9)
        lines = [text[:sp], text[sp + 1:]]
    return "\n".join(lines)


def build(script_path, lead=TITLE_S, tail=3.0):
    parsed = parse_script(script_path)
    seg = parsed["meta"]["title"].split("—")[0].strip().replace(" ", "")
    outdir = os.path.join(RENDER, seg)
    os.makedirs(outdir, exist_ok=True)

    t = 0.0
    chunks = []          # (numpy audio or float silence-seconds)
    events_out = []
    clips = {}
    lines_flat = []

    def add_silence(sec):
        nonlocal t
        chunks.append(float(sec))
        t += sec

    for ei, ev in enumerate(parsed["events"]):
        ev_out = {"visual": ev["visual"], "args": ev["args"], "t0": None, "lines": []}
        if ev["visual"] == "title":
            ev_out["t0"] = t
            add_silence(lead)
        elif ei == 0:
            ev_out["t0"] = t
            add_silence(lead)  # non-title first event still gets a short lead-in
        else:
            if ev["visual"].startswith("recap") and t > 0:
                add_silence(GAP_RECAP)
            ev_out["t0"] = t
        starts_here = {(c, w, p) for (c, w, p) in ev["clip_marks"]}
        for pi, para in enumerate(ev["paras"]):
            for (cid, which, p) in ev["clip_marks"]:
                if which == "start" and p == pi:
                    clips.setdefault(cid, {})["t0"] = t
            for si, sent in enumerate(split_sentences(para)):
                audio = synth(sent)
                dur = len(audio) / RATE
                line = {"text": sent, "t0": round(t, 3), "t1": round(t + dur, 3)}
                ev_out["lines"].append(line)
                lines_flat.append(line)
                chunks.append(audio)
                t += dur
                add_silence(GAP_SENT)
            for (cid, which, p) in ev["clip_marks"]:
                if which == "end" and p == pi:
                    clips.setdefault(cid, {})["t1"] = t
            add_silence(GAP_PARA - GAP_SENT)
        events_out.append(ev_out)
    add_silence(tail)
    total = t
    for i, ev in enumerate(events_out):
        ev["t1"] = round(events_out[i + 1]["t0"] if i + 1 < len(events_out) else total, 3)
        ev["t0"] = round(ev["t0"], 3)

    # bounce audio
    pcm = np.concatenate([
        (np.zeros(int(round(c * RATE)), dtype=np.int16) if isinstance(c, float) else c)
        for c in chunks])
    with wave.open(os.path.join(outdir, "narration.wav"), "wb") as w:
        w.setnchannels(1); w.setsampwidth(2); w.setframerate(RATE)
        w.writeframes(pcm.tobytes())

    timeline = {"seg": seg, "meta": parsed["meta"], "fps": 30,
                "duration": round(total, 3), "events": events_out,
                "clips": {k: {kk: round(vv, 3) for kk, vv in v.items()}
                          for k, v in clips.items()}}
    with open(os.path.join(outdir, "timeline.json"), "w") as f:
        json.dump(timeline, f, indent=1)

    with open(os.path.join(outdir, seg + ".srt"), "w") as f:
        for i, ln in enumerate(lines_flat, 1):
            f.write(f"{i}\n{srt_ts(ln['t0'])} --> {srt_ts(min(ln['t1'] + 0.3, total))}\n"
                    f"{wrap_caption(ln['text'])}\n\n")
    print(f"{seg}: {total:.1f}s, {len(lines_flat)} lines, clips={timeline['clips']}")
    return timeline


if __name__ == "__main__":
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    kw = {}
    for a in sys.argv[1:]:
        if a.startswith("--lead"):
            kw["lead"] = float(a.split("=")[1])
        if a.startswith("--tail"):
            kw["tail"] = float(a.split("=")[1])
    for p in args:
        build(p, **kw)
