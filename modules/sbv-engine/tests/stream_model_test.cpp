#include "detail.hpp"
#include "row_spool.hpp"
#include "stream_census.hpp"
#include "stream_model.hpp"
#include <atomic>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <thread>
#include <unistd.h>

namespace s = symphony::sbv;
namespace d = s::detail;
namespace l = s::logical;
namespace store = s::result_store;
namespace e = symphony::knowledge::engine;
using J = s::Json;
std::uint64_t checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("stream model test line " +
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
J ratio(int n, int denominator = 1) {
  return {{"numerator", std::to_string(n)},
          {"denominator", std::to_string(denominator)}};
}
J outcome(std::uint64_t ordinal) {
  return {{"signal_id", "signal-" + std::to_string(ordinal)},
          {"support",
           J::array({{{"price_nanos", "100"}, {"weight", ratio(1, 3)}},
                     {{"price_nanos", "200"}, {"weight", ratio(2, 3)}}})},
          {"execution_probability",
           {{"status", "supplied"}, {"value", ratio(1, 2)}}},
          {"evidence_reference", "user:/é/" + std::to_string(ordinal)}};
}
J signal(std::uint64_t ordinal) {
  return {{"signal_id", "signal-" + std::to_string(ordinal)}};
}
J model(J rows) {
  return {{"protocol", "symphony.sbv.model-selection.v1"},
          {"id", "external_outcomes"},
          {"version", "1"},
          {"horizon_ns", "10"},
          {"parameters",
           {{"producer",
             {{"id", "test"},
              {"version", "1"},
              {"artifact_sha256", ""},
              {"reproducibility", "uncaptured"}}},
            {"measure", "probability"},
            {"conditioning", "execution"},
            {"calibration_reference", "uncalibrated:fixture"},
            {"outcomes", std::move(rows)}}}};
}
J builtin() {
  return {{"protocol", "symphony.sbv.model-selection.v1"},
          {"id", "observed_trade_levels"},
          {"version", "1"},
          {"horizon_ns", "10"},
          {"parameters",
           {{"levels_per_side", "1"},
            {"include_anchor", true},
            {"thin_support", "unavailable"}}}};
}
s::ModelFrame frame(std::uint64_t ordinal) {
  return {
      "signal-" + std::to_string(ordinal), 1, 100, 110, 150, true, true, {}};
}
J reference(const store::Receipt &r, const std::string &pointer) {
  return {{"kind", "bundle"},
          {"reference",
           {{"manifest_path", r.reference.manifest_path},
            {"manifest_sha256", r.reference.manifest_sha256},
            {"content_sha256", r.reference.content_sha256}}},
          {"selector", {{"kind", "pointer"}, {"pointer", pointer}}},
          {"read_options",
           {{"max_page_bytes", nullptr}, {"cache_bytes", "1048576"}}}};
}
store::Receipt write(const std::string &path, const l::Value &body,
                     store::WriteOptions options = {4096, 4}) {
  store::ResultWriter writer(path, options);
  body.write(writer);
  return writer.finish();
}
l::Value section(l::Value data) {
  return l::Value::object({{"status", l::Value("available")},
                           {"reason", l::Value("")},
                           {"data", std::move(data)}});
}

int main() try {
#ifdef __APPLE__
  char pattern[] = "/private/tmp/sbv-stream-model-XXXXXX";
#else
  char pattern[] = "/tmp/sbv-stream-model-XXXXXX";
#endif
  auto made = ::mkdtemp(pattern);
  check(made);
  const std::string root = made;
  const J signals = J::array({signal(0), signal(1), signal(2)});
  const auto selected = model(J::array({outcome(2), outcome(0), outcome(1)}));
  d::StreamModel admitted(selected, l::Value(signals), end);
  s::AdmittedModel legacy(selected, {"signal-0", "signal-1", "signal-2"});
  for (std::uint64_t i = 0; i != 3; ++i)
    check(admitted.evaluate(frame(i), end) == legacy.evaluate(frame(i), end));
  check(admitted.source_reference().is_null());
  check(admitted.outcomes().materialize() ==
        selected.at("parameters").at("outcomes"));
  check(admitted.horizon_ns() == 10);
  check(admitted.evidence().at("retained_outcome_id_index").at("entries") ==
        "3");
  check(admitted.evidence().at("temporary_census_id_index_released") == true);
  rejects([&] { admitted.evaluate(frame(9), end); });
  rejects([&] { admitted.evaluate(frame(0), 0); });
  auto wrong_horizon = frame(0);
  wrong_horizon.horizon_end_ns = 111;
  rejects([&] { admitted.evaluate(wrong_horizon, end); });
  d::StreamModel retained_inline(selected, l::Value(signals),
                                 admitted.outcomes(), nullptr, end);
  check(retained_inline.evaluate(frame(2), end) ==
        legacy.evaluate(frame(2), end));
  auto different = selected.at("parameters").at("outcomes");
  different[0]["evidence_reference"] = "changed";
  rejects([&] {
    d::StreamModel m(selected, l::Value(signals), l::Value(different), nullptr,
                     end);
  });

  // Every malformed row is rejected in construction, before worker evaluation.
  for (const auto &mode :
       {"duplicate", "unknown", "missing", "mass", "negative", "fill", "extra",
        "undefined_price", "duplicate_price", "empty_support", "large_support",
        "nested_evidence"}) {
    auto bad = selected;
    auto &rows = bad["parameters"]["outcomes"];
    if (mode == std::string("duplicate"))
      rows[2] = rows[0];
    if (mode == std::string("unknown"))
      rows[2]["signal_id"] = "unknown";
    if (mode == std::string("missing"))
      rows.erase(2);
    if (mode == std::string("mass"))
      rows[2]["support"][1]["weight"] = ratio(1, 3);
    if (mode == std::string("negative"))
      rows[2]["support"][0]["weight"] = ratio(-1, 3);
    if (mode == std::string("fill"))
      rows[2]["execution_probability"]["value"] = ratio(2);
    if (mode == std::string("extra"))
      rows[2]["unadmitted"] = J::array({"x"});
    if (mode == std::string("undefined_price"))
      rows[2]["support"][0]["price_nanos"] = "9223372036854775807";
    if (mode == std::string("duplicate_price"))
      rows[2]["support"][1]["price_nanos"] = "100";
    if (mode == std::string("empty_support"))
      rows[2]["support"] = J::array();
    if (mode == std::string("large_support"))
      rows[2]["support"] = J(std::vector<J>(65, rows[2]["support"][0]));
    if (mode == std::string("nested_evidence"))
      rows[2]["evidence_reference"] = J::array({"x"});
    rejects([&] { d::StreamModel m(bad, l::Value(signals), end); });
  }
  auto duplicate_signals = signals;
  duplicate_signals[2] = duplicate_signals[0];
  rejects(
      [&] { d::StreamModel m(selected, l::Value(duplicate_signals), end); });
  auto empty_bad = model(J::array());
  empty_bad["parameters"]["measure"] = "renormalize";
  rejects([&] { d::StreamModel m(empty_bad, l::Value(J::array()), end); });
  d::StreamModel empty(model(J::array()), l::Value(J::array()), end);
  check(empty.evidence().at("outcome_rows") == "0");
  auto signed_model = selected;
  signed_model["parameters"]["measure"] = "signed_coefficient";
  signed_model["parameters"]["conditioning"] = "scenario";
  signed_model["parameters"]["outcomes"][0]["support"][0]["weight"] =
      ratio(-1, 3);
  signed_model["parameters"]["outcomes"][0]["execution_probability"] = {
      {"status", "unavailable"}, {"reason", "not modeled"}};
  d::StreamModel signed_admitted(signed_model, l::Value(signals), end);
  check(signed_admitted.evaluate(frame(2), end) ==
        s::AdmittedModel(signed_model, {"signal-0", "signal-1", "signal-2"})
            .evaluate(frame(2), end));

  // More than the old 4096 ID ceiling; arrays remain disk-backed logical views.
  constexpr std::uint64_t count = 5003;
  store::RowSpool signal_spool(root + "/signal-spool", {4096, 4});
  store::RowSpool outcome_spool(root + "/outcome-spool", {4096, 4});
  for (std::uint64_t i = 0; i != count; ++i) {
    signal_spool.append(signal(i));
    outcome_spool.append(outcome(count - 1 - i));
  }
  const auto saved_signals = signal_spool.close(),
             saved_outcomes = outcome_spool.close();
  l::Value large_signals(saved_signals.rows),
      large_outcomes(saved_outcomes.rows);
  auto body = l::Value(d::base("stream model source"));
  auto sections =
      body.at("sections")
          .with("signals", section(large_signals))
          .with("model_inputs", section(large_outcomes))
          .with("unrelated", section(l::Value("UNRELATED-MARKER" +
                                              std::string(50000, 'x'))));
  // The unrelated scalar deliberately exceeds the default small fixture page.
  // Choose its physical page layout explicitly; it must not be silently split.
  auto saved =
      write(root + "/source", body.with("sections", sections), {65536, 4});
  auto bundle_model = model(J::array());
  bundle_model["parameters"].erase("outcomes");
  bundle_model["parameters"]["outcomes_source"] =
      reference(saved, "/sections/model_inputs/data");
  auto both = bundle_model;
  both["parameters"]["outcomes"] = J::array();
  rejects([&] { d::StreamModel m(both, large_signals, end); });
  auto wrong = bundle_model;
  wrong["parameters"]["outcomes_source"]["reference"]["content_sha256"] =
      std::string(64, '0');
  rejects([&] { d::StreamModel m(wrong, large_signals, end); });
  wrong = bundle_model;
  wrong["parameters"]["outcomes_source"]["selector"]["pointer"] = "";
  rejects([&] { d::StreamModel m(wrong, large_signals, end); });

  // Full source closure is verified even when the selected outcome rows are
  // unaffected: remove an unrelated page from a private copied source.
  std::filesystem::copy(root + "/source", root + "/corrupt-source",
                        std::filesystem::copy_options::recursive);
  bool removed = false;
  for (const auto &entry :
       std::filesystem::directory_iterator(root + "/corrupt-source")) {
    if (entry.path().filename() == "manifest.json")
      continue;
    std::ifstream input(entry.path());
    std::string raw((std::istreambuf_iterator<char>(input)), {});
    if (raw.find("UNRELATED-MARKER") != std::string::npos) {
      std::filesystem::remove(entry.path());
      removed = true;
      break;
    }
  }
  check(removed);
  wrong = bundle_model;
  wrong["parameters"]["outcomes_source"]["reference"]["manifest_path"] =
      root + "/corrupt-source/manifest.json";
  rejects([&] { d::StreamModel m(wrong, large_signals, end); });

  J captured_reference;
  store::Receipt descendant;
  {
    d::StreamModel large(bundle_model, large_signals, end);
    check(large.evidence().at("retained_outcome_id_index").at("entries") ==
          "5003");
    check(large.source_reference().at("verification_extent") ==
          "full_outer_logical_closure");
    captured_reference = large.source_reference();
    for (const auto i : {std::uint64_t{0}, count / 2, count - 1})
      check(large.evaluate(frame(i), end) ==
            s::AdmittedModel(model(J::array({outcome(i)})),
                             {"signal-" + std::to_string(i)})
                .evaluate(frame(i), end));
    std::atomic<bool> valid{true};
    std::vector<std::thread> workers;
    for (std::uint64_t worker = 0; worker != 4; ++worker)
      workers.emplace_back([&, worker] {
        try {
          for (std::uint64_t i = worker; i < 200; i += 4)
            if (large.evaluate(frame(i), end) !=
                s::AdmittedModel(model(J::array({outcome(i)})),
                                 {"signal-" + std::to_string(i)})
                    .evaluate(frame(i), end))
              valid = false;
        } catch (...) {
          valid = false;
        }
      });
    for (auto &worker : workers)
      worker.join();
    check(valid);
    auto output = l::Value(d::base("retained model descendant"));
    auto inline_choice = l::Value(model(J::array()));
    inline_choice = inline_choice.with(
        "parameters",
        inline_choice.at("parameters").with("outcomes", large.outcomes()));
    auto children =
        output.at("sections")
            .with("signals", section(large_signals))
            .with("model_inputs",
                  section(l::Value::object(
                      {{"outcomes", large.outcomes()},
                       {"reference", l::Value(captured_reference)}})))
            .with("choices", section(l::Value(bundle_model)))
            .with("choices_inline", section(inline_choice));
    descendant = write(root + "/descendant", output.with("sections", children));
  }
  d::StreamModel observed(builtin(), large_signals, end);
  check(observed.evidence().at("retained_outcome_id_index").is_null());
  check(observed.evidence().at("temporary_census_id_index").is_null());
  check(observed.evaluate(frame(0), end) ==
        s::AdmittedModel(builtin(), {}).evaluate(frame(0), end));
  rejects([&] { observed.outcomes(); });
  std::filesystem::remove_all(root + "/source");
  std::filesystem::remove_all(root + "/signal-spool");
  std::filesystem::remove_all(root + "/outcome-spool");
  auto retained_signals = d::select_bundle_value(
      reference(descendant, "/sections/signals/data"), end);
  auto retained_rows = d::select_bundle_value(
      reference(descendant, "/sections/model_inputs/data/outcomes"), end);
  d::StreamModel portable(bundle_model, retained_signals.value,
                          retained_rows.value, captured_reference, end);
  check(portable.evidence().at("source_admission") ==
        "retained_source_correspondence");
  check(portable.evaluate(frame(count - 1), end) ==
        s::AdmittedModel(model(J::array({outcome(count - 1)})), {"signal-5002"})
            .evaluate(frame(count - 1), end));
  auto retained_choices = d::select_bundle_value(
      reference(descendant, "/sections/choices/data"), end);
  d::StreamModel logical_retained(retained_choices.value,
                                  retained_signals.value, retained_rows.value,
                                  captured_reference, end);
  check(logical_retained.evaluate(frame(0), end) ==
        portable.evaluate(frame(0), end));
  auto retained_inline_choices = d::select_bundle_value(
      reference(descendant, "/sections/choices_inline/data"), end);
  d::StreamModel logical_inline(retained_inline_choices.value,
                                retained_signals.value, retained_rows.value,
                                nullptr, end);
  check(logical_inline.evaluate(frame(0), end) ==
        portable.evaluate(frame(0), end));
  auto unknown_choices =
      retained_inline_choices.value.with("unexpected", retained_rows.value);
  rejects([&] {
    d::StreamModel m(unknown_choices, retained_signals.value,
                     retained_rows.value, nullptr, end);
  });
  for (const auto *field : {"selected_value_sha256", "selected_content_sha256",
                            "authorship", "verification_extent"}) {
    auto forged = captured_reference;
    forged[field] = "forged";
    rejects([&] {
      d::StreamModel m(bundle_model, retained_signals.value,
                       retained_rows.value, forged, end);
    });
  }
  auto root_node = captured_reference;
  root_node["selected_node_id"] = "0";
  rejects([&] {
    d::StreamModel m(bundle_model, retained_signals.value, retained_rows.value,
                     root_node, end);
  });
  auto changed_selection = bundle_model;
  changed_selection["parameters"]["outcomes_source"]["selector"]["pointer"] =
      "/unrelated";
  rejects([&] {
    d::StreamModel m(changed_selection, retained_signals.value,
                     retained_rows.value, captured_reference, end);
  });
  auto by_node = bundle_model;
  by_node["parameters"]["outcomes_source"]["selector"] = {
      {"kind", "node_id"},
      {"node_id", captured_reference.at("selected_node_id")}};
  auto node_ref = captured_reference;
  node_ref["selector"] =
      by_node.at("parameters").at("outcomes_source").at("selector");
  d::StreamModel node_retained(by_node, retained_signals.value,
                               retained_rows.value, node_ref, end);
  check(node_retained.evaluate(frame(0), end) ==
        portable.evaluate(frame(0), end));
  node_ref["selected_node_id"] = "0";
  rejects([&] {
    d::StreamModel m(by_node, retained_signals.value, retained_rows.value,
                     node_ref, end);
  });
  std::cout << "stream model tests: " << checks << " passed; fixture=" << root
            << '\n';
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
