#!/usr/bin/env python3
"""Assembly (pipeline doc §9): mix music per piece, concat, loudness-normalize,
chapters, videos.yml.stub, pillar SRT.

Also exposes helpers reused by shorts.py.

Usage:
  assemble.py mix <piece.mp4> <bed.wav> <out.mp4>        # amix bed under VO, video copied
  assemble.py loudnorm <in.mp4> <out.mp4> <I>            # two-pass loudnorm, video copied
  assemble.py pillar <renderdir> <outdir> <pillardir>    # concat+loudnorm+chapters+yml+srt
"""
import json, re, subprocess, sys
from pathlib import Path

import imageio_ffmpeg

FF = imageio_ffmpeg.get_ffmpeg_exe()
PIECES = ["GT-TR-01.00", "GT-TR-01.00b",
          "GT-TR-01.01", "GT-TR-01.tr-12",
          "GT-TR-01.02", "GT-TR-01.tr-23",
          "GT-TR-01.03", "GT-TR-01.tr-34",
          "GT-TR-01.04", "GT-TR-01.tr-45",
          "GT-TR-01.05", "GT-TR-01.99"]
SEGMENTS = {"GT-TR-01.01": "Trunked radio", "GT-TR-01.02": "Control channel",
            "GT-TR-01.03": "Talkgroup", "GT-TR-01.04": "Channel grant",
            "GT-TR-01.05": "FDMA & TDMA"}
SLUGS = {"GT-TR-01.01": "trunked-radio", "GT-TR-01.02": "control-channel",
         "GT-TR-01.03": "talkgroup", "GT-TR-01.04": "channel-grant",
         "GT-TR-01.05": "fdma"}


def run(*cmd, capture=False):
    r = subprocess.run(cmd, capture_output=True, text=True)
    if r.returncode:
        raise SystemExit(f"cmd failed: {' '.join(map(str, cmd))}\n{r.stderr[-2000:]}")
    return r.stderr if capture else None


def probe_dur(path):
    err = subprocess.run([FF, "-i", str(path)], capture_output=True, text=True).stderr
    m = re.search(r"Duration: (\d+):(\d+):([\d.]+)", err)
    return int(m.group(1)) * 3600 + int(m.group(2)) * 60 + float(m.group(3))


def mix(piece, bed, out):
    """Duck-free low bed under narration; bed trimmed to piece, 3 s fade-out."""
    dur = probe_dur(piece)
    fade = max(0.0, dur - 3.0)
    run(FF, "-y", "-loglevel", "error", "-i", str(piece),
        "-stream_loop", "-1", "-i", str(bed),
        "-filter_complex",
        f"[1:a]atrim=0:{dur:.3f},afade=t=out:st={fade:.3f}:d=3,volume=1.0[bed];"
        f"[0:a][bed]amix=inputs=2:duration=first:normalize=0[a]",
        "-map", "0:v", "-map", "[a]", "-c:v", "copy",
        "-c:a", "aac", "-b:a", "192k", "-ar", "48000", str(out))


def loudnorm(inp, out, I=-14.0, tp=-1.5):
    """Two-pass loudnorm on audio only; video stream copied."""
    err = run(FF, "-y", "-i", str(inp),
              "-af", f"loudnorm=I={I}:TP={tp}:LRA=11:print_format=json",
              "-f", "null", "/dev/null", capture=True)
    m = re.search(r"\{[^{}]*\}", err[err.rfind("input_i") - 200:] if "input_i" in err else err, re.S)
    j = json.loads(err[err.rfind("{"): err.rfind("}") + 1])
    run(FF, "-y", "-loglevel", "error", "-i", str(inp),
        "-af", (f"loudnorm=I={I}:TP={tp}:LRA=11:measured_I={j['input_i']}:"
                f"measured_TP={j['input_tp']}:measured_LRA={j['input_lra']}:"
                f"measured_thresh={j['input_thresh']}:offset={j['target_offset']}:"
                f"linear=true,alimiter=limit={10**((tp-0.6)/20):.4f}:level=false"),
        "-c:v", "copy", "-c:a", "aac", "-b:a", "192k", "-ar", "48000", str(out))


def pillar(renderdir, outdir, tldir):
    renderdir, outdir, tldir = Path(renderdir), Path(outdir), Path(tldir)
    outdir.mkdir(parents=True, exist_ok=True)
    files = [renderdir / f"{p}.mix.mp4" for p in PIECES]
    for f in files:
        if not f.exists():
            raise SystemExit(f"missing piece {f}")
    lst = renderdir / "concat.txt"
    lst.write_text("".join(f"file '{f.resolve()}'\n" for f in files))
    raw = renderdir / "pillar.raw.mp4"
    run(FF, "-y", "-loglevel", "error", "-f", "concat", "-safe", "0",
        "-i", str(lst), "-c", "copy", str(raw))
    loudnorm(raw, outdir / "GT-TR-01-pillar.mp4", I=-14.0)

    # chapters + yml stub + pillar SRT
    t = 0.0
    chapters, yml, srt_all, cue = [], [], [], 1
    for p in PIECES:
        dur = probe_dur(renderdir / f"{p}.mix.mp4")
        if p in SEGMENTS:
            mm, ss = divmod(int(round(t)), 60)
            chapters.append(f"{mm:02d}:{ss:02d} {SEGMENTS[p]}")
            yml.append(f"  - slug: {SLUGS[p]}\n    seg: {p}\n"
                       f"    start_s: {t:.1f}\n    end_s: {t + dur:.1f}")
        srt = tldir / f"{p}.srt"
        if srt.exists():
            for block in srt.read_text().strip().split("\n\n"):
                lines = block.split("\n")
                if len(lines) < 3:
                    continue
                m = re.match(r"(\d+):(\d+):([\d,]+) --> (\d+):(\d+):([\d,]+)", lines[1])
                def shift(h, mn, s):
                    sec = int(h) * 3600 + int(mn) * 60 + float(s.replace(",", ".")) + t
                    ms = int(round(sec * 1000))
                    return f"{ms//3600000:02d}:{ms//60000%60:02d}:{ms//1000%60:02d},{ms%1000:03d}"
                srt_all.append(f"{cue}\n{shift(m.group(1),m.group(2),m.group(3))} --> "
                               f"{shift(m.group(4),m.group(5),m.group(6))}\n" + "\n".join(lines[2:]) + "\n")
                cue += 1
        t += dur
    chap = "00:00 Intro\n" + "\n".join(chapters) + "\n"
    (outdir / "GT-TR-01-chapters.txt").write_text(chap)
    (outdir / "GT-TR-01-pillar.srt").write_text("\n".join(srt_all))
    stub = "# videos.yml stub — GT-TR-01 pillar chapter offsets\nvideos:\n" + "\n".join(yml) + "\n"
    Path(tldir / "../videos.yml.stub").resolve().write_text(stub)
    print(chap)
    print(f"pillar: {probe_dur(outdir / 'GT-TR-01-pillar.mp4'):.1f}s")


if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "mix":
        mix(*sys.argv[2:5])
    elif cmd == "loudnorm":
        loudnorm(sys.argv[2], sys.argv[3], float(sys.argv[4]))
    elif cmd == "pillar":
        pillar(*sys.argv[2:5])
