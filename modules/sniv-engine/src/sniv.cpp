#include "validation.hpp"
#include <map>
#include <symphony/snv/sniv.hpp>
namespace symphony::snv::sniv {
namespace {
using namespace owner_validation;
void provider(const Json &j) {
  if (j.is_null())
    return;
  keys(j, {"provider", "scope", "native_id", "established", "source_ids"});
  (void)text(j, "provider");
  (void)text(j, "scope", 1024);
  (void)text(j, "native_id", 1024);
  require(j.at("established").is_boolean(), "Established must be boolean");
  unique_strings(j.at("source_ids"), 128);
  require(!j.at("source_ids").empty(), "Provider evidence must have sources");
}
void physical(const Json &j) {
  keys(j, {"record_id", "generation", "predecessor_record_id",
           "physical_node_id", "predecessor_node_id", "status",
           "provider_resource", "assertions", "sources"});
  (void)text(j, "record_id");
  require(integer(j.at("generation")) > 0, "Generation must be positive");
  nullable_text(j.at("predecessor_record_id"));
  nullable_text(j.at("physical_node_id"));
  nullable_text(j.at("predecessor_node_id"));
  choice(j.at("status"), {"active", "retired"});
  auto refs = sources(j.at("sources"));
  require(!refs.empty(), "Identity record requires attributed evidence");
  provider(j.at("provider_resource"));
  if (!j.at("provider_resource").is_null())
    references(j.at("provider_resource").at("source_ids"), refs);
  for (const auto &a : array(j.at("assertions"), 128)) {
    keys(a, {"field", "value", "source_id"});
    choice(a.at("field"),
           {"serial", "firmware_uuid", "provider_label", "infrastructure_ref",
            "user_identity", "habitat_ref"});
    (void)text(a, "value", 1024);
    require(refs.contains(text(a, "source_id")),
            "Identity assertion has unknown source");
  }
}
void episode(const Json &j) {
  keys(j, {"record_id", "generation", "predecessor_record_id",
           "physical_node_id", "incarnation_id", "system_id", "bus_ids",
           "boot_id", "state", "last_event", "effective_at", "sources"});
  (void)text(j, "record_id");
  require(integer(j.at("generation")) > 0, "Generation must be positive");
  nullable_text(j.at("predecessor_record_id"));
  (void)text(j, "physical_node_id");
  (void)text(j, "incarnation_id");
  (void)text(j, "system_id");
  unique_strings(j.at("bus_ids"), 128);
  require(!j.at("bus_ids").empty(),
          "An incarnation requires a supplied bus attachment");
  nullable_text(j.at("boot_id"));
  choice(j.at("state"), {"active", "ended"});
  choice(j.at("last_event"), {"establish", "disconnect", "reconnect", "end",
                              "rejoin", "boot_change", "correction"});
  timestamp(j.at("effective_at"));
  require(!sources(j.at("sources")).empty(),
          "Participation requires attributed evidence");
}
std::string status(const Json &findings) {
  for (const auto &f : findings)
    if (text(f, "code").starts_with("conflicting_"))
      return "conflicting";
  for (const auto &f : findings)
    if (text(f, "code").starts_with("insufficient_"))
      return "insufficient";
  return "valid";
}
void causal(const Json &records, Json &findings, bool participation) {
  std::map<std::string, const Json *> by_id;
  std::set<std::string> predecessors;
  std::map<std::string, std::string> successor_records;
  std::map<std::string, std::pair<std::string, std::string>>
      incarnation_subjects;
  for (const auto &r : records)
    require(by_id.emplace(text(r, "record_id"), &r).second,
            "Duplicate record identifier");
  for (const auto &r : records) {
    if (participation) {
      auto subject =
          Json::array({r.at("physical_node_id"), r.at("system_id")}).dump();
      auto [binding, unique] = incarnation_subjects.emplace(
          text(r, "incarnation_id"),
          std::make_pair(subject, text(r, "record_id")));
      if (!unique && binding->second.first != subject)
        findings.push_back(
            finding("conflicting_incarnation_subjects",
                    Json::array({binding->second.second, r.at("record_id")})));
    }
    const auto &predecessor = r.at("predecessor_record_id");
    if (predecessor.is_null()) {
      require(integer(r.at("generation")) == 1,
              "Initial generation must equal one");
      continue;
    }
    auto p = by_id.find(str(predecessor));
    if (p == by_id.end()) {
      findings.push_back(
          finding("insufficient_predecessor",
                  Json::array({r.at("record_id"), predecessor})));
      continue;
    }
    auto [branch, unique_branch] =
        successor_records.emplace(str(predecessor), text(r, "record_id"));
    if (!unique_branch)
      findings.push_back(
          finding("conflicting_causal_branches",
                  Json::array({branch->second, r.at("record_id")})));
    predecessors.insert(str(predecessor));
    require(integer(r.at("generation")) ==
                integer(p->second->at("generation")) + 1,
            "Invalid causal generation");
    if (participation) {
      require(r.at("physical_node_id") == p->second->at("physical_node_id") &&
                  r.at("system_id") == p->second->at("system_id"),
              "Participation predecessor subject mismatch");
      if (r.at("incarnation_id") != p->second->at("incarnation_id"))
        require(p->second->at("state") == "ended" &&
                    r.at("state") == "active" && r.at("last_event") == "rejoin",
                "New incarnation requires explicit end and rejoin");
      else {
        if (p->second->at("state") == "ended")
          require(r.at("state") == "ended" &&
                      r.at("last_event") == "correction",
                  "Ended incarnation cannot silently reactivate");
        else if (r.at("state") == "ended")
          require(r.at("last_event") == "end",
                  "Episode ending requires explicit event");
        else
          require(r.at("last_event") != "establish" &&
                      r.at("last_event") != "rejoin" &&
                      r.at("last_event") != "end",
                  "Unexpected lifecycle event");
        if (r.at("last_event") == "disconnect" ||
            r.at("last_event") == "reconnect")
          require(r.at("bus_ids") == p->second->at("bus_ids") &&
                      r.at("boot_id") == p->second->at("boot_id"),
                  "Connection observation cannot replace bus or boot identity");
      }
    } else if (r.at("physical_node_id") != p->second->at("physical_node_id")) {
      require(!r.at("physical_node_id").is_null() &&
                  r.at("predecessor_node_id") ==
                      p->second->at("physical_node_id"),
              "New identity needs supplied predecessor Node");
    }
  }
  std::map<std::string, std::string> active_systems, providers;
  std::map<std::string, std::pair<std::string, std::string>>
      assertions_by_subject;
  for (const auto &r : records) {
    if (predecessors.contains(text(r, "record_id")))
      continue;
    if (participation && r.at("state") == "active") {
      auto key = r.at("physical_node_id").dump() + r.at("system_id").dump();
      auto [it, inserted] =
          active_systems.emplace(key, text(r, "incarnation_id"));
      if (!inserted && it->second != text(r, "incarnation_id"))
        findings.push_back(finding("conflicting_active_incarnations",
                                   Json::array({r.at("record_id")})));
    }
    if (!participation) {
      if (r.at("physical_node_id").is_null())
        findings.push_back(finding("insufficient_physical_identity",
                                   Json::array({r.at("record_id")})));
      std::map<std::string, std::set<std::string>> assertion_values;
      if (!r.at("physical_node_id").is_null())
        for (const auto &a : r.at("assertions")) {
          auto key = r.at("physical_node_id").dump() + a.at("field").dump();
          auto [it, inserted] = assertions_by_subject.emplace(
              key,
              std::make_pair(text(a, "value", 1024), text(r, "record_id")));
          if (!inserted && it->second.first != text(a, "value", 1024))
            findings.push_back(
                finding("conflicting_identity_assertions",
                        Json::array({it->second.second, r.at("record_id")})));
        }
      for (const auto &a : r.at("assertions"))
        assertion_values[text(a, "field")].insert(text(a, "value", 1024));
      for (const auto &[field, values] : assertion_values)
        if (values.size() > 1)
          findings.push_back(
              {{"code", "conflicting_identity_assertions"},
               {"field", field},
               {"record_refs", Json::array({r.at("record_id")})}});
      if (r.at("status") == "active" && !r.at("provider_resource").is_null() &&
          !r.at("physical_node_id").is_null()) {
        const auto &p = r.at("provider_resource");
        auto key = p.at("provider").dump() + p.at("scope").dump() +
                   p.at("native_id").dump();
        auto [it, inserted] =
            providers.emplace(key, text(r, "physical_node_id"));
        if (!inserted && it->second != text(r, "physical_node_id"))
          findings.push_back(finding("conflicting_provider_subjects",
                                     Json::array({r.at("record_id")})));
      }
    }
  }
}
const Json &predecessor(const Json &records, const Json &id) {
  for (const auto &r : records)
    if (r.at("record_id") == id)
      return r;
  fail("invalid_input", "Transition predecessor was not supplied");
}
std::string materiality(const Json &profile, const Json &changes) {
  keys(profile, {"profile_id", "version", "rules", "unknown_decision"});
  (void)text(profile, "profile_id");
  (void)text(profile, "version");
  require(profile.at("unknown_decision") == "unresolved",
          "Unlisted hardware changes must remain unresolved");
  std::map<std::string, std::string> rules;
  for (const auto &rule : array(profile.at("rules"), 128)) {
    keys(rule, {"component_kind", "change", "decision"});
    (void)text(rule, "component_kind");
    choice(rule.at("change"), {"added", "removed", "replaced",
                               "capacity_changed", "availability_changed"});
    choice(rule.at("decision"), {"material", "non_material"});
    require(
        rules
            .emplace(text(rule, "component_kind") + ":" + text(rule, "change"),
                     text(rule, "decision"))
            .second,
        "Duplicate materiality rule");
  }
  bool material = false, unknown = false;
  for (const auto &change : array(changes, 2048)) {
    keys(change, {"component_kind", "change", "locality", "evidence_refs"});
    (void)text(change, "component_kind");
    choice(change.at("change"), {"added", "removed", "replaced",
                                 "capacity_changed", "availability_changed"});
    choice(change.at("locality"), {"local", "remote", "software"});
    unique_strings(change.at("evidence_refs"), 128);
    require(!change.at("evidence_refs").empty(),
            "A change requires supplied evidence references");
    if (change.at("locality") != "local")
      continue;
    auto found = rules.find(text(change, "component_kind") + ":" +
                            text(change, "change"));
    if (found == rules.end())
      unknown = true;
    else if (found->second == "material")
      material = true;
  }
  if (material)
    return "material";
  if (unknown)
    return "unresolved";
  return "non_material";
}
Json reduce_physical(const Json &t, const Json &records, Json &findings) {
  keys(t, {"kind", "predecessor_record_id", "candidate_record", "change_kind",
           "materiality_profile", "changes"});
  const auto &old = predecessor(records, t.at("predecessor_record_id"));
  const auto &next = t.at("candidate_record");
  physical(next);
  require(next.at("record_id") != old.at("record_id") &&
              next.at("predecessor_record_id") == old.at("record_id") &&
              integer(next.at("generation")) ==
                  integer(old.at("generation")) + 1,
          "Candidate does not follow exact predecessor");
  choice(t.at("change_kind"),
         {"unchanged", "software_only", "remote_only", "replacement",
          "provider_resource", "hardware_change", "correction", "retirement",
          "restoration"});
  const auto kind = text(t, "change_kind");
  bool new_identity = kind == "replacement";
  std::string classification = "preserved";
  if (kind == "hardware_change") {
    require(!t.at("materiality_profile").is_null(),
            "Hardware change requires selected materiality profile");
    require(!array(t.at("changes"), 2048).empty(),
            "Hardware change list is empty");
    auto decision = materiality(t.at("materiality_profile"), t.at("changes"));
    if (decision == "unresolved") {
      findings.push_back(
          finding("insufficient_materiality",
                  Json::array({old.at("record_id"), next.at("record_id")})));
      return Json::array();
    }
    new_identity = decision == "material";
    classification = decision;
  } else {
    require(t.at("materiality_profile").is_null() &&
                t.at("changes").is_array() && t.at("changes").empty(),
            "Unexpected materiality data");
  }
  const auto &op = old.at("provider_resource");
  const auto &np = next.at("provider_resource");
  if (!op.is_null() && !np.is_null() && op.at("established") == true &&
      np.at("established") == true &&
      (op.at("provider") != np.at("provider") ||
       op.at("scope") != np.at("scope") ||
       op.at("native_id") != np.at("native_id")))
    new_identity = true;
  if (kind == "provider_resource" &&
      (op.is_null() || np.is_null() || op.at("established") != true ||
       np.at("established") != true)) {
    findings.push_back(
        finding("insufficient_provider_establishment",
                Json::array({old.at("record_id"), next.at("record_id")})));
    return Json::array();
  }
  if (old.at("physical_node_id").is_null()) {
    findings.push_back(finding("insufficient_predecessor_identity",
                               Json::array({old.at("record_id")})));
    return Json::array();
  }
  if (new_identity) {
    classification = "new_identity_required";
    if (next.at("physical_node_id").is_null()) {
      findings.push_back(finding("insufficient_successor_identity",
                                 Json::array({next.at("record_id")})));
      return Json::array();
    }
    if (next.at("physical_node_id") == old.at("physical_node_id") ||
        next.at("predecessor_node_id") != old.at("physical_node_id")) {
      findings.push_back(
          finding("conflicting_identity_reuse",
                  Json::array({old.at("record_id"), next.at("record_id")})));
      return Json::array();
    }
  } else if (next.at("physical_node_id") != old.at("physical_node_id") ||
             next.at("predecessor_node_id") != old.at("predecessor_node_id")) {
    findings.push_back(
        finding("conflicting_unjustified_reidentification",
                Json::array({old.at("record_id"), next.at("record_id")})));
    return Json::array();
  }
  if (kind == "retirement")
    require(old.at("status") == "active" && next.at("status") == "retired",
            "Retirement state mismatch");
  else if (kind == "restoration")
    require(old.at("status") == "retired" && next.at("status") == "active",
            "Restoration state mismatch");
  else
    require(next.at("status") == old.at("status") || new_identity,
            "Unexpected record state change");
  findings.push_back({{"code", "physical_transition_classified"},
                      {"classification", classification},
                      {"materiality_profile", t.at("materiality_profile")},
                      {"record_refs", Json::array({old.at("record_id"),
                                                   next.at("record_id")})}});
  return Json::array({next});
}
Json reduce_episode(const Json &t, const Json &records, Json &findings) {
  keys(t, {"kind", "predecessor_record_id", "candidate_record", "action"});
  const auto &next = t.at("candidate_record");
  episode(next);
  choice(t.at("action"), {"establish", "disconnect", "reconnect", "end",
                          "rejoin", "boot_change", "correction"});
  auto action = text(t, "action");
  require(next.at("last_event") == action,
          "Participation event/action mismatch");
  if (action == "establish") {
    require(t.at("predecessor_record_id").is_null() &&
                next.at("predecessor_record_id").is_null() &&
                integer(next.at("generation")) == 1 &&
                next.at("state") == "active",
            "Invalid initial incarnation");
    for (const auto &r : records)
      require(r.at("incarnation_id") != next.at("incarnation_id"),
              "Incarnation identifier already used");
  } else {
    const auto &old = predecessor(records, t.at("predecessor_record_id"));
    require(next.at("predecessor_record_id") == old.at("record_id") &&
                integer(next.at("generation")) ==
                    integer(old.at("generation")) + 1 &&
                next.at("record_id") != old.at("record_id"),
            "Participation candidate does not follow predecessor");
    require(next.at("physical_node_id") == old.at("physical_node_id") &&
                next.at("system_id") == old.at("system_id"),
            "Participation subject changed");
    if (action == "rejoin")
      require(old.at("state") == "ended" && next.at("state") == "active" &&
                  next.at("incarnation_id") != old.at("incarnation_id"),
              "Rejoin requires ended predecessor and supplied new incarnation");
    else {
      require(
          next.at("incarnation_id") == old.at("incarnation_id"),
          "Temporary interruption, boot and correction preserve incarnation");
      if (action == "end")
        require(old.at("state") == "active" && next.at("state") == "ended",
                "End requires active incarnation");
      else
        require(next.at("state") == old.at("state") &&
                    (action == "correction" || old.at("state") == "active"),
                "Interrupted episode cannot reactivate ended incarnation");
      if (action == "disconnect" || action == "reconnect")
        require(
            next.at("bus_ids") == old.at("bus_ids") &&
                next.at("boot_id") == old.at("boot_id"),
            "Connection event cannot silently replace boot or bus association");
    }
  }
  findings.push_back(finding(action == "rejoin" || action == "establish"
                                 ? "incarnation_established"
                                 : "incarnation_preserved",
                             Json::array({next.at("record_id")})));
  return Json::array({next});
}
} // namespace
std::vector<Operation> operations() {
  return {{"identity_validate",
           "symphony.sniv.identity-validate-input.v1",
           "symphony.sniv.identity-validate.v1",
           {"validate", "propose"}}};
}
Json handle(std::string_view operation, const Json &payload,
            std::int64_t deadline_unix_ms) {
  try {
    deadline(deadline_unix_ms);
    require(operation == "identity_validate", "Unsupported SNIV operation");
    keys(payload, {"protocol", "mode", "record_limit", "physical_records",
                   "participations", "transition"});
    require(payload.at("protocol") ==
                "symphony.sniv.identity-validate-input.v1",
            "Unsupported SNIV input protocol");
    choice(payload.at("mode"), {"snapshot", "transition"});
    auto maximum = limit(payload.at("record_limit"));
    const auto &physical_records =
        array(payload.at("physical_records"), maximum);
    const auto &episodes = array(payload.at("participations"), maximum);
    require(physical_records.size() + episodes.size() <= maximum,
            "Combined record capacity exceeded");
    std::set<std::string> record_ids, subject_ids, incarnation_ids;
    for (const auto &r : physical_records) {
      deadline(deadline_unix_ms);
      physical(r);
      require(record_ids.insert(text(r, "record_id")).second,
              "Duplicate record identifier");
      if (!r.at("physical_node_id").is_null())
        subject_ids.insert(text(r, "physical_node_id"));
    }
    for (const auto &r : episodes) {
      deadline(deadline_unix_ms);
      episode(r);
      require(record_ids.insert(text(r, "record_id")).second,
              "Duplicate record identifier");
      subject_ids.insert(text(r, "physical_node_id"));
      incarnation_ids.insert(text(r, "incarnation_id"));
    }
    Json findings = Json::array(), proposals = Json::array();
    causal(physical_records, findings, false);
    causal(episodes, findings, true);
    if (physical_records.empty() && episodes.empty())
      findings.push_back(
          finding("insufficient_identity_records", Json::array()));
    if (payload.at("mode") == "snapshot")
      require(payload.at("transition").is_null(),
              "Snapshot cannot carry transition");
    else {
      const auto &t = payload.at("transition");
      require(t.is_object() && t.contains("kind"), "Transition missing kind");
      choice(t.at("kind"), {"physical_change", "participation"});
      if (t.at("kind") == "physical_change")
        proposals = reduce_physical(t, physical_records, findings);
      else
        proposals = reduce_episode(t, episodes, findings);
      require(physical_records.size() + episodes.size() + proposals.size() <=
                  maximum,
              "Proposed record exceeds selected capacity");
      for (const auto &r : proposals) {
        require(record_ids.insert(text(r, "record_id")).second,
                "Candidate record identifier already retained");
        if (!r.at("physical_node_id").is_null())
          subject_ids.insert(text(r, "physical_node_id"));
        if (r.contains("incarnation_id"))
          incarnation_ids.insert(text(r, "incarnation_id"));
      }
      if (!proposals.empty()) {
        auto candidate_physical = physical_records,
             candidate_episodes = episodes;
        if (t.at("kind") == "physical_change")
          candidate_physical.push_back(proposals.front());
        else
          candidate_episodes.push_back(proposals.front());
        causal(candidate_physical, findings, false);
        causal(candidate_episodes, findings, true);
      }
    }
    auto result_status = status(findings);
    if (result_status != "valid")
      proposals = Json::array();
    deadline(deadline_unix_ms);
    return {{"protocol", "symphony.sniv.identity-validate.v1"},
            {"owner", "sniv"},
            {"owner_version", version},
            {"source_digest", digest(payload)},
            {"subject_ids", ids(subject_ids)},
            {"incarnation_ids", ids(incarnation_ids)},
            {"status", result_status},
            {"records",
             {{"physical_records", physical_records},
              {"participations", episodes}}},
            {"proposed_records", proposals},
            {"findings", findings}};
  } catch (const nlohmann::json::exception &) {
    fail("invalid_input", "Invalid SNIV JSON field type");
  }
}
} // namespace symphony::snv::sniv
