#pragma once

#include <cstddef>
#include <string>
#include <vector>

struct InvariantOwnershipCheckResult {
    bool success;
    std::vector<std::string> messages;
    std::size_t invariants_checked;
    std::size_t adapters_checked;
    std::size_t evidence_references_checked;
    std::size_t violations;
};

InvariantOwnershipCheckResult check_invariant_ownership(const std::string& repo_root);
