# Source-bound W12 preparation

This release-only helper prepares reviewable construction and campaign worklists
and runs the existing core and delivery acceptance checkers. It does not build
product candidates, execute providers, sign, upload, configure Anaconda, assign
people, decide applicability or grant acceptance. The feature freeze remains.

From a clean committed checkout, choose an exact Go 1.26.5 executable and a new
absolute private directory outside the checkout:

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local /absolute/go1.26.5/bin/go build \
  -trimpath -o /absolute/private-tools/beta-readiness tests/beta-readiness/prepare.go
/absolute/private-tools/beta-readiness --repo /absolute/Symphony \
  --go /absolute/go1.26.5/bin/go --work /absolute/new-w12-review
```

The preparation helper and its tests use only Go's standard library. Keep the
compiled helper outside the checkout so source admission remains clean. The
private-tools parent directory must already exist. A shell wrapper may handle the
expected exit 2 explicitly; `go run` does not preserve the program's exit code.

The helper verifies current dependency/release maps, reconciles the approved
51 native / six Go engineering selection with current owner identities and
installed build options, verifies dependency order, and retains separate complete
receipt recipes. It emits an ordered construction worklist for each of the four
core profiles, plus optional closure and exact external-dependency requirements.
These are build instructions and pending inputs, not an invented final inventory.

The Go tools are built offline from this exact clean source into the new private
work directory. The original acceptance plan and source mapping are copied and
hashed. The core packet, independent artifact requirements, companion packet and
independent targets are initialized with explicit pending inputs. Source identity
and source inputs are checked again before a source-bound review is emitted.

Exit **2** means successful preparation of an **incomplete** review, as required
while final artifacts/evidence are absent. Exit 1 means a refusal or failed
preparation; preserve its directory/logs and use a new directory after fixing the
cause. Existing outputs and dirty source refuse. This command never returns an
accepted release. Both original checker outputs remain attached; missing work is
not converted into passing fixture observations.

Outputs include:

- `construction.json`: current topological 51-unit native recipes, six independently
  versioned Go commands per profile, optional choices and external/runtime inputs.
- `workorders.json`: 36 scenario/profile records, 58 criteria and 31 supplemental
  requirements with source method/oracle/owner roles and pending observations.
- `core-packet.json`, `requirements.json`, `companion-packet.json`, `targets.json`:
  actual versioned checker inputs. Use the exact [delivery protocol](../launch-acceptance/DELIVERY-ACCEPTANCE.md)
  to populate artifact selections and real evidence; do not edit generated reports.
- `core-report.json`, `companion-report.json`: concrete current incompleteness.
- `review.json`: clean commit/tree, all input hashes, compiled tool identities,
  output/log hashes, retained scope and false acceptance/publication authority.

After decisions or artifacts change, create a new retained revision, bind it to
that source/candidate cohort, and rerun the existing checkers to new reports. Do
not patch hashes inside an old signed payload. Source and fixture records from
different commits are not one final candidate cohort.

The [campaign handoff](../../tests/public-beta/execution/CAMPAIGN-HANDOFF.md) contains
Anaconda, host, provider, performance, restore and independent review inputs. The
[W12 status](../../tests/public-beta/execution/W12-REVIEW.md) records remaining
blockers without waiving retained promises.

Focused selection/scope refusal regressions:

```sh
GOWORK=off GOENV=off GOTOOLCHAIN=local /absolute/go1.26.5/bin/go test -count=1 \
  tests/beta-readiness/prepare.go tests/beta-readiness/prepare_test.go
```

These cover omitted/duplicate units and criteria, version/entrypoint/build-option
substitution, wrong profile, dependency omission and non-topological order. The
existing Go suites separately test strict evidence, signatures, owner observations,
operational scope and coverage. This helper does not replace those verifiers.
The 20 selection/scope cases are retained; two command-admission cases additionally
prove dirty-source refusal before output creation and immutable existing-output
refusal against a clean disposable Git repository.
An additional parser case preserves the prior source reader's refusal of invalid
UTF-8, instead of allowing JSON decoding to silently replace malformed bytes.
