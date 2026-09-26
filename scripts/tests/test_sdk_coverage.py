import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class SDKCoverageTest(unittest.TestCase):
    def test_missing_fields_fail_unless_a_reviewed_adapter_covers_them(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            sdk = root / "publisher" / "v3"
            sdk.mkdir(parents=True)
            definition = {"type": "object", "properties": {"packageName": {"type": "string"}}}
            (sdk / "publisher-api.json").write_text(json.dumps({"schemas": {"Update": definition}}))
            current = {"type": "object", "properties": dict(definition["properties"], versionCode={"type": "string", "format": "int64"})}
            index = root / "index.json"
            index.write_text(json.dumps({"apis": [{"name": "publisher", "version": "v3"}],
                                        "types": [{"api": "publisher", "name": "Update", "definition": current}]}))
            adapters = root / "adapters.json"
            for exceptions, succeeds in (({}, False), ({"publisher.Update.versionCode": "raw JSON adapter"}, True),
                                         ({"publisher.Update.missing": "stale exception"}, False)):
                adapters.write_text(json.dumps(exceptions))
                result = subprocess.run([sys.executable, str(ROOT / "scripts/check-sdk-fields.py"),
                                         "--sdk-root", str(root), "--index", str(index), "--adapters", str(adapters)],
                                        capture_output=True, text=True, timeout=10)
                if succeeds:
                    self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                else:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertIn("publisher.Update.versionCode", result.stderr)


if __name__ == "__main__":
    unittest.main()
