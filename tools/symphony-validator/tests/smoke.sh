#!/bin/sh
set -e

echo "Running smoke tests..."

# Verify --help
./build/symphony-validator --help > /dev/null
echo "--help passed"

# Verify --version
./build/symphony-validator --version > /dev/null
echo "--version passed"

# Verify valid repo
./build/symphony-validator check --repo . > /dev/null
echo "valid repo passed"

# Verify invalid repo
if ./build/symphony-validator check --repo /definitely/missing/symphony-validator-path > /dev/null 2>&1; then
    echo "error: invalid repo should fail"
    exit 1
fi
echo "invalid repo passed"

echo "All smoke tests passed."
