# Go candidate source and owner closure

This gate binds seven prepared Go executables to one clean source commit/tree,
the selected artifact identities and each actual linked module/checksum. It is
candidate engineering evidence, not SODV release completion, publisher approval,
fresh-host acceptance or runtime activation.

Keep native versions and Go dependency source versions distinct. Current native
candidate identities remain SSIAG `0.1.0-dev`, STAV append authority `0.2.1`, and
Accordare, qxctl, SSH, stdio and the optional profile administrator `0.1.0-dev`.
A same-native-version build uses a separate immutable prefix/custody slot and
packaging build identity; it never overwrites an earlier installation.

The required changed Go dependency is qxctl's Accordare client. Its public
`client.VerifyInstallation` calls the module's strict native receipt reader, so
an older published client does not understand the current guarded owner receipt.
STAV's public client and the SSH contract remain independently pinned; neither
must be relabeled to close this reader boundary. SSIAG's qxctl client is owned
inside qxctl rather than imported from a public SSIAG Go module.

The owner implementation is committed and pushed first. Resolve that exact full
commit through Go1.26.5, with `GOWORK=off`, the public Go proxy/checksum database,
a fresh private cache, and no private/checksum bypass or replacement. Retain the
actual `go mod download -json` result. Let Go determine the pseudo-version; never
construct or publish a pretend tag or a local module archive under an existing
published version. Verify the complete origin commit and module subdirectory,
module sum and go.mod sum before updating the consumer with normal Go tooling.
The selected Accordare prerelease baseline currently produces the
`v0.1.0-dev.0.<UTC-commit-time>-<12hex>` form. These are candidate dependency
coordinates, not a new canonical release ledger entry.

The selected owner source is recorded without local paths in
`cmake/mvp-go-owner-closure.json`: commit
`71bbb60d870ab1da205b1b82d62adec38dca9a68`, resolved by Go to
`v0.1.0-dev.0.20261116054240-71bbb60d870a`. The record retains the actual public
module and go.mod checksums and exact Git origin; the gate compares fresh resolver
evidence to that committed selection. Historical published versions and ledger
entries remain unchanged.

After the reviewed consumer pin and source-bound maps are committed, build the
six owner/adapter candidates with `.github/scripts/build-mvp-go-identities.sh`
and qxctl with `tools/qxctl/tests/reproducible_build.sh`. Both recipes use clean
archived source, immutable dependency metadata and no replacements. Preserve the
existing standalone public-dependency check; workspace success cannot replace it.

Run the closure gate using the pinned Go toolchain on PATH:

```sh
sh tests/launch-acceptance/check-go-candidate-closure.sh \
  -repository /absolute/clean/source \
  -build-root /absolute/go-candidates \
  -qxctl-build-root /absolute/qxctl-reproducible \
  -accordare-download /absolute/evidence/accordare-download.json \
  -accordare-source-commit EXACT_REVIEWED_OWNER_COMMIT \
  -output /absolute/new-evidence/go-candidate-closure.json
```

The gate performs no network request and does not execute candidate binaries.
It writes only a new evidence file outside the selected source checkout.
It reads Go build metadata, checks all linked dependencies against the committed
checksums, refuses replacements/old Accordare readers/mixed source or platforms,
and verifies the dependency commit remains an ancestor with the identical
Accordare subtree. Its output keeps native version, module pseudo-version,
source identity and executable identity separate. It does not authenticate a
caller-invented download report; retain the actual resolver command/output and
independent publication review separately. Missing proof is a refusal.

The real owner fixture in `qualify-go-owners.go` installs into a fresh private
native prefix, exports only the exact binary and receipt into a portable signed
delivery fixture, and installs the removal target afresh through the same owner.
Host-local `.symphony-package-retirement` state and stable locks are never copied
as delivery payload or accepted as another installation's custody. Removal
preserves the unrelated sentinel and the exact retired tombstones/lock files.
It starts no service, enrolls no TOPS and grants no host authority. The signed
fixture uses disposable keys, which cannot authorize production distribution.

Focused checks:

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local go test -race \
  tests/launch-acceptance/go-candidate-closure.go \
  tests/launch-acceptance/go-candidate-closure_test.go
GOWORK=off GOENV=off GOTOOLCHAIN=local go test -race \
  tests/launch-acceptance/qualification-common.go \
  tests/launch-acceptance/qualify-go-owners.go \
  tests/launch-acceptance/qualify-go-owners_test.go
cd tools/qxctl
GOWORK=off GOENV=off GOTOOLCHAIN=local go test -race ./cmd/qxctl \
  -run '^TestAccordareOwnerDependencyClosure$' -count=1
```

The qxctl reader regression builds the actual selected dependency's native owner,
installs it into a fresh private prefix and checks the imported client against
that receipt. It also refuses a legacy capability contract even when its receipt
digest is recomputed. Run it after the genuine owner dependency pin is installed.

Actual current-candidate builds and installed-owner runs are required in addition
to these refusal fixtures. System/cross-user reclamation, arbitrary old consumers,
public release publication and final-host/platform acceptance remain separate.
