#pragma once
#include "native_test.hpp"
namespace native_test {
// Receipt-backed installed-process evidence. This performs no package mutation.
class Evidence {
public:
  Arguments args;
  Json calls = Json::array(), installations = Json::array(),
       verified_receipts = Json::array(), qxctl_inspections = Json::array();
  fs::path directory;
  explicit Evidence(const Arguments &selected)
      : args(selected), directory(fs::absolute(args.require("output"))) {
    args.require("qxctl");
    args.require("prefix");
    args.require("version");
    fs::create_directories(directory);
  }
  Json installation(const std::string &module, const std::string &engine,
                    const std::vector<std::string> &family,
                    const fs::path &receipt_path,
                    const fs::path &supplied_prefix = {},
                    const std::string &supplied_version = "") {
    auto prefix =
        fs::canonical(supplied_prefix.empty() ? fs::path(args.require("prefix"))
                                              : supplied_prefix);
    auto version =
        supplied_version.empty() ? args.require("version") : supplied_version;
    NT_REQUIRE(fs::is_regular_file(receipt_path) &&
               !fs::is_symlink(receipt_path));
    bool connector = module == "shv-graph-duckdb-connector";
    std::vector<std::string> command = {args.require("qxctl"), "shv"};
    command.insert(command.end(), family.begin(), family.end());
    command.insert(command.end(),
                   {"inspect", connector ? "--connector-prefix" : "--prefix",
                    prefix.string(),
                    connector ? "--connector-version" : "--version", version,
                    "--json"});
    auto checked = run(command);
    NT_REQUIRE(checked.returncode == 0);
    require(checked.stderr_text.empty(), checked.stderr_text);
    auto envelope = Json::parse(checked.stdout_text), descriptor = envelope;
    if (envelope.contains("result")) {
      NT_REQUIRE(envelope["outcome"] == "ok" &&
                 envelope == seal(envelope, "response_digest"));
      descriptor = envelope["result"];
    }
    NT_REQUIRE(descriptor == seal(descriptor, "descriptor_digest"));
    NT_REQUIRE(descriptor["engine_id"] == engine &&
               descriptor["engine_version"] == version);
    // qxctl inspect above owns full no-follow compiled-admission verification.
    // Independently bind the emitted evidence to the exact selected receipt
    // bytes.
    auto relative = fs::relative(receipt_path, prefix);
    auto receipt = Json::parse(receipt_read(prefix, relative, 1048576).bytes);
    NT_REQUIRE(receipt == seal(receipt, "receipt_digest"));
    NT_REQUIRE(receipt["module_id"] == module && receipt["version"] == version);
    auto executable = prefix / "libexec/symphony" / module / version / engine;
    Json owned = Json::array();
    for (auto item : receipt["files"])
      if (item["path"] == executable.lexically_relative(prefix).string())
        owned.push_back(item);
    NT_REQUIRE(owned.size() == 1);
    auto binary =
        receipt_read(prefix, executable.lexically_relative(prefix), 4194304);
    NT_REQUIRE(owned[0]["digest"] == digest(binary.bytes));
    NT_REQUIRE(owned[0]["size"] == binary.bytes.size() &&
               (binary.info.st_mode & 0111) != 0);
    Json result = {{"Role", module},
                   {"ModuleID", module},
                   {"EngineID", engine},
                   {"Version", version},
                   {"Prefix", prefix.string()},
                   {"ReceiptPath", receipt_path.string()},
                   {"ReceiptProtocol", receipt["protocol"]},
                   {"ReceiptDigest", receipt["receipt_digest"]},
                   {"ExecutablePath", executable.string()},
                   {"ExecutableDigest", owned[0]["digest"]}};
    installations.push_back(result);
    verified_receipts.push_back(
        {{"path", receipt_path.string()}, {"receipt", receipt}});
    qxctl_inspections.push_back({{"module", module}, {"response", envelope}});
    return result;
  }
  Json response(const ProcessResult &completed, const Json &installation,
                const std::string &operation, const Json &payload, bool success,
                const std::string &error_code = "") {
    require((completed.returncode == 0) == success,
            operation + ": " + completed.stdout_text);
    require(completed.stderr_text.empty(), completed.stderr_text);
    auto value = Json::parse(completed.stdout_text);
    std::set<std::string> keys;
    for (auto it = value.begin(); it != value.end(); ++it)
      keys.insert(it.key());
    NT_REQUIRE(keys == std::set<std::string>(
                           {"protocol", "request_id", "correlation_id",
                            "operation", "engine_id", "engine_version",
                            "outcome", "result", "error", "response_digest"}));
    NT_REQUIRE(value["protocol"] == "symphony.knowledge.engine-process.v1");
    NT_REQUIRE(value["request_id"] == "installed-invariant" &&
               value["correlation_id"] == "installed-invariant");
    NT_REQUIRE(value == seal(value, "response_digest"));
    NT_REQUIRE(value["engine_id"] == installation["EngineID"] &&
               value["engine_version"] == installation["Version"] &&
               value["operation"] == operation);
    NT_REQUIRE(value["outcome"] == (success ? "ok" : "error"));
    if (!success) {
      NT_REQUIRE(!error_code.empty());
      require(value["error"]["code"] == error_code, value.dump());
    } else
      NT_REQUIRE(error_code.empty() && value["error"].is_null());
    calls.push_back({{"operation", operation},
                     {"expected_success", success},
                     {"expected_error_code",
                      error_code.empty() ? Json(nullptr) : Json(error_code)},
                     {"exit_code", completed.returncode},
                     {"payload", payload},
                     {"response", value}});
    return value.value("result", Json(nullptr));
  }
  Json call(const Json &installation, const std::string &operation,
            const Json &payload, const std::string &error_code = "",
            const fs::path &cwd = {}) {
    auto completed = run(
        {installation.at("ExecutablePath").get<std::string>()},
        canonical(native_test::request(installation.at("EngineID"), operation,
                                       payload, "installed-invariant")),
        40, {}, cwd, true);
    return response(completed, installation, operation, payload,
                    error_code.empty(), error_code);
  }
  void finish(const std::string &case_name) {
    int rejected = 0;
    for (auto c : calls)
      if (c["expected_success"] == false)
        ++rejected;
    Json arguments = Json::object();
    for (const auto &[source_key, value] : args.values) {
      std::string key = source_key;
      std::replace(key.begin(), key.end(), '-', '_');
      arguments[key] = value;
    }
    Json result = {{"status", "passed"},
                   {"case", case_name},
                   {"installations", installations},
                   {"verified_receipts", verified_receipts},
                   {"qxctl_inspections", qxctl_inspections},
                   {"arguments", arguments},
                   {"qxctl_digest", digest(read(args.require("qxctl")))},
                   {"calls", calls},
                   {"call_count", calls.size()},
                   {"expected_rejections", rejected},
                   {"installs_or_mutates_packages", false}};
    write_json(directory / "ACCEPTANCE.json", result);
    std::cout << Json({{"status", "passed"},
                       {"case", case_name},
                       {"call_count", calls.size()},
                       {"expected_rejections", rejected}})
                     .dump()
              << '\n';
  }
};
} // namespace native_test
