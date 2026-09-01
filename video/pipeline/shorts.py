#!/usr/bin/env python3
"""Shorts clips (pipeline doc §11): cut the vertical at scripted [CLIP] bounds,
burn a 1-line hook/setup card over the first 2.2 s, loudnorm to -13 LUFS.

Usage: shorts.py <timelines_dir> <verticals_dir> <out_dir>
"""
import json, subprocess, sys, tempfile
from pathlib import Path

import imageio_ffmpeg
from assemble import loudnorm, probe_dur

FF = imageio_ffmpeg.get_ffmpeg_exe()
FONT = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"

HOOKS = {
    "GT-TR-01.01c1": "Why 30 radio channels\nsit silent",
    "GT-TR-01.01c2": "Radio channels that\nexist for one call",
    "GT-TR-01.02c1": "The radio channel\nthat never speaks",
    "GT-TR-01.02c2": "Joining a call you\nnever heard start",
    "GT-TR-01.03c1": "This radio channel\ndoesn't exist",
    "GT-TR-01.03c2": "Two numbers identify\nevery radio call",
    "GT-TR-01.04c1": "One message moves\n100 radios at once",
    "GT-TR-01.04c2": "Inside the message that\nstarts every call",
    "GT-TR-01.05c1": "The simplest way\nradios share the air",
    "GT-TR-01.05c2": "How 4 calls fit on\none frequency",
}


def main():
    tld, vd, od = Path(sys.argv[1]), Path(sys.argv[2]), Path(sys.argv[3])
    od.mkdir(parents=True, exist_ok=True)
    for tlf in sorted(tld.glob("GT-TR-01.0[1-5].timeline.json")):
        tl = json.loads(tlf.read_text())
        seg = tl["seg"]
        vsrc = vd / f"{seg}.vertical.mp4"
        for cname, b in tl.get("clips", {}).items():
            cid = f"{seg}{cname}"
            hook = HOOKS.get(cid, tl["title"])
            start, end = b["start"], min(b["end"] + 0.4, probe_dur(vsrc))
            with tempfile.NamedTemporaryFile("w", suffix=".txt", delete=False) as tf:
                tf.write(hook)
                hf = tf.name
            raw = od / f"{cid}.raw.mp4"
            draw = (f"drawtext=fontfile={FONT}:textfile={hf}:fontsize=54:"
                    f"fontcolor=white:line_spacing=14:box=1:boxcolor=0x0d1117@0.88:"
                    f"boxborderw=26:x=(w-text_w)/2:y=h*0.16:enable='lt(t,2.2)'")
            subprocess.run([FF, "-y", "-loglevel", "error",
                            "-ss", f"{start:.3f}", "-to", f"{end:.3f}", "-i", str(vsrc),
                            "-vf", draw, "-c:v", "libx264", "-preset", "medium", "-crf", "20",
                            "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "192k",
                            str(raw)], check=True)
            loudnorm(raw, od / f"{cid}.mp4", I=-13.0)
            raw.unlink(); Path(hf).unlink()
            print(f"{cid}: {end-start:.1f}s  hook={hook.replace(chr(10),' / ')}")


if __name__ == "__main__":
    main()
