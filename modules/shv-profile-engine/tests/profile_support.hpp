#pragma once
#include "native_test.hpp"
namespace profile_test {
using namespace native_test;
class Process {
public:
  Arguments args;
  fs::path root;
  Json calls = Json::array();
  bool save_cases = false;
  bool include_exit_code = true;
  std::string request_id = "profile-check";
  std::string request_file = "request.json";
  Process(const Arguments &selected, const fs::path &directory)
      : args(selected), root(directory), save_cases(selected.has("cases")) {}
  Json call(const std::string &op, Json p, bool good = true) {
    auto req = request("symphony-shv-profile", op, p, request_id);
    ProcessResult process;
    if (args.has("qxctl")) {
      std::vector<std::string> route;
      if (op == "inspect")
        route = {"profile", "inspect"};
      else if (op == "extraction_diagnose")
        route = {"mapping", "diagnose-source"};
      else if (op == "references_analyze")
        route = {"references", "analyze"};
      else {
        auto at = op.find('_');
        NT_REQUIRE(at != std::string::npos);
        route = {op.substr(0, at), op.substr(at + 1)};
      }
      auto path = root / request_file;
      write(path, canonical(p));
      std::vector<std::string> cmd = {args.require("qxctl"), "shv"};
      cmd.insert(cmd.end(), route.begin(), route.end());
      cmd.insert(cmd.end(), {"--prefix", args.require("prefix"), "--version",
                             args.get("version", "0.2.0-dev"), "--json"});
      if (op != "inspect")
        cmd.insert(cmd.end(), {"--input", path.string()});
      process = native_test::run(cmd);
    } else
      process = native_test::run({args.require("engine")}, canonical(req));
    require((process.returncode == 0) == good,
            op + ": " + process.stdout_text + process.stderr_text);
    auto out = Json::parse(process.stdout_text);
    if (!args.has("qxctl"))
      NT_REQUIRE(out == seal(out, "response_digest"));
    auto result = out.value("result", Json(nullptr));
    if (good)
      NT_REQUIRE(result == seal(result, op == "inspect" ? "descriptor_digest"
                                                        : "digest"));
    Json item = {{"operation", op},
                 {"input", p},
                 {"good", good},
                 {"result", result},
                 {"exit_code", process.returncode}};
    if (!include_exit_code)
      item.erase("exit_code");
    calls.push_back(item);
    if (save_cases) {
      std::ostringstream name;
      name << std::setfill('0') << std::setw(3) << calls.size() << ".json";
      write(root / name.str(), canonical(item));
    }
    return result;
  }
  void summary(Json extra = Json::object()) {
    int rejected = 0;
    for (auto c : calls)
      if (c["good"] == false)
        ++rejected;
    Json summary = {{"status", "passed"},
                    {"calls", calls.size()},
                    {"rejections", rejected}};
    summary.update(extra);
    if (save_cases)
      write_json(root / "SUMMARY.json", summary);
    std::cout << summary.dump() << '\n';
  }
};
} // namespace profile_test
