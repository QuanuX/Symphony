# L27–L30 operator qualification

These are source qualification and operator procedures, not final-download acceptance. The required runtime profiles remain macOS 15 ARM64, Ubuntu 24.04 amd64/ARM64 and Ubuntu 26.04 amd64. Generic Ubuntu ARM64 is the sprint surrogate for DGX Spark; no physical Spark or GPU claim follows.

- [Capacity, export and recovery](RETENTION.md) binds store limits and quiescent backup/restore to the implemented owners.
- [Accounts and authority](AUTHORIZATION.md) separates local user boundaries, policy decisions and resulting filesystem effects.
- [Hardware starter](../../examples/mvp-hardware/README.md) selects one real documentary lineage without promising an atlas.
- [SACV parser example](../../modules/sacv-engine/tests/fixtures/README.md) exercises the shipped JSON/internal-reference subset without registering an API.

## Focused store and authority regressions

With Go 1.26.5, run from `tools/qxctl` and retain JSON events outside the checkout:

```sh
GOWORK=off GOTOOLCHAIN=local go test -count=1 -race -json \
  ./internal/snvstate ./internal/scvstate ./internal/scvcorpus \
  ./internal/shvstate ./internal/shvpublicationstate \
  ./internal/scvtransfer ./internal/shvtransfer ./internal/ssiagclient
```

The existing release-functional workflow runs these packages on all four required profiles. Review skips individually: subprocess helper tests may skip when not invoked as helpers; the different-owner test needs a non-root caller. A package pass is not proof of an opt-in installed or live-service test. No extra duplicate CI matrix is introduced.

New restore tests use only synthetic private fixtures and new in-process store handles. They prove quiescent copy/reopen and retained operation reconciliation, not kernel-crash or power-loss durability, a concurrent filesystem snapshot, or a complete cross-account service deployment. Existing transfer and owner tests separately cover interruption, adjacent recovery and malformed/conflicting evidence. Existing installed SHV/SNV campaigns remain necessary for exact native replay and receipt admission.

Native SACV tests run through the existing installed knowledge-group campaign. For a focused source run, configure `modules/sacv-engine` with `BUILD_TESTING=ON`, build, and run `ctest --output-on-failure` in that build directory. Its tests read the shipped example bytes, validate an internal reference and refuse external file/HTTPS references and unavailable YAML parsing.

Do not mark final launch acceptance complete from these results. Final artifact bytes, promised live provider capabilities and publisher approval remain separate gates.
