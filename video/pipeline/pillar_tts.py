"""Split the pillar-elements script into pieces and synthesize each.

Pieces are delimited by ## / ### headings; each becomes its own timeline
(GT-RF-01.P0-coldopen, .P1-intro, .T1..T5, .P8-outro). The end slate is
music-only and is generated at assembly time.
"""
import os
import re
import sys
import tempfile

import tts

NAMES = {
    "COLD OPEN": ("GT-RF-01.P0-coldopen", 0.6, 0.6),
    "COURSE INTRO": ("GT-RF-01.P1-intro", 0.6, 1.0),
    "T1": ("GT-RF-01.T1", 0.5, 0.4), "T2": ("GT-RF-01.T2", 0.5, 0.4),
    "T3": ("GT-RF-01.T3", 0.5, 0.4), "T4": ("GT-RF-01.T4", 0.5, 0.4),
    "T5": ("GT-RF-01.T5", 0.5, 0.4),
    "OUTRO": ("GT-RF-01.P8-outro", 0.6, 1.2),
}


def main(path):
    text = open(path, encoding="utf-8").read()
    sections = re.split(r"^#{2,3} ", text, flags=re.M)[1:]
    for sec in sections:
        head, _, body = sec.partition("\n")
        key = next((k for k in NAMES if head.strip().upper().startswith(k)), None)
        if key is None:
            continue
        name, lead, tail = NAMES[key]
        # drop the endslate direction from the outro piece (assembled separately)
        body = "\n".join(l for l in body.splitlines() if "[V: endslate" not in l)
        with tempfile.NamedTemporaryFile("w", suffix=".md", delete=False) as f:
            f.write(f"# {name} — pillar element\n\n{body}\n")
            tmp = f.name
        tts.build(tmp, lead=lead, tail=tail)
        os.unlink(tmp)


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else
         os.path.join(os.path.dirname(__file__), "..", "GT-RF-01", "scripts",
                      "GT-RF-01.00-pillar-elements.md"))
