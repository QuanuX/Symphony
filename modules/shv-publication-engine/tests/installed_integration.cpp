#include "installed_support.hpp"
using namespace native_test;
void test_installed_publication_provenance(const Arguments &args) {
  Evidence evidence(args);
  auto prefix = fs::canonical(args.require("prefix"));
  auto receipt = prefix / "share/symphony/receipts/shv-publication-engine" /
                 args.require("version") / "install-receipt.json";
  auto installation = evidence.installation(
      "shv-publication-engine", "symphony-shv-publication",
      {"catalogue", "publication"}, receipt);
  auto partition_prefix = fs::canonical(args.require("partition-prefix"));
  auto partition_receipt =
      partition_prefix / "share/symphony/receipts/shv-partition-engine" /
      args.require("partition-version") / "install-receipt.json";
  auto partition = evidence.installation(
      "shv-partition-engine", "symphony-shv-partition", {"partition"},
      partition_receipt, partition_prefix, args.require("partition-version"));
  TempDir temporary("shv-publication-installed-");
  auto call = [&](const std::string &op, const Json &payload,
                  const std::string &error = "") {
    return evidence.call(installation, op, payload, error, temporary.path);
  };
  auto manifest = seal({{"protocol", "symphony.shv.partition-manifest.v1"},
                        {"entries", Json::array()},
                        {"required_references", Json::array()},
                        {"loaded_count", 0},
                        {"missing_count", 0},
                        {"complete_inventory", true},
                        {"reference_statuses", Json::array()}});
  Json desired = {
      {"catalogue_id", "caller"},
      {"tops_id", "00000000-0000-4000-8000-000000000001"},
      {"manifest", manifest},
      {"policy",
       {{"missing_partitions", "reject"}, {"missing_references", "reject"}}},
      {"partition_installation", partition},
      {"members", Json::array()}};
  Json payload = {{"operation_id", "first"},
                  {"current", nullptr},
                  {"desired", desired},
                  {"reason", "caller test"}};
  auto plan = call("publication_plan", payload);
  NT_REQUIRE(plan["head"]["definition"] == desired);
  auto head = call("publication_reduce",
                   {{"current", nullptr}, {"plan", plan}})["head"];
  NT_REQUIRE(head["generation"] == 1 &&
             head["definition"]["partition_installation"] == partition);
  NT_REQUIRE(call("publication_status", {{"history", {head}}})["head"] == head);
  auto forged = plan;
  forged["head"]["generation"] = 2;
  forged["head"] = seal(forged["head"]);
  call("publication_reduce", {{"current", nullptr}, {"plan", seal(forged)}},
       "shv-publication.invalid");
  call("publication_status", {{"history", {head, head}}},
       "shv-publication.invalid");
  auto substituted = desired;
  substituted["partition_installation"]["Version"] = "latest";
  call("publication_plan", updated(payload, {{"desired", substituted}}),
       "shv-publication.invalid");
  evidence.finish("test_installed_publication_provenance");
}
int main(int argc, char **argv) {
  return test_main(
      [&] { test_installed_publication_provenance(Arguments(argc, argv)); });
}
