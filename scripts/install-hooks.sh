#!/usr/bin/env bash
# Install repo git hooks (run once after clone).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOKS_SRC="$ROOT/scripts/hooks"
HOOKS_DST="$ROOT/.git/hooks"

if [[ ! -d "$ROOT/.git" ]]; then
  echo "error: not a git repository ($ROOT)" >&2
  exit 1
fi

mkdir -p "$HOOKS_DST"
chmod +x "$ROOT/scripts/build.sh" "$HOOKS_SRC/pre-commit"

for hook in "$HOOKS_SRC"/*; do
  name="$(basename "$hook")"
  ln -sf "../../scripts/hooks/$name" "$HOOKS_DST/$name"
  chmod +x "$HOOKS_DST/$name" 2>/dev/null || true
  echo "installed $name"
done

echo
echo "Pre-commit will run scripts/build.sh on every commit."
echo "Skip once: SKIP_BUILD=1 git commit …"
