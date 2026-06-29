#!/usr/bin/env bash
# Physics-IQ dry-run: build flowagent and run director --dry-run (no API).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
go build -o bin/flowagent ./cmd/flowagent
RUN_DIR="$ROOT/runs/physics-iq-dryrun-$$"
mkdir -p "$RUN_DIR"
./bin/flowagent director --stack micro-movie-wan-flash --opening-shot "雨夜霓虹巷口，男子驻足。" --out "$RUN_DIR" --dry-run
echo "dry-run artifacts: $RUN_DIR"
