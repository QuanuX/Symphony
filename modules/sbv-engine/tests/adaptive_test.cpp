#include "../src/consumer_source.hpp"
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
           std::source_location location = std::source_location::current()) {
  ++checks;
  if (!value)
    throw std::runtime_error("adaptive assertion " +
                             std::to_string(location.line()));
}
J data(const J &r, const std::string &name) {
  return r.at("sections").at(name).at("data");
}
J read(const std::string &path) {
  auto r = e::parse_bounded_json(d::read_file(path, e::no_deadline),
                                 d::artifact_bytes, d::artifact_values);
  s::validate_result(r);
  return r;
}
int main(int argc, char **argv) try {
  check(argc == 2);
  const auto catalogue =
      e::parse_bounded_json(d::read_file(argv[1], e::no_deadline),
                            d::artifact_bytes, d::artifact_values);
  const auto templates = catalogue.at("templates");
  char tmp[] = "/private/tmp/sbv-adaptive-XXXXXX";
  const auto *env = std::getenv("SYMPHONY_SBV_ADAPTIVE_FIXTURES");
  const auto *created = env ? env : mkdtemp(tmp);
  check(created);
  const std::filesystem::path root(created);
  std::filesystem::create_directories(root);
  auto path = [&](std::string name) { return (root / name).string(); };
  auto write = [&](std::string name, const J &value) {
    d::create_file(path(name), value.dump(), e::no_deadline);
  };
  auto source = d::base("exact synthetic line; no market outcome claim");
  J observations = J::array();
  for (unsigned i = 0; i < 12; ++i)
    observations.push_back({{"id", d::dec(i)},
                            {"x", d::dec(i)},
                            {"y", d::dec(1 + 2 * i)},
                            {"start_ns", d::dec(10 * i)},
                            {"end_ns", d::dec(10 * i)},
                            {"available_ns", d::dec(10 * i)}});
  source["sections"]["user_extensions"] =
      d::section({{"observations", observations}});
  source = s::seal_result(source);
  write("source.json", source);
  auto split = catalogue.at("variants").at("split").at("nested");
  split["path"] = path("source.json");
  split["expected_sha256"] = source.at("content_sha256");
  split["plan"]["outer"]["first_train_end_ns"] = "80";
  split["plan"]["outer"]["first_fit_cutoff_ns"] = "80";
  split["plan"]["inner"][0]["plan"]["fold_count"] = "1";
  auto trial = [&](std::string id, std::string op, J req, J deps = J::array(),
                   J bindings = J::array()) {
    req.erase("output_path");
    return J{{"id", id},
             {"state", "ready"},
             {"operation", op},
             {"request", req},
             {"reason", ""},
             {"parameters", J::object()},
             {"lineage", J::object()},
             {"depends_on", deps},
             {"bindings", bindings}};
  };
  auto bind = [&](std::string target, std::string id, std::string pointer,
                  std::string mode = "reference") {
    return J{{"target_pointer", target},
             {"trial_id", id},
             {"result_pointer", pointer},
             {"mode", mode}};
  };
  auto event = [&](std::string eid, std::string pid, std::string state,
                   J tid = nullptr) {
    return J{
        {"event_id", eid},
        {"proposal_id", pid},
        {"state", state},
        {"trial_id", tid},
        {"reason", "caller decision"},
        {"parameters", J::object()},
        {"lineage",
         {{"optimizer", "external caller"}, {"prior_history", "unknown"}}}};
  };
  auto exp = catalogue.at("variants").at("experiment").at("adaptive");
  exp["experiment_id"] = "adaptive";
  exp["workers"] = "2";
  exp["trials"] = J::array({trial("split", "split", split)});
  auto fit = templates.at("fit"), predict = templates.at("predict");
  fit["source"] = nullptr;
  predict["source"] = nullptr;
  predict["model"] = nullptr;
  auto comparison = catalogue.at("variants").at("compare").at("selection");
  comparison["candidates"] = J::array();
  comparison["methods"] = J::array({"weighted_sum", "pareto"});
  comparison["objectives"] = J::array({J{
      {"id", "error"},
      {"pointer", "/sections/studies/data/0/data/weighted_squared_error_sum"},
      {"type", "rational"},
      {"direction", "minimize"},
      {"weight", {{"numerator", "1"}, {"denominator", "1"}}},
      {"scale", {{"numerator", "1"}, {"denominator", "1"}}},
      {"unit", "squared_target"}}});
  J compare_bindings = J::array(), compare_deps = J::array();
  for (unsigned i = 0; i < 2; ++i) {
    auto f = fit;
    f["regularization"]["feature_penalties"][0]["numerator"] =
        i == 0 ? "0" : "100";
    auto id = d::dec(i);
    exp["trials"].push_back(trial(
        "fit" + id, "fit", f, J::array({"split"}),
        J::array(
            {bind("/source", "split", "/sections/folds/data/1/train_rows")})));
    exp["trials"].push_back(trial(
        "predict" + id, "predict", predict, J::array({"split", "fit" + id}),
        J::array({bind("/source", "split", "/sections/folds/data/1/test_rows"),
                  bind("/model", "fit" + id, "/sections/model/data")})));
    comparison["candidates"].push_back(
        {{"id", "candidate" + id},
         {"state", "completed"},
         {"reason", ""},
         {"source", nullptr},
         {"parameters", {{"regularization", f.at("regularization")}}},
         {"lineage",
          {{"inner_fold", "outer-0/inner-0"},
           {"fit_trial", "fit" + id},
           {"prediction_trial", "predict" + id}}}});
    compare_deps.push_back("predict" + id);
    compare_bindings.push_back(
        bind("/candidates/" + id + "/source", "predict" + id, ""));
  }
  exp["trials"].push_back(
      trial("select", "compare", comparison, compare_deps, compare_bindings));
  fit["regularization"] = nullptr;
  exp["trials"].push_back(trial(
      "refit", "fit", fit, J::array({"split", "select"}),
      J::array({bind("/source", "split", "/sections/folds/data/0/train_rows"),
                bind("/regularization", "select",
                     "/sections/selection/data/parameters/regularization",
                     "value")})));
  exp["trials"].push_back(trial(
      "outer", "predict", predict, J::array({"split", "refit"}),
      J::array({bind("/source", "split", "/sections/folds/data/0/test_rows"),
                bind("/model", "refit", "/sections/model/data")})));
  exp["proposal_events"] = J::array({event("p0", "model0", "proposed"),
                                     event("a0", "model0", "admitted", "fit0"),
                                     event("p1", "later", "proposed"),
                                     event("p2", "rejected", "proposed"),
                                     event("r2", "rejected", "rejected")});
  unsigned serial = 0;
  auto fresh = [&](J p) {
    p["directory"] = path("journal-" + d::dec(++serial));
    std::filesystem::create_directory(p.at("directory").get<std::string>());
    return p;
  };
  auto call = [&](J p) {
    p["output_path"] = path("summary-" + d::dec(++serial) + ".json");
    s::dispatch("experiment", p, e::no_deadline);
    return read(p.at("output_path"));
  };
  auto reject = [&](J p) {
    p["output_path"] = path("reject-" + d::dec(++serial) + ".json");
    bool rejected = false;
    try {
      s::dispatch("experiment", p, e::no_deadline);
    } catch (const std::exception &) {
      rejected = true;
    }
    check(rejected);
    check(!std::filesystem::exists(p.at("output_path").get<std::string>()));
  };
  auto plan = fresh(exp);
  const auto journal = plan.at("directory").get<std::string>();
  write("nested-selection-request.json", plan);
  auto result = call(plan);
  write("nested-selection-result.json", result);
  check(data(result, "summary").at("completed") == "8");
  const auto selected =
      data(read(journal + "/select.result.json"), "selection");
  check(selected.at("candidate_id") == "candidate0");
  check(selected.at("best_set_size") == "1");
  const auto model = data(read(journal + "/refit.result.json"), "model");
  check(model.at("coefficients") ==
        J::array({J{{"numerator", "1"}, {"denominator", "1"}},
                  J{{"numerator", "2"}, {"denominator", "1"}}}));
  auto outer = read(journal + "/outer.result.json");
  check(data(outer, "training_overlap").at("reused_count") == "0");
  check(!data(outer, "predictions").empty());
  for (const auto &row : data(outer, "predictions"))
    check(row.at("prediction") == row.at("target"));
  auto claim = data(read(journal + "/refit.claim.json"), "search");
  check(claim.at("bindings")[1].at("mode") == "value");
  check(claim.at("bindings")[1].at("value_sha256") ==
        e::sha256_hex(selected.at("parameters").at("regularization").dump()));
  auto resumed = call(plan);
  check(data(resumed, "search").at("executed_this_invocation") == "0");
  check(data(resumed, "search").at("reused_receipts") == "8");
  const auto first_plan = data(result, "search").at("plan_sha256");
  auto next = plan;
  next["revision"] = {{"index", "1"}, {"previous_plan_sha256", first_plan}};
  next["workers"] = "1";
  next["worker_budget"] = nullptr;
  auto bad = templates.at("fit");
  bad["source"] = J{{"path", path("source.json")},
                    {"expected_sha256", source.at("content_sha256")},
                    {"pointer", "/sections/user_extensions/data/observations"}};
  bad["features"] = J::array();
  next["trials"].push_back(trial("bad", "fit", bad));
  next["trials"].push_back({{"id", "pruned"},
                            {"state", "pruned"},
                            {"request", nullptr},
                            {"operation", nullptr},
                            {"reason", "caller pruned"},
                            {"parameters", J::object()},
                            {"lineage", J::object()}});
  next["proposal_events"].push_back(event("a1", "later", "admitted", "bad"));
  next["proposal_events"].push_back(event("p3", "pruned", "proposed"));
  next["proposal_events"].push_back(event("r3", "pruned", "pruned"));
  auto r1 = call(next);
  check(data(r1, "search").at("executed_this_invocation") == "1");
  check(data(r1, "search").at("reused_receipts") == "8");
  check(data(r1, "summary").at("failed") == "1");
  check(data(r1, "summary").at("pruned") == "1");
  check(data(r1, "proposal_states").at("later") == "admitted");
  check(read(journal + "/refit.claim.json")
            .at("sections")
            .at("search")
            .at("data")
            .at("plan_sha256") == first_plan);
  auto again = call(next);
  check(data(again, "search").at("executed_this_invocation") == "0");
  check(data(again, "search").at("reused_receipts") == "9");
  reject(plan);
  auto q = next;
  q["revision"]["previous_plan_sha256"] = std::string(64, '0');
  reject(q);
  q = next;
  q["trials"][0]["reason"] = "rewrite";
  reject(q);
  q = next;
  q["proposal_events"][0]["reason"] = "rewrite";
  reject(q);
  q = next;
  q["trials"].push_back(q["trials"][0]);
  reject(q);
  q = next;
  q["proposal_events"].push_back(event("again", "later", "admitted", "fit1"));
  reject(q);
  q = next;
  q["proposal_events"].push_back(event("new", "other", "proposed"));
  q["proposal_events"].push_back(
      event("duplicate", "other", "admitted", "bad"));
  reject(q);
  // A missing receipt is ambiguous even in a later revision; no retry/adoption.
  std::filesystem::remove(journal + "/bad.receipt.json");
  auto r2 = call(next);
  check(data(r2, "summary").at("ambiguous") == "1");
  check(data(r2, "search").at("executed_this_invocation") == "0");
  q = next;
  q["revision"] = {
      {"index", "2"},
      {"previous_plan_sha256", data(r1, "search").at("plan_sha256")}};
  q["trials"].push_back(trial(
      "followup", "predict", predict, J::array({"split", "refit"}),
      J::array({bind("/source", "split", "/sections/folds/data/0/test_rows"),
                bind("/model", "refit", "/sections/model/data")})));
  q["proposal_events"].push_back(
      event("followup-proposed", "followup", "proposed"));
  q["proposal_events"].push_back(
      event("followup-admitted", "followup", "admitted", "followup"));
  auto r3 = call(q);
  check(data(r3, "summary").at("ambiguous") == "1");
  check(data(r3, "summary").at("completed") == "9");
  check(data(r3, "search").at("executed_this_invocation") == "1");
  check(data(call(q), "search").at("executed_this_invocation") == "0");
  reject(next);
  // Explicit tie/no-winner semantics; value binding blocks before refit claim.
  q = exp;
  q["trials"][3]["request"]["regularization"] =
      q["trials"][1]["request"]["regularization"];
  auto tied = fresh(q);
  auto tied_result = call(tied);
  check(data(tied_result, "summary").at("blocked") == "2");
  check(!std::filesystem::exists(tied.at("directory").get<std::string>() +
                                 "/refit.claim.json"));
  auto cmp =
      data(read(tied.at("directory").get<std::string>() + "/select.claim.json"),
           "search")
          .at("resolved_request");
  cmp["candidates"][0]["id"] = "z";
  cmp["candidates"][1]["id"] = "a";
  for (auto method : {"weighted_sum", "pareto"})
    for (auto policy : {"input_order", "candidate_id", "require_unique"}) {
      cmp["selection"] = {{"method", method}, {"tie_break", policy}};
      cmp["output_path"] = path("compare-" + d::dec(++serial) + ".json");
      s::dispatch("compare", cmp, e::no_deadline);
      auto decision = data(read(cmp.at("output_path")), "selection");
      check(decision.at("best_set_size") == "2");
      if (std::string(policy) != "require_unique")
        check(decision.at("candidate_id") ==
              (std::string(policy) == "input_order" ? "z" : "a"));

      check(decision.at("status") == (std::string(policy) == "require_unique"
                                          ? "unavailable"
                                          : "selected"));
    }
  cmp["candidates"] = J::array();
  cmp["output_path"] = path("empty-compare.json");
  s::dispatch("compare", cmp, e::no_deadline);
  check(data(read(cmp.at("output_path")), "selection").at("reason") ==
        "no eligible candidate");
  q = exp;
  q["trials"][6]["bindings"][1]["mode"] = "automatic";
  reject(fresh(q));
  q = exp;
  q["trials"][6]["request"]["protocol"] = nullptr;
  q["trials"][6]["bindings"][1]["target_pointer"] = "/protocol";
  reject(fresh(q));
  // The user may deliberately favor the worse error; no hidden promotion gate.
  q = exp;
  q["trials"][5]["request"]["objectives"][0]["direction"] = "maximize";
  auto override = fresh(q);
  check(data(call(override), "summary").at("completed") == "8");
  const auto override_dir = override.at("directory").get<std::string>();
  check(data(read(override_dir + "/select.result.json"), "selection")
            .at("candidate_id") == "candidate1");
  check(data(read(override_dir + "/refit.claim.json"), "search")
            .at("resolved_request")
            .at("regularization")
            .at("feature_penalties")[0]
            .at("numerator") == "100");
  // Same automatic graph with bundle children and an owned referenced plan.
  J options{{"max_page_bytes", nullptr}, {"cache_bytes", "1048576"}};
  auto output = [&](std::string name) {
    const auto base = path(name + "-" + d::dec(++serial));
    return J{
        {"kind", "partitioned"},
        {"bundle_path", base},
        {"workspace_path", base + "-work"},
        {"write_options", {{"page_bytes", "16384"}, {"index_fanout", "16"}}},
        {"scratch_read_options", options}};
  };
  auto publish_plan = [&](J &request, const J &trials) {
    auto body = d::base("caller proposal table");
    body["sections"]["summary"] =
        d::section({{"trials", d::dec(trials.size())}});
    body["sections"]["trials"] = d::section(trials);
    auto source = d::partitioned_result(
        {{"output", output("proposal-table")}}, "experiment", e::no_deadline,
        [&](d::PartitionedOutput &) { return s::logical::Value(body); });
    check(source.at("status") == "completed");
    request["trials"] = {
        {"kind", "referenced_rows"},
        {"source",
         {{"kind", "bundle"},
          {"reference", source.at("storage").at("reference")},
          {"selector",
           {{"kind", "pointer"}, {"pointer", "/sections/trials/data"}}},
          {"read_options", options}}}};
    request["plan_output"] = output("owned-plan");
    request["plan_read_options"] = options;
    return source.at("storage")
        .at("reference")
        .at("manifest_path")
        .get<std::string>();
  };
  auto partitioned = fresh(exp);
  J table = partitioned.at("trials");
  for (auto &trial : table) {
    trial["request"]["output"] = output(trial.at("id"));
    trial["result_read_options"] = options;
  }
  auto input_manifest = publish_plan(partitioned, table);
  auto bundle_result = call(partitioned);
  check(data(bundle_result, "summary").at("completed") == "8");
  const auto bundle_claim =
      data(read(partitioned.at("directory").get<std::string>() +
                "/refit.claim.json"),
           "search");
  check(bundle_claim.at("resolved_request").at("regularization") ==
        selected.at("parameters").at("regularization"));
  check(bundle_claim.at("bindings")[1].at("reference").at("kind") == "bundle");
  std::filesystem::remove_all(
      std::filesystem::path(input_manifest).parent_path());
  auto bundle_resume = call(partitioned);
  check(data(bundle_resume, "search").at("executed_this_invocation") == "0");
  check(data(bundle_resume, "search").at("reused_receipts") == "8");
  table.push_back({{"id", "new-pruned"},
                   {"state", "pruned"},
                   {"operation", nullptr},
                   {"request", nullptr},
                   {"reason", "external caller"},
                   {"parameters", J::object()},
                   {"lineage", J::object()}});
  partitioned["revision"] = {{"index", "1"},
                             {"previous_plan_sha256",
                              data(bundle_result, "search").at("plan_sha256")}};
  publish_plan(partitioned, table);
  auto bundle_append = call(partitioned);
  check(data(bundle_append, "search").at("executed_this_invocation") == "0");
  check(data(bundle_append, "search").at("reused_receipts") == "8");
  check(data(bundle_append, "summary").at("pruned") == "1");
  if (!env)
    std::filesystem::remove_all(root);
  std::cout << "adaptive checks: " << checks << "\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << "\n";
  return 1;
}
