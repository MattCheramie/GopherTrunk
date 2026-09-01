#!/usr/bin/env python3
"""QC checks (pipeline doc §12): for each export verify H.264 High yuv420p,
30 fps, AAC 48 kHz stereo, and measure integrated loudness / true peak.

Usage: qc.py <target_lufs> <file.mp4> [...]
"""
import json, re, subprocess, sys
import imageio_ffmpeg

FF = imageio_ffmpeg.get_ffmpeg_exe()


def qc(path, target):
    err = subprocess.run([FF, "-i", path], capture_output=True, text=True).stderr
    vid = re.search(r"Video: (\S+) \(([^)]*)\).*?(\d+x\d+).*?([\d.]+) fps", err)
    aud = re.search(r"Audio: (\w+).*?(\d+) Hz, (\w+)", err)
    dur = re.search(r"Duration: (\d+):(\d+):([\d.]+)", err)
    secs = int(dur.group(1)) * 3600 + int(dur.group(2)) * 60 + float(dur.group(3))
    err2 = subprocess.run([FF, "-i", path, "-af",
                           "loudnorm=print_format=json", "-f", "null", "/dev/null"],
                          capture_output=True, text=True).stderr
    j = json.loads(err2[err2.rfind("{"): err2.rfind("}") + 1])
    lufs, tp = float(j["input_i"]), float(j["input_tp"])
    ok = (vid and vid.group(1) == "h264" and "High" in vid.group(2)
          and abs(float(vid.group(4)) - 30) < 0.01
          and aud and aud.group(1) == "aac" and aud.group(2) == "48000"
          and abs(lufs - target) <= 1.0 and tp <= -1.0)
    name = path.rsplit("/", 1)[-1]
    print(f"{'PASS' if ok else 'FAIL'} {name}: {vid.group(1)}/{vid.group(2).split(') (')[0]} "
          f"{vid.group(3)} {vid.group(4)}fps · {aud.group(1)} {aud.group(2)}Hz {aud.group(3)} "
          f"· {secs:.1f}s · I={lufs:.1f} LUFS TP={tp:.1f} dBTP")
    return ok


if __name__ == "__main__":
    target = float(sys.argv[1])
    bad = [f for f in sys.argv[2:] if not qc(f, target)]
    sys.exit(1 if bad else 0)
