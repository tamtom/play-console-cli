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

# Version code: live readback of the internal track, plus one. This keeps
# the version sequence of the app monotonic and small.
echo "Reading current version codes on the $TRACK track..."
RELEASES_JSON="$("$GPLAY" tracks releases list --package "$FIXTURE_PACKAGE" --track "$TRACK" --output json)"
MAX_CODE="$(printf '%s' "$RELEASES_JSON" | python3 -c '
import json, sys
data = json.load(sys.stdin)
codes = []
for release in data.get("releases") or []:
    for code in release.get("versionCodes") or []:
        codes.append(int(code))
print(max(codes) if codes else 0)
')"
NEXT_CODE=$((MAX_CODE + 1))
echo "Max version code on $TRACK: $MAX_CODE; building $NEXT_CODE"

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
    for code in release.get('versionCodes') or []:
        codes.add(int(code))
expected = $NEXT_CODE
if expected not in codes:
    raise SystemExit(f'readback failed: version code {expected} not on track, saw {sorted(codes)}')
print(f'readback OK: version code {expected} is live on the internal track')
"
