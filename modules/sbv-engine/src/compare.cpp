#include "consumer_source.hpp"
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
  std::string id;
  std::optional<std::size_t> rank, front;
  bool eligible = false;
  std::vector<w::R> values;
  w::R utility;
};
std::optional<logical::Value> point(logical::Value value,
                                    const std::string &path) {
  (void)Json::json_pointer(path);
  std::size_t at = 0;
  while (at < path.size()) {
    const auto next = path.find('/', at + 1);
    const auto edge =
        path.substr(at + 1, next == std::string::npos ? next : next - at - 1);
    std::string decoded;
    for (std::size_t i = 0; i < edge.size(); ++i) {
      if (edge[i] == '~')
        decoded += edge[++i] == '0' ? '~' : '/';
      else
        decoded += edge[i];
    }
    if (value.kind() == "array") {
      std::uint64_t index = 0;
      const auto parsed = std::from_chars(
          decoded.data(), decoded.data() + decoded.size(), index);
      if (decoded.empty() || (decoded.size() > 1 && decoded.front() == '0') ||
          parsed.ec != std::errc{} ||
          parsed.ptr != decoded.data() + decoded.size() ||
          index >= value.size())
        return {};
      value = value.at(index);
    } else {
      if (value.kind() != "object" || !value.contains(decoded))
        return {};
      value = value.at(decoded);
    }
    if (next == std::string::npos)
      break;
    at = next;
  }
  return value;
}
} // namespace
static logical::Value compare_body(const Json &p, PartitionedOutput *context,
                                   std::int64_t end) {
  keys_optional(p,
                {"protocol", "candidates", "objectives", "methods", "missing",
                 "comparison_description", "extensions"},
                {"output_path", "output"});
  need(p.at("extensions").is_object(), "extensions object required");
  need(p.at("missing") == "exclude_candidate" || p.at("missing") == "reject",
       "objective missing policy required");
  need(!str(p.at("comparison_description")).empty() &&
           str(p.at("comparison_description")).size() <= 4096,
       "bounded comparison description required");
  auto candidate_input =
      consumer::input_rows(p.at("candidates"), "comparison_candidates", end);
  auto objective_input =
      consumer::input_rows(p.at("objectives"), "comparison_objectives", end);
  need(objective_input.value.size() > 0, "at least one objective required");
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
  auto objective_cursor = objective_input.value.children();
  while (auto child = objective_cursor.next()) {
    deadline(end);
    const auto o = child->value.materialize();
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
  consumer::Rows rows(context, "candidate-rows"), sources(context, "sources");
  std::size_t total_bytes = 0, eligible = 0;
  bool has_bundle_source = false;
  auto candidate_cursor = candidate_input.value.children();
  while (auto child = candidate_cursor.next()) {
    deadline(end);
    const auto c = child->value.materialize();
    keys_optional(c, {"id", "state", "reason", "parameters", "lineage"},
                  {"path", "expected_sha256", "source"});
    need(c.contains("source")
             ? !c.contains("path") && !c.contains("expected_sha256")
             : c.contains("path") && c.contains("expected_sha256"),
         "exactly one candidate identity form required");
    auto id = str(c.at("id")), state = str(c.at("state"));
    need(!id.empty() && id.size() <= 128 && ids.insert(id).second,
         "unique bounded candidate id required");
    need(state == "completed" || state == "failed" || state == "pruned",
         "candidate state required");
    need(str(c.at("reason")).size() <= 4096 && c.at("parameters").is_object() &&
             c.at("lineage").is_object(),
         "candidate reason/parameters/lineage required");
    Candidate candidate;
    candidate.id = id;
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
      auto source = [&]() -> logical::Value {
        if (c.contains("source")) {
          has_bundle_source = true;
          auto selected = consumer::result_reference(c.at("source"), end);
          row["source"] = {
              {"reference", selected.reference},
              {"content_sha256",
               selected.value.at("content_sha256").materialize()},
              {"result_status", selected.value.at("status").materialize()},
              {"origin", selected.value.at("origin").materialize()}};
          return selected.value;
        }
        need(c.at("path").is_string() && c.at("expected_sha256").is_string(),
             "completed trial requires result identity");
        auto bytes = read_file(str(c.at("path")), end);
        need(bytes.size() <=
                 std::numeric_limits<std::size_t>::max() - total_bytes,
             "comparison source byte accounting overflow");
        total_bytes += bytes.size();
        auto result =
            e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
        validate_result(result);
        need(result.at("content_sha256") == c.at("expected_sha256"),
             "candidate result identity mismatch");
        row["source"] = {{"path", c.at("path")},
                         {"content_sha256", result.at("content_sha256")},
                         {"result_status", result.at("status")},
                         {"origin", result.at("origin")}};
        return logical::Value(std::move(result));
      }();
      sources.append(row.at("source"));
      bool missing_field = false;
      for (const auto &o : objectives) {
        deadline(end);
        const auto selected = point(source, o.pointer);
        if (!selected) {
          need(p.at("missing") != "reject", "objective field absent");
          missing_field = true;
          values.push_back({{"id", o.id},
                            {"status", "unavailable"},
                            {"reason", "source field absent"},
                            {"value", nullptr}});
          candidate.values.push_back({});
          continue;
        }
        auto v = o.type == "integer"
                     ? w::R{w::integer(selected->materialize()), 1}
                     : w::artifact(selected->materialize());
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
      need((c.contains("source") ? c.at("source").is_null()
                                 : c.at("path").is_null() &&
                                       c.at("expected_sha256").is_null()) &&
               !str(c.at("reason")).empty(),
           "failed/pruned trial requires reason and null result identity");
    rows.append(row);
    candidates.push_back(std::move(candidate));
  }
  std::vector<std::size_t> order;
  for (std::size_t i = 0; i < candidates.size(); ++i)
    if (candidates[i].eligible)
      order.push_back(i);
  auto prepared_rows = rows.close();
  consumer::Rows ranked(context, "weighted-order"),
      fronts(context, "pareto-fronts");
  if (methods.contains("weighted_sum")) {
    std::stable_sort(order.begin(), order.end(), [&](auto a, auto b) {
      deadline(end);
      return w::compare(candidates[a].utility, candidates[b].utility) > 0;
    });
    std::size_t rank = 1;
    for (std::size_t k = 0; k < order.size(); ++k) {
      auto idx = order[k];
      if (k && w::compare(candidates[idx].utility,
                          candidates[order[k - 1]].utility) != 0)
        rank = k + 1;
      candidates[idx].rank = rank;
      ranked.append(Json(candidates[idx].id));
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
      consumer::Rows members(context, "pareto-members-" + dec(level));
      std::vector<std::size_t> next;
      for (auto a : front) {
        candidates[a].front = level;
        members.append(Json(candidates[a].id));
        ++processed;
        for (auto b : dominated[a])
          if (--counts[b] == 0)
            next.push_back(b);
      }
      fronts.append(
          logical::Value::object({{"front", logical::Value(Json(dec(level++)))},
                                  {"candidate_ids", members.close()}}));
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
      {{"candidates", nullptr},
       {"weighted_order", nullptr},
       {"pareto_fronts", nullptr},
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
       {"lineage", nullptr},
       {"scope", "caller-supplied trial ledger; completed artifacts verified, "
                 "failed/pruned states are caller declarations"}});
  auto choices = p;
  choices.erase("output_path");
  choices.erase("output");
  s["choices"] = section(choices);
  s["provenance"] = section({{"engine_version", version},
                             {"sources", nullptr},
                             {"provider_requests", "0"},
                             {"additional_spend_usd", "0"}});
  s["resources"] =
      section({{"backend", "cpu"},
               {"actual_workers", candidates.empty() ? "0" : "1"},
               {"source_bytes_read",
                has_bundle_source ? Json(nullptr) : Json(dec(total_bytes))}});
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
  if (has_bundle_source) {
    s["resources"]["data"]["legacy_source_bytes_read"] = dec(total_bytes);
    s["resources"]["data"]["source_bytes_read_scope"] =
        "aggregate referenced-source physical I/O not collected here; per-reader counters "
        "available via bundle operations";
  }
  consumer::Rows final_rows(context, "ranked-candidates");
  auto cursor = prepared_rows.children();
  std::size_t index = 0;
  while (auto child = cursor.next()) {
    deadline(end);
    const auto &c = candidates.at(index++);
    final_rows.append(
        child->value
            .with("weighted_rank",
                  logical::Value(c.rank ? Json(dec(*c.rank)) : Json(nullptr)))
            .with("pareto_front", logical::Value(c.front ? Json(dec(*c.front))
                                                         : Json(nullptr))));
  }
  auto out =
      logical::Value(s)
          .with("comparisons",
                logical::Value(s.at("comparisons"))
                    .with("data", logical::Value(s.at("comparisons").at("data"))
                                      .with("candidates", final_rows.close())
                                      .with("weighted_order", ranked.close())
                                      .with("pareto_fronts", fronts.close())))
          .with("search",
                logical::Value(s.at("search"))
                    .with("data", logical::Value(s.at("search").at("data"))
                                      .with("lineage", candidate_input.value)))
          .with("provenance",
                logical::Value(s.at("provenance"))
                    .with("data", logical::Value(s.at("provenance").at("data"))
                                      .with("sources", sources.close())))
          .with("objective_choices", consumer::section(objective_input.value));
  if (candidate_input.evidence)
    out = out.with("candidate_input",
                   consumer::section(*candidate_input.evidence));
  if (objective_input.evidence)
    out = out.with("objective_input",
                   consumer::section(*objective_input.evidence));
  return logical::Value(result).with("sections", out);
}
Json compare(const Json &p, std::int64_t end) {
  need(p.contains("output") != p.contains("output_path"),
       "select exactly one comparison output");
  if (p.contains("output"))
    return partitioned_result(p, "compare", end,
                              [&](PartitionedOutput &context) {
                                return compare_body(p, &context, end);
                              });
  return persist(compare_body(p, nullptr, end)
                     .materialize({artifact_bytes, artifact_values}),
                 p, "compare", end);
}
} // namespace symphony::sbv::detail
