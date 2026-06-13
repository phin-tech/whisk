#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
import re
from pathlib import Path

text = Path("build/config.yml").read_text()
match = re.search(r'(?m)^\s{2}version:\s*"([^"]+)"', text)
if not match:
    raise SystemExit("failed to read build/config.yml info.version")
print(match.group(1))
PY
