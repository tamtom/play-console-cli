#!/usr/bin/env python3
"""Generate docs/api/endpoints.txt from the Google Play API discovery document."""

import json
import os

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
ROOT_DIR = os.path.dirname(SCRIPT_DIR)
DISCOVERY_PATH = os.path.join(ROOT_DIR, "docs", "api", "discovery.json")
ENDPOINTS_PATH = os.path.join(ROOT_DIR, "docs", "api", "endpoints.txt")


def extract(resources, prefix=""):
    lines = []
    for name, res in sorted(resources.items()):
        path = prefix + "." + name if prefix else name
        for mname, method in sorted(res.get("methods", {}).items()):
            http_method = method["httpMethod"]
            api_path = method.get("path", "N/A")
            lines.append(f"{http_method:6s} {api_path:60s} {path}.{mname}")
        lines.extend(extract(res.get("resources", {}), path))
    return lines


def main():
    with open(DISCOVERY_PATH) as f:
        doc = json.load(f)
    lines = extract(doc.get("resources", {}))
    with open(ENDPOINTS_PATH, "w") as f:
        f.write("\n".join(lines) + "\n")
    print(f"  {len(lines)} endpoints indexed.")


if __name__ == "__main__":
    main()
