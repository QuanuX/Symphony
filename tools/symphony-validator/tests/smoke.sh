#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
VALIDATOR_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
REPO_ROOT=$(CDPATH= cd -- "$VALIDATOR_ROOT/../.." && pwd)
VALIDATOR_BIN=${SYMPHONY_VALIDATOR_BIN:-"$VALIDATOR_ROOT/build/symphony-validator"}
FIXTURE_BIN=${SYMPHONY_VALIDATOR_FIXTURE_BIN:-"$VALIDATOR_ROOT/build/validator-fixture-prepare"}

# The default remains the complete documented smoke command. Use --cli-only
# after CTest has already run the native unit tests against this same build.
CLI_ONLY=false
case "${1:-}" in
    '') ;;
    --cli-only) CLI_ONLY=true; shift ;;
    *) echo "usage: $0 [--cli-only]" >&2; exit 1 ;;
esac
[ "$#" -eq 0 ] || { echo "usage: $0 [--cli-only]" >&2; exit 1; }

SMOKE_ROOT=$(mktemp -d)
SMOKE_ROOT=$(CDPATH= cd -- "$SMOKE_ROOT" && pwd -P)
trap 'rm -rf "$SMOKE_ROOT"' EXIT
trap 'exit 1' HUP INT TERM
EVIDENCE_DIR=${SYMPHONY_SMOKE_EVIDENCE_DIR:-"$SMOKE_ROOT/evidence"}
mkdir -p "$EVIDENCE_DIR" "$SMOKE_ROOT/fixtures"

fail() {
    echo "error: $*" >&2
    exit 1
}
contains() {
    grep -F -- "$2" "$1" >/dev/null || fail "$1 is missing: $2"
}
absent() {
    if grep -F -- "$2" "$1" >/dev/null; then
        fail "$1 unexpectedly contains: $2"
    fi
}
prepare() {
    FIXTURE="$SMOKE_ROOT/fixtures/$1"
    "$FIXTURE_BIN" prepare "$SCRIPT_DIR/$2" "$FIXTURE" \
        > "$EVIDENCE_DIR/$1.prepare.log"
}
check() {
    CASE_LOG="$EVIDENCE_DIR/$1.log"
    set +e
    "$VALIDATOR_BIN" check --repo "$2" > "$CASE_LOG" 2>&1
    CASE_EXIT=$?
    set -e
    if [ "$CASE_EXIT" -ne "$3" ]; then
        cat "$CASE_LOG" >&2
        fail "$1 expected exit $3, got $CASE_EXIT"
    fi
    [ "$(grep -c '^summary ' "$CASE_LOG")" -eq 1 ] ||
        fail "$1 must have exactly one summary footer"
    contains "$CASE_LOG" "$4"
    printf 'PASS %s exit=%s expected=%s\n' "$1" "$CASE_EXIT" "$4"
}

if [ "$CLI_ONLY" = false ]; then
    "${SCLV_TEMPORAL_TEST_BIN:-$VALIDATOR_ROOT/build/sclv-temporal-tests}"
    "${SCLV_CROSS_REFERENCE_TEST_BIN:-$VALIDATOR_ROOT/build/sclv-cross-reference-tests}"
    "${CALLER_AUTHORITY_TEST_BIN:-$VALIDATOR_ROOT/build/caller-authority-tests}"
    "${SODV_RELEASE_TEST_BIN:-$VALIDATOR_ROOT/build/sodv-release-tests}" "$REPO_ROOT"
    "${FEATURE_ADMINISTRATION_TEST_BIN:-$VALIDATOR_ROOT/build/feature-administration-tests}" "$REPO_ROOT"
    "${ROOT_SUMMARY_TEST_BIN:-$VALIDATOR_ROOT/build/root-summary-tests}" "$REPO_ROOT"
    "${INVARIANT_OWNERSHIP_TEST_BIN:-$VALIDATOR_ROOT/build/invariant-ownership-tests}" "$REPO_ROOT"
fi

