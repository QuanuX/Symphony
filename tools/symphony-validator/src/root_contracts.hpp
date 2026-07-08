#pragma once
#include <string>
#include <vector>

struct RootContractShapeResult {
    bool success;
    std::vector<std::string> messages;
};

RootContractShapeResult check_root_contract_shapes(const std::string& repo_root);
