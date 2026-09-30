#include "native_test.hpp"

namespace {
using namespace native_test;

std::string owned_path(const Json& receipt, const std::string& suffix) {
  std::vector<std::string> matches;
  for (const auto& entry : receipt.at("files")) {
    const auto path = entry.at("path").get<std::string>();
    if (path.ends_with(suffix)) matches.push_back(path);
  }
  NT_REQUIRE(matches.size() == 1);
  return matches.front();
}

Json source_snapshot(const fs::path& prefix, const Json& receipt) {
  Json snapshot = Json::object();
  for (const auto& entry : receipt.at("files"))
    snapshot[entry.at("path").get<std::string>()] = digest(read(prefix / entry.at("path").get<std::string>()));
  return snapshot;
}

void test_sdk_receipt_admission(const Arguments& args) {
  const auto source = fs::absolute(args.require("source"));
  const auto original = fs::absolute(args.require("prefix"));
  const auto compiler = args.require("compiler");
  const fs::path relative = "share/symphony/receipts/snv-common-cpp/0.1.0-dev/install-receipt.json";
  const auto original_receipt = read_json(original / relative);
  const auto original_receipt_bytes = read(original / relative);
  const auto before = source_snapshot(original, original_receipt);
  TempDir temporary("snv-sdk-admission-");
  Json results = Json::array();
  for (const auto& name : {"valid", "omitted_header", "omitted_config", "duplicate_path"}) {
    const std::string case_name = name;
    const auto prefix = temporary.path / case_name / "prefix";
    fs::create_directories(prefix.parent_path());
    fs::copy(original, prefix, fs::copy_options::recursive | fs::copy_options::copy_symlinks);
    auto receipt = read_json(prefix / relative);
    const auto header = owned_path(receipt, "/symphony/snv/common.hpp");
    const auto config = owned_path(receipt, "/SymphonySnvCommonConfig.cmake");
    const auto foundation = read_json(prefix / "share/symphony/receipts/knowledge-vector-engine-cpp/0.2.0-dev/install-receipt.json");
    const auto foundation_config = owned_path(foundation, "/SymphonyKnowledgeVectorEngineConfig.cmake");
    std::string rejection;
    if (case_name == "omitted_header" || case_name == "omitted_config") {
      const auto omitted = case_name == "omitted_header" ? header : config;
      Json retained = Json::array();
      for (const auto& entry : receipt.at("files")) if (entry.at("path") != omitted) retained.push_back(entry);
      receipt["files"] = retained;
      write(prefix / omitted, read(prefix / omitted) + "\n" +
        (case_name == "omitted_header" ? "//" : "#") + " changed unowned bytes\n");
      rejection = case_name == "omitted_header" ? "SDK exported public file is not receipt-owned" : "SDK exported CMake file is not receipt-owned";
    } else if (case_name == "duplicate_path") {
      receipt["files"].push_back(receipt.at("files").at(0));
      rejection = "SDK receipt repeats an owned path";
    }
    if (!rejection.empty()) write_json(prefix / relative, seal(receipt, "receipt_digest"));
    const std::vector<std::string> command{"cmake", "-S", (source / "modules/sciv-engine").string(),
      "-B", (temporary.path / case_name / "build").string(), "-G", "Ninja",
      "-DCMAKE_CXX_COMPILER=" + compiler, "-DBUILD_TESTING=OFF",
      "-DSYMPHONY_SNV_USE_INSTALLED_DEPENDENCIES=ON",
      "-DSymphonySnvCommon_DIR=" + (prefix / config).parent_path().string(),
      "-DSymphonyKnowledgeVectorEngine_DIR=" + (prefix / foundation_config).parent_path().string(),
      "-DCMAKE_PREFIX_PATH=" + prefix.string(), "-DCMAKE_BUILD_TYPE=Release"};
    const auto result = native_test::run(command, "", 120);
    const bool passed = rejection.empty() ? result.returncode == 0 :
      result.returncode != 0 && (result.stdout_text + result.stderr_text).find(rejection) != std::string::npos;
    results.push_back(Json{{"case", case_name}, {"passed", passed}, {"command", command},
      {"exit", result.returncode}, {"expected_rejection", rejection},
      {"stdout", result.stdout_text}, {"stderr", result.stderr_text}});
  }
  const bool preserved = before == source_snapshot(original, original_receipt) &&
    original_receipt_bytes == read(original / relative);
  Json report{{"protocol", "symphony.snv.sdk-admission-test.v1"}, {"source_prefix", original.string()},
    {"original_prefix_modified", !preserved}, {"temporary_cases_removed_on_exit", true}, {"results", results}};
  if (args.has("output")) write_json(args.require("output"), report);
  NT_REQUIRE(preserved);
  for (const auto& result : results) NT_REQUIRE(result.at("passed") == true);
  std::cout << "PASS native SDK valid/omitted-header/omitted-config/duplicate-path admission and original-prefix preservation\n";
}
}

int main(int argc, char** argv) {
  return native_test::test_main([&] { test_sdk_receipt_admission(Arguments(argc, argv)); });
}
