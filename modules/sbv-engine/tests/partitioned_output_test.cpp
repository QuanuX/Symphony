#include "partitioned_output.hpp"
#include <filesystem>
#include <iostream>
#include <stdexcept>
#include <string>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/error.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <unistd.h>

namespace s = symphony::sbv::result_store;
namespace d = symphony::sbv::detail;
namespace l = symphony::sbv::logical;
namespace k = symphony::knowledge::engine;
using Json = s::Json;
std::uint64_t checks = 0;
void require(bool yes, const char *message) {
  ++checks;
  if (!yes)
    throw std::runtime_error(message);
}
template <class Function> void rejects(Function &&f, const char *message) {
  bool refused = false;
  try {
    f();
  } catch (const std::exception &) {
    refused = true;
  }
  require(refused, message);
}
Json body() {
  Json sections = Json::object();
  for (const auto *name : {"summary", "signals", "execution", "distributions",
                           "studies", "replay", "comparisons", "search",
                           "resources", "diagnostics", "choices", "provenance"})
    sections[name] = {{"data", nullptr},
                      {"reason", "not selected"},
                      {"status", "not_selected"}};
  sections["summary"] = {{"data", {{"signal_count", "3"}}},
                         {"status", "available"},
                         {"reason", ""}};
  return {{"protocol", "symphony.sbv.result.v1"},
          {"origin", "workspace-fixture"},
          {"status", "partial"},
          {"sections", sections}};
}
Json request(const std::string &base, const std::string &name) {
  return {
      {"protocol", "symphony.sbv.generate-census-input.v1"},
      {"output",
       {{"kind", "partitioned"},
        {"workspace_path", base + "/" + name + "-work"},
        {"bundle_path", base + "/" + name + "-result"},
        {"write_options", {{"page_bytes", "2048"}, {"index_fanout", "2"}}}}},
      {"extensions", Json::object()}};
}
s::Reference reference(const Json &receipt) {
  const auto &r = receipt.at("storage").at("reference");
  return {r.at("manifest_path"), r.at("manifest_sha256"),
          r.at("content_sha256")};
}
l::Value produce_rows(d::PartitionedOutput &context) {
  require(std::filesystem::is_directory(context.workspace_path()) &&
              !std::filesystem::exists(context.bundle_path()),
          "workspace exists and final bundle absent in producer");
  require(!context.read_options().max_page_bytes &&
              context.read_options().cache_bytes == 0,
          "linear scratch reader options");
  auto spool = context.spool("census-signals");
  for (std::uint64_t i = 0; i < 3; ++i)
    spool->append(Json{{"signal_id", "signal-" + std::to_string(i)}});
  auto closed = spool->close();
  auto original = l::Value(body());
  auto sections =
      original.at("sections")
          .with("signals",
                l::Value::object({{"data", l::Value(closed.rows)},
                                  {"reason", l::Value("")},
                                  {"status", l::Value("available")}}));
  return original.with("sections", std::move(sections));
}

