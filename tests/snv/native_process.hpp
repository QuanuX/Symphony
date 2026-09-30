#pragma once
#include "native_test.hpp"

namespace snv_native_test {
using namespace native_test;

inline void test_installed_requests(const Arguments& args,
                                    const std::string& owner) {
    const auto engine = "symphony-" + owner;
    const auto binary = installed_engine(args.require("prefix"), owner + "-engine",
                                         engine, "0.1.0-dev", "vector_engine", owner);
    std::ifstream input(args.require("input"));
    const auto payload = Json::parse(input);
    const auto operation = args.require("operation");
    TempDir temporary("snv-native-process-");
    auto descriptor_call = native_test::run({binary, "--descriptor"});
    NT_REQUIRE(descriptor_call.returncode == 0 && descriptor_call.stderr_text.empty());
    const auto descriptor = Json::parse(descriptor_call.stdout_text);
    NT_REQUIRE(descriptor == seal(descriptor, "descriptor_digest"));
    NT_REQUIRE(descriptor["engine_id"] == engine && descriptor["engine_version"] == "0.1.0-dev");
    bool advertised = false;
    for (const auto& item : descriptor["operations"])
        if (item["operation_name"] == operation) advertised = true;
    NT_REQUIRE(advertised);

    auto req = request(engine, operation, payload, "snv-native-process");
    req["deadline_unix_ms"] = now_ms() + 5000;
    auto completed = native_test::run({binary}, canonical(req), 8, {}, temporary.path, true);
    NT_REQUIRE(completed.returncode == 0 && completed.stderr_text.empty());
    const auto response = Json::parse(completed.stdout_text);
    NT_REQUIRE(response == seal(response, "response_digest"));
    NT_REQUIRE(response["request_id"] == req["request_id"] && response["correlation_id"] == req["correlation_id"] &&
               response["operation"] == operation && response["engine_id"] == engine && response["engine_version"] == "0.1.0-dev" && response["outcome"] == "ok");
    const auto result = response["result"];
    NT_REQUIRE(result["owner"] == (owner == "snv" ? "symphony-snv" : owner) && result["owner_version"] == "0.1.0-dev" &&
               result["source_digest"] == digest(canonical(payload)));
    if (owner == "snv") {
        NT_REQUIRE(result["subject_ids"].is_object() && result["coverage"].is_object());
    } else {
        NT_REQUIRE(result["subject_ids"].is_array());
    }

    req["payload"]["credential"] = "private-marker";
    completed = native_test::run({binary}, canonical(req), 8, {}, temporary.path, true);
    NT_REQUIRE(completed.returncode != 0 && completed.stderr_text.empty() &&
               completed.stdout_text.find("private-marker") == std::string::npos);
    auto refusal = Json::parse(completed.stdout_text);
    NT_REQUIRE(refusal["outcome"] == "error" && refusal["result"].is_null());
    req["payload"] = payload;
    req["target_engine"] = "symphony-unselected";
    completed = native_test::run({binary}, canonical(req), 8, {}, temporary.path, true);
    NT_REQUIRE(completed.returncode != 0 && Json::parse(completed.stdout_text)["error"]["code"] == "engine.target_mismatch");
    completed = native_test::run({binary}, "{\"x\":1,\"x\":2}", 8, {}, temporary.path, true);
    NT_REQUIRE(completed.returncode != 0 && Json::parse(completed.stdout_text)["error"]["code"] == "json.duplicate_key");
    NT_REQUIRE(fs::is_empty(temporary.path));
    std::cout << "PASS " << owner << " receipt-backed installed native process and refusal boundaries\n";
}
}
