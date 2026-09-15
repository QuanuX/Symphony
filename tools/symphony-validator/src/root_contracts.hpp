#pragma once
#include <cstddef>
#include <string>
#include <vector>

struct RootContractShapeResult {
    bool success;
    std::vector<std::string> messages;
};

RootContractShapeResult check_root_contract_shapes(const std::string& repo_root);
RootContractShapeResult check_first_party_repository_policy(const std::string& repo_root, std::size_t maximum_entries = 100000U);
