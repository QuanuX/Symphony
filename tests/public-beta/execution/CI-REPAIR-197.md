# PR 197: repository-policy CI repair

All four installed knowledge SDK jobs in [run 43401497996](https://github.com/QuanuX/Symphony/actions/runs/43401497996)
failed at `release-installed-validator` on source `d43506ce` with the same 11
violations: seven new Python helpers/tests and four build/workflow invocations.
The root repository contract excludes first-party Python implementations and
dependencies, including acceptance helpers. The validator correctly refused them.

The helpers now use Go 1.26.5 and its standard library. Their actual owner,
signature, numerical-oracle, source-binding and refusal checks remain in place.
Build recipes and workflows compile the explicit standalone file sets. CodeQL's
manual Go extraction now explicitly builds the standalone release tools as well
as workspace modules; hosted extraction of this change remains unverified until
its next run finishes.

The unchanged native validator then exposed a second issue: `docs/` is a reserved
publication surface, including an empty directory. The preparation dossier moved
to `tests/public-beta/`, with active links and evidence paths corrected. Recorded
source hashes, original commands and earlier generated packets retain their
historical identities. This repair grants no publication authority and changes
neither repository policy nor product behavior.

Local evidence:

- The unchanged native validator passes with zero violations; its canonical
  surface regressions pass. Four existing historical-reference warnings remain.
- Dossier validation passes 16 named cases; preparation passes 23, delivery helper
  checks pass 20, and the relocated acceptance companion passes 64. The external
  helper passes 18 unique named cases across its initial and focused follow-up
  runs. Race checking was enabled for these Go suites.
- The independent installed public-SDK campaign passes all 119 checks. Actual
  staged SSIAG/STAV/Accordare owner inspection, native candidate extraction, qxctl
  and both SOV executable inspections pass with their tamper, output-reuse and
  removal checks. These are local fixture/source qualifications, not final hosts.
- Follow-up JSON admission tests refuse invalid UTF-8 and malformed surrogate
  escapes where applicable, preserve canonical line-separator bytes for external
  receipt verification, and reject numeric-zero capability operations. The valid
  SDK/delivery campaigns preceded those stricter parser changes; focused parser
  checks cover the follow-up rather than claiming a repeated full campaign.
- Ten standalone CodeQL build file sets compile locally. Dependency/release-map,
  workflow syntax and whitespace checks pass. Local builds are not hosted CodeQL
  extraction or a new alert-disposition audit.

The original [CodeQL run](https://github.com/QuanuX/Symphony/actions/runs/43401497828)
passed all six jobs, including analysis/upload and extraction-evidence steps.
The separate [macOS profile cancellation](https://github.com/QuanuX/Symphony/actions/runs/43401497706/job/131805175554)
never acquired a runner (`runner_id=0`, no steps); GitHub reported hosted-runner
acquisition failure and macOS ARM capacity constraints. It did not execute tests.
Passing old jobs are not selectively rerun. The next branch push triggers new
head checks; their completion is deliberately not awaited.

[Machine-readable repair evidence](ci-repair-197.json) records local evidence
hashes and the affected jobs. The clean-source preparation integration is recorded
separately after the repair commit exists. Acceptance and publication remain
false, and the feature freeze remains in force.
