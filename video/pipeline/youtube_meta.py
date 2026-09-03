#!/usr/bin/env python3
"""Generate the YouTube upload manifest: every GT-TR-01 video with its title,
description, tags, category, and scheduled publishAt (UTC) per the posting
plan (video/GT-TR-01/GT-TR-01-posting-plan.md; ET = UTC-4 in September).

Usage: youtube_meta.py <renderdir GT-TR-01> <out.json>
"""
import json, sys
from pathlib import Path

SITE = "https://gophertrunk.org/reference"
UTM = "?utm_source=youtube&utm_medium=video&utm_campaign=gt-tr-01"
TAGS_BASE = ["SDR", "software defined radio", "trunked radio", "radio scanner",
             "P25", "DMR", "TETRA", "RTL-SDR", "GopherTrunk", "scanner radio",
             "ham radio", "trunking"]
FOOT = ("\n\nFull written article with diagrams and sources:\n{art}\n\n"
        "GopherTrunk is an open-source SDR trunking scanner/decoder.\n"
        "Field Guide: https://gophertrunk.org/reference/" + UTM + "\n"
        "#SDR #TrunkedRadio #RadioScanner")

SEGS = {
    "01": ("trunked-radio", "Trunked radio",
           "What Is Trunked Radio? How 100 Groups Share 5 Channels",
           "How trunked radio shares a small pool of frequencies among hundreds of user groups — the channel pool, grants, and the statistics that make it work."),
    "02": ("control-channel", "Control channel",
           "What Is a Control Channel? The Frequency That Never Speaks",
           "The data-only coordinator of every trunked radio system: registrations, grants, system info — and why you decode it first."),
    "03": ("talkgroup", "Talkgroup",
           "What Is a Talkgroup? The Radio Channel That Doesn't Exist",
           "Talkgroups are virtual channels: calls hop across physical frequencies while the group ID stays put. Here's how that works."),
    "04": ("channel-grant", "Channel grant",
           "What Is a Channel Grant? The Message That Starts Every Radio Call",
           "Inside the control-channel message that assigns every call to a voice channel: target, channel number, timeslot, updates, and late entry."),
    "05": ("fdma", "FDMA & TDMA",
           "FDMA vs TDMA: The Two Ways Radio Calls Share the Air",
           "One call per frequency, or calls taking turns in timeslots — how FDMA and TDMA work and why TDMA grants name a slot."),
}
CLIP_TITLES = {
    "01c1": "Why 30 radio channels sit silent #shorts",
    "01c2": "Radio channels that exist for one call #shorts",
    "02c1": "The radio channel that never speaks #shorts",
    "02c2": "Joining a call you never heard start #shorts",
    "03c1": "This radio channel doesn't exist #shorts",
    "03c2": "Two numbers identify every radio call #shorts",
    "04c1": "One message moves 100 radios at once #shorts",
    "04c2": "Inside the message that starts every call #shorts",
    "05c1": "The simplest way radios share the air #shorts",
    "05c2": "How 4 calls fit on one frequency #shorts",
}
# publishAt (UTC, 2026; 18:00 ET=22:00Z, 10:00 ET=14:00Z)
SCHED = {
    "02c1": "2026-09-07T22:00:00Z", "pillar": "2026-09-08T22:00:00Z",
    "01c1": "2026-09-09T22:00:00Z", "01c2": "2026-09-11T22:00:00Z",
    "v01": "2026-09-13T14:00:00Z",
    "03c1": "2026-09-14T22:00:00Z", "seg01": "2026-09-15T22:00:00Z",
    "02c2": "2026-09-16T22:00:00Z", "03c2": "2026-09-18T22:00:00Z",
    "v02": "2026-09-20T14:00:00Z",
    "04c1": "2026-09-21T22:00:00Z", "seg02": "2026-09-22T22:00:00Z",
    "04c2": "2026-09-23T22:00:00Z", "seg03": "2026-09-24T22:00:00Z",
    "05c1": "2026-09-25T22:00:00Z", "v03": "2026-09-27T14:00:00Z",
    "05c2": "2026-09-28T22:00:00Z", "seg04": "2026-09-29T22:00:00Z",
    "seg05": "2026-10-01T22:00:00Z",
    "v04": "2026-10-03T14:00:00Z", "v05": "2026-10-04T14:00:00Z",
}


def main():
    rd, outp = Path(sys.argv[1]), Path(sys.argv[2])
    chapters = (rd / "final/GT-TR-01-chapters.txt").read_text().strip()
    items = []

    art = f"{SITE}/trunked-radio/{UTM}"
    items.append({
        "key": "pillar", "file": str(rd / "final/GT-TR-01-pillar.mp4"),
        "title": "How Trunked Radio Works — the complete course (P25, DMR, TETRA)",
        "description": ("Five ideas, one course: the channel pool, the control channel, talkgroups, "
                        "channel grants, and FDMA vs TDMA. By the end you can read a trunked system like a map.\n\n"
                        "Chapters:\n" + chapters + FOOT.format(art=art)),
        "tags": TAGS_BASE + ["trunked radio explained", "how trunking works"],
        "publishAt": SCHED["pillar"],
        "thumb": "brand/thumbs/GT-TR-01-thumb.png",
        "srt": str(rd / "final/GT-TR-01-pillar.srt"),
        "playlist": True,
    })
    for n, (slug, term, title, hook) in SEGS.items():
        art = f"{SITE}/{slug}/{UTM}"
        items.append({
            "key": f"seg{n}", "file": str(rd / f"final/GT-TR-01.{n}.mp4"),
            "title": title,
            "description": hook + "\n\nPart of the Trunked Radio course — full course video on this channel."
                           + FOOT.format(art=art),
            "tags": TAGS_BASE + [term.lower()],
            "publishAt": SCHED[f"seg{n}"],
            "thumb": f"brand/thumbs/GT-TR-01.{n}-thumb.png",
            "srt": str(rd / f"tts/GT-TR-01.{n}.srt"),
            "playlist": True,
        })
        items.append({
            "key": f"v{n}", "file": str(rd / f"final/GT-TR-01.{n}-vertical.mp4"),
            "title": f"{term} in 3 minutes #shorts",
            "description": hook + FOOT.format(art=art),
            "tags": TAGS_BASE + ["shorts"],
            "publishAt": SCHED[f"v{n}"],
        })
    for cid, title in CLIP_TITLES.items():
        slug = SEGS[cid[:2]][0]
        art = f"{SITE}/{slug}/{UTM}"
        items.append({
            "key": cid, "file": str(rd / f"shorts/GT-TR-01.{cid}.mp4"),
            "title": title,
            "description": SEGS[cid[:2]][3] + FOOT.format(art=art),
            "tags": TAGS_BASE + ["shorts"],
            "publishAt": SCHED[cid],
        })
    for it in items:
        it.setdefault("categoryId", "28")  # Science & Technology
    outp.parent.mkdir(parents=True, exist_ok=True)
    outp.write_text(json.dumps({"playlistTitle": "Trunked Radio — the GopherTrunk Field Guide course",
                                "playlistDescription": "The trunked radio pillar, one idea at a time: pool, control channel, talkgroups, grants, FDMA & TDMA.",
                                "items": items}, indent=1))
    print(f"{len(items)} videos → {outp}")
    for it in items:
        missing = "" if Path(it["file"]).exists() else "  [MISSING FILE]"
        print(f"  {it['key']:6s} {it['publishAt']}  {it['title'][:60]}{missing}")


if __name__ == "__main__":
    main()
