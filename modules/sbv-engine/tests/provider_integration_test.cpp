#include "../../sqav-databento-dbn-cpp/tests/public_fixture.hpp"
#include "../src/census.hpp"
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
    throw std::runtime_error("provider integration assertion line " +
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
J ratio(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
struct Artifact {
  std::string path;
  J result;
  J reference() const {
    return {{"path", path},
            {"expected_sha256", result.at("content_sha256")},
            {"pointer", ""}};
  }
};

int main(int argc, char **argv) try {
  check(argc == 3);
  const auto library = std::filesystem::absolute(argv[1]).string();
  const auto library_sha = e::sha256_hex(d::read_file(library, e::no_deadline));
  const auto templates = read(argv[2]).at("templates");
  char temp[] = "/private/tmp/sbv-provider-integration-XXXXXX";
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
  auto call = [&](const char *op, J p) {
    const auto path = fresh(op);
    p["output_path"] = path;
    const auto receipt = s::dispatch(op, p, e::no_deadline);
    auto result = read(path);
    s::validate_result(result);
    check(result.at("content_sha256") == receipt.at("content_sha256"));
    return Artifact{path, std::move(result)};
  };
  auto reject = [&](const char *op, J p) {
    const auto path = fresh("rejected");
    p["output_path"] = path;
    bool failed = false;
    try {
      s::dispatch(op, p, e::no_deadline);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(!std::filesystem::exists(path));
  };

  struct Event {
    char action, side;
    std::uint64_t order;
    std::int64_t price;
    std::uint32_t size;
    unsigned flags = 128;
  };
  const std::vector<Event> events{
      {'R', 'N', 0, 0, 0},
      {'A', 'B', 1, 99000000000LL, 20},
      {'A', 'A', 2, 102000000000LL, 20},
      {'T', 'B', 91, 100000000123LL, 7},
      {'F', 'B', 91, 100000000123LL, 1},
      {'C', 'B', 1, 99000000000LL, 1},
      {'A', 'B', 3, 98000000000LL, 3},
      {'T', 'A', 92, 101000000456LL, 3},
      {'N', 'N', UINT64_MAX - 19, INT64_MIN + 100, 0, 136}};
  const std::uint64_t start = 1609160400000000000ULL;
  auto fixture_bytes = [&](bool old_version, bool ts_out) {
    const auto original =
        old_version ? public_fixture::fixture_v1() : public_fixture::fixture();
    const std::size_t metadata = old_version ? 206 : 360;
    std::vector<unsigned char> bytes(original.begin(),
                                     original.begin() + metadata);
    bytes[old_version ? 60 : 52] = ts_out ? 1 : 0;
    auto put = [&](std::size_t offset, std::uint64_t value, unsigned count) {
      for (unsigned i = 0; i < count; ++i)
        bytes[offset + i] = static_cast<unsigned char>(value >> (8 * i));
    };
    for (std::size_t i = 0; i < events.size(); ++i) {
      const auto offset = bytes.size();
      bytes.insert(bytes.end(), original.begin() + metadata,
                   original.begin() + metadata + 56);
      if (ts_out)
        bytes.resize(bytes.size() + 8);
      bytes[offset] = ts_out ? 16 : 14;
      put(offset + 2, 1, 2);
      put(offset + 4, 5482, 4);
      put(offset + 8, start + i * 100 - 17, 8);
      put(offset + 16, events[i].order, 8);
      put(offset + 24, static_cast<std::uint64_t>(events[i].price), 8);
      put(offset + 32, events[i].size, 4);
      bytes[offset + 36] = events[i].flags;
      bytes[offset + 37] = 9;
      bytes[offset + 38] = events[i].action;
      bytes[offset + 39] = events[i].side;
      // A same-clock later F event must remain in a zero-duration model window.
      put(offset + 40, start + (i == 4 ? 3 : i) * 100, 8);
      put(offset + 48, static_cast<std::uint32_t>(-53 - static_cast<int>(i)),
          4);
      put(offset + 52, 310 + i * 7, 4);
      if (ts_out)
        put(offset + 56,
            i == events.size() - 1 ? UINT64_MAX : start + i * 100 + 31, 8);
    }
    return std::string(reinterpret_cast<const char *>(bytes.data()),
                       bytes.size());
  };
  const auto raw = fixture_bytes(false, true);
  const auto source_path = root + "/fixture.dbn";
  write(source_path, raw);
  J provider{
      {"protocol", "symphony.sbv.native-provider-selection.v1"},
      {"library", {{"path", library}, {"expected_sha256", library_sha}}},
      {"id", "sbv-test-provider"},
      {"version", "1"},
      {"role", "strategy"},
      {"input_profile", "symphony.sbv.provider-databento-mbo-event.v1"},
      {"concurrency", "serialized_instance"},
      {"parameters",
       {{"action", "T"}, {"emit_every", "1"}, {"emissions_per_event", "2"}}},
      {"dependencies",
       {{"capture", "uncaptured"},
        {"description", "Public independently built fixture; platform runtime "
                        "dependencies declared in descriptor"},
        {"artifacts", J::array()}}},
      {"extensions", {{"user-choice", "retained verbatim"}}}};
  J generate{{"protocol", "symphony.sbv.generate-census-input.v1"},
             {"source_path", source_path},
             {"source_sha256", e::sha256_hex(raw)},
             {"dataset", "GLBX.MDP3"},
             {"provider", provider},
             {"extensions", J::object()}};
  auto generated = call("generate_census", generate);
  const auto signals = data(generated.result, "signals");
  const auto census = data(generated.result, "census");
  check(signals.size() == 4);
  check(signals[0].at("signal_id") == "provider-3-0");
  check(signals[1].at("signal_id") == "provider-3-1");
  check(signals[2].at("signal_id") == "provider-7-0");
  check(signals[3].at("signal_id") == "provider-7-1");
  for (std::size_t i = 0; i < signals.size(); ++i) {
    const auto ordinal = i < 2 ? 3U : 7U;
    check(signals[i].at("source_ordinal") == d::dec(ordinal));
    check(signals[i].at("available_ns") == d::dec(start + ordinal * 100));
    check(signals[i].at("causal_end_ordinal_exclusive") == d::dec(ordinal + 1));
    check(signals[i].at("anchor_price_nanos") == d::dec(events[ordinal].price));
  }
  check(census.at("kind") == "native_provider");
  check(census.at("mode") == "native_provider_prefix");
  check(census.at("identity_domain") == "native_provider_declaration");
  check(census.at("census_sha256") ==
        e::sha256_hex(census.at("declaration").dump()));
  check(census.at("declaration").at("provider") == provider);
  check(data(generated.result, "choices").at("provider") == provider);
  check(data(generated.result, "provider")
            .at("evidence")
            .at("selection_sha256") == e::sha256_hex(provider.dump()));
  check(data(generated.result, "provider")
            .at("evidence")
            .at("loading")
            .at("isolation") == false);
  check(data(generated.result, "summary").at("execution_evaluated") == false);
  check(generated.result.at("sections").at("execution").at("status") ==
        "not_selected");
  check(generated.result.at("sections").at("replay").at("status") ==
        "not_selected");
  const auto &completion = data(generated.result, "provider").at("completion");
  check(completion.at("diagnostics")[0] ==
        "full MBO and invocation prefix checks passed");
  const J expected_last{{"events", "9"},
                        {"last_source_ordinal", "8"},
                        {"last_publisher_id", "1"},
                        {"last_instrument_id", "5482"},
                        {"last_ts_event", d::dec(start + 783)},
                        {"last_order_id", d::dec(UINT64_MAX - 19)},
                        {"last_price_nanos", d::dec(INT64_MIN + 100)},
                        {"last_size", "0"},
                        {"last_flags", "136"},
                        {"last_channel_id", "9"},
                        {"last_action", "78"},
                        {"last_side", "78"},
                        {"last_ts_recv", d::dec(start + 800)},
                        {"last_ts_in_delta", "-61"},
                        {"last_sequence", "366"},
                        {"last_has_ts_out", "1"},
                        {"last_ts_out", d::dec(UINT64_MAX)}};
  for (auto it = expected_last.begin(); it != expected_last.end(); ++it)
    check(completion.at("extensions").at(it.key()) == it.value());

  // RAM reuses the exact raw source and provider configuration, not a second
  // simplified strategy feed. The complete normalized census remains equal.
  auto resident_request = generate;
  resident_request["output_path"] = fresh("resident-census");
  auto resident_dataset = d::load_dataset(resident_request, e::no_deadline);
  d::generate_census(resident_request, e::no_deadline, resident_dataset.get());
  auto resident_result =
      read(resident_request.at("output_path").get<std::string>());
  check(data(resident_result, "census") == census);
  check(data(resident_result, "resources").at("dataset_feed").at("mode") ==
        "resident");
  check(data(call("generate_census", generate).result, "census") == census);

  auto old_bytes = fixture_bytes(true, false);
  write(root + "/v1.dbn", old_bytes);
  auto old_request = generate;
  old_request["source_path"] = root + "/v1.dbn";
  old_request["source_sha256"] = e::sha256_hex(old_bytes);
  auto old = call("generate_census", old_request);
  check(data(old.result, "signals") == signals);
  check(data(old.result, "provenance").at("dbn_version") == "1");
  check(data(old.result, "provider")
            .at("completion")
            .at("extensions")
            .at("last_has_ts_out") == "0");
  check(data(old.result, "provider")
            .at("completion")
            .at("extensions")
            .at("last_ts_out") == "0");
  check(data(old.result, "census").at("census_sha256") !=
        census.at("census_sha256"));

  auto model_provider = provider;
  model_provider["role"] = "model";
  model_provider["parameters"] = {{"model_variant", "1"}};
  J evaluation{
      {"protocol", "symphony.sbv.evaluate-input.v1"},
      {"source_path", source_path},
      {"source_sha256", generate.at("source_sha256")},
      {"dataset", "GLBX.MDP3"},
      {"census", generated.reference()},
      {"model",
       {{"protocol", "symphony.sbv.model-selection.v1"},
        {"id", "native_provider"},
        {"version", "1"},
        {"horizon_ns", "500"},
        {"parameters",
         {{"provider", model_provider},
          {"measure", "probability"},
          {"conditioning", "execution"},
          {"calibration_reference", ""},
          {"on_unavailable", "reject"}}}}},
      {"replay",
       {{"before_ns", "500"}, {"after_ns", "500"}, {"retain_events", true}}},
      {"studies",
       J::array({"signal_summary", "model_summary", "path_excursion"})},
      {"workers", "3"},
      {"extensions", J::object()}};
  auto a = call("evaluate", evaluation);
  check(data(a.result, "resources").at("requested_workers") == "3");
  check(data(a.result, "resources").at("actual_workers") == "1");
  check(data(a.result, "resources").at("provider_instances") == "1");
  check(data(a.result, "resources").at("provider_concurrency") ==
        "serialized_instance");
  check(data(a.result, "signals") == signals);
  check(data(a.result, "census") == census);
  for (std::size_t i = 0; i < signals.size(); ++i) {
    const auto &row = data(a.result, "execution")[i];
    check(row.at("model_id") == "native_provider");
    check(row.at("evidence_origin") == "native_provider");
    check(row.at("producer").at("artifact_sha256") == library_sha);
    check(row.at("evidence_reference") ==
          "fixture-full-mbo-prefix-and-followup-checked");
    check(row.at("fill_probability").at("status") == "external_assumption");
    check(row.at("fill_probability").at("value") == ratio("1", "2"));
    check(row.at("support")[0].at("price_nanos") ==
          d::dec(d::i64(signals[i].at("anchor_price_nanos")) - 1));
    check(row.at("support")[1].at("price_nanos") ==
          d::dec(d::i64(signals[i].at("anchor_price_nanos")) + 1));
  }
  for (const auto *concurrency :
       {"per_worker_instances", "shared_reentrant_instance"}) {
    auto q = evaluation;
    q["model"]["parameters"]["provider"]["concurrency"] = concurrency;
    auto parallel = call("evaluate", q);
    check(data(parallel.result, "resources").at("actual_workers") == "3");
    check(data(parallel.result, "resources").at("provider_instances") ==
          (std::string(concurrency) == "per_worker_instances" ? "3" : "1"));
    for (const auto *name :
         {"signals", "census", "execution", "replay", "studies", "summary"})
      check(data(parallel.result, name) == data(a.result, name));
  }
  auto b_request = evaluation;
  b_request["model"]["parameters"]["provider"]["parameters"]["model_variant"] =
      "2";
  auto b = call("evaluate", b_request);
  check(data(b.result, "census") == census);
  check(data(b.result, "signals") == signals);
  check(data(b.result, "execution") != data(a.result, "execution"));
  for (const auto &row : data(b.result, "execution"))
    check(row.at("fill_probability").at("value") == ratio("3", "4"));
  auto zero_window = evaluation;
  zero_window["model"]["horizon_ns"] = "0";
  check(data(call("evaluate", zero_window).result, "summary")
            .at("signal_count") == "4");
  auto next = b_request;
  next["census"] = a.reference();
  check(data(call("evaluate", next).result, "census") == census);
  auto nonprobability = evaluation;
  nonprobability["model"]["parameters"]["measure"] = "scenario_weight";
  nonprobability["model"]["parameters"]["provider"]["parameters"]["mode"] =
      "invalid_probability";
  auto scenarios = call("evaluate", nonprobability);
  check(data(scenarios.result, "execution")[0].at("support")[0].at("weight") ==
        ratio("1", "2"));
  check(data(scenarios.result, "execution")[0].at("support")[1].at("weight") ==
        ratio("1", "3"));
  nonprobability["model"]["parameters"]["measure"] = "signed_coefficient";
  nonprobability["model"]["parameters"]["provider"]["parameters"]["mode"] =
      "signed_coefficients";
  auto coefficients = call("evaluate", nonprobability);
  check(data(coefficients.result, "execution")[0].at("support")[0].at(
            "weight") == ratio("-1", "2"));
  check(data(coefficients.result, "execution")[0].at("support")[1].at(
            "weight") == ratio("1", "3"));
  nonprobability["model"]["parameters"]["measure"] = "scenario_weight";
  reject("evaluate", nonprobability);
  resident_request = evaluation;
  resident_request["output_path"] = fresh("resident-evaluate");
  d::evaluate(resident_request, e::no_deadline, resident_dataset.get());
  resident_result = read(resident_request.at("output_path").get<std::string>());
  check(data(resident_result, "execution") == data(a.result, "execution"));

  // A retained census is self-contained evidence. A later built-in model must
  // not load the original strategy binary or its captured dependency files.
  const auto transient_library = root + "/retired-provider.dylib";
  const auto transient_dependency = root + "/retired-config.txt";
  std::filesystem::copy_file(library, transient_library);
  write(transient_dependency, "captured strategy configuration\n");
  auto transient_request = generate;
  transient_request["provider"]["library"]["path"] = transient_library;
  transient_request["provider"]["dependencies"] = {
      {"capture", "declared_artifacts"},
      {"description", "Optional user-declared fixture configuration"},
      {"artifacts",
       J::array({{{"path", transient_dependency},
                  {"expected_sha256",
                   e::sha256_hex("captured strategy configuration\n")}}})}};
  auto retired = call("generate_census", transient_request);
  check(std::filesystem::remove(transient_library));
  check(std::filesystem::remove(transient_dependency));
  auto retained_request = evaluation;
  retained_request["census"] = retired.reference();
  retained_request["model"] = {{"protocol", "symphony.sbv.model-selection.v1"},
                               {"id", "observed_trade_levels"},
                               {"version", "1"},
                               {"horizon_ns", "500"},
                               {"parameters",
                                {{"levels_per_side", "1"},
                                 {"include_anchor", true},
                                 {"thin_support", "unavailable"}}}};
  auto retained = call("evaluate", retained_request);
  check(data(retained.result, "census") == data(retired.result, "census"));

  auto economic_request = [&](const Artifact &parent, std::size_t signal) {
    auto p = templates.at("economics");
    p.erase("path");
    p.erase("expected_sha256");
    p["source"] = parent.reference();
    p["studies"] = J::array();
    p["on_incompatible"] = "reject";
    p["selections"] = J::array(
        {{{"signal_id", signals[signal].at("signal_id")},
          {"mode", "execution_mixture"},
          {"nonexecution_pnl", ratio("0")},
          {"transform",
           {{"id", "linear_price_pnl"},
            {"version", "1"},
            {"reference_price_nanos", signals[signal].at("anchor_price_nanos")},
            {"price_unit_nanos", "1"},
            {"position_units", ratio("1")},
            {"value_per_price_unit", ratio("1")},
            {"cost_per_outcome", ratio("0")},
            {"return_basis", ratio("100")},
            {"pnl_unit", "fixture arithmetic unit"}}}}});
    return p;
  };
  auto ea = call("economics", economic_request(a, 0));
  auto eb = call("economics", economic_request(b, 2));
  check(data(ea.result, "census") == census);
  check(data(eb.result, "census") == census);
  check(data(ea.result, "source_context").at("provider") ==
        data(a.result, "provider"));
  check(data(eb.result, "source_context").at("provider") ==
        data(b.result, "provider"));
  check(data(ea.result, "economics")[0].at("atoms")[0].at("weight") ==
        ratio("1", "4"));
  check(data(eb.result, "economics")[0].at("atoms")[0].at("weight") ==
        ratio("3", "8"));
  auto composition = templates.at("compose_economics");
  composition["components"] = J::array();
  const std::vector<Artifact> parents{ea, eb};
  for (std::size_t i = 0; i < parents.size(); ++i)
    composition["components"].push_back(
        {{"id", "component-" + d::dec(i)},
         {"source", parents[i].reference()},
         {"outcome_pointer", "/sections/economics/data/0"},
         {"signal_id", signals[i * 2].at("signal_id")},
         {"conditioning", "execution_and_nonexecution"},
         {"conversion", nullptr},
         {"extensions", J::object()}});
  composition["studies"] = J::array({{{"id", "terminal_moments"},
                                      {"version", "1"},
                                      {"parameters", J::object()}}});
  auto composed = call("compose_economics", composition);
  check(data(composed.result, "summary").at("path_count") == "9");
  check(data(composed.result, "studies")[0].at("data").at("mean") ==
        ratio("1"));
  check(data(composed.result, "distributions")
            .at("terminal_atoms")
            .front()
            .at("state") == ratio("4851", "5000"));
  check(data(composed.result, "distributions")
            .at("terminal_atoms")
            .back()
            .at("state") == ratio("5151", "5000"));

  auto book_request = templates.at("book");
  book_request["source_path"] = source_path;
  book_request["source_sha256"] = generate.at("source_sha256");
  book_request["dataset"] = "GLBX.MDP3";
  book_request["census_result"] = {
      {"path", a.path}, {"expected_sha256", a.result.at("content_sha256")}};
  book_request["signal_ids"] =
      J::array({signals[0].at("signal_id"), signals[1].at("signal_id"),
                signals[2].at("signal_id")});
  book_request["initial_state"] = {{"mode", "uninitialized"}};
  book_request["on_anomaly"] = "reject";
  book_request["replay"] = {
      {"before_ns", "500"}, {"after_ns", "100"}, {"retain_events", true}};
  book_request["frames"] = {
      {"cadence", "signals"}, {"levels_per_side", "1"}, {"maximum", "16"}};
  auto book = call("book", book_request);
  check(data(book.result, "summary").at("parent_census_sha256") ==
        census.at("census_sha256"));
  check(data(book.result, "book_frames")[0].at("signal_ids").size() == 2);
  check(data(book.result, "book_frames")[0].at("status") == "available");

  check(std::filesystem::remove(a.path));
  check(std::filesystem::remove(b.path));
  auto detached_composition = call("compose_economics", composition);
  check(data(detached_composition.result, "distributions") ==
        data(composed.result, "distributions"));
  check(data(detached_composition.result, "composition_sources")[0].at(
            "provider") == data(a.result, "provider"));
  check(data(detached_composition.result, "composition_sources")[0].at(
            "model_selection") == evaluation.at("model"));
  // A valid source result hash cannot conceal inconsistent copied provider
  // attribution in economics consumed later by the composer.
  auto inconsistent_economic = ea.result;
  inconsistent_economic["sections"]["source_context"]["data"]["provider"]
                       ["evidence"]["id"] = "wrong-model-provider";
  inconsistent_economic = s::seal_result(std::move(inconsistent_economic));
  const auto inconsistent_path = fresh("inconsistent-economic");
  write(inconsistent_path, inconsistent_economic.dump());
  auto inconsistent_composition = composition;
  inconsistent_composition["components"][0]["source"] = {
      {"path", inconsistent_path},
      {"expected_sha256", inconsistent_economic.at("content_sha256")},
      {"pointer", ""}};
  reject("compose_economics", inconsistent_composition);

  // Declared provider failures and malformed candidates never publish a partial
  // closed census or a half-computed alternative evaluation.
  for (const auto *mode :
       {"duplicate_id", "bad_price", "provider_failure", "malformed_json"}) {
    auto q = generate;
    q["provider"]["parameters"]["mode"] = mode;
    reject("generate_census", q);
  }
  auto q = generate;
  q["provider"]["parameters"]["fail_after"] = "5";
  reject("generate_census", q);
  q = generate;
  q["source_sha256"] = std::string(64, '0');
  reject("generate_census", q);
  q = generate;
  q["provider"]["library"]["expected_sha256"] = std::string(64, '0');
  reject("generate_census", q);
  for (const auto *mode :
       {"provider_failure", "malformed_json", "invalid_probability"}) {
    q = evaluation;
    q["model"]["parameters"]["provider"]["parameters"]["mode"] = mode;
    reject("evaluate", q);
  }
  for (const auto *concurrency :
       {"per_worker_instances", "shared_reentrant_instance"}) {
    q = evaluation;
    q["model"]["parameters"]["provider"]["concurrency"] = concurrency;
    q["model"]["parameters"]["provider"]["parameters"]["mode"] =
        "provider_failure";
    reject("evaluate", q);
  }
  q = evaluation;
  q["model"]["parameters"]["provider"]["parameters"]["mode"] = "unavailable";
  reject("evaluate", q);
  q["model"]["parameters"]["on_unavailable"] = "unavailable";
  auto unavailable = call("evaluate", q);
  check(data(unavailable.result, "summary").at("census_sha256") ==
        census.at("census_sha256"));
  for (const auto &row : data(unavailable.result, "execution")) {
    check(row.at("status") == "unavailable");
    check(row.at("support").empty());
    check(J::parse(row.at("reason").get<std::string>()).at("message") ==
          "selected model unavailable");
  }
  q = evaluation;
  q["model"]["parameters"]["measure"] = "undeclared_measure";
  reject("evaluate", q);
  q = generate;
  q["provider"]["parameters"]["emissions_per_event"] = "0";
  auto empty = call("generate_census", q);
  check(data(empty.result, "signals").empty());
  q = evaluation;
  q["census"] = empty.reference();
  auto empty_evaluation = call("evaluate", q);
  check(data(empty_evaluation.result, "resources").at("actual_workers") == "0");
  check(data(empty_evaluation.result, "resources").at("provider_instances") ==
        "0");
  // Cross the historical 4096-signal profile in generation explicitly. Later
  // consumers retain their disclosed representation/profile limits until R3.
  q = generate;
  q["provider"]["parameters"]["emissions_per_event"] = "2049";
  auto many = call("generate_census", q);
  check(data(many.result, "signals").size() == 4098);
  check(data(many.result, "summary").at("signal_count") == "4098");

  auto malformed = generated.result;
  malformed["sections"]["census"]["data"]["declaration"]["provider"]
           ["parameters"]["action"] = "A";
  malformed = s::seal_result(malformed);
  const auto malformed_path = fresh("malformed-census");
  write(malformed_path, malformed.dump());
  q = evaluation;
  q["census"] = {{"path", malformed_path},
                 {"expected_sha256", malformed.at("content_sha256")},
                 {"pointer", ""}};
  reject("evaluate", q);

  // Mutations keep all outer digests coherent, so these are correspondence
  // checks rather than failures caused only by an unsealed artifact.
  const auto reject_evidence = [&](const auto &mutate) {
    auto forged = generated.result;
    auto &c = forged["sections"]["census"]["data"];
    mutate(c["declaration"]["provider_evidence"]);
    c["census_sha256"] = e::sha256_hex(c.at("declaration").dump());
    forged["sections"]["provider"]["data"]["evidence"] =
        c.at("declaration").at("provider_evidence");
    forged["sections"]["summary"]["data"]["census_sha256"] =
        c.at("census_sha256");
    forged["sections"]["provenance"]["data"]["census_sha256"] =
        c.at("census_sha256");
    forged = s::seal_result(std::move(forged));
    const auto path = fresh("forged-evidence");
    write(path, forged.dump());
    auto request = evaluation;
    request["census"] = {{"path", path},
                         {"expected_sha256", forged.at("content_sha256")},
                         {"pointer", ""}};
    reject("evaluate", request);
  };
  reject_evidence([](J &evidence) {
    auto descriptor =
        J::parse(evidence.at("descriptor_json").get<std::string>());
    descriptor["id"] = "other-provider";
    evidence["descriptor_json"] = descriptor.dump();
    evidence["descriptor_sha256"] = e::sha256_hex(descriptor.dump());
  });
  reject_evidence([](J &evidence) { evidence["abi_version"] = "2"; });
  reject_evidence([](J &evidence) { evidence["loading"]["isolation"] = true; });
  reject_evidence([](J &evidence) {
    evidence["dependencies"]["description"] = "changed capture";
  });
  q = evaluation;
  q["source_path"] = old_request.at("source_path");
  q["source_sha256"] = old_request.at("source_sha256");
  reject("evaluate", q);

  if (const auto *destination =
          std::getenv("SYMPHONY_SBV_PROVIDER_FIXTURE_OUT")) {
    const std::filesystem::path path(destination);
    check(path.is_absolute() && std::filesystem::is_directory(path));
    write((path / "fixture.dbn").string(), raw);
    auto emitted_generate = generate;
    emitted_generate["source_path"] = (path / "fixture.dbn").string();
    emitted_generate["output_path"] = (path / "generated.json").string();
    write((path / "generate-request.json").string(), emitted_generate.dump(2));
    auto emitted_evaluate = evaluation;
    emitted_evaluate["source_path"] = (path / "fixture.dbn").string();
    emitted_evaluate["output_path"] = (path / "evaluated.json").string();
    emitted_evaluate["census"] = nullptr;
    write((path / "evaluate-request.json").string(), emitted_evaluate.dump(2));
  }
  check(e::sha256_hex(d::read_file(library, e::no_deadline)) == library_sha);
  std::cout << checks << " SBV native provider integration assertions passed\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
