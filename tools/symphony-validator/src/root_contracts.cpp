#include "root_contracts.hpp"
#include "evidence.hpp"
#include <filesystem>
#include <fstream>
#include <regex>
#include <sstream>
#include <string>
#include <vector>

namespace fs = std::filesystem;

struct AnchorTarget {
  std::string identifier;
  std::string search_text;
};

struct ContractFileTarget {
  std::string path;
  std::vector<AnchorTarget> anchors;
};

RootContractShapeResult
check_root_contract_shapes(const std::string &repo_root) {
  RootContractShapeResult result;
  result.success = true;
  fs::path root(repo_root);

  std::vector<ContractFileTarget> targets = {
      {"README.md",
       {{"Identity", "## Identity"},
        {"Architecture", "## Architecture"},
        {"Emerging_Vector_Architecture", "## Emerging Vector Architecture"},
        {"Root-Level_Governance_Role", "## Root-Level Governance Role"},
        {"Doctrine", "## Doctrine"},
        {"Python_Doctrine", "## Python Doctrine"}}},
      {"knowledge/platform/INTENT.md",
       {{"Root_Intent", "# QuanuX Symphony Intent"},
        {"Purpose", "## Purpose"},
        {"Scope", "## Scope"},
        {"Non-Scope", "## Non-Scope"},
        {"Relationship_to_Modules", "## Relationship to Modules"},
        {"Relationships", "## Relationships"},
        {"Relationship_to_Emerging_Runtime_Architecture",
         "## Relationship to Emerging Runtime Architecture"},
        {"Installability_Expectations", "## Installability Expectations"},
        {"Owner_Ratification_Boundaries",
         "## Owner Ratification Boundaries"}}}};

  for (const auto &file_target : targets) {
    fs::path p = root / file_target.path;
    if (!fs::exists(p)) {
      result.success = false;
      result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                                                "root_contract.unreadable",
                                                "path=" + file_target.path));
      continue;
    }

    std::ifstream file(p);
    if (!file.is_open()) {
      result.success = false;
      result.messages.push_back(format_evidence(EvidenceCategory::Violation,
                                                "root_contract.unreadable",
                                                "path=" + file_target.path));
      continue;
    }

    std::string content((std::istreambuf_iterator<char>(file)),
                        std::istreambuf_iterator<char>());

    for (const auto &anchor : file_target.anchors) {
      if (content.find(anchor.search_text) != std::string::npos) {
        result.messages.push_back(format_evidence(
            EvidenceCategory::Pass, "root_contract.anchor_present",
            "path=" + file_target.path + " anchor=" + anchor.identifier));
      } else {
        result.success = false;
        result.messages.push_back(format_evidence(
            EvidenceCategory::Violation, "root_contract.anchor_missing",
            "path=" + file_target.path + " anchor=" + anchor.identifier));
      }
    }
  }

  const auto policy = check_first_party_repository_policy(repo_root);
  result.success = result.success && policy.success;
  result.messages.insert(result.messages.end(), policy.messages.begin(),
                         policy.messages.end());
  return result;
}

RootContractShapeResult
check_first_party_repository_policy(const std::string &repo_root,
                                    const std::size_t maximum_entries) {
  RootContractShapeResult result{true, {}};
  const fs::path root(repo_root);
  auto violation = [&](const std::string &rule, const std::string &path) {
    result.success = false;
    result.messages.push_back(
        format_evidence(EvidenceCategory::Violation, rule, "path=" + path));
  };
  for (const auto *misplaced :
       {"INTENT.md", "FEATURES.md", "research/ssiag-phase10b-e"}) {
    if (fs::exists(fs::symlink_status(root / misplaced))) {
      violation("repository_layout.misplaced_surface", misplaced);
    }
  }
  const std::regex python_command(
      R"((^|[^A-Za-z0-9_])(python([0-9]+(\.[0-9]+)*)?|pip[0-9]*|poetry)([^A-Za-z0-9_]|$))",
      std::regex::icase);
  const std::regex python_cmake(
      R"((find_package\s*\(\s*Python|Python[0-9]*_(EXECUTABLE|FOUND)|PYTHON_EXECUTABLE))",
      std::regex::icase);
  std::size_t inspected = 0;
  std::error_code error;
  fs::recursive_directory_iterator it(root, fs::directory_options::none, error),
      end;
  for (; !error && it != end; it.increment(error)) {
    const auto &entry = *it;
    const auto relative =
        entry.path().lexically_relative(root).generic_string();
    const auto name = entry.path().filename().string();
    const auto status = entry.symlink_status(error);
    if (error)
      break;
    if (++inspected > maximum_entries) {
      violation("repository_policy.entry_limit", relative);
      break;
    }
    if (fs::is_directory(status)) {
      if (name == ".git" || name == "build")
        it.disable_recursion_pending();
      continue;
    }
    const auto extension = entry.path().extension().string();
    if (extension == ".py" || extension == ".pyi" || extension == ".pyw" ||
        extension == ".pyc" || extension == ".pyo" || extension == ".ipynb" ||
        name == "pyproject.toml" || name == "Pipfile" ||
        name == "Pipfile.lock" || name == "poetry.lock" || name == "uv.lock" ||
        (name.starts_with("requirements") && extension == ".txt")) {
      violation("repository_policy.python_source_or_dependency", relative);
      continue;
    }
    if (!fs::is_regular_file(status))
      continue;
    std::ifstream input(entry.path(), std::ios::binary);
    if (!input) {
      violation("repository_policy.unreadable", relative);
      continue;
    }
    // Bounded first line catches extensionless interpreter entry points.
    std::string first;
    for (char c; first.size() < 1024U && input.get(c) && c != '\n';)
      first += c;
    if (first.starts_with("#!") && std::regex_search(first, python_command)) {
      violation("repository_policy.python_interpreter", relative);
    }
    if (name != "CMakeLists.txt" && extension != ".cmake" &&
        extension != ".sh" && name != "Makefile" && extension != ".yml" &&
        extension != ".yaml")
      continue;
    const auto build_size = fs::file_size(entry.path(), error);
    if (error)
      break;
    if (build_size > 8U * 1024U * 1024U) {
      violation("repository_policy.build_file_limit", relative);
      continue;
    }
    input.clear();
    input.seekg(0);
    std::string line;
    std::size_t bytes = 0;
    while (std::getline(input, line)) {
      bytes += line.size();
      if (bytes > 8U * 1024U * 1024U) {
        violation("repository_policy.build_file_limit", relative);
        break;
      }
      const auto first_nonspace = line.find_first_not_of(" \t\r");
      if (first_nonspace == std::string::npos || line[first_nonspace] == '#')
        continue;
      if (std::regex_search(line, python_cmake) ||
          std::regex_search(line, python_command)) {
        violation("repository_policy.python_build_dependency", relative);
        break;
      }
    }
  }
  if (error)
    violation("repository_policy.scan_failed", ".");
  if (result.success)
    result.messages.push_back(format_evidence(
        EvidenceCategory::Pass, "repository_policy.native_tooling_and_layout",
        "scope=first-party-repository"));
  return result;
}
