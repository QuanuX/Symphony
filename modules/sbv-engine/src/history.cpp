#include "consumer_source.hpp"
#include <algorithm>
#include <limits>
namespace symphony::sbv::detail {
namespace {
void text(const Json &v) {
  need(v.is_string() && !str(v).empty(), "nonempty history label required");
}
void sha(const Json &v) {
  auto s = str(v);
  need(s.size() == 64 && std::all_of(s.begin(), s.end(),
                                     [](char c) {
                                       return (c >= '0' && c <= '9') ||
                                              (c >= 'a' && c <= 'f');
                                     }),
       "history SHA256 required");
}
logical::Value data(const logical::Value &r, const char *name) {
  const auto s = r.at("sections").at(name);
  need(s.at("status").materialize() == "available",
       "history source section unavailable");
  return s.at("data");
}
struct Source {
  logical::Value result;
  Json reference;
};
Source load(const Json &ref, std::size_t &total, bool &bundled,
            std::int64_t end) {
  if (ref.contains("kind")) {
    bundled = true;
    auto admitted = select_logical_result(ref, end);
    return {admitted.value, admitted.reference};
  }
  keys(ref, {"path", "expected_sha256", "pointer"});
  sha(ref.at("expected_sha256"));
  auto bytes = read_file(str(ref.at("path")), end);
  need(bytes.size() <= std::numeric_limits<std::size_t>::max() - total,
       "history byte accounting overflow");
  total += bytes.size();
  auto outer = e::parse_bounded_json(bytes, artifact_bytes, artifact_values);
  validate_result(outer);
  need(outer.at("content_sha256") == ref.at("expected_sha256"),
       "history source digest mismatch");
  const Json::json_pointer pointer(str(ref.at("pointer")));
  auto selected = outer.at(pointer);
  validate_result(selected);
  return {logical::Value(selected),
          {{"path", ref.at("path")},
           {"content_sha256", outer.at("content_sha256")},
           {"pointer", ref.at("pointer")},
           {"selected_sha256", selected.at("content_sha256")},
           {"file_sha256", e::sha256_hex(bytes)},
           {"bytes", dec(bytes.size())}}};
}
std::vector<std::string> ids(const logical::Value &values, std::int64_t end) {
  need(values.kind() == "array",
       "history observation identities must be an array");
  std::set<std::string> unique;
  std::vector<std::string> out;
  auto cursor = values.children();
  while (auto child = cursor.next()) {
    deadline(end);
    const auto v = child->value.materialize();
    text(v);
    auto id = str(v);
    need(unique.insert(id).second, "duplicate history observation identity");
    out.push_back(id);
  }
  return out;
}
struct Use {
  std::string entry_id, kind;
  bool targets;
};
struct Observation {
  std::vector<Use> uses;
};
} // namespace
static logical::Value history_body(const Json &p, PartitionedOutput *context,
                                   std::int64_t end) {
  keys_optional(p,
                {"protocol", "entries", "selections", "duplicate_artifacts",
                 "on_unavailable", "retain_observations",
                 "ordering_description", "coverage_description", "extensions"},
                {"output_path", "output"});
  need(p.at("extensions").is_object(), "history extensions required");
  auto entry_input =
      consumer::input_rows(p.at("entries"), "history_entries", end);
  auto selection_input =
      consumer::input_rows(p.at("selections"), "history_selections", end);
  need(p.at("duplicate_artifacts") == "collapse" ||
           p.at("duplicate_artifacts") == "count_entries",
       "history duplicate policy required");
  need(p.at("on_unavailable") == "retain" || p.at("on_unavailable") == "reject",
       "history unavailable policy required");
  need(p.at("retain_observations").is_boolean(),
       "history observation retention required");
  text(p.at("ordering_description"));
  text(p.at("coverage_description"));
  const bool retain = p.at("retain_observations").get<bool>();
  std::map<std::pair<std::string, std::string>, Observation> observations;
  std::map<std::string, std::string> artifacts;
  std::set<std::string> names;
  consumer::Rows entries(context, "usage-entries"), sources(context, "sources"),
      selections(context, "selection-history");
  bool bundled = false;
  std::size_t total_bytes = 0, unresolved = 0, counted = 0, collapsed = 0,
              uses = 0;
  auto entry_cursor = entry_input.value.children();
  while (auto child = entry_cursor.next()) {
    deadline(end);
    const auto entry = child->value.materialize();
    keys(entry, {"id", "kind", "source", "role", "extensions"});
    text(entry.at("id"));
    text(entry.at("role"));
    auto id = str(entry.at("id")), kind = str(entry.at("kind"));
    need(names.insert(id).second && (kind == "fit" || kind == "predict") &&
             entry.at("extensions").is_object(),
         "unique history entry, fit/predict kind and extensions required");
    auto source = load(entry.at("source"), total_bytes, bundled, end);
    sources.append(source.reference);
    const auto r = source.result;
    const auto choices = data(r, "choices").materialize(),
               summary = data(r, "summary").materialize(),
               provenance = data(r, "provenance").materialize();
    need(str(choices.at("protocol")) == "symphony.sbv." + kind + "-input.v1",
         "history kind does not match selected producer contract");
    text(choices.at("identity_namespace"));
    auto ns = str(choices.at("identity_namespace"));
    sha(provenance.at("source_result_sha256"));
    auto source_pointer = provenance.at("source_pointer");
    if (!source_pointer.is_null())
      (void)Json::json_pointer(str(source_pointer));
    need(str(provenance.at("identity_namespace")) == ns,
         "history identity namespace mismatch");
    if (choices.at("source").contains("kind")) {
      const auto &declaration = choices.at("source"),
                 &selector = declaration.at("selector");
      const auto selected_pointer = selector.at("kind") == "pointer"
                                        ? selector.at("pointer")
                                        : Json(nullptr);
      validate_logical_reference_lineage(
          declaration, data(r, "source_selection").materialize());
      need(declaration.at("reference").at("content_sha256") ==
                   provenance.at("source_result_sha256") &&
               selected_pointer == source_pointer,
           "history bundle input/provenance source mismatch");
    } else
      need(choices.at("source").at("expected_sha256") ==
                   provenance.at("source_result_sha256") &&
               choices.at("source").at("pointer") == source_pointer,
           "history input/provenance source mismatch");
    Json purpose = nullptr;
    if (kind == "predict") {
      text(choices.at("purpose"));
      purpose = choices.at("purpose");
    } else
      need(choices.at("target").is_object(), "fit target declaration required");
    const auto digest = str(r.at("content_sha256").materialize());
    auto prior = artifacts.find(digest);
    Json duplicate =
        prior == artifacts.end() ? Json(nullptr) : Json(prior->second);
    if (prior == artifacts.end())
      artifacts.emplace(digest, id);
    bool known = true, targets = kind == "fit";
    auto model_training = logical::Value(Json(nullptr));
    std::vector<std::string> selected_ids;
    if (kind == "fit") {
      const auto model = r.at("sections").at("model");
      if (model.at("status").materialize() == "unavailable") {
        need(model.at("data").materialize().is_null() &&
                 summary.at("model_status") == "unavailable",
             "inconsistent unavailable fit");
        known = false;
      } else {
        need(model.at("status").materialize() == "available" &&
                 summary.at("model_status") == "available",
             "unsupported fit model status");
        const auto m = model.at("data");
        need(m.at("protocol").materialize() == "symphony.sbv.linear-model.v1",
             "unsupported history model profile");
        model_training = m.at("training");
        need(str(model_training.at("identity_namespace").materialize()) == ns &&
                 model_training.at("source_result_sha256").materialize() ==
                     provenance.at("source_result_sha256") &&
                 model_training.at("source_pointer").materialize() ==
                     source_pointer,
             "inconsistent training provenance");
        if (choices.at("source").contains("kind"))
          need(model_training.contains("source_selection") &&
                   model_training.at("source_selection").materialize() ==
                       data(r, "source_selection").materialize(),
               "model training selection differs from fit source");
        selected_ids = ids(model_training.at("observation_ids"), end);
        need(u64(summary.at("training_rows")) == selected_ids.size(),
             "training identity count mismatch");
      }
    } else {
      need(summary.at("target_supplied").is_boolean(),
           "prediction target availability required");
      targets = summary.at("target_supplied").get<bool>();
      need(targets == !choices.at("target").is_null(),
           "prediction target availability mismatch");
      const auto predictions = data(r, "predictions");
      need(predictions.kind() == "array", "prediction observations required");
      std::set<std::string> unique;
      auto predicted = predictions.children();
      while (auto row = predicted.next()) {
        deadline(end);
        const auto value = row->value.at("id").materialize();
        text(value);
        need(unique.insert(str(value)).second,
             "duplicate history observation identity");
        selected_ids.push_back(str(value));
        need(targets != row->value.at("target").materialize().is_null(),
             "prediction target row mismatch");
      }
      need(u64(summary.at("predicted_rows")) == selected_ids.size(),
           "prediction identity count mismatch");
      model_training = data(r, "model_reference").at("training");
    }
    if (!known) {
      need(p.at("on_unavailable") == "retain",
           "history observation identities unavailable");
      ++unresolved;
    }
    const bool include =
        known &&
        (duplicate.is_null() || p.at("duplicate_artifacts") == "count_entries");
    if (known && !include)
      ++collapsed;
    consumer::Rows reused(context, "entry-reuse-" + dec(entries.size()));
    std::set<std::string> previous_entries;
    std::size_t repeated = 0, prior_fit = 0, prior_predict = 0,
                prior_targets = 0;
    if (include) {
      ++counted;
      for (const auto &obs_id : selected_ids) {
        deadline(end);
        auto &record = observations[{ns, obs_id}];
        bool fitted = false, predicted = false, exposed = false;
        Json prior_ids = Json::array();
        for (const auto &use : record.uses) {
          fitted |= use.kind == "fit";
          predicted |= use.kind == "predict";
          exposed |= use.targets;
          previous_entries.insert(use.entry_id);
          if (retain)
            prior_ids.push_back(use.entry_id);
        }
        if (!record.uses.empty()) {
          ++repeated;
          if (retain)
            reused.append({{"id", obs_id}, {"prior_entry_ids", prior_ids}});
        }
        prior_fit += fitted;
        prior_predict += predicted;
        prior_targets += exposed;
        record.uses.push_back({id, kind, targets});
        ++uses;
      }
    }
    Json retained_entry{
        {"id", id},
        {"ordinal", dec(entries.size())},
        {"kind", kind},
        {"role", entry.at("role")},
        {"source", source.reference},
        {"identity_namespace", ns},
        {"source_purpose", purpose},
        {"observation_source",
         {{"content_sha256", provenance.at("source_result_sha256")},
          {"pointer", source_pointer}}},
        {"identities_available", known},
        {"observation_count",
         known ? Json(dec(selected_ids.size())) : Json(nullptr)},
        {"counted", include},
        {"duplicate_of", duplicate},
        {"targets_supplied", targets},
        {"prior_entry_count",
         include ? Json(dec(previous_entries.size())) : Json(nullptr)},
        {"reused_observation_count",
         include ? Json(dec(repeated)) : Json(nullptr)},
        {"previously_fitted_observation_count",
         include ? Json(dec(prior_fit)) : Json(nullptr)},
        {"previously_predicted_observation_count",
         include ? Json(dec(prior_predict)) : Json(nullptr)},
        {"previously_targeted_observation_count",
         include ? Json(dec(prior_targets)) : Json(nullptr)},
        {"reused_observations", nullptr},
        {"model_training_reference", nullptr},
        {"reason",
         !known
             ? "fit retained no model observation identities; usage is unknown"
         : !include ? "duplicate artifact collapsed by selection"
                    : ""},
        {"extensions", entry.at("extensions")}};
    entries.append(logical::Value(retained_entry)
                       .with("reused_observations",
                             retain && include ? reused.close()
                                               : logical::Value(Json(nullptr)))
                       .with("model_training_reference", model_training));
  }
  std::set<std::string> selection_names;
  auto selection_cursor = selection_input.value.children();
  while (auto child = selection_cursor.next()) {
    deadline(end);
    const auto selection = child->value.materialize();
    keys(selection,
         {"id", "source", "chosen_candidate_ids", "reason", "extensions"});
    text(selection.at("id"));
    text(selection.at("reason"));
    need(selection_names.insert(str(selection.at("id"))).second &&
             selection.at("extensions").is_object(),
         "unique selection identity and extensions required");
    auto source = load(selection.at("source"), total_bytes, bundled, end);
    sources.append(source.reference);
    const auto r = source.result, choices = data(r, "choices"),
               comparison = data(r, "comparisons");
    need(choices.at("protocol").materialize() ==
             "symphony.sbv.compare-input.v1",
         "selection requires a supplied comparison artifact");
    const auto candidates = comparison.at("candidates");
    need(candidates.kind() == "array", "comparison candidates required");
    std::map<std::string, logical::Value> by_id;
    auto candidate_cursor = candidates.children();
    while (auto child = candidate_cursor.next()) {
      auto c = child->value;
      auto id = c.at("id").materialize();
      text(id);
      need(by_id.emplace(str(id), c).second, "duplicate comparison candidate");
    }
    consumer::Rows selected(context,
                            "selected-candidates-" + dec(selections.size()));
    for (const auto &id :
         ids(logical::Value(selection.at("chosen_candidate_ids")), end)) {
      need(by_id.contains(id), "selected candidate absent from comparison");
      selected.append(by_id.at(id));
    }
    auto retained_selection =
        logical::Value(
            Json{{"id", selection.at("id")},
                 {"ordinal", dec(selections.size())},
                 {"source", source.reference},
                 {"chosen_candidate_ids", selection.at("chosen_candidate_ids")},
                 {"chosen_candidates", nullptr},
                 {"reason", selection.at("reason")},
                 {"candidates", nullptr},
                 {"objectives", nullptr},
                 {"methods", choices.at("methods").materialize()},
                 {"weighted_order", nullptr},
                 {"pareto_fronts", nullptr},
                 {"extensions", selection.at("extensions")}})
            .with("chosen_candidates", selected.close())
            .with("candidates", candidates)
            .with("objectives", r.at("sections").contains("objective_choices")
                                    ? data(r, "objective_choices")
                                    : choices.at("objectives"))
            .with("weighted_order", comparison.at("weighted_order"))
            .with("pareto_fronts", comparison.at("pareto_fronts"));
    selections.append(retained_selection);
  }
  consumer::Rows identity_rows(context, "observation-usage");
  std::size_t repeated = 0;
  std::set<std::string> namespaces;
  for (const auto &[key, record] : observations) {
    deadline(end);
    namespaces.insert(key.first);
    repeated += record.uses.size() > 1;
    if (retain) {
      Json listed = Json::array();
      for (const auto &use : record.uses)
        listed.push_back({{"entry_id", use.entry_id},
                          {"kind", use.kind},
                          {"targets_supplied", use.targets}});
      identity_rows.append({{"identity_namespace", key.first},
                            {"id", key.second},
                            {"occurrence_count", dec(record.uses.size())},
                            {"occurrences", listed}});
    }
  }
  auto result = base("supplied research usage and selection history");
  auto &s = result["sections"];
  if (unresolved)
    result["status"] = "partial";
  s["summary"] =
      section({{"entries", dec(entries.size())},
               {"counted_entries", dec(counted)},
               {"collapsed_entries", dec(collapsed)},
               {"unresolved_entries", dec(unresolved)},
               {"counted_observation_occurrences", dec(uses)},
               {"unique_namespaced_observations", dec(observations.size())},
               {"reused_namespaced_observations", dec(repeated)},
               {"counted_namespaces", dec(namespaces.size())},
               {"selection_records", dec(selections.size())}});

  auto choices = p;
  choices.erase("output_path");
  choices.erase("output");
  s["choices"] = section(choices);
  s["resources"] =
      section({{"backend", "cpu"},
               {"actual_workers", entry_input.value.size() == 0 &&
                                          selection_input.value.size() == 0
                                      ? "0"
                                      : "1"},
               {"source_bytes_read",
                bundled ? Json(nullptr) : Json(dec(total_bytes))}});
  s["provenance"] = section({{"engine_version", version},
                             {"sources", nullptr},
                             {"provider_requests", "0"},
                             {"additional_spend_usd", "0"}});
  s["diagnostics"] = section(Json::array(
      {"Counts describe supplied records in caller array order, not "
       "authenticated event times, physical reads or a complete access log.",
       "Matching namespace and observation ID permits exact identity "
       "comparison; different namespaces are not compared or assumed disjoint.",
       "A repeated artifact is not proof of repeated execution. The explicit "
       "duplicate-artifact policy determines counting; all entries remain "
       "visible.",
       "Retained fit IDs and prediction rows include zero-weight observations; "
       "excluded rows and unavailable fit identities are not inferred.",
       "Targets supplied records declared fit/prediction input availability, "
       "not human inspection, causal label overlap, independence or untouched "
       "holdouts.",
       "Selection records bind the supplied comparison and caller choices. "
       "Choices may include lower-ranked, failed or pruned candidates; no "
       "automatic promotion gate.",
       "Artifact hashes establish supplied content identity, not authorship or "
       "completeness. This operation does not recompute source estimates or "
       "infer missing history."}));
  s["user_extensions"] = section(p.at("extensions"));
  if (bundled) {
    s["resources"]["data"]["legacy_source_bytes_read"] = dec(total_bytes);
    s["resources"]["data"]["source_bytes_read_scope"] =
        "aggregate bundle physical I/O not collected here; per-reader counters "
        "available via bundle operations";
  }
  auto out =
      logical::Value(s)
          .with("research_usage", consumer::section(entries.close()))
          .with("selection_history", consumer::section(selections.close()))
          .with("observation_usage",
                retain
                    ? consumer::section(identity_rows.close())
                    : consumer::section(
                          logical::Value(Json(nullptr)), "not_selected",
                          "per-observation occurrence retention not selected"))
          .with("provenance",
                logical::Value(s.at("provenance"))
                    .with("data", logical::Value(s.at("provenance").at("data"))
                                      .with("sources", sources.close())));
  if (entry_input.evidence)
    out = out.with("entry_input", consumer::section(*entry_input.evidence));
  if (selection_input.evidence)
    out = out.with("selection_input",
                   consumer::section(*selection_input.evidence));
  return logical::Value(result).with("sections", out);
}
Json research_history(const Json &p, std::int64_t end) {
  need(p.contains("output") != p.contains("output_path"),
       "select exactly one history output");
  if (p.contains("output"))
    return partitioned_result(p, "research-history", end,
                              [&](PartitionedOutput &context) {
                                return history_body(p, &context, end);
                              });
  return persist(history_body(p, nullptr, end)
                     .materialize({artifact_bytes, artifact_values}),
                 p, "research-history", end);
}
} // namespace symphony::sbv::detail
