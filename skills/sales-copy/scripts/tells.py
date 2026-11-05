#!/usr/bin/env python3
"""Flag the marks of machine-written Portuguese in a piece before a person reads it.

Usage: python3 tells.py draft.txt     (or "-" for stdin)

FAIL is mechanical and always fixed: invisible characters, dashes and curly quotes no keyboard types.
WARN needs judgement: a stock phrase from tells.json or a structure a person rarely writes. Rewrite or keep it out loud.
Exit 1 when anything FAILs, so a pipeline can stop on it.
"""

import json
import re
import sys
import unicodedata
from pathlib import Path

DASHES = {"—": "travessão", "–": "meia-risca"}
QUOTES = {"“": '"', "”": '"', "‘": "'", "’": "'", "…": "..."}
MAX_HASHTAGS = 5


def invisible(ch: str) -> bool:
    # Category Cf holds every zero-width and tag character; a keyboard never produces one.
    return unicodedata.category(ch) == "Cf" or ch in "   "


def line_of(text: str, index: int) -> int:
    return text.count("\n", 0, index) + 1


def check(text: str, rules: dict) -> list[tuple[str, int, str]]:
    found = []
    for i, ch in enumerate(text):
        if invisible(ch):
            found.append(("FAIL", line_of(text, i), f"caractere invisível U+{ord(ch):04X}"))
        elif ch in DASHES:
            found.append(("FAIL", line_of(text, i), f"{DASHES[ch]}: vira ponto, vírgula ou parênteses"))
        elif ch in QUOTES:
            found.append(("FAIL", line_of(text, i), f"aspas ou reticências tipográficas: use {QUOTES[ch]}"))

    lowered = text.lower()
    for entry in rules["lexicon"]:
        for match in re.finditer(r"(?<!\w)" + re.escape(entry["pt"]) + r"(?!\w)", lowered):
            found.append(("WARN", line_of(text, match.start()), f"\"{entry['pt']}\" → {entry['use']}"))

    for entry in rules["structures"]:
        for match in re.finditer(entry["regex"], text):
            found.append(("WARN", line_of(text, match.start()), entry["name"]))

    hashtags = re.findall(r"(?<!\w)#\w+", text)
    if len(hashtags) > MAX_HASHTAGS:
        found.append(("WARN", 0, f"{len(hashtags)} hashtags: a plataforma aceita até {MAX_HASHTAGS}"))

    first = text.lstrip()[:1]
    if first and unicodedata.category(first) == "So":
        found.append(("WARN", 1, "emoji como primeiro caractere: some no corte do feed"))

    return sorted(found, key=lambda f: (f[1], f[0]))


def main() -> int:
    if len(sys.argv) != 2:
        print(__doc__.strip(), file=sys.stderr)
        return 2
    source = sys.argv[1]
    text = sys.stdin.read() if source == "-" else Path(source).read_text(encoding="utf-8")
    rules = json.loads((Path(__file__).parent / "tells.json").read_text(encoding="utf-8"))

    found = check(text, rules)
    for level, line, message in found:
        print(f"{level}  linha {line}: {message}" if line else f"{level}  {message}")
    fails = sum(1 for f in found if f[0] == "FAIL")
    warns = len(found) - fails
    print(f"{'FAIL' if fails else 'OK'}  {fails} para corrigir, {warns} para decidir")
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
