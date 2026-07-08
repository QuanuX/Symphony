#pragma once
#include <string>
#include <vector>
#include "sclv_changelog.hpp"

struct SclvLedgerContinuityResult {
    bool success;
    std::vector<std::string> messages;
};

SclvLedgerContinuityResult check_sclv_ledger_continuity(const SclvCheckResult& sclv_result);
