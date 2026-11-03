# First bounded source run

This guide builds the current development CLI, not a released download. Use a reviewed immutable source revision. Required release profiles are macOS 15 ARM64, Ubuntu 24.04 amd64/ARM64 and Ubuntu 26.04 amd64. Intel macOS is supplemental development evidence. Ubuntu ARM64 is accepted for the sprint; physical DGX Spark/DGX OS, GPU behavior and native Windows are not qualified.

Install the exact Go version from `tools/qxctl/go.mod` (currently 1.26.5) through your existing trusted toolchain process. Full native qualification additionally needs CMake 3.31+, the selected C++26 compiler (AppleClang 21, Clang 18 on Ubuntu 24.04 or Clang 21 on Ubuntu 26.04), Git and the platform SDK. Do not change a system compiler or install a package manager implicitly. Versions and independent SDK requirements are in `cmake/mvp-dependencies.json`; individual project minimums are not the aggregate recipe minimum.

From the source root, create a private directory outside the checkout:

```sh
repo=$(pwd -P)
work=$(mktemp -d "${TMPDIR:-/tmp}/symphony-first-run.XXXXXX")
umask 077
(cd tools/qxctl && GOWORK=off GOTOOLCHAIN=local GOENV=off \
  GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org \
  go build -mod=readonly -trimpath -o "$work/qxctl" ./cmd/qxctl)
sh tests/launch-acceptance/source-smoke.sh "$work/qxctl" "$repo" "$work/evidence"
```

The smoke checks executable identity, reads the invariant registry without installed engines, and verifies refusal of a missing repository. The registry response does not assert semantic validity: full validation requires the separately installed validator. The CLI development identity is `0.1.0-dev`; unavailable source stamps in an ordinary local build are expected. The reproducibility workflow separately supplies verified source/build identity. No service, user state, credentials or provider is activated.

For an installed native foundation, follow `tests/mvp-dependencies/README.md`: `sh tests/mvp-dependencies/run.sh bootstrap /absolute/new-work`. For a meaningful offline quantitative scenario, install the exact quantitative DAG using that recipe, then run `tests/sbv-launch-boundary/run.sh` against its prefix. The SBV harness explicitly uses a small synthetic census, same-host transport and no provider requests. Keep its evidence separate from real-provider and final-download acceptance.

## Upgrade, rollback and removal

Keep old and new exact versions in separate private prefixes until the new receipt, CLI/owner compatibility and bounded scenario are verified. Never overwrite an immutable same-version receipt. Save each generated uninstaller, receipt support scripts and configured identity sidecar. For an unmanaged prefix, use the module's `INSTALL.md` removal command and exact configured uninstaller; remove dependants before dependencies. Preserve data stores and unrelated files. A managed shared root requires its authenticated qxctl lifecycle, not direct CMake removal.

Rollback selects the previous verified executable/prefix explicitly. It does not downgrade or rewrite retained data; check that owner's retained-version admissions first. Reconcile uncertain lifecycle outcomes before retrying. For this CLI-only example, removal is deletion of the explicitly selected standalone binary after saving its evidence; no service or system entry was installed. Keep failed work directories for diagnosis rather than reusing partial stages.

Package relocation, tamper refusal, interrupted transforms, retries and rollback mechanics are tested in `tests/package-release/README.md`. Those fixtures are not released product archives. Final downloadable installation commands remain pending an approved candidate URL, trust policy and artifact manifest; do not substitute an unverified URL or claim these source commands prove fresh-machine acceptance.
