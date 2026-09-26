#!/usr/bin/env python3
"""Compare discovery fields with the pinned Go SDK and reviewed raw adapters."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import subprocess
import sys
import urllib.request

ROOT = Path(__file__).resolve().parents[1]


def fields(schema, prefix):
    result = set()
    for name, definition in schema.get("properties", {}).items():
        path = prefix + "." + name
        result.add(path)
        result.update(fields(definition, path))
    for name, suffix in (("items", "[]"), ("additionalProperties", "{}")):
        if isinstance(schema.get(name), dict):
            result.update(fields(schema[name], prefix + suffix))
    return result


def structure(value):
    if isinstance(value, dict):
        return {key: structure(item) for key, item in value.items()
                if key not in ("description", "enumDescriptions")}
    if isinstance(value, list):
        return [structure(item) for item in value]
    return value


def fetch(api):
    documents = []
    for _ in range(3):
        request = urllib.request.Request(api["discovery_url"], headers={"User-Agent": "gplay-sdk-coverage/1"})
        with urllib.request.urlopen(request, timeout=30) as response:
            documents.append(json.load(response))
    return api["name"], max(documents, key=lambda document: document.get("revision", ""))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--index", type=Path, default=ROOT / "internal/apischema/schema-index.json")
    parser.add_argument("--adapters", type=Path, default=ROOT / "docs/api/sdk-field-adapters.json")
    parser.add_argument("--sdk-root", type=Path)
    parser.add_argument("--live", action="store_true", help="also compare with current official discovery schemas")
    args = parser.parse_args()
    try:
        index = json.loads(args.index.read_text())
        adapters = json.loads(args.adapters.read_text())
        sdk_root = args.sdk_root
        if sdk_root is None:
            sdk_root = Path(subprocess.check_output(
                ["go", "list", "-m", "-f", "{{.Dir}}", "google.golang.org/api"], cwd=ROOT, text=True).strip())
        reviewed = {api["name"]: {} for api in index["apis"]}
        for entry in index["types"]:
            reviewed[entry["api"]][entry["name"]] = entry["definition"]
        schemas = reviewed
        failures = []
        if args.live:
            with ThreadPoolExecutor(max_workers=8) as pool:
                live = dict(pool.map(fetch, index["apis"]))
            schemas = {name: document.get("schemas", {}) for name, document in live.items()}
            for name in reviewed:
                if structure(reviewed[name]) != structure(schemas[name]):
                    failures.append(name + ": current discovery schema differs from the reviewed index")
        missing = set()
        total = 0
        for api in index["apis"]:
            directory = sdk_root / api["name"] / api["version"]
            matches = list(directory.glob("*-api.json"))
            if len(matches) != 1:
                raise ValueError(f"expected one SDK discovery document in {directory}")
            sdk = json.loads(matches[0].read_text()).get("schemas", {})
            for name, definition in schemas[api["name"]].items():
                prefix = api["name"] + "." + name
                expected = fields(definition, prefix)
                total += len(expected)
                missing.update(expected - fields(sdk.get(name, {}), prefix))
        for field in sorted(missing - adapters.keys()):
            failures.append(field + ": absent from SDK and reviewed adapters")
        for field, reason in sorted(adapters.items()):
            if field not in missing or not isinstance(reason, str) or not reason.strip():
                failures.append(field + ": stale or undocumented adapter exception")
        if failures:
            print("SDK field coverage failed:\n- " + "\n- ".join(failures), file=sys.stderr)
            return 1
        print(f"SDK field coverage passed: {total} fields, {len(missing)} reviewed adapter fields, {len(index['apis'])} APIs.")
        return 0
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        print(f"SDK field coverage incomplete: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
