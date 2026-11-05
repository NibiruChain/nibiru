#!/usr/bin/env python3
"""Focused release-version and publication checks; no builds or network calls."""

import argparse
from contextlib import redirect_stdout
import copy
import importlib.util
import io
from pathlib import Path
import subprocess
import sys
import tempfile
import tomllib
import unittest
from unittest.mock import patch


sys.dont_write_bytecode = True
SCRIPTS = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("optimizer_release", SCRIPTS / "optimizer-release.py")
assert SPEC is not None and SPEC.loader is not None
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)
PACKAGE = release.configuration()


class ReleaseVersions(unittest.TestCase):
    def configuration(self, version):
        package = copy.deepcopy(PACKAGE)
        package["version"] = version
        return patch.object(release, "configuration", return_value=package)

    def test_stable_and_candidates_use_exact_slash_tags(self):
        for version in ("0.1.0", "1.2.3", "0.1.1-rc.1", "1.2.3-rc.12"):
            with self.subTest(version=version), self.configuration(version):
                self.assertEqual(release.version(), version)
                release.validate_tag(f"sai-wasm-optimizer/v{version}")
                for tag in (f"sai-wasm-optimizer-v{version}", f"v{version}", "sai-wasm-optimizer/v9.9.9"):
                    with self.assertRaises(ValueError):
                        release.validate_tag(tag)

    def test_unsupported_versions_are_rejected(self):
        for version in ("0.1", "01.1.0", "0.01.0", "0.1.00", "0.1.0-rc.0", "0.1.0-rc.01", "0.1.0-rc", "0.1.0-beta.1", "0.1.0+build"):
            with self.subTest(version=version), self.configuration(version):
                with self.assertRaises(ValueError):
                    release.version()

    def test_manifests_and_archives_keep_the_full_version(self):
        for version in ("0.1.0", "0.1.1-rc.1"):
            with self.subTest(version=version), self.configuration(version):
                manifest = tomllib.loads(release.install_manifest())["package"]
                self.assertEqual(manifest["version"], version)
                metadata = manifest["metadata"]["binstall"]
                self.assertIn("/releases/download/sai-wasm-optimizer/v{ version }/", metadata["pkg-url"])
                self.assertEqual(metadata["disabled-strategies"], ["quick-install", "compile"])
                for target in release.TARGETS:
                    self.assertEqual(release.filename(target), f"sai-wasm-optimizer-{version}-{target}.tar.gz")

    def test_publication_dry_runs_mark_only_candidates_as_prereleases(self):
        for version in ("0.1.0", "0.1.1-rc.1"):
            with self.subTest(version=version), self.configuration(version):
                with tempfile.TemporaryDirectory() as temporary:
                    output = io.StringIO()
                    args = argparse.Namespace(
                        tag=f"sai-wasm-optimizer/v{version}",
                        artifacts_dir=Path(temporary), run=False,
                    )
                    with patch.object(release, "verify", return_value=[]), patch.object(release.subprocess, "run") as remote:
                        with redirect_stdout(output):
                            release.publish(args)
                        remote.assert_not_called()
                    self.assertIn("--latest=false", output.getvalue())
                    self.assertEqual("--prerelease" in output.getvalue(), "-rc." in version)

    def test_installer_accepts_the_same_versions_before_dependency_checks(self):
        versions = {
            "0.1.0": 127, "0.1.1-rc.1": 127, "1.2.3-rc.12": 127,
            "01.1.0": 2, "0.1.0-rc.0": 2, "0.1.0-rc.01": 2,
            "0.1.0-beta.1": 2, "latest": 2,
        }
        for version, expected in versions.items():
            with self.subTest(version=version):
                result = subprocess.run(
                    ["/bin/bash", str(SCRIPTS / "install-wasm-optimizer.sh"), version],
                    env={"PATH": "/nonexistent"}, capture_output=True, text=True,
                )
                self.assertEqual(result.returncode, expected, result.stderr)


if __name__ == "__main__":
    unittest.main()
