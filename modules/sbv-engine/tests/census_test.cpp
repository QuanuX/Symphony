#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include "../src/dataset.hpp"
#include "../src/detail.hpp"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <unistd.h>

namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
using J = s::Json;
unsigned checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("census assertion line " +
                             std::to_string(at.line()));
}
const J &data(const J &r, const char *name) {
  return r.at("sections").at(name).at("data");
}
J read(const std::string &path) {
  std::ifstream f(path);
  J r;
  f >> r;
  return r;
}
void write(const std::string &path, const std::string &bytes) {
  std::ofstream f(path, std::ios::binary);
  f << bytes;
  check(f.good());
}
J rat(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
struct Artifact {
  std::string path;
  J result;
  J ref(const std::string &pointer = "") const {
    return {{"path", path},
            {"expected_sha256", result.at("content_sha256")},
            {"pointer", pointer}};
  }
};

int main(int argc, char **argv) try {
  check(argc == 2);
  const auto templates = read(argv[1]).at("templates");
  char temp[] = "/private/tmp/sbv-census-XXXXXX";
  check(::mkdtemp(temp));
  const std::string root = temp;
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  unsigned serial = 0;
  auto fresh = [&](const char *label) {
    return root + "/" + label + "-" + std::to_string(++serial) + ".json";
  };
  auto store = [&](J r) {
    const auto path = fresh("source");
    r = s::seal_result(std::move(r));
    write(path, r.dump());
    return Artifact{path, std::move(r)};
  };
  auto call = [&](const char *operation, J request) {
    const auto path = fresh(operation);
    request["output_path"] = path;
    const auto receipt = s::dispatch(operation, request, e::no_deadline);
    auto result = read(path);
    s::validate_result(result);
    check(result.at("content_sha256") == receipt.at("content_sha256"));
    return Artifact{path, std::move(result)};
  };
  auto reject = [&](const char *operation, J request) {
    const auto path = fresh("rejected");
    request["output_path"] = path;
    bool failed = false;
    try {
      s::dispatch(operation, request, e::no_deadline);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(path));
  };

  // Public DBN bytes provide the wire format; the prices and reset/order
  // sequence below are deliberately synthetic, not calibrated market evidence.
  const auto fixture = public_fixture::fixture();
  std::vector<unsigned char> bytes(fixture.begin(), fixture.begin() + 360);
  const std::uint64_t start = 1609160400000000000ULL;
  auto put = [&](std::size_t offset, std::uint64_t value, unsigned size) {
    for (unsigned i = 0; i < size; ++i)
      bytes[offset + i] = static_cast<unsigned char>(value >> (i * 8));
  };
  std::size_t event_count = 0;
  auto event = [&](char action, std::int64_t price, std::uint32_t size,
                   char side, std::uint64_t id) {
    const auto offset = bytes.size();
    bytes.insert(bytes.end(), fixture.begin() + 360, fixture.begin() + 416);
    put(offset + 8, start + event_count * 100, 8);
    put(offset + 40, start + event_count * 100, 8);
    put(offset + 16, id, 8);
    put(offset + 24, static_cast<std::uint64_t>(price), 8);
    put(offset + 32, size, 4);
    put(offset + 52, event_count, 4);
    bytes[offset + 36] = 128;
    bytes[offset + 37] = 0;
    bytes[offset + 38] = action;
    bytes[offset + 39] = side;
    ++event_count;
  };
  event('R', 0, 0, 'N', 0);
  event('A', 99000000000LL, 20, 'B', 1);
  event('A', 101000000000LL, 20, 'A', 2);
  for (auto price : {100, 95, 105, 96, 104, 97, 103, 98, 102, 99, 101, 100})
    event('T', price * 1000000000LL, 1, 'B', 0);
  const std::string source(reinterpret_cast<const char *>(bytes.data()),
                           bytes.size());
  const auto source_path = root + "/fixture.dbn";
  write(source_path, source);
  const auto source_sha = e::sha256_hex(source);
  J run_request{
      {"protocol", "symphony.sbv.run-input.v1"},
      {"source_path", source_path},
      {"source_sha256", source_sha},
      {"dataset", "GLBX.MDP3"},
      {"criteria",
       {{"rule", "spaced_trades"},
        {"spacing_ns", "200"},
        {"min_trade_size", "1"},
        {"direction", "any"},
        {"max_signals", "2"}}},
      {"execution",
       {{"model", "none"},
        {"horizon_ns", "1100"},
        {"price_offsets_nanos", J::array()},
        {"probability_numerator", "0"},
        {"probability_denominator", "1"}}},
      {"replay",
       {{"before_ns", "300"}, {"after_ns", "1100"}, {"retain_events", true}}},
      {"studies", J::array({"signal_summary"})},
      {"workers", "1"},
      {"extensions", {{"private-note", "original census"}}}};
  auto native = call("run", run_request);
  const auto original_signals = data(native.result, "signals");
  const auto native_evidence = data(native.result, "census");
  const auto original_digest = e::sha256_hex(original_signals.dump());
  check(original_signals.size() == 2);
  check(original_signals[0].at("source_ordinal") == "3");
  check(original_signals[1].at("source_ordinal") == "5");
  // The preceding trade was not itself a selected signal.
  check(original_signals[1].at("previous_trade_price_nanos") == "95000000000");
  check(native_evidence.at("protocol") == "symphony.sbv.census-evidence.v1");
  check(native_evidence.at("kind") == "native");
  check(native_evidence.at("identity_domain") == "native_signals");
  check(native_evidence.at("mode") == "native_causal_prefix");
  check(native_evidence.at("producer").is_null());
  check(native_evidence.at("source_sha256") == source_sha);
  check(native_evidence.at("instrument_id") == "5482");
  check(native_evidence.at("signals") == original_signals);
  check(native_evidence.at("census_sha256") == original_digest);
  check(native_evidence.at("declaration").at("criteria") ==
        run_request.at("criteria"));
  check(native_evidence.at("declaration").at("selection_cap") == "2");
  check(native_evidence.at("declaration").at("additional_eligible_after_cap") ==
        data(native.result, "summary").at("additional_eligible_after_cap"));
  check(data(native.result, "summary").at("additional_eligible_after_cap") !=
        "0");

  J producer{{"id", "independent-user-model"},
             {"version", "1"},
             {"artifact_sha256", ""},
             {"reproducibility", "uncaptured"}};
  J evaluation{{"protocol", "symphony.sbv.evaluate-input.v1"},
               {"source_path", source_path},
               {"source_sha256", source_sha},
               {"dataset", "GLBX.MDP3"},
               {"census", native.ref()},
               {"model",
                {{"protocol", "symphony.sbv.model-selection.v1"},
                 {"id", "observed_trade_levels"},
                 {"version", "1"},
                 {"horizon_ns", "1100"},
                 {"parameters",
                  {{"levels_per_side", "1"},
                   {"include_anchor", false},
                   {"thin_support", "unavailable"}}}}},
               {"replay", run_request.at("replay")},
               {"studies", J::array({"signal_summary", "model_summary"})},
               {"workers", "1"},
               {"extensions", J::object()}};
  auto alternative = [&](J request, const J &signals) {
    J outcomes = J::array();
    for (const auto &signal : signals) {
      const auto anchor =
          std::stoll(signal.at("anchor_price_nanos").get<std::string>());
      outcomes.push_back(
          {{"signal_id", signal.at("signal_id")},
           {"evidence_reference", "user://deliberate-scenario"},
           {"execution_probability",
            {{"status", "supplied"}, {"value", rat("3", "5")}}},
           {"support",
            J::array({{{"price_nanos", std::to_string(anchor - 1000000000LL)},
                       {"weight", rat("1", "2")}},
                      {{"price_nanos", std::to_string(anchor + 1000000000LL)},
                       {"weight", rat("1", "2")}}})}});
    }
    request["model"]["id"] = "external_outcomes";
    request["model"]["parameters"] = {{"producer", producer},
                                      {"measure", "probability"},
                                      {"conditioning", "execution"},
                                      {"calibration_reference", ""},
                                      {"outcomes", outcomes}};
    return request;
  };
  auto first = call("evaluate", evaluation);
  auto model_b = alternative(evaluation, original_signals);
  auto second = call("evaluate", model_b);
  for (const auto *r : {&first.result, &second.result}) {
    check(data(*r, "signals") == original_signals);
    check(data(*r, "census") == native_evidence);
    check(data(*r, "summary").at("census_sha256") == original_digest);
    check(data(*r, "summary").at("causality") == "native_causal_prefix");
    check(data(*r, "choices").at("census") == native.ref());
    const auto &ref = data(*r, "census_reference");
    check(ref.at("path") == native.path);
    check(ref.at("expected_sha256") == native.result.at("content_sha256"));
    check(ref.at("selected_content_sha256") ==
          native.result.at("content_sha256"));
    check(ref.at("pointer") == "");
    check(ref.at("authorship") == "not_verified");
    check(ref.at("file_sha256") ==
          e::sha256_hex(d::read_file(native.path, e::no_deadline)));
  }
  check(data(first.result, "execution") != data(second.result, "execution"));
  auto q = evaluation;
  q["workers"] = "4";
  auto threaded = call("evaluate", q);
  for (const auto *name :
       {"signals", "census", "execution", "replay", "studies", "summary"})
    check(data(threaded.result, name) == data(first.result, name));
  // A source path is a locator, not its content identity.
  write(root + "/relocated.dbn", source);
  q = evaluation;
  q["source_path"] = root + "/relocated.dbn";
  auto relocated = call("evaluate", q);
  check(data(relocated.result, "census") == native_evidence);
  check(data(relocated.result, "execution") == data(first.result, "execution"));
  q = evaluation;
  q["output_path"] = fresh("resident");
  auto dataset = d::load_dataset(q, e::no_deadline);
  d::evaluate(q, e::no_deadline, dataset.get());
  auto resident = read(q.at("output_path").get<std::string>());
  for (const auto *name :
       {"signals", "census", "execution", "replay", "studies", "summary"})
    check(data(resident, name) == data(first.result, name));
  check(data(resident, "resources").at("dataset_feed").at("mode") ==
        "resident");

  // Re-evaluation is self-contained even if an earlier ancestor was removed.
  auto disposable = store(native.result);
  q = evaluation;
  q["census"] = disposable.ref();
  auto descendant = call("evaluate", q);
  std::filesystem::remove(disposable.path);
  q["census"] = descendant.ref();
  auto descendant_again = call("evaluate", q);
  check(data(descendant_again.result, "census") == native_evidence);
  check(data(descendant_again.result, "signals") == original_signals);

  // The selected result must be complete; wrapper and inner hashes both bind.
  auto outer = d::base("container for a retained complete native result");
  outer["sections"]["user_extensions"] =
      d::section({{"nested/a~b", native.result}});
  auto embedded = store(outer);
  q = evaluation;
  const std::string nested_pointer =
      "/sections/user_extensions/data/nested~1a~0b";
  q["census"] = embedded.ref(nested_pointer);
  auto nested = call("evaluate", q);
  check(data(nested.result, "census") == native_evidence);
  check(data(nested.result, "census_reference").at("expected_sha256") ==
        embedded.result.at("content_sha256"));
  check(data(nested.result, "census_reference").at("selected_content_sha256") ==
        native.result.at("content_sha256"));
  check(data(nested.result, "census_reference").at("pointer") ==
        nested_pointer);

  J imported_signals = J::array();
  for (std::size_t i = 0; i < 2; ++i) {
    auto signal = original_signals[0];
    signal.erase("previous_trade_price_nanos");
    signal["signal_id"] = "intent/" + std::to_string(i) + "~";
    signal["context_reference"] = "user://unchanged-private-context";
    signal["anchor_price_nanos"] = "123000000000";
    signal["causal_end_ordinal_exclusive"] = std::to_string(event_count);
    imported_signals.push_back(signal);
  }
  J declaration{{"protocol", "symphony.sbv.external-census.v1"},
                {"source_sha256", source_sha},
                {"mode", "retrospective"},
                {"producer", producer},
                {"signals", imported_signals}};
  auto imported_request = alternative(evaluation, imported_signals);
  imported_request["census"] = declaration;
  auto imported = call("evaluate", imported_request);
  const auto imported_evidence = data(imported.result, "census");
  check(imported_evidence.at("kind") == "external");
  check(imported_evidence.at("identity_domain") == "external_declaration");
  check(imported_evidence.at("declaration") == declaration);
  check(imported_evidence.at("producer") == producer);
  check(imported_evidence.at("mode") == "retrospective");
  check(imported_evidence.at("census_sha256") ==
        e::sha256_hex(declaration.dump()));
  q = imported_request;
  q["census"] = imported.ref();
  auto imported_again = call("evaluate", q);
  check(data(imported_again.result, "census") == imported_evidence);
  check(data(imported_again.result, "signals") == imported_signals);
  check(data(imported_again.result, "summary")
            .at("external_causality_verified") == false);
  check(data(imported_again.result, "provenance").at("census_producer") ==
        producer);
  auto empty_run_request = run_request;
  empty_run_request["criteria"]["min_trade_size"] = "2";
  auto empty_native = call("run", empty_run_request);
  q = evaluation;
  q["census"] = empty_native.ref();
  auto empty_evaluation = call("evaluate", q);
  check(data(empty_evaluation.result, "signals").empty());
  check(data(empty_evaluation.result, "census") ==
        data(empty_native.result, "census"));
  check(data(empty_evaluation.result, "resources").at("actual_workers") == "0");

  // Older actual run/evaluate shapes retain their original digest domains.
  auto old_run = native.result;
  old_run["sections"].erase("census");
  auto legacy_native = store(old_run);
  q = evaluation;
  q["census"] = legacy_native.ref();
  check(data(call("evaluate", q).result, "census") == native_evidence);
  auto old_imported = imported.result;
  old_imported["sections"].erase("census");
  old_imported["sections"].erase("census_reference");
  auto legacy_imported = store(old_imported);
  q = imported_request;
  q["census"] = legacy_imported.ref();
  check(data(call("evaluate", q).result, "census") == imported_evidence);

  auto economic_request = [&](const Artifact &parent, const J &signals) {
    auto p = templates.at("economics");
    p["path"] = parent.path;
    p["expected_sha256"] = parent.result.at("content_sha256");
    p["selections"] = J::array();
    p["studies"] = J::array();
    p["on_incompatible"] = "reject";
    for (const auto &signal : signals)
      p["selections"].push_back(
          {{"signal_id", signal.at("signal_id")},
           {"mode", "execution_mixture"},
           {"nonexecution_pnl", rat("0")},
           {"transform",
            {{"id", "linear_price_pnl"},
             {"version", "1"},
             {"reference_price_nanos", signal.at("anchor_price_nanos")},
             {"price_unit_nanos", "1000000000"},
             {"position_units", rat("1")},
             {"value_per_price_unit", rat("1")},
             {"cost_per_outcome", rat("0")},
             {"return_basis", rat("100")},
             {"pnl_unit", "fixture-dollar"}}}});
    return p;
  };
  auto economic = call("economics", economic_request(second, original_signals));
  check(data(economic.result, "summary").at("source_census_sha256") ==
        original_digest);
  check(data(economic.result, "signals") == original_signals);
  check(economic.result.at("sections").at("replay") ==
        second.result.at("sections").at("replay"));
  for (const auto &row : data(economic.result, "economics")) {
    check(row.at("atoms").size() == 3);
    check(row.at("atoms")[0].at("return") == rat("-1", "100"));
    check(row.at("atoms")[1].at("return") == rat("1", "100"));
    check(row.at("atoms")[0].at("weight") == rat("3", "10"));
    check(row.at("atoms")[2].at("weight") == rat("2", "5"));
  }
  auto imported_economics =
      call("economics", economic_request(imported_again, imported_signals));
  check(data(imported_economics.result, "summary").at("source_census_sha256") ==
        imported_evidence.at("census_sha256"));
  // A raw run's touch/no-model rows are not typed execution-model outcomes.
  reject("economics", economic_request(native, original_signals));

  auto book_request = templates.at("book");
  book_request["source_path"] = source_path;
  book_request["source_sha256"] = source_sha;
  book_request["dataset"] = "GLBX.MDP3";
  book_request["census_result"] = {
      {"path", second.path},
      {"expected_sha256", second.result.at("content_sha256")}};
  book_request["signal_ids"] = J::array();
  for (const auto &signal : original_signals)
    book_request["signal_ids"].push_back(signal.at("signal_id"));
  book_request["initial_state"] = {{"mode", "uninitialized"}};
  book_request["on_anomaly"] = "reject";
  book_request["replay"] = {
      {"before_ns", "500"}, {"after_ns", "100"}, {"retain_events", true}};
  book_request["frames"] = {
      {"cadence", "signals"}, {"levels_per_side", "1"}, {"maximum", "16"}};
  auto book = call("book", book_request);
  check(data(book.result, "summary").at("parent_census_sha256") ==
        original_digest);
  check(data(book.result, "signals") == original_signals);
  check(data(book.result, "book_frames").size() == 2);
  for (const auto &frame : data(book.result, "book_frames"))
    check(frame.at("status") == "available");

  // Invalid source declarations must fail before publication even if a caller
  // recomputes the outer result hash; a hash is not semantic validation.
  auto forged_reject = [&](J malformed) {
    auto bad = store(std::move(malformed));
    auto request = evaluation;
    request["census"] = bad.ref();
    reject("evaluate", request);
  };
  for (const auto &pointer :
       {"/sections/signals/data", "/absent", "/bad~2pointer"}) {
    q = evaluation;
    q["census"]["pointer"] = pointer;
    reject("evaluate", q);
  }
  q = evaluation;
  q["census"]["expected_sha256"] = std::string(64, '0');
  reject("evaluate", q);
  q = evaluation;
  q["census"]["extra"] = true;
  reject("evaluate", q);
  q = evaluation;
  q["census"]["protocol"] = "symphony.sbv.external-census.v1";
  reject("evaluate", q);
  q = evaluation;
  q["census"] = nullptr;
  reject("evaluate", q);
  auto broken_inner = outer;
  broken_inner["sections"]["user_extensions"]["data"]["nested/a~b"]
              ["content_sha256"] = std::string(64, '0');
  auto invalid_wrapper = store(broken_inner);
  q = evaluation;
  q["census"] = invalid_wrapper.ref(nested_pointer);
  reject("evaluate", q);
  for (const auto *pointer : {"/sections/summary/data/census_sha256",
                              "/sections/provenance/data/source_sha256",
                              "/sections/choices/data/source_sha256",
                              "/sections/census/data/source_sha256",
                              "/sections/census/data/census_sha256"}) {
    auto malformed = native.result;
    malformed.at(J::json_pointer(pointer)) = std::string(64, 'a');
    forged_reject(malformed);
  }
  for (const auto *pointer :
       {"/sections/provenance/data/dataset", "/sections/choices/data/dataset",
        "/sections/census/data/dataset"}) {
    auto malformed = native.result;
    malformed.at(J::json_pointer(pointer)) = "different-dataset";
    forged_reject(malformed);
  }
  for (const auto *pointer : {"/sections/provenance/data/instrument_id",
                              "/sections/census/data/instrument_id"}) {
    auto malformed = native.result;
    malformed.at(J::json_pointer(pointer)) = "1";
    forged_reject(malformed);
  }
  auto malformed = native.result;
  malformed["sections"]["summary"]["data"]["closed_census"] = false;
  forged_reject(malformed);
  malformed = native.result;
  malformed["sections"]["summary"]["data"]["signal_count"] = "1";
  forged_reject(malformed);
  malformed = native.result;
  malformed["sections"]["choices"]["data"]["protocol"] =
      "symphony.sbv.fit-input.v1";
  forged_reject(malformed);
  malformed = native.result;
  malformed["sections"]["census"]["data"] = nullptr;
  forged_reject(malformed);
  malformed = native.result;
  malformed["sections"]["census"]["status"] = "unavailable";
  forged_reject(malformed);
  malformed = native.result;
  malformed["sections"]["census"]["data"]["declaration"]["criteria"]
           ["spacing_ns"] = "100";
  forged_reject(malformed);
  for (const auto *field :
       {"available_ns", "causal_end_ordinal_exclusive", "anchor_price_nanos"}) {
    malformed = native.result;
    malformed["sections"]["signals"]["data"][0][field] = "0";
    forged_reject(malformed);
  }
  // Reconcile every redundant signal digest too: actual source coordinates and
  // the native causal-prefix contract must still be checked independently.
  for (const auto *field :
       {"available_ns", "causal_end_ordinal_exclusive", "anchor_price_nanos"}) {
    malformed = native.result;
    auto signals = original_signals;
    signals[0][field] = "0";
    auto digest = e::sha256_hex(signals.dump());
    malformed["sections"]["signals"]["data"] = signals;
    malformed["sections"]["census"]["data"]["signals"] = signals;
    malformed["sections"]["census"]["data"]["census_sha256"] = digest;
    malformed["sections"]["summary"]["data"]["census_sha256"] = digest;
    if (malformed["sections"]["provenance"]["data"].contains("census_sha256"))
      malformed["sections"]["provenance"]["data"]["census_sha256"] = digest;
    forged_reject(malformed);
  }
  malformed = imported.result;
  malformed["sections"]["census"]["data"]["mode"] = "causal_declared";
  forged_reject(malformed);
  malformed = imported.result;
  malformed["sections"]["census"]["data"]["producer"]["id"] = "rewritten";
  forged_reject(malformed);
  malformed = imported.result;
  malformed["sections"]["census"]["data"]["declaration"]["signals"][0]
           ["context_reference"] = "rewritten";
  forged_reject(malformed);
  malformed = second.result;
  malformed["sections"]["census"]["data"]["census_sha256"] =
      std::string(64, '0');
  const auto invalid_evaluation = store(malformed);
  reject("economics", economic_request(invalid_evaluation, original_signals));
  q = book_request;
  q["census_result"] = {
      {"path", invalid_evaluation.path},
      {"expected_sha256", invalid_evaluation.result.at("content_sha256")}};
  reject("book", q);

  // Real graph bindings insert references; two alternative models share one
  // immutable census, and a second invocation reuses all three receipts.
  auto trial = [&](const char *id, const char *op, J request,
                   J dependencies = J::array()) {
    request.erase("output_path");
    J bindings = J::array();
    if (!dependencies.empty()) {
      request["census"] = nullptr;
      bindings.push_back({{"target_pointer", "/census"},
                          {"trial_id", "census"},
                          {"result_pointer", ""}});
    }
    return J{{"id", id},
             {"state", "ready"},
             {"operation", op},
             {"request", request},
             {"reason", ""},
             {"parameters", J::object()},
             {"lineage", J::object()},
             {"depends_on", dependencies},
             {"bindings", bindings}};
  };
  auto plan = templates.at("experiment");
  plan["experiment_id"] = "retained-census-alternatives";
  plan["directory"] = root + "/journal";
  std::filesystem::create_directory(root + "/journal");
  plan["workers"] = "2";
  plan["trials"] =
      J::array({trial("model-a", "evaluate", evaluation, J::array({"census"})),
                trial("model-b", "evaluate", model_b, J::array({"census"})),
                trial("census", "run", run_request)});
  auto graph = call("experiment", plan);
  check(data(graph.result, "summary").at("completed") == "3");
  check(data(graph.result, "search").at("wave_count") == "2");
  const auto graph_a = read(root + "/journal/model-a.result.json");
  const auto graph_b = read(root + "/journal/model-b.result.json");
  check(data(graph_a, "census") == native_evidence);
  check(data(graph_b, "census") == native_evidence);
  check(data(graph_a, "execution") == data(first.result, "execution"));
  check(data(graph_b, "execution") == data(second.result, "execution"));
  auto resumed = call("experiment", plan);
  check(data(resumed.result, "search").at("executed_this_invocation") == "0");
  check(data(resumed.result, "search").at("reused_receipts") == "3");

  if (const auto *directory = std::getenv("SYMPHONY_SBV_CENSUS_EMIT_FIXTURE")) {
    const std::filesystem::path path(directory);
    check(path.is_absolute() && std::filesystem::is_directory(path));
    write((path / "fixture.dbn").string(), source);
    auto emitted_run = run_request;
    emitted_run["source_path"] = (path / "fixture.dbn").string();
    emitted_run["output_path"] = (path / "native.json").string();
    write((path / "run-request.json").string(), emitted_run.dump(2));
    auto emitted_evaluation = model_b;
    emitted_evaluation["source_path"] = (path / "fixture.dbn").string();
    emitted_evaluation["census"] = nullptr;
    emitted_evaluation["output_path"] = (path / "evaluation.json").string();
    write((path / "evaluate-request.json").string(),
          emitted_evaluation.dump(2));
  }
  std::cout << checks << " SBV retained census continuity assertions passed\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
