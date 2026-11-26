# Route 1 — private SPDX license triage (Issue #89)

**Status: EVIDENCE COLLECTION ONLY — NOT LICENSE CLEARANCE OR PUBLICATION APPROVAL.**

This supplements [the exact-version Go license inventory](dependency-license-review.md) and [pre-publication NO-GO](pre-publication-go-no-go.md). It does **not** replace their 28 exact-version upstream checks or the existing 35-package image SBOM inspection (29 `NOASSERTION` records in the historical snapshot). A new scan may have different counts.

## Procedure on a trusted owner workstation

1. Obtain a fresh *private* SPDX 2.x JSON SBOM from the read-only **Private Release Validation / Local Image SBOM** run. Download privately before artifact retention expires. Do not paste SBOM content or private source metadata into issues or CI logs.
2. Save the downloaded file outside your repository checkout, ideally in a restricted local directory. Record the immutable source SHA and workflow run ID separately. A SBOM generated from another commit is **not** evidence for the current SHA.
3. Run from the trusted private checkout:

   ```powershell
   py -3 scripts/triage_spdx_licenses.py "C:\private-audit\candidate-sbom.spdx.json" --matrix "C:\private-audit\license-review.csv"
   ```

   Linux/macOS equivalent: `python3 scripts/triage_spdx_licenses.py /private/audit/candidate-sbom.spdx.json --matrix /private/audit/license-review.csv`.

4. The CLI prints only aggregate package/classification counts and SHA-256 of the supplied SBOM. If requested, it creates a local CSV **outside this worktree**, refusing to overwrite previous evidence. Keep the CSV private: package metadata and reviewer annotations may reveal sensitive source information.
5. For **each** row, compare the exact component version and source/package license text; add evidence location, all relevant obligations, and separate `source_disposition` / `image_disposition` conclusions **only after review**. Verify `filesAnalyzed`, dual/custom license expressions, copyleft/base-image obligations, and applicable upstream NOTICE files. Do not assume `MIT` or `Apache-2.0` scanner labels alone discharge attribution requirements.
6. Independently audit any copied/generated/vendor source, historical attachments and ownership terms. Retain sensitive contracts, employer identifiers and findings solely outside GitHub.
7. A reviewer must explicitly reconcile the complete matrix, exact image contents, source tree, original license texts, unresolved findings, and artifact SHA before Issue #89 can be considered cleared.

### Classification semantics

- `REQUIRES_REVIEW`: either SPDX field missing/`NONE`/`NOASSERTION`, or a `LicenseRef-` expression appears.
- `REVIEW_RECORDED_NOT_APPROVED`: both scanner fields contain nonempty standard-looking expressions. **This is still not a clearance.**
- All source and image disposition fields default to **PENDING**, including for apparently conventional SPDX expressions.
- The script deliberately does **not** infer the legal implications of static linking, vendoring, Go build tags, package selection, or container distribution. It is not a replacement for independent provenance and licensing analysis.

### Audit boundaries

The script is read-only with respect to GitHub, makes **no network calls**, does not run in CI against real SBOM data, and does not publish files or secrets. CI executes only synthetic unit tests. Never commit completed CSV matrices, actual SBOMs containing sensitive metadata, third-party license bundles before clearance, or any personal/employer search pattern.

**Exit:** Issue #89 and parent #80 remain open; public source, GHCR images, tags, history edits and releases remain unauthorized. The owner-run historical Actions log screen (#87) is a separate confidentiality gate.
