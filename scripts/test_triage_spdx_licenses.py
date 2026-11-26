"""Offline tests for SPDX triage (synthetic data only)."""
import importlib.util
from pathlib import Path
import tempfile
import unittest


PATH = Path(__file__).with_name("triage_spdx_licenses.py")
spec = importlib.util.spec_from_file_location("triage_spdx_licenses", PATH)
triage = importlib.util.module_from_spec(spec)
spec.loader.exec_module(triage)


class SPDXTriageTests(unittest.TestCase):
    def test_missing_classification_fails_closed(self):
        data = {"spdxVersion": "SPDX-2.3", "packages": [
            {"name": "example", "SPDXID": "SPDXRef-a", "licenseDeclared": "NOASSERTION",
             "licenseConcluded": "MIT"},
            {"name": "example2", "SPDXID": "SPDXRef-b", "licenseDeclared": "MIT",
             "licenseConcluded": "MIT"},
            {"name": "example3", "SPDXID": "SPDXRef-c", "licenseDeclared": "LicenseRef-custom",
             "licenseConcluded": "MIT"}]}
        rows = triage.rows_from_spdx(data)
        self.assertEqual([r["classification"] for r in rows], [
            "REQUIRES_REVIEW", "REVIEW_RECORDED_NOT_APPROVED", "REQUIRES_REVIEW"])
        self.assertTrue(all(r["source_disposition"] == "PENDING" for r in rows))

    def test_invalid_and_empty_documents_rejected(self):
        for document in ({}, {"spdxVersion": "SPDX-2.3", "packages": []},
                         {"spdxVersion": "SPDX-3.0", "packages": [{"name": "x"}]},
                         {"spdxVersion": "SPDX-2.3", "packages": [None]}):
            with self.subTest(document=document):
                with self.assertRaises(ValueError):
                    triage.rows_from_spdx(document)

    def test_private_matrix_create_no_overwrite(self):
        rows = triage.rows_from_spdx({"spdxVersion": "SPDX-2.3",
                                       "packages": [{"name": "synthetic"}]})
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "matrix.csv"
            triage.write_matrix(target, rows)
            self.assertIn("REQUIRES_REVIEW", target.read_text())
            with self.assertRaises(ValueError):
                triage.write_matrix(target, rows)

    def test_refuses_output_within_checkout(self):
        with self.assertRaises(ValueError):
            triage.write_matrix(PATH.parent / "matrix.csv", [])


if __name__ == "__main__":
    unittest.main()
