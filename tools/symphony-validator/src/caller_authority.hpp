#pragma once
#include <string>
#include <vector>

struct CallerAuthorityCheckResult {
    bool success;
    std::vector<std::string> messages;
};

CallerAuthorityCheckResult check_caller_authority(const std::string& repo_root);

#ifdef SYMPHONY_VALIDATOR_TESTING
enum class CallerAuthorityTestFault {
    tellg_failure,
    metadata_failure,
    iterator_construction_failure,
    iterator_increment_failure,
};

CallerAuthorityCheckResult check_caller_authority_with_test_fault(
    const std::string& repo_root,
    CallerAuthorityTestFault fault,
    const std::string& relative_path);
#endif
