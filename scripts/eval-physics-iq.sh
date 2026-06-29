#!/usr/bin/env bash
# Physics-IQ dry-run: build flowagent and run director --dry-run (no API).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
go build -o bin/flowagent ./cmd/flowagent
./bin/flowagent run micro-movie \
  --stack micro-movie-wan-flash \
  --plot "雨夜霓虹巷口，男子驻足。" \
  --dry-run \
  --auto-gate
echo "physics-iq dry-run OK"