int main() {
  try {
#if defined(__APPLE__)
    const std::string tmp = "/private/tmp";
#else
    const std::string tmp = "/tmp";
#endif
    const auto base =
        tmp + "/sbv-partitioned-output-" + std::to_string(::getpid());
    require(std::filesystem::create_directory(base),
            "fresh workspace test root");
    const auto normal_request = request(base, "normal");
    const auto normal = d::partitioned_result(normal_request, "generate-census",
                                              k::no_deadline, produce_rows);
    require(normal.at("status") == "partial" && normal.size() == 5,
            "logical partial status with exact receipt shape");
    require(normal.at("storage").at("workspace_retained") == true &&
                normal.at("storage").at("write_options") ==
                    normal_request.at("output").at("write_options"),
            "exact storage choices retained");
    require(normal.at("summary") == body().at("sections").at("summary"),
            "bounded exact summary retained");
    std::filesystem::remove_all(
        normal_request.at("output").at("workspace_path").get<std::string>());
    s::ResultReader portable(reference(normal));
    require(portable.verify_closure().at("verification_extent") ==
                    "full_logical_closure" &&
                portable.select("/sections/signals/data").read_value().size() ==
                    3,
            "published descendant independent from workspace");

    const auto failed_request = request(base, "failed-producer");
    const auto failed = d::partitioned_result(
        failed_request, "generate-census", k::no_deadline,
        [&](d::PartitionedOutput &context) -> l::Value {
          auto spool = context.spool("census-signals");
          spool->append(Json{{"signal_id", "first"}});
          throw std::runtime_error("provider failed after emitted row");
        });
    require(failed.at("code") == "sbv.partitioned_result_incomplete" &&
                failed.at("recovery").at("phase") == "produce" &&
                failed.at("recovery").at("workspace_created") == true &&
                failed.at("recovery").at("workspace_creation_durable") ==
                    true &&
                failed.at("recovery").at("final_storage").is_null(),
            "provider failure yields bounded typed workspace recovery");
    require(
        failed.at("request_sha256") == k::sha256_hex(failed_request.dump()) &&
            failed.at("recovery").at("output") == failed_request.at("output"),
        "recovery binds exact request and output");
    require(failed.at("cause") == Json{{"category", "unexpected"},
                                       {"code", "sbv.unexpected_exception"}} &&
                failed.dump().find("provider failed after") ==
                    std::string::npos,
            "bounded cause excludes arbitrary provider exception text");
    require(std::filesystem::is_directory(base + "/failed-producer-work") &&
                !std::filesystem::exists(base + "/failed-producer-result"),
            "failed work retained and final directory absent");

    const auto invalid = d::partitioned_result(
        request(base, "invalid-result"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) {
          auto b = body();
          b["sections"].erase("execution");
          return l::Value(std::move(b));
        });
    require(invalid.at("recovery").at("phase") == "finalize_result" &&
                !invalid.at("recovery").at("final_storage").is_null() &&
                invalid.at("recovery")
                        .at("final_storage")
                        .at("manifest_published") == false,
            "final writer semantic failure retains its exact recovery");
    require(
        !std::filesystem::exists(base + "/invalid-result-result/manifest.json"),
        "invalid final result never publishes root");
    require(invalid.at("cause") ==
                Json{{"category", "storage"}, {"code", "bundle.contract"}},
            "native storage cause preserved");
    const auto contract = d::partitioned_result(
        request(base, "contract-cause"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) -> l::Value {
          throw k::Error("sbv.contract", "arbitrary private message", 2);
        });
    require(contract.at("cause") ==
                Json{{"category", "contract"}, {"code", "sbv.contract"}},
            "native contract cause preserved");
    const auto deadline = d::partitioned_result(
        request(base, "deadline-cause"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) -> l::Value {
          throw k::Error("deadline.exceeded", "arbitrary private message", 3);
        });
    require(deadline.at("cause") ==
                Json{{"category", "deadline"}, {"code", "deadline.exceeded"}},
            "native deadline cause preserved");
    const auto allocation = d::partitioned_result(
        request(base, "allocation-cause"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) -> l::Value { throw std::bad_alloc(); });
    require(allocation.at("cause") == Json{{"category", "allocation"},
                                           {"code", "sbv.allocation_failed"}},
            "allocation cause uses fixed code");
    const auto invalid_code = d::partitioned_result(
        request(base, "invalid-code"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) -> l::Value {
          throw s::StoreError(std::string(129, 'x'), "private-message");
        });
    require(invalid_code.at("cause") ==
                Json{{"category", "unexpected"},
                     {"code", "sbv.invalid_native_error_code"}},
            "oversized symbolic code replaced without truncation");
    const auto nonstandard = d::partitioned_result(
        request(base, "nonstandard-cause"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) -> l::Value { throw 42; });
    require(nonstandard.at("cause") == Json{{"category", "unexpected"},
                                            {"code", "sbv.unknown_exception"}},
            "nonstandard exception uses fixed cause");
    const auto oversize = d::partitioned_result(
        request(base, "oversize-summary"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) {
          auto b = body();
          for (std::size_t i = 0; i < 18; ++i)
            b["sections"]["summary"]["data"]["field-" + std::to_string(i)] =
                std::string(65536, 's');
          return l::Value(std::move(b));
        });
    require(oversize.at("status") == "recovery_required" &&
                oversize.at("recovery").at("phase") == "finalize_result" &&
                oversize.at("recovery").at("final_storage").is_null() &&
                !std::filesystem::exists(base + "/oversize-summary-result"),
            "summary byte framing refused before final writer creation");
    const auto too_many = d::partitioned_result(
        request(base, "many-summary"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &) {
          auto b = body();
          b["sections"]["summary"]["data"]["values"] = Json::array();
          for (std::size_t i = 0; i < 33000; ++i)
            b["sections"]["summary"]["data"]["values"].push_back("");
          return l::Value(std::move(b));
        });
    require(too_many.at("recovery").at("final_storage").is_null() &&
                !std::filesystem::exists(base + "/many-summary-result"),
            "summary value framing refused before final writer creation");

    bool called = false;
    auto callback = [&](d::PartitionedOutput &) {
      called = true;
      return l::Value(body());
    };
    auto both = request(base, "both");
    both["output_path"] = base + "/legacy.json";
    rejects(
        [&] {
          d::partitioned_result(both, "generate-census", k::no_deadline,
                                callback);
        },
        "conflicting output forms reject before mutation");
    require(!called && !std::filesystem::exists(base + "/both-work"),
            "static conflict invokes no producer");
    auto identical = request(base, "identical");
    identical["output"]["bundle_path"] = identical["output"]["workspace_path"];
    rejects(
        [&] {
          d::partitioned_result(identical, "generate-census", k::no_deadline,
                                callback);
        },
        "same destination rejected");
    auto nested = request(base, "nested");
    nested["output"]["bundle_path"] = base + "/nested-work/child";
    rejects(
        [&] {
          d::partitioned_result(nested, "generate-census", k::no_deadline,
                                callback);
        },
        "ancestor destination rejected");
    auto long_path = request(base, "long");
    long_path["output"]["workspace_path"] = "/" + std::string(4030, 'w');
    rejects(
        [&] {
          d::partitioned_result(long_path, "generate-census", k::no_deadline,
                                callback);
        },
        "all private derived paths preflighted");
    require(!called && !std::filesystem::exists(base + "/identical-work") &&
                !std::filesystem::exists(base + "/nested-work"),
            "static destination checks have no workspace side effect");
    std::filesystem::create_directory(base + "/actual-parent");
    std::filesystem::create_directory_symlink(base + "/actual-parent",
                                              base + "/alias-parent");
    auto symlink = request(base, "symlink");
    symlink["output"]["workspace_path"] = base + "/alias-parent/work";
    rejects(
        [&] {
          d::partitioned_result(symlink, "generate-census", k::no_deadline,
                                callback);
        },
        "symlinked parent rejected");
    require(!called && !std::filesystem::exists(base + "/actual-parent/work"),
            "symlink preflight invokes no producer or mutation");
    const auto bad_name = d::partitioned_result(
        request(base, "bad-name"), "generate-census", k::no_deadline,
        [](d::PartitionedOutput &context) {
          (void)context.spool("../escape");
          return l::Value(body());
        });
    require(bad_name.at("recovery").at("phase") == "produce" &&
                !std::filesystem::exists(base + "/escape"),
            "private spool names cannot escape workspace");
    for (const auto &[workspace_leaf, bundle_leaf] :
         {std::pair<std::string, std::string>{"case-alias", "CASE-ALIAS"},
          std::pair<std::string, std::string>{"unicode-é", "unicode-é"}}) {
      auto alias = request(base, "unused");
      alias["output"]["workspace_path"] = base + "/" + workspace_leaf;
      alias["output"]["bundle_path"] = base + "/" + bundle_leaf;
      bool producer_called = false;
      const auto result =
          d::partitioned_result(alias, "generate-census", k::no_deadline,
                                [&](d::PartitionedOutput &) {
                                  producer_called = true;
                                  return l::Value(body());
                                });
      if (producer_called)
        require(result.at("status") == "partial",
                "distinct filesystem leaves remain permitted");
      else
        require(result.at("status") == "recovery_required" &&
                    result.at("recovery").at("phase") == "create_workspace" &&
                    result.at("recovery").at("workspace_created") == true &&
                    result.at("recovery").at("final_storage").is_null(),
                "actual filesystem alias caught before producer");
    }
    std::cout << Json{{"status", "passed"},
                      {"checks", std::to_string(checks)},
                      {"scratch", base},
                      {"success", normal},
                      {"producer_failure", failed},
                      {"finalization_failure", invalid}}
                     .dump()
              << '\n';
    return 0;
  } catch (const std::exception &error) {
    std::cerr << error.what() << '\n';
    return 1;
  }
}
