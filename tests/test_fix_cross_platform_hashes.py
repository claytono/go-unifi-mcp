import contextlib
import importlib.machinery
import importlib.util
import io
import json
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch


SCRIPT_PATH = (
    Path(__file__).resolve().parent.parent / "scripts" / "fix-cross-platform-hashes"
)


def load_script_module():
    loader = importlib.machinery.SourceFileLoader(
        "fix_cross_platform_hashes", str(SCRIPT_PATH)
    )
    spec = importlib.util.spec_from_loader("fix_cross_platform_hashes", loader)
    if spec.loader is None:
        raise RuntimeError("Unable to load fix-cross-platform-hashes script")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class SourceHashRefreshTest(unittest.TestCase):
    def test_deduplicates_sources_shared_across_systems(self):
        module = load_script_module()
        shared = {
            "pname": "python-kacl",
            "url": "https://example.test/python-kacl.tar.gz",
            "hash": "sha256-old",
            "mode": "recursive",
        }
        sources = [
            {**shared, "system": "aarch64-darwin"},
            {**shared, "system": "x86_64-linux"},
            {
                "pname": "mcp-cli",
                "url": "https://example.test/mcp-cli",
                "hash": "sha256-cli",
                "mode": "flat",
                "system": "x86_64-linux",
            },
        ]

        entries = module.unique_source_entries(sources)

        self.assertEqual(2, len(entries))
        self.assertEqual(
            {
                "https://example.test/python-kacl.tar.gz",
                "https://example.test/mcp-cli",
            },
            {entry["url"] for entry in entries},
        )

    def test_prefetches_recursive_and_flat_sources(self):
        module = load_script_module()
        calls = []

        def fake_run_nix(args, check=True):
            calls.append(args)
            return subprocess.CompletedProcess(
                args,
                0,
                stdout=json.dumps({"hash": f"sha256-{len(calls)}"}),
            )

        with patch.object(module, "run_nix", side_effect=fake_run_nix):
            recursive = module.prefetch_url(
                "https://example.test/source.tar.gz", "recursive"
            )
            flat = module.prefetch_url("https://example.test/binary", "flat")

        self.assertEqual("sha256-1", recursive)
        self.assertEqual("sha256-2", flat)
        self.assertEqual(
            [
                "store",
                "prefetch-file",
                "--json",
                "--hash-type",
                "sha256",
                "--unpack",
                "https://example.test/source.tar.gz",
            ],
            calls[0],
        )
        self.assertEqual(
            [
                "store",
                "prefetch-file",
                "--json",
                "--hash-type",
                "sha256",
                "https://example.test/binary",
            ],
            calls[1],
        )

    def test_rejects_one_old_hash_for_different_source_contents(self):
        module = load_script_module()
        entries = [
            {
                "pname": "tool",
                "url": "https://example.test/one",
                "hash": "sha256-old",
                "mode": "flat",
                "system": "aarch64-linux",
            },
            {
                "pname": "tool",
                "url": "https://example.test/two",
                "hash": "sha256-old",
                "mode": "flat",
                "system": "x86_64-linux",
            },
        ]

        with (
            patch.object(
                module, "prefetch_url", side_effect=["sha256-one", "sha256-two"]
            ),
            contextlib.redirect_stdout(io.StringIO()),
        ):
            with self.assertRaisesRegex(
                ValueError, "One configured hash maps to multiple sources"
            ):
                module.find_hash_updates(entries)

    def test_reports_only_stale_hashes(self):
        module = load_script_module()
        entries = [
            {
                "pname": "current",
                "url": "https://example.test/current",
                "hash": "sha256-current",
                "mode": "flat",
                "system": "x86_64-linux",
            },
            {
                "pname": "stale",
                "url": "https://example.test/stale",
                "hash": "sha256-old",
                "mode": "recursive",
                "system": "x86_64-linux",
            },
        ]

        with (
            patch.object(
                module,
                "prefetch_url",
                side_effect=["sha256-current", "sha256-new"],
            ),
            contextlib.redirect_stdout(io.StringIO()),
        ):
            updates = module.find_hash_updates(entries)

        self.assertEqual({"sha256-old": "sha256-new"}, updates)

    def test_main_updates_a_stale_declared_source_hash(self):
        module = load_script_module()
        sources = [
            {
                "pname": "python-kacl",
                "url": "https://example.test/python-kacl.tar.gz",
                "hash": "sha256-old",
                "mode": "recursive",
                "system": "x86_64-linux",
            }
        ]

        with tempfile.TemporaryDirectory() as tempdir:
            flake_file = Path(tempdir) / "flake.nix"
            flake_file.write_text('hash = "sha256-old";\n')
            with (
                patch.object(module, "FLAKE_FILE", flake_file),
                patch.object(module, "get_all_package_sources", return_value=sources),
                patch.object(module, "prefetch_url", return_value="sha256-new"),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                module.main()

            self.assertEqual('hash = "sha256-new";\n', flake_file.read_text())


if __name__ == "__main__":
    unittest.main()
