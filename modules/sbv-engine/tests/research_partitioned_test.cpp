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
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("partitioned research assertion " +
                             std::to_string(at.line()));
}
J read(const std::string &path) {
  std::ifstream file(path);
  J j;
  file >> j;
  return j;
}
store::Reference reference(const J &receipt) {
  const auto &r = receipt.at("storage").at("reference");
  return {r.at("manifest_path"), r.at("manifest_sha256"),
          r.at("content_sha256")};
}
int main(int argc, char **argv) try {
  check(argc == 2);
  const auto templates = read(argv[1]).at("templates");
  const bool large = std::getenv("SYMPHONY_SBV_RESEARCH_LARGE");
  char temporary[] = "/private/tmp/sbv-research-partitioned-XXXXXX";
  check(::mkdtemp(temporary));
  const std::string root = temporary;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  J options{{"max_page_bytes", nullptr}, {"cache_bytes", "16777216"}};
  auto output = [&](const std::string &name) {
    return J{
        {"kind", "partitioned"},
        {"bundle_path", root + "/" + name},
        {"workspace_path", root + "/" + name + "-work"},
        {"write_options", {{"page_bytes", "16384"}, {"index_fanout", "16"}}},
        {"scratch_read_options", options}};
  };
  auto row = [](std::uint64_t i) {
    return J{{"id", d::dec(i)},
             {"value", "1"},
             {"start_ns", d::dec(100 + i * 10)},
             {"end_ns", d::dec(100 + i * 10)},
             {"available_ns", d::dec(100 + i * 10)},
             {"optional", J{{"not_predicted", true}}}};
  };
  auto body =
      d::base("independent constant observations and separated point labels");
  J small = J::array();
  for (unsigned i = 0; i < 6; ++i)
    small.push_back(row(i));
  body["sections"]["values"] = d::section(small);
  body["sections"]["summary"] = d::section(J{{"rows", "6"}});
  body["sections"]["choices"] = d::section(J{{"caller", "fixture"}});
  body["sections"]["provenance"] =
      d::section(J{{"claim", "mechanical; no market calibration"}});
  const auto sealed = s::seal_result(body);
  d::create_file(root + "/source.json", sealed.dump(), e::no_deadline);
  auto source = d::partitioned_result(
      {{"output", output("source")}}, "run", e::no_deadline,
      [&](d::PartitionedOutput &) { return s::logical::Value(body); });
  check(source["content_sha256"] == sealed["content_sha256"]);
  auto tag = [&](const J &receipt) {
    return J{{"kind", "bundle"},
             {"reference", receipt["storage"]["reference"]},
             {"selector", {{"kind", "pointer"}, {"pointer", ""}}},
             {"read_options", options}};
  };
  auto request = [&](const std::string &op) {
    J q = templates.at(op);
    q["path"] = root + "/source.json";
    q["expected_sha256"] = sealed["content_sha256"];
    q["pointer"] = "/sections/values/data";
    if (op == "analyze") {
      q["value_pointer"] = "/value";
      q["value_type"] = "integer";
      q["weighting"] = "equal_probability";
      q["weight_pointer"] = nullptr;
      q["series_role"] = "equity";
      q["studies"] = J::array({"series_summary", "series_moments",
                               "series_quantiles", "equity_drawdown"});
      q["quantiles"] = J::array({{{"numerator", "1"}, {"denominator", "2"}}});
      q["variance_estimator"] = "population";
    } else if (op == "resample") {
      q["value_pointer"] = "/value";
      q["value_type"] = "integer";
      q["replicates"] = "3";
      q["sample_size"] = "7";
      q["block_size"] = "1";
      q["scheme"] = "iid";
      q["workers"] = "2";
      q["retain_indices"] = true;
    } else {
      q["id_pointer"] = "/id";
      q["start_pointer"] = "/start_ns";
      q["end_pointer"] = "/end_ns";
      q["available_pointer"] = "/available_ns";
      q["plan"] = {{"kind", "explicit"},
                   {"folds", J::array({{{"id", "fold"},
                                        {"train_start_ns", "0"},
                                        {"train_end_ns", "130"},
                                        {"test_start_ns", "130"},
                                        {"test_end_ns", "10000000"},
                                        {"fit_cutoff_ns", "130"}}})}};
      q["chronology"] = "past_only";
      q["purge"] = "label_overlap";
      q["purge_before_ns"] = "0";
      q["purge_after_ns"] = "0";
      q["embargo_ns"] = "0";
      q["availability"] = "by_fit_cutoff";
      q["retain_rows"] = true;
    }
    return q;
  };
  std::vector<J> produced;
  unsigned sequence = 0;
  auto run = [&](const std::string &op, J q) {
    q.erase("path");
    q.erase("expected_sha256");
    q.erase("output_path");
    if (!q.contains("source"))
      q["source"] = tag(source);
    q["output"] = output(op + "-" + d::dec(++sequence));
    std::cout << "begin " << op << " case " << sequence << std::endl;
    auto result = s::dispatch(op, q, e::no_deadline);
    check(result["status"] == "completed");
    std::cout << "completed " << op << " case " << sequence << std::endl;
    produced.push_back(result);
    return result;
  };
  for (const auto *op : {"analyze", "resample", "split"}) {
    auto q = request(op);
    q["output_path"] = root + "/legacy-" + op + ".json";
    s::dispatch(op, q, e::no_deadline);
    auto classical = read(q["output_path"]);
    auto receipt = run(op, q);
    store::ResultReader reader(reference(receipt), {std::nullopt, 16777216});
    auto modern = reader.select_node(0).read_value();
    for (const auto &[name, value] : classical["sections"].items())
      if (name != "choices" && name != "resources")
        check(modern["sections"][name] == value);
  }
  if (large) {
    constexpr std::uint64_t count = 65537;
    auto big_source = d::partitioned_result(
        {{"output", output("large-source")}}, "run", e::no_deadline,
        [&](d::PartitionedOutput &context) {
          auto spool = context.spool("source-rows");
          for (std::uint64_t i = 0; i < count; ++i)
            spool->append(row(i));
          auto rows = spool->close();
          return s::logical::Value(body).with(
              "sections",
              s::logical::Value(body["sections"])
                  .with("values",
                        d::consumer::section(s::logical::Value(rows.rows))));
        });
    check(big_source["status"] == "completed");
    auto analysis = request("analyze");
    analysis["source"] = tag(big_source);
    auto result = run("analyze", analysis);
    store::ResultReader reader(reference(result), {std::nullopt, 16777216});
    check(reader.select("/sections/series/data").describe()["children"] ==
          "65537");
    auto studies = reader.select("/sections/studies/data");
    check(studies.child(0).child("data").child("weighted_sum").read_value() ==
          J({{"numerator", "1"}, {"denominator", "1"}}));
    check(studies.child(1)
              .child("data")
              .child("variance")
              .child("value")
              .read_value() == J({{"numerator", "0"}, {"denominator", "1"}}));
    check(
        studies.child(3).child("data").child("curve").describe()["children"] ==
        "65537");
    auto bootstrap = request("resample");
    bootstrap["replicates"] = "1025";
    bootstrap["sample_size"] = "1025";
    bootstrap["retain_indices"] = false;
    result = run("resample", bootstrap);
    store::ResultReader bootstrap_reader(reference(result),
                                         {std::nullopt, 16777216});
    check(bootstrap_reader.select("/sections/resamples/data")
              .describe()["children"] == "1025");
    check(bootstrap_reader.select("/sections/resources/data/sampled_values")
              .read_value() == "1050625");
    bootstrap["source"] = tag(big_source);
    bootstrap["replicates"] = "1";
    bootstrap["sample_size"] = "65537";
    bootstrap["retain_indices"] = true;
    result = run("resample", bootstrap);
    store::ResultReader retained(reference(result), {std::nullopt, 16777216});
    check(retained.select("/sections/resamples/data/0/source_indices")
              .describe()["children"] == "65537");
    check(retained.select("/sections/resamples/data/0/mean").read_value() ==
          J({{"numerator", "1"}, {"denominator", "1"}}));
    auto split = request("split");
    split["source"] = tag(big_source);
    split["retain_rows"] = false;
    auto fold = split["plan"]["folds"][0];
    for (unsigned i = 1; i < 3; ++i) {
      fold["id"] = "fold-" + d::dec(i);
      split["plan"]["folds"].push_back(fold);
    }
    result = run("split", split);
    store::ResultReader partitions(reference(result), {std::nullopt, 16777216});
    check(partitions.select("/sections/resources/data/row_fold_pairs")
              .read_value() == "196611");
    check(partitions.select("/sections/folds/data/2/test_indices")
              .describe()["children"] == "65534");
    auto wide = body;
    J wide_rows = small;
    for (auto &r : wide_rows) {
      r["padding"] = J::array();
      for (unsigned i = 0; i < 8; ++i)
        r["padding"].push_back(std::string(16384, 'x'));
    }
    wide["sections"]["values"] = d::section(wide_rows);
    auto wide_source = d::partitioned_result(
        {{"output", output("wide-source")}}, "run", e::no_deadline,
        [&](d::PartitionedOutput &) { return s::logical::Value(wide); });
    check(wide_source["status"] == "completed");
    split = request("split");
    split["source"] = tag(wide_source);
    fold = split["plan"]["folds"][0];
    for (unsigned i = 1; i < 129; ++i) {
      fold["id"] = "fold-" + d::dec(i);
      split["plan"]["folds"].push_back(fold);
    }
    result = run("split", split);
    store::ResultReader wide_reader(reference(result),
                                    {std::nullopt, 16777216});
    check(wide_reader.select("/sections/folds/data").describe()["children"] ==
          "129");
    check(
        d::u64(wide_reader.select("/sections/resources/data/retained_row_bytes")
                   .read_value()) > 67108864);
    check(wide_reader
              .select(
                  "/sections/folds/data/128/test_rows/2/optional/not_predicted")
              .read_value() == true);
    check(wide_reader.verify_closure()["verification_extent"] ==
          "full_logical_closure");
    std::filesystem::remove_all(root + "/large-source");
    std::filesystem::remove_all(root + "/large-source-work");
    std::filesystem::remove_all(root + "/wide-source");
  }
  std::filesystem::remove_all(root + "/source");
  std::filesystem::remove(root + "/source.json");
  for (std::size_t i = 0; i < 3; ++i) {
    store::ResultReader reader(reference(produced[i]),
                               {std::nullopt, 16777216});
    check(reader.verify_closure()["verification_extent"] ==
          "full_logical_closure");
  }
  std::cout << "research checks " << checks << " large " << large << std::endl;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
