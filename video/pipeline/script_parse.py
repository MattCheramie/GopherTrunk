"""Parse a segment script .md into an ordered list of visual events with VO paragraphs.

Script grammar (see gophertrunk-video-pipeline.md §5):
  **[V: name | arg | arg]**  — starts a visual event (name = first field, args kept)
  plain paragraphs           — VO attached to the current visual event
  [CLIP cN start] / [CLIP cN end] — clip bounds (on V-lines or at paragraph ends)
"""
import re

CLIP_RE = re.compile(r"\[CLIP (c\d+) (start|end)\]")
V_RE = re.compile(r"^\*\*\[V:\s*([^\]|]+?)(?:\s*\|\s*(.*?))?\]\*\*(.*)$")


def clean_vo(text):
    text = CLIP_RE.sub("", text)
    text = re.sub(r"\*\*(.+?)\*\*", r"\1", text)
    text = re.sub(r"\*(.+?)\*", r"\1", text)
    return re.sub(r"\s+", " ", text).strip()


def split_sentences(par):
    parts = re.split(r"(?<=[.!?…])\s+", par)
    return [p.strip() for p in parts if p.strip()]


def parse_script(path):
    """Returns {meta, events:[{visual, args, paras:[str], clip_marks:[(cid, which, para_idx)]}]}."""
    meta, events, cur = {}, [], None
    lines = open(path, encoding="utf-8").read().splitlines()
    for raw in lines:
        line = raw.strip()
        if not line:
            continue
        if line.startswith("# "):
            meta["title"] = line[2:].strip()
            continue
        m = re.match(r"^(slug|also_slugs|type|target|note):\s*(.*)$", line)
        if m and cur is None:
            meta[m.group(1)] = m.group(2)
            continue
        vm = V_RE.match(line)
        if vm:
            cur = {"visual": vm.group(1).strip(), "args": vm.group(2) or "",
                   "paras": [], "clip_marks": []}
            events.append(cur)
            for cid, which in CLIP_RE.findall(vm.group(3) or ""):
                cur["clip_marks"].append((cid, which, len(cur["paras"])))
            continue
        if cur is not None:
            for cid, which in CLIP_RE.findall(line):
                # start marker before the paragraph text is read → applies to this para;
                # end marker → applies to this para too (its end)
                cur["clip_marks"].append((cid, which, len(cur["paras"])))
            vo = clean_vo(line)
            if vo:
                cur["paras"].append(vo)
    return {"meta": meta, "events": events}
