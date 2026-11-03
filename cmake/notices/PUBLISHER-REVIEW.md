# Selected dependency material and publisher decisions

This is a preparation record for native/Conda candidates, not redistribution
approval or a legal conclusion. Exact artifact identity, source commit/tree,
platform, dependency closure and publisher authority still belong to the release
record. A successful build or collector cannot supply that authority. No new
vendor SDK, data entitlement, license option or publication destination is chosen
here.

| Component | Existing selected version and linkage | Material retained or required |
|---|---|---|
| Symphony | Exact candidate source commit; native static/shared libraries and Go executables | Repository AGPL-3.0 text; publisher must provide matching corresponding source, modifications and the build/install material for the distributed work, and review applicable network-use obligations. A source URL to an unrelated head is insufficient. |
| curl | 8.7.1; current private recipe builds shared libcurl | Exact `COPYING` is pinned in `sources.json` and installed with direct SDK consumers and SQAV request. Preserve its copyright/permission notice. Its version alone does not identify TLS, compression, SSH or HTTP-library closure. |
| zstd | 1.5.7; `zstd::libzstd_static` in Databento reference, then linked into request | Exact upstream BSD `LICENSE`, already selected by the prior build, is pinned and receipt-owned. Its alternate `COPYING` is not silently substituted. Preserve notice/conditions/disclaimer with binary material; review final static consumers' notice closure. |
| TLS / platform runtime | macOS recipe selects Secure Transport; Linux selects discovered OpenSSL. Exact OS/patch and transitive library identity are build-specific | Capture CMake TLS selection and actual loader references. Before release resolve each referenced file/package, exact version, license, notice and any source obligation. OpenSSL 3+ uses Apache-2.0; older releases have different terms. A macOS system framework is recorded as an OS dependency, not silently copied into the bundle. |
| DuckDB | 1.5.5, commit `d8cdaa33fda8df955cc76ef58a280f68f4cd43fa`; exact shared slices per connector provenance | Both connector receipts retain exact DuckDB MIT notice and the full 20-entry third-party appendix from the official source archive. Keep the exact upstream source archive/build provenance associated with the selected binary; do not replace the appendix with the top-level MIT notice alone. |
| PDFium qualification selection | 158.0.8125.0 (`chromium/8125`), V8/XFA disabled; independent `bblanchon/pdfium-binaries` recipe commit `4bdb7acd45b39ef6a7fc2a5b4c83f0398e45c9a6`; runtime-loaded, caller-selected path/digest | Existing `tests/shv-release-qualification/dependencies.json` pins four archives and every supplied platform-specific license file. Keep top-level LICENSE, `licenses/*`, VERSION, args.gn and build/source provenance together if the publisher decides to distribute it. Review each embedded component's exact selected license/NOTICE/source requirements; the PDFium BSD notice alone is insufficient. Test fixture PDFs stay separate. |
| nlohmann/json | Vendored 3.12.0, header compiled into native owners; installed foundation header | Exact vendored MIT notice retained by foundation and existing consumers; final distribution inventory must include it even when only a statically linked consumer is shipped. Header hash is bound in the dependency map. |
| Go runtime/toolchain | 1.26.5 in the selected recipe; runtime/standard-library code linked into executables | Retain the matching Go LICENSE/PATENTS and relevant standard-library vendor material. Collector checks its source VERSION against executable buildinfo. Host C/OS libraries under CGO remain separately inventoried loader dependencies. |
| Go modules | Exact resolved graph in `cmake/mvp-dependencies.json`; actual selected executable buildinfo determines linked module versions/sums | Collector copies exact available LICENSE/COPYING/NOTICE/PATENTS/COPYRIGHT files after module verification. Resolved test/tool-only modules are not labeled linked. Review nested third-party notices and actual linked package scope; do not infer compatibility merely because all files were found. |

## Explicit publisher review record still required

For each exact candidate, record the publisher/reviewer identity and decision,
with evidence for these concrete choices:

1. Which native executables, SDK archives/headers, Go executables and optional
   external runtimes are actually in the archive/Conda package? Which are system
   dependencies, and which are shipped? Bind paths and hashes to the candidate
   inventory, not a generic dependency list.
2. Does that chosen payload include the complete matching notice set, including
   statically linked/transitively embedded material, platform-specific PDFium and
   DuckDB appendices, and Go runtime/module notices? Resolve every unclassified
   dynamic library before approving the closure. Auto-discovered curl libraries
   may differ across build hosts.
3. Where is the corresponding source/build/install material for the exact
   Symphony candidate and any selected dependency whose terms require it?
   Retain original notices, attribution, changes and required NOTICE/source
   material. Verify that the chosen source delivery method matches the actual
   release mechanism before upload; a checksum/signature does not answer this.
4. Is the optional decoder/SDK/plugin/data payload actually authorized? IBKR SDK
   10.45 and Decimal glue remain externally supplied and separately reviewed;
   no SDK is downloaded or included by this work. Databento/FRED/news credentials
   and paid/private data are excluded. Public Databento wire fixtures retain
   their own Apache-2.0 attribution under the owner's test directory. Their
   permissive fixture license is not a market-data entitlement.
5. Record approval or explicit rejection in the existing release-governance
   evidence, with unresolved items retained. This document and the collector's
   `publisher_review=pending` cannot be used as approval.

## Primary material

- [curl 8.7.1 notice](https://raw.githubusercontent.com/curl/curl/curl-8_7_1/COPYING)
- [zstd 1.5.7 selected BSD notice](https://raw.githubusercontent.com/facebook/zstd/v1.5.7/LICENSE)
- [DuckDB exact-source license](https://raw.githubusercontent.com/duckdb/duckdb/d8cdaa33fda8df955cc76ef58a280f68f4cd43fa/LICENSE) and connector `DUCKDB-PROVENANCE.json` source appendix pins
- [PDFium source and license](https://pdfium.googlesource.com/pdfium/+/refs/heads/main/LICENSE), with the exact independently built archive/recipe identities in the SHV manifest
- [nlohmann/json v3.12.0 license](https://github.com/nlohmann/json/blob/v3.12.0/LICENSE.MIT)
- [Go license](https://go.dev/LICENSE) and exact selected toolchain/module source files
- [OpenSSL version-dependent license scope](https://openssl-library.org/source/license/index.html)
- [AGPL-3.0 source and network provisions](https://www.gnu.org/licenses/agpl-3.0.html), with the exact repository LICENSE retained in the candidate

Primary curl, zstd, Go and OpenSSL material checked 2026-11-01. Exact local source
notice hashes and source archive hashes are recorded in `sources.json`; live
project pages never replace the exact candidate's pinned material.
