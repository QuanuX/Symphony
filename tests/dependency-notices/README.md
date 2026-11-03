# Installed dependency and notice evidence

Run after the installed SDK/receipt gate:

```sh
sh tests/dependency-notices/run.sh /absolute/prefix /absolute/new-evidence \
  /absolute/optional-external-prefix /absolute/optional-qxctl
```

The last two paths may be absent for a matrix group. Absence is recorded; it is
not a successful qualification of that component. Set `GOMODCACHE` to the exact
cache used for the selected qxctl build (the MVP installed harness uses
`mvp-installed/go/modules`). Use its exact Go 1.26.5 toolchain. This harness makes
no network request, builds no native package, executes no selected native
artifact, and invokes no `ldd`. It reads ELF/Mach-O loader tables using `readelf`
or `otool`; unknown transitive runtime identities remain an explicit gate.

The evidence contains:

- Every selected installed receipt and exact hashes of its owned notices,
  executable and library files. A digest/size mismatch fails. Missing newly
  required curl/zstd notices or either DuckDB notice fails. Full receipt
  canonical/signature validation remains the preceding installed gate's job.
- The source dependency map, including exact resolved Go graphs, separated from
  actual linked modules reported by the selected Go executable's buildinfo.
  Replacement modules are refused. `go mod verify` checks the exact module cache
  before notice collection. The workflow first runs `go mod download` with the public checksum-verified proxy to populate the complete declared graph, including metadata for platform-excluded modules; this preparation does not add them to the linked-artifact inventory. Missing legal files for a linked module fails.
- Copies and hashes of actual Go module legal files, matching Go toolchain
  license/patent and standard-library vendor notices, and current main-source
  license. These retained files can be a superset of linked packages; their
  presence is not a license-compatibility determination.
- Actual private curl/zstd build selection and loader references if supplied;
  exact platform-specific PDFium archive/license hashes from the existing SHV
  qualification manifest. PDFium is caller-loaded, so this inventory does not
  pretend it is installed or redistributed with every SHV package.

`inventory.json` always keeps `publication_authorized=false`,
`release_ready=false` and publisher review pending. `notices/` contains retained
legal/evidence material; `logs/` contains bounded test and verification records.
A successful collector run means inventory preparation succeeded, not that
redistribution has been authorized. Do not upload module caches, SDKs, source
archives or vendor/private data as notice evidence.

The CMake regression configures with system/package-manager searches disabled,
installs the exact selected notices, and rejects altered, missing and symlinked
notice inputs. Go tests reject path escapes and symlinks and preserve legal-file
selection. The committed source notices are the exact existing curl COPYING and
zstd BSD LICENSE selections; no Homebrew installation or generic prefix/LICENSE
is consulted. Required publisher decisions are in
`cmake/notices/PUBLISHER-REVIEW.md`.
