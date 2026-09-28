#include "native_test.hpp"
using namespace native_test;
void test_installed_requests(const Arguments &args) {
  const auto binary = installed_engine(args.require("prefix"),
      "sqav-request-engine", "symphony-sqav-request", "0.1.0-dev",
      "vector_engine", "sqav");
  TempDir temporary("sqav-request-process-");
  for (const auto *adapter : {"fred", "databento_historical", "databento_reference"}) {
    const auto path = fs::path(SQAV_FIXTURES) / (std::string(adapter) + ".json");
    std::ifstream input(path);
    const auto payload = Json::parse(input);
    auto req = request("symphony-sqav-request", "request_validate", payload, "sqav-native");
    req["deadline_unix_ms"] = now_ms() + 4000;
    auto completed = native_test::run({binary}, canonical(req), 6, {}, temporary.path, true);
    NT_REQUIRE(completed.returncode == 0 && completed.stderr_text.empty());
    auto response = Json::parse(completed.stdout_text);
    NT_REQUIRE(response == seal(response, "response_digest"));
    NT_REQUIRE(response["request_id"] == req["request_id"] &&
        response["correlation_id"] == req["correlation_id"] &&
        response["engine_id"] == req["target_engine"] &&
        response["engine_version"] == "0.1.0-dev" &&
        response["operation"] == "request_validate" && response["outcome"] == "ok");
    const auto result = response["result"];
    NT_REQUIRE(result == seal(result, "result_digest"));
    NT_REQUIRE(result["request_digest"] == digest(canonical(payload)) &&
        result["adapter"] == adapter && result["provider_observation"] == "not_performed");
    req["payload"]["credential"] = "private-marker";
    completed = native_test::run({binary}, canonical(req), 6, {}, temporary.path, true);
    NT_REQUIRE(completed.returncode == 2 && completed.stderr_text.empty());
    NT_REQUIRE(completed.stdout_text.find("private-marker") == std::string::npos);
    response = Json::parse(completed.stdout_text);
    NT_REQUIRE(response["outcome"] == "error" && response["result"].is_null());
    NT_REQUIRE(response["error"]["code"] == "sqav.request.invalid");
  }
  auto rejected = native_test::run({binary}, "{\"x\":1,\"x\":2}", 6, {}, temporary.path, true);
  NT_REQUIRE(rejected.returncode == 2 && rejected.stderr_text.empty());
  NT_REQUIRE(Json::parse(rejected.stdout_text)["error"]["code"] == "json.duplicate_key");
  NT_REQUIRE(fs::is_empty(temporary.path));
  std::cout << "PASS installed receipt-bound SQAV process: three variants, correspondence, refusals, empty work directory\n";
}
int main(int argc, char **argv) {
  return test_main([&] { test_installed_requests(Arguments(argc, argv)); });
}
