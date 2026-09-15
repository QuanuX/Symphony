#include "native_test.hpp"
using namespace native_test;
std::string H(int n) { return digest(std::to_string(n)); }
Json fixture() {
  return {{"dependencies",
           {{"source_revision_digest", H(1)},
            {"source_engine",
             {{"engine_id", "symphony-shv-source"},
              {"version", "0.1.0-dev"},
              {"executable_digest", H(2)}}},
            {"kernel_engine",
             {{"engine_id", "symphony-shv"},
              {"version", "0.2.0-dev"},
              {"executable_digest", H(3)}}},
            {"captures", Json::array({{{"capture_id", "page"},
                                       {"capture_digest", H(4)},
                                       {"content_digest", H(5)},
                                       {"bytes", 7}}})},
            {"mapping_digest", H(6)},
            {"catalogue_digest", H(7)}}},
          {"subject_ids", {"cpu-b", "cpu-a"}}};
}
void test_native_partition_contract(Engine &e) {
  auto call = [&](std::string op, Json p, bool fail = false) {
    return e.call(op, p, !fail);
  };
  auto desc = call("inspect", Json::object());
  NT_REQUIRE(desc["operations"].size() == 4);
  for (std::string sv : {"0.1.0-dev", "0.2.0-dev", "0.3.0-dev", "latest"})
    for (std::string kv :
         {"0.1.0-dev", "0.2.0-dev", "0.3.0-dev", "0.4.0-dev", "0.5.0-dev"}) {
      auto sample = fixture();
      sample["dependencies"]["source_engine"]["version"] = sv;
      sample["dependencies"]["kernel_engine"]["version"] = kv;
      bool accepted =
          (sv == "0.1.0-dev" ||
           (desc["engine_version"] == "0.4.0-dev" && sv == "0.2.0-dev")) &&
          kv != "0.5.0-dev" &&
          (kv != "0.4.0-dev" || desc["engine_version"] == "0.4.0-dev");
      call("partition_build", sample, !accepted);
    }
  auto f = fixture(), part = call("partition_build", f);
  NT_REQUIRE(part["subject_ids"] == Json::array({"cpu-a", "cpu-b"}));
  std::reverse(f["subject_ids"].begin(), f["subject_ids"].end());
  NT_REQUIRE(call("partition_build", f) == part);
  Json refs = Json::array(
      {{{"partition_digest", part["digest"]}, {"subject_id", "cpu-a"}},
       {{"partition_digest", H(8)}, {"subject_id", "cpu-c"}},
       {{"partition_digest", part["digest"]}, {"subject_id", "cpu-x"}},
       {{"partition_digest", H(9)}, {"subject_id", "cpu-d"}}});
  Json mi = {
      {"entries",
       Json::array({{{"partition_digest", part["digest"]}, {"partition", part}},
                    {{"partition_digest", H(8)}, {"partition", nullptr}}})},
      {"required_references", refs}};
  auto m = call("manifest_build", mi);
  NT_REQUIRE(m["missing_count"] == 1 && m["complete_inventory"] == false);
  Json statuses = Json::array();
  for (auto row : m["reference_statuses"])
    statuses.push_back(row["status"]);
  NT_REQUIRE(statuses ==
             Json::array({"found", "missing_partition", "missing_subject",
                          "unlisted_partition"}));
  Json qi = {{"manifest", m},
             {"selection", refs},
             {"limit", 1},
             {"cursor", nullptr}},
       rows = Json::array();
  for (int i = 0; i < 4; ++i) {
    auto q = call("manifest_query", qi);
    NT_REQUIRE(q["offset"] == i);
    for (auto row : q["rows"])
      rows.push_back(row);
    qi["cursor"] = q["next_cursor"];
  }
  NT_REQUIRE(qi["cursor"].is_null() && rows == m["reference_statuses"]);
  qi["cursor"] = nullptr;
  auto q = call("manifest_query", qi);
  for (auto change : Json::array({{{"manifest_digest", H(0)}},
                                  {{"selection_digest", H(0)}},
                                  {{"offset", 0}},
                                  {{"offset", 4}},
                                  {{"offset", true}},
                                  {{"offset", 1.5}}}))
    call("manifest_query",
         updated(qi, {{"cursor", updated(q["next_cursor"], change)}}), true);
  for (auto change : Json::array({{{"complete_inventory", true}},
                                  {{"missing_count", 0}},
                                  {{"reference_statuses", Json::array()}}}))
    call("manifest_query",
         updated(qi, {{"manifest", seal(updated(m, change))}}), true);
  for (auto change : Json::array({{{"subject_ids", {"x", "x"}}},
                                  {{"subject_ids", Json::array({nullptr})}},
                                  {{"subject_ids", nullptr}},
                                  {{"extra", true}}}))
    call("partition_build", updated(f, change), true);
  for (auto change : Json::array({{{"bytes", true}},
                                  {{"bytes", -1}},
                                  {{"bytes", 1048577}},
                                  {{"bytes", 1.5}},
                                  {{"capture_digest", "invalid"}}})) {
    auto bad = f;
    bad["dependencies"]["captures"][0].update(change);
    call("partition_build", bad, true);
  }
  auto bad = f;
  bad["dependencies"]["captures"].push_back(bad["dependencies"]["captures"][0]);
  call("partition_build", bad, true);
  for (auto change : Json::array({{{"source_engine",
                                    {{"engine_id", "other"},
                                     {"version", "0.1.0-dev"},
                                     {"executable_digest", H(2)}}}},
                                  {{"captures", Json::array()}}})) {
    bad = f;
    bad["dependencies"].update(change);
    call("partition_build", bad, true);
  }
  auto duplicate = mi["entries"];
  for (auto row : mi["entries"])
    duplicate.push_back(row);
  call("manifest_build", updated(mi, {{"entries", duplicate}}), true);
  call("manifest_build",
       updated(mi, {{"entries", Json::array({{{"partition_digest", H(0)},
                                              {"partition", part}}})}}),
       true);
  auto duplicate_refs = refs;
  for (auto row : refs)
    duplicate_refs.push_back(row);
  call("manifest_build", updated(mi, {{"required_references", duplicate_refs}}),
       true);
  auto empty = call("manifest_build", {{"entries", Json::array()},
                                       {"required_references", Json::array()}});
  auto out = call("manifest_query", {{"manifest", empty},
                                     {"selection", Json::array()},
                                     {"limit", 32},
                                     {"cursor", nullptr}});
  NT_REQUIRE(out["rows"].empty() && out["next_cursor"].is_null());
  call("manifest_query", updated(qi, {{"limit", 33}}), true);
  call("manifest_query", updated(qi, {{"limit", 0}}), true);
  auto second = call(
      "partition_build",
      updated(f, {{"dependencies",
                   updated(f["dependencies"], {{"mapping_digest", H(22)}})}}));
  Json entries = Json::array(), required = Json::array();
  for (auto item : {part, second}) {
    entries.push_back(
        {{"partition_digest", item["digest"]}, {"partition", item}});
    required.push_back(
        {{"partition_digest", item["digest"]}, {"subject_id", "cpu-a"}});
  }
  auto both = call("manifest_build",
                   {{"entries", entries}, {"required_references", required}});
  for (auto row : both["reference_statuses"])
    NT_REQUIRE(row["status"] == "found");
  // All inventory cardinality boundaries, including mixed missing references.
  f = fixture();
  Json caps = Json::array();
  for (int i = 0; i < 5; ++i)
    caps.push_back({{"capture_id", "page-" + std::to_string(i)},
                    {"capture_digest", H(100 + i)},
                    {"content_digest", H(200 + i)},
                    {"bytes", 1048576}});
  f["dependencies"]["captures"] = caps;
  call("partition_build", f, true);
  caps.erase(4);
  f["dependencies"]["captures"] = caps;
  auto base = call("partition_build", f);
  entries = Json::array();
  for (int i = 0; i < 64; ++i) {
    auto value = base;
    value["dependencies"]["mapping_digest"] = H(300 + i);
    value = seal(value);
    entries.push_back(
        {{"partition_digest", value["digest"]}, {"partition", value}});
  }
  refs = Json::array();
  for (auto entry : entries)
    for (std::string id : {"cpu-a", "cpu-b"})
      refs.push_back({{"partition_digest", entry["partition_digest"]},
                      {"subject_id", id}});
  m = call("manifest_build",
           {{"entries", entries}, {"required_references", refs}});
  NT_REQUIRE(m["loaded_count"] == 64 && m["reference_statuses"].size() == 128);
  auto overflow = entries;
  overflow.push_back({{"partition_digest", H(999)}, {"partition", nullptr}});
  call("manifest_build",
       {{"entries", overflow}, {"required_references", Json::array()}}, true);
  auto too_many = refs;
  too_many.push_back({{"partition_digest", H(999)}, {"subject_id", "cpu"}});
  call("manifest_build",
       {{"entries", entries}, {"required_references", too_many}}, true);
  q = call("manifest_query", {{"manifest", m},
                              {"selection", refs},
                              {"limit", 32},
                              {"cursor", nullptr}});
  NT_REQUIRE(q["rows"].size() == 32 && q["next_cursor"]["offset"] == 32);
}
void test_installed_process(const std::string &prefix) {
  Engine e{installed_engine(prefix, "shv-partition-engine",
                            "symphony-shv-partition", "0.2.0-dev",
                            "vector_engine"),
           "symphony-shv-partition"};
  test_native_partition_contract(e);
  e.summary();
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments a(argc, argv);
    require(a.has("engine") != a.has("prefix"), "Select --engine or --prefix");
    if (a.has("prefix")) {
      test_installed_process(a.require("prefix"));
    } else {
      Engine e{a.require("engine"), "symphony-shv-partition"};
      test_native_partition_contract(e);
      e.summary();
    }
  });
}
