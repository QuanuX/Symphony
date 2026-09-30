#include "fixture.hpp"
#include <iostream>
#include <limits>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/snv/snrv.hpp>
using symphony::snv::Json;
namespace {
std::size_t checks = 0;
void expect(bool value, const char *message) {
  ++checks;
  if (!value)
    throw std::runtime_error(message);
}
Json run(const Json &j) {
  return symphony::snv::snrv::handle("resources_validate", j,
                                     std::numeric_limits<std::int64_t>::max());
}
void rejected(const Json &j) {
  ++checks;
  try {
    (void)run(j);
  } catch (const symphony::knowledge::engine::Error &e) {
    if (e.code() == "invalid_input" && e.exit_status() == 2)
      return;
    throw;
  }
  throw std::runtime_error("Expected invalid-input rejection");
}
Json transition(std::string kind = "resource_change") {
  auto j = fixture();
  auto next = j["inventories"][0];
  next["record_id"] = "inventory-2";
  next["predecessor_record_id"] = "inventory-1";
  next["generation"] = 2;
  j["mode"] = "transition";
  j["transition"] = {{"kind", kind},
                     {"predecessor_record_id", "inventory-1"},
                     {"candidate_record", next}};
  return j;
}
} // namespace
int main() {
  try {
    auto j = fixture();
    auto r = run(j);
    expect(r["status"] == "valid", "Valid inventory");
    expect(r["records"][0]["local_resources"][0]["capacity"]["value"] ==
               "18446744073709551615",
           "Full uint64 exact string");
    expect(
        r["records"][0]["local_resources"][0]["available_capacity"]["value"] ==
            "9007199254740993",
        "Beyond JSON safe integer preserved");
    expect(r["records"][0]["remote_attachments"].size() == 1,
           "Remote attachment preserved separately");
    auto t = transition();
    t["transition"]["candidate_record"]["local_resources"][0]["capacity"]
     ["value"] = "18446744073709551614";
    auto diff = run(t);
    expect(diff["resource_changes"][0]["change"] == "capacity_changed" &&
               diff["identity_handoff"]["required"] == true,
           "Physical local capacity change handed to SNIV");
    expect(diff["identity_handoff"]["physical_identity_changed"] == false,
           "SNRV does not mutate physical identity");
    auto available = transition();
    available["transition"]["candidate_record"]["local_resources"][0]
             ["available_capacity"]["value"] = "0";
    expect(run(available)["identity_handoff"]["required"] == false &&
               run(available)["resource_changes"][0]["change"] ==
                   "availability_changed",
           "Availability distinct from composition");
    auto remote = transition();
    remote["transition"]["candidate_record"]["remote_attachments"][0]
          ["allocation_ref"] = "allocation-2";
    expect(run(remote)["identity_handoff"]["required"] == false,
           "Remote change never local physical change");
    auto partial = transition();
    partial["transition"]["candidate_record"]["coverage"] = "partial";
    partial["transition"]["candidate_record"]["local_resources"] =
        Json::array();
    expect(run(partial)["resource_changes"][0]["change"] == "unknown",
           "Missing partial inventory is not removal");
    auto complete = partial;
    complete["transition"]["candidate_record"]["coverage"] = "complete";
    expect(run(complete)["resource_changes"][0]["change"] == "removed",
           "Explicit complete capture supports removal");
    auto exposed = t;
    exposed["inventories"][0]["local_resources"][0]["capacity_kind"] =
        "exposed";
    exposed["transition"]["candidate_record"]["local_resources"][0]
           ["capacity_kind"] = "exposed";
    expect(run(exposed)["identity_handoff"]["required"] == false,
           "Changed guest exposure does not prove physical change");
    auto conflicting = fixture();
    auto other = conflicting["inventories"][0];
    other["record_id"] = "independent-record";
    other["local_resources"][0]["instance_ref"] = "other-dimm";
    conflicting["inventories"].push_back(other);
    expect(run(conflicting)["status"] == "conflicting",
           "Independent contradictions preserved");
    auto unknown = conflicting;
    unknown["inventories"][1]["local_resources"][0]["instance_ref"] = nullptr;
    expect(run(unknown)["status"] == "valid", "Unknown is not contradiction");
    auto offering = fixture();
    offering["inventories"][0]["local_resources"] = Json::array();
    offering["inventories"][0]["coverage"] = "unknown";
    expect(run(offering)["records"][0]["local_resources"].empty(),
           "SCV offering does not fabricate hardware");
    auto retire = transition("retirement");
    retire["transition"]["candidate_record"]["status"] = "retired";
    expect(run(retire)["proposed_records"][0]["status"] == "retired",
           "Explicit retirement preserves record");
    auto fork = fixture();
    auto branch = run(transition("correction"))["proposed_records"][0];
    fork["inventories"].push_back(branch);
    branch["record_id"] = "inventory-branch";
    fork["inventories"].push_back(branch);
    expect(run(fork)["status"] == "conflicting",
           "Competing inventory corrections retain conflict");
    auto bad = fixture();
    bad["inventories"][0]["local_resources"][0]["capacity"]["value"] =
        "18446744073709551616";
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"][0]["capacity"]["value"] = "01";
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"][0]["capacity"]["value"] = "1.5";
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"][0]["capacity"]["value"] = 123;
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"][0]["available_capacity"]["unit"] =
        "count";
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"][0]["capacity"]["value"] = "0";
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"][0]["presence"] = "absent";
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"][0]["source_ids"] =
        Json::array({"missing"});
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["local_resources"].push_back(
        bad["inventories"][0]["local_resources"][0]);
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["generation"] = 2;
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["observed_at"] = "2026-13-01T00:00:00Z";
    rejected(bad);
    bad = fixture();
    bad["inventories"][0]["extra"] = true;
    rejected(bad);
    auto rebound = transition();
    rebound["transition"]["candidate_record"]["physical_node_id"] = "node-2";
    rebound["transition"]["candidate_record"]["identity_rebind"] = {
        {"sniv_record_ref", "physical-record-2"},
        {"predecessor_node_id", "node-1"},
        {"successor_node_id", "node-2"}};
    auto rebound_result = run(rebound);
    expect(rebound_result["status"] == "valid" &&
               rebound_result["identity_handoff"]["required"] == true,
           "Supplied SNIV reference qualifies successor inventory");
    auto retained = fixture();
    retained["inventories"].push_back(rebound_result["proposed_records"][0]);
    expect(run(retained)["records"][1]["identity_rebind"]["sniv_record_ref"] ==
               "physical-record-2",
           "Retained rebind survives later snapshot");
    auto later = retained;
    later["mode"] = "transition";
    auto later_candidate = retained["inventories"][1];
    later_candidate["record_id"] = "inventory-after-rebind";
    later_candidate["generation"] = 3;
    later_candidate["predecessor_record_id"] =
        retained["inventories"][1]["record_id"];
    later_candidate["local_resources"][0]["operational_state"] = "unavailable";
    later["transition"] = {
        {"kind", "correction"},
        {"predecessor_record_id", retained["inventories"][1]["record_id"]},
        {"candidate_record", later_candidate}};
    auto later_result = run(later);
    expect(later_result["status"] == "valid" &&
               later_result["identity_handoff"]["required"] == false,
           "Later availability correction does not repeat physical handoff");
    expect(later_result["proposed_records"][0]["identity_rebind"] ==
               retained["inventories"][1]["identity_rebind"],
           "Later correction retains exact original identity proof");
    auto wrong_subject = transition();
    wrong_subject["transition"]["candidate_record"]["physical_node_id"] =
        "node-2";
    rejected(wrong_subject);
    try {
      (void)symphony::snv::snrv::handle("resources_validate", fixture(), -1);
      throw std::runtime_error("Deadline not rejected");
    } catch (const symphony::knowledge::engine::Error &e) {
      expect(e.code() == "deadline_exceeded" && e.exit_status() == 3,
             "Deadline error and process status classification");
    }
    for (int limit : {512, 1024, 2048}) {
      auto capacity = fixture();
      capacity["record_limit"] = limit;
      auto base = capacity["inventories"][0]["local_resources"][0];
      capacity["inventories"][0]["local_resources"] = Json::array();
      capacity["inventories"][0]["remote_attachments"] = Json::array();
      for (int n = 0; n < limit; ++n) {
        auto resource = base;
        resource["resource_id"] = "memory-" + std::to_string(n);
        capacity["inventories"][0]["local_resources"].push_back(resource);
      }
      expect(run(capacity)["records"][0]["local_resources"].size() ==
                 static_cast<std::size_t>(limit),
             "Selected resource capacity exact");
      capacity["inventories"][0]["local_resources"].push_back(base);
      rejected(capacity);
    }
    std::cout << "SNRV " << checks << " independent checks passed\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
