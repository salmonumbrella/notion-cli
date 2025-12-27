#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPAT_FILE="$ROOT_DIR/internal/workers/compat.json"
WORKERS_CMD_FILE="$ROOT_DIR/internal/cmd/workers.go"
README_FILE="$ROOT_DIR/README.md"
HELP_FILE="$ROOT_DIR/internal/cmd/help.txt"

for file in "$COMPAT_FILE" "$WORKERS_CMD_FILE" "$README_FILE" "$HELP_FILE"; do
  if [[ ! -f "$file" ]]; then
    echo "required file missing: $file" >&2
    exit 1
  fi
done

compat_cli_version="$(grep -E '"workers_cli_version"' "$COMPAT_FILE" | head -n1 | sed -E 's/.*"workers_cli_version"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')"
fallback_cli_version="$(grep -E 'fallbackWorkersCLIVersion[[:space:]]*=' "$WORKERS_CMD_FILE" | head -n1 | sed -E 's/.*"([^"]+)".*/\1/')"

if [[ -z "$compat_cli_version" || -z "$fallback_cli_version" ]]; then
  echo "failed to parse workers versions from compat/workers.go" >&2
  exit 1
fi

if [[ "$compat_cli_version" != "$fallback_cli_version" ]]; then
  echo "workers CLI version drift detected:" >&2
  echo "  compat.json workers_cli_version: $compat_cli_version" >&2
  echo "  workers.go fallbackWorkersCLIVersion: $fallback_cli_version" >&2
  exit 1
fi

if grep -Eq 'npx --yes ntn@[0-9]+\.[0-9]+\.[0-9]+ workers' "$README_FILE"; then
  echo "README workers proxy command should not hardcode ntn@x.y.z" >&2
  exit 1
fi

if ! grep -q 'workers_cli_version from internal/workers/compat.json' "$README_FILE"; then
  echo "README should reference compat.json for workers CLI version source of truth" >&2
  exit 1
fi

if ! grep -q 'workers status' "$README_FILE" || ! grep -q 'workers status' "$HELP_FILE"; then
  echo "workers status command docs are missing in README/help.txt" >&2
  exit 1
fi

if ! grep -q 'workers upgrade . --plan' "$HELP_FILE"; then
  echo "help.txt should document workers upgrade --plan" >&2
  exit 1
fi

echo "Workers drift check passed."
