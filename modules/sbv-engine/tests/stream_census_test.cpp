#include "row_spool.hpp"
#include "stream_census.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace l = s::logical;
namespace store = s::result_store;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("stream census test line " +
                             std::to_string(at.line()));
}
template <class F> void rejects(F f) {
  bool bad = false;
  try {
    f();
  } catch (const std::exception &) {
    bad = true;
  }
  check(bad);
}
const auto end = e::no_deadline;
J producer{{"id", "external-user"},
           {"version", "1"},
           {"artifact_sha256", ""},
           {"reproducibility", "uncaptured"}};
J row(unsigned i, unsigned ordinal = 1) {
  return {{"signal_id", "s-" + std::to_string(i)},
          {"source_ordinal", std::to_string(ordinal)},
          {"available_ns", std::to_string(100 + ordinal)},
          {"anchor_price_nanos", "777"},
          {"causal_end_ordinal_exclusive", std::to_string(ordinal + 1)},
          {"context_reference", "user:/é~"}};
}
J external(J rows, const d::Dataset &source) {
  return {{"protocol", "symphony.sbv.external-census.v1"},
          {"source_sha256", source.sha256},
          {"mode", "causal_declared"},
          {"producer", producer},
          {"signals", rows}};
}
store::Receipt write_body(const std::string &path, const l::Value &body,
                          store::WriteOptions options = {4096, 4}) {
  store::ResultWriter w(path, options);
  body.write(w);
  return w.finish();
}
J reference(const store::Receipt &r, const std::string &pointer) {
  return {
      {"kind", "bundle"},
      {"reference",
       {{"manifest_path", r.reference.manifest_path},
        {"manifest_sha256", r.reference.manifest_sha256},
        {"content_sha256", r.reference.content_sha256}}},
      {"selector", {{"kind", "pointer"}, {"pointer", pointer}}},
      {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "0"}}}};
}
int main() try {
#ifdef __APPLE__
  char pattern[] = "/private/tmp/sbv-stream-census-XXXXXX";
#else
  char pattern[] = "/tmp/sbv-stream-census-XXXXXX";
#endif
  auto made = ::mkdtemp(pattern);
  check(made);
  const std::string root = made;
  d::Dataset dataset;
  dataset.path = root + "/not-needed.dbn";
  dataset.sha256 = std::string(64, 'a');
  dataset.dataset_name = "MECHANICAL";
  dataset.events.resize(5);
  for (unsigned i = 0; i < 5; ++i) {
    auto &ev = dataset.events[i];
    ev.instrument_id = 7;
    ev.ts_recv = 100 + i;
    ev.action = 'T';
    ev.price = 1000 + i;
    ev.size = 1;
  }
  auto declaration = external(J::array({row(0), row(1), row(2, 3)}), dataset);
  J legacy_reference;
  auto old = d::admit_census(declaration, dataset, legacy_reference, end);
  auto admitted = d::admit_stream_census(declaration, dataset, end);
  check(admitted.reference.is_null());
  check(admitted.value.materialize() == old);
  check(old.at("census_sha256") == e::sha256_hex(declaration.dump()));
  auto invalid = declaration;
  invalid["signals"][2]["signal_id"] = "s-0";
  rejects([&] { d::admit_stream_census(invalid, dataset, end); });
  invalid = declaration;
  invalid["signals"][0]["available_ns"] = "102";
  rejects([&] { d::admit_stream_census(invalid, dataset, end); });
  invalid = declaration;
  invalid["signals"][0]["causal_end_ordinal_exclusive"] = "3";
  rejects([&] { d::admit_stream_census(invalid, dataset, end); });
  invalid["mode"] = "retrospective";
  check(d::admit_stream_census(invalid, dataset, end)
            .value.at("mode")
            .materialize() == "retrospective");
  invalid["signals"][0]["causal_end_ordinal_exclusive"] = "6";
  rejects([&] { d::admit_stream_census(invalid, dataset, end); });
  check(d::admit_stream_census(external(J::array(), dataset), dataset, end)
            .value.at("signals")
            .size() == 0);
  auto plain = d::base("typed external declarations");
  plain["sections"]["user_declaration"] = d::section(declaration);
  auto saved = write_body(root + "/declaration", l::Value(plain));
  auto selection = reference(saved, "/sections/user_declaration/data");
  auto from_bundle = d::admit_stream_census(selection, dataset, end);
  check(from_bundle.value.materialize() == old);
  check(from_bundle.reference.at("selected_content_sha256").is_null());
  check(from_bundle.reference.at("selected_value_sha256") ==
        old.at("census_sha256"));
  auto evaluation = d::base("retained evaluation context");
  auto &sections = evaluation["sections"];
  sections["signals"] = d::section(old.at("signals"));
  sections["census"] = d::section(old);
  sections["census_reference"] = d::section(from_bundle.reference);
  sections["provenance"] = d::section({{"source_sha256", dataset.sha256},
                                       {"dataset", dataset.dataset_name},
                                       {"instrument_id", "7"},
                                       {"census_producer", producer}});
  sections["choices"] =
      d::section({{"protocol", "symphony.sbv.evaluate-input.v1"},
                  {"source_sha256", dataset.sha256},
                  {"dataset", dataset.dataset_name},
                  {"census", selection}});
  sections["summary"] = d::section({{"closed_census", true},
                                    {"signal_count", "3"},
                                    {"census_sha256", old.at("census_sha256")},
                                    {"causality", "causal_declared"}});
  check(d::stream_census_evidence(l::Value(evaluation), end).materialize() ==
        old);
  auto forged = evaluation;
  forged["sections"]["census_reference"]["data"]["selected_value_sha256"] =
      std::string(64, 'b');
  rejects([&] { d::stream_census_evidence(l::Value(forged), end); });
  forged = evaluation;
  forged["sections"]["census_reference"]["data"]["selector"]["pointer"] = "/~2";
  forged["sections"]["choices"]["data"]["census"]["selector"]["pointer"] =
      "/~2";
  rejects([&] { d::stream_census_evidence(l::Value(forged), end); });
  forged = evaluation;
  forged["sections"]["census"]["status"] = "unavailable";
  rejects([&] { d::stream_census_evidence(l::Value(forged), end); });
  // Missing legacy evidence is normalized from the complete inline original.
  // A present malformed declaration must never take that fallback path.
  auto legacy_inline = evaluation;
  legacy_inline["sections"]["choices"]["data"]["census"] = declaration;
  legacy_inline["sections"].erase("census");
  legacy_inline["sections"].erase("census_reference");
  check(d::stream_census_evidence(l::Value(legacy_inline), end).materialize() ==
        d::census_evidence(legacy_inline));
  for (const auto &bad : {J(nullptr), J::object(), declaration}) {
    auto present = legacy_inline;
    present["sections"]["census"] = d::section(bad);
    rejects([&] { d::stream_census_evidence(l::Value(present), end); });
  }
  auto generated_missing = legacy_inline;
  generated_missing["sections"]["choices"]["data"]["protocol"] =
      "symphony.sbv.generate-census-input.v1";
  rejects([&] {
    d::stream_census_evidence(l::Value(generated_missing), end);
  });
  auto full = s::seal_result(evaluation);
  d::validate_logical_result(l::Value(full), end);
  auto outer = d::base("embedded evaluation");
  outer["sections"]["embedded"] = d::section(full);
  auto nested = write_body(root + "/nested", l::Value(outer));
  auto selected = d::select_logical_result(
      reference(nested, "/sections/embedded/data"), end);
  check(selected.value.materialize() == full);
  check(selected.reference.at("selected_content_sha256") ==
        full.at("content_sha256"));
  auto broken = outer;
  broken["sections"]["embedded"]["data"]["content_sha256"] =
      std::string(64, 'b');
  auto broken_ref = write_body(root + "/inner-broken", l::Value(broken));
  rejects([&] {
    d::select_logical_result(reference(broken_ref, "/sections/embedded/data"),
                             end);
  });
  rejects([&] {
    d::select_logical_result(
        reference(saved, "/sections/user_declaration/data"), end);
  });
  auto whole = d::select_logical_result(reference(nested, ""), end);
  check(whole.reference.at("selected_node_id") == "0" &&
        whole.reference.at("selected_content_sha256") ==
            nested.reference.content_sha256);
  // Re-evaluation from an actual complete root binds both root coordinates
  // and selected-content identity, even after the ancestor is unavailable.
  const auto parent_saved = write_body(root + "/evaluation", l::Value(evaluation));
  const auto parent_selection = reference(parent_saved, "");
  const auto parent_admitted = d::admit_stream_census(parent_selection, dataset, end);
  auto descendant = evaluation;
  descendant["sections"]["choices"]["data"]["census"] = parent_selection;
  descendant["sections"]["census_reference"] = d::section(parent_admitted.reference);
  check(d::stream_census_evidence(l::Value(descendant), end).materialize() == old);
  for (const auto &change : {
           J{{"selected_node_id", "2"}},
           J{{"selected_content_sha256", std::string(64, 'c')}},
           J{{"selected_content_sha256", nullptr}},
           J{{"verification_extent", "selected_subtree"}},
           J{{"authorship", "verified"}}}) {
    auto mismatched = descendant;
    mismatched["sections"]["census_reference"]["data"].update(change);
    rejects([&] { d::stream_census_evidence(l::Value(mismatched), end); });
  }
  auto node_selected = descendant;
  node_selected["sections"]["choices"]["data"]["census"]["selector"] =
      J{{"kind", "node_id"}, {"node_id", "0"}};
  node_selected["sections"]["census_reference"]["data"]["selector"] =
      J{{"kind", "node_id"}, {"node_id", "0"}};
  check(d::stream_census_evidence(l::Value(node_selected), end).materialize() == old);
  for (const auto &bad_selector : {
           J{{"kind", "pointer"}, {"pointer", "/sections/embedded/data"}},
           J{{"kind", "node_id"}, {"node_id", "2"}},
           J{{"kind", "pointer"}, {"pointer", "/bad~2escape"}}}) {
    auto mismatched = descendant;
    mismatched["sections"]["choices"]["data"]["census"]["selector"] = bad_selector;
    mismatched["sections"]["census_reference"]["data"]["selector"] = bad_selector;
    rejects([&] { d::stream_census_evidence(l::Value(mismatched), end); });
  }
  std::filesystem::remove_all(root + "/evaluation");
  check(d::stream_census_evidence(l::Value(descendant), end).materialize() == old);
  // A large external declaration is stored as streamed rows. Exact ID matching
  // spans many pages and same-event candidates retain their original order.
  store::RowSpool spool(root + "/large-rows", {4096, 4});
  for (unsigned i = 0; i < 4097; ++i)
    spool.append(row(i));
  auto rows = spool.close();
  auto large = l::Value(external(J::array(), dataset))
                   .with("signals", l::Value(rows.rows));
  auto carrier = l::Value(d::base("large external declaration"));
  auto ss = carrier.at("sections")
                .with("declaration",
                      l::Value(d::section(nullptr)).with("data", large));
  carrier = carrier.with("sections", ss);
  auto large_saved = write_body(root + "/large-declaration", carrier);
  auto big = d::admit_stream_census(
      reference(large_saved, "/sections/declaration/data"), dataset, end);
  check(big.value.at("signals").size() == 4097);
  check(big.value.at("signals").at(std::uint64_t{4096}).materialize() ==
        row(4096));
  check(big.value.at("census_sha256").materialize() == large.sha256());
  // Keep a large original inline declaration and unrelated aggregate choices
  // as owning values; normalization must neither rewrite nor truncate them.
  auto inline_choices = l::Value(legacy_inline.at("sections").at("choices").at("data"))
                            .with("census", large)
                            .with("extensions", l::Value::object({
                                {"unknown_array", l::Value(rows.rows)}}));
  auto large_summary = sections.at("summary").at("data");
  large_summary["signal_count"] = "4097";
  large_summary["census_sha256"] = big.value.at("census_sha256").materialize();
  auto large_sections = l::Value(legacy_inline.at("sections"))
                            .with("signals", l::Value(d::section(nullptr)).with("data", l::Value(rows.rows)))
                            .with("choices", l::Value(d::section(nullptr)).with("data", inline_choices))
                            .with("summary", l::Value(d::section(large_summary)));
  auto metadata_array = J::array();
  for (unsigned i = 0; i < 20; ++i)
    metadata_array.push_back(std::string(60000, char('a' + i)));
  auto provenance = legacy_inline.at("sections").at("provenance").at("data");
  provenance["user_unknown_array"] = metadata_array;
  large_sections = large_sections.with("provenance", l::Value(d::section(provenance)));
  auto inline_result = l::Value(legacy_inline).with("sections", large_sections);
  const auto before_choices = inline_choices.sha256();
  const auto large_inline = write_body(root + "/inline-large", inline_result, {131072, 4});
  auto selected_inline = d::select_logical_result(reference(large_inline, ""), end);
  const auto normalized_inline = d::stream_census_evidence(selected_inline.value, end);
  check(normalized_inline.sha256() == big.value.sha256());
  check(selected_inline.value.at("sections").at("choices").at("data").sha256() == before_choices);
  check(selected_inline.value.at("sections").at("provenance").at("data")
            .at("user_unknown_array").sha256() == e::sha256_hex(metadata_array.dump()));
  // Reconcile every declaration hash in a multi-page distant-duplicate case:
  // duplicate admission, not an unrelated hash mismatch, must reject it.
  store::RowSpool duplicate_spool(root + "/duplicate-rows", {4096, 4});
  for (unsigned i = 0; i < 4097; ++i)
    duplicate_spool.append(row(i == 4096 ? 0 : i));
  auto duplicates = duplicate_spool.close();
  check(duplicates.scratch_receipt.page_files_created > 2);
  auto duplicate_declaration = l::Value(external(J::array(), dataset))
                                   .with("signals", l::Value(duplicates.rows));
  auto duplicate_evidence = big.value.with("signals", l::Value(duplicates.rows))
                               .with("declaration", duplicate_declaration)
                               .with("census_sha256", l::Value(J(duplicate_declaration.sha256())));
  rejects([&] { d::validate_stream_census(duplicate_evidence, end); });
  std::filesystem::remove_all(root + "/large-rows");
  check(big.value.at("signals").at(std::uint64_t{0}).materialize() == row(0));
  // Native identity hashes signals alone; null cap retains exact trade anchors.
  J native_rows = J::array();
  for (unsigned i = 0; i < 5; ++i)
    native_rows.push_back(
        {{"signal_id", "n-" + std::to_string(i)},
         {"source_ordinal", std::to_string(i)},
         {"available_ns", std::to_string(100 + i)},
         {"anchor_price_nanos", std::to_string(1000 + i)},
         {"causal_end_ordinal_exclusive", std::to_string(i + 1)},
         {"previous_trade_price_nanos",
          i ? J(std::to_string(999 + i)) : d::missing("first trade")}});
  J native = {{"protocol", "symphony.sbv.census-evidence.v1"},
              {"kind", "native"},
              {"identity_domain", "native_signals"},
              {"census_sha256", e::sha256_hex(native_rows.dump())},
              {"source_sha256", dataset.sha256},
              {"dataset", dataset.dataset_name},
              {"instrument_id", "7"},
              {"mode", "native_causal_prefix"},
              {"producer", nullptr},
              {"signals", native_rows},
              {"declaration",
               {{"criteria",
                 {{"rule", "spaced_trades"},
                  {"spacing_ns", "0"},
                  {"min_trade_size", "1"},
                  {"direction", "any"},
                  {"max_signals", nullptr}}},
                {"engine_version", "admission-fixture"},
                {"selection_cap", nullptr},
                {"additional_eligible_after_cap", "0"}}}};
  d::validate_stream_census(l::Value(native), end);
  d::validate_stream_census_source(l::Value(native), dataset, end);
  check(true);
  auto wrong = native;
  wrong["signals"][2]["anchor_price_nanos"] = "999";
  wrong["census_sha256"] = e::sha256_hex(wrong.at("signals").dump());
  d::validate_stream_census(l::Value(wrong), end);
  rejects(
      [&] { d::validate_stream_census_source(l::Value(wrong), dataset, end); });
  wrong = native;
  wrong["declaration"]["additional_eligible_after_cap"] = "1";
  rejects([&] { d::validate_stream_census(l::Value(wrong), end); });
  // The historic finite-cap native fallback has exactly the old identity.
  auto historic_native = native;
  historic_native["declaration"]["criteria"]["max_signals"] = "5";
  historic_native["declaration"]["selection_cap"] = "5";
  auto native_result = d::base("legacy native fallback");
  native_result["sections"]["signals"] = d::section(native_rows);
  native_result["sections"]["choices"] = d::section({
      {"protocol", "symphony.sbv.run-input.v1"},
      {"source_sha256", dataset.sha256}, {"dataset", dataset.dataset_name},
      {"criteria", historic_native.at("declaration").at("criteria")}});
  native_result["sections"]["provenance"] = d::section({
      {"source_sha256", dataset.sha256}, {"dataset", dataset.dataset_name},
      {"instrument_id", "7"}, {"engine_version", "admission-fixture"}});
  native_result["sections"]["summary"] = d::section({
      {"closed_census", true}, {"signal_count", "5"},
      {"census_sha256", native.at("census_sha256")},
      {"selection_cap", "5"}, {"additional_eligible_after_cap", "0"}});
  check(d::stream_census_evidence(l::Value(native_result), end).materialize() == historic_native);
  check(d::census_evidence(native_result) == historic_native);
  for (const auto &bad : {J(nullptr), J::object(), declaration}) {
    auto present = native_result;
    present["sections"]["census"] = d::section(bad);
    rejects([&] { d::stream_census_evidence(l::Value(present), end); });
  }
  auto native_evaluation = evaluation;
  native_evaluation["sections"]["signals"] = d::section(native_rows);
  native_evaluation["sections"]["census"] = d::section(native);
  native_evaluation["sections"]["provenance"]["data"]["census_producer"] = nullptr;
  native_evaluation["sections"]["summary"]["data"]["causality"] = "native_causal_prefix";
  native_evaluation["sections"]["summary"]["data"]["signal_count"] = "5";
  native_evaluation["sections"]["summary"]["data"]["census_sha256"] = native.at("census_sha256");
  native_evaluation["sections"]["census_reference"]["data"]["selected_value_sha256"] = native.at("census_sha256");
  rejects([&] { d::stream_census_evidence(l::Value(native_evaluation), end); });
  std::cout
      << J{{"checks", std::to_string(checks)},
           {"status", "passed"},
           {"directory", root},
           {"claim_scope",
            "streamed census admission only; no large strategy execution"}}
             .dump()
      << "\n";
} catch (const std::exception &ex) {
  std::cerr << ex.what() << "\n";
  return 1;
}
