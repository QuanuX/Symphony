# Protected transport profile administration

Build with the exact Go 1.26.5 toolchain from this directory using `GOWORK=off go build .`. Run focused tests with `GOWORK=off go test -race ./...`; the disposable-root installed fixture skips unless explicitly enabled. Refresh/check generated schemas and qxctl mirrors with `go run ./scripts/generate_contract.go` / `--check`.

Read [SPEC.md](SPEC.md) for the authority and recovery contract, and [INSTALL.md](INSTALL.md) for the separate protected host and package prerequisites. qxctl's schema/template commands work before installation. I2 has no carrier execution or session operation, and does not change SSH v1.

## I2 local verification — 7 November 2026

Source increment based on C0/I1 `63ac911c9fa831762f5df23fcd14e67d69d2cf20`, isolated on `codex/cloudflare-i2-profiles`.

- Exact Go 1.26.5: helper module `go test -race ./...`, generated-contract parity and public dependency checksums passed. The explicit disposable-root installed-process test was skipped locally; no protected host store, sudoers entry, service or package was installed on the development host.
- Store fault tests passed for prepared intent, lost intent reply, publication, lost outcome reply and archived outcome. Explicit complete/abort, stale state, root/sentinel identity, tombstone, package closure/drift and v1-only carrier refusal are covered.
- Focused SSIAG root/system/TOPS and exact receipt cases, independent audit golden vectors, real local Unix-peer refusal and affected producer vocabulary tests passed.
- qxctl focused profile, invariant v4/v5, 495-command discovery and expected registry, complete text/JSON projection and existing SOV SSH boundaries passed. Direct carrier option/path tests and its contract package passed. No files in the SSH v1 carrier or existing remote client changed.
- Native invariant, feature-administration and root-summary tests passed. Native source validation and SSFV's actual repository check passed. Dependency/release inventories cover 72 component roots, nine Go modules and seven production Go commands.
- The helper cross-compiled for Linux amd64/arm64 and Darwin arm64; the development host built Darwin amd64. Cross-compilation is not execution qualification.

The new four-platform CI job exercises the disposable installed-helper fixture with a simulated committed append sink, not a deployed SSIAG/STAV chain. Its results are not claimed here. Actual host sudo policy, system audit deployment, I3 user transport admission and I5 live Cloudflare/SSH remain separate gates. Full SCV/SHV and I6 integrated campaigns remain deferred. Remote checks are not awaited after pushing this increment.
