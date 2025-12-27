#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPAT_FILE="$ROOT_DIR/internal/workers/compat.json"
WORKERS_CMD_FILE="$ROOT_DIR/internal/cmd/workers.go"

if [[ ! -f "$COMPAT_FILE" ]]; then
  echo "compat file not found: $COMPAT_FILE" >&2
  exit 1
fi

for cmd in git jq npm; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "missing required command: $cmd" >&2
    exit 1
  fi
done

NTN_VERSION="$(npm view ntn version --silent | tr -d '[:space:]')"
if [[ -z "$NTN_VERSION" ]]; then
  echo "failed to resolve ntn npm version" >&2
  exit 1
fi

TEMPLATE_COMMIT="$(git ls-remote https://github.com/makenotion/workers-template HEAD | awk '{print $1}')"
if [[ -z "$TEMPLATE_COMMIT" ]]; then
  echo "failed to resolve workers-template HEAD commit" >&2
  exit 1
fi

UPDATED_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

TMP_FILE="$(mktemp)"
jq \
  --arg ntn "$NTN_VERSION" \
  --arg commit "$TEMPLATE_COMMIT" \
  --arg updated "$UPDATED_AT" \
  '.workers_cli_version = $ntn | .template_commit = $commit | .updated_at = $updated' \
  "$COMPAT_FILE" > "$TMP_FILE"
mv "$TMP_FILE" "$COMPAT_FILE"

TMP_GO="$(mktemp)"
sed -E "s/(fallbackWorkersCLIVersion[[:space:]]*=[[:space:]]*)\"[^\"]+\"/\1\"$NTN_VERSION\"/" "$WORKERS_CMD_FILE" > "$TMP_GO"
mv "$TMP_GO" "$WORKERS_CMD_FILE"

# Fast sanity check that the embedded config still parses.
( cd "$ROOT_DIR" && go test ./internal/workers -run Current >/dev/null )

echo "Updated workers pins:"
echo "  workers_cli_version: $NTN_VERSION"
echo "  template_commit:     $TEMPLATE_COMMIT"
echo "  updated_at:          $UPDATED_AT"
