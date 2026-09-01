"""Companion package: per-video transcript .md + per-video graphics folder + docs,
zipped for download. One transcript and one folder per DELIVERED video file, named
exactly like the video (basename).

Layout (video/_render/companion/):
  gophertrunk-video-pipeline.md   GT-RF-01-scope-and-next.md   README.md
  pilot/<name>.md + <name>/       segments-16x9/...   segments-9x16/...   clips/...
"""
import json
import os
import shutil
import subprocess

from playwright.sync_api import sync_playwright

from assemble import SLUGS, probe_dur
from common import CHROME, CHROME_ARGS, RENDER, VIDEO
from render_all import PIECES

COMP = os.path.join(RENDER, "companion")
G = os.path.join(VIDEO, "GT-RF-01", "graphics")
EXP = os.path.join(RENDER, "exports")
VOFF = 2.0  # vertical/clip title-card offset

PIECE_LABEL = {"GT-RF-01.P0-coldopen": "Cold open", "GT-RF-01.P1-intro": "Course intro",
               "GT-RF-01.T1": "Transition", "GT-RF-01.T2": "Transition",
               "GT-RF-01.T3": "Transition", "GT-RF-01.T4": "Transition",
               "GT-RF-01.T5": "Transition", "GT-RF-01.P8-outro": "Outro",
               "GT-RF-01.P9-endslate": "End slate (music only)"}


def ts(t):
    t = max(0, int(round(t)))
    return f"{t//60}:{t%60:02d}"


def tl_of(seg):
    return json.load(open(os.path.join(RENDER, seg, "timeline.json")))


def beats_md(tl, shift=0.0, span=None, drop_title=False):
    """Markdown transcript body from a timeline: one block per visual beat."""
    out = []
    for ev in tl["events"]:
        name = ev["visual"].split("—")[0].split("|")[0].strip()
        if drop_title and name == "title":
            continue
        lines = [l for l in ev.get("lines", [])
                 if span is None or (l["t0"] >= span[0] - 0.05 and l["t0"] < span[1])]
        if not lines and (span is not None or name not in ("title", "endslate")):
            continue
        t0 = (lines[0]["t0"] if lines else ev["t0"]) + shift
        if span:
            t0 = (lines[0]["t0"] if lines else ev["t0"]) - span[0]
        head = {"title": "Title card", "recap": "Recap card"}.get(name, name)
        out.append(f"**[{ts(max(0, t0))}] {head}**")
        if lines:
            out.append(" ".join(l["text"] for l in lines))
        out.append("")
    return "\n".join(out)


def write_md(path, title, meta, body):
    with open(path, "w", encoding="utf-8") as f:
        f.write(f"# {title}\n\n_{meta}_\n\n{body}")


def seg_title(seg):
    return SLUGS[seg][1] if seg in SLUGS else PIECE_LABEL.get(seg, seg)


