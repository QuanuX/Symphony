# MVP candidate release identities

[mvp-release-identities.json](mvp-release-identities.json) selects candidate identities for all 70 current component roots and all six production Go commands. The dependency projection remains authoritative for current SDK exports, receipts, owner interfaces and immutable public Go source pins. Run `cmake -P cmake/VerifyMVPReleaseIdentities.cmake` to verify their correspondence and retained owner versions.

The catalog selects identities for build qualification. L01 still approves the shipped capability set; it does not publish a tag or binary, approve a download channel, or activate a provider. A proposal and shared authoring source need no invented runtime package version. The DuckDB recipes select exact upstream files for macOS arm64/amd64 and Linux arm64/amd64. `mvp-platform-profiles.json` records the four required sprint profiles; physical DGX Spark remains optional future evidence.

| Go command | Candidate package identity | Source-build default | Selection |
| --- | --- | --- | --- |
| qxctl | `0.1.0-dev` | `0.1.0-dev` | Matches the existing command registry's `IntroducedIn` version. |
| symphony-stav-append-authority | `0.2.1` | `dev` | Explicit binary candidate from the current `v0.2.1` source line; the public Go tag and compiled binary identity remain separate. |
| symphony-ssiag | `0.1.0-dev` | `dev` | Explicit initial unpublished binary candidate. The recipe sets the linker-supported version; ordinary developer builds retain their existing default. |
| symphony-accordare-stav-producer | `0.1.0-dev` | `0.1.0-dev` | Retains the current independently owned identity. |
| sov-stdio-adapter | `0.1.0-dev` | `0.1.0-dev` | Retains the current identity; `--version` reads no transport input. |
| sov-ssh-adapter | `0.1.0-dev` | `0.1.0-dev` | Retains the current identity; `--version` starts no transport. |

Native package receipt versions, exported SDK versions and exact owner admission lists remain unchanged. Their different numbers serve different contracts. The catalog records all 24 owner declarations, including historical SCV selections. Existing `dev` binaries and the retained SOV `qxctl dev` selection are not silently rewritten. SOV still compares both the selected version and executable digest with the actual serving tool.

`qxctl --version` prints the current command version. `qxctl version --json` reports the `symphony.qxctl.build-info.v1` record, actual Go runtime and target, and executable file digest. A normal unstamped build reports unavailable source fields. A complete, syntactically valid source stamp is labeled `stamped`; that label is not independent verification, publisher trust or release approval. Partial or malformed stamps fail explicitly.

From a clean committed checkout, the following recipes create private local build evidence:

```sh
sh tools/qxctl/tests/reproducible_build.sh /absolute/new-qxctl-work
sh .github/scripts/build-mvp-go-identities.sh /absolute/new-go-candidate-work
```

The qxctl recipe captures one Git commit, archives its source into two paths outside the checkout, uses exact Go 1.26.5 with fresh build caches, disables workspace replacements, selects public checksum-verified dependencies, and builds with fixed flags and the source commit's timestamp. It independently compares the reported source commit/tree, dependency-map digest, toolchain, target and executable digest. Equal bytes from the two builds establish reproducibility only on that recorded profile. No wall-clock build timestamp or host path is injected.

The other Go candidate recipe builds the five remaining production commands, checks their actual compiled `--version` output, and retains per-artifact sidecars with executable SHA-256, selected version, exact source commit/tree, dependency/identity catalog digests, toolchain, target and recipe. It does not claim two-build reproducibility. STAV and SSIAG candidate versions are supplied through their explicit linker variables; requested installation version, receipt and compiled version must still agree.

Native qualification binds its recorded source and compiler/dependency profile to checked receipt self-digests and owned-file hashes. The Swift provider retains `0.1.0-draft`; its operational credential use and production signing qualification remain separate requirements. These source/build identities do not certify final relocation, stripping, signing, notarization, physical DGX Spark/DGX OS behavior or download acceptance. Those launch gates remain explicit under L08 and L10-L14/L26.

CI retains identity JSON and logs, not candidate executables for download. During this sprint, pushed work can proceed to the next authorized objective while CI is pending. Observed results are reviewed in a subsequent run before merge and branch cleanup; a pending result is never recorded as a pass.
