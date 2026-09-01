#!/usr/bin/env python3
"""Segment renderer: script markdown → scratch-VO preview video + SRT captions.

What it does (per segment):
  1. Parses the script's [V:]-directed narration paragraphs.
  2. Synthesizes scratch narration per paragraph with espeak-ng (STAND-IN for
     the real recorded VO — same timings workflow, robotic voice).
  3. Renders 30 fps motion-graphic frames: 2 s title card → one scene per
     paragraph (custom scene from segment_plans.py, else a storyboard-slate
     animatic) → recap card on the final paragraph.
  4. Pipes frames to ffmpeg (libx264, yuv420p) with the concatenated VO,
     loudness-normalized to −14 LUFS, and writes an SRT timed to the VO.

Usage:
  python3 render.py ../GT-RF-01/scripts/GT-RF-01.02-radio-wave.md [--size 1280x720]
  python3 render.py --srt-only ../GT-RF-01/scripts/*.md
Output: ../GT-RF-01/exports/<seg>.mp4, ../GT-RF-01/captions/<seg>.srt,
        ../GT-RF-01/audio/vo-scratch/<seg>.wav
"""
import os, re, subprocess, sys, wave, contextlib

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "..", "_brand"))
import brand as B
import gen_cards
import scenes
from segment_plans import PLANS

FPS = 30
TITLE_S = 2.0
PAD_S = 0.45
RECAP_HOLD_S = 3.0
SR = 22050


def ffmpeg_exe():
    import imageio_ffmpeg
    return imageio_ffmpeg.get_ffmpeg_exe()


def parse_script(path):
    """Returns (title, seg_id, slug, [(visual, narration), ...])."""
    text = open(path).read()
    m = re.match(r"# (GT-[A-Z]+-\d+\.\d+) — (.+)", text)
    seg_id, title = (m.group(1), m.group(2).strip()) if m else ("GT-XX-00.00", "Untitled")
    sm = re.search(r"slug:\s*([a-z0-9-]+)", text)
    slug = sm.group(1) if sm else "unknown"
    body = text.split("## Clip picks")[0]
    blocks, cur_v, cur_n = [], None, []

    def flush():
        if cur_v is not None and cur_n:
            narr = " ".join(cur_n)
            narr = re.sub(r"\[CLIP[^\]]*\]", "", narr)
            narr = re.sub(r"\*\*(.+?)\*\*", r"\1", narr)
            narr = re.sub(r"\*(.+?)\*", r"\1", narr)
            narr = re.sub(r"\s+", " ", narr).strip()
            if narr:
                blocks.append((cur_v, narr))

    for line in body.splitlines():
        ls = line.strip()
        vm = re.match(r"\*?\*?\[V:\s*(.+?)\]\*?\*?", ls)
        if vm:
            flush()
            cur_v, cur_n = vm.group(1), []
        elif ls and not ls.startswith("#") and not ls.startswith("slug:") and cur_v is not None:
            cur_n.append(ls)
        elif not ls and cur_v is not None and cur_n:
            flush()
            cur_v, cur_n = None, []
    flush()
    return title, seg_id, slug, blocks


def tts(text, out_wav):
    subprocess.run(["espeak-ng", "-v", "en-us+m3", "-s", "158", "-p", "38",
                    "-a", "185", "-w", out_wav, text], check=True)
    with contextlib.closing(wave.open(out_wav)) as wf:
        return wf.getnframes() / wf.getframerate()


def concat_wavs(paths_and_pads, out_wav):
    """paths_and_pads: list of (wav_path_or_None, pad_seconds_after)."""
    frames = b""
    for p, pad in paths_and_pads:
        if p:
            with contextlib.closing(wave.open(p)) as wf:
                frames += wf.readframes(wf.getnframes())
        frames += b"\x00\x00" * int(pad * SR)
    with contextlib.closing(wave.open(out_wav, "w")) as w:
        w.setnchannels(1); w.setsampwidth(2); w.setframerate(SR)
        w.writeframes(frames)


def split_cues(narr, start, dur):
    """Sentence-proportional SRT cues within [start, start+dur]."""
    sents = re.split(r"(?<=[.!?…]) +", narr)
    chunks = []
    for s in sents:
        while len(s) > 84:
            cut = s.rfind(" ", 0, 84)
            cut = cut if cut > 30 else 84
            chunks.append(s[:cut]); s = s[cut:].strip()
        if s:
            chunks.append(s)
    total = sum(len(c) for c in chunks) or 1
    cues, t = [], start
    for c in chunks:
        d = dur * len(c) / total
        cues.append((t, t + d, c))
        t += d
    return cues


def fmt_ts(t):
    ms = int(round(t * 1000))
    return f"{ms//3600000:02d}:{ms%3600000//60000:02d}:{ms%60000//1000:02d},{ms%1000:03d}"


