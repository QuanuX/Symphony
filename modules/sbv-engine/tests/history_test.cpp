#include "../src/detail.hpp"
#include <cstdlib>
#include <filesystem>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool value,
           std::source_location loc = std::source_location::current()) {
  ++checks;
  if (!value)
    throw std::runtime_error("history assertion " + d::dec(loc.line()));
}
J data(const J &r, const char *name) {
  return r.at("sections").at(name).at("data");
}
int main(int argc, char **argv) try {
  check(argc == 2);
  const auto templates =
      e::parse_bounded_json(d::read_file(argv[1], e::no_deadline),
                            d::artifact_bytes, d::artifact_values)
          .at("templates");
  char tmp[] = "/private/tmp/sbv-history-XXXXXX";
  auto env = std::getenv("SYMPHONY_SBV_HISTORY_FIXTURES");
  auto made = env ? env : mkdtemp(tmp);
  check(made);
  std::filesystem::path root(made);
  std::filesystem::create_directories(root);
  unsigned seq = 0;
  auto path = [&](std::string n) { return (root / n).string(); };
  auto read = [&](std::string p) {
    auto r = e::parse_bounded_json(d::read_file(p, e::no_deadline),
                                   d::artifact_bytes, d::artifact_values);
    s::validate_result(r);
    return r;
  };
  J last_request, last_receipt;
  auto call = [&](std::string op, J p) {
    p["output_path"] = path(op + "-" + d::dec(++seq) + ".json");
    last_request = p;
    last_receipt = s::dispatch(op, p, e::no_deadline);
    auto r = read(p["output_path"]);
    check(last_receipt["content_sha256"] == r["content_sha256"]);
    return r;
  };
  auto ref = [&](const J &receipt, std::string pointer = "") {
    return J{{"path", receipt.at("path")},
             {"expected_sha256", receipt.at("content_sha256")},
             {"pointer", pointer}};
  };
  auto save = [&](J artifact) {
    artifact = s::seal_result(artifact);
    auto p = path("source-" + d::dec(++seq) + ".json");
    d::create_file(p, artifact.dump(), e::no_deadline);
    return J{{"path", p},
             {"expected_sha256", artifact["content_sha256"]},
             {"pointer", ""}};
  };
  auto observations = d::base("independent exact research observations");
  J rows = J::array();
  for (int i = 0; i < 6; ++i)
    rows.push_back(
        {{"id", d::dec(i)}, {"x", d::dec(i)}, {"y", d::dec(2 * i + 1)}});
  observations["sections"]["rows"] = d::section(rows);
  observations["sections"]["train"] =
      d::section(J::array({rows[0], rows[1], rows[2], rows[3]}));
  observations["sections"]["test"] =
      d::section(J::array({rows[3], rows[4], rows[5]}));
  observations["sections"]["singular"] = d::section(J::array({rows[0]}));
  auto src = save(observations);
  auto fit = templates.at("fit");
  fit["source"] = src;
  fit["source"]["pointer"] = "/sections/train/data";
  fit["identity_namespace"] = "ns";
  fit["retain_training_predictions"] = false;
  auto fitted = call("fit", fit);
  auto fit_ref = ref(last_receipt);
  auto predict = templates.at("predict");
  predict["model"] = fit_ref;
  predict["model"]["pointer"] = "/sections/model/data";
  predict["source"] = src;
  predict["source"]["pointer"] = "/sections/test/data";
  predict["identity_namespace"] = "ns";
  auto predicted = call("predict", predict);
  auto pred_ref = ref(last_receipt);
  predict["target"] = nullptr;
  call("predict", predict);
  auto unlabeled = ref(last_receipt);
  predict["identity_namespace"] = "another_ns";
  predict["target"] = templates.at("predict").at("target");
  call("predict", predict);
  auto other = ref(last_receipt);
  auto entry = [&](std::string id, std::string kind, J source) {
    return J{{"id", id},
             {"kind", kind},
             {"source", source},
             {"role", "user holdout label; not a secrecy claim"},
             {"extensions", J::object()}};
  };
  auto p = templates.at("research_history");
  p["entries"] = J::array({entry("fit", "fit", fit_ref),
                           entry("test", "predict", pred_ref),
                           entry("repeat", "predict", pred_ref),
                           entry("inference", "predict", unlabeled),
                           entry("other", "predict", other)});
  auto history = call("research_history", p);
  auto usage = data(history, "research_usage"),
       summary = data(history, "summary");
  check(summary.at("entries") == "5");
  check(summary.at("counted_entries") == "4");
  check(summary.at("collapsed_entries") == "1");
  check(summary.at("counted_observation_occurrences") == "13");
  check(summary.at("unique_namespaced_observations") == "9");
  check(summary.at("reused_namespaced_observations") == "3");
  check(summary.at("counted_namespaces") == "2");
  check(usage[1]["previously_fitted_observation_count"] == "1");
  check(usage[1]["reused_observation_count"] == "1");
  check(usage[2]["duplicate_of"] == "test");
  check(usage[2]["reused_observation_count"].is_null());
  check(usage[3]["previously_predicted_observation_count"] == "3");
  check(usage[3]["previously_targeted_observation_count"] == "3");
  check(usage[3]["prior_entry_count"] == "2");
  check(usage[4]["reused_observation_count"] == "0");
  auto q = p;
  q["duplicate_artifacts"] = "count_entries";
  auto counted = call("research_history", q);
  check(data(counted, "summary")["counted_observation_occurrences"] == "16");
  check(data(counted, "research_usage")[3]["prior_entry_count"] == "3");
  check(data(counted, "research_usage")[2]["duplicate_of"] == "test");
  q["retain_observations"] = false;
  auto compact = call("research_history", q);
  check(data(compact, "summary") == data(counted, "summary"));
  check(compact["sections"]["observation_usage"]["status"] == "not_selected");
  for (const auto &u : data(compact, "research_usage"))
    check(u["reused_observations"].is_null());
  // Reordering is explicit: counts use supplied order, never inferred
  // timestamps.
  q = p;
  std::swap(q["entries"][0], q["entries"][1]);
  auto reordered = call("research_history", q);
  check(data(reordered,
             "research_usage")[0]["previously_fitted_observation_count"] ==
        "0");
  check(data(reordered,
             "research_usage")[1]["previously_predicted_observation_count"] ==
        "1");
  q = p;
  q["entries"] = J::array({entry("inference-first", "predict", unlabeled),
                           entry("later-targets", "predict", pred_ref)});
  auto targetless_first = data(call("research_history", q), "research_usage");
  check(targetless_first[1]["previously_predicted_observation_count"] == "3");
  check(targetless_first[1]["previously_targeted_observation_count"] == "0");
  check(targetless_first[1]["previously_fitted_observation_count"] == "0");
  // Source artifacts can be selected inside an immutable enclosing result.
  auto wrapper = d::base("explicit embedded result");
  wrapper["sections"]["nested"] = d::section(predicted);
  auto nested = save(wrapper);
  nested["pointer"] = "/sections/nested/data";
  q = p;
  q["entries"][2]["source"] = nested;
  auto embedded = call("research_history", q);
  check(data(embedded, "research_usage")[2]["duplicate_of"] == "test");
  check(data(embedded, "summary") == summary);
  auto reject = [&](J req, std::int64_t deadline = e::no_deadline) {
    req["output_path"] = path("reject-" + d::dec(++seq) + ".json");
    bool failed = false;
    try {
      s::dispatch("research_history", req, deadline);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(req.at("output_path").get<std::string>()));
  };
  q = p;
  q["entries"][1]["kind"] = "fit";
  reject(q);
  q = p;
  q["entries"][0]["source"]["expected_sha256"] = std::string(64, '0');
  reject(q);
  q = p;
  q["entries"][0]["id"] = "test";
  reject(q);
  q = p;
  q["duplicate_artifacts"] = "guess";
  reject(q);
  q = p;
  q["entries"][0]["role"] = "";
  reject(q);
  q = p;
  q["selections"] = nullptr;
  reject(q);
  q = p;
  q["on_unavailable"] = "guess";
  reject(q);
  q = p;
  q["entries"][0]["source"]["pointer"] = "/bad~2pointer";
  reject(q);
  reject(p, e::unix_time_ms() - 1);
  auto altered = fitted;
  altered["sections"]["model"]["data"]["training"]["observation_ids"].push_back(
      "0");
  q = p;
  q["entries"][0]["source"] = save(altered);
  reject(q);
  altered = fitted;
  altered["sections"]["summary"]["data"]["training_rows"] = "100";
  q = p;
  q["entries"][0]["source"] = save(altered);
  reject(q);
  altered = fitted;
  altered["sections"]["provenance"]["data"]["identity_namespace"] = "wrong";
  q = p;
  q["entries"][0]["source"] = save(altered);
  reject(q);
  altered = predicted;
  altered["sections"]["predictions"]["data"][0]["target"] = nullptr;
  q = p;
  q["entries"][0] = entry("corrupt", "predict", save(altered));
  reject(q);
  altered = fitted;
  altered["sections"]["choices"]["data"]["source"]["expected_sha256"] =
      std::string(64, '0');
  q = p;
  q["entries"][0]["source"] = save(altered);
  reject(q);
  check(usage[1]["source_purpose"] == predict["purpose"]);
  check(usage[0]["source_purpose"].is_null());
  // Unavailable fits cannot be represented as an empty/untouched data history.
  auto singular = fit;
  singular["source"]["pointer"] = "/sections/singular/data";
  call("fit", singular);
  auto unavailable = ref(last_receipt);
  q = p;
  q["entries"].push_back(entry("unknown", "fit", unavailable));
  auto partial = call("research_history", q);
  check(partial["status"] == "partial");
  check(data(partial, "summary")["unresolved_entries"] == "1");
  check(data(partial, "research_usage")[5]["observation_count"].is_null());
  q["on_unavailable"] = "reject";
  reject(q);
  // Explicit comparison-backed selection may choose lower-ranked or failed
  // candidates.
  auto score = d::base("external supplied metric fixture");
  score["sections"]["metric"] = d::section("1");
  auto low = save(score);
  score["sections"]["metric"] = d::section("3");
  auto high = save(score);
  auto candidate = [&](std::string id, J r) {
    return J{{"id", id},
             {"state", "completed"},
             {"reason", ""},
             {"path", r["path"]},
             {"expected_sha256", r["expected_sha256"]},
             {"parameters", J::object()},
             {"lineage", J::object()}};
  };
  auto failed = candidate("failed", low);
  failed["state"] = "failed";
  failed["reason"] = "caller recorded failed trial";
  failed["path"] = nullptr;
  failed["expected_sha256"] = nullptr;
  auto comparison = templates.at("compare");
  comparison["candidates"] =
      J::array({candidate("winner", low), candidate("chosen", high), failed});
  comparison["objectives"] =
      J::array({J{{"id", "loss"},
                  {"pointer", "/sections/metric/data"},
                  {"type", "integer"},
                  {"direction", "minimize"},
                  {"weight", {{"numerator", "1"}, {"denominator", "1"}}},
                  {"scale", {{"numerator", "1"}, {"denominator", "1"}}},
                  {"unit", "user unit"}}});
  call("compare", comparison);
  auto compared = ref(last_receipt);
  J selection{
      {"id", "decision"},
      {"source", compared},
      {"chosen_candidate_ids", J::array({"chosen", "failed"})},
      {"reason", "User elects further study despite recorded rank/status"},
      {"extensions", J::object()}};
  q = p;
  q["selections"] = J::array({selection});
  auto selected = call("research_history", q);
  auto record = data(selected, "selection_history")[0];
  check(record["weighted_order"][0] == "winner");
  check(record["chosen_candidates"][0]["weighted_rank"] == "2");
  check(record["chosen_candidates"][1]["state"] == "failed");
  check(selected["status"] == "completed");
  check(data(selected, "summary")["selection_records"] == "1");
  auto final_request = q;
  auto final_result = selected;
  q["selections"][0]["chosen_candidate_ids"] = J::array();
  check(data(call("research_history", q),
             "selection_history")[0]["chosen_candidates"]
            .empty());
  q["selections"][0]["chosen_candidate_ids"] = J::array({"unknown"});
  reject(q);
  q["selections"][0]["chosen_candidate_ids"] = J::array({"chosen", "chosen"});
  reject(q);
  q["selections"][0]["chosen_candidate_ids"] = J::array({"chosen"});
  q["selections"].push_back(q["selections"][0]);
  reject(q);
  // Empty selected coverage is explicit, not proof of untouched data.
  q = p;
  q["entries"] = J::array();
  auto empty = call("research_history", q);
  check(data(empty, "summary")["entries"] == "0");
  check(data(empty, "summary")["unique_namespaced_observations"] == "0");
  // Append selected history as an ordinary dependent native graph producer.
  auto exp = templates.at("experiment");
  auto dir = path("journal");
  std::filesystem::create_directory(dir);
  exp["directory"] = dir;
  exp["workers"] = "1";
  auto fit_req = fit;
  fit_req.erase("output_path");
  auto hist = final_request;
  hist.erase("output_path");
  hist["entries"] = J::array({entry("fit", "fit", nullptr)});
  auto trial = [&](std::string id, std::string op, J req) {
    return J{{"id", id},
             {"operation", op},
             {"request", req},
             {"state", "ready"},
             {"reason", ""},
             {"parameters", J::object()},
             {"lineage", J::object()}};
  };
  auto h = trial("history", "research_history", hist);
  h["depends_on"] = J::array({"fit"});
  h["bindings"] = J::array({J{{"target_pointer", "/entries/0/source"},
                              {"trial_id", "fit"},
                              {"result_pointer", ""}}});
  exp["trials"] = J::array({h, trial("fit", "fit", fit_req)});
  auto graph = call("experiment", exp);
  check(data(graph, "summary")["completed"] == "2");
  auto resumed = call("experiment", exp);
  check(data(resumed, "search")["executed_this_invocation"] == "0");
  if (env) {
    final_request["output_path"] = path("installed-history.json");
    d::create_file(path("history-request.json"), final_request.dump(),
                   e::no_deadline);
    d::create_file(path("history-expected.json"), final_result.dump(),
                   e::no_deadline);
  }
  std::cout << checks << " research history assertions passed\n";
  return 0;
} catch (const std::exception &x) {
  std::cerr << x.what() << '\n';
  return 1;
}
