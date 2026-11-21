#!/usr/bin/env python3
"""Tests for the release-version guard: scripts/release-refs.py preflight,
the release workflow that has to call it, and the Makefile version string.

Release run #140 was started from the Actions tab with the dispatch form's
placeholder still in the box. It built every artifact as "v0.0.0", published
them as a pre-release and tagged the v1.4.2 commit; `git describe` then named
every source build of main v0.0.0-N-g<sha>. Each test below fails on the tree
as it was that day.

Run with `make release-refs-test` (standard library only; needs git, and make
for the Makefile test).
"""
from __future__ import annotations

import contextlib
import importlib.util
import io
import os
import pathlib
import re
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
SCRIPT = ROOT / "scripts" / "release-refs.py"
WORKFLOW = ROOT / ".github" / "workflows" / "release.yml"

sys.dont_write_bytecode = True  # no scripts/__pycache__ left in the working tree
_spec = importlib.util.spec_from_file_location("release_refs", SCRIPT)
release_refs = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(release_refs)

# Keep the developer's git configuration (signing, hooks, templates) out of
# the throwaway repositories.
GIT = [
    "git",
    "-c", "user.name=release-refs-test",
    "-c", "user.email=release-refs-test@example.invalid",
    "-c", "commit.gpgsign=false",
    "-c", "tag.gpgsign=false",
    "-c", "init.defaultBranch=main",
    "-c", "protocol.file.allow=always",
]


class Repo:
    """A throwaway git repository with empty commits and tags."""

    def __init__(self, path: pathlib.Path):
        self.path = path
        path.mkdir(parents=True, exist_ok=True)
        self.git("init", "--quiet")

    def git(self, *args: str) -> str:
        out = subprocess.run(GIT + list(args), cwd=self.path, capture_output=True, text=True)
        if out.returncode != 0:
            raise AssertionError(f"git {' '.join(args)} failed: {out.stderr}")
        return out.stdout.strip()

    def commit(self, message: str) -> str:
        self.git("commit", "--quiet", "--allow-empty", "-m", message)
        return self.git("rev-parse", "HEAD")

    def tag(self, name: str, annotated: bool = False) -> None:
        if annotated:
            self.git("tag", "-a", name, "-m", name)
        else:
            self.git("tag", name)


def preflight(repo: Repo, version: str, **kw) -> tuple[int, str]:
    """Run preflight against a throwaway repository; (exit status, stderr)."""
    err = io.StringIO()
    with contextlib.redirect_stderr(err), contextlib.redirect_stdout(io.StringIO()):
        status = release_refs.preflight(version, repo=repo.path, **kw)
    return status, err.getvalue()


