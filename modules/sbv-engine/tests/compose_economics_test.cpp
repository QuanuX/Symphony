#include "../src/detail.hpp"
#include "../src/wide_rational.hpp"
#include <cstdlib>
#include <filesystem>
#include <iostream>
#include <source_location>
#include <unistd.h>

namespace s = symphony::sbv;
namespace d = s::detail;
namespace e = symphony::knowledge::engine;
namespace w = d::wide_rational;
using J = s::Json;
unsigned checks = 0;
void check(bool value,
           std::source_location at = std::source_location::current()) {
  ++checks;
  if (!value)
    throw std::runtime_error("compose economics assertion " +
                             d::dec(at.line()));
}
J rat(const char *n, const char *den = "1") {
  return {{"numerator", n}, {"denominator", den}};
}
J data(const J &j, const char *section) {
  return j.at("sections").at(section).at("data");
}
// Independently supplied, coherent fixtures. Expected composed outcomes below
// are hand-calculated; this builder only authors per-signal source artifacts.
J source(std::string id, std::vector<std::int64_t> prices,
         J weights = J::array({rat("1", "2"), rat("1", "2")}),
         std::string measure = "probability", std::string unit = "USD") {
  check(prices.size() == weights.size());
  J transform{{"id", "linear_price_pnl"},
              {"version", "1"},
              {"reference_price_nanos", "100"},
              {"price_unit_nanos", "1"},
              {"position_units", rat("1")},
              {"value_per_price_unit", rat("1")},
              {"cost_per_outcome", rat("0")},
              {"return_basis", rat("100")},
              {"pnl_unit", unit}};
  J support = J::array(), transformed = J::array(), atoms = J::array();
  for (std::size_t i = 0; i < prices.size(); ++i) {
    auto pnl = w::R{prices[i] - 100, 1};
    auto value = w::wire(w::divide(pnl, {100, 1}));
    support.push_back(
        {{"price_nanos", d::dec(prices[i])}, {"weight", weights[i]}});
    transformed.push_back({{"price_nanos", d::dec(prices[i])},
                           {"weight", weights[i]},
                           {"gross_pnl", w::wire(pnl)},
                           {"cost", rat("0")},
                           {"net_pnl", w::wire(pnl)},
                           {"return", value}});
    atoms.push_back({{"kind", "price_outcome"},
                     {"support_index", d::dec(i)},
                     {"return", value},
                     {"weight", weights[i]}});
  }
  J row{{"protocol", "symphony.sbv.economic-outcome.v1"},
        {"signal_id", id},
        {"status", "available"},
        {"reason", ""},
        {"measure", measure},
        {"conditioning", "execution"},
        {"mode", "support_only"},
        {"transform", transform},
        {"transformed_support", transformed},
        {"nonexecution_pnl", nullptr},
        {"atoms", atoms},
        {"calibration", "not_verified"}};
  J model{{"protocol", "symphony.sbv.model-outcome.v1"},
          {"signal_id", id},
          {"status", "available"},
          {"reason", ""},
          {"model_id", "external_outcomes"},
          {"model_version", "1"},
          {"measure", measure},
          {"conditioning", "execution"},
          {"support", support},
          {"price_unit", "price_nanos"},
          {"price_scale", "1e-9"},
          {"fill_probability", d::missing("not selected")},
          {"no_fill_probability", d::missing("not selected")}};
  J choice{{"signal_id", id},
           {"transform", transform},
           {"mode", "support_only"},
           {"nonexecution_pnl", nullptr}};
  J producer{{"id", "composition-fixture"},
             {"version", "1"},
             {"artifact_sha256", ""},
             {"reproducibility", "deterministic_declared"}};
  J signal{{"signal_id", id},
           {"source_ordinal", "0"},
           {"available_ns", "9007199254740993"},
           {"anchor_price_nanos", "100"},
           {"causal_end_ordinal_exclusive", "1"},
           {"context_reference", "fixture://arithmetic"}};
  J declaration{{"protocol", "symphony.sbv.external-census.v1"},
                {"source_sha256", std::string(64, 'b')},
                {"mode", "causal_declared"},
                {"producer", producer},
                {"signals", J::array({signal})}};
  const auto census_sha = e::sha256_hex(declaration.dump());
  J census{{"protocol", "symphony.sbv.census-evidence.v1"},
           {"kind", "external"},
           {"identity_domain", "external_declaration"},
           {"census_sha256", census_sha},
           {"source_sha256", std::string(64, 'b')},
           {"dataset", "fixture"},
           {"instrument_id", "1"},
           {"mode", "causal_declared"},
           {"producer", producer},
           {"signals", J::array({signal})},
           {"declaration", declaration}};
  auto parent =
      d::base("independently supplied evaluation fixture; no market evidence");
  parent["sections"]["census"] = d::section(census);
  parent["sections"]["signals"] = d::section(J::array({signal}));
  parent["sections"]["execution"] = d::section(J::array({model}));
  parent["sections"]["choices"] = d::section(
      {{"protocol", "symphony.sbv.evaluate-input.v1"},
       {"source_sha256", std::string(64, 'b')},
       {"source_path", "fixture://market"},
       {"dataset", "fixture"},
       {"census", declaration},
       {"model",
        {{"protocol", "symphony.sbv.model-selection.v1"},
         {"id", "external_outcomes"},
         {"version", "1"},
         {"horizon_ns", "0"},
         {"parameters",
          {{"producer", producer},
           {"measure", measure},
           {"conditioning", "execution"},
           {"calibration_reference", ""},
           {"outcomes", J::array({{{"signal_id", id},
                                   {"evidence_reference", "fixture arithmetic"},
                                   {"execution_probability",
                                    {{"status", "unavailable"},
                                     {"reason", "not selected"}}},
                                   {"support", support}}})}}}}},
       {"replay",
        {{"before_ns", "0"}, {"after_ns", "0"}, {"retain_events", false}}},
       {"studies", J::array()},
       {"workers", "1"},
       {"extensions", J::object()}});
  parent["sections"]["provenance"] =
      d::section({{"engine_version", s::version},
                  {"source_sha256", std::string(64, 'b')},
                  {"dataset", "fixture"},
                  {"instrument_id", "1"},
                  {"census_sha256", census_sha},
                  {"census_producer", producer}});
  parent["sections"]["summary"] =
      d::section({{"signal_count", "1"},
                  {"closed_census", true},
                  {"census_sha256", census_sha},
                  {"causality", "causal_declared"}});
  parent = s::seal_result(parent);
  auto result = d::base("independently supplied composition arithmetic "
                        "fixture; no market evidence");
  result["sections"]["economics"] = d::section(J::array({row}));
  result["sections"]["execution"] = d::section(J::array({model}));
  result["sections"]["signals"] = d::section(J::array({signal}));
  result["sections"]["census"] = d::section(census);
  result["sections"]["choices"] =
      d::section({{"protocol", "symphony.sbv.economics-input.v1"},
                  {"path", "fixture://evaluation"},
                  {"expected_sha256", parent.at("content_sha256")},
                  {"selections", J::array({choice})}});
  result["sections"]["summary"] =
      d::section({{"source_census_sha256", census_sha},
                  {"source_signal_count", "1"},
                  {"selected_signal_count", "1"}});
  result["sections"]["provenance"] =
      d::section({{"source_path", "fixture://evaluation"},
                  {"source_content_sha256", parent.at("content_sha256")},
                  {"source_file_sha256", e::sha256_hex(parent.dump())},
                  {"source_authorship", "not_verified"}});
  result["sections"]["source_context"] =
      d::section({{"census", census},
                  {"choices", parent.at("sections").at("choices")},
                  {"provenance", parent.at("sections").at("provenance")}});
  result["sections"]["replay"] =
      d::section({{"events", J::array({{{"time", "18446744073709551615"}}})}});
  return result;
}
int main() try {
  char tmp[] = "/private/tmp/sbv-composition-XXXXXX";
  auto env = std::getenv("SYMPHONY_SBV_COMPOSITION_FIXTURES");
  auto made = env ? env : mkdtemp(tmp);
  check(made);
  std::filesystem::path root(made);
  std::filesystem::create_directories(root);
  struct Cleanup {
    std::filesystem::path root;
    bool retain;
    ~Cleanup() {
      if (!retain)
        std::filesystem::remove_all(root);
    }
  } cleanup{root, env != nullptr};
  unsigned seq = 0;
  auto path = [&](std::string name) { return (root / name).string(); };
  auto read = [&](const std::string &name) {
    auto result = e::parse_bounded_json(d::read_file(name, e::no_deadline),
                                        d::artifact_bytes, d::artifact_values);
    s::validate_result(result);
    return result;
  };
  auto save = [&](J result) {
    result = s::seal_result(result);
    auto file = path("source-" + d::dec(++seq) + ".json");
    d::create_file(file, result.dump(), e::no_deadline);
    return J{{"path", file},
             {"expected_sha256", result.at("content_sha256")},
             {"pointer", ""}};
  };
  auto component = [&](std::string id, J ref, std::string signal) {
    return J{{"id", id},
             {"source", ref},
             {"outcome_pointer", "/sections/economics/data/0"},
             {"signal_id", signal},
             {"conditioning", "execution"},
             {"conversion", nullptr},
             {"extensions", J::object()}};
  };
  auto a = source("a", {99, 101}), b = source("b", {99, 101});
  auto ar = save(a), br = save(b);
  auto original_a = d::read_file(ar.at("path"), e::no_deadline);
  J p{{"protocol", "symphony.sbv.compose-economics-input.v1"},
      {"output_path", "unused"},
      {"components",
       J::array({component("first", ar, "a"), component("second", br, "b")})},
      {"dependence",
       {{"kind", "independent"},
        {"marginal_policy", nullptr},
        {"paths", J::array()},
        {"description", "Caller-selected independent outcomes under retained "
                        "source conditions."}}},
      {"economics",
       {{"kind", "multiplicative_return"},
        {"initial_state", rat("1")},
        {"state_unit", "wealth-factor"},
        {"state_domain", "signed"},
        {"description", "Compound declared source returns, without account "
                        "feasibility inference."}}},
      {"conditioning_description", "Each price support is conditional on "
                                   "execution; no unconditional fill claim."},
      {"on_unavailable", "reject"},
      {"retain_paths", true},
      {"retain_prefix_distributions", true},
      {"studies", J::array()},
      {"limits", {{"max_paths", nullptr}, {"max_terminal_atoms", nullptr}}},
      {"extensions", J::object()}};
  J last_request, last_receipt;
  auto call = [&](J request) {
    request["output_path"] = path("result-" + d::dec(++seq) + ".json");
    last_request = request;
    last_receipt = s::dispatch("compose_economics", request, e::no_deadline);
    check(last_receipt.at("protocol") == "symphony.sbv.compose-economics.v1");
    auto result = read(request.at("output_path"));
    check(result.at("content_sha256") == last_receipt.at("content_sha256"));
    return result;
  };
  auto rejects = [&](J request, std::int64_t end = e::no_deadline) {
    request["output_path"] = path("rejected-" + d::dec(++seq) + ".json");
    bool failed = false;
    try {
      s::dispatch("compose_economics", request, end);
    } catch (const std::exception &) {
      failed = true;
    }
    check(failed);
    check(
        !std::filesystem::exists(request.at("output_path").get<std::string>()));
  };
  auto result = call(p);
  auto distribution = data(result, "distributions");
  auto terminal = distribution.at("terminal_atoms");
  check(terminal ==
        J::array(
            {{{"state", rat("9801", "10000")}, {"probability", rat("1", "4")}},
             {{"state", rat("9999", "10000")}, {"probability", rat("1", "2")}},
             {{"state", rat("10201", "10000")},
              {"probability", rat("1", "4")}}}));
  check(data(result, "summary").at("path_count") == "4");
  check(distribution.at("paths").size() == 4);
  check(distribution.at("paths")[1].at("atom_indices") == J::array({"0", "1"}));
  check(distribution.at("paths")[1].at("state") ==
        J::array({rat("1"), rat("99", "100"), rat("9999", "10000")}));
  check(distribution.at("prefix_distributions").size() == 3);
  check(distribution.at("prefix_distributions")[0].at("atoms") ==
        J::array({{{"state", rat("1")}, {"probability", rat("1")}}}));
  check(distribution.at("prefix_distributions")[1].at("atoms")[0].at(
            "probability") == rat("1", "2"));
  check(distribution.at("marginals")[0].at("weights_match") == true);
  check(data(result, "composition_sources")[0].at("economic_outcome") ==
        data(a, "economics")[0]);
  check(data(result, "composition_sources")[0].at("source_summary") ==
        a.at("sections").at("summary"));
  check(data(result, "composition_sources")[0]
            .at("replay_reference")
            .at("expected_sha256") == ar.at("expected_sha256"));
  check(result.at("sections").at("studies").at("status") == "not_selected");
  check(data(result, "studies").empty());
  check(d::read_file(ar.at("path"), e::no_deadline) == original_a);

  auto q = p;
  q["retain_paths"] = false;
  q["retain_prefix_distributions"] = false;
  auto compact = call(q);
  check(data(compact, "distributions").at("terminal_atoms") == terminal);
  check(data(compact, "distributions").at("paths").empty());
  check(data(compact, "distributions").at("prefix_distributions").empty());
  check(data(compact, "summary").at("path_count") == "4");
  check(data(compact, "summary").at("retained_path_count") == "0");
  q = p;
  q["studies"] = J::array();
  q["studies"].push_back({{"id", "terminal_moments"},
                          {"version", "1"},
                          {"parameters", J::object()}});
  q["studies"].push_back({{"id", "terminal_quantiles"},
                          {"version", "1"},
                          {"parameters",
                           {{"levels", J::array({rat("0"), rat("1", "4"),
                                                 rat("1", "2"), rat("1")})}}}});
  auto analyzed = call(q);
  auto studies = data(analyzed, "studies");
  check(studies[0].at("data").at("mean") == rat("1"));
  check(studies[0].at("data").at("population_variance") ==
        rat("20001", "100000000"));
  check(studies[1].at("data").at("quantiles")[0].at("state") ==
        rat("9801", "10000"));
  check(studies[1].at("data").at("quantiles")[1].at("state") ==
        rat("9801", "10000"));
  check(studies[1].at("data").at("quantiles")[2].at("state") ==
        rat("9999", "10000"));
  check(studies[1].at("data").at("quantiles")[3].at("state") ==
        rat("10201", "10000"));
  q["studies"][1]["parameters"]["levels"] = J::array();
  check(data(call(q), "studies")[1].at("data").at("quantiles").empty());

  auto ha =
      save(source("ha", {99, 101}, J::array({rat("1", "4"), rat("3", "4")})));
  auto hb =
      save(source("hb", {98, 103}, J::array({rat("2", "5"), rat("3", "5")})));
  q = p;
  q["components"] =
      J::array({component("a", ha, "ha"), component("b", hb, "hb")});
  auto heterogeneous = call(q);
  check(
      data(heterogeneous, "distributions").at("terminal_atoms") ==
      J::array(
          {{{"state", rat("4851", "5000")}, {"probability", rat("1", "10")}},
           {{"state", rat("4949", "5000")}, {"probability", rat("3", "10")}},
           {{"state", rat("10197", "10000")}, {"probability", rat("3", "20")}},
           {{"state", rat("10403", "10000")},
            {"probability", rat("9", "20")}}}));
  q = p;
  q["components"][1] = component("repeat", ar, "a");
  auto repeated = call(q);
  check(data(repeated, "resources").at("distinct_source_references") == "1");
  check(data(repeated, "distributions").at("terminal_atoms") == terminal);
  q["components"][1]["id"] = q["components"][0]["id"];
  rejects(q);
  auto hc = save(source("hc", {99, 100, 102},
                        J::array({rat("0"), rat("1", "4"), rat("3", "4")})));
  q = p;
  q["components"][1] = component("c", hc, "hc");
  auto unequal = call(q);
  check(data(unequal, "summary").at("path_count") == "6");
  check(data(unequal, "distributions").at("paths")[0].at("probability") ==
        rat("0"));

  auto joint = p;
  joint["dependence"] = {
      {"kind", "joint_indices"},
      {"marginal_policy", "require_source_match"},
      {"description", "Only two mixed-sign outcomes are selected."},
      {"paths", J::array({{{"id", "up-down"},
                           {"atom_indices", J::array({"1", "0"})},
                           {"probability", rat("1", "2")}},
                          {{"id", "down-up"},
                           {"atom_indices", J::array({"0", "1"})},
                           {"probability", rat("1", "2")}}})}};
  auto coupled = call(joint);
  check(
      data(coupled, "distributions").at("terminal_atoms") ==
      J::array({{{"state", rat("9999", "10000")}, {"probability", rat("1")}}}));
  check(data(coupled, "distributions").at("paths")[0].at("id") == "up-down");
  check(data(coupled, "distributions").at("marginals")[1].at("weights_match") ==
        true);
  q = joint;
  q["dependence"]["paths"] = J::array({{{"id", "both-up"},
                                        {"atom_indices", J::array({"1", "1"})},
                                        {"probability", rat("1")}}});
  rejects(q);
  q["dependence"]["marginal_policy"] = "support_only";
  auto override_result = call(q);
  check(data(override_result, "distributions")
            .at("terminal_atoms")[0]
            .at("probability") == rat("1"));
  check(data(override_result, "distributions")
            .at("marginals")[0]
            .at("weights_match") == false);
  check(data(override_result, "distributions")
            .at("marginals")[0]
            .at("atoms")[1]
            .at("induced_probability") == rat("1"));
  q = joint;
  q["dependence"]["paths"][1]["atom_indices"] = J::array({"1", "0"});
  q["dependence"]["marginal_policy"] = "support_only";
  check(data(call(q), "summary").at("path_count") == "2");
  for (const auto &index : {J("2"), J("-1"), J("01"), J(1)}) {
    q = joint;
    q["dependence"]["paths"][0]["atom_indices"][0] = index;
    rejects(q);
  }
  q = joint;
  q["dependence"]["paths"][1]["id"] = "up-down";
  rejects(q);
  q = joint;
  q["dependence"]["paths"][0]["atom_indices"] = J::array({"0"});
  rejects(q);
  q = joint;
  q["dependence"]["paths"][0]["probability"] = rat("-1", "2");
  rejects(q);
  q = joint;
  q["dependence"]["paths"][0]["probability"] = rat("1", "3");
  rejects(q);
  q = joint;
  q["dependence"]["paths"] = J::array();
  rejects(q);

  auto signed_source =
      save(source("signed", {99, 101}, J::array({rat("-1"), rat("2")}),
                  "signed_coefficient"));
  q = p;
  q["components"] = J::array({component("signed", signed_source, "signed")});
  rejects(q);
  q["on_unavailable"] = "unavailable";
  auto unavailable = call(q);
  check(unavailable.at("status") == "partial");
  check(data(unavailable, "distributions").at("terminal_atoms").is_null());
  check(data(unavailable, "summary").at("probability_mass").is_null());
  check(data(unavailable, "composition_findings").size() == 1);
  q["dependence"] = {{"kind", "joint_indices"},
                     {"marginal_policy", "support_only"},
                     {"description",
                      "Explicit probability scenarios use the signed source "
                      "only for economic support."},
                     {"paths", J::array({{{"id", "one"},
                                          {"atom_indices", J::array({"0"})},
                                          {"probability", rat("1")}}})}};
  auto reweighted = call(q);
  check(data(reweighted, "distributions")
            .at("marginals")[0]
            .at("source_measure") == "signed_coefficient");
  check(data(reweighted, "distributions").at("terminal_atoms")[0].at("state") ==
        rat("99", "100"));

  auto mixture = source("mix", {110}, J::array({rat("1")}));
  auto &mr = mixture["sections"]["economics"]["data"][0];
  mr["mode"] = "execution_mixture";
  mr["conditioning"] = "execution_and_nonexecution";
  mr["nonexecution_pnl"] = rat("-1");
  mr["atoms"][0]["weight"] = rat("1", "4");
  mr["atoms"].push_back({{"kind", "nonexecution"},
                         {"support_index", nullptr},
                         {"return", rat("-1", "100")},
                         {"weight", rat("3", "4")}});
  auto &mm = mixture["sections"]["execution"]["data"][0];
  mm["fill_probability"] = {{"status", "external_assumption"},
                            {"value", rat("1", "4")}};
  mm["no_fill_probability"] = {{"status", "external_assumption"},
                               {"value", rat("3", "4")}};
  mixture["sections"]["choices"]["data"]["selections"][0]["mode"] =
      "execution_mixture";
  mixture["sections"]["choices"]["data"]["selections"][0]["nonexecution_pnl"] =
      rat("-1");
  q = p;
  q["components"] = J::array({component("mixture", save(mixture), "mix")});
  q["components"][0]["conditioning"] = "execution_and_nonexecution";
  auto mixed = call(q);
  check(
      data(mixed, "distributions").at("terminal_atoms") ==
      J::array({{{"state", rat("99", "100")}, {"probability", rat("3", "4")}},
                {{"state", rat("11", "10")}, {"probability", rat("1", "4")}}}));
  auto absent = a;
  absent["status"] = "partial";
  absent["sections"]["economics"]["data"][0]["status"] = "unavailable";
  absent["sections"]["economics"]["data"][0]["reason"] =
      "explicit fixture unavailable";
  absent["sections"]["economics"]["data"][0]["atoms"] = J::array();
  q = p;
  q["components"][0]["source"] = save(absent);
  rejects(q);
  q["on_unavailable"] = "unavailable";
  q["studies"] = J::array({{{"id", "terminal_moments"},
                            {"version", "1"},
                            {"parameters", J::object()}}});
  auto partial = call(q);
  check(data(partial, "summary").at("path_count").is_null());
  check(data(partial, "studies")[0].at("status") == "unavailable");
  check(data(partial, "studies")[0].at("data").is_null());
  auto partial_outer = a;
  partial_outer["status"] = "partial";
  q = p;
  q["components"][0]["source"] = save(partial_outer);
  check(call(q).at("status") == "completed");

  auto eur =
      save(source("eur", {99, 101}, J::array({rat("1", "2"), rat("1", "2")}),
                  "probability", "EUR"));
  q = p;
  q["components"][1] = component("eur", eur, "eur");
  q["economics"]["kind"] = "additive_pnl";
  q["economics"]["initial_state"] = rat("10");
  q["economics"]["state_unit"] = "USD";
  q["components"][0]["conversion"] = {{"factor", rat("1")},
                                      {"output_unit", "USD"},
                                      {"description", "Identity conversion."}};
  q["components"][1]["conversion"] = {
      {"factor", rat("2")},
      {"output_unit", "USD"},
      {"description", "Caller scenario conversion; not market FX."}};
  auto additive = call(q);
  check(data(additive, "distributions").at("terminal_atoms") ==
        J::array({{{"state", rat("7")}, {"probability", rat("1", "4")}},
                  {{"state", rat("9")}, {"probability", rat("1", "4")}},
                  {{"state", rat("11")}, {"probability", rat("1", "4")}},
                  {{"state", rat("13")}, {"probability", rat("1", "4")}}}));
  check(data(additive, "composition_sources")[1]
            .at("economic_outcome")
            .at("transform")
            .at("pnl_unit") == "EUR");
  q["components"][1]["conversion"]["output_unit"] = "EUR";
  rejects(q);
  q = p;
  q["economics"]["kind"] = "additive_pnl";
  rejects(q);
  q = p;
  q["components"][0]["conversion"] = {
      {"factor", rat("1")}, {"output_unit", "USD"}, {"description", "unused"}};
  rejects(q);

  auto negative = save(source("negative", {-100}, J::array({rat("1")})));
  q = p;
  q["components"] = J::array({component("n1", negative, "negative"),
                              component("n2", negative, "negative")});
  auto signed_states = call(q);
  check(data(signed_states, "distributions").at("paths")[0].at("state") ==
        J::array({rat("1"), rat("-1"), rat("1")}));
  q["economics"]["state_domain"] = "nonnegative";
  rejects(q);
  q = p;
  q["economics"]["initial_state"] = rat("-1");
  check(data(call(q), "distributions").at("terminal_atoms")[0].at("state") ==
        rat("-10201", "10000"));
  q["economics"]["state_domain"] = "nonnegative";
  rejects(q);
  q = p;
  q["economics"]["initial_state"] = rat("0");
  check(data(call(q), "distributions").at("terminal_atoms") ==
        J::array({{{"state", rat("0")}, {"probability", rat("1")}}}));
  auto zero_growth = save(source("zero", {0}, J::array({rat("1")})));
  q = p;
  q["components"] = J::array({component("zero", zero_growth, "zero")});
  q["economics"]["state_domain"] = "nonnegative";
  check(data(call(q), "distributions").at("terminal_atoms")[0].at("state") ==
        rat("0"));
  auto double_growth = save(source("double", {200}, J::array({rat("1")})));
  q = p;
  q["components"] = J::array({component("double", double_growth, "double")});
  q["economics"]["initial_state"] = rat("18446744073709551616");
  check(data(call(q), "distributions").at("terminal_atoms")[0].at("state") ==
        rat("36893488147419103232"));
  q["economics"]["initial_state"] =
      rat("170141183460469231731687303715884105727");
  rejects(q);
  for (const auto &bad : {rat("1", "0"), rat("01"), rat("-0"),
                          rat("170141183460469231731687303715884105728")}) {
    q = p;
    q["economics"]["initial_state"] = bad;
    rejects(q);
  }
  q = p;
  q["limits"]["max_paths"] = "3";
  rejects(q);
  q = p;
  q["limits"]["max_terminal_atoms"] = "2";
  rejects(q);
  q = p;
  q["limits"]["max_paths"] = "4";
  q["limits"]["max_terminal_atoms"] = "3";
  check(data(call(q), "summary").at("path_count") == "4");
  q = p;
  q["components"] = J::array();
  auto empty = call(q);
  check(data(empty, "summary").at("path_count") == "1");
  check(data(empty, "distributions").at("terminal_atoms") ==
        J::array({{{"state", rat("1")}, {"probability", rat("1")}}}));
  q["limits"]["max_paths"] = "0";
  rejects(q);

  auto wrapped = d::base("embedded immutable economics result");
  wrapped["sections"]["wrapped"] = d::section(read(ar.at("path")));
  auto wrapped_ref = save(wrapped);
  wrapped_ref["pointer"] = "/sections/wrapped/data";
  q = p;
  q["components"][0]["source"] = wrapped_ref;
  auto embedded = call(q);
  check(data(embedded, "distributions").at("terminal_atoms") == terminal);
  check(data(embedded, "composition_sources")[0].at("source").at(
            "selected_sha256") == ar.at("expected_sha256"));
  check(data(embedded, "composition_sources")[0]
            .at("replay_reference")
            .at("pointer") == "/sections/wrapped/data/sections/replay");
  auto legacy = a;
  legacy["sections"].erase("census");
  legacy["sections"]["source_context"]["data"]["census"] =
      a.at("sections").at("census").at("data").at("declaration");
  q = p;
  q["components"][0]["source"] = save(legacy);
  auto legacy_result = call(q);
  check(data(legacy_result, "composition_sources")[0].at("census") ==
        a.at("sections").at("census").at("data"));
  check(data(legacy_result, "distributions").at("terminal_atoms") == terminal);
  auto referenced = a;
  auto &rc = referenced["sections"]["choices"]["data"];
  J parent_ref{{"path", rc.at("path")},
               {"expected_sha256", rc.at("expected_sha256")},
               {"pointer", ""}};
  rc.erase("path");
  rc.erase("expected_sha256");
  rc["source"] = parent_ref;
  auto &rp = referenced["sections"]["provenance"]["data"];
  rp["source_outer_content_sha256"] = parent_ref.at("expected_sha256");
  rp["source_pointer"] = "";
  auto retained_ref = parent_ref;
  retained_ref["file_sha256"] = rp.at("source_file_sha256");
  retained_ref["selected_content_sha256"] = rp.at("source_content_sha256");
  retained_ref["authorship"] = "not_verified";
  referenced["sections"]["source_context"]["data"]["reference"] = retained_ref;
  q = p;
  q["components"][0]["source"] = save(referenced);
  check(data(call(q), "distributions").at("terminal_atoms") == terminal);
  for (int which = 0; which < 12; ++which) {
    auto broken = which == 10 ? referenced : a;
    auto &bs = broken["sections"];
    if (which == 0)
      bs["summary"]["data"]["source_census_sha256"] = std::string(64, '0');
    if (which == 1)
      bs["signals"]["data"][0]["available_ns"] = "9007199254740994";
    if (which == 2)
      bs["source_context"]["data"]["census"]["instrument_id"] = "2";
    if (which == 3)
      bs.erase("replay");
    if (which == 4)
      bs.erase("source_context");
    if (which == 5)
      bs["source_context"]["data"]["provenance"]["data"]["census_sha256"] =
          std::string(64, '0');
    if (which == 6)
      bs["source_context"]["data"]["choices"]["data"]["model"]["id"] =
          "not_the_model";
    if (which == 7)
      bs["provenance"]["data"]["source_content_sha256"] = std::string(64, '0');
    if (which == 8)
      bs["census"]["data"]["protocol"] = "unknown";
    if (which == 9)
      bs["summary"]["data"]["source_signal_count"] = "2";
    if (which == 10)
      bs["source_context"]["data"]["reference"]["pointer"] = "/changed";
    if (which == 11) {
      bs["execution"]["data"][0]["conditioning"] = "unconditional";
      bs["economics"]["data"][0]["conditioning"] = "unconditional";
    }
    q = p;
    q["components"][0]["source"] = save(broken);
    if (which == 11)
      q["components"][0]["conditioning"] = "unconditional";
    q["on_unavailable"] = "unavailable";
    rejects(q);
  }
  q = p;
  q["components"][0]["source"]["expected_sha256"] = std::string(64, '0');
  rejects(q);
  q = p;
  q["components"][0]["signal_id"] = "wrong";
  rejects(q);
  q = p;
  q["components"][0]["conditioning"] = "unconditional";
  rejects(q);
  q = p;
  q["components"][0]["outcome_pointer"] = "/sections/execution/data/0";
  rejects(q);
  q = p;
  q["components"][0]["source"]["pointer"] = "/sections/economics/data/0";
  rejects(q);
  for (int which = 0; which < 8; ++which) {
    auto broken = a;
    auto &row = broken["sections"]["economics"]["data"][0];
    if (which == 0)
      row["atoms"][0]["return"] = rat("0");
    if (which == 1)
      row["atoms"][0]["weight"] = rat("1");
    if (which == 2)
      row["transformed_support"][0]["net_pnl"] = rat("0");
    if (which == 3)
      row["transform"]["return_basis"] = rat("0");
    if (which == 4)
      row["signal_id"] = "another";
    if (which == 5)
      broken["sections"]["execution"]["data"][0]["measure"] = "scenario_weight";
    if (which == 6)
      broken["sections"]["choices"]["data"]["selections"][0]["transform"]
            ["pnl_unit"] = "EUR";
    if (which == 7)
      broken["sections"]["signals"]["data"].push_back(
          broken["sections"]["signals"]["data"][0]);
    q = p;
    q["components"][0]["source"] = save(broken);
    q["on_unavailable"] = "unavailable";
    rejects(q);
  }
  q = p;
  q["dependence"]["kind"] = "default";
  rejects(q);
  q = p;
  q["conditioning_description"] = "";
  rejects(q);
  q = p;
  q["studies"] = J::array(
      {{{"id", "unknown"}, {"version", "1"}, {"parameters", J::object()}}});
  rejects(q);
  q = p;
  q["retain_paths"] = "false";
  rejects(q);
  rejects(p, e::unix_time_ms() - 1);

  // Zero-probability support remains, but inverse-CDF studies do not treat it
  // as a positive-probability tail.
  q = joint;
  q["dependence"]["marginal_policy"] = "support_only";
  q["dependence"]["paths"][0]["atom_indices"] = J::array({"0", "0"});
  q["dependence"]["paths"][0]["probability"] = rat("0");
  q["dependence"]["paths"][1]["probability"] = rat("1");
  q["studies"] = J::array();
  q["studies"].push_back(
      {{"id", "terminal_quantiles"},
       {"version", "1"},
       {"parameters", {{"levels", J::array({rat("0"), rat("1")})}}}});
  auto zero_path = call(q);
  check(data(zero_path, "distributions").at("terminal_atoms").size() == 2);
  check(data(zero_path, "distributions")
            .at("terminal_atoms")[0]
            .at("probability") == rat("0"));
  check(data(zero_path, "studies")[0].at("data").at("quantiles")[0].at(
            "state") == rat("9999", "10000"));
  check(data(zero_path, "studies")[0].at("data").at("quantiles")[1].at(
            "state") == rat("9999", "10000"));
  if (env) {
    d::create_file(path("independent-request.json"), p.dump(), e::no_deadline);
    d::create_file(path("joint-request.json"), joint.dump(), e::no_deadline);
  }
  std::cout << checks << " composition economics assertions passed\n";
  return 0;
} catch (const std::exception &error) {
  std::cerr << error.what() << '\n';
  return 1;
}
