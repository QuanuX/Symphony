#include <filesystem>
#include <fstream>
#include <iostream>
#include <limits>
#include <source_location>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <symphony/snv/sniv.hpp>
#include <symphony/snv/snrv.hpp>
#include <symphony/snv/snv.hpp>
namespace {
using symphony::snv::Json;
namespace snv = symphony::snv::snv;
namespace engine = symphony::knowledge::engine;
int checks = 0;
void check(bool condition, const char *message,
           const std::source_location where = std::source_location::current()) {
  ++checks;
  if (!condition)
    throw std::runtime_error(std::string(message) + " at line " +
                             std::to_string(where.line()));
}
template <class F>
void rejects(
    F f, const std::source_location where = std::source_location::current()) {
  bool failed = false;
  try {
    f();
  } catch (const engine::Error &) {
    failed = true;
  }
  check(failed, "expected rejection", where);
}
Json call(const char *op, const Json &input) {
  return snv::handle(op, input, std::numeric_limits<std::int64_t>::max());
}
Json inspect(Json bundle, std::string mode = "replay",
             std::int64_t offset = 0) {
  return {{"protocol", "symphony.snv.inspect-input.v1"},
          {"mode", mode},
          {"bundle", bundle},
          {"baseline", nullptr},
          {"offset", offset},
          {"limit", 64}};
}
Json file(const std::filesystem::path &path) {
  std::ifstream in(path);
  check(bool(in), "fixture unavailable");
  return engine::parse_bounded_json(engine::read_bounded(in, 1U << 20),
                                    1U << 20, symphony::snv::max_json_values);
}
Json artifact(const char *owner, const char *op, Json input) {
  const auto bytes = input.dump();
  return {{"owner", owner},
          {"owner_version", snv::version},
          {"operation", op},
          {"source_utf8", bytes},
          {"source_digest", engine::tagged_sha256(bytes)}};
}
bool finding(const Json &result, const char *code) {
  for (const auto &item : result.at("findings"))
    if (item.at("code") == code)
      return true;
  return false;
}
Json input_of(const Json &bundle, const char *owner) {
  for (const auto &a : bundle.at("artifacts"))
    if (a.at("owner") == owner)
      return engine::parse_bounded_json(a.at("source_utf8").get<std::string>(),
                                        65536, symphony::snv::max_json_values);
  throw std::runtime_error("owner fixture absent");
}
void replace_owner(Json &bundle, const char *owner, const char *op,
                   const Json &input) {
  for (auto &a : bundle.at("artifacts"))
    if (a.at("owner") == owner) {
      a = artifact(owner, op, input);
      return;
    }
  throw std::runtime_error("owner fixture absent");
}
} // namespace
int main() {
  try {
    const auto root = std::filesystem::path(__FILE__)
                          .parent_path()
                          .parent_path()
                          .parent_path();
    Json bundle = {{"protocol", "symphony.snv.bundle.v1"},
                   {"bundle_id", "test-original"},
                   {"artifacts", Json::array()},
                   {"relations", Json::array()}};
    auto partial = call("snv_inspect", inspect(bundle));
    check(partial.at("partial") == true, "empty coverage must be partial");
    const auto identity =
        file(root / "sniv-engine/tests/fixtures/identity_validate.json");
    const auto resources =
        file(root / "snrv-engine/tests/fixtures/resources_validate.json");
    bundle["artifacts"].push_back(
        artifact("sniv", "identity_validate", identity));
    bundle["artifacts"].push_back(
        artifact("snrv", "resources_validate", resources));
    auto replay = call("snv_inspect", inspect(bundle));
    check(replay.at("views").size() == 2, "replay both owners");
    check(replay == call("snv_inspect", inspect(bundle)),
          "deterministic native reconstruction");
    auto historical_resources = resources;
    auto inventory_revision = resources["inventories"][0];
    inventory_revision["record_id"] = "inventory-history";
    inventory_revision["generation"] = 2;
    inventory_revision["predecessor_record_id"] =
        resources["inventories"][0]["record_id"];
    historical_resources["inventories"].push_back(inventory_revision);
    auto historical_bundle = bundle;
    historical_bundle["artifacts"][1] =
        artifact("snrv", "resources_validate", historical_resources);
    check(call("snv_inspect", inspect(historical_bundle))
                  .at("subject_ids")
                  .at("snrv") == replay.at("subject_ids").at("snrv"),
          "historical instances do not duplicate projected subject identity");
    auto complete_input =
        file(root / "snv-engine/tests/fixtures/snv_inspect.json");
    auto complete = call("snv_inspect", complete_input);
    check(complete.at("views").size() == 4 && complete.at("partial") == false,
          "complete typed owner composition");
    auto naming_history = complete_input;
    naming_history["mode"] = "history";
    auto history = call("snv_inspect", naming_history);
    bool found_name = false;
    for (const auto &row : history.at("history"))
      if (row.at("owner") == "scnv" && row.at("kind") == "records")
        found_name = true;
    check(found_name, "original naming associations preserved in history");
    auto relation_name = complete_input;
    for (auto &a : relation_name["bundle"]["artifacts"])
      if (a["owner"] == "scnv") {
        auto names = engine::parse_bounded_json(
            a["source_utf8"].get<std::string>(), 1U << 20,
            symphony::snv::max_json_values);
        names["records"][0]["subject"] = {{"kind", "relationship"},
                                          {"id", "explicit-relation"}};
        a = artifact("scnv", "names_validate", names);
      }
    relation_name["bundle"]["relations"].push_back(
        {{"relationship_id", "explicit-relation"},
         {"kind", "caller-relation"},
         {"from", {{"owner", "sniv"}, {"kind", "node"}, {"id", "node-a"}}},
         {"to", {{"owner", "sniv"}, {"kind", "node"}, {"id", "node-b"}}},
         {"source_ref", "supplied-relation"}});
    check(call("snv_inspect", relation_name).at("partial") == false,
          "name resolves an independently supplied relationship");
    relation_name["bundle"]["relations"] = Json::array();
    check(call("snv_inspect", relation_name).at("partial") == true,
          "a name does not establish its relationship subject");
    auto wrong_type = complete_input;
    wrong_type["bundle"]["relations"].push_back(
        {{"relationship_id", "wrong-kind"},
         {"kind", "test"},
         {"from", {{"owner", "sniv"}, {"kind", "resource"}, {"id", "node-a"}}},
         {"to", {{"owner", "snrv"}, {"kind", "resource"}, {"id", "memory-1"}}},
         {"source_ref", "test-source"}});
    rejects([&] { call("snv_inspect", wrong_type); });
    auto identity_change = identity, resource_change = resources;
    identity_change["mode"] = "transition";
    auto next_identity = identity["physical_records"][0];
    next_identity["record_id"] = "physical-successor";
    next_identity["generation"] = 2;
    next_identity["predecessor_record_id"] =
        identity["physical_records"][0]["record_id"];
    next_identity["physical_node_id"] = "node-successor";
    next_identity["predecessor_node_id"] = "node-1";
    identity_change["transition"] = {
        {"kind", "physical_change"},
        {"predecessor_record_id", identity["physical_records"][0]["record_id"]},
        {"candidate_record", next_identity},
        {"change_kind", "hardware_change"},
        {"materiality_profile",
         {{"profile_id", "caller-hardware"},
          {"version", "1"},
          {"rules", Json::array({Json{{"component_kind", "memory"},
                                      {"change", "replaced"},
                                      {"decision", "material"}}})},
          {"unknown_decision", "unresolved"}}},
        {"changes",
         Json::array(
             {Json{{"component_kind", "memory"},
                   {"change", "replaced"},
                   {"locality", "local"},
                   {"evidence_refs",
                    Json::array({"inventory-1", "inventory-successor"})}}})}};
    resource_change["mode"] = "transition";
    auto next_inventory = resources["inventories"][0];
    next_inventory["record_id"] = "inventory-successor";
    next_inventory["generation"] = 2;
    next_inventory["predecessor_record_id"] = "inventory-1";
    next_inventory["physical_node_id"] = "node-successor";
    next_inventory["identity_rebind"] = {
        {"sniv_record_ref", "physical-successor"},
        {"predecessor_node_id", "node-1"},
        {"successor_node_id", "node-successor"}};
    next_inventory["local_resources"][0]["instance_ref"] = "dimm-successor";
    resource_change["transition"] = {{"kind", "resource_change"},
                                     {"predecessor_record_id", "inventory-1"},
                                     {"candidate_record", next_inventory}};
    auto compound = bundle;
    compound["artifacts"] =
        Json::array({artifact("sniv", "identity_validate", identity_change),
                     artifact("snrv", "resources_validate", resource_change)});
    auto handoff = call("snv_inspect", inspect(compound));
    bool verified = false;
    for (const auto &f : handoff["findings"])
      if (f["code"] == "verified_resource_identity_handoff")
        verified = true;
    check(verified, "exact two-owner materiality handoff");
    auto wrong_handoff = identity_change;
    wrong_handoff["transition"]["changes"][0]["evidence_refs"] =
        Json::array({"wrong-before", "wrong-after"});
    compound["artifacts"][0] =
        artifact("sniv", "identity_validate", wrong_handoff);
    rejects([&] { call("snv_inspect", inspect(compound)); });
    for (const auto *kind : {"replacement", "provider_resource"}) {
      auto boundary = identity_change;
      boundary["transition"]["change_kind"] = kind;
      boundary["transition"]["materiality_profile"] = nullptr;
      boundary["transition"]["changes"] = Json::array();
      if (std::string(kind) == "provider_resource")
        boundary["transition"]["candidate_record"]["provider_resource"]
                ["native_id"] = "provider-successor";
      check(
          symphony::snv::sniv::handle("identity_validate", boundary,
                                      std::numeric_limits<std::int64_t>::max())
                  .at("status") == "valid",
          "categorical physical boundary independently admitted");
      auto categorical = compound;
      categorical["artifacts"][0] =
          artifact("sniv", "identity_validate", boundary);
      check(finding(call("snv_inspect", inspect(categorical)),
                    "verified_resource_identity_handoff"),
            "categorical whole-resource boundary needs no hardware-materiality "
            "rule");
    }
    auto retired_identity = identity_change,
         retired_resources = resource_change;
    auto &retired_node = retired_identity["transition"]["candidate_record"];
    retired_node["physical_node_id"] = "node-1";
    retired_node["predecessor_node_id"] = nullptr;
    retired_node["status"] = "retired";
    retired_identity["transition"]["change_kind"] = "retirement";
    retired_identity["transition"]["materiality_profile"] = nullptr;
    retired_identity["transition"]["changes"] = Json::array();
    auto &retired_inventory =
        retired_resources["transition"]["candidate_record"];
    retired_resources["transition"]["kind"] = "retirement";
    retired_inventory["physical_node_id"] = "node-1";
    retired_inventory["identity_rebind"] = nullptr;
    retired_inventory["status"] = "retired";
    auto &removed_resource = retired_inventory["local_resources"][0];
    removed_resource["presence"] = "absent";
    removed_resource["capacity"] = nullptr;
    removed_resource["available_capacity"] = nullptr;
    removed_resource["operational_state"] = "unavailable";
    check(symphony::snv::snrv::handle("resources_validate", retired_resources,
                                      std::numeric_limits<std::int64_t>::max())
                  .at("status") == "valid",
          "categorical removal independently admitted");
    auto retirement_bundle = compound;
    retirement_bundle["artifacts"][0] =
        artifact("sniv", "identity_validate", retired_identity);
    retirement_bundle["artifacts"][1] =
        artifact("snrv", "resources_validate", retired_resources);
    check(finding(call("snv_inspect", inspect(retirement_bundle)),
                  "verified_resource_identity_handoff"),
          "retirement/removal keeps its own categorical boundary");
    auto false_retirement = retired_resources;
    false_retirement["transition"]["candidate_record"]["local_resources"] =
        resource_change["transition"]["candidate_record"]["local_resources"];
    retirement_bundle["artifacts"][1] =
        artifact("snrv", "resources_validate", false_retirement);
    rejects([&] { call("snv_inspect", inspect(retirement_bundle)); });
    auto historical_alias = identity_change;
    auto alias_record = identity_change["transition"]["candidate_record"];
    alias_record["record_id"] = "historical-alias";
    alias_record["generation"] = 1;
    alias_record["predecessor_record_id"] = nullptr;
    alias_record["provider_resource"]["native_id"] =
        "historical-provider-alias";
    historical_alias["physical_records"].push_back(alias_record);
    check(
        symphony::snv::sniv::handle("identity_validate", historical_alias,
                                    std::numeric_limits<std::int64_t>::max())
                .at("status") == "valid",
        "historical alias fixture independently admits current SNIV proposal");
    auto wrong_binding = resource_change;
    wrong_binding["transition"]["candidate_record"]["identity_rebind"]
                 ["sniv_record_ref"] = "historical-alias";
    auto wrong_binding_bundle = compound;
    wrong_binding_bundle["artifacts"][0] =
        artifact("sniv", "identity_validate", historical_alias);
    wrong_binding_bundle["artifacts"][1] =
        artifact("snrv", "resources_validate", wrong_binding);
    rejects([&] { call("snv_inspect", inspect(wrong_binding_bundle)); });
    auto proposed_name =
        file(root / "scnv-engine/tests/fixtures/names_assign.json");
    proposed_name["intent"]["new_record"]["subject"]["id"] = "unobserved-node";
    proposed_name["intent"]["new_record"]["scope"]["id"] = "cluster-1";
    auto naming_candidate = complete_input;
    replace_owner(naming_candidate["bundle"], "scnv", "names_validate",
                  proposed_name);
    check(finding(call("snv_inspect", naming_candidate),
                  "unresolved_subject_reference"),
          "proposed naming association does not establish its missing Node");
    proposed_name["intent"]["new_record"]["subject"]["id"] = "node-a";
    replace_owner(naming_candidate["bundle"], "scnv", "names_validate",
                  proposed_name);
    naming_candidate["bundle"]["relations"].push_back(
        {{"relationship_id", "proposed-name-reference"},
         {"kind", "caller-relation"},
         {"from",
          {{"owner", "scnv"},
           {"kind", "name_association"},
           {"id", "association-annex"}}},
         {"to", {{"owner", "sniv"}, {"kind", "node"}, {"id", "node-a"}}},
         {"source_ref", "caller-reference"}});
    check(call("snv_inspect", naming_candidate).at("partial") == false,
          "admitted proposed naming association is typed-resolvable");
    auto proposed_identity = input_of(complete_input.at("bundle"), "sniv");
    auto proposed_episode = proposed_identity["participations"][0];
    proposed_episode["record_id"] = "participation-proposed";
    proposed_episode["incarnation_id"] = "incarnation-proposed";
    proposed_episode["system_id"] = "research-2";
    proposed_identity["mode"] = "transition";
    proposed_identity["transition"] = {{"kind", "participation"},
                                       {"predecessor_record_id", nullptr},
                                       {"candidate_record", proposed_episode},
                                       {"action", "establish"}};
    auto wrong_cluster = input_of(complete_input.at("bundle"), "sciv");
    wrong_cluster["nodes"][1]["incarnation_id"] = "incarnation-proposed";
    wrong_cluster["memberships"][1]["incarnation_id"] = "incarnation-proposed";
    auto incarnation_candidate = complete_input;
    replace_owner(incarnation_candidate["bundle"], "sniv", "identity_validate",
                  proposed_identity);
    replace_owner(incarnation_candidate["bundle"], "sciv", "sciv_validate",
                  wrong_cluster);
    rejects([&] { call("snv_inspect", incarnation_candidate); });
    auto matching_cluster = wrong_cluster;
    matching_cluster["cluster"]["system_id"] = "research-2";
    matching_cluster["nodes"] = Json::array({wrong_cluster["nodes"][0]});
    matching_cluster["nodes"][0]["system_id"] = "research-2";
    matching_cluster["nodes"][0]["incarnation_id"] = "incarnation-proposed";
    matching_cluster["memberships"] =
        Json::array({wrong_cluster["memberships"][0]});
    matching_cluster["memberships"][0]["incarnation_id"] =
        "incarnation-proposed";
    matching_cluster["observations"] = Json::array();
    replace_owner(incarnation_candidate["bundle"], "sciv", "sciv_validate",
                  matching_cluster);
    check(call("snv_inspect", incarnation_candidate).at("partial") == false,
          "proposed incarnation binds its exact Node and System");
    auto rejected_identity = input_of(complete_input.at("bundle"), "sniv");
    auto rejected_episode = rejected_identity["participations"][0];
    rejected_episode["record_id"] = "participation-rejected";
    rejected_episode["incarnation_id"] = "incarnation-rejected";
    rejected_identity["mode"] = "transition";
    rejected_identity["transition"] = {{"kind", "participation"},
                                       {"predecessor_record_id", nullptr},
                                       {"candidate_record", rejected_episode},
                                       {"action", "establish"}};
    const auto rejected_owner =
        symphony::snv::sniv::handle("identity_validate", rejected_identity,
                                    std::numeric_limits<std::int64_t>::max());
    check(rejected_owner.at("status") == "conflicting" &&
              rejected_owner.at("proposed_records").empty(),
          "independent conflicting incarnation proposal is not admitted");
    auto rejected_cluster = input_of(complete_input.at("bundle"), "sciv");
    rejected_cluster["nodes"][0]["incarnation_id"] = "incarnation-rejected";
    rejected_cluster["memberships"][0]["incarnation_id"] =
        "incarnation-rejected";
    auto rejected_candidate = complete_input;
    replace_owner(rejected_candidate["bundle"], "sniv", "identity_validate",
                  rejected_identity);
    replace_owner(rejected_candidate["bundle"], "sciv", "sciv_validate",
                  rejected_cluster);
    check(finding(call("snv_inspect", rejected_candidate),
                  "unresolved_subject_reference"),
          "supplied rejected candidate census does not authorize typed "
          "reference");
    const auto base_cluster = input_of(complete_input.at("bundle"), "sciv");
    auto proposed_connection = base_cluster["observations"][0];
    proposed_connection["record_id"] = "connection-proposed";
    proposed_connection["bus_id"] = "bus-proposed";
    proposed_connection["observed_unix_ms"] = 10001;
    Json cluster_transition = {
        {"protocol", "symphony.snv.sciv.transition.v1"},
        {"evidence", base_cluster},
        {"expected_source_digest", engine::tagged_sha256(base_cluster.dump())},
        {"change", {{"kind", "connection"}, {"record", proposed_connection}}}};
    auto cluster_candidate = complete_input;
    replace_owner(cluster_candidate["bundle"], "sciv", "sciv_transition",
                  cluster_transition);
    cluster_candidate["bundle"]["relations"].push_back(
        {{"relationship_id", "candidate-fabric"},
         {"kind", "caller-relation"},
         {"from", {{"owner", "sciv"}, {"kind", "bus"}, {"id", "bus-proposed"}}},
         {"to", {{"owner", "sciv"}, {"kind", "cluster"}, {"id", "cluster-1"}}},
         {"source_ref", "caller-fabric"}});
    check(call("snv_inspect", cluster_candidate).at("partial") == false,
          "admitted candidate Bus is typed-resolvable");
    auto connection_name = input_of(cluster_candidate.at("bundle"), "scnv");
    connection_name["records"][0]["subject"] = {{"kind", "relationship"},
                                                {"id", "connection-proposed"}};
    replace_owner(cluster_candidate["bundle"], "scnv", "names_validate",
                  connection_name);
    check(call("snv_inspect", cluster_candidate).at("partial") == false,
          "name binds admitted candidate connection relationship");
    cluster_candidate["mode"] = "history";
    const auto candidate_history = call("snv_inspect", cluster_candidate);
    bool found_connection = false;
    for (const auto &row : candidate_history.at("history"))
      if (row.at("record").contains("record_id") &&
          row.at("record").at("record_id") == "connection-proposed")
        found_connection = row.at("kind") == "proposal";
    check(found_connection, "SCIV candidate proposal is preserved in history");
    auto inventory_addition = resource_change,
         classified_addition = identity_change;
    auto &addition_identity =
        classified_addition["transition"]["candidate_record"];
    addition_identity["physical_node_id"] = "node-1";
    addition_identity["predecessor_node_id"] = nullptr;
    classified_addition["transition"]["materiality_profile"]["rules"] =
        Json::array({Json{{"component_kind", "cpu"},
                          {"change", "added"},
                          {"decision", "non_material"}}});
    classified_addition["transition"]["changes"] = Json::array(
        {Json{{"component_kind", "cpu"},
              {"change", "added"},
              {"locality", "local"},
              {"evidence_refs",
               Json::array({"inventory-1", "inventory-successor"})}}});
    auto &addition_inventory =
        inventory_addition["transition"]["candidate_record"];
    addition_inventory["physical_node_id"] = "node-1";
    addition_inventory["identity_rebind"] = nullptr;
    addition_inventory["local_resources"] =
        resources["inventories"][0]["local_resources"];
    auto cpu = addition_inventory["local_resources"][0];
    cpu["resource_id"] = "cpu-proposed";
    cpu["component_kind"] = "cpu";
    cpu["instance_ref"] = "cpu-instance";
    cpu["capacity"] = {{"unit", "count"}, {"value", "1"}};
    cpu["available_capacity"] = cpu["capacity"];
    addition_inventory["local_resources"].push_back(cpu);
    auto resource_candidate = compound;
    resource_candidate["artifacts"][0] =
        artifact("sniv", "identity_validate", classified_addition);
    resource_candidate["artifacts"][1] =
        artifact("snrv", "resources_validate", inventory_addition);
    resource_candidate["relations"].push_back(
        {{"relationship_id", "candidate-resource"},
         {"kind", "caller-relation"},
         {"from",
          {{"owner", "snrv"}, {"kind", "resource"}, {"id", "cpu-proposed"}}},
         {"to", {{"owner", "sniv"}, {"kind", "node"}, {"id", "node-1"}}},
         {"source_ref", "caller-resource"}});
    const auto resource_projection =
        call("snv_inspect", inspect(resource_candidate));
    bool found_cpu = false;
    for (const auto &id : resource_projection.at("subject_ids").at("snrv"))
      if (id == "cpu-proposed")
        found_cpu = true;
    check(found_cpu &&
              !finding(resource_projection, "unresolved_subject_reference"),
          "admitted proposed resource is projected and typed-resolvable");
    check(resource_projection.at("views").at("snrv").at("source_digest") ==
              engine::tagged_sha256(inventory_addition.dump()),
          "candidate resource keeps independent original owner source binding");
    auto retained_identity = identity, retained_resources = resources;
    retained_identity["physical_records"].push_back(
        handoff["views"]["sniv"]["proposed_records"][0]);
    retained_resources["inventories"].push_back(
        handoff["views"]["snrv"]["proposed_records"][0]);
    auto availability = retained_resources["inventories"][1];
    availability["record_id"] = "inventory-availability";
    availability["generation"] = 3;
    availability["predecessor_record_id"] = "inventory-successor";
    availability["local_resources"][0]["operational_state"] = "unavailable";
    retained_resources["mode"] = "transition";
    retained_resources["transition"] = {
        {"kind", "correction"},
        {"predecessor_record_id", "inventory-successor"},
        {"candidate_record", availability}};
    auto later_bundle = compound;
    later_bundle["artifacts"][0] =
        artifact("sniv", "identity_validate", retained_identity);
    later_bundle["artifacts"][1] =
        artifact("snrv", "resources_validate", retained_resources);
    const auto later = call("snv_inspect", inspect(later_bundle));
    check(later.at("views").at("snrv").at("identity_handoff").at("required") ==
                  false &&
              !finding(later, "insufficient_identity_handoff"),
          "retained rebind needs no repeated handoff for availability "
          "correction");
    check(later.at("views").at("snrv").at("proposed_records")[0].at(
              "identity_rebind") == availability.at("identity_rebind"),
          "retained rebind source survives parent history replay");
    auto original = bundle;
    bundle["artifacts"][0]["source_utf8"] = "  " + identity.dump() + "\n";
    bundle["artifacts"][0]["source_digest"] = engine::tagged_sha256(
        bundle["artifacts"][0]["source_utf8"].get<std::string>());
    check(call("snv_inspect", inspect(bundle)).at("views").at("sniv") ==
              replay.at("views").at("sniv"),
          "original byte representation preserved separately from semantics");
    auto tampered = bundle;
    tampered["artifacts"][0]["source_digest"] = "sha256:bad";
    rejects([&] { call("snv_inspect", inspect(tampered)); });
    tampered = bundle;
    tampered["artifacts"][0]["owner_version"] = "0.2.0-dev";
    rejects([&] { call("snv_inspect", inspect(tampered)); });
    tampered = bundle;
    tampered["artifacts"][0]["result"] = replay;
    rejects([&] { call("snv_inspect", inspect(tampered)); });
    tampered = bundle;
    tampered["artifacts"].push_back(tampered["artifacts"][0]);
    rejects([&] { call("snv_inspect", inspect(tampered)); });
    auto difference = inspect(bundle, "diff");
    difference["baseline"] = original;
    check(call("snv_inspect", difference).at("changes").empty(),
          "same semantic source projection");
    check(call("snv_inspect", inspect(bundle, "history")).at("total") > 0,
          "history retained");
    const auto manifest =
        call("snv_inspect", inspect(bundle, "export_manifest"))
            .at("export_manifest");
    Json chunks = Json::array();
    for (std::size_t i = 0; i < manifest.at("chunks").size(); ++i)
      chunks.push_back(
          call("snv_inspect",
               inspect(bundle, "export_chunk", static_cast<std::int64_t>(i)))
              .at("export_chunk"));
    Json prepare = {
        {"protocol", "symphony.snv.evidence-plan-input.v1"},
        {"operation_id", "retain-test"},
        {"mode", "import"},
        {"bundle", nullptr},
        {"export_records", {{"manifest", manifest}, {"chunks", chunks}}}};
    check(call("snv_evidence_plan", prepare).at("bundle") == bundle,
          "lossless independent import");
    auto invalid = prepare;
    invalid["export_records"]["chunks"].push_back(chunks[0]);
    rejects([&] { call("snv_evidence_plan", invalid); });
    invalid = prepare;
    invalid["export_records"]["chunks"][0]["hex"] = "00";
    rejects([&] { call("snv_evidence_plan", invalid); });
    invalid = prepare;
    invalid["export_records"]["chunks"] = Json::array();
    rejects([&] { call("snv_evidence_plan", invalid); });
    Json plan_input = {
        {"protocol", "symphony.snv.state-plan-input.v1"},
        {"view", {{"tops_id", "test-tops"}, {"view_id", "test-view"}}},
        {"operation_id", "select-1"},
        {"expected_state_digest", nullptr},
        {"prior_head", nullptr},
        {"change_kind", "select"},
        {"bundle", bundle},
        {"reason", "explicit test"},
        {"migration", nullptr}};
    const auto plan = call("snv_state_plan", plan_input);
    check(plan.at("head").at("generation") == 1, "initial generation");
    auto transition = call("snv_state_reduce",
                           {{"protocol", "symphony.snv.state-reduce-input.v1"},
                            {"input", plan_input},
                            {"plan", plan}});
    check(transition.at("effect") == "proposed_only",
          "native planning has no write");
    auto migration_input = plan_input;
    migration_input["prior_head"] = plan.at("head");
    migration_input["expected_state_digest"] = plan.at("head").at("digest");
    migration_input["operation_id"] = "migration-candidate";
    migration_input["bundle"]["bundle_id"] = "explicit-migration-candidate";
    migration_input["migration"] = {
        {"from_owner_version", snv::version},
        {"to_owner_version", snv::version},
        {"source_bundle_digest", plan.at("head").at("bundle_digest")},
        {"method", "caller-selected-migration"},
        {"method_version", "1"}};
    const auto migration_plan = call("snv_state_plan", migration_input);
    check(migration_plan.at("migration") == migration_input.at("migration") &&
              migration_plan.at("head").at("bundle_digest") ==
                  engine::tagged_sha256(migration_input.at("bundle").dump()) &&
              migration_plan.at("head").at("previous_digest") ==
                  plan.at("head").at("digest"),
          "explicit admitted-writer migration preserves predecessor and "
          "proposes the supplied candidate");
    check(call("snv_state_reduce",
               {{"protocol", "symphony.snv.state-reduce-input.v1"},
                {"input", migration_input},
                {"plan", migration_plan}})
                  .at("effect") == "proposed_only",
          "migration is pure and never auto-selected");
    for (const auto *field :
         {"from_owner_version", "to_owner_version", "source_bundle_digest"}) {
      auto wrong = migration_input;
      wrong["migration"][field] = "wrong";
      rejects([&] { call("snv_state_plan", wrong); });
    }
    auto missing_prior = migration_input;
    missing_prior["prior_head"] = nullptr;
    missing_prior["expected_state_digest"] = nullptr;
    rejects([&] { call("snv_state_plan", missing_prior); });
    tampered = plan;
    tampered["head"]["bundle_digest"] = "wrong";
    rejects([&] {
      call("snv_state_reduce",
           {{"protocol", "symphony.snv.state-reduce-input.v1"},
            {"input", plan_input},
            {"plan", tampered}});
    });
    plan_input["prior_head"] = plan.at("head");
    plan_input["expected_state_digest"] = plan.at("head").at("digest");
    plan_input["operation_id"] = "unselect-2";
    plan_input["change_kind"] = "unselect";
    plan_input["bundle"] = nullptr;
    const auto unselect = call("snv_state_plan", plan_input);
    check(unselect.at("head").at("tombstone") == true &&
              unselect.at("head").at("generation") == 2,
          "explicit tombstone with history");
    invalid = plan_input;
    invalid["expected_state_digest"] = "stale";
    rejects([&] { call("snv_state_plan", invalid); });
    rejects([&] { snv::handle("snv_inspect", inspect(bundle), 0); });
    std::cout << "SNV parent " << checks << " checks passed\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
