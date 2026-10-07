#include "public_fixture.hpp"
#include "source_owners.hpp"
#include <csignal>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <source_location>
#include <symphony/knowledge/engine/path.hpp>
#include <symphony/sqav/databento/dbn.hpp>
#include <sys/resource.h>
#include <sys/stat.h>
#include <unistd.h>

using namespace symphony;
namespace s = sbv;
namespace d = sbv::detail;
namespace e = knowledge::engine;
namespace db = sqav::databento;
using J = s::Json;
unsigned checks = 0;
void check(bool ok, std::source_location at = std::source_location::current()) {
  ++checks;
  if (!ok)
    throw std::runtime_error("check failed at " + std::to_string(at.line()));
}
template <class F> void rejects(F f) {
  bool caught = false;
  try {
    f();
  } catch (const e::Error &) {
    caught = true;
  }
  check(caught);
}
J read_json(const std::string &p) {
  return J::parse(d::read_file(p, e::no_deadline));
}
void save(const std::string &p, const J &j) {
  d::create_file(p, j.dump(), e::no_deadline);
}
std::string text(sqav::ByteView b) {
  return {reinterpret_cast<const char *>(b.data()), b.size()};
}
void directory(const std::string &p) { check(::mkdir(p.c_str(), 0700) == 0); }
struct FileSizeLimit {
  struct rlimit before{};
  using Handler = void (*)(int);
  Handler prior = SIG_DFL;
  explicit FileSizeLimit(rlim_t maximum) {
    check(::getrlimit(RLIMIT_FSIZE, &before) == 0);
    prior = std::signal(SIGXFSZ, SIG_IGN);
    check(prior != SIG_ERR);
    auto selected_limit = before;
    selected_limit.rlim_cur = maximum;
    check(::setrlimit(RLIMIT_FSIZE, &selected_limit) == 0);
  }
  ~FileSizeLimit() {
    (void)::setrlimit(RLIMIT_FSIZE, &before);
    (void)std::signal(SIGXFSZ, prior);
  }
};
J request(const std::string &root, const std::string &raw,
          const db::Metadata &m) {
  return {
      {"protocol", "symphony.sbv.source-retain-input.v1"},
      {"output_path", root + "/source.json"},
      {"source",
       {{"path", root + "/original.dbn"},
        {"expected_sha256", e::sha256_hex(raw)},
        {"dataset", m.dataset}}},
      {"capture_description",
       {{"source",
         {{"provider_ref", "databento"},
          {"interface_ref", "retained-local-DBN-file"},
          {"interface_version", "offline-import.v1"},
          {"operation", "offline-file-admission"},
          {"adapter_ref", "caller-declaration"},
          {"adapter_version", "caller-declaration"},
          {"dataset_id", m.dataset},
          {"dataset_revision", "provider-revision-unspecified"},
          {"selection_ref", "source-sha256:" + e::sha256_hex(raw)},
          {"native_schema_ref", db::native_schema},
          {"native_encoding_ref", db::encoding_for_version(m.version)},
          {"access_scope", "private:user-research"}}},
        {"attempt_id", "retained-adapter-fixture"},
        {"attribution_ref", "caller:SBV-offline-tests"},
        {"source_position", ""},
        {"coverage", "partial"},
        {"coverage_scope",
         "record-limited retained source; not a complete market interval"},
        {"coverage_evidence_ref", "native test declaration"},
        {"source_record_count", nullptr},
        {"times",
         J::array({{{"role", "acquisition"},
                    {"value", "2026-10-07T03:00:00Z"},
                    {"format_ref", "ISO8601"},
                    {"clock_ref", "fixture-declaration"},
                    {"precision_ref", "second"},
                    {"evidence_ref", "offline admission fixture time; not "
                                     "provider download time"}}})}}},
      {"access_evidence",
       {{"producer_ref", "caller:user-research"},
        {"evidence_ref", "retained source; no new authorization assertion"}}},
      {"owner_profile", "sqv-local-retained-capture.v1"},
      {"limits",
       {{"dbn",
         {{"max_file_bytes", "8388608"},
          {"max_metadata_bytes", "1048576"},
          {"max_records", "1048576"}}},
        {"capture",
         {{"max_capture_bytes", "8388608"},
          {"max_metadata_bytes", "16384"},
          {"max_field_bytes", "4096"}}},
        {"metadata",
         {{"max_manifest_bytes", "65536"},
          {"max_field_bytes", "4096"},
          {"max_evidence_refs", "128"}}},
        {"flow",
         {{"max_payload_bytes", "8388608"},
          {"max_frame_bytes", "8396800"},
          {"max_descriptor_bytes", "4096"},
          {"global_allocation_bytes", "67108864"},
          {"max_ports", "4"}}},
        {"store",
         {{"max_frame_bytes", "8396800"},
          {"max_store_bytes", "33554432"},
          {"max_batches", "4"}}}}},
      {"retention",
       {{"mode", "create"},
        {"root", root + "/store"},
        {"partition", "selected-partition"},
        {"producer_generation", "00000000000000000000000000000001"},
        {"store_generation", "00000000000000000000000000000002"},
        {"first_sequence", "7"},
        {"batch_sequence", "7"}}},
      {"extensions", J::object()}};
}
J feed(const std::string &path, const J &result) {
  return {{"reference",
           {{"path", path},
            {"expected_sha256", result.at("content_sha256")},
            {"pointer", "/sections/source/data"}}},
          {"delivery",
           {{"profile", "retained_before_delivery"},
            {"view_id", "selected-view"},
            {"recipient_id", "sbv-test-dataset"},
            {"recipient_interface", "symphony.sbv.dataset-feed.v1"},
            {"first_sequence", "7"},
            {"outstanding_byte_credit", "8388608"},
            {"max_unacknowledged_batches", "1"},
            {"processed_ack", "dataset_admitted"},
            {"checkpoint", nullptr}}}};
}
J exercise(const std::string &root, const std::string &raw, bool adversarial) {
  directory(root);
  directory(root + "/store");
  d::create_file(root + "/original.dbn", raw, e::no_deadline);
  db::FileView original;
  const auto byte_span = sqav::ByteView(
      reinterpret_cast<const std::uint8_t *>(raw.data()), raw.size());
  check(db::FileView::inspect_dataset(byte_span, {}, original) ==
        db::Status::ok);
  auto p = request(root, raw, original.metadata());
  save(root + "/request.json", p);
  const auto receipt = d::source_retain(p, e::no_deadline);
  check(receipt.at("protocol") == "symphony.sbv.source-retain.v1");
  const auto result = read_json(root + "/source.json");
  s::validate_result(result);
  check(result.at("origin") == "source_retain");
  auto selection = feed(root + "/source.json", result);
  save(root + "/feed.json", selection);
  const auto &source = result.at("sections").at("source").at("data");
  check(source.at("original").at("source_sha256") == e::sha256_hex(raw));
  check(source.at("capture").at("description").at("source").at("adapter_ref") ==
        db::adapter_id);
  check(source.at("batch").at("descriptor").at("record_count") == "1");
  check(result.at("sections")
            .at("choices")
            .at("data")
            .at("capture_description")
            .at("source_record_count")
            .is_null());
  auto again = p;
  again["retention"]["mode"] = "open";
  again["output_path"] = root + "/duplicate.json";
  (void)d::source_retain(again, e::no_deadline);
  const auto duplicate = read_json(root + "/duplicate.json")
                             .at("sections")
                             .at("source")
                             .at("data");
  check(duplicate.at("retention").at("append_status") == "duplicate");
  check(duplicate.at("retention").at("receipt") ==
        source.at("retention").at("receipt"));
  check(duplicate.at("retention").at("snapshot").at("committed_batches") ==
        "1");
  if (adversarial) {
    rejects([&] { (void)d::source_retain(p, e::no_deadline); });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/wrong-source.json";
      q["source"]["expected_sha256"] = std::string(64, '0');
      (void)d::source_retain(q, e::no_deadline);
    });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/wrong-count.json";
      q["capture_description"]["source_record_count"] = "999";
      (void)d::source_retain(q, e::no_deadline);
    });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/wrong-options.json";
      q["limits"]["store"]["max_batches"] = "5";
      (void)d::source_retain(q, e::no_deadline);
    });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/wrong-integer.json";
      q["limits"]["metadata"]["max_evidence_refs"] = "65536";
      (void)d::source_retain(q, e::no_deadline);
    });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/expired.json";
      (void)d::source_retain(q, 1);
    });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/store/invalid-sbv-result.json";
      (void)d::source_retain(q, e::no_deadline);
    });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/store";
      (void)d::source_retain(q, e::no_deadline);
    });
    rejects([&] {
      auto q = again;
      q["output_path"] = root + "/oversize-partition.json";
      q["retention"]["partition"] = std::string(4097, 'p');
      q["limits"]["flow"]["max_descriptor_bytes"] = "65536";
      (void)d::source_retain(q, e::no_deadline);
    });
    if (std::filesystem::exists(root + "/STORE") &&
        std::filesystem::equivalent(root + "/store", root + "/STORE")) {
      rejects([&] {
        auto q = again;
        q["output_path"] = root + "/STORE/alias-result.json";
        (void)d::source_retain(q, e::no_deadline);
      });
      check(!std::filesystem::exists(root + "/store/alias-result.json"));
    }
    directory(root + "/\xc3\xa9-store");
    const auto unicode_alias = root + "/e\xcc\x81-store";
    if (std::filesystem::exists(unicode_alias) &&
        std::filesystem::equivalent(root + "/\xc3\xa9-store", unicode_alias)) {
      rejects([&] {
        auto q = p;
        q["retention"]["root"] = root + "/\xc3\xa9-store";
        q["output_path"] = unicode_alias + "/alias-result.json";
        (void)d::source_retain(q, e::no_deadline);
      });
      check(std::filesystem::is_empty(root + "/\xc3\xa9-store"));
    }
    auto appended = again;
    appended["output_path"] = root + "/second-batch.json";
    appended["retention"]["batch_sequence"] = "8";
    (void)d::source_retain(appended, e::no_deadline);
    check(read_json(root + "/second-batch.json")
              .at("sections")
              .at("source")
              .at("data")
              .at("retention")
              .at("snapshot")
              .at("committed_batches") == "2");
    // The immutable older descriptor still refers to the actual retained first
    // batch after the same store grows; its snapshot is historical evidence.
  }
  check(std::filesystem::remove(root + "/original.dbn"));
  // No manifest artifact was ever written by source_retain. Admission uses only
  // the selected sealed descriptor plus actual store and owner libraries.
  d::RetainedOriginal admitted(selection, e::no_deadline);
  check(text(admitted.original_bytes()) == raw);
  check(admitted.identity() == J{{"source_path", root + "/original.dbn"},
                                 {"source_sha256", e::sha256_hex(raw)},
                                 {"dataset", original.metadata().dataset}});
  db::FileView replay;
  check(db::FileView::inspect_dataset(admitted.original_bytes(), {}, replay) ==
        db::Status::ok);
  // Equivalent of the production Dataset byte ownership boundary.
  std::vector<db::Mbo> owned;
  owned.resize(static_cast<std::size_t>(replay.metadata().record_count));
  for (std::uint64_t i = 0; i < replay.metadata().record_count; ++i) {
    db::Mbo expected;
    check(replay.record(i, owned[i]) == db::Status::ok &&
          original.record(i, expected) == db::Status::ok &&
          owned[i] == expected);
  }
  d::RetainedOriginal moved(std::move(admitted));
  rejects([&] { (void)admitted.original_bytes(); });
  const auto delivered = moved.after_admission();
  check(delivered.at("taken").at("unacknowledged_batches") == "1");
  check(
      delivered.at("after_ack_before_release").at("next_processed_sequence") ==
      "8");
  check(delivered.at("after_ack_before_release").at("outstanding_bytes") ==
        source.at("capture").at("bytes"));
  check(delivered.at("after_release").at("outstanding_bytes") == "0");
  check(delivered.at("after_release").at("unacknowledged_batches") == "0");
  check(delivered.at("checkpoint_persistence").at("durable") == false);
  check(d::retained_delivery_identity(selection, delivered) ==
        moved.identity());
  if (adversarial) {
    auto bad_delivery = delivered;
    bad_delivery["after_release"]["unacknowledged_batches"] = "1";
    rejects(
        [&] { (void)d::retained_delivery_identity(selection, bad_delivery); });
    bad_delivery = delivered;
    bad_delivery["choices"]["view_id"] = "different";
    rejects(
        [&] { (void)d::retained_delivery_identity(selection, bad_delivery); });
    std::filesystem::rename(root + "/store",
                            root + "/store-offline-helper-proof");
    struct Restore {
      std::string root;
      ~Restore() {
        std::error_code error;
        std::filesystem::rename(root + "/store-offline-helper-proof",
                                root + "/store", error);
      }
    } restore{root};
    check(d::retained_delivery_identity(selection, delivered) ==
          moved.identity());
  }
  rejects([&] { (void)moved.after_admission(); });
  rejects([&] { (void)moved.original_bytes(); });
  save(root + "/delivery.json", delivered);
  auto no_ack = selection;
  no_ack["delivery"]["processed_ack"] = "none";
  {
    d::RetainedOriginal reading(no_ack, e::no_deadline);
    const auto state = reading.after_admission();
    check(d::retained_delivery_identity(no_ack, state) == reading.identity());
    check(state.at("after_release").at("unacknowledged_batches") == "1");
    check(state.at("final_checkpoint").at("next_sequence") == "7");
    save(root + "/delivery-no-ack.json", state);
  }
  {
    d::RetainedOriginal abandoned(selection, e::no_deadline);
    check(text(abandoned.original_bytes()) == raw);
  }
  {
    d::RetainedOriginal resumed(selection, e::no_deadline);
    const auto state = resumed.after_admission();
    check(state.at("baseline_checkpoint").at("next_sequence") == "7");
  }
  for (const std::string kind : {"original", "capture", "manifest"}) {
    J exporting{{"protocol", "symphony.sbv.source-export-input.v1"},
                {"retained_source", no_ack},
                {"kind", kind},
                {"output_path", root + "/export-" + kind + ".bin"},
                {"receipt_path", root + "/export-" + kind + ".json"},
                {"extensions", J::object()}};
    const auto exported = d::source_export(exporting, e::no_deadline);
    check(exported.at("protocol") == "symphony.sbv.source-export.v1");
    const auto export_result = read_json(exported.at("path"));
    s::validate_result(export_result);
    const auto &binary = export_result.at("sections").at("binary").at("data");
    const auto contents = d::read_file(binary.at("path"), e::no_deadline);
    check(binary.at("sha256") == e::sha256_hex(contents));
    check(binary.at("bytes") == d::dec(contents.size()));
    check(export_result.at("sections")
              .at("delivery")
              .at("data")
              .at("after_release")
              .at("unacknowledged_batches") == "1");
    check(export_result.at("sections")
              .at("delivery")
              .at("data")
              .at("final_checkpoint")
              .at("next_sequence") == "7");
    if (kind == "original")
      check(contents == raw);
    else
      check(binary.at("sha256") ==
            source.at(kind == "capture" ? "capture" : "metadata")
                .at("encoded_sha256"));
    rejects([&] { (void)d::source_export(exporting, e::no_deadline); });
    check(d::read_file(binary.at("path"), e::no_deadline) == contents);
  }
  if (adversarial) {
    rejects([&] {
      auto q = selection;
      q["reference"]["expected_sha256"] = std::string(64, '0');
      d::RetainedOriginal x(q, e::no_deadline);
    });
    rejects([&] {
      auto q = selection;
      q["delivery"]["checkpoint"] = J::object();
      d::RetainedOriginal x(q, e::no_deadline);
    });
    rejects([&] {
      auto q = selection;
      q["delivery"]["first_sequence"] = "8";
      d::RetainedOriginal x(q, e::no_deadline);
    });
    rejects([&] {
      auto q = selection;
      q["delivery"]["processed_ack"] = "backtest_completed";
      d::RetainedOriginal x(q, e::no_deadline);
    });
    rejects([&] {
      auto q = selection;
      q["delivery"]["outstanding_byte_credit"] = "1";
      d::RetainedOriginal x(q, e::no_deadline);
    });
    rejects([&] { d::RetainedOriginal x(selection, 1); });
    rejects([&] {
      (void)d::source_export(
          {{"protocol", "symphony.sbv.source-export-input.v1"},
           {"retained_source", selection},
           {"kind", "original"},
           {"output_path", root + "/export-ack-invalid.bin"},
           {"receipt_path", root + "/export-ack-invalid.json"},
           {"extensions", J::object()}},
          e::no_deadline);
    });
    rejects([&] {
      (void)d::source_export(
          {{"protocol", "symphony.sbv.source-export-input.v1"},
           {"retained_source", no_ack},
           {"kind", "original"},
           {"output_path", root + "/store/export-invalid.bin"},
           {"receipt_path", root + "/export-store-invalid.json"},
           {"extensions", J::object()}},
          e::no_deadline);
    });
    if (std::filesystem::exists(root + "/STORE") &&
        std::filesystem::equivalent(root + "/store", root + "/STORE")) {
      rejects([&] {
        (void)d::source_export(
            {{"protocol", "symphony.sbv.source-export-input.v1"},
             {"retained_source", no_ack},
             {"kind", "original"},
             {"output_path", root + "/STORE/alias-export.bin"},
             {"receipt_path", root + "/alias-export.json"},
             {"extensions", J::object()}},
            e::no_deadline);
      });
      check(!std::filesystem::exists(root + "/store/alias-export.bin"));
    }
    // Constrain only this test process's output file size. The small manifest
    // binary publishes durably, but the much larger JSON receipt cannot.
    // Restore both resource policy and signal disposition before inspecting
    // evidence.
    J reconciliation;
    const J canonical_probe{
        {"native_canonical_probe",
         "HTML <>&; Unicode \xe2\x80\xa8\xe2\x80\xa9\xc3\xa9; literal \\u2028; "
         "control \n\t"}};
    const J partial_request{
        {"protocol", "symphony.sbv.source-export-input.v1"},
        {"retained_source", no_ack},
        {"kind", "manifest"},
        {"output_path", root + "/partial-publication.sqmv"},
        {"receipt_path", root + "/partial-publication.json"},
        {"extensions", canonical_probe}};
    {
      FileSizeLimit limit(4096);
      reconciliation = d::source_export(partial_request, e::no_deadline);
    }
    d::keys(reconciliation,
            {"protocol", "status", "code", "stage", "request_sha256",
             "output_path", "receipt_path", "automatic_retry",
             "rollback_performed", "recovery"});
    check(reconciliation.at("status") == "recovery_required" &&
          reconciliation.at("code") == "sbv.source_export_unpublished" &&
          reconciliation.at("stage") == "receipt_publication");
    check(reconciliation.at("request_sha256") ==
          e::sha256_hex(partial_request.dump()));
    check(reconciliation.at("recovery").at("binary_publication_confirmed") ==
              true &&
          reconciliation.at("automatic_retry") == false &&
          reconciliation.at("rollback_performed") == false);
    check(!std::filesystem::exists(root + "/partial-publication.json"));
    check(e::sha256_hex(d::read_file(root + "/partial-publication.sqmv",
                                     e::no_deadline)) ==
          d::str(source.at("metadata").at("encoded_sha256")));
    save(root + "/partial-publication-request.json", partial_request);
    save(root + "/partial-publication-recovery.json", reconciliation);
    auto before_binary = partial_request;
    before_binary["output_path"] = root + "/failed-binary.sqmv";
    before_binary["receipt_path"] = root + "/failed-binary.json";
    {
      FileSizeLimit limit(1);
      reconciliation = d::source_export(before_binary, e::no_deadline);
    }
    check(reconciliation.at("stage") == "binary_publication" &&
          reconciliation.at("recovery").at("binary_publication_confirmed") ==
              false);
    check(reconciliation.at("request_sha256") ==
          e::sha256_hex(before_binary.dump()));
    check(!std::filesystem::exists(root + "/failed-binary.sqmv") &&
          !std::filesystem::exists(root + "/failed-binary.json"));
    save(root + "/failed-binary-request.json", before_binary);
    save(root + "/failed-binary-recovery.json", reconciliation);
    const auto retained_fault = root + "/retain-recovery";
    directory(retained_fault);
    directory(retained_fault + "/store");
    d::create_file(retained_fault + "/original.dbn", raw, e::no_deadline);
    auto retaining = request(retained_fault, raw, original.metadata());
    retaining["extensions"] = canonical_probe;
    {
      FileSizeLimit limit(4096);
      reconciliation = d::source_retain(retaining, e::no_deadline);
    }
    d::keys(reconciliation, {"protocol", "status", "code", "stage",
                             "request_sha256", "output_path", "automatic_retry",
                             "rollback_performed", "recovery"});
    check(reconciliation.at("status") == "recovery_required" &&
          reconciliation.at("code") == "sbv.source_retention_unpublished" &&
          reconciliation.at("stage") == "result_publication");
    check(reconciliation.at("request_sha256") ==
              e::sha256_hex(retaining.dump()) &&
          reconciliation.at("recovery").at("retention_confirmed") == true);
    check(!reconciliation.at("recovery").at("receipt").is_null() &&
          !std::filesystem::exists(retained_fault + "/source.json"));
    save(retained_fault + "/recovery-request.json", retaining);
    save(retained_fault + "/recovery-payload.json", reconciliation);
    retaining["retention"]["mode"] = "open";
    const auto retried = d::source_retain(retaining, e::no_deadline);
    check(retried.at("status") == "completed");
    const auto restored = read_json(retained_fault + "/source.json");
    check(restored.at("sections")
              .at("source")
              .at("data")
              .at("retention")
              .at("receipt") == reconciliation.at("recovery").at("receipt"));
    unsigned variant = 0;
    auto altered = [&](auto mutate) {
      auto bad = result;
      mutate(bad["sections"]["source"]["data"]);
      bad = s::seal_result(std::move(bad));
      const auto path =
          root + "/altered-" + std::to_string(++variant) + ".json";
      save(path, bad);
      rejects([&] { d::RetainedOriginal x(feed(path, bad), e::no_deadline); });
    };
    altered(
        [](J &x) { x["original"]["source_sha256"] = std::string(64, '0'); });
    altered([](J &x) { x["original"]["record_count"] = "999"; });
    altered(
        [](J &x) { x["metadata"]["encoded_sha256"] = std::string(64, '0'); });
    altered([](J &x) { x["metadata"]["description"]["dataset_id"] = "wrong"; });
    altered([](J &x) { x["capture"]["description"]["coverage"] = "complete"; });
    altered([](J &x) { x["access_evidence"]["evidence_ref"] = "different"; });
    altered([](J &x) {
      x["retention"]["receipt"]["commit_sha256"] = std::string(64, '0');
    });
    altered([](J &x) { x["batch"]["descriptor"]["record_count"] = "1000"; });
    altered([](J &x) { x["batch"]["descriptor"].erase("batch_sequence"); });
    altered(
        [](J &x) { x["retention"]["options"]["limits"]["max_batches"] = "5"; });
    altered(
        [](J &x) { x["owner_versions"]["sqdv-delivery-cpp"] = "0.4.0-dev"; });
    altered([](J &x) { x["delivery"]["status"] = "processed"; });
    altered([](J &x) { x["retention"]["snapshot"]["next_sequence"] = "999"; });
    altered([](J &x) {
      x["retention"]["snapshot"]["committed_batches"] = "3";
      x["retention"]["snapshot"]["next_sequence"] = "10";
    });
    altered([](J &x) { x["retention"]["snapshot"]["store_bytes"] = "1"; });
    altered([](J &x) {
      x["retention"]["snapshot"]["committed_batches"] = "2";
      x["retention"]["snapshot"]["next_sequence"] = "9";
    });
    // Embedded source records retain the outer immutable result identity.
    auto outer = d::base("test_wrapper");
    outer["sections"]["source"] = d::section(J{{"nested", source}});
    outer = s::seal_result(std::move(outer));
    save(root + "/embedded.json", outer);
    auto nested = feed(root + "/embedded.json", outer);
    nested["reference"]["pointer"] = "/sections/source/data/nested";
    d::RetainedOriginal embedded(nested, e::no_deadline);
    check(text(embedded.original_bytes()) == raw);
    (void)embedded.after_admission();
  }
  return {{"root", root},
          {"status", "passed"},
          {"source_sha256", e::sha256_hex(raw)},
          {"records", d::dec(owned.size())},
          {"source_removed", true},
          {"manifest_ancestor_required", false},
          {"case_alias_test",
           adversarial && std::filesystem::exists(root + "/STORE") &&
                   std::filesystem::equivalent(root + "/store", root + "/STORE")
               ? "passed"
               : "not_applicable"},
          {"unicode_alias_test",
           adversarial && std::filesystem::exists(root + "/e\xcc\x81-store") &&
                   std::filesystem::equivalent(root + "/\xc3\xa9-store",
                                               root + "/e\xcc\x81-store")
               ? "passed"
               : "not_applicable"},
          {"binary_published_receipt_failed_test",
           adversarial ? "passed" : "not_selected"},
          {"exactly_once_claimed", false}};
}
int main(int argc, char **argv) try {
  check(argc == 1 || argc == 2 || argc == 4);
  std::string root;
  if (argc == 1) {
#ifdef __APPLE__
    char generated[] = "/private/tmp/sbvo-XXXXXX";
#else
    char generated[] = "/tmp/sbvo-XXXXXX";
#endif
    check(::mkdtemp(generated) != nullptr);
    root = generated;
  } else {
    root = argv[1];
    directory(root);
  }
  const auto fixture = public_fixture::fixture();
  const std::string raw(reinterpret_cast<const char *>(fixture.data()),
                        fixture.size());
  J rows = J::array({exercise(root + "/mechanical", raw, true)});
  // Last representable owner sequence is valid and exhausts explicitly.
  {
    const auto edge = root + "/last-sequence";
    directory(edge);
    directory(edge + "/store");
    d::create_file(edge + "/original.dbn", raw, e::no_deadline);
    db::FileView view;
    check(db::FileView::inspect_dataset(
              {reinterpret_cast<const std::uint8_t *>(raw.data()), raw.size()},
              {}, view) == db::Status::ok);
    auto p = request(edge, raw, view.metadata());
    p["retention"]["first_sequence"] = d::dec(UINT64_MAX);
    p["retention"]["batch_sequence"] = d::dec(UINT64_MAX);
    (void)d::source_retain(p, e::no_deadline);
    const auto r = read_json(edge + "/source.json");
    auto selection = feed(edge + "/source.json", r);
    selection["delivery"]["first_sequence"] = d::dec(UINT64_MAX);
    d::RetainedOriginal reading(selection, e::no_deadline);
    const auto outcome = reading.after_admission();
    check(d::retained_delivery_identity(selection, outcome) ==
          reading.identity());
    check(outcome.at("final_checkpoint").at("sequence_exhausted") == true);
    check(outcome.at("final_checkpoint").at("next_sequence") ==
          d::dec(UINT64_MAX));
    save(edge + "/delivery.json", outcome);
  }
  for (int i = 2; i < argc; ++i) {
    const std::string input = argv[i];
    check(input.starts_with('/'));
    const auto sample = e::read_regular_file_no_follow(
        "/", input.substr(1), 8U << 20, e::no_deadline);
    const auto expected =
        i == 2
            ? "a7e0a9b8cdf10405cf9d7bf37b721d4376916907988f3f3724474126871ce217"
            : "76073879167564256ce1cd1544c84cdb88041008534b7f2907b235b20be200a"
              "0";
    check(e::sha256_hex(sample) == expected);
    rows.push_back(
        exercise(root + (i == 2 ? "/AAPL" : "/ESZ6"), sample, false));
  }
  const J report{{"protocol", "research.sbv.source-owners-adapter-test.v1"},
                 {"status", "passed"},
                 {"checks", d::dec(checks)},
                 {"cases", rows}};
  save(root + "/report.json", report);
  std::cout << report.dump(2) << '\n';
  return 0;
} catch (const std::exception &error) {
  std::cerr << "source-owners adapter tests failed after " << checks
            << " checks: " << error.what() << '\n';
  return 1;
}
