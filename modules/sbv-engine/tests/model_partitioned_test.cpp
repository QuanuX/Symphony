#include <private_temp.hpp>
#include "../src/consumer_source.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace store = s::result_store;
using J = s::Json;
std::uint64_t checks = 0;
void check(bool good,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!good)
    throw std::runtime_error("partitioned model assertion " +
                             std::to_string(loc.line()));
}
J read(const std::string &path) {
  std::ifstream f(path);
  J j;
  f >> j;
  return j;
}
store::Reference bundle_ref(const J &r) {
  const auto &v = r.at("storage").at("reference");
  return {v.at("manifest_path"), v.at("manifest_sha256"),
          v.at("content_sha256")};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  const auto templates = read(argv[1]).at("templates");
  const bool large = std::getenv("SYMPHONY_SBV_MODEL_LARGE"),
             bytes_case = std::getenv("SYMPHONY_SBV_COMPARE_BYTES");
  const std::uint64_t count = large ? 65537 : 7;
  auto tmp = symphony::testing::temporary_pattern("sbv-model-partitioned-XXXXXX");
  check(::mkdtemp(tmp.data()));
  const std::string root = tmp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  J options{{"max_page_bytes", nullptr}, {"cache_bytes", "33554432"}};
  unsigned sequence = 0;
  auto output = [&](const std::string &name) {
    auto path = root + "/" + name + "-" + d::dec(sequence++);
    return J{
        {"kind", "partitioned"},
        {"bundle_path", path},
        {"workspace_path", path + "-work"},
        {"write_options", {{"page_bytes", "16384"}, {"index_fanout", "16"}}},
        {"scratch_read_options", options}};
  };
  auto tag = [&](const J &receipt, const std::string &pointer = "") {
    return J{{"kind", "bundle"},
             {"reference", receipt.at("storage").at("reference")},
             {"selector", {{"kind", "pointer"}, {"pointer", pointer}}},
             {"read_options", options}};
  };
  auto publish = [&](J body, const std::string &name) {
    body["sections"]["summary"] = d::section(J{{"fixture", name}});
    auto receipt = d::partitioned_result(
        {{"output", output(name)}}, "fit", e::no_deadline,
        [&](d::PartitionedOutput &) { return s::logical::Value(body); });
    check(receipt.at("status") != "recovery_required");
    return receipt;
  };
  auto call = [&](const std::string &op, J p) {
    p.erase("output_path");
    p["output"] = output(op);
    std::cout << "begin " << op << ' ' << sequence << std::endl;
    auto r = s::dispatch(op, p, e::no_deadline);
    if (r.at("status") == "recovery_required")
      throw std::runtime_error(r.dump());
    return r;
  };
  auto materialize = [&](const J &r) {
    store::ResultReader reader(bundle_ref(r), {std::nullopt, 33554432});
    return reader.select_node(0).read_value();
  };
  auto classical = [&](const std::string &op, J p) {
    p.erase("output");
    p["output_path"] = root + "/classical-" + d::dec(sequence++) + ".json";
    auto receipt = s::dispatch(op, p, e::no_deadline);
    return std::pair{read(p.at("output_path")), receipt};
  };
  auto remove = [&](const J &receipt) {
    const auto path = bundle_ref(receipt).manifest_path;
    std::filesystem::remove_all(path.substr(0, path.rfind('/')));
  };
  auto row = [](std::uint64_t i) {
    return J{{"id", "observation-" + d::dec(i)},
             {"x", d::dec(i % 3)},
             {"y", d::dec(1 + 2 * (i % 3))},
             {"caller_extension", {{"retained_meaning", "not a target"}}}};
  };
  auto body = d::base(
      "independent exact line y=1+2x, uniform weights and repeated x values");
  body["sections"]["summary"] = d::section(J{{"rows", d::dec(count)}});
  auto source = d::partitioned_result(
      {{"output", output("observations")}}, "fit", e::no_deadline,
      [&](d::PartitionedOutput &context) {
        auto spool = context.spool("observations");
        for (std::uint64_t i = 0; i < count; ++i)
          spool->append(row(i));
        return s::logical::Value(body).with(
            "sections",
            s::logical::Value(body["sections"])
                .with("observations", d::consumer::section(s::logical::Value(
                                          spool->close().rows))));
      });
  check(source.at("status") != "recovery_required");
  const auto rows_pointer = "/sections/observations/data";
  auto fit = templates.at("fit");
  fit["source"] = tag(source, rows_pointer);
  store::ResultReader source_reader(bundle_ref(source),
                                    {std::nullopt, 33554432});
  fit["source"]["selector"] = {
      {"kind", "node_id"},
      {"node_id", source_reader.select(rows_pointer).describe().at("node_id")}};
  auto fitted = call("fit", fit);
  store::ResultReader fit_reader(bundle_ref(fitted), {std::nullopt, 33554432});
  check(fit_reader.select("/sections/model/data/coefficients").read_value() ==
        J::array({{{"numerator", "1"}, {"denominator", "1"}},
                  {{"numerator", "2"}, {"denominator", "1"}}}));
  check(fit_reader.select("/sections/model/data/training/observation_ids")
            .describe()["children"] == d::dec(count));
  check(fit_reader
            .select("/sections/fit_diagnostics/data/weighted_squared_error_sum")
            .read_value() == J({{"numerator", "0"}, {"denominator", "1"}}));
  check(fit_reader.select("/sections/model/data/training/source_pointer")
            .read_value()
            .is_null());
  J old_fit, old_fit_receipt;
  if (!large) {
    J rows = J::array();
    for (std::uint64_t i = 0; i < count; ++i)
      rows.push_back(row(i));
    body["sections"]["observations"] = d::section(rows);
    auto sealed = s::seal_result(body);
    const auto path = root + "/source.json";
    d::create_file(path, sealed.dump(), e::no_deadline);
    auto old = fit;
    old["source"] = {{"path", path},
                     {"expected_sha256", sealed.at("content_sha256")},
                     {"pointer", rows_pointer}};
    std::tie(old_fit, old_fit_receipt) = classical("fit", old);
    auto current = materialize(fitted);
    for (auto name :
         {"training_predictions", "fit_diagnostics", "exclusions", "summary"})
      check(current["sections"][name] == old_fit["sections"][name]);
  }
  auto predict = templates.at("predict");
  predict["source"] = tag(source, rows_pointer);
  predict["model"] = tag(fitted);
  predict["model"]["selector"] = {
      {"kind", "node_id"},
      {"node_id",
       fit_reader.select("/sections/model/data").describe().at("node_id")}};
  auto predicted = call("predict", predict);
  store::ResultReader predicted_reader(bundle_ref(predicted),
                                       {std::nullopt, 33554432});
  check(predicted_reader.select("/sections/predictions/data")
            .describe()["children"] == d::dec(count));
  check(predicted_reader.select("/sections/training_overlap/data/reused_count")
            .read_value() == d::dec(count));
  check(predicted_reader.select("/sections/predictions/data/0/prediction")
            .read_value() == J({{"numerator", "1"}, {"denominator", "1"}}));
  if (!large) {
    auto old = predict;
    old["model"] = {{"path", old_fit_receipt.at("path")},
                    {"expected_sha256", old_fit_receipt.at("content_sha256")},
                    {"pointer", "/sections/model/data"}};
    auto result = classical("predict", old).first,
         modern = materialize(predicted);
    for (auto name : {"predictions", "studies", "training_overlap", "summary"})
      check(result["sections"][name] == modern["sections"][name]);
  }
  remove(source);
  std::filesystem::remove(root + "/source.json");
  // More candidates and objectives than the old policies, exact tied ranks.
  J candidates = J::array(), objectives = J::array();
  const auto candidate_count = large ? 3 : 129;
  for (int i = 0; i < candidate_count; ++i)
    candidates.push_back({{"id", "candidate-" + d::dec(i)},
                          {"state", "completed"},
                          {"reason", ""},
                          {"source", tag(fitted)},
                          {"parameters", J::object()},
                          {"lineage", J::object()}});
  for (unsigned i = 0; i < 17; ++i) {
    auto o = templates.at("compare").at("objectives")[0];
    o["id"] = "objective-" + d::dec(i);
    o["pointer"] = "/sections/fit_diagnostics/data/weighted_squared_error_sum";
    objectives.push_back(o);
  }
  auto choices = d::base("caller comparison choices");
  choices["sections"]["candidates"] = d::section(candidates);
  choices["sections"]["objectives"] = d::section(objectives);
  auto choices_result = publish(choices, "comparison-choices");
  auto compare = templates.at("compare");
  compare["candidates"] = {
      {"kind", "referenced_rows"},
      {"source", tag(choices_result, "/sections/candidates/data")}};
  compare["objectives"] = {
      {"kind", "referenced_rows"},
      {"source", tag(choices_result, "/sections/objectives/data")}};
  auto compared = call("compare", compare);
  store::ResultReader compared_reader(bundle_ref(compared),
                                      {std::nullopt, 33554432});
  check(compared_reader.select("/sections/comparisons/data/candidates")
            .describe()["children"] == d::dec(candidate_count));
  check(compared_reader
            .select("/sections/comparisons/data/pareto_fronts/0/candidate_ids")
            .describe()["children"] == d::dec(candidate_count));
  check(compared_reader
            .select("/sections/comparisons/data/candidates/1/weighted_rank")
            .read_value() == "1");
  J entries = J::array({{{"id", "fit"},
                         {"kind", "fit"},
                         {"source", tag(fitted)},
                         {"role", "exploration"},
                         {"extensions", J::object()}},
                        {{"id", "predict"},
                         {"kind", "predict"},
                         {"source", tag(predicted)},
                         {"role", "same-observation evaluation"},
                         {"extensions", J::object()}},
                        {{"id", "same-fit"},
                         {"kind", "fit"},
                         {"source", tag(fitted)},
                         {"role", "repeat reference, not repeated execution"},
                         {"extensions", J::object()}}});
  J selections = J::array({{{"id", "choice"},
                            {"source", tag(compared)},
                            {"chosen_candidate_ids", J::array({"candidate-1"})},
                            {"reason", "user preference"},
                            {"extensions", {{"unknown", "preserved"}}}}});
  auto history_choices = d::base("caller research ledger");
  history_choices["sections"]["entries"] = d::section(entries);
  history_choices["sections"]["selections"] = d::section(selections);
  auto ledger = publish(history_choices, "ledger");
  auto history = templates.at("research_history");
  history["entries"] = {{"kind", "referenced_rows"},
                        {"source", tag(ledger, "/sections/entries/data")}};
  history["selections"] = {
      {"kind", "referenced_rows"},
      {"source", tag(ledger, "/sections/selections/data")}};
  auto recorded = call("research_history", history);
  store::ResultReader recorded_reader(bundle_ref(recorded),
                                      {std::nullopt, 33554432});
  check(recorded_reader.select("/sections/summary/data/counted_entries")
            .read_value() == "2");
  check(recorded_reader.select("/sections/summary/data/collapsed_entries")
            .read_value() == "1");
  check(recorded_reader
            .select("/sections/summary/data/reused_namespaced_observations")
            .read_value() == d::dec(count));
  check(recorded_reader.select("/sections/observation_usage/data")
            .describe()["children"] == d::dec(count));
  check(recorded_reader
            .select("/sections/selection_history/data/0/extensions/unknown")
            .read_value() == "preserved");
  if (!large) {
    auto old = history;
    old["entries"] = entries;
    old["selections"] = selections;
    auto a = classical("research_history", old).first,
         b = materialize(recorded);
    for (auto name : {"summary", "research_usage", "observation_usage",
                      "selection_history"})
      check(a["sections"][name] == b["sections"][name]);
  }
  remove(fitted);
  remove(predicted);
  remove(compared);
  remove(ledger);
  remove(choices_result);
  store::ResultReader portable(bundle_ref(recorded), {std::nullopt, 33554432});
  check(portable.verify_closure()["verification_extent"] ==
        "full_logical_closure");
  if (bytes_case) {
    auto wide = d::base("legacy aggregate byte accounting qualification");
    wide["sections"]["metrics"] =
        d::section(J{{"gain", {{"numerator", "1"}, {"denominator", "1"}}}});
    J padding = J::array();
    for (unsigned i = 0; i < 33; ++i)
      padding.push_back(std::string(65536, 'x'));
    wide["sections"]["padding"] = d::section(padding);
    wide = s::seal_result(wide);
    const auto path = root + "/wide.json";
    d::create_file(path, wide.dump(), e::no_deadline);
    auto q = templates.at("compare");
    q["objectives"][0]["pointer"] = "/sections/metrics/data/gain";
    q["candidates"] = J::array();
    for (unsigned i = 0; i < 129; ++i)
      q["candidates"].push_back({{"id", d::dec(i)},
                                 {"state", "completed"},
                                 {"reason", ""},
                                 {"path", path},
                                 {"expected_sha256", wide.at("content_sha256")},
                                 {"parameters", J::object()},
                                 {"lineage", J::object()}});
    auto result = call("compare", q);
    store::ResultReader reader(bundle_ref(result), {std::nullopt, 33554432});
    check(d::u64(reader.select("/sections/resources/data/source_bytes_read")
                     .read_value()) > 268435456);
    std::cout << "comparison aggregate source bytes passed" << std::endl;
  }
  std::cout << "model checks " << checks << " rows " << count << std::endl;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
