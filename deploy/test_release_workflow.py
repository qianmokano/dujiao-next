"""Verify fork tag naming and publication gates without publishing artifacts."""
import fnmatch
from pathlib import Path
import subprocess
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "deploy/verify-release-tag.sh"


class ReleaseWorkflowTest(unittest.TestCase):
    def workflow(self, name):
        return yaml.load((ROOT / ".github/workflows" / name).read_text(), Loader=yaml.BaseLoader)

    def test_valid_and_invalid_release_tags(self):
        for tag in ("v0.1.0-1", "v0.1.0-2", "v0.2.0-1", "v10.20.30-123"):
            with self.subTest(tag=tag):
                result = subprocess.run(["bash", str(SCRIPT), tag], capture_output=True)
                self.assertEqual(result.returncode, 0, result.stderr.decode())
        for args in ([], ["v0.1.0-1", "extra"], ["v0.1.0"], ["v0.1.0-kano.19"],
                     ["v0.1.0-rc1"], ["v0.1.0-0"], ["v0.1.0-01"], ["v0.1.0-1-extra"],
                     ["vv0.1.0-1"], ["0.1.0-1"], ["v0.1-1"]):
            with self.subTest(args=args):
                result = subprocess.run(["bash", str(SCRIPT), *args], capture_output=True)
                self.assertNotEqual(result.returncode, 0)

    def test_image_and_binary_releases_share_the_new_tag_rule(self):
        for name in ("publish-image.yml", "release.yml"):
            with self.subTest(workflow=name):
                patterns = self.workflow(name)["on"]["push"]["tags"]
                for tag in ("v0.1.0-1", "v0.1.0-2", "v0.2.0-1"):
                    self.assertTrue(any(fnmatch.fnmatchcase(tag, pattern) for pattern in patterns))
                for tag in ("v0.1.0", "v0.1.0-kano.19", "v0.1.0-rc1"):
                    self.assertFalse(any(fnmatch.fnmatchcase(tag, pattern) for pattern in patterns))

    def test_release_validation_precedes_tests_and_publication(self):
        workflow = self.workflow("publish-image.yml")
        self.assertEqual(workflow["jobs"]["verify-go"]["needs"], "verify-release")
        self.assertEqual(workflow["jobs"]["verify-frontends"]["needs"], "verify-release")
        self.assertEqual(workflow["jobs"]["publish"]["needs"], ["verify-go", "verify-frontends"])
        for name, job in (("publish-image.yml", "verify-release"), ("release.yml", "goreleaser")):
            steps = self.workflow(name)["jobs"][job]["steps"]
            self.assertEqual(steps[0]["with"]["fetch-depth"], "0")
            guard = next(step for step in steps if step.get("name") == "Verify release tag and source")
            self.assertEqual(guard["env"]["RELEASE_TAG"], "${{ github.ref_name }}")
            self.assertIn('bash deploy/verify-release-tag.sh "$RELEASE_TAG"', guard["run"])
            self.assertIn('git merge-base --is-ancestor "$GITHUB_SHA" origin/main', guard["run"])

    def test_image_version_and_platforms_remain_fixed(self):
        steps = self.workflow("publish-image.yml")["jobs"]["publish"]["steps"]
        build = next(step["with"] for step in steps if step.get("uses") == "docker/build-push-action@v6")
        self.assertEqual(build["tags"], "ghcr.io/qianmokano/dujiao-next:${{ github.ref_name }}")
        self.assertEqual(build["platforms"], "linux/amd64,linux/arm64")
        self.assertIn("APP_VERSION=${{ github.ref_name }}", build["build-args"])


if __name__ == "__main__":
    unittest.main()