class PreflightTest(unittest.TestCase):
    def setUp(self):
        self.tmp = pathlib.Path(tempfile.mkdtemp(prefix="release-refs-test-"))
        self.addCleanup(shutil.rmtree, self.tmp, ignore_errors=True)

    def released(self, *versions: str) -> Repo:
        """A repository with one commit per version, each tagged with it."""
        repo = Repo(self.tmp / "repo")
        for v in versions:
            repo.commit(f"release: {v}")
            repo.tag(v)
        return repo

    def assertRefused(self, repo: Repo, version: str, why: str, **kw) -> None:
        status, err = preflight(repo, version, **kw)
        self.assertEqual(status, 1, f"{version!r} was accepted")
        self.assertIn(why, err)

    def assertAccepted(self, repo: Repo, version: str, **kw) -> None:
        status, err = preflight(repo, version, **kw)
        self.assertEqual(status, 0, f"{version!r} was refused: {err}")

    def test_placeholder_version_is_refused(self):
        # Run #140: main at the v1.4.2 release commit, "v0.0.0" in the box.
        repo = self.released("v1.4.1", "v1.4.2")
        self.assertRefused(repo, "v0.0.0", "not newer than v1.4.2")

    def test_repository_as_the_incident_left_it(self):
        # v0.0.0 and v1.4.2 now sit on the same commit.
        repo = self.released("v1.4.1", "v1.4.2")
        repo.tag("v0.0.0")
        # The placeholder is still refused, although its tag is on this commit.
        self.assertRefused(repo, "v0.0.0", "not newer than v1.4.2")
        # Re-running the release that commit really is stays possible.
        self.assertAccepted(repo, "v1.4.2")
        repo.commit("a fix")
        self.assertAccepted(repo, "v1.4.3")
        # From the next commit on, v1.4.2 names a different commit.
        self.assertRefused(repo, "v1.4.2", "already exists on another commit")

    def test_version_must_move_forward(self):
        repo = self.released("v1.4.1", "v1.4.2")
        repo.commit("a fix")
        self.assertRefused(repo, "v1.4.1", "already exists on another commit")
        self.assertRefused(repo, "v1.4.0", "not newer than v1.4.2")
        self.assertRefused(repo, "v1.3.10", "not newer than v1.4.2")
        self.assertRefused(repo, "v1.4.02", "not newer than v1.4.2")
        self.assertRefused(repo, "v0.1.0", "not newer than v1.4.2")
        for ok in ("v1.4.3", "v1.5.0", "v2.0.0", "v1.4.3-rc.1"):
            self.assertAccepted(repo, ok)

    def test_malformed_version_is_refused(self):
        repo = self.released("v1.4.2")
        repo.commit("a fix")
        for bad in ("", "1.4.3", "v1.4", "v1.4.3.1", "latest", "v1.4.3 ", "v1.4.3\n", "v1.4.3;id", "$(id)", "main"):
            self.assertRefused(repo, bad, "not a vX.Y.Z version")

    def test_patch_release_from_an_older_line(self):
        # docs/release.md "Patch releases": branch off the old tag, cherry-pick,
        # tag from the branch. Newer releases exist, but not in this history.
        repo = self.released("v1.3.8", "v1.3.9")
        repo.git("branch", "release/v1.3")
        repo.commit("feature work")
        repo.tag("v1.4.0", annotated=True)
        repo.git("checkout", "--quiet", "release/v1.3")
        repo.commit("cherry-picked fix")
        self.assertAccepted(repo, "v1.3.10")
        self.assertRefused(repo, "v1.3.9", "already exists on another commit")
        self.assertRefused(repo, "v1.3.7", "not newer than v1.3.9")
        self.assertRefused(repo, "v1.4.0", "already exists on another commit")
        # And the same version checked against main is refused there.
        self.assertRefused(repo, "v1.3.10", "not newer than v1.4.0", commit="main")

    def test_pushed_tag_is_its_own_release(self):
        # The tag-push path: the tag exists already, on the commit released.
        repo = self.released("v1.4.2")
        repo.commit("a fix")
        repo.tag("v1.4.3", annotated=True)
        self.assertAccepted(repo, "v1.4.3")
        repo.tag("v1.4.1")  # an older number pushed by mistake
        self.assertRefused(repo, "v1.4.1", "not newer than v1.4.3")

    def test_prerelease_order(self):
        repo = self.released("v1.4.2", "v1.5.0-rc.9")
        repo.commit("a fix")
        self.assertAccepted(repo, "v1.5.0-rc.10")  # numerically, not as text
        self.assertAccepted(repo, "v1.5.0")
        self.assertRefused(repo, "v1.5.0-rc.2", "not newer than v1.5.0-rc.9")
        self.assertRefused(repo, "v1.5.0-beta.1", "not newer than v1.5.0-rc.9")
        self.assertRefused(repo, "v1.4.9", "not newer than v1.5.0-rc.9")

    def test_shallow_checkout_is_refused(self):
        # actions/checkout's default depth is 1: the history is not there to
        # be checked, and the guard must not pass for that reason.
        src = self.released("v1.4.1", "v1.4.2")
        src.commit("a fix")
        clone = self.tmp / "shallow"
        subprocess.run(
            GIT + ["clone", "--quiet", "--depth", "1", src.path.as_uri(), str(clone)],
            check=True, capture_output=True,
        )
        shallow = Repo.__new__(Repo)
        shallow.path = clone
        self.assertRefused(shallow, "v1.4.3", "shallow")
        self.assertRefused(shallow, "v0.0.0", "shallow")

    def test_checkout_without_tags_is_refused(self):
        repo = Repo(self.tmp / "repo")
        repo.commit("first")
        self.assertRefused(repo, "v0.0.0", "no earlier release tag is visible")
        self.assertRefused(repo, "v1.4.3", "no earlier release tag is visible")
        self.assertAccepted(repo, "v0.1.0", first_release=True)

    def test_command_line(self):
        # The workflow calls the script, not the function.
        out = subprocess.run(
            [sys.executable, str(SCRIPT), "preflight", "not-a-version"],
            capture_output=True, text=True,
        )
        self.assertEqual(out.returncode, 1, out.stderr)
        self.assertIn("preflight: refusing to release 'not-a-version'", out.stderr)


class SemverKeyTest(unittest.TestCase):
    def test_precedence(self):
        # semver.org §11's own example, plus the stable/prerelease boundary.
        order = [
            "v1.0.0-alpha", "v1.0.0-alpha.1", "v1.0.0-alpha.beta", "v1.0.0-beta",
            "v1.0.0-beta.2", "v1.0.0-beta.11", "v1.0.0-rc.1", "v1.0.0",
            "v1.0.1", "v1.0.10", "v1.1.0", "v2.0.0-rc.1", "v2.0.0",
        ]
        self.assertEqual(sorted(reversed(order), key=release_refs.semver_key), order)


def job_blocks(text: str) -> dict[str, str]:
    """release.yml's jobs, by id: each job's own lines."""
    lines = text.split("\n")
    start = lines.index("jobs:")
    jobs: dict[str, list[str]] = {}
    current = None
    for line in lines[start + 1:]:
        m = re.match(r"^  ([A-Za-z0-9_-]+):\s*$", line)
        if m:
            current = m.group(1)
            jobs[current] = []
        elif current is not None:
            jobs[current].append(line)
    return {k: "\n".join(v) for k, v in jobs.items()}


