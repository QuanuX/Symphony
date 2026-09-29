#include "native_test.hpp"
using namespace native_test;
void test_installed_requests(const Arguments &args) {
  const auto binary = installed_engine(
      args.require("prefix"), "sqpv-inspection-engine",
      "symphony-sqpv-inspection", "0.1.0-dev", "vector_engine", "sqpv");
  std::ifstream input(args.require("input"));
  const auto payload = Json::parse(input);
  auto req = request("symphony-sqpv-inspection", "store_inspect", payload,
                     "sqv-admin-native");
  req["deadline_unix_ms"] = now_ms() + 4000;
  TempDir temporary("sqv-administration-process-");
  auto completed =
      native_test::run({binary}, canonical(req), 6, {}, temporary.path, true);
  NT_REQUIRE(completed.returncode == 0 && completed.stderr_text.empty());
  auto response = Json::parse(completed.stdout_text);
  NT_REQUIRE(response == seal(response, "response_digest"));
  NT_REQUIRE(response["request_id"] == req["request_id"] &&
             response["correlation_id"] == req["correlation_id"] &&
             response["engine_id"] == "symphony-sqpv-inspection" &&
             response["engine_version"] == "0.1.0-dev" &&
             response["operation"] == "store_inspect" &&
             response["outcome"] == "ok");
  const auto result = response["result"];
  NT_REQUIRE(result == seal(result, "result_digest") &&
             result["request_digest"] == digest(canonical(payload)) &&
             result["owner"] == "sqpv" &&
             result["persistent_mutation"] == false &&
             result["provider_observation"] == "not_performed");
  req["payload"]["credential"] = "private-marker";
  completed =
      native_test::run({binary}, canonical(req), 6, {}, temporary.path, true);
  NT_REQUIRE(completed.returncode == 2 && completed.stderr_text.empty() &&
             completed.stdout_text.find("private-marker") == std::string::npos);
  response = Json::parse(completed.stdout_text);
  NT_REQUIRE(response["outcome"] == "error" && response["result"].is_null() &&
             response["error"]["code"] == "sqv.admin.invalid");
  completed = native_test::run({binary}, "{\"x\":1,\"x\":2}", 6, {},
                               temporary.path, true);
  NT_REQUIRE(completed.returncode == 2 && completed.stderr_text.empty() &&
             Json::parse(completed.stdout_text)["error"]["code"] ==
                 "json.duplicate_key");
  NT_REQUIRE(fs::is_empty(temporary.path));
  std::cout << "PASS sqpv-inspection-engine installed native process, "
               "correspondence and refusal boundaries\n";
}
int main(int argc, char **argv) {
  return test_main([&] { test_installed_requests(Arguments(argc, argv)); });
}
