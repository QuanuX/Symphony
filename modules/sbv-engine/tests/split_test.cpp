#include "../src/detail.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <limits>
#include <random>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool b, std::source_location l = std::source_location::current()) {
  ++checks;
  if (!b)
    throw std::runtime_error("split assertion " + std::to_string(l.line()));
}
int main() try {
  char tmp[] = "/private/tmp/sbv-split-XXXXXX";
  const auto *env = std::getenv("SYMPHONY_SBV_SPLIT_FIXTURES");
  std::filesystem::path root = env ? env : mkdtemp(tmp);
  std::filesystem::create_directories(root);
  unsigned seq = 0;
  auto path = [&](const std::string &x) { return (root / x).string(); };
  auto read = [&](const std::string &x) {
    auto r = e::parse_bounded_json(d::read_file(x, e::unix_time_ms() + 30000),
                                   d::artifact_bytes, d::artifact_values);
    s::validate_result(r);
    return r;
  };
  auto save_source = [&](J rows) {
    auto r = d::base("mechanical interval fixture, not market labels");
    r["sections"]["observations"] = d::section(rows);
    r = s::seal_result(r);
    auto file = path("source-" + std::to_string(++seq) + ".json");
    d::create_file(file, r.dump(), e::unix_time_ms() + 30000);
    return J{{"path", file}, {"expected_sha256", r.at("content_sha256")}};
  };
  auto obs = [](const char *id, unsigned a, unsigned b, unsigned available) {
    return J{{"id", id},
             {"start_ns", d::dec(a)},
             {"end_ns", d::dec(b)},
             {"available_ns", d::dec(available)},
             {"value", d::dec(a)},
             {"private", J::array({"retain", true})}};
  };
  // Deliberately unsorted; a shared endpoint overlaps, gaps in test labels do
  // not.
  J rows = J::array({obs("after", 41, 42, 44), obs("touch", 10, 20, 20),
                     obs("test-a", 20, 25, 25), obs("early", 1, 9, 9),
                     obs("late", 8, 9, 50), obs("test-b", 30, 40, 40),
                     obs("gap", 27, 28, 28), obs("outside", 90, 95, 95),
                     obs("embargo-edge", 45, 46, 46),
                     obs("after-embargo", 46, 47, 47)});
  auto src = save_source(rows);
  J fold{{"id", "custom"},       {"train_start_ns", "0"},
         {"train_end_ns", "60"}, {"test_start_ns", "20"},
         {"test_end_ns", "26"},  {"fit_cutoff_ns", "48"}};
  J p{{"protocol", "symphony.sbv.split-input.v1"},
      {"path", src["path"]},
      {"expected_sha256", src["expected_sha256"]},
      {"output_path", ""},
      {"pointer", "/sections/observations/data"},
      {"id_pointer", "/id"},
      {"start_pointer", "/start_ns"},
      {"end_pointer", "/end_ns"},
      {"available_pointer", "/available_ns"},
      {"clock_domain", "mechanical_ns"},
      {"input_description", "labels and availability explicitly supplied"},
      {"plan", {{"kind", "explicit"}, {"folds", J::array({fold})}}},
      {"chronology", "unrestricted"},
      {"purge", "label_overlap"},
      {"purge_before_ns", "0"},
      {"purge_after_ns", "0"},
      {"embargo_ns", "5"},
      {"availability", "by_fit_cutoff"},
      {"retain_rows", true},
      {"extensions", {{"private-study", "\u202e\u001b"}}}};
  auto call = [&](J q) {
    q["output_path"] = path("result-" + std::to_string(++seq) + ".json");
    auto receipt = s::dispatch("split", q, e::unix_time_ms() + 30000);
    auto r = read(q["output_path"]);
    check(r["content_sha256"] == receipt["content_sha256"]);
    return r;
  };
  auto reject = [&](J q) {
    q["output_path"] = path("refused-" + std::to_string(++seq) + ".json");
    bool failed = false;
    try {
      s::dispatch("split", q, e::unix_time_ms() + 30000);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(q["output_path"].get<std::string>()));
  };
  auto r = call(p), f = r["sections"]["folds"]["data"][0];
  check(f["train_indices"] == J::array({"0", "3", "8", "9"}));
  check(f["test_indices"] == J::array({"2"}));
  check(f["excluded"][0]["reasons"] == J::array({"label_overlap_or_padding"}));
  check(f["train_rows"][0] == rows[0]);
  check(f["test_rows"][0] == rows[2]);
  check(r["sections"]["user_extensions"]["data"] == p["extensions"]);
  check(f["embargo_interval"]["start_exclusive_ns"] == "25");
  check(f["embargo_interval"]["end_inclusive_ns"] == "30");
  // Disabling filters retains the deliberate overlap and late sample, with
  // diagnostics.
  auto q = p;
  q["purge"] = "none";
  q["availability"] = "ignore";
  q["embargo_ns"] = "0";
  q["retain_rows"] = false;
  auto permissive = call(q)["sections"]["folds"]["data"][0];
  check(permissive["train_indices"] ==
        J::array({"0", "1", "3", "4", "5", "6", "8", "9"}));
  check(permissive["diagnostics"]["retained_overlaps"] == "1");
  check(permissive["diagnostics"]["retained_late"] == "1");
  check(permissive["train_rows"].is_null());
  q["available_pointer"] = nullptr;
  check(call(q)["sections"]["folds"]["data"][0]["diagnostics"]
               ["unknown_availability_candidates"] == "9");
  q["availability"] = "by_fit_cutoff";
  reject(q);
  // Time windows grow/roll independently of sparse or out-of-order sample
  // counts.
  q = p;
  q["chronology"] = "past_only";
  q["embargo_ns"] = "0";
  q["plan"] = {{"kind", "expanding"},
               {"train_start_ns", "0"},
               {"first_train_end_ns", "20"},
               {"first_fit_cutoff_ns", "20"},
               {"gap_ns", "0"},
               {"test_duration_ns", "10"},
               {"step_ns", "10"},
               {"fold_count", "3"}};
  auto expanding = call(q)["sections"]["folds"]["data"];
  check(expanding.size() == 3);
  check(expanding[0]["train_indices"] == J::array({"3"}));
  check(expanding[0]["test_indices"] == J::array({"2", "6"}));
  check(expanding[1]["windows"]["train_start_ns"] == "0");
  check(expanding[1]["train_indices"] == J::array({"1", "2", "3", "6"}));
  q["plan"]["kind"] = "rolling";
  auto rolling = call(q)["sections"]["folds"]["data"];
  check(rolling[1]["windows"]["train_start_ns"] == "10");
  check(rolling[1]["train_indices"] == J::array({"1", "2", "6"}));
  check(rolling[2]["train_indices"] == J::array({"2", "5", "6"}));
  // Brute-force independent oracle: unsorted variable-length labels, both
  // endpoint contacts, duplicate times, test gaps, no-test folds and combined
  // exclusion reasons.
  std::mt19937 rng(7319);
  J random_rows = J::array();
  for (unsigned i = 0; i < 180; ++i) {
    unsigned start = 10 + rng() % 130, finish = start + rng() % 25;
    auto row = obs("temporary", start, finish, finish + rng() % 20);
    row["id"] = d::dec(i);
    random_rows.push_back(row);
  }
  auto random_src = save_source(random_rows);
  q = p;
  q["path"] = random_src["path"];
  q["expected_sha256"] = random_src["expected_sha256"];
  q["purge_before_ns"] = "3";
  q["purge_after_ns"] = "7";
  for (unsigned trial = 0; trial < 24; ++trial) {
    unsigned lo = 20 + rng() % 140, hi = lo + 1 + rng() % 15,
             cutoff = 50 + rng() % 120;
    auto win = fold;
    win["test_start_ns"] = d::dec(lo);
    win["test_end_ns"] = d::dec(hi);
    win["fit_cutoff_ns"] = d::dec(cutoff);
    win["train_end_ns"] = "200";
    q["plan"]["folds"] = J::array({win});
    auto got = call(q)["sections"]["folds"]["data"][0];
    J want_train = J::array(), want_test = J::array(),
      want_excluded = J::array();
    unsigned last = 0;
    for (unsigned i = 0; i < random_rows.size(); ++i) {
      auto a = d::u64(random_rows[i]["start_ns"]);
      if (a >= lo && a < hi) {
        want_test.push_back(d::dec(i));
        last = std::max(
            last, static_cast<unsigned>(d::u64(random_rows[i]["end_ns"])));
      }
    }
    for (unsigned i = 0; i < random_rows.size(); ++i) {
      auto a = d::u64(random_rows[i]["start_ns"]),
           b = d::u64(random_rows[i]["end_ns"]);
      J why = J::array();
      if (a >= lo && a < hi)
        why.push_back("test_member");
      bool hit = false;
      for (const auto &idx : want_test) {
        const auto &t = random_rows[d::u64(idx)];
        hit |= a <= d::u64(t["end_ns"]) + 7 && b >= d::u64(t["start_ns"]) - 3;
      }
      if (hit)
        why.push_back("label_overlap_or_padding");
      if (!want_test.empty() && a > last && a <= last + 5)
        why.push_back("embargo");
      if (d::u64(random_rows[i]["available_ns"]) > cutoff)
        why.push_back("unavailable_at_fit_cutoff");
      if (why.empty())
        want_train.push_back(d::dec(i));
      else
        want_excluded.push_back(
            {{"source_index", d::dec(i)}, {"reasons", why}});
    }
    check(got["train_indices"] == want_train);
    check(got["test_indices"] == want_test);
    check(got["excluded"] == want_excluded);
    for (std::size_t i = 0; i < want_train.size(); ++i)
      check(got["train_rows"][i] == random_rows[d::u64(want_train[i])]);
  }
  // Exact large unsigned timestamps and endpoint overflow never wrap or round.
  auto big = rows;
  const std::uint64_t shift = 9007199254740993ULL;
  for (auto &row : big)
    for (auto k : {"start_ns", "end_ns", "available_ns"})
      row[k] = d::dec(d::u64(row[k]) + shift);
  auto big_src = save_source(big);
  q = p;
  q["path"] = big_src["path"];
  q["expected_sha256"] = big_src["expected_sha256"];
  for (auto k : {"train_start_ns", "train_end_ns", "test_start_ns",
                 "test_end_ns", "fit_cutoff_ns"})
    q["plan"]["folds"][0][k] = d::dec(d::u64(fold[k]) + shift);
  check(call(q)["sections"]["folds"]["data"][0]["train_indices"] ==
        f["train_indices"]);
  for (unsigned bad = 0; bad < 15; ++bad) {
    q = p;
    switch (bad) {
    case 0:
      q["expected_sha256"] = std::string(64, '0');
      break;
    case 1:
      q["chronology"] = "past_only";
      break;
    case 2:
      q["plan"]["folds"][0]["train_end_ns"] = "0";
      break;
    case 3:
      q["plan"]["folds"].push_back(fold);
      break;
    case 4:
      q["purge_before_ns"] = "21";
      break;
    case 5:
      q["purge_after_ns"] = "18446744073709551615";
      break;
    case 6:
      q["embargo_ns"] = "18446744073709551615";
      break;
    case 7:
      q["embargo_ns"] = "-1";
      break;
    case 8:
      q["embargo_ns"] = "01";
      break;
    case 9:
      q["embargo_ns"] = 1;
      break;
    case 10:
      q["purge"] = "none";
      q["purge_before_ns"] = "1";
      break;
    case 11:
      q["start_pointer"] = "/missing";
      break;
    case 12:
      q["start_pointer"] = "invalid pointer";
      break;
    case 13:
      q["extra"] = true;
      break;
    case 14:
      q["plan"]["kind"] = "invented";
      break;
    }
    reject(q);
  }
  auto malformed = rows;
  malformed[1]["id"] = malformed[0]["id"];
  src = save_source(malformed);
  q = p;
  q["path"] = src["path"];
  q["expected_sha256"] = src["expected_sha256"];
  reject(q);
  malformed = rows;
  malformed[0]["end_ns"] = "0";
  src = save_source(malformed);
  q["path"] = src["path"];
  q["expected_sha256"] = src["expected_sha256"];
  reject(q);
  src = save_source(J::array());
  q["path"] = src["path"];
  q["expected_sha256"] = src["expected_sha256"];
  check(call(q)["sections"]["folds"]["data"][0]["status"] == "unavailable");
  q = p;
  q["plan"]["folds"] = J::array();
  check(call(q)["sections"]["folds"]["data"].empty());
  // Generated-window arithmetic, bounds and retention refuse before
  // publication.
  auto generated = p;
  generated["chronology"] = "past_only";
  generated["plan"] = {{"kind", "rolling"},
                       {"train_start_ns", "0"},
                       {"first_train_end_ns", "20"},
                       {"first_fit_cutoff_ns", "20"},
                       {"gap_ns", "0"},
                       {"test_duration_ns", "10"},
                       {"step_ns", "10"},
                       {"fold_count", "3"}};
  for (auto key : {"step_ns", "test_duration_ns", "fold_count"}) {
    auto bad = generated;
    bad["plan"][key] = "0";
    reject(bad);
  }
  q = generated;
  q["plan"]["first_train_end_ns"] = "18446744073709551615";
  reject(q);
  q = generated;
  q["plan"]["first_fit_cutoff_ns"] = "21";
  reject(q);
  q = generated;
  q["plan"]["fold_count"] = "129";
  reject(q);
  q = generated;
  q["plan"]["step_ns"] = "18446744073709551615";
  reject(q);
  q = p;
  q["output_path"] = path("expired.json");
  bool expired = false;
  try {
    s::dispatch("split", q, 0);
  } catch (const std::exception &) {
    expired = true;
  }
  check(expired &&
        !std::filesystem::exists(q["output_path"].get<std::string>()));
  auto payload_rows =
      J::array({obs("large-train", 1, 2, 2), obs("large-test", 21, 22, 22)});
  for (auto &row : payload_rows)
    row["large"] = J::array();
  for (auto &row : payload_rows)
    for (unsigned i = 0; i < 33; ++i)
      row["large"].push_back(std::string(65535, 'x'));
  auto payload = save_source(payload_rows);
  q = p;
  q["path"] = payload["path"];
  q["expected_sha256"] = payload["expected_sha256"];
  for (unsigned i = 1; i < 17; ++i) {
    auto next = fold;
    next["id"] = d::dec(i);
    q["plan"]["folds"].push_back(next);
  }
  reject(q);
  q["retain_rows"] = false;
  check(call(q)["sections"]["folds"]["data"].size() == 17);
  // Split output is directly consumable by native series analysis via retained
  // rows.
  J a;
  std::ifstream templates(std::filesystem::path(__FILE__).parent_path() /
                          "../schemas/v1/admin.templates.json");
  templates >> a;
  a = a["templates"]["analyze"];
  a["path"] = path("result-2.json");
  a["expected_sha256"] = r["content_sha256"];
  a["output_path"] = path("training-analysis.json");
  a["pointer"] = "/sections/folds/data/0/train_rows";
  a["value_type"] = "integer";
  s::dispatch("analyze", a, e::unix_time_ms() + 30000);
  check(read(a["output_path"])["sections"]["series"]["data"].size() == 4);
  // Split is admitted to durable experiments; reuse verifies the exact
  // plan/result.
  auto directory = path("experiment");
  std::filesystem::create_directory(directory);
  auto child = p;
  child.erase("output_path");
  J exp{{"protocol", "symphony.sbv.experiment-input.v1"},
        {"output_path", path("experiment.json")},
        {"directory", directory},
        {"experiment_id", "temporal"},
        {"workers", "1"},
        {"on_failure", "continue"},
        {"extensions", J::object()},
        {"trials", J::array({{{"id", "folds"},
                              {"state", "ready"},
                              {"operation", "split"},
                              {"request", child},
                              {"reason", ""},
                              {"parameters", J::object()},
                              {"lineage", J::object()}}})}};
  s::dispatch("experiment", exp, e::unix_time_ms() + 30000);
  check(read(exp["output_path"])["sections"]["search"]["data"]
                                ["executed_this_invocation"] == "1");
  exp["output_path"] = path("reconciled.json");
  s::dispatch("experiment", exp, e::unix_time_ms() + 30000);
  check(read(exp["output_path"])["sections"]["search"]["data"]
                                ["reused_receipts"] == "1");
  if (env) {
    p["output_path"] = path("terminal-split.json");
    d::create_file(path("split-request.json"), p.dump(2),
                   e::unix_time_ms() + 30000);
  } else
    std::filesystem::remove_all(root);
  std::cout << checks << " temporal partition assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
