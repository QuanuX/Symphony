#pragma once
#include <string>
#include <vector>

struct KnowledgeContractShapeResult {
    bool success;
    std::vector<std::string> messages;
};

KnowledgeContractShapeResult check_knowledge_contract_shapes(const std::string& repo_root);
