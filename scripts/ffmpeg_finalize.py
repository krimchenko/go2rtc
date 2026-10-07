#!/usr/bin/env python3

import os
import subprocess
import sys
import time

BASE_PATH = sys.argv[1]
MIN_AGE_SECONDS = float(sys.argv[2])
ACTIVE_FILES = {os.path.abspath(path) for path in sys.argv[3:]}

for directory, _, filenames in os.walk(BASE_PATH):
    for filename in sorted(filenames):
        if not (filename.startswith(".") and filename.endswith("_raw.mp4")):
            continue

        path = os.path.join(directory, filename)
        if os.path.abspath(path) in ACTIVE_FILES:
            continue
        try:
            if time.time() - os.stat(path).st_mtime < MIN_AGE_SECONDS:
                continue
        except FileNotFoundError:
            continue

        stem = filename[: -len("_raw.mp4")]
        finalized_path = os.path.join(directory, stem + "_finalized.mp4")
        result = subprocess.run(
            ["ffmpeg", "-y", "-i", path, "-c", "copy", "-strict", "-2", finalized_path],
            check=False,
        )
        if result.returncode != 0:
            continue

        clean_path = os.path.join(directory, stem[1:] + ".mp4")
        os.replace(finalized_path, clean_path)
        os.remove(path)
