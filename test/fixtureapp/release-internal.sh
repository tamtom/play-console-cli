#!/usr/bin/env bash
# Build the fixture AAB and release it to the INTERNAL track only.
# The compiled gplay binary performs the upload, the publish, and the
# readback verification. See README.md for the required secrets.
set -euo pipefail

FIXTURE_PACKAGE="com.itdeveapps.stepsshare"
TRACK="internal"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
APP_DIR="$REPO_ROOT/test/fixtureapp"
GPLAY="$REPO_ROOT/gplay"

if [ ! -x "$GPLAY" ]; then
  echo "gplay binary not found at $GPLAY; run 'make build' first" >&2
  exit 1
fi
for var in FIXTURE_KEYSTORE FIXTURE_KEYSTORE_PASSWORD FIXTURE_KEY_ALIAS FIXTURE_KEY_PASSWORD; do
  if [ -z "${!var:-}" ]; then
    echo "$var is required" >&2
    exit 1
  fi
done

# Version code: the store rejects any code that was ever uploaded, not only
# the codes that are active on a track. The bundle and APK lists inside an
# edit are the authoritative record of used codes, so we read those through
# one disposable edit and delete the edit immediately.
echo "Reading used version codes (bundles and APKs)..."
EDIT_ID="$("$GPLAY" edits create --package "$FIXTURE_PACKAGE" --output json | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
cleanup_edit() {
  "$GPLAY" edits delete --package "$FIXTURE_PACKAGE" --edit "$EDIT_ID" --confirm >/dev/null 2>&1 || true
}
trap cleanup_edit EXIT

BUNDLES_JSON="$("$GPLAY" bundles list --package "$FIXTURE_PACKAGE" --edit "$EDIT_ID" --output json)"
APKS_JSON="$("$GPLAY" apks list --package "$FIXTURE_PACKAGE" --edit "$EDIT_ID" --output json)"
cleanup_edit
trap - EXIT

MAX_CODE="$(printf '%s\n---\n%s' "$BUNDLES_JSON" "$APKS_JSON" | python3 -c '
import json, sys
bundles_raw, apks_raw = sys.stdin.read().split("\n---\n")
codes = []
for artifact in (json.loads(bundles_raw).get("bundles") or []):
    codes.append(int(artifact["versionCode"]))
for artifact in (json.loads(apks_raw).get("apks") or []):
    codes.append(int(artifact["versionCode"]))
print(max(codes) if codes else 0)
')"
NEXT_CODE=$((MAX_CODE + 1))
echo "Max used version code: $MAX_CODE; building $NEXT_CODE"

echo "Building fixture AAB..."
(cd "$APP_DIR" && gradle --no-daemon bundleRelease "-PfixtureVersionCode=$NEXT_CODE")
AAB="$APP_DIR/app/build/outputs/bundle/release/app-release.aab"
test -f "$AAB"

echo "Releasing version code $NEXT_CODE to the $TRACK track..."
"$GPLAY" release \
  --package "$FIXTURE_PACKAGE" \
  --track "$TRACK" \
  --bundle "$AAB" \
  --release-notes "Automated fixture release (CI run ${GITHUB_RUN_ID:-local})" \
  --output json

echo "Readback: verify the release is on the $TRACK track..."
VERIFY_JSON="$("$GPLAY" tracks releases list --package "$FIXTURE_PACKAGE" --track "$TRACK" --output json)"
printf '%s' "$VERIFY_JSON" | python3 -c "
import json, sys
data = json.load(sys.stdin)
codes = set()
for release in data.get('releases') or []:
    for artifact in release.get('activeArtifacts') or []:
        codes.add(int(artifact['versionCode']))
expected = $NEXT_CODE
if expected not in codes:
    raise SystemExit(f'readback failed: version code {expected} not on track, saw {sorted(codes)}')
print(f'readback OK: version code {expected} is live on the internal track')
"