def write_srt(cues, out):
    with open(out, "w") as f:
        for i, (a, b, txt) in enumerate(cues, 1):
            f.write(f"{i}\n{fmt_ts(a)} --> {fmt_ts(b)}\n{txt}\n\n")


def scene_for(slug, idx, visual, is_last, plan):
    if plan and idx < len(plan) and plan[idx]:
        name, kwargs = plan[idx]
        return lambda t, dur, size: scenes.SCENES[name](t, dur, size, **kwargs)
    if is_last:
        return None  # recap card image handled by caller
    return lambda t, dur, size: scenes.storyboard_scene(t, dur, size, direction=visual)


def render_segment(script_path, size=(1280, 720), srt_only=False):
    from PIL import Image
    title, seg_id, slug, blocks = parse_script(script_path)
    if not blocks:
        print(f"!! no narration blocks parsed in {script_path}"); return
    seg_dir = os.path.dirname(os.path.dirname(os.path.abspath(script_path)))
    for sub in ("exports", "captions", "audio/vo-scratch", "graphics/_tmp"):
        os.makedirs(os.path.join(seg_dir, sub), exist_ok=True)
    base = f"{seg_id}-{slug}"
    plan = PLANS.get(slug)

    # --- TTS per paragraph
    tmp = os.path.join(seg_dir, "graphics/_tmp")
    durs, wavs = [], []
    for i, (_v, narr) in enumerate(blocks):
        wv = os.path.join(tmp, f"{base}-p{i:02d}.wav")
        durs.append(tts(narr, wv)); wavs.append(wv)

    # --- timeline + captions
    t = TITLE_S
    cues, para_spans = [], []
    for i, ((_v, narr), d) in enumerate(zip(blocks, durs)):
        cues.extend(split_cues(narr, t, d))
        para_spans.append((t, d))
        t += d + PAD_S
    total_s = t + RECAP_HOLD_S
    write_srt(cues, os.path.join(seg_dir, "captions", f"{base}.srt"))

    vo = os.path.join(seg_dir, "audio/vo-scratch", f"{base}.wav")
    concat_wavs([(None, TITLE_S)] + [(w, PAD_S) for w in wavs] + [(None, RECAP_HOLD_S)], vo)
    if srt_only:
        print(f"{base}: SRT + VO written ({total_s:.0f} s)"); return

    # --- cards
    tcard_p = os.path.join(tmp, f"{base}-title.png")
    gen_cards.title_card(title, seg_id, tcard_p, size=size)
    rcard_p = os.path.join(tmp, f"{base}-recap.png")
    bullets = (plan or {}).get("recap") if isinstance(plan, dict) else None
    if isinstance(plan, dict):
        plan_list = plan.get("scenes")
    else:
        plan_list = plan
    gen_cards.recap_card(title, bullets or ["See the full write-up",
                                            "gophertrunk.org/reference/" + slug],
                         seg_id, rcard_p, size=size)
    tcard = Image.open(tcard_p).convert("RGB")
    rcard = Image.open(rcard_p).convert("RGB")

    # --- ffmpeg pipe
    suffix = "-9x16" if size[1] > size[0] else ""
    out_mp4 = os.path.join(seg_dir, "exports", f"{base}{suffix}.mp4")
    cmd = [ffmpeg_exe(), "-y", "-f", "rawvideo", "-pix_fmt", "rgb24",
           "-s", f"{size[0]}x{size[1]}", "-r", str(FPS), "-i", "-",
           "-i", vo, "-c:v", "libx264", "-preset", "fast", "-crf", "20",
           "-pix_fmt", "yuv420p", "-af", "loudnorm=I=-14:TP=-1.5",
           "-c:a", "aac", "-b:a", "192k", "-shortest", out_mp4]
    proc = subprocess.Popen(cmd, stdin=subprocess.PIPE,
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    nframes = int(total_s * FPS)
    last_i = len(blocks) - 1
    for fi in range(nframes):
        ft = fi / FPS
        if ft < TITLE_S:
            img = tcard
        else:
            img = None
            for i, (start, d) in enumerate(para_spans):
                if ft < start + d + PAD_S:
                    if i == last_i:
                        img = rcard
                    else:
                        fn = scene_for(slug, i, blocks[i][0], False, plan_list)
                        img = fn(ft - start, d, size)
                    break
            if img is None:
                img = rcard
        proc.stdin.write(img.tobytes())
    proc.stdin.close(); proc.wait()
    if proc.returncode != 0:
        raise RuntimeError("ffmpeg failed")
    print(f"{base}: {out_mp4} ({total_s:.0f} s, {nframes} frames)")


if __name__ == "__main__":
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    srt_only = "--srt-only" in sys.argv
    size = (1280, 720)
    for a in sys.argv[1:]:
        if a.startswith("--size"):
            size = tuple(int(x) for x in a.split("=")[1].split("x"))
    for p in args:
        render_segment(p, size=size, srt_only=srt_only)
