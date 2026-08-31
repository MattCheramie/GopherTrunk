"""Assemble the pilot: per-piece audio mix (VO + music bed) → concat → 2-pass
loudnorm → chapters + full SRT + videos.yml stub + standalone segment exports.

Run after render_all.py. Outputs under video/_render/exports/.
"""
import json
import os
import re
import subprocess
import sys

import music
from common import FF, FPS, RENDER
from render_all import PIECES

EXP = os.path.join(RENDER, "exports")
SLUGS = {  # segment → (slug, chapter title = exact Field Guide term)
    "GT-RF-01.01": ("radio-wave", "Radio wave"),
    "GT-RF-01.02": ("frequency", "Frequency"),
    "GT-RF-01.03": ("modulation", "Modulation"),
    "GT-RF-01.04": ("bandwidth", "Bandwidth"),
    "GT-RF-01.05": ("decibel", "Decibel (dB)"),
    "GT-RF-01.06": ("signal-to-noise-ratio", "Signal-to-noise ratio (SNR)"),
}


def run(cmd, **kw):
    return subprocess.run(cmd, check=True, capture_output=True, text=True, **kw)


def probe_dur(path):
    r = subprocess.run([FF, "-i", path], capture_output=True, text=True)
    m = re.search(r"Duration: (\d+):(\d+):(\d+\.\d+)", r.stderr)
    h, mn, s = float(m.group(1)), float(m.group(2)), float(m.group(3))
    return h * 3600 + mn * 60 + s


def mix_piece(seg, i):
    d = os.path.join(RENDER, seg)
    vid = os.path.join(d, "video.mp4")
    narr = os.path.join(d, "narration.wav")
    dur = probe_dur(vid)
    bed = os.path.join(d, "bed.wav")
    music.generate(dur, bed, seed=i, gain_db=-30.0 if seg.endswith("endslate") else -34.0)
    out = os.path.join(d, "full.mp4")
    run([FF, "-y", "-i", vid, "-i", narr, "-i", bed, "-filter_complex",
         "[1:a]aresample=48000:resampler=soxr,aformat=channel_layouts=stereo,apad[vo];"
         "[vo][2:a]amix=inputs=2:duration=shortest:normalize=0[a]",
         "-map", "0:v", "-map", "[a]", "-c:v", "copy", "-c:a", "aac", "-b:a", "192k",
         "-shortest", out])
    return out, dur


def loudnorm2(src, out, target_i=-14.0):
    p1 = subprocess.run([FF, "-i", src, "-af",
                         f"loudnorm=I={target_i}:TP=-1.5:LRA=11:print_format=json",
                         "-f", "null", "-"], capture_output=True, text=True)
    blob = "{" + p1.stderr.rsplit("{", 1)[1]
    j = json.loads(blob[:blob.index("}") + 1])  # loudnorm JSON is flat; drop trailing ffmpeg chatter
    af = (f"loudnorm=I={target_i}:TP=-1.5:LRA=11:measured_I={j['input_i']}:"
          f"measured_TP={j['input_tp']}:measured_LRA={j['input_lra']}:"
          f"measured_thresh={j['input_thresh']}:offset={j['target_offset']}:linear=true")
    run([FF, "-y", "-i", src, "-map", "0:v", "-map", "0:a", "-c:v", "copy",
         "-af", af, "-c:a", "aac", "-b:a", "192k", out])
    return j


def srt_shift(path, offset, start_index):
    """Return (shifted srt text, next index)."""
    txt = open(path, encoding="utf-8").read().strip()
    if not txt:
        return "", start_index
    out, idx = [], start_index

    def ts(x):
        ms = int(round(x * 1000))
        return f"{ms//3600000:02d}:{ms//60000%60:02d}:{ms//1000%60:02d},{ms%1000:03d}"

    for block in txt.split("\n\n"):
        lines = block.splitlines()
        m = re.match(r"(\d+):(\d+):(\d+),(\d+) --> (\d+):(\d+):(\d+),(\d+)", lines[1])
        g = [int(x) for x in m.groups()]
        a = g[0] * 3600 + g[1] * 60 + g[2] + g[3] / 1000 + offset
        b = g[4] * 3600 + g[5] * 60 + g[6] + g[7] / 1000 + offset
        out.append(f"{idx}\n{ts(a)} --> {ts(b)}\n" + "\n".join(lines[2:]))
        idx += 1
    return "\n\n".join(out) + "\n\n", idx


def main():
    os.makedirs(EXP, exist_ok=True)
    os.makedirs(os.path.join(EXP, "segments-16x9"), exist_ok=True)
    pieces, t = [], 0.0
    chapters, yml_rows = [], []
    srt_all, srt_idx = [], 1
    for i, (seg, _scene) in enumerate(PIECES):
        full, dur = mix_piece(seg, i)
        vdur = probe_dur(full)
        pieces.append(full)
        if seg in SLUGS:
            slug, term = SLUGS[seg]
            chapters.append((t, term))
            yml_rows.append((slug, seg, term, int(round(t)), int(round(t + vdur))))
            # standalone segment export (loudnormed)
            loudnorm2(full, os.path.join(EXP, "segments-16x9", f"{seg}-{slug}.mp4"))
        srt_path = os.path.join(RENDER, seg, seg + ".srt")
        if os.path.exists(srt_path):
            s, srt_idx = srt_shift(srt_path, t, srt_idx)
            srt_all.append(s)
        t += vdur
    # concat
    lst = os.path.join(RENDER, "concat.txt")
    open(lst, "w").write("".join(f"file '{p}'\n" for p in pieces))
    raw = os.path.join(RENDER, "pilot_raw.mp4")
    run([FF, "-y", "-f", "concat", "-safe", "0", "-i", lst, "-c", "copy", raw])
    final = os.path.join(EXP, "GT-RF-01-radio-fundamentals-pilot.mp4")
    stats = loudnorm2(raw, final)
    # chapters (YouTube format) — 00:00 Intro is required
    def cts(x):
        x = int(x)
        return f"{x//60:02d}:{x%60:02d}" if x < 3600 else f"{x//3600}:{x//60%60:02d}:{x%60:02d}"
    chap_lines = ["00:00 Intro"] + [f"{cts(a)} {name}" for a, name in chapters]
    open(os.path.join(EXP, "chapters.txt"), "w").write("\n".join(chap_lines) + "\n")
    # videos.yml stub
    with open(os.path.join(EXP, "videos.yml.stub"), "w") as f:
        f.write("# paste into docs/_data/videos.yml once the pillar is uploaded\n"
                "pillars:\n  gt-rf-01:\n    youtube_id: \"REPLACE\"\n"
                "    title: \"Radio Fundamentals: How Signals Actually Work (Act I)\"\n"
                f"    upload_date: \"REPLACE\"\n    duration_s: {int(t)}\n\nsegments:\n")
        for slug, seg, term, a, b in yml_rows:
            f.write(f"  {slug}:\n    pillar: gt-rf-01\n    seg_id: \"{seg}\"\n"
                    f"    title: \"{term} in {max(1,round((b-a)/60))} minutes\"\n"
                    f"    start_s: {a}\n    end_s: {b}\n")
    open(os.path.join(EXP, "GT-RF-01-pilot.srt"), "w").write("".join(srt_all))
    print("final:", final, f"{t:.1f}s")
    print("loudnorm in:", stats["input_i"], "LUFS →", stats.get("output_i", "-14"))
    print("\n".join(chap_lines))


if __name__ == "__main__":
    main()
