#pragma once
#include <string>
#include <vector>

struct ValidatorContractShapeResult {
    bool success;
    std::vector<std::string> messages;
};

ValidatorContractShapeResult check_validator_contract_shapes(const std::string& repo_root);
