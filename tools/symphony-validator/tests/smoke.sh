#!/bin/sh
set -e

echo "Running smoke tests..."

# Verify --help
./build/symphony-validator --help > /dev/null
echo "--help passed"

# Verify --version
./build/symphony-validator --version > /dev/null
echo "--version passed"

# Verify perfectly valid fixture
./build/symphony-validator check --repo ./tests/fixtures_valid > /dev/null
echo "valid fixture passed"

# Verify current repo
./build/symphony-validator check --repo ../.. > /dev/null
echo "current repo passed strict validation"

# Verify invalid repo
if ./build/symphony-validator check --repo /definitely/missing/symphony-validator-path > /dev/null 2>&1; then
    echo "error: invalid repo should fail"
    exit 1
fi
echo "invalid repo passed"

# Verify repo with missing INDEX.md (e.g. the tools directory itself doesn't have knowledge/skvi/INDEX.md)
if ./build/symphony-validator check --repo . > /dev/null 2>&1; then
    echo "error: repo missing INDEX.md should fail"
    exit 1
fi
echo "repo missing INDEX.md failed as expected"

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

# Verify skvi_references path not indexed
if ./build/symphony-validator check --repo ./tests/fixtures_skvi_ref_unindexed > /dev/null 2>&1; then
    echo "error: skvi_ref_unindexed fixture should fail"
    exit 1
fi
echo "skvi_ref_unindexed fixture failed as expected"

# Verify affected_surfaces path absent
if ./build/symphony-validator check --repo ./tests/fixtures_affected_surface_absent > /dev/null 2>&1; then
    echo "error: affected_surface_absent fixture should fail"
    exit 1
fi
echo "affected_surface_absent fixture failed as expected"

# Verify affected_surfaces path existing but unindexed
./build/symphony-validator check --repo ./tests/fixtures_affected_surface_unindexed > /dev/null
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
echo "invalid_sclv_change_type fixture failed as expected"

echo "All smoke tests passed."
