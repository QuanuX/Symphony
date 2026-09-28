#include <symphony/sqdv/checkpoint.hpp>

#include <algorithm>
#include <array>
#include <cerrno>
#include <filesystem>
#include <iostream>
#include <limits>
#include <stdexcept>
#include <string>
#include <system_error>
#include <unistd.h>
#include <vector>

namespace sqdv = symphony::sqdv;
namespace sqpv = symphony::sqpv;
namespace sqmv = symphony::sqmv;
namespace sqfv = symphony::sqfv;

namespace {
template <typename Status>
bool expect(Status actual, Status wanted, const char* label) {
  if (actual == wanted) return true;
  std::cerr << label << ": expected " << static_cast<int>(wanted)
            << ", received " << static_cast<int>(actual) << '\n';
  return false;
}

struct TemporaryRoot {
  std::filesystem::path path;
  TemporaryRoot() {
    std::string name = (std::filesystem::temp_directory_path() /
                        "sqdv-installed-consumer-XXXXXX").string();
    std::vector<char> writable(name.begin(), name.end());
    writable.push_back('\0');
    if (!::mkdtemp(writable.data()))
      throw std::system_error(errno, std::generic_category(), "mkdtemp");
    path = std::filesystem::canonical(writable.data());
  }
  ~TemporaryRoot() {
    std::error_code ignored;
    std::filesystem::remove_all(path, ignored);
  }
};

sqmv::Manifest metadata(std::string revision = "revision-7") {
  sqmv::Description description{
      "fixture-dataset", std::move(revision), "schema-1", "row-v1",
      "fixture-scope", "fixture-producer",
      {{sqmv::EvidenceRole::schema, "schema-owner", "schema-evidence-1"},
       {sqmv::EvidenceRole::layout, "layout-owner", "layout-evidence-1"},
       {sqmv::EvidenceRole::access, "access-owner", "scope-evidence-1"}}};
  sqmv::Manifest result;
  if (!expect(sqmv::Manifest::create(description, {4096, 256, 8}, result),
              sqmv::Status::ok, "fixture manifest"))
    throw std::runtime_error("manifest fixture failed");
  return result;
}

sqfv::Context context() {
  sqfv::Context result;
  if (!expect(sqfv::Context::create({4096, 8192, 2048, 1'048'576, 8}, result),
              sqfv::Status::ok, "fixture flow context"))
    throw std::runtime_error("context fixture failed");
  return result;
}

sqdv::Config config(std::uint64_t first = 10) {
  return {"fixture-view", "recipient-a", "fixture.receiver.v1", "partition-a",
          {1}, first, sqdv::Profile::disposable};
}

sqpv::Options store_options() {
  return {"partition-a", {1}, {7}, 10, {8192, 1'048'576, 16}};
}

sqfv::Batch batch(sqfv::Context& flow, const sqmv::Manifest& manifest,
                  std::uint64_t sequence, std::uint8_t value) {
  sqfv::Binding binding;
  if (!expect(manifest.binding(binding), sqmv::Status::ok, "fixture binding"))
    throw std::runtime_error("binding fixture failed");
  sqfv::Descriptor descriptor{binding, "partition-a", "fixture-source",
                              "position-" + std::to_string(sequence), {1}, sequence, 1};
  const std::array<std::uint8_t, 8> payload{value, 1, 2, 3, 4, 5, 6, 7};
  sqfv::Batch result;
  if (!expect(flow.prepare_copy(descriptor, payload, result), sqfv::Status::ok,
              "fixture immutable batch"))
    throw std::runtime_error("batch fixture failed");
  return result;
}

bool installed_delivery_identity_rejection() {
  TemporaryRoot root;
  TemporaryRoot different_root;
  auto manifest = metadata();
  auto flow = context();
  auto selected = config();
  selected.profile = sqdv::Profile::retained_before_delivery;
  const auto options = store_options();
  sqdv::RetainedSource source;
  if (!expect(sqdv::RetainedSource::create(root.path.string(), manifest, options, source),
              sqdv::Status::ok, "identity source")) return false;
  sqdv::Session session;
  if (!expect(sqdv::Session::create(flow, manifest, selected, {16, 2}, &source,
                                   nullptr, session), sqdv::Status::ok,
              "identity session")) return false;
  sqdv::Checkpoint resume;
  if (!expect(session.checkpoint(resume), sqdv::Status::ok, "identity checkpoint"))
    return false;
  sqdv::Session output;
  if (!expect(sqdv::Session::create(flow, manifest, config(99), {16, 2}, nullptr,
                                   nullptr, output), sqdv::Status::ok,
              "output sentinel session")) return false;
  const std::string output_reference(output.view_reference());
  const auto preserved = [&] {
    sqdv::SessionStats stats;
    return expect(output.stats(stats), sqdv::Status::ok, "output remains live") &&
           output.view_reference() == output_reference &&
           stats.next_offer_sequence == 99 && stats.next_processed_sequence == 99 &&
           stats.unacknowledged_batches == 0;
  };
  std::array<sqdv::Config, 5> changed{selected, selected, selected, selected, selected};
  changed[0].view_id = "other-view";
  changed[1].recipient_id = "other-recipient";
  changed[2].recipient_interface = "fixture.receiver.v2";
  changed[3].partition = "other-partition";
  ++changed[4].producer_generation[0];
  for (const auto& candidate : changed) {
    if (!expect(sqdv::Session::create(flow, manifest, candidate, {16, 2}, &source,
                                     &resume, output), sqdv::Status::binding_mismatch,
                "changed resume identity refused") || !preserved()) return false;
  }
  auto other_manifest = metadata("other-revision");
  if (!expect(sqdv::Session::create(flow, other_manifest, selected, {16, 2}, &source,
                                   &resume, output), sqdv::Status::binding_mismatch,
              "changed manifest refused") || !preserved()) return false;
  sqdv::RetainedSource other_source;
  if (!expect(sqdv::RetainedSource::create(different_root.path.string(), manifest,
                                          options, other_source), sqdv::Status::ok,
              "other source path fixture") ||
      !expect(sqdv::Session::create(flow, manifest, selected, {16, 2}, &other_source,
                                   &resume, output), sqdv::Status::binding_mismatch,
              "changed source path refused") || !preserved()) return false;
  // Replace only this consumer's closed, disposable fixture at the same path,
  // isolating the store generation from the source path in the resume identity.
  session.reset();
  source.reset();
  for (const auto& entry : std::filesystem::directory_iterator(root.path))
    std::filesystem::remove_all(entry.path());
  auto other_options = options;
  ++other_options.store_generation[0];
  if (!expect(sqdv::RetainedSource::create(root.path.string(), manifest, other_options,
                                          source), sqdv::Status::ok,
              "other store generation fixture")) return false;
  return expect(sqdv::Session::create(flow, manifest, selected, {16, 2}, &source,
                                     &resume, output), sqdv::Status::binding_mismatch,
                "changed store generation refused") && preserved();
}

bool installed_delivery_cutover_and_resume() {
  TemporaryRoot root;
  auto manifest = metadata();
  auto flow = context();
  auto selected = config();
  selected.profile = sqdv::Profile::retained_before_delivery;
  const auto options = store_options();
  sqdv::RetainedSource source;
  if (!expect(sqdv::RetainedSource::create(root.path.string(), manifest, options, source),
              sqdv::Status::ok, "cutover source")) return false;
  auto first = batch(flow, manifest, 10, 10);
  auto second = batch(flow, manifest, 11, 11);
  sqdv::RetainedBatch first_proof;
  sqdv::RetainedBatch second_proof;
  if (!expect(source.commit(flow, first, first_proof), sqdv::Status::ok,
              "first retention commit") ||
      !expect(source.commit(flow, second, second_proof), sqdv::Status::ok,
              "second retention commit")) return false;
  sqdv::Session session;
  if (!expect(sqdv::Session::create(flow, manifest, selected, {16, 2}, &source,
                                   nullptr, session), sqdv::Status::ok,
              "cutover session")) return false;
  // A future live proof must not skip the exact retained next position.
  if (!expect(session.offer_next(flow, &second_proof), sqdv::Status::ok,
              "retained exact-next fallback")) return false;
  sqdv::Delivery delivery;
  if (!expect(session.take(delivery), sqdv::Status::ok, "take retained first") ||
      delivery.sequence() != 10 || delivery.origin() != sqdv::Origin::retained ||
      !delivery.retention_receipt() || delivery.payload().size() != 8 ||
      delivery.payload().front() != 10 ||
      delivery.content_id() != first.content_id()) return false;
  if (!expect(session.acknowledge_processed(delivery), sqdv::Status::ok,
              "process retained first")) return false;
  delivery.reset();
  if (!expect(session.offer_next(flow, &second_proof), sqdv::Status::ok,
              "exact retained live candidate") ||
      !expect(session.take(delivery), sqdv::Status::ok, "take live second") ||
      delivery.sequence() != 11 || delivery.origin() != sqdv::Origin::live ||
      !delivery.retention_receipt() || delivery.payload().size() != 8 ||
      delivery.payload().front() != 11 ||
      delivery.retention_receipt()->commit_sha256 != second_proof.receipt().commit_sha256)
    return false;
  sqdv::Checkpoint resume;
  if (!expect(session.checkpoint(resume), sqdv::Status::ok, "pre-ack checkpoint") ||
      resume.next_sequence != 11 ||
      !expect(session.acknowledge_processed(delivery), sqdv::Status::ok,
              "process live second") ||
      !expect(session.checkpoint(resume), sqdv::Status::ok, "processed checkpoint") ||
      resume.next_sequence != 12 || resume.sequence_exhausted) return false;
  delivery.reset();
  session.reset();
  source.reset();
  if (!expect(sqdv::RetainedSource::open(root.path.string(), manifest, options, source),
              sqdv::Status::ok, "reopen exact retained source")) return false;
  auto third = batch(flow, manifest, 12, 12);
  sqdv::RetainedBatch third_proof;
  if (!expect(source.commit(flow, third, third_proof), sqdv::Status::ok,
              "third retention commit") ||
      !expect(sqdv::Session::create(flow, manifest, selected, {16, 2}, &source,
                                   &resume, session), sqdv::Status::ok,
              "resume exact processed checkpoint") ||
      !expect(session.offer_next(flow, &second_proof), sqdv::Status::binding_mismatch,
              "closed-source proof cannot authorize reopened source")) return false;
  sqdv::SessionStats resumed_stats;
  if (!expect(session.stats(resumed_stats), sqdv::Status::ok, "rejected proof cursor") ||
      resumed_stats.next_offer_sequence != 12 || resumed_stats.pending_deliveries != 0 ||
      !expect(session.offer_next(flow), sqdv::Status::ok,
              "resumed exact retained position") ||
      !expect(session.take(delivery), sqdv::Status::ok, "take resumed third") ||
      delivery.sequence() != 12 || delivery.origin() != sqdv::Origin::retained ||
      delivery.content_id() != third.content_id() || delivery.payload().size() != 8 ||
      delivery.payload().front() != 12)
    return false;
  return expect(session.acknowledge_processed(delivery), sqdv::Status::ok,
                "process resumed third") &&
         expect(session.checkpoint(resume), sqdv::Status::ok, "final resume position") &&
         resume.next_sequence == 13;
}

bool installed_delivery_ack_credit_and_terminal() {
  auto manifest = metadata();
  auto flow = context();
  auto selected = config();
  sqdv::Session session;
  sqdv::Session other_session;
  if (!expect(sqdv::Session::create(flow, manifest, selected, {8, 1}, nullptr,
                                   nullptr, session), sqdv::Status::ok,
              "one-slot credit session") ||
      !expect(sqdv::Session::create(flow, manifest, selected, {8, 1}, nullptr,
                                   nullptr, other_session), sqdv::Status::ok,
              "different opaque ticket owner")) return false;
  auto first = batch(flow, manifest, 10, 10);
  auto second = batch(flow, manifest, 11, 11);
  auto third = batch(flow, manifest, 12, 12);
  sqdv::Delivery delivery;
  if (!expect(session.offer_live(first), sqdv::Status::ok, "offer credit first") ||
      !expect(session.take(delivery), sqdv::Status::ok, "take credit first") ||
      !expect(other_session.acknowledge_processed(delivery), sqdv::Status::binding_mismatch,
              "wrong session ticket refused")) return false;
  sqdv::SessionStats other_stats;
  if (!expect(other_session.stats(other_stats), sqdv::Status::ok, "wrong owner unchanged") ||
      other_stats.next_processed_sequence != 10 || other_stats.unacknowledged_batches != 0 ||
      !expect(session.acknowledge_processed(delivery), sqdv::Status::ok,
              "ack while payload held")) return false;
  sqdv::SessionStats stats;
  if (!expect(session.stats(stats), sqdv::Status::ok, "ack stats") ||
      stats.outstanding_bytes != 8 || stats.unacknowledged_batches != 0 ||
      stats.next_processed_sequence != 11 || !delivery.has_payload() ||
      !expect(session.offer_live(second), sqdv::Status::blocked,
              "held acknowledged payload still consumes byte credit")) return false;
  delivery.release_payload();
  if (!expect(session.stats(stats), sqdv::Status::ok, "released acknowledged credit") ||
      stats.outstanding_bytes != 0 || stats.next_processed_sequence != 11 ||
      !expect(session.offer_live(second), sqdv::Status::ok, "offer after credit release") ||
      !expect(session.take(delivery), sqdv::Status::ok, "take second")) return false;
  delivery.release_payload();
  if (!expect(session.stats(stats), sqdv::Status::ok, "released unacknowledged stats") ||
      stats.outstanding_bytes != 0 || stats.unacknowledged_batches != 1 ||
      stats.next_processed_sequence != 11 ||
      !expect(session.offer_live(third), sqdv::Status::blocked,
              "unacknowledged ledger stays bounded after release") ||
      !expect(session.acknowledge_processed(delivery), sqdv::Status::ok,
              "acknowledge released payload ticket") ||
      !expect(session.offer_live(third), sqdv::Status::ok,
              "offer after ledger release")) return false;

  const auto maximum = std::numeric_limits<std::uint64_t>::max();
  auto terminal_config = config(maximum);
  sqdv::Session terminal;
  if (!expect(sqdv::Session::create(flow, manifest, terminal_config, {8, 1}, nullptr,
                                   nullptr, terminal), sqdv::Status::ok,
              "terminal session")) return false;
  auto last = batch(flow, manifest, maximum, 99);
  sqdv::Delivery terminal_delivery;
  if (!expect(terminal.offer_live(last), sqdv::Status::ok, "offer terminal sequence") ||
      !expect(terminal.take(terminal_delivery), sqdv::Status::ok, "take terminal sequence") ||
      !expect(terminal.acknowledge_processed(terminal_delivery), sqdv::Status::ok,
              "ack terminal sequence")) return false;
  terminal_delivery.release_payload();
  sqdv::Checkpoint checkpoint;
  if (!expect(terminal.stats(stats), sqdv::Status::ok, "terminal stats") ||
      !stats.offer_exhausted || !stats.processed_exhausted ||
      stats.next_offer_sequence != maximum || stats.next_processed_sequence != maximum ||
      !expect(terminal.checkpoint(checkpoint), sqdv::Status::ok, "terminal checkpoint") ||
      !checkpoint.sequence_exhausted || checkpoint.next_sequence != maximum ||
      terminal.offer_live(last) == sqdv::Status::ok) return false;
  sqdv::Session resumed;
  return expect(sqdv::Session::create(flow, manifest, terminal_config, {8, 1}, nullptr,
                                     &checkpoint, resumed), sqdv::Status::ok,
                "resume exhausted checkpoint") &&
         expect(resumed.stats(stats), sqdv::Status::ok, "resumed terminal stats") &&
         stats.offer_exhausted && stats.processed_exhausted &&
         resumed.offer_live(last) != sqdv::Status::ok;
}
bool installed_checkpoint_recovery() {
  TemporaryRoot root;auto manifest=metadata();auto flow=context();
  sqdv::Config cfg{"durable-view","reader","interface","part",{},1,sqdv::Profile::disposable};cfg.producer_generation[0]=1;
  sqdv::Session session;if(!expect(sqdv::Session::create(flow,manifest,cfg,{1024,2},nullptr,nullptr,session),sqdv::Status::ok,"checkpoint session"))return false;
  sqdv::Checkpoint initial;if(!expect(session.checkpoint(initial),sqdv::Status::ok,"baseline"))return false;
  sqdv::CheckpointOptions options{{},4,1U<<20};options.generation[0]=5;sqdv::CheckpointJournal journal;
  if(!expect(sqdv::CheckpointJournal::create(root.path.string(),session,options,journal),sqdv::Status::ok,"create journal"))return false;
  sqdv::Checkpoint result;
  if(!expect(journal.save_retained(session,result),sqdv::Status::unsupported,"disposable has no replay proof"))return false;
  journal.reset();if(!expect(sqdv::CheckpointJournal::open(root.path.string(),initial,options,journal),sqdv::Status::ok,"reopen journal"))return false;
  if(!expect(journal.load(result),sqdv::Status::ok,"load checkpoint")||result.next_sequence!=1)return false;
  cfg.recipient_id="other";sqdv::Session foreign;
  if(!expect(sqdv::Session::create(flow,manifest,cfg,{1024,2},nullptr,nullptr,foreign),sqdv::Status::ok,"other session"))return false;
  return expect(journal.save(foreign,result),sqdv::Status::binding_mismatch,"foreign checkpoint refused")&&result.next_sequence==1;
}
} // namespace

int main() {
  try {
    if (!installed_delivery_identity_rejection() ||
        !installed_delivery_cutover_and_resume() ||
        !installed_delivery_ack_credit_and_terminal() ||
        !installed_checkpoint_recovery()) return 1;
    std::cout << "installed SQDV C++26 consumer: identity-bound resume, retained/live cutover, distinct acknowledgement and credit, terminal sequence\n";
  } catch (const std::exception& error) {
    std::cerr << error.what() << '\n';
    return 1;
  }
}
