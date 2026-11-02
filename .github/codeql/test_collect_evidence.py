import importlib.util
import json
import pathlib
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("collector", pathlib.Path(__file__).with_name("collect_evidence.py"))
collector = importlib.util.module_from_spec(spec)
spec.loader.exec_module(collector)


class EvidenceTests(unittest.TestCase):
    def test_missing_analysis_cannot_be_a_clean_scan(self):
        with tempfile.TemporaryDirectory() as root:
            with self.assertRaisesRegex(ValueError, "absent"):
                collector.summarize(pathlib.Path(root), "a" * 40, "/language:c-cpp/os:ubuntu-24.04")

    def test_zero_results_keeps_diagnostics_and_no_clearance(self):
        with tempfile.TemporaryDirectory() as root:
            path = pathlib.Path(root) / "cpp.sarif"
            path.write_text(json.dumps({"version": "2.1.0", "runs": [{"results": [], "invocations": [{"toolExecutionNotifications": [{"level": "warning", "message": {"text": "missing header"}}]}], "properties": {"metricResults": [{"value": 575}]}}]}))
            report = collector.summarize(pathlib.Path(root), "a" * 40, "/language:c-cpp/os:ubuntu-24.04")
            self.assertEqual(report["security_clearance"], "not_asserted")
            run = report["original_sarif"][0]["runs"][0]
            self.assertEqual(run["diagnostic_notifications"][0]["message"]["text"], "missing header")
            self.assertEqual(run["properties"]["metricResults"][0]["value"], 575)
            self.assertFalse(run["provenance_present_in_original_sarif"])
            self.assertEqual(run["coverage_completeness"], "not_asserted")
            self.assertEqual(len(report["original_sarif"][0]["sha256"]), 64)

    def test_other_category_cannot_replace_selected_evidence(self):
        with tempfile.TemporaryDirectory() as root:
            (pathlib.Path(root) / "cpp.sarif").write_text(json.dumps({"version": "2.1.0", "runs": [{"automationDetails": {"id": "/language:c-cpp/os:macos-15-intel/"}}]}))
            with self.assertRaisesRegex(ValueError, "category differs"):
                collector.summarize(pathlib.Path(root), "a" * 40, "/language:c-cpp/os:ubuntu-24.04")

    def test_other_source_revision_is_rejected(self):
        with tempfile.TemporaryDirectory() as root:
            (pathlib.Path(root) / "cpp.sarif").write_text(json.dumps({"version": "2.1.0", "runs": [{"versionControlProvenance": [{"revisionId": "b" * 40}]}]}))
            with self.assertRaisesRegex(ValueError, "differs"):
                collector.summarize(pathlib.Path(root), "a" * 40, "/language:c-cpp/os:ubuntu-24.04")


if __name__ == "__main__":
    unittest.main()
