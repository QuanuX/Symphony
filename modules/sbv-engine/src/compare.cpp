#include "wide_rational.hpp"
#include <algorithm>
#include <numeric>
namespace symphony::sbv::detail {
namespace {
namespace w = wide_rational;
struct Objective {
  std::string id, pointer, type, direction;
  w::R weight, scale;
};
struct Candidate {
  bool eligible = false;
  std::vector<w::R> values;
  w::R utility;
};
} // namespace
Json compare(const Json &p, std::int64_t end) {
  keys(p, {"protocol", "output_path", "candidates", "objectives", "methods",
           "missing", "comparison_description", "extensions"});
  need(p.at("extensions").is_object(), "extensions object required");
  need(p.at("missing") == "exclude_candidate" || p.at("missing") == "reject",
       "objective missing policy required");
  need(!str(p.at("comparison_description")).empty() &&
           str(p.at("comparison_description")).size() <= 4096,
       "bounded comparison description required");
  need(p.at("candidates").is_array() && p.at("candidates").size() <= 128,
       "candidate bound");
  need(p.at("objectives").is_array() && !p.at("objectives").empty() &&
           p.at("objectives").size() <= 16,
       "objective bound 1..16");
  need(p.at("methods").is_array() && p.at("methods").size() <= 2,
       "comparison method bound");
  std::set<std::string> methods;
  for (const auto &m : p.at("methods")) {
    auto id = str(m);
    need((id == "weighted_sum" || id == "pareto") && methods.insert(id).second,
         "unknown/duplicate comparison method");
  }
  std::set<std::string> ids;
  std::vector<Objective> objectives;
  for (const auto &o : p.at("objectives")) {
    keys(o, {"id", "pointer", "type", "direction", "weight", "scale", "unit"});
    auto id = str(o.at("id")), pointer = str(o.at("pointer"));
    need(!id.empty() && id.size() <= 128 && ids.insert(id).second,
         "unique bounded objective id required");
    need(pointer.size() <= 4096 && (pointer.empty() || pointer[0] == '/'),
         "bounded objective pointer required");
    (void)Json::json_pointer(pointer);
    need(o.at("type") == "integer" || o.at("type") == "rational",
         "exact numeric objective type required");
    need(o.at("direction") == "maximize" || o.at("direction") == "minimize",
         "objective direction required");
    need(!str(o.at("unit")).empty() && str(o.at("unit")).size() <= 128,
         "objective unit required");
    auto weight = w::parameter(o.at("weight")),
         scale = w::parameter(o.at("scale"));
    need(scale.n > 0, "positive objective scale required");
    objectives.push_back({id, pointer, str(o.at("type")),
                          str(o.at("direction")), weight, scale});
  }
  ids.clear();
  std::vector<Candidate> candidates;
  Json rows = Json::array(), sources = Json::array();
  std::size_t total_bytes = 0, eligible = 0;
  for (const auto &c : p.at("candidates")) {
    deadline(end);
    keys(c, {"id", "state", "reason", "path", "expected_sha256", "parameters",
             "lineage"});
    auto id = str(c.at("id")), state = str(c.at("state"));
    need(!id.empty() && id.size() <= 128 && ids.insert(id).second,
         "unique bounded candidate id required");
    need(state == "completed" || state == "failed" || state == "pruned",
         "candidate state required");
    need(str(c.at("reason")).size() <= 4096 && c.at("parameters").is_object() &&
             c.at("lineage").is_object(),
         "candidate reason/parameters/lineage required");
    Candidate candidate;
    Json values = Json::array();
    Json row{
        {"id", id},
        {"state", state},
        {"reason", c.at("reason")},
        {"source", nullptr},
        {"objectives", Json::array()},
        {"eligibility", "excluded"},
        {"exclusion_reason", state == "completed"
                                 ? "objective field absent"
                                 : "caller-declared failed/pruned trial"},
        {"utility", missing("weighted sum not selected or candidate excluded")},
        {"weighted_rank", nullptr},
        {"pareto_front", nullptr}};
    if (state == "completed") {
      need(c.at("path").is_string() && c.at("expected_sha256").is_string(),
           "completed trial requires result identity");
      auto bytes = read_file(str(c.at("path")), end);
      need(bytes.size() <= 256U * 1024 * 1024 - total_bytes,
           "aggregate comparison source-byte bound");
      total_bytes += bytes.size();
      auto source =
          e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
      validate_result(source);
      need(!str(c.at("expected_sha256")).empty() &&
               source.at("content_sha256") == c.at("expected_sha256"),
           "candidate result identity mismatch");
      row["source"] = {{"path", c.at("path")},
                       {"content_sha256", source.at("content_sha256")},
                       {"result_status", source.at("status")},
                       {"origin", source.at("origin")}};
      sources.push_back(row.at("source"));
      bool missing_field = false;
      for (const auto &o : objectives) {
        deadline(end);
        Json::json_pointer ptr(o.pointer);
        if (!source.contains(ptr)) {
          need(p.at("missing") != "reject", "objective field absent");
          missing_field = true;
          values.push_back({{"id", o.id},
                            {"status", "unavailable"},
                            {"reason", "source field absent"},
                            {"value", nullptr}});
          candidate.values.push_back({});
          continue;
        }
        auto v = o.type == "integer" ? w::R{w::integer(source.at(ptr)), 1}
                                     : w::artifact(source.at(ptr));
        candidate.values.push_back(v);
        values.push_back({{"id", o.id},
                          {"status", "available"},
                          {"reason", ""},
                          {"value", w::wire(v)}});
      }
      candidate.eligible = !missing_field;
      row["objectives"] = values;
      if (candidate.eligible) {
        ++eligible;
        row["eligibility"] = "eligible";
        row["exclusion_reason"] = "";
        if (methods.contains("weighted_sum")) {
          for (std::size_t j = 0; j < objectives.size(); ++j) {
            const auto &o = objectives[j];
            auto v = w::divide(candidate.values[j], o.scale);
            if (o.direction == "minimize")
              v.n = -v.n;
            candidate.utility =
                w::plus(candidate.utility, w::times(v, o.weight));
          }
          row["utility"] = {{"status", "available"},
                            {"value", w::wire(candidate.utility)}};
        }
      }
    } else
      need(c.at("path").is_null() && c.at("expected_sha256").is_null() &&
               !str(c.at("reason")).empty(),
           "failed/pruned trial requires reason and null result identity");
    rows.push_back(std::move(row));
    candidates.push_back(std::move(candidate));
  }
  std::vector<std::size_t> order;
  for (std::size_t i = 0; i < candidates.size(); ++i)
    if (candidates[i].eligible)
      order.push_back(i);
  Json ranked = Json::array(), fronts = Json::array();
  if (methods.contains("weighted_sum")) {
    std::stable_sort(order.begin(), order.end(), [&](auto a, auto b) {
      return w::compare(candidates[a].utility, candidates[b].utility) > 0;
    });
    std::size_t rank = 1;
    for (std::size_t k = 0; k < order.size(); ++k) {
      auto idx = order[k];
      if (k && w::compare(candidates[idx].utility,
                          candidates[order[k - 1]].utility) != 0)
        rank = k + 1;
      rows[idx]["weighted_rank"] = dec(rank);
      ranked.push_back(rows[idx].at("id"));
    }
  }
  if (methods.contains("pareto")) {
    auto dominates = [&](std::size_t a, std::size_t b) {
      bool strict = false;
      for (std::size_t j = 0; j < objectives.size(); ++j) {
        auto cmp = w::compare(candidates[a].values[j], candidates[b].values[j]);
        if (objectives[j].direction == "minimize")
          cmp = -cmp;
        if (cmp < 0)
          return false;
        strict = strict || cmp > 0;
      }
      return strict;
    };
    std::vector<std::vector<std::size_t>> dominated(candidates.size());
    std::vector<std::size_t> counts(candidates.size());
    for (std::size_t a = 0; a < candidates.size(); ++a)
      if (candidates[a].eligible)
        for (std::size_t b = 0; b < candidates.size(); ++b)
          if (a != b && candidates[b].eligible) {
            deadline(end);
            if (dominates(a, b)) {
              dominated[a].push_back(b);
              ++counts[b];
            }
          }
    std::vector<std::size_t> front;
    for (std::size_t a = 0; a < candidates.size(); ++a)
      if (candidates[a].eligible && counts[a] == 0)
        front.push_back(a);
    std::size_t level = 0, processed = 0;
    while (!front.empty()) {
      Json members = Json::array();
      std::vector<std::size_t> next;
      for (auto a : front) {
        rows[a]["pareto_front"] = dec(level);
        members.push_back(rows[a].at("id"));
        ++processed;
        for (auto b : dominated[a])
          if (--counts[b] == 0)
            next.push_back(b);
      }
      fronts.push_back({{"front", dec(level++)}, {"candidate_ids", members}});
      std::sort(next.begin(), next.end());
      front = std::move(next);
    }
    need(processed == eligible, "Pareto partial order invariant");
  }
  auto result = base("comparison of supplied candidate result artifacts");
  auto &s = result["sections"];
  s["summary"] = section({{"candidates", dec(candidates.size())},
                          {"eligible", dec(eligible)},
                          {"excluded", dec(candidates.size() - eligible)},
                          {"methods", p.at("methods")}});
  s["comparisons"] = section(
      {{"candidates", rows},
       {"weighted_order", ranked},
       {"pareto_fronts", fronts},
       {"weighted_definition",
        "sum(direction_sign * exact_value / positive_scale * "
        "signed_user_weight); descending utility; competition ranks; stable "
        "input order within ties"},
       {"pareto_definition",
        "maximize/minimize each selected raw objective; strict improvement on "
        "at least one and no worse on all; weights ignored; zero-based fronts; "
        "stable input order within each front"}});
  s["search"] = section(
      {{"executed_trials", "0"},
       {"lineage", p.at("candidates")},
       {"scope", "caller-supplied trial ledger; completed artifacts verified, "
                 "failed/pruned states are caller declarations"}});
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  s["provenance"] = section({{"engine_version", version},
                             {"sources", sources},
                             {"provider_requests", "0"},
                             {"additional_spend_usd", "0"}});
  s["resources"] = section({{"backend", "cpu"},
                            {"actual_workers", candidates.empty() ? "0" : "1"},
                            {"source_bytes_read", dec(total_bytes)}});
  s["diagnostics"] = section(Json::array(
      {"Ranking does not establish statistical significance, holdout "
       "independence or strategy execution feasibility.",
       "Source metric pointers, units, scales, directions, signed weights, "
       "failed/pruned states and lineage are selected by the caller.",
       "Missing objective fields exclude the entire candidate or reject as "
       "selected; malformed present numbers reject; no score imputation or "
       "normalization.",
       "Pareto ranking is independent of scalarization weights, including zero "
       "or negative weights.",
       "This operation evaluates supplied trials; it does not run an optimizer "
       "or execute candidate strategies."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "compare", end);
}
} // namespace symphony::sbv::detail
