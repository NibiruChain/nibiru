"""Exercise repository/image selection without release downloads or publication."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("publish-release-image.sh")


class ReleaseImageRepositoryTests(unittest.TestCase):
    def dry_run(self, repository, *args, extra_env=None):
        env = {**os.environ, "GITHUB_REPOSITORY": repository}
        env.update(extra_env or {})
        return subprocess.run(
            ["bash", str(SCRIPT), "--release-tag", "v2.23.0", "--version", "2.23.0", "--dry-run", *args],
            env=env, text=True, capture_output=True, check=True,
        ).stderr

    def test_both_organization_names(self):
        for owner in ("NibiruChain", "SaiDotFun"):
            with self.subTest(owner=owner):
                output = self.dry_run(f"{owner}/nibiru")
                self.assertIn(f"release {owner}/nibiru@v2.23.0", output)
                self.assertIn(f"ghcr.io/{owner.lower()}/nibiru:2.23.0", output)

    def test_repository_override_selects_corresponding_image(self):
        output = self.dry_run("NibiruChain/nibiru", "--repo", "SaiDotFun/nibiru")
        self.assertIn("release SaiDotFun/nibiru@v2.23.0 for ghcr.io/saidotfun/nibiru:2.23.0", output)

    def test_explicit_image_override(self):
        output = self.dry_run("SaiDotFun/nibiru", "--image", "ghcr.io/recovery/nibiru")
        self.assertIn("for ghcr.io/recovery/nibiru:2.23.0", output)

    def test_local_checkout_uses_canonical_github_repository(self):
        with tempfile.TemporaryDirectory() as directory:
            gh = Path(directory) / "gh"
            gh.write_text("#!/bin/sh\n[ \"$*\" = 'repo view --json nameWithOwner --jq .nameWithOwner' ] || exit 99\nprintf 'SaiDotFun/nibiru\\n'\n")
            gh.chmod(0o755)
            output = self.dry_run("", extra_env={"PATH": directory + os.pathsep + os.environ["PATH"]})
            self.assertIn("release SaiDotFun/nibiru@v2.23.0 for ghcr.io/saidotfun/nibiru:2.23.0", output)


if __name__ == "__main__":
    unittest.main()
