#!/usr/bin/env bash
# Show stock Autonomous OS commits that touch files we overlaid.
#
# Stock:   git remote `upstream` → https://github.com/autonomous-ai/autonomous-os
# Ours:    git remote `origin`   → https://github.com/K95M65/kestrel
#
# Usage (from repo root or anywhere):
#   ./scripts/check-upstream-divergence.sh
#   ./scripts/check-upstream-divergence.sh --fetch   # git fetch upstream first
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PATHS_FILE="$ROOT/scripts/upstream-watch-paths.txt"
if [[ ! -f "$PATHS_FILE" ]]; then
  echo "missing $PATHS_FILE" >&2
  exit 1
fi

WATCH=()
while IFS= read -r line; do
  WATCH+=("$line")
done < <(grep -vE '^\s*(#|$)' "$PATHS_FILE")

if [[ "${1:-}" == "--fetch" ]]; then
  git fetch upstream
fi

if ! git rev-parse --verify upstream/main >/dev/null 2>&1; then
  echo "no upstream/main — add remote:" >&2
  echo "  git remote add upstream https://github.com/autonomous-ai/autonomous-os.git" >&2
  echo "  git fetch upstream" >&2
  exit 1
fi

BASE="$(git merge-base HEAD upstream/main)"
OURS="$(git rev-parse --short HEAD)"
STOCK="$(git rev-parse --short upstream/main)"

echo "ours     $OURS  ($(git log -1 --format=%s HEAD))"
echo "stock    $STOCK  ($(git log -1 --format=%s upstream/main))"
echo "forked   $(git rev-parse --short "$BASE")"
echo
echo "=== stock commits since fork that touch our overlay paths ==="
echo "(empty means nothing to merge from stock on these files)"
echo
git --no-pager log --oneline --decorate "$BASE"..upstream/main -- "${WATCH[@]}"
echo
echo "=== our commits since fork that touch the same paths ==="
echo
git --no-pager log --oneline "$BASE"..HEAD -- "${WATCH[@]}"
echo
echo "Re-run with --fetch to refresh upstream/main."
echo "Catalog: docs/divergence-from-stock.md"
