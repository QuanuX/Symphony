#!/bin/sh
set -e

echo "Running smoke tests..."

./build/sclv-temporal-tests
echo "SCLV temporal tests passed"

# Verify caller authority
./build/caller-authority-tests
echo "Caller authority tests passed"


# Verify --help
./build/symphony-validator --help > /dev/null
echo "--help passed"

# Verify --version
./build/symphony-validator --version > /dev/null
echo "--version passed"

# Verify perfectly valid fixture
OUT=$(./build/symphony-validator check --repo ./tests/fixtures_valid)
if ! echo "$OUT" | grep -q "violation=0 exit=0"; then
    echo "error: valid fixture missing violation=0 exit=0 in summary"
    exit 1
fi
if [ $(echo "$OUT" | grep -c "^summary ") -ne 1 ]; then
    echo "error: valid fixture should have exactly one summary footer"
    exit 1
fi
echo "valid fixture passed"

# Verify caller-authority regression (exit 21)
TEMP_FIXTURE=$(mktemp -d)
trap 'rm -rf "$TEMP_FIXTURE"' EXIT
cp -a ./tests/fixtures_valid/* "$TEMP_FIXTURE/"
echo "AI agents may never apply." >> "$TEMP_FIXTURE/README.md"
set +e
OUT_AUTH=$(./build/symphony-validator check --repo "$TEMP_FIXTURE" 2>&1)
EXIT_CODE=$?
set -e
if [ $EXIT_CODE -ne 21 ]; then
    echo "error: caller authority violation should exit 21, got $EXIT_CODE"
    exit 1
fi
if ! echo "$OUT_AUTH" | grep -q "evidence violation caller_authority.class_subject_modal path=README.md line="; then
    echo "error: missing expected class_subject_modal evidence"
    exit 1
fi
if [ $(echo "$OUT_AUTH" | grep -c "^summary ") -ne 1 ]; then
    echo "error: invalid auth fixture should have exactly one summary footer"
    exit 1
fi
rm -rf "$TEMP_FIXTURE"
trap - EXIT
echo "caller authority validation passed"

# Verify current repo
OUT_REPO=$(./build/symphony-validator check --repo ../..)
if [ $(echo "$OUT_REPO" | grep -c "^summary ") -ne 1 ]; then
    echo "error: current repo should have exactly one summary footer"
    exit 1
fi
if ! echo "$OUT_REPO" | grep -q "caller_authority.scan_complete files=94 .* findings=0"; then
    echo "error: current repo missing expected caller_authority.scan_complete status or findings=0"
    exit 1
fi
if [ $(echo "$OUT_REPO" | grep -c "artifact.canonical_json_authorized") -ne 28 ]; then
    echo "error: current repo should authorize exactly the 28 ratified STAV JSON artifacts"
    exit 1
fi
echo "current repo passed strict validation"

# Verify invalid repo
set +e
OUT_INV=$(./build/symphony-validator check --repo /definitely/missing/symphony-validator-path 2>&1)
EXIT_CODE=$?
set -e
if [ $EXIT_CODE -eq 0 ]; then
    echo "error: invalid repo should fail"
    exit 1
fi
if ! echo "$OUT_INV" | grep -q "summary pass="; then
    echo "error: invalid repo missing summary footer"
    exit 1
fi
if [ $(echo "$OUT_INV" | grep -c "^summary ") -ne 1 ]; then
    echo "error: invalid repo should have exactly one summary footer"
    exit 1
fi
echo "invalid repo passed"

# Verify repo with missing INDEX.md (e.g. the tools directory itself doesn't have knowledge/skvi/INDEX.md)
if ./build/symphony-validator check --repo . > /dev/null 2>&1; then
    echo "error: repo missing INDEX.md should fail"
    exit 1
fi
echo "repo missing INDEX.md failed as expected"

# Verify fixture missing root surface
if ./build/symphony-validator check --repo ./tests/fixtures_missing_root_surface > /dev/null 2>&1; then
    echo "error: fixture missing root surface should fail"
    exit 1
fi
echo "fixture missing root surface failed as expected"

# Verify fixture missing root anchor
if ./build/symphony-validator check --repo ./tests/fixtures_missing_root_anchor > /dev/null 2>&1; then
    echo "error: fixture missing root anchor should fail"
    exit 1
fi
echo "fixture missing root anchor failed as expected"

# Verify fixture missing runtime module surface
if ./build/symphony-validator check --repo ./tests/fixtures_missing_runtime_module_surface > /dev/null 2>&1; then
    echo "error: fixture missing runtime module surface should fail"
    exit 1
fi
echo "fixture missing runtime module surface failed as expected"

# Verify fixture missing knowledge surface
if ./build/symphony-validator check --repo ./tests/fixtures_missing_knowledge_surface > /dev/null 2>&1; then
    echo "error: fixture missing knowledge surface should fail"
    exit 1
fi
echo "fixture missing knowledge surface failed as expected"

# Verify fixture missing validator surface
if ./build/symphony-validator check --repo ./tests/fixtures_missing_validator_surface > /dev/null 2>&1; then
    echo "error: fixture missing validator surface should fail"
    exit 1
fi
echo "fixture missing validator surface failed as expected"

# Verify fixture missing validator anchor
if ./build/symphony-validator check --repo ./tests/fixtures_missing_validator_anchor > /dev/null 2>&1; then
    echo "error: fixture missing validator anchor should fail"
    exit 1
fi
echo "fixture missing validator anchor failed as expected"

# Verify fixture missing runtime anchor
if ./build/symphony-validator check --repo ./tests/fixtures_missing_runtime_anchor > /dev/null 2>&1; then
    echo "error: fixture missing runtime anchor should fail"
    exit 1
fi
echo "fixture missing runtime anchor failed as expected"

# Verify malformed SKVI fixture (missing title, owner, etc.)
if ./build/symphony-validator check --repo ./tests/fixtures > /dev/null 2>&1; then
    echo "error: malformed fixture should fail"
    exit 1
fi
echo "malformed fixture failed as expected"

# Verify malformed SKVI fixture missing notes
if ./build/symphony-validator check --repo ./tests/fixtures_notes > /dev/null 2>&1; then
    echo "error: malformed fixture missing notes should fail"
    exit 1
fi
echo "malformed fixture missing notes failed as expected"

# Verify malformed SKVI fixture missing relationships
if ./build/symphony-validator check --repo ./tests/fixtures_relationships > /dev/null 2>&1; then
    echo "error: malformed fixture missing relationships should fail"
    exit 1
fi
echo "malformed fixture missing relationships failed as expected"

# Verify repo path missing knowledge/sclv/CHANGELOG.md
# We can use fixtures_valid but remove CHANGELOG.md temporarily, or create a new fixture.
# Actually, fixtures_notes has SKVI but no SCLV CHANGELOG!
if ./build/symphony-validator check --repo ./tests/fixtures_notes > /dev/null 2>&1; then
    echo "error: repo missing CHANGELOG.md should fail"
    exit 1
fi
echo "repo missing CHANGELOG.md failed as expected"

# Verify malformed SCLV fixture
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_malformed > /dev/null 2>&1; then
    echo "error: malformed SCLV fixture should fail"
    exit 1
fi
echo "malformed SCLV fixture failed as expected"

# Verify SCLV record_id/related_pr mismatch
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_record_pr_mismatch > /dev/null 2>&1; then
    echo "error: fixtures_sclv_record_pr_mismatch should fail"
    exit 1
fi
echo "fixtures_sclv_record_pr_mismatch failed as expected"

# Verify SCLV duplicate record_id
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_duplicate_record_id > /dev/null 2>&1; then
    echo "error: fixtures_sclv_duplicate_record_id should fail"
    exit 1
fi
echo "fixtures_sclv_duplicate_record_id failed as expected"

# Verify SCLV duplicate related_pr
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_duplicate_related_pr > /dev/null 2>&1; then
    echo "error: fixtures_sclv_duplicate_related_pr should fail"
    exit 1
fi
echo "fixtures_sclv_duplicate_related_pr failed as expected"

# Verify SCLV duplicate merge_commit
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_duplicate_merge_commit > /dev/null 2>&1; then
    echo "error: fixtures_sclv_duplicate_merge_commit should fail"
    exit 1
fi
echo "fixtures_sclv_duplicate_merge_commit failed as expected"

# Verify sparse SCLV PR-number namespace
OUT_SPARSE=$(./build/symphony-validator check --repo ./tests/fixtures_sclv_ledger_gap_warning)
if ! echo "$OUT_SPARSE" | grep -q "sclv_ledger.sparse_pr_namespace"; then
    echo "error: sparse SCLV fixture missing sparse namespace evidence"
    exit 1
fi
if echo "$OUT_SPARSE" | grep -q "sclv_ledger.record_gap"; then
    echo "error: sparse SCLV fixture emitted a false record-gap warning"
    exit 1
fi
echo "sparse SCLV PR-number namespace passed"

# Verify skvi_references path not indexed
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_ref_unindexed > /dev/null 2>&1; then
    echo "error: skvi_ref_unindexed fixture should fail"
    exit 1
fi
echo "skvi_ref_unindexed fixture failed as expected"

# Verify SCLV skvi_reference is indexed in SKVI
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_skvi_reference_unindexed > /dev/null 2>&1; then
    echo "error: fixtures_sclv_skvi_reference_unindexed fixture should fail"
    exit 1
fi
echo "fixtures_sclv_skvi_reference_unindexed fixture failed as expected"

# Verify affected_surfaces path absent
if ./build/symphony-validator check --repo ./tests/fixtures_affected_surface_absent > /dev/null 2>&1; then
    echo "error: affected_surface_absent fixture should fail"
    exit 1
fi
echo "affected_surface_absent fixture failed as expected"

# Verify affected_surfaces path existing but unindexed
OUT_WARN=$(./build/symphony-validator check --repo ./tests/fixtures_affected_surface_unindexed)
if ! echo "$OUT_WARN" | grep -qE "summary pass=.* warning=[1-9][0-9]* violation=0 exit=0"; then
    echo "error: affected_surface_unindexed missing warning > 0 or exit=0"
    exit 1
fi
echo "affected_surface_unindexed fixture passed with warning"

# Verify invalid SKVI status
if ./build/symphony-validator check --repo ./tests/fixtures_invalid_skvi_status > /dev/null 2>&1; then
    echo "error: invalid_skvi_status fixture should fail"
    exit 1
fi
echo "invalid_skvi_status fixture failed as expected"

# Verify invalid SCLV status
if ./build/symphony-validator check --repo ./tests/fixtures_invalid_sclv_status > /dev/null 2>&1; then
    echo "error: invalid_sclv_status fixture should fail"
    exit 1
fi
echo "invalid_sclv_status fixture failed as expected"

# Verify invalid SCLV change_type
if ./build/symphony-validator check --repo ./tests/fixtures_invalid_sclv_change_type > /dev/null 2>&1; then
    echo "error: invalid_sclv_change_type fixture should fail"
    exit 1
fi
# Verify invalid SCLV related_pr shape
if ./build/symphony-validator check --repo ./tests/fixtures_invalid_sclv_related_pr > /dev/null 2>&1; then
    echo "error: invalid_sclv_related_pr fixture should fail"
    exit 1
fi
echo "invalid_sclv_related_pr fixture failed as expected"

# Verify invalid SCLV merge_commit shape
if ./build/symphony-validator check --repo ./tests/fixtures_invalid_sclv_merge_commit > /dev/null 2>&1; then
    echo "error: invalid_sclv_merge_commit fixture should fail"
    exit 1
fi
echo "invalid_sclv_merge_commit fixture failed as expected"

# Verify unauthorized docs/ fixture
if ./build/symphony-validator check --repo ./tests/fixtures_unauthorized_docs > /dev/null 2>&1; then
    echo "error: unauthorized_docs fixture should fail"
    exit 1
fi
echo "unauthorized_docs fixture failed as expected"

# Verify unauthorized mint.json fixture
if ./build/symphony-validator check --repo ./tests/fixtures_unauthorized_mint_json > /dev/null 2>&1; then
    echo "error: unauthorized_mint_json fixture should fail"
    exit 1
fi
echo "unauthorized_mint_json fixture failed as expected"

# Verify unauthorized projection file
if ./build/symphony-validator check --repo ./tests/fixtures_unauthorized_projection > /dev/null 2>&1; then
    echo "error: unauthorized_projection fixture should fail"
    exit 1
fi
echo "unauthorized_projection fixture failed as expected"

# Verify unauthorized qxctl integration
if ./build/symphony-validator check --repo ./tests/fixtures_unauthorized_qxctl > /dev/null 2>&1; then
    echo "error: unauthorized_qxctl fixture should fail"
    exit 1
fi
echo "unauthorized_qxctl fixture failed as expected"

# Verify unauthorized schema/template fixture
if ./build/symphony-validator check --repo ./tests/fixtures_unauthorized_schema > /dev/null 2>&1; then
    echo "error: unauthorized_schema fixture should fail"
    exit 1
fi
echo "unauthorized_schema fixture failed as expected"

# Verify doctrine vocabulary drift checks
if ./build/symphony-validator check --repo ./tests/fixtures_vocab_execution_node > /dev/null 2>&1; then
    echo "error: fixtures_vocab_execution_node should fail"
    exit 1
fi
echo "fixtures_vocab_execution_node failed as expected"

if ./build/symphony-validator check --repo ./tests/fixtures_vocab_native_execution > /dev/null 2>&1; then
    echo "error: fixtures_vocab_native_execution should fail"
    exit 1
fi
echo "fixtures_vocab_native_execution failed as expected"

if ./build/symphony-validator check --repo ./tests/fixtures_vocab_bus_agent > /dev/null 2>&1; then
    echo "error: fixtures_vocab_bus_agent should fail"
    exit 1
fi
echo "fixtures_vocab_bus_agent failed as expected"

if ./build/symphony-validator check --repo ./tests/fixtures_vocab_core > /dev/null 2>&1; then
    echo "error: fixtures_vocab_core should fail"
    exit 1
fi
echo "fixtures_vocab_core failed as expected"

if ! ./build/symphony-validator check --repo ./tests/fixtures_vocab_score > /dev/null 2>&1; then
    echo "error: fixtures_vocab_score should pass"
    exit 1
fi
echo "fixtures_vocab_score passed as expected"

if ! ./build/symphony-validator check --repo ./tests/fixtures_vocab_c_o_r_e > /dev/null 2>&1; then
    echo "error: fixtures_vocab_c_o_r_e should pass"
    exit 1
fi
echo "fixtures_vocab_c_o_r_e passed as expected"

if ./build/symphony-validator check --repo ./tests/fixtures_vocab_markdown_wins > /dev/null 2>&1; then
    echo "error: fixtures_vocab_markdown_wins should fail"
    exit 1
fi
echo "fixtures_vocab_markdown_wins failed as expected"

if ./build/symphony-validator check --repo ./tests/fixtures_vocab_seeds_1 > /dev/null 2>&1; then
    echo "error: fixtures_vocab_seeds_1 should fail"
    exit 1
fi
echo "fixtures_vocab_seeds_1 failed as expected"

if ./build/symphony-validator check --repo ./tests/fixtures_vocab_seeds_2 > /dev/null 2>&1; then
    echo "error: fixtures_vocab_seeds_2 should fail"
    exit 1
fi
echo "fixtures_vocab_seeds_2 failed as expected"

if ./build/symphony-validator check --repo ./tests/fixtures_vocab_seeds_3 > /dev/null 2>&1; then
    echo "error: fixtures_vocab_seeds_3 should fail"
    exit 1
fi
echo "fixtures_vocab_seeds_3 failed as expected"


# Verify SKVI coverage missing entry
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_coverage_missing > /dev/null 2>&1; then
    echo "error: fixtures_skvi_coverage_missing should fail"
    exit 1
fi
echo "fixtures_skvi_coverage_missing failed as expected"

# Verify SKVI coverage duplicate entry
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_coverage_duplicate > /dev/null 2>&1; then
    echo "error: fixtures_skvi_coverage_duplicate should fail"
    exit 1
fi
echo "fixtures_skvi_coverage_duplicate failed as expected"

# Verify SKVI paths missing entry
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_paths_missing > /dev/null 2>&1; then
    echo "error: fixtures_skvi_paths_missing should fail"
    exit 1
fi
echo "fixtures_skvi_paths_missing failed as expected"

# Verify SKVI paths absolute entry
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_paths_absolute > /dev/null 2>&1; then
    echo "error: fixtures_skvi_paths_absolute should fail"
    exit 1
fi
echo "fixtures_skvi_paths_absolute failed as expected"

# Verify SKVI paths traversal entry
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_paths_traversal > /dev/null 2>&1; then
    echo "error: fixtures_skvi_paths_traversal should fail"
    exit 1
fi
echo "fixtures_skvi_paths_traversal failed as expected"

# Verify SKVI paths directory entry
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_paths_directory > /dev/null 2>&1; then
    echo "error: fixtures_skvi_paths_directory should fail"
    exit 1
fi
echo "fixtures_skvi_paths_directory failed as expected"

# Verify SCLV reference missing affected path
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_reference_missing_affected > /dev/null 2>&1; then
    echo "error: fixtures_sclv_reference_missing_affected should fail"
    exit 1
fi
echo "fixtures_sclv_reference_missing_affected failed as expected"

# Verify SCLV reference missing skvi path
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_reference_missing_skvi > /dev/null 2>&1; then
    echo "error: fixtures_sclv_reference_missing_skvi should fail"
    exit 1
fi
echo "fixtures_sclv_reference_missing_skvi failed as expected"

# Verify SCLV reference absolute path
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_reference_absolute > /dev/null 2>&1; then
    echo "error: fixtures_sclv_reference_absolute should fail"
    exit 1
fi
echo "fixtures_sclv_reference_absolute failed as expected"

# Verify SCLV reference traversal path
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_reference_traversal > /dev/null 2>&1; then
    echo "error: fixtures_sclv_reference_traversal should fail"
    exit 1
fi
echo "fixtures_sclv_reference_traversal failed as expected"

# Verify SCLV reference directory path
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_reference_directory > /dev/null 2>&1; then
    echo "error: fixtures_sclv_reference_directory should fail"
    exit 1
fi
# Verify SCLV-SKVI unindexed reference
if ./build/symphony-validator check --repo ./tests/fixtures_sclv_skvi_reference_unindexed > /dev/null 2>&1; then
    echo "error: fixtures_sclv_skvi_reference_unindexed should fail"
    exit 1
fi
echo "fixtures_sclv_skvi_reference_unindexed failed as expected"

# Verify validator_build duplicate source
if ./build/symphony-validator check --repo ./tests/fixtures_validator_build_duplicate_source > /dev/null 2>&1; then
    echo "error: fixtures_validator_build_duplicate_source should fail"
    exit 1
fi
echo "fixtures_validator_build_duplicate_source failed as expected"

# Verify validator_build missing source
if ./build/symphony-validator check --repo ./tests/fixtures_validator_build_missing_source > /dev/null 2>&1; then
    echo "error: fixtures_validator_build_missing_source should fail"
    exit 1
fi
echo "fixtures_validator_build_missing_source failed as expected"

# Verify validator_build unlisted source
if ./build/symphony-validator check --repo ./tests/fixtures_validator_build_unlisted_source > /dev/null 2>&1; then
    echo "error: fixtures_validator_build_unlisted_source should fail"
    exit 1
fi
echo "fixtures_validator_build_unlisted_source failed as expected"

# Verify validator_build outside src
if ./build/symphony-validator check --repo ./tests/fixtures_validator_build_outside_src > /dev/null 2>&1; then
    echo "error: fixtures_validator_build_outside_src should fail"
    exit 1
fi
echo "fixtures_validator_build_outside_src failed as expected"

# Verify validator_build invalid extension
if ./build/symphony-validator check --repo ./tests/fixtures_validator_build_invalid_extension > /dev/null 2>&1; then
    echo "error: fixtures_validator_build_invalid_extension should fail"
    exit 1
fi
echo "fixtures_validator_build_invalid_extension failed as expected"

# Verify validator_build traversal
if ./build/symphony-validator check --repo ./tests/fixtures_validator_build_traversal > /dev/null 2>&1; then
    echo "error: fixtures_validator_build_traversal should fail"
    exit 1
fi
echo "fixtures_validator_build_traversal failed as expected"

echo "All smoke tests passed."
