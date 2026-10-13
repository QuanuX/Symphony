#include <private_temp.hpp>
#include "../src/consumer_source.hpp"
#include <cstdlib>
#include <filesystem>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace store = s::result_store;
using J = s::Json;
std::uint64_t checks = 0;
void check(bool ok,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("selection view assertion " +
                             std::to_string(loc.line()));
}
int main() try {
  auto temporary = symphony::testing::temporary_pattern("sbv-view-XXXXXX");
  check(::mkdtemp(temporary.data()));
  const std::string root = temporary;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  const auto count = std::getenv("SYMPHONY_SBV_VIEW_ROWS")
                         ? d::u64(std::getenv("SYMPHONY_SBV_VIEW_ROWS"))
                         : 33;
  check(count >= 3);
  J read{{"max_page_bytes", nullptr}, {"cache_bytes", "16777216"}};
  auto output = [&](const std::string &name) {
    return J{
        {"kind", "partitioned"},
        {"bundle_path", root + "/" + name},
        {"workspace_path", root + "/" + name + "-work"},
        {"write_options", {{"page_bytes", "65536"}, {"index_fanout", "32"}}},
        {"scratch_read_options", read}};
  };
  auto source = d::partitioned_result(
      {{"output", output("source")}}, "run", e::no_deadline,
      [&](d::PartitionedOutput &context) {
        check(context.read_options().cache_bytes == 16777216 &&
              !context.read_options().max_page_bytes);
        auto spool = context.spool("rows");
        for (std::uint64_t i = 0; i < count; ++i)
          spool->append(J{{"rank", d::dec(i % 3)},
                          {"serial", d::dec(i)},
                          {"unanticipated/key~", J{{"retained", true}}}});
        auto rows = spool->close();
        auto body = d::base("mechanical selection fixture");
        body["sections"]["summary"] = d::section(J{{"count", d::dec(count)}});
        return s::logical::Value(body).with(
            "sections", s::logical::Value(body["sections"])
                            .with("values", d::consumer::section(
                                                s::logical::Value(rows.rows))));
      });
  check(source["status"] == "completed");
  J selected{{"kind", "bundle"},
             {"reference", source["storage"]["reference"]},
             {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
             {"read_options", read}};
  J request{{"protocol", "symphony.sbv.result-select-input.v1"},
            {"source", selected},
            {"pointer", "/sections/values/data"},
            {"filters", J::array()},
            {"order_by", J::array({{{"pointer", "/rank"},
                                    {"type", "integer"},
                                    {"direction", "desc"},
                                    {"missing", "reject"}}})},
            {"columns", J::array()},
            {"ordering_storage", "memory_index"},
            {"max_index_rows", nullptr},
            {"output", output("view")}};
  auto view = s::dispatch("result_select", request, e::no_deadline);
  check(view["status"] == "completed");
  check(view["summary"]["data"]["source_rows"] == d::dec(count));
  check(view["summary"]["data"]["matched_rows"] == d::dec(count));
  std::vector<std::uint64_t> expected;
  for (auto rank : {2, 1, 0})
    for (std::uint64_t i = rank; i < count; i += 3)
      expected.push_back(i);
  // Independent exact order oracle, including ties, projection and missingness.
  auto reject_budget = request;
  reject_budget["max_index_rows"] = "1";
  reject_budget["output"] = output("budget");
  check(s::dispatch("result_select", reject_budget, e::no_deadline)["status"] ==
        "recovery_required");
  auto filtered = request;
  filtered["output"] = output("filtered");
  filtered["order_by"] = J::array();
  filtered["filters"] = J::array({{{"pointer", "/rank"},
                                   {"type", "integer"},
                                   {"op", "eq"},
                                   {"value", "1"},
                                   {"missing", "reject"}}});
  filtered["columns"] =
      J::array({{{"name", "identity"}, {"pointer", "/serial"}},
                {{"name", "optional"}, {"pointer", "/absent"}}});
  const auto subset = s::dispatch("result_select", filtered, e::no_deadline);
  check(subset["status"] == "completed");
  J subset_page{{"protocol", request["protocol"]},
                {"view", subset["storage"]["reference"]},
                {"read_options", read},
                {"limit", "1"},
                {"cursor", ""}};
  auto first = s::dispatch("result_select", subset_page, e::no_deadline);
  check(first["rows"][0]["source_index"] == "1");
  check(first["rows"][0]["value"]["identity"]["value"] == "1");
  check(first["rows"][0]["value"]["optional"]["status"] == "unavailable");
  // The original data and scratch become unavailable. Every continuation must
  // still work using only the immutable prepared view's selected-page closure.
  std::filesystem::remove_all(root + "/source");
  std::filesystem::remove_all(root + "/source-work");
  std::filesystem::remove_all(root + "/view-work");
  for (const auto *size : {"1", "17", "256"}) {
    J page{{"protocol", request["protocol"]},
           {"view", view["storage"]["reference"]},
           {"read_options", read},
           {"limit", size},
           {"cursor", ""}};
    std::uint64_t at = 0, calls = 0, maximum_files = 0;
    for (;;) {
      const auto result = s::dispatch("result_select", page, e::no_deadline);
      check(result["offset"] == d::dec(at));
      check(result["verification_extent"] ==
            "pinned_manifest_and_selected_page_closure");
      maximum_files =
          std::max(maximum_files, d::u64(result["io_stats"]["files_read"]));
      ++calls;
      for (const auto &row : result["rows"]) {
        check(d::u64(row["source_index"]) == expected.at(at));
        check(row["value"]["serial"] == d::dec(expected.at(at)));
        check(row["value"]["unanticipated/key~"]["retained"] == true);
        ++at;
      }
      if (result["complete"] == true)
        break;
      page["cursor"] = result["next_cursor"];
    }
    check(at == count);
    check(maximum_files < 100); // proof that each page does not scan all rows
    std::cout << "rows " << count << " page_size " << size << " pages " << calls
              << " maximum_page_files " << maximum_files << std::endl;
    page["limit"] = "2";
    bool rejected = false;
    try {
      s::dispatch("result_select", page, e::no_deadline);
    } catch (...) {
      rejected = true;
    }
    if (calls > 1)
      check(rejected);
  }
  std::cout << "view checks " << checks << '\n';
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
