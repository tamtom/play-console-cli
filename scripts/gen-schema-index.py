#!/usr/bin/env python3
"""Generate or verify the compact embedded Google Play schema index."""

import argparse
import json
import pathlib
import sys
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[1]
INDEX = ROOT / "internal" / "apischema" / "schema-index.json"
MANIFEST = ROOT / "docs" / "api" / "google-play-api-manifest.json"

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


def fetch(url):
    documents = []
    for _ in range(3):
        request = urllib.request.Request(url, headers={"User-Agent": "gplay-api-discovery/1"})
        with urllib.request.urlopen(request, timeout=30) as response:
            documents.append(json.load(response))
    return max(documents, key=lambda item: item.get("revision", ""))


def parameter(name, value):
    result = {
        "name": name,
        "location": value.get("location", ""),
        "type": value.get("type", ""),
    }
    for source, target in (
        ("format", "format"),
        ("description", "description"),
        ("required", "required"),
        ("repeated", "repeated"),
        ("deprecated", "deprecated"),
        ("enum", "enum"),
        ("enumDescriptions", "enum_descriptions"),
        ("default", "default"),
        ("minimum", "minimum"),
        ("maximum", "maximum"),
        ("pattern", "pattern"),
    ):
        if source in value:
            result[target] = value[source]
    return result


def endpoint(api_name, api_version, method):
    result = {
        "api": api_name,
        "version": api_version,
        "id": method["id"],
        "http_method": method.get("httpMethod", ""),
        "path": method.get("path", ""),
    }
    if method.get("description"):
        result["description"] = method["description"]
    parameters = [parameter(name, value) for name, value in method.get("parameters", {}).items()]
    parameters.sort(key=lambda item: (item["location"], item["name"]))
    if parameters:
        result["parameters"] = parameters
    if method.get("parameterOrder"):
        result["parameter_order"] = method["parameterOrder"]
    if method.get("request", {}).get("$ref"):
        result["request_type"] = method["request"]["$ref"]
    if method.get("response", {}).get("$ref"):
        result["response_type"] = method["response"]["$ref"]
    if method.get("scopes"):
        result["scopes"] = sorted(method["scopes"])
    media = method.get("mediaUpload", {})
    if media:
        media_result = {}
        for source, target in (("accept", "accept"), ("maxSize", "max_size")):
            if source in media:
                media_result[target] = media[source]
        for protocol in ("simple", "resumable"):
            path = media.get("protocols", {}).get(protocol, {}).get("path")
            if path:
                media_result[protocol + "_path"] = path
        result["media_upload"] = media_result
    return result


def collect_methods(resources, api_name, api_version, result):
    for resource in resources.values():
        for method in resource.get("methods", {}).values():
            result.append(endpoint(api_name, api_version, method))
        collect_methods(resource.get("resources", {}), api_name, api_version, result)


def build_index():
    apis = []
    endpoints = []
    schemas = []
    for api_name, url in sorted(APIS.items()):
        document = fetch(url)
        version = document.get("version", "")
        apis.append(
            {
                "name": api_name,
                "version": version,
                "revision": document.get("revision", ""),
                "discovery_url": url,
                "base_url": document.get("rootUrl", document.get("baseUrl", "")),
                "service_path": document.get("servicePath", document.get("basePath", "")),
            }
        )
        for method in document.get("methods", {}).values():
            endpoints.append(endpoint(api_name, version, method))
        collect_methods(document.get("resources", {}), api_name, version, endpoints)
        for name, definition in document.get("schemas", {}).items():
            schemas.append({"api": api_name, "name": name, "definition": definition})

    endpoints.sort(key=lambda item: item["id"])
    schemas.sort(key=lambda item: (item["api"], item["name"]))
    return {"schema_version": 1, "apis": apis, "endpoints": endpoints, "types": schemas}


def check_index():
    try:
        index = json.loads(INDEX.read_text())
        manifest = json.loads(MANIFEST.read_text())
    except (OSError, json.JSONDecodeError) as error:
        print(f"schema index check failed: {error}", file=sys.stderr)
        return 1

    failures = []
    if index.get("schema_version") != 1:
        failures.append("unsupported or missing schema_version")

    endpoint_ids = [item.get("id", "") for item in index.get("endpoints", [])]
    if endpoint_ids != sorted(endpoint_ids):
        failures.append("endpoints are not sorted by method ID")
    if len(endpoint_ids) != len(set(endpoint_ids)):
        failures.append("duplicate endpoint method IDs")

    indexed_apis = {item.get("name"): item for item in index.get("apis", [])}
    for api_name, reviewed in sorted(manifest.get("apis", {}).items()):
        indexed = indexed_apis.get(api_name)
        if indexed is None:
            failures.append(f"{api_name}: missing API metadata")
            continue
        if indexed.get("revision") != reviewed.get("revision"):
            failures.append(
                f"{api_name}: revision {indexed.get('revision')} != manifest {reviewed.get('revision')}"
            )
        actual = sorted(
            item["id"] for item in index.get("endpoints", []) if item.get("api") == api_name
        )
        expected = sorted(reviewed.get("methods", []))
        if actual != expected:
            failures.append(
                f"{api_name}: endpoint IDs differ; added={sorted(set(actual) - set(expected))}, "
                f"missing={sorted(set(expected) - set(actual))}"
            )

    if failures:
        print("Embedded Google Play schema index is inconsistent:", file=sys.stderr)
        for failure in failures:
            print(f"- {failure}", file=sys.stderr)
        print("Run python3 scripts/gen-schema-index.py --update after reviewing official drift.", file=sys.stderr)
        return 1

    print(
        f"Embedded schema index is consistent: {len(index.get('endpoints', []))} endpoints, "
        f"{len(index.get('types', []))} types."
    )
    return 0


def main():
    parser = argparse.ArgumentParser()
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--update", action="store_true", help="fetch official discovery and replace the index")
    action.add_argument("--check", action="store_true", help="verify the index against the reviewed manifest")
    args = parser.parse_args()

    if args.check:
        return check_index()

    index = build_index()
    INDEX.parent.mkdir(parents=True, exist_ok=True)
    INDEX.write_text(json.dumps(index, indent=2, sort_keys=True) + "\n")
    print(
        f"updated {INDEX}: {len(index['endpoints'])} endpoints, {len(index['types'])} types"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
