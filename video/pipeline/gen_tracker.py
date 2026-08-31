"""Generate GT-RF-01-tracker.csv from actual production state (per playbook Part 5)."""
import csv
import glob
import json
import os
import re

from common import RENDER, VIDEO

SEGDIR = os.path.join(VIDEO, "GT-RF-01")


def main():
    rows = []
    yml = {}
    stub = os.path.join(RENDER, "exports", "videos.yml.stub")
    if os.path.exists(stub):
        cur = None
        for line in open(stub):
            m = re.match(r"  (\S+):$", line)
            if m:
                cur = m.group(1)
            m = re.match(r"    (start_s|end_s): (\d+)", line)
            if m and cur:
                yml.setdefault(cur, {})[m.group(1)] = m.group(2)
    for path in sorted(glob.glob(os.path.join(SEGDIR, "scripts", "GT-RF-01.[0-9]*.md"))):
        base = os.path.basename(path)
        seg = base.split("-", 3)
        seg_id = "GT-RF-01." + base.split(".")[2].split("-")[0]
        head = open(path, encoding="utf-8").read(600)
        slug = (re.search(r"^slug: (\S+)", head, re.M) or [None, ""])[1]
        title = (re.search(r"— (.+)$", head.splitlines()[0]) or [None, ""])[1]
        typ = (re.search(r"^type: (\S+)", head, re.M) or [None, "term"])[1]
        rendered = os.path.exists(os.path.join(RENDER, seg_id, "video.mp4"))
        vert = bool(glob.glob(os.path.join(RENDER, "exports", "segments-9x16", seg_id + "*")))
        nclips = len(glob.glob(os.path.join(RENDER, "exports", "clips", seg_id + "c*")))
        y = yml.get(slug, {})
        rows.append([seg_id, slug, typ, title, "done", "done",
                     "done" if rendered else "", "done" if rendered else "",
                     "done" if rendered else "", "done" if vert else "",
                     f"{nclips}/3" if nclips else "",
                     y.get("start_s", ""), y.get("end_s", ""), "", "",
                     "" if rendered else "package-only (script+storyboard); not yet rendered"])
    out = os.path.join(SEGDIR, "GT-RF-01-tracker.csv")
    with open(out, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow("seg_id,slug,type,title,script,storyboard,shot,edited,qc,vertical,clips,start_s,end_s,posted_tiktok,posted_shorts,notes".split(","))
        w.writerows(rows)
        w.writerow(["GT-RF-01.P*", "", "pillar", "cold open + intro + 5 transitions + outro + end slate",
                    "done", "done", "done", "done", "done", "", "", "", "", "", "",
                    "pilot connective tissue (Act I cut)"])
    print("wrote", out, len(rows), "segments")


if __name__ == "__main__":
    main()
