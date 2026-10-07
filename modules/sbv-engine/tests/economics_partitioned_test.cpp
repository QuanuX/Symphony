#include "economic_census_stream.hpp"
#include "economics_kernel.hpp"
#include "partitioned_output.hpp"
#include "stream_census.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/sbv/models.hpp>
#include <unistd.h>
namespace s = symphony::sbv;
namespace d = s::detail;
namespace l = s::logical;
namespace store = s::result_store;
namespace e = symphony::knowledge::engine;
using J = s::Json;
namespace symphony::sbv::detail {
Json economics_partitioned(const Json &, std::int64_t);
}
unsigned checks = 0, serial = 0;
constexpr auto end = e::no_deadline;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("partitioned economics assertion line " +
                             std::to_string(at.line()));
}
J ratio(const char *n, const char *denominator = "1") {
  return {{"numerator", n}, {"denominator", denominator}};
}
J producer{{"id", "mechanical-economics-fixture"},
           {"version", "1"},
           {"artifact_sha256", ""},
           {"reproducibility", "uncaptured"}};
J signal(std::uint64_t index) {
  return {{"signal_id", "signal-" + std::to_string(index)},
          {"source_ordinal", "0"},
          {"available_ns", "1000"},
          {"anchor_price_nanos", "100000000000"},
          {"causal_end_ordinal_exclusive", "1"},
          {"context_reference", "user://mechanical-fixture"}};
}
J supplied(std::uint64_t index) {
  return {
      {"signal_id", signal(index).at("signal_id")},
      {"evidence_reference", "mechanical arithmetic, not market evidence"},
      {"execution_probability",
       {{"status", "supplied"}, {"value", ratio("3", "5")}}},
      {"support",
       J::array(
           {{{"price_nanos", "99000000000"}, {"weight", ratio("1", "2")}},
            {{"price_nanos", "101000000000"}, {"weight", ratio("1", "2")}}})}};
}
J model_selection() {
  return {{"protocol", "symphony.sbv.model-selection.v1"},
          {"id", "external_outcomes"},
          {"version", "1"},
          {"horizon_ns", "10"},
          {"parameters",
           {{"producer", producer},
            {"measure", "probability"},
            {"conditioning", "execution"},
            {"calibration_reference", ""},
            {"outcomes", J::array({supplied(0)})}}}};
}
J transform() {
  return {{"id", "linear_price_pnl"},
          {"version", "1"},
          {"reference_price_nanos", "100000000000"},
          {"price_unit_nanos", "1000000000"},
          {"position_units", ratio("2")},
          {"value_per_price_unit", ratio("5")},
          {"cost_per_outcome", ratio("1")},
          {"return_basis", ratio("100")},
          {"pnl_unit", "caller-unit"}};
}
J choice(std::uint64_t i) {
  return {{"signal_id", signal(i).at("signal_id")},
          {"transform", transform()},
          {"mode", "execution_mixture"},
          {"nonexecution_pnl", ratio("-2")}};
}
J studies() {
  J result = J::array();
  for (const auto *id : {"weighted_return_sum", "return_moments"})
    result.push_back(
        {{"id", id}, {"version", "1"}, {"parameters", J::object()}});
  result.push_back(
      {{"id", "return_quantiles"},
       {"version", "1"},
       {"parameters",
        {{"levels", J::array({ratio("0"), ratio("1", "2"), ratio("1")})}}}});
  return result;
}
l::Value section(l::Value data) {
  return l::Value(d::section(nullptr)).with("data", std::move(data));
}
store::Receipt write_body(const std::string &path, const l::Value &body) {
  store::ResultWriter writer(path, {16384, 4});
  body.write(writer);
  return writer.finish();
}
J reference(const store::Receipt &receipt, const std::string &pointer = "") {
  const auto &r = receipt.reference;
  return {
      {"kind", "bundle"},
      {"reference",
       {{"manifest_path", r.manifest_path},
        {"manifest_sha256", r.manifest_sha256},
        {"content_sha256", r.content_sha256}}},
      {"selector", {{"kind", "pointer"}, {"pointer", pointer}}},
      {"read_options", {{"max_page_bytes", nullptr}, {"cache_bytes", "0"}}}};
}
store::Reference result_reference(const J &receipt) {
  const auto &r = receipt.at("storage").at("reference");
  return {r.at("manifest_path"), r.at("manifest_sha256"),
          r.at("content_sha256")};
}
// Source is produced row by row. No aggregate signal/model/input Json is built,
// including for the >4096 case. Models intentionally reverse signal order.
l::Value fixture(const std::string &path, std::uint64_t n) {
  std::filesystem::create_directory(path);
  store::RowSpool signals(path + "/signals", {16384, 4});
  store::RowSpool models(path + "/models", {16384, 4});
  store::RowSpool inputs(path + "/inputs", {16384, 4});
  const auto selection = model_selection();
  s::AdmittedModel model(selection, {"signal-0"});
  const auto base_model = model.evaluate(
      {"signal-0", 0, 1000, 1010, 100000000000, true, true, {}}, end);
  for (std::uint64_t i = 0; i < n; ++i) {
    signals.append(signal(i));
    inputs.append(supplied(i));
    auto outcome = base_model;
    outcome["signal_id"] = signal(n - i - 1).at("signal_id");
    models.append(outcome);
  }
  const auto signal_rows = l::Value(signals.close().rows);
  const auto input_rows = l::Value(inputs.close().rows);
  const auto model_rows = l::Value(models.close().rows);
  auto declaration = l::Value(J{{"protocol", "symphony.sbv.external-census.v1"},
                                {"source_sha256", std::string(64, 'a')},
                                {"mode", "causal_declared"},
                                {"producer", producer}})
                         .with("signals", signal_rows);
  const auto census_sha = declaration.sha256();
  auto selected_model = l::Value(selection).with(
      "parameters",
      l::Value(selection.at("parameters")).with("outcomes", input_rows));
  auto choices =
      l::Value(J{{"protocol", "symphony.sbv.evaluate-input.v1"},
                 {"source_sha256", std::string(64, 'a')},
                 {"dataset", "MECHANICAL"},
                 {"extensions", {{"unknown", J::array({"must", "survive"})}}}})
          .with("census", declaration)
          .with("model", selected_model);
  auto body =
      l::Value(d::base("mechanical economic fixture; not market evidence"));
  auto sections =
      body.at("sections")
          .with("signals", section(signal_rows))
          .with("execution", section(model_rows))
          .with("choices", section(choices))
          .with("census_reference",
                l::Value(d::section(nullptr, "not_selected",
                                    "inline external declaration")))
          .with("summary",
                l::Value(d::section({{"census_sha256", census_sha},
                                     {"closed_census", true},
                                     {"signal_count", d::dec(n)},
                                     {"causality", "causal_declared"}})))
          .with("provenance",
                l::Value(d::section({{"source_sha256", std::string(64, 'a')},
                                     {"dataset", "MECHANICAL"},
                                     {"instrument_id", "1"},
                                     {"census_producer", producer},
                                     {"census_sha256", census_sha}})))
          .with("replay",
                l::Value(d::section(
                    {{"synthetic", true},
                     {"events",
                      J::array({{{"ts_recv", "9007199254740993"}}})}})));
  body = body.with("sections", sections);
  return body.with(
      "sections",
      sections.with("census", section(d::stream_census_evidence(body, end))));
}
J output(const std::string &root) {
  const auto prefix = root + "/operation-" + std::to_string(++serial);
  return {{"kind", "partitioned"},
          {"workspace_path", prefix + "-work"},
          {"bundle_path", prefix + "-result"},
          {"write_options", {{"page_bytes", "16384"}, {"index_fanout", "4"}}}};
}
J request(const std::string &root, const J &source, J selections) {
  return {{"protocol", "symphony.sbv.economics-input.v1"},
          {"source", source},
          {"output", output(root)},
          {"selections", std::move(selections)},
          {"studies", studies()},
          {"on_incompatible", "unavailable"},
          {"extensions", {{"caller", "exact choices"}}}};
}
J compact(J selection) {
  auto economics = choice(0);
  economics.erase("signal_id");
  return {{"kind", "apply"}, {"signals", selection}, {"economics", economics}};
}
J run(J p) {
  auto receipt = d::economics(p, end);
  if (receipt.at("status") == "recovery_required")
    throw std::runtime_error("unexpected recovery " + receipt.dump());
  check(receipt.at("protocol") == "symphony.sbv.economics.v1");
  store::ResultReader reader(result_reference(receipt));
  check(reader.verify_closure().at("verification_extent") ==
        "full_logical_closure");
  auto result = reader.select("").read_value();
  check(d::economic_stream_census(l::Value(reader.select("")), end)
            .at("signals")
            .size() == 3);
  check(result.at("content_sha256") == receipt.at("content_sha256"));
  std::filesystem::remove_all(
      p.at("output").at("workspace_path").get<std::string>());
  store::ResultReader portable(result_reference(receipt));
  check(portable.verify_closure().at("reference").at("content_sha256") ==
        result.at("content_sha256"));
  return result;
}
void rejects(J p) {
  const auto receipt = d::economics(p, end);
  check(receipt.at("status") == "recovery_required" &&
        receipt.at("code") == "sbv.partitioned_result_incomplete");
  check(!std::filesystem::exists(
      p.at("output").at("bundle_path").get<std::string>() + "/manifest.json"));
}
int main(int argc, char **argv) try {
  check(argc == 1 || (argc == 2 && std::string(argv[1]) == "--small-only"));
  const bool large_executed = argc == 1;
#ifdef __APPLE__
  char pattern[] = "/private/tmp/sbv-economics-partitioned-XXXXXX";
#else
  char pattern[] = "/tmp/sbv-economics-partitioned-XXXXXX";
#endif
  auto made = ::mkdtemp(pattern);
  check(made);
  const std::string root = made;
  auto small = fixture(root + "/small-spools", 3);
  const auto small_receipt = write_body(root + "/small-source", small);
  const auto bundle_source = reference(small_receipt);
  auto json_source = s::seal_result(small.materialize());
  const auto source_path = root + "/source.json";
  {
    std::ofstream file(source_path);
    file << json_source.dump();
  }
  const J legacy_source{{"path", source_path},
                        {"expected_sha256", json_source.at("content_sha256")},
                        {"pointer", ""}};
  auto choices = J::array({choice(2), choice(0)});
  choices[1]["transform"]["position_units"] = ratio("-3");
  choices[1]["transform"]["cost_per_outcome"] = ratio("2");
  choices[1]["transform"]["return_basis"] = ratio("-20");
  choices[1]["transform"]["pnl_unit"] = "different-caller-unit";
  choices[1]["mode"] = "support_only";
  choices[1]["nonexecution_pnl"] = nullptr;
  auto old_request = request(root, legacy_source, choices);
  old_request.erase("output");
  old_request["output_path"] = root + "/legacy-economics.json";
  d::economics(old_request, end);
  J legacy;
  {
    std::ifstream file(old_request.at("output_path").get<std::string>());
    file >> legacy;
  }
  auto inline_request = request(root, legacy_source, choices);
  const auto inline_result = run(inline_request);
  for (const auto *name :
       {"summary", "economics", "studies", "signals", "execution", "replay",
        "census", "source_context", "choices", "provenance", "diagnostics",
        "user_extensions"})
    check(inline_result.at("sections").at(name) ==
          legacy.at("sections").at(name));
  const auto bundle_result = run(request(root, bundle_source, choices));
  for (const auto *name : {"summary", "economics", "studies", "signals",
                           "execution", "replay", "census"})
    check(bundle_result.at("sections").at(name) ==
          legacy.at("sections").at(name));
  check(bundle_result.at("sections")
            .at("provenance")
            .at("data")
            .contains("source_reference"));
  check(!bundle_result.at("sections")
             .at("provenance")
             .at("data")
             .contains("source_file_sha256"));
  const auto all =
      run(request(root, bundle_source, compact({{"kind", "all"}})));
  check(all.at("sections").at("selections").at("data") ==
        J::array({choice(0), choice(1), choice(2)}));
  check(all.at("sections").at("summary").at("data").at("selection_sha256") ==
        e::sha256_hex(J::array({choice(0), choice(1), choice(2)}).dump()));
  auto reject_context = [&](J forged) {
    bool refused = false;
    try {
      d::economic_stream_census(l::Value(s::seal_result(forged)), end);
    } catch (const std::exception &) {
      refused = true;
    }
    check(refused);
  };
  auto historical = inline_result;
  historical["sections"]["choices"]["data"].erase("source");
  historical["sections"]["choices"]["data"]["path"] = "fixture://evaluation";
  historical["sections"]["choices"]["data"]["expected_sha256"] =
      json_source.at("content_sha256");
  historical["sections"]["summary"]["data"].erase("selection_sha256");
  historical["sections"]["source_context"]["data"].erase("reference");
  historical["sections"]["provenance"]["data"].erase(
      "source_outer_content_sha256");
  historical["sections"]["provenance"]["data"].erase("source_pointer");
  historical["sections"]["provenance"]["data"]["source_path"] =
      "fixture://evaluation";
  for (auto &model : historical["sections"]["execution"]["data"])
    for (const auto *field :
         {"producer", "evidence_origin", "calibration_reference"})
      model.erase(field);
  check(d::economic_stream_census(l::Value(s::seal_result(historical)), end)
            .at("signals")
            .size() == 3);
  historical["sections"]["execution"]["data"][0]["evidence_origin"] = "forged";
  reject_context(historical);
  auto broken_context = all;
  broken_context["sections"]["source_context"]["data"]["reference"]
                ["selected_content_sha256"] = std::string(64, 'b');
  reject_context(broken_context);
  broken_context = all;
  std::swap(broken_context["sections"]["selections"]["data"][0],
            broken_context["sections"]["selections"]["data"][2]);
  broken_context["sections"]["summary"]["data"]["selection_sha256"] =
      e::sha256_hex(broken_context["sections"]["selections"]["data"].dump());
  reject_context(broken_context);
  broken_context = all;
  broken_context["sections"]["selections"]["data"][0]["transform"]
                ["cost_per_outcome"] = ratio("9");
  broken_context["sections"]["summary"]["data"]["selection_sha256"] =
      e::sha256_hex(broken_context["sections"]["selections"]["data"].dump());
  reject_context(broken_context);
  broken_context = all;
  broken_context["sections"]["execution"]["data"][0]["signal_id"] = "signal-1";
  reject_context(broken_context);
  broken_context = all;
  broken_context["sections"]["signals"]["data"][0]["context_reference"] =
      "user://forged";
  reject_context(broken_context);
  broken_context = all;
  broken_context["sections"]["source_context"]["data"]["choices"]["data"]
                ["model"]["parameters"]["outcomes"][0]["signal_id"] =
                    "signal-1";
  reject_context(broken_context);
  const auto range = run(request(
      root, bundle_source,
      compact({{"kind", "range"}, {"first", "1"}, {"count", nullptr}})));
  check(range.at("sections").at("selections").at("data") ==
        J::array({choice(1), choice(2)}));
  const auto empty = run(
      request(root, bundle_source,
              compact({{"kind", "range"}, {"first", "3"}, {"count", "0"}})));
  check(empty.at("sections").at("economics").at("data").empty());
  check(empty.at("sections").at("summary").at("data").at("selection_sha256") ==
        e::sha256_hex("[]"));
  for (const auto *order : {"supplied", "census"}) {
    const auto result =
        run(request(root, bundle_source,
                    compact({{"kind", "ids"},
                             {"ids", J::array({"signal-2", "signal-0"})},
                             {"order", order}})));
    check(result.at("sections").at("selections").at("data") ==
          (std::string(order) == "supplied"
               ? J::array({choice(2), choice(0)})
               : J::array({choice(0), choice(2)})));
  }
  auto input_body = l::Value(d::base("selection inputs"));
  input_body = input_body.with(
      "sections",
      input_body.at("sections")
          .with("ids", l::Value(d::section(J::array({"signal-2", "signal-0"}))))
          .with("rows", l::Value(d::section(choices))));
  auto input_receipt = write_body(root + "/selection-source", input_body);
  auto reference_ids =
      compact({{"kind", "referenced_ids"},
               {"source", reference(input_receipt, "/sections/ids/data")},
               {"order", "census"}});
  const auto from_ids = run(request(root, bundle_source, reference_ids));
  check(from_ids.at("sections").at("selection_input").at("data").at("value") ==
        J::array({"signal-2", "signal-0"}));
  broken_context = from_ids;
  broken_context["sections"]["selection_input"]["data"]["reference"]
                ["selected_value_sha256"] = std::string(64, 'b');
  reject_context(broken_context);
  check(from_ids.at("sections").at("selections").at("data") ==
        J::array({choice(0), choice(2)}));
  const auto from_rows = run(
      request(root, bundle_source,
              {{"kind", "referenced_rows"},
               {"source", reference(input_receipt, "/sections/rows/data")}}));
  check(from_rows.at("sections").at("selections").at("data") == choices);
  check(from_rows.at("sections").at("economics") ==
        legacy.at("sections").at("economics"));
  check(
      from_rows.at("sections").at("source_context").at("data").at("choices") ==
      json_source.at("sections").at("choices"));
  for (const auto &bad :
       {compact({{"kind", "range"}, {"first", "4"}, {"count", "0"}}),
        compact({{"kind", "range"}, {"first", "1"}, {"count", "3"}}),
        compact({{"kind", "range"},
                 {"first", "0"},
                 {"count", "18446744073709551615"}}),
        compact({{"kind", "ids"},
                 {"ids", J::array({"signal-0", "signal-0"})},
                 {"order", "supplied"}}),
        compact({{"kind", "ids"},
                 {"ids", J::array({"missing"})},
                 {"order", "census"}}),
        J::array({choice(0), choice(0)})})
    rejects(request(root, bundle_source, bad));
  // Every source model is admitted, including unselected rows and the distant
  // row in an unordered join. Resealing does not excuse domain/lineage
  // errors.
  auto forged = json_source;
  forged["sections"]["execution"]["data"][0]["support"][0]["weight"] =
      ratio("2");
  forged.erase("content_sha256");
  const auto forged_receipt =
      write_body(root + "/forged-model", l::Value(forged));
  rejects(request(root, reference(forged_receipt), J::array({choice(0)})));
  for (const auto *case_name :
       {"support", "fill", "reference", "unavailable"}) {
    auto contradict = [&](J &model) {
      if (std::string(case_name) == "support") {
        model["support"][0]["weight"] = ratio("1", "3");
        model["support"][1]["weight"] = ratio("2", "3");
      } else if (std::string(case_name) == "fill") {
        model["fill_probability"]["value"] = ratio("1", "2");
        model["no_fill_probability"]["value"] = ratio("1", "2");
      } else if (std::string(case_name) == "reference")
        model["evidence_reference"] = "contradictory attribution";
      else {
        model["fill_probability"] = d::missing("new unsupported absence");
        model["no_fill_probability"] = d::missing("new unsupported absence");
      }
    };
    forged = json_source;
    contradict(forged["sections"]["execution"]["data"][0]);
    forged.erase("content_sha256");
    const auto contradiction =
        write_body(root + "/contradiction-" + case_name, l::Value(forged));
    rejects(request(root, reference(contradiction), J::array({choice(0)})));
    auto retained = all;
    contradict(retained["sections"]["execution"]["data"][0]);
    reject_context(retained);
  }
  forged = json_source;
  forged["sections"]["execution"]["data"][0]["model_version"] = "wrong";
  forged.erase("content_sha256");
  const auto attribution =
      write_body(root + "/forged-attribution", l::Value(forged));
  rejects(request(root, reference(attribution), J::array({choice(0)})));
  forged = json_source;
  forged["sections"]["model_inputs"] = d::section(
      {{"outcomes", J::array({supplied(0)})}, {"reference", nullptr}});
  forged.erase("content_sha256");
  const auto input_mismatch =
      write_body(root + "/forged-inputs", l::Value(forged));
  rejects(request(root, reference(input_mismatch), J::array({choice(0)})));
  auto wrong_outer = bundle_source;
  wrong_outer["reference"]["content_sha256"] = std::string(64, 'b');
  rejects(request(root, wrong_outer, choices));
  auto wrong_role = reference(input_receipt, "/sections/ids/data");
  rejects(request(root, wrong_role, choices));
  // Referenced external inputs are admitted from the retained array/reference,
  // never by reopening the original outcomes source after its deletion.
  auto outcome_body = l::Value(d::base("external outcome input source"));
  const auto original_outcomes = json_source.at("sections")
                                     .at("choices")
                                     .at("data")
                                     .at("model")
                                     .at("parameters")
                                     .at("outcomes");
  outcome_body = outcome_body.with(
      "sections", outcome_body.at("sections")
                      .with("inputs", l::Value(d::section(original_outcomes))));
  const auto outcome_receipt =
      write_body(root + "/original-outcomes", outcome_body);
  const auto outcome_choice =
      reference(outcome_receipt, "/sections/inputs/data");
  const auto outcome_admission = d::select_bundle_value(outcome_choice, end);
  auto archived_evaluation = json_source;
  archived_evaluation.erase("content_sha256");
  auto &archived_parameters =
      archived_evaluation["sections"]["choices"]["data"]["model"]["parameters"];
  archived_parameters.erase("outcomes");
  archived_parameters["outcomes_source"] = outcome_choice;
  archived_evaluation["sections"]["model_inputs"] =
      d::section({{"outcomes", original_outcomes},
                  {"reference", outcome_admission.reference}});
  const auto archive_receipt =
      write_body(root + "/archived-evaluation", l::Value(archived_evaluation));
  std::filesystem::remove_all(root + "/original-outcomes");
  const auto from_archived_inputs =
      run(request(root, reference(archive_receipt), choices));
  check(from_archived_inputs.at("sections")
            .at("source_context")
            .at("data")
            .at("model_inputs")
            .at("outcomes") == original_outcomes);
  std::filesystem::remove_all(root + "/archived-evaluation");
  check(d::economic_stream_census(l::Value(from_archived_inputs), end)
            .at("signals")
            .size() == 3);
  broken_context = from_archived_inputs;
  broken_context["sections"]["source_context"]["data"]["model_inputs"]
                ["reference"]["selected_value_sha256"] = std::string(64, 'b');
  reject_context(broken_context);
  // Retained inline source arrays and replay remain usable after the ancestor
  // source, its input spools and the request's original file are removed.
  const auto portable_request = request(root, bundle_source, choices);
  const auto portable_receipt = d::economics(portable_request, end);
  check(portable_receipt.at("status") == "completed");
  std::filesystem::remove_all(root + "/small-source");
  std::filesystem::remove_all(root + "/small-spools");
  std::filesystem::remove(source_path);
  std::filesystem::remove_all(
      portable_request.at("output").at("workspace_path").get<std::string>());
  store::ResultReader portable(result_reference(portable_receipt));
  check(portable.verify_closure().at("verification_extent") ==
        "full_logical_closure");
  check(portable.select("/sections/source_context/data/choices").read_value() ==
        json_source.at("sections").at("choices"));
  check(d::economic_stream_census(l::Value(portable.select("")), end)
            .at("signals")
            .size() == 3);
  check(portable.select("/sections/replay").read_value() ==
        json_source.at("sections").at("replay"));
  // >4096 real economics rows, streamed from closed census/model arrays. This
  // exercises the operation, not merely a large storage-only wrapper.
  std::cout << "phase small_cases_complete" << std::endl;
  if (large_executed) {
    std::cout << "phase large_fixture" << std::endl;
    const auto large = fixture(root + "/large-spools", 4097);
    std::cout << "phase large_source_retention" << std::endl;
    const auto large_source = write_body(root + "/large-source", large);
    auto large_request =
        request(root, reference(large_source), compact({{"kind", "all"}}));
    large_request["studies"] = J::array();
    std::cout << "phase large_economics" << std::endl;
    const auto large_receipt = d::economics(large_request, end);
    check(large_receipt.at("status") == "completed");
    check(large_receipt.at("summary").at("data").at("selected_signal_count") ==
          "4097");
    std::cout << "phase large_result_validation" << std::endl;
    store::ResultReader large_result(result_reference(large_receipt));
    check(large_result.verify_closure().at("verification_extent") ==
          "full_logical_closure");
    check(d::economic_stream_census(l::Value(large_result.select("")), end)
              .at("signals")
              .size() == 4097);
    const auto economic_rows = large_result.select("/sections/economics/data");
    check(economic_rows.describe().at("children") == "4097");
    const auto last = economic_rows.child(std::uint64_t{4096}).read_value();
    check(last.at("signal_id") == "signal-4096" &&
          last.at("atoms").size() == 3);
    check(last.at("atoms")[0].at("weight") == ratio("3", "10"));
    check(last.at("atoms")[2].at("weight") == ratio("2", "5"));
  }
  std::cout << "large_executed " << (large_executed ? "true" : "false") << '\n';
  std::cout << "checks " << checks << "\nretained_root " << root << '\n';
} catch (const std::exception &ex) {
  std::cerr << ex.what() << '\n';
  return 1;
}
