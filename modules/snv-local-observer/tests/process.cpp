#include "native_test.hpp"

namespace {
using namespace native_test;
void test_installed_observation(const Arguments& args) {
  const auto prefix = fs::absolute(args.require("prefix"));
  const std::string module = "snv-local-observer", collector = "symphony-snv-local-observer";
  const std::string release = "0.1.0-dev";
  const auto receipt = Json::parse(receipt_read(prefix,
    "share/symphony/receipts/snv-local-observer/0.1.0-dev/install-receipt.json", 262144).bytes);
  NT_REQUIRE(receipt == seal(receipt, "receipt_digest"));
  NT_REQUIRE(receipt.at("protocol") == "symphony.knowledge.install-receipt.v2" &&
    receipt.at("format_version") == 2 && receipt.at("component_id") == module &&
    receipt.at("module_id") == module && receipt.at("package_id") == module &&
    receipt.at("component_kind") == "module" && receipt.at("version") == release &&
    receipt.at("engine_id").is_null() && receipt.at("vector_id").is_null());
  const std::string relative = "libexec/symphony/snv-local-observer/0.1.0-dev/symphony-snv-local-observer";
  NT_REQUIRE(receipt.at("entry_points") == Json::array({Json{{"entry_point_id", collector},
    {"kind", "executable"}, {"path", relative},
    {"protocols", Json::array({"symphony.knowledge.engine-process.v1"})}}}));
  Json owned = Json::array();
  for (const auto& file : receipt.at("files")) if (file.at("path") == relative) owned.push_back(file);
  NT_REQUIRE(owned.size() == 1 && owned.at(0).at("kind") == "executable");
  const auto bytes = receipt_read(prefix, relative, 67108864);
  NT_REQUIRE((bytes.info.st_mode & 0111) != 0 && bytes.bytes.size() == owned.at(0).at("size") &&
    digest(bytes.bytes) == owned.at(0).at("digest").get<std::string>());
  const auto binary = (prefix / relative).string();
  TempDir temporary("snv-observer-process-");
  auto called = native_test::run({binary, "--descriptor"}, "", 8, {}, temporary.path, true);
  NT_REQUIRE(called.returncode == 0 && called.stderr_text.empty());
  const auto descriptor = Json::parse(called.stdout_text);
  NT_REQUIRE(descriptor == seal(descriptor, "descriptor_digest") &&
    descriptor.at("module_id") == module && descriptor.at("engine_id") == collector &&
    descriptor.at("vector_id").is_null() && descriptor.at("operations").size() == 1 &&
    descriptor.at("operations").at(0).at("operation_name") == "observe");
  std::ifstream input(args.require("input"));
  const auto payload = Json::parse(input);
  auto envelope = request(collector, "observe", payload, "observer-process");
  envelope["deadline_unix_ms"] = now_ms() + 5000;
  called = native_test::run({binary}, canonical(envelope), 8, {}, temporary.path, true);
  NT_REQUIRE(called.returncode == 0 && called.stderr_text.empty());
  const auto response = Json::parse(called.stdout_text);
  NT_REQUIRE(response == seal(response, "response_digest") && response.at("outcome") == "ok" &&
    response.at("request_id") == envelope.at("request_id") &&
    response.at("correlation_id") == envelope.at("correlation_id") &&
    response.at("operation") == "observe" && response.at("engine_id") == collector);
  const auto& result = response.at("result");
  NT_REQUIRE(result.at("owner") == "local-observer" && result.at("acquisition_route") == "native_fixed_sources" &&
    result.at("owner_version") == release && result.at("source_digest") == digest(canonical(payload)) &&
    result.at("observation_id") == payload.at("observation_id") && result.at("node_ref") == payload.at("node_ref") &&
    result.at("fields").size() == payload.at("fields").size() &&
    result.at("physical_inventory_complete") == false && result.at("allocation_verified") == false &&
    result.at("capture").at("atomic_inventory") == false && !result.contains("raw_boot_id"));
  envelope["payload"]["credential"] = "private-evidence-marker";
  called = native_test::run({binary}, canonical(envelope), 8, {}, temporary.path, true);
  NT_REQUIRE(called.returncode != 0 && called.stderr_text.empty() &&
    called.stdout_text.find("private-evidence-marker") == std::string::npos);
  const auto refusal = Json::parse(called.stdout_text);
  NT_REQUIRE(refusal.at("outcome") == "error" && refusal.at("result").is_null() &&
    refusal.at("request_id") == envelope.at("request_id") &&
    refusal.at("correlation_id") == envelope.at("correlation_id"));
  envelope["payload"] = payload;
  envelope["target_engine"] = "symphony-snv";
  called = native_test::run({binary}, canonical(envelope), 8, {}, temporary.path, true);
  NT_REQUIRE(called.returncode != 0 && Json::parse(called.stdout_text).at("error").at("code") == "engine.target_mismatch");
  envelope["target_engine"] = collector;
  envelope["deadline_unix_ms"] = now_ms() - 1;
  called = native_test::run({binary}, canonical(envelope), 8, {}, temporary.path, true);
  NT_REQUIRE(called.returncode != 0 && Json::parse(called.stdout_text).at("error").at("code") == "request.deadline_expired");
  NT_REQUIRE(fs::is_empty(temporary.path));
  std::cout << "PASS independently installed observer module/process ownership and refusals\n";
}
}
int main(int argc, char** argv) {
  return native_test::test_main([&] { test_installed_observation(native_test::Arguments(argc, argv)); });
}
