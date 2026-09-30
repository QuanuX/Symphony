#include "fixture.hpp"
#include <functional>
#include <iostream>
#include <limits>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/snv/sniv.hpp>
using symphony::snv::Json;
namespace {
std::size_t checks = 0;
void expect(bool value, const char *message) {
  ++checks;
  if (!value)
    throw std::runtime_error(message);
}
Json run(const Json &input) {
  return symphony::snv::sniv::handle("identity_validate", input,
                                     std::numeric_limits<std::int64_t>::max());
}
void rejected(const Json &input) {
  ++checks;
  try {
    (void)run(input);
  } catch (const symphony::knowledge::engine::Error &e) {
    if (e.code() == "invalid_input" && e.exit_status() == 2)
      return;
    throw;
  }
  throw std::runtime_error("Expected invalid-input rejection");
}
Json physical_change(std::string kind) {
  auto j = fixture();
  auto next = j["physical_records"][0];
  next["record_id"] = "physical-record-2";
  next["predecessor_record_id"] = "physical-record-1";
  next["generation"] = 2;
  j["mode"] = "transition";
  j["transition"] = {{"kind", "physical_change"},
                     {"predecessor_record_id", "physical-record-1"},
                     {"candidate_record", next},
                     {"change_kind", kind},
                     {"materiality_profile", nullptr},
                     {"changes", Json::array()}};
  return j;
}
Json participation(std::string action) {
  auto j = fixture();
  auto next = j["participations"][0];
  next["record_id"] = "participation-2";
  next["predecessor_record_id"] = "participation-1";
  next["generation"] = 2;
  next["last_event"] = action;
  j["mode"] = "transition";
  j["transition"] = {{"kind", "participation"},
                     {"predecessor_record_id", "participation-1"},
                     {"candidate_record", next},
                     {"action", action}};
  return j;
}
} // namespace
int main() {
  try {
    auto j = fixture();
    auto result = run(j);
    expect(result["status"] == "valid", "Valid identity snapshot");
    expect(result["subject_ids"] == Json::array({"node-1"}),
           "Physical subject binding");
    expect(result["source_digest"].get<std::string>().starts_with("sha256:"),
           "Source digest bound");
    for (const auto &kind : {"unchanged", "software_only", "remote_only"}) {
      auto t = physical_change(kind);
      auto r = run(t);
      expect(r["status"] == "valid" &&
                 r["proposed_records"][0]["physical_node_id"] == "node-1",
             "Software/remote continuity");
    }
    auto replacement = physical_change("replacement");
    replacement["transition"]["candidate_record"]["physical_node_id"] =
        "node-2";
    replacement["transition"]["candidate_record"]["predecessor_node_id"] =
        "node-1";
    expect(run(replacement)["proposed_records"][0]["physical_node_id"] ==
               "node-2",
           "Supplied successor admitted");
    auto missing = replacement;
    missing["transition"]["candidate_record"]["physical_node_id"] = nullptr;
    expect(run(missing)["status"] == "insufficient" &&
               run(missing)["proposed_records"].empty(),
           "Missing successor not invented");
    auto reuse = physical_change("replacement");
    expect(run(reuse)["status"] == "conflicting",
           "Replacement cannot reuse physical ID");
    auto provider = replacement;
    provider["transition"]["change_kind"] = "provider_resource";
    provider["transition"]["candidate_record"]["provider_resource"]
            ["native_id"] = "resource-2";
    expect(run(provider)["status"] == "valid",
           "Separate established provider resource");
    auto unestablished = provider;
    unestablished["physical_records"][0]["provider_resource"]["established"] =
        false;
    expect(
        run(unestablished)["status"] == "insufficient",
        "Unconfirmed predecessor provider association cannot prove continuity");
    provider["transition"]["candidate_record"]["physical_node_id"] = "node-1";
    expect(run(provider)["status"] == "conflicting",
           "Provider replacement requires new supplied Node");
    auto hardware = replacement;
    hardware["transition"]["change_kind"] = "hardware_change";
    hardware["transition"]["materiality_profile"] = {
        {"profile_id", "my-rules"},
        {"version", "2"},
        {"rules", Json::array({{{"component_kind", "memory"},
                                {"change", "capacity_changed"},
                                {"decision", "material"}}})},
        {"unknown_decision", "unresolved"}};
    hardware["transition"]["changes"] = Json::array(
        {{{"component_kind", "memory"},
          {"change", "capacity_changed"},
          {"locality", "local"},
          {"evidence_refs", Json::array({"inventory-1", "inventory-2"})}}});
    expect(run(hardware)["status"] == "valid", "Versioned user materiality");
    hardware["transition"]["materiality_profile"]["rules"][0]["decision"] =
        "non_material";
    expect(run(hardware)["status"] == "conflicting",
           "Nonmaterial profile preserves identity");
    hardware["transition"]["materiality_profile"]["rules"] = Json::array();
    expect(run(hardware)["status"] == "insufficient",
           "Unlisted local change unresolved");
    hardware["transition"]["changes"][0]["locality"] = "remote";
    hardware["transition"]["candidate_record"]["physical_node_id"] = "node-1";
    hardware["transition"]["candidate_record"]["predecessor_node_id"] = nullptr;
    expect(run(hardware)["status"] == "valid",
           "Remote change never material local change");
    auto conflict = fixture();
    auto a = conflict["physical_records"][0]["assertions"][0];
    a["value"] = "contradictory-serial";
    conflict["physical_records"][0]["assertions"].push_back(a);
    expect(run(conflict)["status"] == "conflicting",
           "Contradictory identity preserved");
    auto theoretical = fixture();
    theoretical["physical_records"][0]["physical_node_id"] = nullptr;
    expect(run(theoretical)["status"] == "insufficient",
           "Unidentified resource unresolved");
    for (const auto &action : {"disconnect", "reconnect", "boot_change"}) {
      auto t = participation(action);
      if (std::string(action) == "boot_change")
        t["transition"]["candidate_record"]["boot_id"] = "boot-2";
      auto r = run(t);
      expect(r["status"] == "valid" &&
                 r["proposed_records"][0]["incarnation_id"] == "incarnation-1",
             "Temporary disconnect/restart preserve episode");
    }
    auto ended = participation("end");
    ended["transition"]["candidate_record"]["state"] = "ended";
    auto end_result = run(ended);
    expect(end_result["status"] == "valid", "Explicit episode end");
    auto rejoin = fixture();
    rejoin["participations"].push_back(end_result["proposed_records"][0]);
    auto next = end_result["proposed_records"][0];
    next["record_id"] = "participation-3";
    next["predecessor_record_id"] = "participation-2";
    next["generation"] = 3;
    next["incarnation_id"] = "incarnation-2";
    next["state"] = "active";
    next["last_event"] = "rejoin";
    rejoin["mode"] = "transition";
    rejoin["transition"] = {{"kind", "participation"},
                            {"predecessor_record_id", "participation-2"},
                            {"candidate_record", next},
                            {"action", "rejoin"}};
    expect(run(rejoin)["status"] == "valid",
           "Explicit rejoin creates separate supplied episode");
    rejoin["transition"]["candidate_record"]["incarnation_id"] =
        "incarnation-1";
    rejected(rejoin);
    auto multi = fixture();
    auto episode = multi["participations"][0];
    episode["record_id"] = "participation-other";
    episode["incarnation_id"] = "incarnation-other";
    episode["system_id"] = "other-system";
    multi["participations"].push_back(episode);
    expect(run(multi)["status"] == "valid",
           "Same physical Node in separate systems");
    auto alias = multi;
    alias["participations"][1]["incarnation_id"] = "incarnation-1";
    expect(run(alias)["status"] == "conflicting",
           "One incarnation cannot alias different systems");
    alias["participations"][1]["system_id"] = "system-1";
    alias["participations"][1]["physical_node_id"] = "other-node";
    expect(run(alias)["status"] == "conflicting",
           "One incarnation cannot alias different physical Nodes");
    multi["participations"][1]["system_id"] = "system-1";
    expect(run(multi)["status"] == "conflicting",
           "Overlapping active episodes in same system visible");
    auto retire = physical_change("retirement");
    retire["transition"]["candidate_record"]["status"] = "retired";
    expect(run(retire)["status"] == "valid", "Explicit retirement");
    auto invalid_reconnect = fixture();
    invalid_reconnect["participations"].push_back(
        end_result["proposed_records"][0]);
    auto resumed = end_result["proposed_records"][0];
    resumed["record_id"] = "participation-bad";
    resumed["predecessor_record_id"] = "participation-2";
    resumed["generation"] = 3;
    resumed["state"] = "active";
    resumed["last_event"] = "reconnect";
    invalid_reconnect["participations"].push_back(resumed);
    rejected(invalid_reconnect);
    auto fork = fixture();
    auto branch = run(physical_change("software_only"))["proposed_records"][0];
    fork["physical_records"].push_back(branch);
    branch["record_id"] = "physical-branch";
    fork["physical_records"].push_back(branch);
    expect(run(fork)["status"] == "conflicting",
           "Competing causal successors remain conflict");
    auto bad = fixture();
    bad["extra"] = true;
    rejected(bad);
    bad = fixture();
    bad["physical_records"][0]["generation"] = 1.0;
    rejected(bad);
    bad = fixture();
    bad["physical_records"][0]["sources"][0]["observed_at"] =
        "2026-02-30T00:00:00Z";
    rejected(bad);
    bad = fixture();
    bad["physical_records"][0]["assertions"][0]["source_id"] = "missing";
    rejected(bad);
    bad = fixture();
    bad["physical_records"][0]["record_id"] = std::string("bad\0id", 6);
    rejected(bad);
    bad = fixture();
    bad["physical_records"].push_back(bad["physical_records"][0]);
    rejected(bad);
    try {
      (void)symphony::snv::sniv::handle("identity_validate", fixture(), -1);
      throw std::runtime_error("Deadline not rejected");
    } catch (const symphony::knowledge::engine::Error &e) {
      expect(e.code() == "deadline_exceeded" && e.exit_status() == 3,
             "Deadline error and process status classification");
    }
    for (int limit : {512, 1024, 2048}) {
      auto capacity = fixture();
      capacity["record_limit"] = limit;
      capacity["participations"] = Json::array();
      auto base = capacity["physical_records"][0];
      capacity["physical_records"] = Json::array();
      for (int n = 0; n < limit; ++n) {
        auto r = base;
        r["record_id"] = "record-" + std::to_string(n);
        r["physical_node_id"] = "node-" + std::to_string(n);
        r["provider_resource"]["native_id"] = "resource-" + std::to_string(n);
        capacity["physical_records"].push_back(r);
      }
      expect(run(capacity)["subject_ids"].size() ==
                 static_cast<std::size_t>(limit),
             "Selected record capacity exact");
      capacity["physical_records"].push_back(base);
      rejected(capacity);
    }
    std::cout << "SNIV " << checks << " independent checks passed\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