"$VALIDATOR_BIN" --help > "$EVIDENCE_DIR/help.log"
"$VALIDATOR_BIN" --version > "$EVIDENCE_DIR/version.log"
set +e
OUT_APPLY=$("$VALIDATOR_BIN" apply --repo "$REPO_ROOT" 2>&1)
APPLY_EXIT=$?
set -e
[ "$APPLY_EXIT" -eq 1 ] &&
    [ "$OUT_APPLY" = 'error: apply is unavailable; symphony-validator is read-only' ] ||
    fail 'validator apply prohibition drifted'
printf '%s\n' "$OUT_APPLY" > "$EVIDENCE_DIR/apply.log"

# Keep absence of SSFV and FEATURES explicit in the historical positive case.
prepare valid fixtures_valid
[ ! -e "$FIXTURE/modules/ssfv-engine" ] || fail 'pre-SSFV fixture contains an engine'
if find "$FIXTURE" -name FEATURES.md -print | grep . >/dev/null; then
    fail 'pre-SSFV fixture contains inferred FEATURES.md'
fi
check valid "$FIXTURE" 0 'violation=0 exit=0'

prepare incomplete-ssfv fixtures_valid
mkdir -p "$FIXTURE/modules/ssfv-engine"
check incomplete-ssfv "$FIXTURE" 11 \
    'runtime_contract.unreadable path=modules/ssfv-engine/INTENT.md'

prepare unratified-ssfv-schema fixtures_valid
cp -R "$REPO_ROOT/knowledge/ssfv" "$FIXTURE/knowledge/"
cp "$REPO_ROOT/knowledge/FEATURE-ADMINISTRATION-PROFILE.json" "$FIXTURE/knowledge/"
mkdir -p "$FIXTURE/tools/qxctl"
cp "$REPO_ROOT/tools/qxctl/COMMANDS.json" "$FIXTURE/tools/qxctl/"
printf '{}\n' > "$FIXTURE/knowledge/ssfv/schemas/v2/unratified.schema.json"
check unratified-ssfv-schema "$FIXTURE" 8 \
    'artifact.unauthorized path=knowledge/ssfv/schemas/v2/unratified.schema.json'

prepare caller-authority fixtures_valid
printf '\nAI agents may never apply.\n' >> "$FIXTURE/README.md"
check caller-authority "$FIXTURE" 21 \
    'evidence violation caller_authority.class_subject_modal path=README.md line='

prepare sacv-registry fixtures_valid
cp -R "$REPO_ROOT/knowledge/sacv/." "$FIXTURE/knowledge/sacv/"
sed 's/^None\.$/- api_id: invalid-only/' "$REPO_ROOT/knowledge/sacv/REGISTRY.md" \
    > "$FIXTURE/knowledge/sacv/REGISTRY.md"
check sacv-registry "$FIXTURE" 22 'evidence violation sacv.registry.field_invalid'

prepare sodv-releases fixtures_valid
cp -R "$REPO_ROOT/knowledge/sodv/." "$FIXTURE/knowledge/sodv/"
sed -n '/^- release_record_id:/,$p' "$REPO_ROOT/knowledge/sodv/RELEASES.md" \
    >> "$FIXTURE/knowledge/sodv/RELEASES.md"
check sodv-releases "$FIXTURE" 23 'evidence violation sodv.releases.record_id'

prepare feature-administration fixtures_valid
cp -R "$REPO_ROOT/knowledge/ssfv" "$FIXTURE/knowledge/"
cp "$REPO_ROOT/knowledge/FEATURE-ADMINISTRATION-PROFILE.json" "$FIXTURE/knowledge/"
mkdir -p "$FIXTURE/tools/qxctl"
sed 's/"registry_digest": "sha256:/"registry_digest": "sha257:/' \
    "$REPO_ROOT/tools/qxctl/COMMANDS.json" > "$FIXTURE/tools/qxctl/COMMANDS.json"
check feature-administration "$FIXTURE" 24 \
    'evidence violation feature_administration.commands_digest'

prepare invariant-ownership fixtures_valid
sed 's/"registry_digest": "sha256:/"registry_digest": "sha257:/' \
    "$REPO_ROOT/knowledge/INVARIANT-OWNERSHIP.json" > "$FIXTURE/knowledge/INVARIANT-OWNERSHIP.json"
check invariant-ownership "$FIXTURE" 26 'evidence violation invariant_ownership.registry_digest'

check missing-repository /definitely/missing/symphony-validator-path 2 \
    'evidence absent repository.path path absent'
