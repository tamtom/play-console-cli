#!/usr/bin/env python3
"""Compare official Google Play discovery methods with the reviewed manifest."""

import argparse
import json
import pathlib
import sys
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[1]
DEFAULT_MANIFEST = ROOT / "docs" / "api" / "google-play-api-manifest.json"

APIS = {
    "androiddeveloperidstatus": "https://androiddeveloperidstatus.googleapis.com/$discovery/rest?version=v1",
    "androidpublisher": "https://androidpublisher.googleapis.com/$discovery/rest?version=v3",
    "checks": "https://checks.googleapis.com/$discovery/rest?version=v1alpha",
    "gamesconfiguration": "https://gamesconfiguration.googleapis.com/$discovery/rest?version=v1configuration",
    "gamesmanagement": "https://gamesmanagement.googleapis.com/$discovery/rest?version=v1management",
    "playcustomapp": "https://playcustomapp.googleapis.com/$discovery/rest?version=v1",
    "playdeveloperreporting": "https://playdeveloperreporting.googleapis.com/$discovery/rest?version=v1beta1",
    "playintegrity": "https://playintegrity.googleapis.com/$discovery/rest?version=v1",
}


def method_ids(document):
    result = []

    def walk(resources):
        for resource in resources.values():
            for method in resource.get("methods", {}).values():
                result.append(method["id"])
            walk(resource.get("resources", {}))

    walk(document.get("resources", {}))
    for method in document.get("methods", {}).values():
        result.append(method["id"])
    return sorted(result)


def fetch(url):
    request = urllib.request.Request(url, headers={"User-Agent": "gplay-api-drift-check/1"})
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def current_manifest():
    result = {}
    for name, url in APIS.items():
        document = fetch(url)
        result[name] = {
            "discoveryUrl": url,
            "revision": document.get("revision", ""),
            "version": document.get("version", ""),
            "methods": method_ids(document),
        }
    return {"schemaVersion": 1, "apis": result}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", type=pathlib.Path, default=DEFAULT_MANIFEST)
    parser.add_argument("--update", action="store_true", help="replace the reviewed manifest")
    args = parser.parse_args()

    live = current_manifest()
    if args.update:
        args.manifest.parent.mkdir(parents=True, exist_ok=True)
        args.manifest.write_text(json.dumps(live, indent=2, sort_keys=True) + "\n")
        print(f"updated {args.manifest}")
        return 0

    reviewed = json.loads(args.manifest.read_text())
    failures = []
    for name in sorted(APIS):
        old = reviewed.get("apis", {}).get(name)
        new = live["apis"][name]
        if old is None:
            failures.append(f"{name}: missing from reviewed manifest")
            continue
        added = sorted(set(new["methods"]) - set(old.get("methods", [])))
        removed = sorted(set(old.get("methods", [])) - set(new["methods"]))
        if added or removed:
            failures.append(f"{name}: methods changed; added={added}, removed={removed}")
        elif new["revision"] != old.get("revision"):
            failures.append(f"{name}: discovery revision {old.get('revision')} -> {new['revision']} (methods unchanged; review schema drift)")

    if failures:
        print("Official Google Play API discovery drift detected:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        print("Review the official changes, update integrations/tests, then run make update-api-manifest.", file=sys.stderr)
        return 1
    print("Official Google Play API manifest is current.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
