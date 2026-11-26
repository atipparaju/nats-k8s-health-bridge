"""Offline tests: no GitHub requests, credentials, or real company identifiers."""
import io
import re
import unittest
import urllib.error
import urllib.request
import zipfile
from unittest import mock

import audit_actions_log_disclosure as audit


def example_zip(files):
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w") as out:
        for path, value in files.items():
            out.writestr(path, value)
    return buffer.getvalue()


class ArchiveTests(unittest.TestCase):
    def setUp(self):
        self.pattern = re.compile("ForbiddenMarkerAlpha", re.IGNORECASE)

    def test_clean_archive(self):
        data = example_zip({"steps/one.txt": "ordinary build output"})
        self.assertFalse(audit.archive_matches(data, self.pattern))

    def test_matched_text(self):
        data = example_zip({"steps/one.txt": "log contains ForbiddenMarkerAlpha"})
        self.assertTrue(audit.archive_matches(data, self.pattern))

    def test_matched_archive_filename(self):
        data = example_zip({"ForbiddenMarkerAlpha.txt": "ordinary text"})
        self.assertTrue(audit.archive_matches(data, self.pattern))

    def test_empty_or_invalid_archive_is_unverified(self):
        for data in [b"not a zip", example_zip({})]:
            with self.assertRaises(audit.AuditError):
                audit.archive_matches(data, self.pattern)

    def test_oversized_member_fails_closed(self):
        data = example_zip({"large.txt": "tiny"})
        with mock.patch.object(audit, "MAX_ENTRY_BYTES", 1):
            with self.assertRaises(audit.AuditError):
                audit.archive_matches(data, self.pattern)


class ApiBoundaryTests(unittest.TestCase):
    def test_trusted_signed_download_hosts(self):
        audit.validate_download_host("https://pipelines.actions.githubusercontent.com/a?sig=secret")
        audit.validate_download_host("https://storage.blob.core.windows.net/a")
        for value in ["http://pipelines.actions.githubusercontent.com/a",
                      "https://example.invalid/a", "https://githubusercontent.com.evil.invalid/a",
                      "https://username:password@pipelines.actions.githubusercontent.com/a"]:
            with self.assertRaises(audit.AuditError):
                audit.validate_download_host(value)

    def test_authenticated_redirect_not_forwarded_to_signed_url(self):
        signed_url = "https://pipelines.actions.githubusercontent.com/download?sig=hidden"
        headers = {"Location": signed_url}
        http_error = urllib.error.HTTPError(
            "https://api.github.com/repos/a/b/actions/runs/42/attempts/1/logs",
            302, "Found", headers, None,
        )
        signed_data = example_zip({"job.txt": "ordinary content"})
        mock_response = mock.MagicMock()
        mock_response.read.return_value = signed_data
        mock_response.__enter__.return_value = mock_response
        with mock.patch.object(audit.NO_REDIRECT, "open", side_effect=http_error), \
             mock.patch.object(urllib.request, "urlopen", return_value=mock_response) as request:
            result = audit.download_archive("a/b", "secret-test-token", 42, 1)
        self.assertEqual(result, signed_data)
        url_request = request.call_args.args[0]
        self.assertNotIn("Authorization", url_request.headers)
        self.assertNotIn("secret-test-token", str(url_request.headers))

    def test_list_runs_rejects_missing_count(self):
        with mock.patch.object(audit, "api_json", return_value={
            "total_count": 3,
            "workflow_runs": [{"id": 1, "run_attempt": 1}],
        }):
            with self.assertRaises(audit.AuditError):
                audit.list_runs("a/b", "test-token")

    def test_inspect_only_prints_counts_and_run_ids(self):
        runs = [{"id": 42, "run_attempt": 2, "name": "build"},
                {"id": 43, "run_attempt": 1, "name": "build"}]
        docs = {(42, 1): example_zip({"job.txt": "ForbiddenMarkerAlpha"}),
                (42, 2): example_zip({"job.txt": "ordinary text"}),
                (43, 1): example_zip({"job.txt": "ordinary text"})}
        def download(repo, token, run_id, attempt):
            return docs[(run_id, attempt)]
        stdout = io.StringIO()
        with mock.patch.object(audit, "list_runs", return_value=runs), \
             mock.patch.object(audit, "download_archive", side_effect=download), \
             mock.patch("sys.stdout", stdout):
            status = audit.inspect("a/b", "token", re.compile("ForbiddenMarkerAlpha"))
        output = stdout.getvalue()
        self.assertEqual(status, 2)
        self.assertIn("Runs with potential private-term matches: 1", output)
        self.assertIn("42", output)
        self.assertNotIn("ForbiddenMarkerAlpha", output)
        self.assertNotIn("secret-test-token", output)

    def test_inspect_unavailable_logs_fail_closed(self):
        with mock.patch.object(audit, "list_runs", return_value=[
            {"id": 9, "run_attempt": 1, "name": "build"}
        ]), mock.patch.object(audit, "download_archive",
             side_effect=audit.AuditError("unavailable")):
            with mock.patch("sys.stdout", io.StringIO()):
                status = audit.inspect("a/b", "token", re.compile("MissingMarker"))
        self.assertEqual(status, 2)


if __name__ == "__main__":
    unittest.main()
