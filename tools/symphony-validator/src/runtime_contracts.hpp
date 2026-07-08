#pragma once
#include <string>
#include <vector>

struct RuntimeContractShapeResult {
    bool success;
    std::vector<std::string> messages;
};

RuntimeContractShapeResult check_runtime_contract_shapes(const std::string& repo_root);
