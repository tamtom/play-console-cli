#!/usr/bin/env python3
"""Run the 1.0.0 audit regressions as a temporary Go test overlay."""

import json
import os
from pathlib import Path
import subprocess
import tempfile


def main():
    evidence = Path(__file__).resolve().parent
    repository = evidence.parents[2]
    replacements = {
        "cmd/audit_regressions_test.go": "cmd_regressions_test.go.txt",
        "internal/preflight/audit_policy_test.go": "policy_regressions_test.go.txt",
        "internal/update/audit_update_test.go": "update_regressions_test.go.txt",
    }
    with tempfile.TemporaryDirectory(prefix="gplay-release-audit-") as work:
        work = Path(work)
        overlay = work / "overlay.json"
        overlay.write_text(json.dumps({
            "Replace": {
                str(repository / target): str(evidence / source)
                for target, source in replacements.items()
            }
        }))
        environment = os.environ.copy()
        environment.update({
            "GPLAY_AUDIT": "0",
            "GPLAY_NO_UPDATE": "1",
            "GPLAY_CONFIG_PATH": str(work / "absent-config.json"),
        })
        return subprocess.run(
            ["go", "test", "-overlay", str(overlay), "./cmd",
             "./internal/preflight", "./internal/update",
             "-run", "^TestAudit", "-count=1", "-v"],
            cwd=repository,
            env=environment,
            check=False,
        ).returncode


if __name__ == "__main__":
    raise SystemExit(main())
