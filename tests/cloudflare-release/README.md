# Cloudflare I6 local qualification

I6 combines the complete new administration/profile/carrier suites, the affected
SOV/SSIAG/SCV boundaries, installed discovery, projections and **LIVE-01**. Passing
CodeQL or local fixtures does not close the first milestone. No release, provider
activation, profile publication or credential acquisition is performed here.

`manifest.json` names the local suites and required test anchors. Acceptance IDs
identify relevant coverage, not proof that every positive/negative case is closed.
`gate.go` reduces original Go JSON event streams and CTest JUnit files. It checks
artifact hashes, required tests, terminal package/test events, failures and skips.
A missing, skipped, truncated or failed case cannot become a passing local suite.
Names within Go subtests overlap parent tests; event counts are not independent
acceptance-case counts. The reducer never reports live or milestone completion.
Its source/platform selection is a caller declaration, not remote attestation.

Use Go 1.26.5 and explicitly select the repository workspace for source integration.
Standalone module/distribution qualification is a separate existing release gate.
Keep each source revision/platform's evidence in a fresh directory. Record the
exact compiler, toolchain/bundle, installation and executable digests alongside it.

Run from the owning module (retain `go test -json` output, stderr and exit status):

| Suite | Command / required setup |
| --- | --- |
| profiles | `go test -race -json -count=1 ./...` in `tools/sov-transport-profile-admin`; the installed-root case requires the documented disposable host. |
| carrier | `go test -race -json -count=1 ./...` in `tools/sov-ssh-adapter`; actual OpenSSH fixture requires the documented disposable Linux host. |
| qxctl | `go test -race -json -count=1 ./internal/sovcloudflare ./internal/transportsession ./internal/transportprofile/... ./internal/sshprofile ./internal/sovremote ./internal/invariantregistry` in `tools/qxctl`; select exact installed Cloudflare and SBV prefixes through the existing test variables. |
| discovery | `go test -json -count=1 ./cmd/qxctl -run 'TestCloudflare\|TestTransport\|TestSession\|TestSSH\|TestSOVSSH\|TestCommandRegistry\|TestCheckedInExpected\|TestV[45]Owner'` in `tools/qxctl`. |
| ssiag | `go test -race -json -count=1 ./internal/server ./internal/stavproducer -run 'TestCloudflare\|TestTransport\|Test.*Candidate\|Test.*Producer\|Test.*Receipt'` in `modules/secure-identity-access-governance`. |
| native | Build the complete `sov-cloudflare-adapter` package with the exact installed foundation/credential SDKs, then `ctest --test-dir BUILD --output-junit EVIDENCE/native.xml --output-on-failure`. |
| provider | Build `ssiag-provider-private-store` with tests enabled, then retain its complete CTest JUnit report as above. |

Also run installed receipt/helper/consumer checks, the real pinned CLI's offline
qualification and guard probes, generated profile contracts, native SSFV/SKVI and
SCV projection checks, dependency/release manifests, and the source validator.
These remain separately named evidence; they are not inferred from this reducer.
Never run the disposable-root fixture on a personal host or substitute a simulated
audit sink for deployed SSIAG/STAV. Do not disable a required test to get exit zero.

Create a selection file with `source_revision` (40-character Git SHA), `platform`
(`darwin-amd64`, `darwin-arm64`, `linux-amd64` or `linux-arm64`) and `artifacts`.
Each artifact has `suite`, a relative `path` under the evidence directory and the
file's lowercase untagged SHA-256. Unknown/duplicate suite IDs and symlink/escaping
paths reject. No tokens, raw provider replies or Access assertions belong here.

```sh
go test tests/cloudflare-release/gate.go tests/cloudflare-release/gate_test.go
go run tests/cloudflare-release/gate.go \
  --manifest tests/cloudflare-release/manifest.json \
  --selection /absolute/evidence/selection.json \
  --evidence /absolute/evidence > /absolute/evidence/local-gate.json
```

Exit 0 means **local suites passed**. Exit 2 means a local suite is missing,
failed or incomplete. Exit 1 rejects invalid evidence. `first_milestone` stays
`open` and `live_qualification` stays `not_evaluated` in every result. Evaluate
live authority, actual runtime/platform, residual-resource and release evidence
separately against the original contract before any supported-product claim.
See `knowledge/sov/cloudflare/I6-QUALIFICATION.md` for the current gate decision.