def main():
    shutil.rmtree(COMP, ignore_errors=True)
    for d in ("pilot", "segments-16x9", "segments-9x16", "clips"):
        os.makedirs(os.path.join(COMP, d))

    durs = {seg: probe_dur(os.path.join(RENDER, seg, "full.mp4")) for seg, _ in PIECES}

    # ---------- stills (one Chromium session) ----------
    stills = {}  # seg -> [(label, path)]
    os.makedirs(os.path.join(COMP, "_stills"))
    with sync_playwright() as p:
        b = p.chromium.launch(executable_path=CHROME, args=CHROME_ARGS)
        pg = b.new_page(viewport={"width": 1920, "height": 1080}, device_scale_factor=0.5)
        for seg, scene in PIECES:
            tl = tl_of(seg)
            pg.add_init_script("window.TIMELINE = " + json.dumps(tl))
            pg.goto("file://" + os.path.join(G, scene))
            pg.wait_for_function("window.ready === true && document.fonts.status === 'loaded'")
            stills[seg] = []
            for i, ev in enumerate(tl["events"]):
                name = ev["visual"].split("—")[0].split("|")[0].strip()
                mid = (ev["t0"] + ev["t1"]) / 2
                pg.evaluate(f"window.seek({mid})")
                sp = os.path.join(COMP, "_stills", f"{seg}-{i:02d}-{name}.jpg")
                pg.screenshot(path=sp, type="jpeg", quality=82)
                stills[seg].append((name, mid, sp))
        # vertical stills for the six segments
        vpg = b.new_page(viewport={"width": 1080, "height": 1920}, device_scale_factor=0.5)
        vstills = {}
        for seg in SLUGS:
            tl = tl_of(seg)
            from vertical_all import HOOKS
            vcfg = {"scene": "file://" + os.path.join(G, seg + ".html"),
                    "term": SLUGS[seg][1], "hook": HOOKS[seg]}
            vpg.add_init_script("window.TIMELINE = " + json.dumps(tl) +
                                "; window.VCFG = " + json.dumps(vcfg))
            vpg.goto("file://" + os.path.join(os.path.dirname(__file__), "vertical.html"))
            vpg.wait_for_function("window.ready === true")
            vstills[seg] = []
            dur = tl["duration"] - VOFF
            for label, t in [("hook", 0.5), ("early", dur * 0.25), ("mid", dur * 0.55),
                             ("late", dur * 0.8), ("endslate", dur - 0.8)]:
                vpg.evaluate(f"window.seek({t})")
                sp = os.path.join(COMP, "_stills", f"{seg}-vert-{label}.jpg")
                vpg.screenshot(path=sp, type="jpeg", quality=82)
                vstills[seg].append((label, t, sp))
        b.close()

    made = []  # (video basename, md path, folder path)

    def folder_for(base, subdir):
        d = os.path.join(COMP, subdir, base)
        os.makedirs(d, exist_ok=True)
        return d

    # ---------- 16:9 segments ----------
    for seg, (slug, term) in SLUGS.items():
        base = f"{seg}-{slug}"
        tl = tl_of(seg)
        md = os.path.join(COMP, "segments-16x9", base + ".md")
        write_md(md, f"{term} — transcript ({base}.mp4)",
                 f"GT-RF-01 segment · {ts(tl['duration'])} · 1920×1080 · timestamps match the video",
                 beats_md(tl))
        fd = folder_for(base, "segments-16x9")
        shutil.copy(os.path.join(G, seg + ".html"), fd)
        for name, mid, sp in stills[seg]:
            shutil.copy(sp, os.path.join(fd, f"{ts(mid).replace(':', 'm')}s-{name}.jpg"))
        thumb = os.path.join(VIDEO, "brand", "thumbnails", f"{base}.png")
        if os.path.exists(thumb):
            shutil.copy(thumb, fd)
        made.append((base + ".mp4", md, fd))

    # ---------- verticals ----------
    for seg, (slug, term) in SLUGS.items():
        base = f"{seg}-{slug}-vertical"
        tl = tl_of(seg)
        md = os.path.join(COMP, "segments-9x16", base + ".md")
        write_md(md, f"{term} — transcript ({base}.mp4)",
                 f"9:16 re-edit · {ts(tl['duration'] - VOFF)} · 1080×1920 · title card replaced by a 2 s burned hook",
                 beats_md(tl, shift=-VOFF, drop_title=True))
        fd = folder_for(base, "segments-9x16")
        for label, t, sp in vstills[seg]:
            shutil.copy(sp, os.path.join(fd, f"{label}.jpg"))
        made.append((base + ".mp4", md, fd))

    # ---------- clips ----------
    for seg, (slug, term) in SLUGS.items():
        tl = tl_of(seg)
        for cid, span in tl.get("clips", {}).items():
            base = f"{seg}{cid}"
            md = os.path.join(COMP, "clips", base + ".md")
            write_md(md, f"{term} — clip {cid} transcript ({base}.mp4)",
                     f"Shorts clip · {ts(span['t1'] - span['t0'])} · cut from the vertical at [{ts(span['t0'] - VOFF)}–{ts(span['t1'] - VOFF)}]",
                     beats_md(tl, span=(span["t0"], span["t1"])))
            fd = folder_for(base, "clips")
            for name, mid, sp in stills[seg]:
                if span["t0"] - 2 <= mid <= span["t1"] + 2:
                    shutil.copy(sp, os.path.join(fd, f"{ts(mid).replace(':', 'm')}s-{name}.jpg"))
            made.append((base + ".mp4", md, fd))

    # ---------- pilot + parts ----------
    def pilot_md(pieces, base, title):
        body, off = [], 0.0
        for seg, _ in pieces:
            tl = tl_of(seg)
            label = seg_title(seg)
            body.append(f"\n## [{ts(off)}] {label}\n")
            if seg != "GT-RF-01.P9-endslate":
                body.append(beats_md(tl, shift=off))
            off += durs[seg]
        md = os.path.join(COMP, "pilot", base + ".md")
        write_md(md, title, f"{ts(off)} · 1920×1080 · chapter timestamps match the video",
                 "\n".join(body))
        fd = folder_for(base, "pilot")
        for seg, _ in pieces:
            name, mid, sp = stills[seg][min(2, len(stills[seg]) - 1)]
            shutil.copy(sp, os.path.join(fd, f"{seg}-{name}.jpg"))
        thumb = os.path.join(VIDEO, "brand", "thumbnails", "GT-RF-01-pillar.png")
        if os.path.exists(thumb):
            shutil.copy(thumb, fd)
        made.append((base + ".mp4", md, fd))

    pilot_md(PIECES, "GT-RF-01-radio-fundamentals-pilot",
             "Radio Fundamentals — pilot transcript")
    pilot_md(PIECES[:5], "GT-RF-01-pilot-part1", "Pilot part 1 transcript (cold open → Frequency)")
    pilot_md(PIECES[5:10], "GT-RF-01-pilot-part2", "Pilot part 2 transcript (Modulation → Bandwidth)")
    pilot_md(PIECES[10:], "GT-RF-01-pilot-part3", "Pilot part 3 transcript (Decibels → end slate)")

    # ---------- docs ----------
    shutil.copy(os.path.join(VIDEO, "gophertrunk-video-pipeline.md"), COMP)
    shutil.copy(os.path.join(VIDEO, "GT-RF-01", "GT-RF-01-scope-and-next.md"), COMP)
    with open(os.path.join(COMP, "README.md"), "w") as f:
        f.write("# GT-RF-01 companion package\n\nFor every delivered video file there is a"
                " transcript `.md` and a graphics folder with the SAME basename as the"
                " video. Folders hold that video's beat stills (JPEG), the animation"
                " scene source where applicable, and the thumbnail.\n\nAlso included:"
                " `gophertrunk-video-pipeline.md` (the reproducible production recipe)"
                " and `GT-RF-01-scope-and-next.md` (scope of this video and the next).\n")

    # ---------- verify parity against delivered files ----------
    delivered = [os.path.basename(x) for pat in
                 ("*.mp4", "segments-16x9/*.mp4", "segments-9x16/*.mp4", "clips/*.mp4")
                 for x in __import__("glob").glob(os.path.join(EXP, pat))]
    have = {m[0] for m in made}
    missing = [d for d in delivered if d not in have]
    if missing:
        raise SystemExit("missing transcript/folder for: " + ", ".join(missing))
    shutil.rmtree(os.path.join(COMP, "_stills"))

    # ---------- zip ----------
    zpath = os.path.join(RENDER, "GT-RF-01-companion-package.zip")
    if os.path.exists(zpath):
        os.remove(zpath)
    subprocess.run(["zip", "-q", "-r", zpath, "."], cwd=COMP, check=True)
    print("videos covered:", len(made), "| zip:",
          f"{os.path.getsize(zpath)/1048576:.1f} MiB", zpath)


if __name__ == "__main__":
    main()
