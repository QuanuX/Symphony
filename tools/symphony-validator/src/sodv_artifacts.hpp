#pragma once
#include <cstddef>
#include <string>
#include <vector>

struct SodvArtifactCheckResult {
    bool success = true;
    std::size_t records_checked = 0;
    std::size_t completed_transactions = 0;
    std::vector<std::string> messages;
};

// Local consistency only: no signature authentication, network or publication.
SodvArtifactCheckResult check_sodv_artifacts(const std::string& repository);
