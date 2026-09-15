#include "native_test.hpp"
using namespace native_test;
void test_native_publication_contract(Engine &e) {
  auto call = [&](std::string op, Json p, bool fail = false) {
    return e.call(op, p, !fail);
  };
  std::string h = "sha256:" + std::string(64, 'a');
  Json inst = {
      {"Role", "shv-partition-engine"},
      {"ModuleID", "shv-partition-engine"},
      {"EngineID", "symphony-shv-partition"},
      {"Version", "0.2.0-dev"},
      {"Prefix", "/partition"},
      {"ReceiptPath", "/partition/share/symphony/receipts/shv-partition-engine/"
                      "0.2.0-dev/install-receipt.json"},
      {"ReceiptDigest", h},
      {"ReceiptProtocol", "symphony.knowledge.install-receipt.v2"},
      {"ExecutablePath", "/partition/libexec/symphony/shv-partition-engine/"
                         "0.2.0-dev/symphony-shv-partition"},
      {"ExecutableDigest", h}};
  auto manifest = seal({{"protocol", "symphony.shv.partition-manifest.v1"},
                        {"entries", Json::array()},
                        {"required_references", Json::array()},
                        {"loaded_count", 0},
                        {"missing_count", 0},
                        {"complete_inventory", true},
                        {"reference_statuses", Json::array()}});
  Json d = {
      {"catalogue_id", "research"},
      {"tops_id", "00000000-0000-4000-8000-000000000001"},
      {"manifest", manifest},
      {"policy",
       {{"missing_partitions", "reject"}, {"missing_references", "reject"}}},
      {"partition_installation", inst},
      {"members", Json::array()}};
  Json base = {{"operation_id", "first"},
               {"current", nullptr},
               {"desired", d},
               {"reason", "caller selection"}};
  auto descriptor = call("inspect", Json::object());
  NT_REQUIRE(descriptor["operations"].size() == 4);
  auto plan = call("publication_plan", base);
  auto head = call("publication_reduce",
                   {{"current", nullptr}, {"plan", plan}})["head"];
  NT_REQUIRE(head["generation"] == 1);
  NT_REQUIRE(call("publication_status", {{"history", {head}}})["head"] == head);
  call("publication_plan", updated(base, {{"current", head}}), true);
  for (auto change :
       Json::array({{{"catalogue_id", "other"}},
                    {{"tops_id", "00000000-0000-4000-8000-000000000002"}}}))
    call("publication_plan",
         updated(base, {{"current", head}, {"desired", updated(d, change)}}),
         true);
  for (auto change : Json::array(
           {{{"generation", 2}}, {{"previous_digest", h}}, {{"digest", h}}})) {
    auto bad = plan;
    bad["head"].update(change);
    if (!change.contains("digest"))
      bad["head"] = seal(bad["head"]);
    call("publication_reduce", {{"current", nullptr}, {"plan", seal(bad)}},
         true);
  }
  for (auto change : Json::array({{{"members", nullptr}},
                                  {{"members", Json::array({Json::object()})}},
                                  {{"policy", Json::object()}},
                                  {{"extra", true}}}))
    call("publication_plan", updated(base, {{"desired", updated(d, change)}}),
         true);
  for (auto change : Json::array({{{"Version", "0.3.0-dev"}},
                                  {{"Prefix", "/partition/../bad"}},
                                  {{"ReceiptDigest", "x"}}}))
    call("publication_plan",
         updated(base, {{"desired", updated(d, {{"partition_installation",
                                                 updated(inst, change)}})}}),
         true);
  auto missing = seal(
      updated(manifest, {{"entries", Json::array({{{"partition_digest", h},
                                                   {"partition", nullptr}}})},
                         {"missing_count", 1},
                         {"complete_inventory", false}}));
  call("publication_plan",
       updated(base, {{"desired", updated(d, {{"manifest", missing}})}}), true);
  auto allowed = updated(d, {{"manifest", missing},
                             {"policy",
                              {{"missing_partitions", "allow"},
                               {"missing_references", "allow"}}}});
  NT_REQUIRE(call("publication_plan",
                  updated(base, {{"desired",
                                  allowed}}))["head"]["definition"] == allowed);
  Json refs = Json::array({{{"partition_digest", h}, {"subject_id", "cpu"}}});
  missing = seal(
      updated(missing, {{"required_references", refs},
                        {"reference_statuses",
                         Json::array({{{"reference", refs[0]},
                                       {"status", "missing_partition"}}})}}));
  call("publication_plan",
       updated(base,
               {{"desired",
                 updated(allowed, {{"manifest", missing},
                                   {"policy",
                                    {{"missing_partitions", "allow"},
                                     {"missing_references", "reject"}}}})}}),
       true);
  call("publication_plan",
       updated(base, {{"desired", updated(allowed, {{"manifest", missing}})}}));
  call("publication_status", {{"history", Json::array()}}, true);
  call("publication_status", {{"history", {head, head}}}, true);
  Json history = {head};
  for (int n = 2; n <= 32; ++n) {
    auto next = call(
        "publication_plan",
        {{"operation_id", "r" + std::to_string(n)},
         {"current", history.back()},
         {"desired",
          updated(d, {{"policy",
                       {{"missing_partitions", n % 2 == 0 ? "allow" : "reject"},
                        {"missing_references", "reject"}}}})},
         {"reason", "caller policy change"}});
    call("publication_reduce", {{"current", nullptr}, {"plan", next}}, true);
    history.push_back(call("publication_reduce", {{"current", history.back()},
                                                  {"plan", next}})["head"]);
  }
  NT_REQUIRE(
      call("publication_status", {{"history", history}})["history_digests"]
          .size() == 32);
  call("publication_plan", updated(base, {{"current", history.back()}}), true);
  auto overflow = history;
  overflow.push_back(history.back());
  call("publication_status", {{"history", overflow}}, true);
  for (std::string version :
       {"0.2.0-dev", "0.3.0-dev", "0.4.0-dev", "0.5.0-dev", "latest"}) {
    auto selected = inst;
    for (std::string key : {"ReceiptPath", "ExecutablePath"})
      selected[key] =
          replace(selected[key].get<std::string>(), "0.2.0-dev", version);
    selected["Version"] = version;
    bool accepted = version == "0.2.0-dev" ||
                    (descriptor["engine_version"] == "0.4.0-dev" &&
                     (version == "0.3.0-dev" || version == "0.4.0-dev"));
    call("publication_plan",
         updated(base, {{"desired",
                         updated(d, {{"partition_installation", selected}})}}),
         !accepted);
  }
}
int main(int argc, char **argv) {
  return test_main([&] {
    Arguments a(argc, argv);
    Engine e{a.require("engine"), "symphony-shv-publication",
             "publication-test"};
    test_native_publication_contract(e);
    e.summary();
  });
}
