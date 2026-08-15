#!/usr/bin/env python3
"""Convert a public generic DTC CSV into the ledger seed format."""

from __future__ import annotations

import csv
import re
import sys
from pathlib import Path

CODE = re.compile(r"^[PCBU][0-3][0-9A-F]{3}$", re.I)


def convert(src: Path, dest: Path) -> int:
    seen: dict[str, str] = {}
    with src.open(newline="", encoding="utf-8", errors="replace") as fh:
        sample = fh.read(2048)
        fh.seek(0)
        dialect = csv.Sniffer().sniff(sample, delimiters=",;")
        reader = csv.reader(fh, dialect)
        for row in reader:
            if len(row) < 2:
                continue
            code = row[0].strip().upper()
            title = row[1].strip()
            if not CODE.match(code) or not title or title.lower() == "description":
                continue
            if code.startswith("P1") or code.startswith("U1"):
                continue
            seen[code] = title
    dest.parent.mkdir(parents=True, exist_ok=True)
    with dest.open("w", newline="", encoding="utf-8") as out:
        w = csv.writer(out)
        w.writerow(["code", "category", "title", "source"])
        for code, title in sorted(seen.items()):
            family = code[0].lower() + "0"
            w.writerow([code, f"sae_{family}", title, "todrobbins_dtcdb_public"])
    return len(seen)


if __name__ == "__main__":
    src = Path(sys.argv[1] if len(sys.argv) > 1 else "/tmp/dtcdb-generic.csv")
    dest = Path(sys.argv[2] if len(sys.argv) > 2 else "cloud-backend/seeds/p0xxx.csv")
    n = convert(src, dest)
    print(f"wrote {n} codes to {dest}")