class WorkflowTest(unittest.TestCase):
    """release.yml has to run the guard before anything is built."""

    def setUp(self):
        self.text = WORKFLOW.read_text(encoding="utf-8")
        self.jobs = job_blocks(self.text)

    def has(self, pattern: str, text: str, msg: str) -> None:
        # assertRegex would print the whole workflow on a failure.
        self.assertTrue(re.search(pattern, text, re.M), msg)

    def lacks(self, pattern: str, text: str, msg: str) -> None:
        self.assertFalse(re.search(pattern, text, re.M), msg)

    def test_dispatch_form_has_no_default_version(self):
        # The placeholder default is what was published. With no default the
        # form refuses to start a run until a version is typed.
        m = re.search(r"^  workflow_dispatch:\n((?:    .*\n|\n)+)", self.text, re.M)
        self.assertIsNotNone(m, "release.yml has no workflow_dispatch block")
        block = m.group(1)
        self.has(r"^      version:$", block, "the dispatch form has no version input")
        self.has(r"required:\s*true", block, "the version input must be required")
        self.lacks(r"^\s*default:", block, "the version input must not have a default")

    def test_preflight_job_runs_the_guard_on_full_history(self):
        self.assertTrue("preflight" in self.jobs, "release.yml has no preflight job")
        job = self.jobs["preflight"]
        self.has(r"scripts/release-refs\.py preflight", job, "the preflight job does not run the guard")
        self.has(r"fetch-depth:\s*0\b", job, "the guard needs the full history and its tags")
        self.lacks(r"^    needs:", job, "the guard must not wait for a build")
        # The typed version reaches the shell through the environment, never
        # spliced into the script text.
        self.has(r"\$\{\{\s*github\.event\.inputs\.version\s*\}\}", job, "the guard never reads the typed version")
        run = job.split("run: |", 1)[1]
        self.assertFalse("github.event.inputs.version" in run, "the typed version is spliced into the shell script")

    def test_nothing_is_built_or_published_before_the_guard(self):
        for job in ("windows", "linux", "macos"):
            self.assertTrue(job in self.jobs, f"release.yml has no {job} job")
            self.has(r"^    needs:.*\bpreflight\b", self.jobs[job],
                     f"the {job} build job does not wait for the preflight job")
        # Every job that is not a build job hangs off one that is.
        builders = {"windows", "linux", "macos"}
        for job, block in self.jobs.items():
            if job == "preflight" or job in builders:
                continue
            self.has(r"^    needs:", block, f"the {job} job waits for nothing, so it runs unchecked")
        self.has(r"^    needs:.*\bwindows\b.*\blinux\b.*\bmacos\b", self.jobs["release"],
                 "the publish job does not wait for every build job")
        self.has(r"^    needs:.*\brelease\b", self.jobs["version-refs"],
                 "the version-refs job does not wait for the publish job")

    def test_tag_is_created_on_the_commit_that_was_checked(self):
        # Without target_commitish GitHub creates a missing tag on the default
        # branch's head at publish time, not on the commit that was checked
        # and built.
        self.has(r"target_commitish:\s*\$\{\{\s*github\.sha\s*\}\}", self.jobs["release"],
                 "the publish step does not pin the tag to the released commit")


def make_has_eval() -> bool:
    """GNU make 3.82 added --eval; macOS still ships 3.81."""
    if not shutil.which("make"):
        return False
    out = subprocess.run(["make", "--version"], capture_output=True, text=True).stdout
    m = re.search(r"GNU Make (\d+)\.(\d+)", out)
    return bool(m) and (int(m.group(1)), int(m.group(2))) >= (3, 82)


@unittest.skipUnless(make_has_eval(), "needs GNU make 3.82 or later")
class MakefileVersionTest(unittest.TestCase):
    """`make build` must not name a source build after the stray v0.0.0 tag."""

    def version(self, repo: Repo) -> str:
        env = {k: v for k, v in os.environ.items()
               if k not in ("VERSION", "MAKEFLAGS", "MFLAGS", "MAKEOVERRIDES", "MAKELEVEL")}
        out = subprocess.run(
            ["make", "-s", "-f", str(ROOT / "Makefile"),
             "--eval", "release-refs-print-version: ; @echo $(VERSION)",
             "release-refs-print-version"],
            cwd=repo.path, env=env, capture_output=True, text=True,
        )
        self.assertEqual(out.returncode, 0, out.stderr)
        return out.stdout.strip()

    def test_describe_skips_the_placeholder_tag(self):
        tmp = pathlib.Path(tempfile.mkdtemp(prefix="release-refs-test-"))
        self.addCleanup(shutil.rmtree, tmp, ignore_errors=True)
        repo = Repo(tmp / "repo")
        repo.commit("release: v1.4.2")
        repo.tag("v0.0.0")  # both tags on one commit, as on origin
        repo.tag("v1.4.2")
        self.assertEqual(self.version(repo), "v1.4.2")
        repo.commit("a fix")
        sha = repo.git("rev-parse", "--short", "HEAD")
        self.assertEqual(self.version(repo), f"v1.4.2-1-g{sha}")


if __name__ == "__main__":
    unittest.main()