prepare missing-index fixtures_valid
rm "$FIXTURE/knowledge/skvi/INDEX.md"
check missing-index "$FIXTURE" 9 \
    'canonical_surface.manifest.surface_unreadable path=knowledge/skvi/INDEX.md'
# The previous script accidentally reused fixtures_notes for this case, which
# actually contains CHANGELOG.md. Remove the ledger from an otherwise valid copy.
prepare missing-changelog fixtures_valid
rm "$FIXTURE/knowledge/sclv/CHANGELOG.md"
check missing-changelog "$FIXTURE" 9 \
    'canonical_surface.manifest.surface_unreadable path=knowledge/sclv/CHANGELOG.md'

# Every negative fixture must reach its intended rule. A bootstrap failure or
# an unrelated malformed field can no longer satisfy a mere nonzero-exit check.
while IFS='|' read -r NAME EXPECTED_EXIT EXPECTED_EVIDENCE; do
    prepare "$NAME" "$NAME"
    check "$NAME" "$FIXTURE" "$EXPECTED_EXIT" "$EXPECTED_EVIDENCE"
done <<'CASES'
fixtures_missing_root_surface|9|canonical_surface.manifest.bootstrap_unreadable path=README.md
fixtures_missing_root_anchor|13|root_contract.anchor_missing path=README.md anchor=Doctrine
fixtures_missing_runtime_module_surface|9|canonical_surface.manifest.surface_unreadable path=modules/node-troll/INTENT.md
fixtures_missing_knowledge_surface|9|canonical_surface.manifest.surface_unreadable path=knowledge/INTENT.md
fixtures_missing_validator_surface|9|canonical_surface.manifest.surface_unreadable path=tools/symphony-validator/INTENT.md
fixtures_missing_validator_anchor|10|validator_contract.anchor_missing path=tools/symphony-validator/INTENT.md anchor=Purpose
fixtures_missing_runtime_anchor|11|runtime_contract.anchor_missing path=modules/hotpath-runtime/INTENT.md anchor=Identity
fixtures|3|skvi.entry.missing_field path=README.md field=owner
fixtures_notes|3|skvi.entry.missing_field path=README.md field=notes
fixtures_relationships|3|skvi.entry.missing_field path=README.md field=relationships
fixtures_sclv_malformed|4|sclv.record.missing_field record_id=SCLV-PR-011 field=notes
fixtures_sclv_record_pr_mismatch|14|sclv_ledger.record_pr_mismatch record_id=SCLV-PR-012
fixtures_sclv_duplicate_record_id|14|sclv_ledger.record_id_duplicate record_id=SCLV-PR-011
fixtures_sclv_duplicate_related_pr|14|sclv_ledger.related_pr_duplicate related_pr=https://github.com/QuanuX/Symphony/pull/11
fixtures_sclv_duplicate_merge_commit|14|sclv_ledger.merge_commit_duplicate merge_commit=f2d65890f679107fdd114e51c5c8a22ab6eb2af2
fixtures_invalid_skvi_status|6|skvi.status.invalid path=README.md status=some_invalid_status
fixtures_invalid_sclv_status|6|sclv.status.invalid record_id=SCLV-PR-010 status=not_canonical
fixtures_invalid_sclv_change_type|6|sclv.change_type.invalid record_id=SCLV-PR-010 change_type=invalid_change_type
fixtures_invalid_sclv_related_pr|7|sclv.related_pr.shape_invalid record_id=SCLV-PR-010
fixtures_invalid_sclv_merge_commit|7|sclv.merge_commit.shape_invalid record_id=SCLV-PR-010
fixtures_unauthorized_docs|8|artifact.unauthorized path=docs reason=publication_not_authorized
fixtures_unauthorized_mint_json|8|artifact.unauthorized path=mint.json reason=publication_not_authorized
fixtures_unauthorized_projection|8|artifact.unauthorized path=knowledge/generated_report.json reason=projection_file_not_authorized
fixtures_unauthorized_qxctl|20|validator_build.source_file_unlisted path=src/qxctl_integration.cpp
fixtures_unauthorized_schema|8|artifact.unauthorized path=schema reason=schema_template_not_authorized
fixtures_vocab_execution_node|15|doctrine_vocab.stale_namespace path=README.md term=execution-node
fixtures_vocab_native_execution|15|doctrine_vocab.stale_namespace path=README.md term=native-execution
fixtures_vocab_bus_agent|15|doctrine_vocab.stale_namespace path=README.md term=bus-agent
fixtures_vocab_core|15|doctrine_vocab.forbidden_active_term path=README.md term=core
fixtures_vocab_markdown_wins|15|doctrine_vocab.rejected_truth_hierarchy path=README.md phrase=Markdown_always_wins
fixtures_vocab_seeds_1|15|doctrine_vocab.prohibited_runtime_enforcement_wording path=README.md phrase=contract_seeds_enforce_runtime_behavior
fixtures_vocab_seeds_2|15|doctrine_vocab.prohibited_runtime_enforcement_wording path=README.md phrase=contract_seed_enforces_runtime_behavior
fixtures_vocab_seeds_3|15|doctrine_vocab.prohibited_runtime_enforcement_wording path=README.md phrase=seeds_enforce_runtime_behavior
fixtures_skvi_coverage_missing|16|skvi_coverage.declared_surface_unindexed path=tools/symphony-validator/CMakeLists.txt count=0
fixtures_skvi_coverage_duplicate|16|skvi_coverage.declared_surface_indexed_multiple path=README.md count=2
fixtures_skvi_paths_missing|17|skvi_path.indexed_path_missing path=does/not/exist.md
fixtures_skvi_paths_absolute|17|skvi_path.invalid_relative_path path=/tmp/absolute_path.md
fixtures_skvi_paths_traversal|17|skvi_path.invalid_relative_path path=../traversal_path.md
fixtures_skvi_paths_directory|17|skvi_path.indexed_path_not_file path=knowledge
fixtures_sclv_reference_absolute|18|sclv_reference.invalid_relative_path record_id=SCLV-PR-010 field=affected_surfaces path=/absolute/path.md
fixtures_sclv_reference_traversal|18|sclv_reference.invalid_relative_path record_id=SCLV-PR-010 field=affected_surfaces path=../traversal.md
fixtures_validator_build_duplicate_source|20|validator_build.source_list_duplicate path=src/main.cpp
fixtures_validator_build_missing_source|20|validator_build.source_missing path=src/missing.cpp
fixtures_validator_build_unlisted_source|20|validator_build.source_file_unlisted path=src/unlisted.cpp
fixtures_validator_build_outside_src|20|validator_build.source_outside_src path=tests/smoke.cpp
fixtures_validator_build_invalid_extension|20|validator_build.invalid_source_extension path=src/main.c
fixtures_validator_build_traversal|20|validator_build.invalid_source_path path=src/../src/main.cpp
CASES

