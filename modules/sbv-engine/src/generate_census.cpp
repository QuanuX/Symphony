#include "census.hpp"
#include "provider.hpp"
#include <symphony/sqav/databento/dbn.hpp>

namespace symphony::sbv::detail {
Json generate_census(const Json &p, std::int64_t end, const Dataset *resident) {
  keys_optional(p, {"protocol", "provider", "output_path", "extensions"},
                {"memory_budget_bytes", "dataset_limits", "source_path",
                 "source_sha256", "dataset", "retained_source"});
  need(p.at("extensions").is_object(), "extensions object required");
  const auto &selection = p.at("provider");
  need(selection.at("role") == "strategy" &&
           selection.at("concurrency") == "serialized_instance",
       "census generation requires a serialized strategy provider");
  const auto owned = resident ? nullptr : load_dataset(p, end);
  const auto &source = resident ? *resident : *owned;
  source.bind(p);
  const auto &events = source.events;
  NativeProvider provider(selection, end);
  auto instance = provider.create(end);
  Json signals = Json::array();
  std::set<std::string> ids;
  // Only invocation-scoped prefix readers enter strategy callbacks. The
  // trusted library shares this process; this is not memory or I/O isolation.
  for (std::size_t ordinal = 0; ordinal < events.size(); ++ordinal) {
    deadline(end);
    auto reply = instance->event(events, ordinal, end);
    need(reply.status == SBV_PROVIDER_OK,
         ("strategy provider failed: " + reply.error.dump()).c_str());
    if (reply.value.is_null())
      continue;
    need(reply.value.is_array(), "strategy candidate array required");
    for (const auto &candidate : reply.value) {
      deadline(end);
      keys(candidate, {"signal_id", "anchor_price_nanos", "context_reference"});
      const auto id = str(candidate.at("signal_id"));
      need(!id.empty() && id.size() <= 256 && ids.insert(id).second,
           "unique bounded provider signal identity required");
      need(i64(candidate.at("anchor_price_nanos")) != INT64_MAX,
           "provider signal anchor cannot be undefined");
      (void)str(candidate.at("context_reference"));
      need(ordinal < UINT64_MAX, "provider causal ordinal overflows");
      signals.push_back(
          {{"signal_id", id},
           {"source_ordinal", dec(ordinal)},
           {"available_ns", dec(events[ordinal].ts_recv)},
           {"anchor_price_nanos", candidate.at("anchor_price_nanos")},
           {"causal_end_ordinal_exclusive", dec(ordinal + 1)},
           {"context_reference", candidate.at("context_reference")}});
    }
  }
  auto finished = instance->finish(end);
  need(finished.status == SBV_PROVIDER_OK,
       ("strategy completion failed: " + finished.error.dump()).c_str());
  auto completion =
      finished.value.is_null()
          ? Json{{"diagnostics", Json::array()}, {"extensions", Json::object()}}
          : finished.value;
  keys(completion, {"diagnostics", "extensions"});
  need(completion.at("diagnostics").is_array() &&
           completion.at("extensions").is_object(),
       "provider completion must contain diagnostics and extensions");
  for (const auto &message : completion.at("diagnostics"))
    (void)str(message);
  // Finish may return diagnostics, never more signals. Destroy only after
  // every copied/released callback buffer is no longer provider-owned.
  instance.reset();
  const auto &descriptor = provider.descriptor();
  const Json author{
      {"id", descriptor.at("id")},
      {"version", descriptor.at("version")},
      {"artifact_sha256", selection.at("library").at("expected_sha256")},
      {"reproducibility", descriptor.at("reproducibility")}};
  Json declaration{{"protocol", "symphony.sbv.native-provider-census.v1"},
                   {"source_sha256", source.sha256},
                   {"dataset", source.dataset_name},
                   {"instrument_id", dec(events.front().instrument_id)},
                   {"provider", selection},
                   {"provider_evidence", provider.evidence()},
                   {"signals", signals},
                   {"completion", completion}};
  const auto census_digest = e::sha256_hex(declaration.dump());
  Json census{{"protocol", "symphony.sbv.census-evidence.v1"},
              {"kind", "native_provider"},
              {"identity_domain", "native_provider_declaration"},
              {"census_sha256", census_digest},
              {"source_sha256", source.sha256},
              {"dataset", source.dataset_name},
              {"instrument_id", dec(events.front().instrument_id)},
              {"mode", "native_provider_prefix"},
              {"producer", author},
              {"signals", signals},
              {"declaration", declaration}};
  validate_census_evidence(census);
  auto result = base("native strategy provider closed-census generation");
  result["status"] = "partial";
  auto &s = result["sections"];
  s["summary"] = section({{"signal_count", dec(signals.size())},
                          {"record_count", dec(events.size())},
                          {"closed_census", true},
                          {"census_sha256", census_digest},
                          {"causality", "native_provider_prefix"},
                          {"provider_causality_verified", false},
                          {"execution_evaluated", false}});
  s["signals"] = section(signals);
  s["census"] = section(census);
  s["provider"] =
      section({{"evidence", provider.evidence()}, {"completion", completion}});
  s["execution"] = section(Json::array(), "not_selected",
                           "census pass does not invoke execution models");
  s["replay"] = section(Json::object(), "not_selected",
                        "select before/after replay in a later evaluation");
  s["studies"] = section(Json::array(), "not_selected",
                         "no study is selected by census generation");
  auto choices = p;
  choices.erase("output_path");
  s["choices"] = section(choices);
  const auto &meta = source.metadata;
  s["provenance"] = section(
      {{"engine_version", version},
       {"source_sha256", source.sha256},
       {"source_path", source.path},
       {"dataset", source.dataset_name},
       {"adapter", sqav::databento::adapter_id},
       {"adapter_version", sqav::databento::adapter_version},
       {"dbn_version", dec(meta.version)},
       {"instrument_id", dec(events.front().instrument_id)},
       {"source_record_limit", dec(meta.limit)},
       {"source_cap_reached", meta.limit != 0 && events.size() >= meta.limit},
       {"requested_start_ns", dec(meta.start)},
       {"requested_end_ns_exclusive", dec(meta.end)},
       {"observed_first_ns", dec(events.front().ts_recv)},
       {"observed_last_ns", dec(events.back().ts_recv)},
       {"census_producer", author},
       {"census_sha256", census_digest},
       {"provider_requests", "0"},
       {"additional_spend_usd", "0"},
       {"authorship",
        "selected trusted native library; no signature verified"}});
  s["resources"] =
      section({{"backend", "cpu"},
               {"actual_workers", "1"},
               {"provider_instances", "1"},
               {"provider_concurrency", "serialized_instance"},
               {"event_calls", dec(events.size())},
               {"finish_calls", "1"},
               {"dataset_feed", source.evidence(resident != nullptr)}});
  s["diagnostics"] = section(Json::array(
      {"Strategy callbacks receive a checked prefix ending at the current "
       "event. No model callback runs during census generation.",
       "Trusted in-process code can perform other I/O or retain independent "
       "state. Prefix admission is not sandbox isolation or proof of "
       "causality.",
       "Signals retain provider order within each event; source coordinates "
       "and the complete census artifact are authored by the native host.",
       "No policy signal cap is imposed by this operation. Host addressing and "
       "the current portable artifact representation still limit publication.",
       "Provider dependency and reproducibility declarations remain attributed "
       "claims; original market completeness and book initialization "
       "unproven."}));
  s["user_extensions"] = section(p.at("extensions"));
  return persist(std::move(result), p, "generate-census", end);
}
} // namespace symphony::sbv::detail
