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

echo "All smoke tests passed."