prepare sparse-pr-namespace fixtures_valid
cp "$SCRIPT_DIR/fixtures_sclv_ledger_gap_warning/knowledge/sclv/CHANGELOG.md" \
    "$FIXTURE/knowledge/sclv/CHANGELOG.md"
check sparse-pr-namespace "$FIXTURE" 0 'sclv_ledger.sparse_pr_namespace'
absent "$CASE_LOG" 'sclv_ledger.record_gap'

# Historical references retain their original ledger bytes. Current SCLV
# semantics permit later absence/non-membership and report provenance warnings.
# Current SKVI coverage rejection is independently asserted in the table above.
prepare historical-skvi-reference fixtures_valid
cp "$SCRIPT_DIR/fixtures_skvi_ref_unindexed/knowledge/sclv/CHANGELOG.md" \
    "$FIXTURE/knowledge/sclv/CHANGELOG.md"
check historical-skvi-reference "$FIXTURE" 0 \
    'evidence warning sclv_reference.historical_path_absent'
contains "$CASE_LOG" 'evidence warning sclv_skvi_reference.historical'

prepare historical-unindexed fixtures_sclv_skvi_reference_unindexed
check historical-unindexed "$FIXTURE" 0 'evidence warning sclv_skvi_reference.historical'
prepare historical-missing fixtures_sclv_reference_missing_skvi
check historical-missing "$FIXTURE" 0 'evidence warning sclv_reference.historical_path_absent'
# Preserve the original final repeated check of this historical boundary.
prepare historical-unindexed-repeat fixtures_sclv_skvi_reference_unindexed
check historical-unindexed-repeat "$FIXTURE" 0 'evidence warning sclv_skvi_reference.historical'

prepare affected-surface-absent fixtures_valid
cp "$SCRIPT_DIR/fixtures_affected_surface_absent/knowledge/sclv/CHANGELOG.md" \
    "$FIXTURE/knowledge/sclv/CHANGELOG.md"
check affected-surface-absent "$FIXTURE" 0 \
    'sclv.affected_surface.provenance_summary records=1 occurrences=1 unique_paths=1 present_paths=0 absent_paths=1 unknown_paths=0 indexed_paths=0 unindexed_paths=1'
absent "$CASE_LOG" 'sclv.affected_surface.absent'
absent "$CASE_LOG" 'sclv.affected_surface.unindexed'

prepare affected-surface-unindexed fixtures_valid
cp "$SCRIPT_DIR/fixtures_affected_surface_unindexed/knowledge/sclv/CHANGELOG.md" \
    "$FIXTURE/knowledge/sclv/CHANGELOG.md"
cp "$SCRIPT_DIR/fixtures_affected_surface_unindexed/existing_unindexed.md" "$FIXTURE/"
for DIRECTORY in src tests cmake; do
    cp -R "$SCRIPT_DIR/fixtures_affected_surface_unindexed/$DIRECTORY" "$FIXTURE/"
done
check affected-surface-unindexed "$FIXTURE" 0 \
    'sclv.affected_surface.provenance_summary records=2 occurrences=5 unique_paths=4 present_paths=4 absent_paths=0 unknown_paths=0 indexed_paths=0 unindexed_paths=4'
absent "$CASE_LOG" 'sclv.affected_surface.absent'
absent "$CASE_LOG" 'sclv.affected_surface.unindexed'

for NAME in fixtures_vocab_score fixtures_vocab_c_o_r_e; do
    prepare "$NAME" "$NAME"
    check "$NAME" "$FIXTURE" 0 'violation=0 exit=0'
done

# Check the live repository once in each supported evidence projection.
check current-repository "$REPO_ROOT" 0 'violation=0 exit=0'
[ "$(grep -c 'caller_authority.scan_complete ' "$CASE_LOG")" -eq 1 ] ||
    fail 'current repository must have exactly one caller-authority summary'
grep 'caller_authority.scan_complete ' "$CASE_LOG" | grep 'findings=0' >/dev/null ||
    fail 'current repository has caller-authority findings'
contains "$CASE_LOG" 'sodv.releases.scan_complete records=3 transactions=1 violations=0'
grep 'feature_administration.scan_complete features=' "$CASE_LOG" | grep 'violations=0' >/dev/null ||
    fail 'current repository has feature-administration violations'
"$FIXTURE_BIN" json-inventory "$REPO_ROOT" "$CASE_LOG"

"$VALIDATOR_BIN" check --repo "$REPO_ROOT" --json > "$EVIDENCE_DIR/current-repository.json"
for FIELD in '"protocol":"symphony.validation.result.v1"' '"evaluation":null' \
    '"evidence_digest":"sha256:' '"result_digest":"sha256:'; do
    [ "$(grep -c "$FIELD" "$EVIDENCE_DIR/current-repository.json")" -eq 1 ] ||
        fail "structured validator projection missing exact field: $FIELD"
done
"$VALIDATOR_BIN" root-summary --repo "$REPO_ROOT" --json > "$EVIDENCE_DIR/root-summary.json"
contains "$EVIDENCE_DIR/root-summary.json" '"protocol": "symphony.repository.root-summary.v1"'
contains "$EVIDENCE_DIR/root-summary.json" '"summary_digest": "sha256:'
"$VALIDATOR_BIN" root-summary --repo "$REPO_ROOT" > "$EVIDENCE_DIR/root-summary.md"
[ "$(grep -c '^<!-- symphony:root-summary:v1:begin -->$' "$EVIDENCE_DIR/root-summary.md")" -eq 1 ] &&
    [ "$(grep -c '^<!-- symphony:root-summary:v1:end -->$' "$EVIDENCE_DIR/root-summary.md")" -eq 1 ] ||
    fail 'root-summary Markdown projection has invalid markers'

echo 'All smoke tests passed'
