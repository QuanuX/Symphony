#include <symphony/sqdv/delivery.hpp>

#include <array>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <limits>
#include <new>
#include <signal.h>
#include <string>
#include <sys/resource.h>
#include <sys/wait.h>
#include <unistd.h>
#include <vector>

namespace {
std::atomic<long> failing_allocation{-1};
std::atomic<long> allocation_count{0};
} // namespace

void *operator new(std::size_t size) {
  const auto fail = failing_allocation.load();
  if (fail >= 0 && allocation_count.fetch_add(1) == fail)
    throw std::bad_alloc();
  if (auto *memory = std::malloc(size == 0 ? 1 : size))
    return memory;
  throw std::bad_alloc();
}
void *operator new[](std::size_t size) { return ::operator new(size); }
void operator delete(void *memory) noexcept { std::free(memory); }
void operator delete[](void *memory) noexcept { std::free(memory); }

namespace {
using namespace symphony;

void require(bool condition, const char *message) {
  if (!condition) {
    std::fprintf(stderr, "sqdv test: %s\n", message);
    std::abort();
  }
}

struct Root {
  std::string path;
  Root() {
    char name[] = "/private/tmp/sqdv-native-XXXXXX";
    auto *created = ::mkdtemp(name);
    require(created != nullptr, "private temporary root");
    path = created;
  }
  ~Root() { std::filesystem::remove_all(path); }
};

sqmv::Manifest manifest(std::string revision = "revision:1") {
  sqmv::Description description{
      "dataset:test",
      std::move(revision),
      "schema:1",
      "layout:1",
      "private:test",
      "producer:test",
      {{sqmv::EvidenceRole::schema, "producer:test", "evidence:schema"},
       {sqmv::EvidenceRole::layout, "producer:test", "evidence:layout"},
       {sqmv::EvidenceRole::access, "producer:test", "evidence:access"}}};
  sqmv::Manifest result;
  require(sqmv::Manifest::create(description, {65536, 4096, 128}, result) ==
              sqmv::Status::ok,
          "fixture metadata");
  return result;
}

sqpv::Options store_options() {
  sqpv::Options options;
  options.partition = "partition:a";
  options.producer_generation[0] = 1;
  options.store_generation[0] = 2;
  options.first_sequence = 7;
  options.limits = {73728, 1U << 20, 16};
  return options;
}

sqfv::Context context() {
  sqfv::Context result;
  require(sqfv::Context::create({65536, 73728, 4096, 16U << 20, 16}, result) ==
              sqfv::Status::ok,
          "fixture flow context");
  return result;
}

sqfv::Batch batch(sqfv::Context &context, const sqmv::Manifest &manifest,
                  const sqpv::Options &options, std::uint64_t sequence,
                  std::uint8_t value = 1, std::size_t size = 64) {
  sqfv::Descriptor descriptor;
  require(manifest.binding(descriptor.binding) == sqmv::Status::ok,
          "fixture flow binding");
  descriptor.partition = options.partition;
  descriptor.producer_generation = options.producer_generation;
  descriptor.batch_sequence = sequence;
  descriptor.record_count = 1;
  descriptor.source_binding = "acquisition:fixture";
  descriptor.source_position = "source-position:" + std::to_string(sequence);
  std::vector<std::uint8_t> payload(size, value);
  sqfv::Batch result;
  require(context.prepare_copy(descriptor, payload, result) == sqfv::Status::ok,
          "fixture immutable batch");
  return result;
}

void corrupt_first_byte(const std::string &path) {
  std::fstream stream(path, std::ios::in | std::ios::out | std::ios::binary);
  char value = 0;
  stream.read(&value, 1);
  value ^= 1;
  stream.seekp(0);
  stream.write(&value, 1);
  require(static_cast<bool>(stream), "corrupt fixture byte");
}

std::string sequence_path(const std::string &root, const char *prefix,
                          std::uint64_t sequence) {
  char suffix[21];
  std::snprintf(suffix, sizeof(suffix), "%020llu",
                static_cast<unsigned long long>(sequence));
  return root + "/" + prefix + suffix;
}

void status(sqdv::Status actual, sqdv::Status expected, const char *message) {
  if (actual != expected) {
    std::fprintf(stderr, "sqdv test: %s; status %d expected %d\n", message,
                 static_cast<int>(actual), static_cast<int>(expected));
    std::abort();
  }
}

sqdv::Config config(const sqpv::Options &options,
                    sqdv::Profile profile = sqdv::Profile::disposable) {
  return {"view:full-batch",
          "recipient:test",
          "recipient-interface:1",
          options.partition,
          options.producer_generation,
          options.first_sequence,
          profile};
}

sqdv::SessionStats stats(const sqdv::Session &session) {
  sqdv::SessionStats value;
  status(session.stats(value), sqdv::Status::ok, "session statistics");
  return value;
}

sqdv::Checkpoint checkpoint(const sqdv::Session &session) {
  sqdv::Checkpoint value;
  status(session.checkpoint(value), sqdv::Status::ok, "session checkpoint");
  return value;
}

void retained_live_cutover_and_exact_resume() {
  Root root;
  auto metadata = manifest();
  auto options = store_options();
  auto flow = context();
  sqdv::RetainedSource source;
  status(sqdv::RetainedSource::create(root.path, metadata, options, source),
         sqdv::Status::ok, "create retained source");
  std::array<sqdv::RetainedBatch, 3> proof;
  for (std::uint64_t sequence = 7; sequence != 10; ++sequence) {
    auto input = batch(flow, metadata, options, sequence,
                       static_cast<std::uint8_t>(sequence));
    status(source.commit(flow, input, proof[sequence - 7]), sqdv::Status::ok,
           "commit input and mint proof");
  }
  const auto same_commit = proof[2].receipt().commit_sha256;
  status(source.commit(flow, proof[2].batch(), proof[2]),
         sqdv::Status::duplicate,
         "exact committed retry publishes actual proof with output aliasing");
  require(proof[2].receipt().commit_sha256 == same_commit,
          "duplicate proof preserves original commit evidence");
  auto cfg = config(options, sqdv::Profile::retained_before_delivery);
  sqdv::Session session;
  status(sqdv::Session::create(flow, metadata, cfg, {256, 4}, &source, nullptr,
                               session),
         sqdv::Status::ok, "create retained session");

  // A newer exact-source live candidate must not bypass the older resume
  // position. The first two positions are retrieved through the store.
  for (std::uint64_t sequence = 7; sequence != 10; ++sequence) {
    status(session.offer_next(flow, &proof[2]), sqdv::Status::ok,
           "offer exact next position");
    sqdv::Delivery delivered;
    status(session.take(delivered), sqdv::Status::ok, "take exact next");
    require(delivered.sequence() == sequence &&
                delivered.payload().size() == 64 &&
                delivered.payload()[0] == sequence,
            "no omitted or reordered batch at source cutover");
    require(delivered.origin() ==
                (sequence == 9 ? sqdv::Origin::live : sqdv::Origin::retained),
            "actual source of delivered bytes is reported");
    require(delivered.content_id() == proof[sequence - 7].batch().content_id(),
            "live and retained positions preserve content identity");
    const auto *receipt = delivered.retention_receipt();
    require(receipt && receipt->commit_sha256 ==
                           proof[sequence - 7].receipt().commit_sha256,
            "delivery includes actual exact commit evidence");
    if (sequence != 9)
      status(session.acknowledge_processed(delivered), sqdv::Status::ok,
             "acknowledge contiguous processing");
    else
      delivered.release_payload(); // Deliberately lose its acknowledgement.
  }
  auto resume = checkpoint(session);
  require(resume.next_sequence == 9 && !resume.sequence_exhausted,
          "checkpoint excludes the delivered but unacknowledged batch");
  session.reset();
  source.reset();
  // Proof handles preserve immutable bytes but must not retain the store lock.
  status(sqdv::RetainedSource::open(root.path, metadata, options, source),
         sqdv::Status::ok, "reopen while old immutable proof handles live");
  status(sqdv::Session::create(flow, metadata, cfg, {256, 4}, &source, &resume,
                               session),
         sqdv::Status::ok, "resume exact checkpoint after source reopen");
  status(session.offer_next(flow), sqdv::Status::ok,
         "replay lost acknowledgement");
  sqdv::Delivery replay;
  status(session.take(replay), sqdv::Status::ok, "take redelivery");
  require(replay.sequence() == 9 && replay.origin() == sqdv::Origin::retained &&
              replay.content_id() == proof[2].batch().content_id(),
          "lost acknowledgement is visibly redelivered at the same identity");
  status(session.acknowledge_processed(replay), sqdv::Status::ok,
         "acknowledge redelivery");
  require(checkpoint(session).next_sequence == 10,
          "redelivery advances processing only after acknowledgement");
  status(session.offer_next(flow), sqdv::Status::missing,
         "future unavailable position is explicit");
  require(stats(session).next_offer_sequence == 10,
          "unavailable history does not skip position");
}

void acknowledgement_and_lease_are_independent() {
  auto metadata = manifest();
  auto options = store_options();
  auto flow = context();
  auto cfg = config(options);
  sqdv::Session session;
  status(sqdv::Session::create(flow, metadata, cfg, {64, 2}, nullptr, nullptr,
                               session),
         sqdv::Status::ok, "create credit session");
  auto first = batch(flow, metadata, options, 7);
  auto second = batch(flow, metadata, options, 8);
  status(session.offer_live(first), sqdv::Status::ok, "offer credit batch");
  sqdv::Delivery held;
  status(session.take(held), sqdv::Status::ok, "take credit batch");
  require(held.retention_receipt() == nullptr,
          "disposable delivery makes no retention assertion");
  allocation_count = 0;
  failing_allocation = 0;
  const auto acknowledgement_result = session.acknowledge_processed(held);
  failing_allocation = -1;
  status(acknowledgement_result, sqdv::Status::ok,
         "processing acknowledgement while lease is held");
  require(allocation_count == 0,
          "processing acknowledgement needs no allocation");
  auto observed = stats(session);
  require(observed.next_processed_sequence == 8 &&
              observed.outstanding_bytes == 64 &&
              observed.unacknowledged_batches == 0 && held.has_payload(),
          "acknowledgement preserves physical borrow and credit");
  status(session.offer_live(second), sqdv::Status::blocked,
         "acknowledgement cannot free held payload credit");
  allocation_count = 0;
  failing_allocation = 0;
  held.release_payload();
  failing_allocation = -1;
  require(allocation_count == 0, "payload release needs no allocation");
  require(!held.has_payload() && held.payload().empty() && held.sequence() == 7,
          "payload release preserves processing ticket identity");
  status(session.acknowledge_processed(held), sqdv::Status::duplicate,
         "repeat acknowledgement is explicit");
  status(session.offer_live(second), sqdv::Status::ok,
         "payload release permits subsequent credit admission");
  sqdv::Delivery unacknowledged;
  status(session.take(unacknowledged), sqdv::Status::ok,
         "take unacknowledged batch");
  unacknowledged.release_payload();
  observed = stats(session);
  require(observed.outstanding_bytes == 0 &&
              observed.unacknowledged_batches == 1 &&
              checkpoint(session).next_sequence == 8,
          "payload release never acknowledges processing");

  // Out-of-order ACKs are rejected, and their tickets remain usable later.
  auto third = batch(flow, metadata, options, 9);
  status(session.offer_live(third), sqdv::Status::ok,
         "offer second pending ACK");
  sqdv::Delivery later;
  status(session.take(later), sqdv::Status::ok, "take later pending ACK");
  later.release_payload();
  status(session.acknowledge_processed(later), sqdv::Status::gap,
         "out-of-order processing acknowledgement is explicit");
  require(checkpoint(session).next_sequence == 8,
          "out-of-order ACK cannot advance checkpoint");
  auto fourth = batch(flow, metadata, options, 10);
  status(session.offer_live(fourth), sqdv::Status::blocked,
         "released but unacknowledged tickets retain finite ledger allowance");
  status(session.acknowledge_processed(unacknowledged), sqdv::Status::ok,
         "close first processing gap");
  status(session.acknowledge_processed(later), sqdv::Status::ok,
         "previously rejected later ACK succeeds in sequence");
  status(session.offer_live(fourth), sqdv::Status::ok,
         "contiguous acknowledgements return ledger allowance");
  sqdv::Delivery surviving;
  status(session.take(surviving), sqdv::Status::ok, "take surviving lease");
  session.reset();
  flow = sqfv::Context{};
  first = sqfv::Batch{};
  second = sqfv::Batch{};
  third = sqfv::Batch{};
  fourth = sqfv::Batch{};
  require(surviving.payload().size() == 64 && surviving.payload()[0] == 1,
          "taken delivery outlives session, context and producer handles");
}

void independent_recipient_allowances() {
  auto metadata = manifest();
  auto options = store_options();
  auto flow = context();
  auto slow_cfg = config(options);
  auto fast_cfg = slow_cfg;
  fast_cfg.recipient_id = "recipient:fast";
  sqdv::Session slow, fast;
  status(sqdv::Session::create(flow, metadata, slow_cfg, {64, 1}, nullptr,
                               nullptr, slow),
         sqdv::Status::ok, "slow recipient");
  status(sqdv::Session::create(flow, metadata, fast_cfg, {64, 1}, nullptr,
                               nullptr, fast),
         sqdv::Status::ok, "fast recipient");
  auto first = batch(flow, metadata, options, 7);
  status(slow.offer_live(first), sqdv::Status::ok, "slow first admission");
  sqdv::Delivery slow_held;
  status(slow.take(slow_held), sqdv::Status::ok, "slow held lease");
  for (std::uint64_t sequence = 7; sequence != 12; ++sequence) {
    auto input = batch(flow, metadata, options, sequence);
    status(fast.offer_live(input), sqdv::Status::ok,
           "independent fast recipient continues");
    sqdv::Delivery delivery;
    status(fast.take(delivery), sqdv::Status::ok, "fast take");
    status(fast.acknowledge_processed(delivery), sqdv::Status::ok, "fast ACK");
  }
  auto next = batch(flow, metadata, options, 8);
  status(slow.offer_live(next), sqdv::Status::blocked,
         "slow branch reaches its own bound");
  require(checkpoint(fast).next_sequence == 12 &&
              checkpoint(slow).next_sequence == 7,
          "recipient processing positions remain independent");
  slow_held.release_payload();
  status(slow.offer_live(next), sqdv::Status::blocked,
         "released slow lease still awaits required processing ACK");
  status(slow.acknowledge_processed(slow_held), sqdv::Status::ok, "slow ACK");
  status(slow.offer_live(next), sqdv::Status::ok, "slow exact retry after ACK");

  // Compose all four owners: a bounded slow recipient cannot stop a second
  // recipient from consuming later proven commits. It catches up by exact
  // retained position after its original borrow and acknowledgement finish.
  Root root;
  sqdv::RetainedSource source;
  status(sqdv::RetainedSource::create(root.path, metadata, options, source),
         sqdv::Status::ok, "fan-out retained source");
  slow.reset();
  fast.reset();
  slow_cfg.profile = sqdv::Profile::retained_before_delivery;
  fast_cfg.profile = sqdv::Profile::retained_before_delivery;
  status(sqdv::Session::create(flow, metadata, slow_cfg, {64, 1}, &source,
                               nullptr, slow),
         sqdv::Status::ok, "retained slow recipient");
  status(sqdv::Session::create(flow, metadata, fast_cfg, {64, 1}, &source,
                               nullptr, fast),
         sqdv::Status::ok, "retained fast recipient");
  for (std::uint64_t sequence = 7; sequence != 12; ++sequence) {
    auto input = batch(flow, metadata, options, sequence,
                       static_cast<std::uint8_t>(sequence));
    sqdv::RetainedBatch proof;
    status(source.commit(flow, input, proof), sqdv::Status::ok,
           "commit retained fan-out input");
    if (sequence == 7) {
      status(slow.offer_next(flow, &proof), sqdv::Status::ok,
             "slow receives first actual committed live batch");
      status(slow.take(slow_held), sqdv::Status::ok,
             "slow holds committed lease");
    } else {
      status(slow.offer_next(flow, &proof), sqdv::Status::blocked,
             "slow retained recipient remains bounded");
    }
    status(fast.offer_next(flow, &proof), sqdv::Status::ok,
           "fast retained recipient continues beyond slow boundary");
    sqdv::Delivery delivery;
    status(fast.take(delivery), sqdv::Status::ok,
           "fast committed live delivery");
    require(delivery.sequence() == sequence &&
                delivery.origin() == sqdv::Origin::live &&
                delivery.retention_receipt() != nullptr,
            "fast delivery carries actual commit proof");
    status(fast.acknowledge_processed(delivery), sqdv::Status::ok,
           "fast acknowledges committed delivery");
  }
  require(checkpoint(fast).next_sequence == 12 &&
              checkpoint(slow).next_sequence == 7,
          "retained recipient checkpoints remain independent");
  slow_held.release_payload();
  status(slow.offer_next(flow), sqdv::Status::blocked,
         "retained source cannot bypass exhausted ACK allowance");
  status(slow.acknowledge_processed(slow_held), sqdv::Status::ok,
         "slow acknowledges its original committed delivery");
  for (std::uint64_t sequence = 8; sequence != 12; ++sequence) {
    status(slow.offer_next(flow), sqdv::Status::ok,
           "slow exact retained catch-up");
    sqdv::Delivery delivery;
    status(slow.take(delivery), sqdv::Status::ok,
           "slow takes retained catch-up");
    require(delivery.sequence() == sequence &&
                delivery.origin() == sqdv::Origin::retained &&
                delivery.payload()[0] == sequence,
            "slow catch-up preserves exact retained order");
    status(slow.acknowledge_processed(delivery), sqdv::Status::ok,
           "slow acknowledges exact retained catch-up");
  }
  require(checkpoint(slow).next_sequence == 12,
          "retained catch-up reaches the same known processed boundary");
}

void exact_identity_and_foreign_tickets() {
  Root root, foreign_root;
  auto metadata = manifest();
  auto options = store_options();
  auto flow = context();
  auto cfg = config(options, sqdv::Profile::retained_before_delivery);
  sqdv::RetainedSource source, foreign_source;
  status(sqdv::RetainedSource::create(root.path, metadata, options, source),
         sqdv::Status::ok, "identity source");
  status(sqdv::RetainedSource::create(foreign_root.path, metadata, options,
                                      foreign_source),
         sqdv::Status::ok,
         "separately owned source with same advertised options");
  auto input = batch(flow, metadata, options, 7);
  sqdv::RetainedBatch proof, foreign_proof;
  status(source.commit(flow, input, proof), sqdv::Status::ok, "source proof");
  status(foreign_source.commit(flow, input, foreign_proof), sqdv::Status::ok,
         "foreign actual source proof");
  sqdv::Session session;
  status(sqdv::Session::create(flow, metadata, cfg, {256, 4}, &source, nullptr,
                               session),
         sqdv::Status::ok, "identity session");
  const auto resume = checkpoint(session);
  status(sqdv::Session::create(flow, metadata, cfg, {256, 4}, &foreign_source,
                               &resume, session),
         sqdv::Status::binding_mismatch,
         "checkpoint cannot silently move to another retained store root");
  status(session.offer_next(flow, &foreign_proof),
         sqdv::Status::binding_mismatch,
         "matching public fields cannot substitute foreign proof identity");
  require(stats(session).next_offer_sequence == 7,
          "foreign proof leaves cursor unchanged");
  const auto old_reference = std::string(session.view_reference());
  auto reject_config = [&](const sqdv::Config &changed) {
    status(sqdv::Session::create(flow, metadata, changed, {256, 4}, &source,
                                 &resume, session),
           sqdv::Status::binding_mismatch, "resume exact contract mismatch");
    require(session.view_reference() == old_reference,
            "failed session replacement preserves previous session");
  };
  auto changed = cfg;
  changed.view_id += "other";
  reject_config(changed);
  changed = cfg;
  changed.recipient_id += "other";
  reject_config(changed);
  changed = cfg;
  changed.recipient_interface += "other";
  reject_config(changed);
  changed = cfg;
  changed.partition += "other";
  reject_config(changed);
  changed = cfg;
  changed.producer_generation[1] = 1;
  reject_config(changed);
  changed = cfg;
  changed.first_sequence++;
  reject_config(changed);

  auto wrong_metadata = manifest("revision:other");
  status(sqdv::Session::create(flow, wrong_metadata, cfg, {256, 4}, &source,
                               &resume, session),
         sqdv::Status::binding_mismatch, "exact metadata mismatch");
  auto disposable_cfg = cfg;
  disposable_cfg.profile = sqdv::Profile::disposable;
  status(sqdv::Session::create(flow, metadata, disposable_cfg, {256, 4},
                               nullptr, &resume, session),
         sqdv::Status::binding_mismatch,
         "profile cannot reuse retained checkpoint");

  status(session.offer_next(flow, &proof), sqdv::Status::ok,
         "own source proof");
  sqdv::Delivery old_ticket;
  status(session.take(old_ticket), sqdv::Status::ok, "ticket before reconnect");
  auto lost_ack = checkpoint(session);
  session.reset();
  status(sqdv::Session::create(flow, metadata, cfg, {256, 4}, &source,
                               &lost_ack, session),
         sqdv::Status::ok, "same recipient reconnects");
  status(session.offer_next(flow, &proof), sqdv::Status::ok,
         "redelivery after reconnect");
  sqdv::Delivery current_ticket;
  status(session.take(current_ticket), sqdv::Status::ok,
         "current delivery ticket");
  status(session.acknowledge_processed(old_ticket),
         sqdv::Status::binding_mismatch,
         "late old-session ACK cannot acknowledge new redelivery");
  require(checkpoint(session).next_sequence == 7,
          "foreign attempt leaves processed cursor unchanged");
  status(session.acknowledge_processed(current_ticket), sqdv::Status::ok,
         "current attempt ACK succeeds");

  // Inherited operational handles reject before touching mutex state. The
  // child exits directly because inherited C++ handle cleanup is unsupported.
  const auto child = ::fork();
  require(child >= 0, "fork inherited-handle rejection fixture");
  if (child == 0) {
    sqdv::Checkpoint child_checkpoint;
    sqdv::SessionStats child_stats;
    sqdv::Delivery child_delivery;
    sqdv::RetainedBatch child_proof;
    sqdv::RetainedSource child_source;
    const bool rejected =
        session.offer_next(flow) == sqdv::Status::stale &&
        session.take(child_delivery) == sqdv::Status::stale &&
        session.acknowledge_processed(current_ticket) == sqdv::Status::stale &&
        session.checkpoint(child_checkpoint) == sqdv::Status::stale &&
        session.stats(child_stats) == sqdv::Status::stale &&
        source.commit(flow, input, child_proof) == sqdv::Status::stale &&
        source.retain(child_source) == sqdv::Status::stale &&
        proof.retain(child_proof) == sqdv::Status::stale;
    ::_exit(rejected ? 0 : 91);
  }
  int child_status = 0;
  require(::waitpid(child, &child_status, 0) == child &&
              WIFEXITED(child_status) && WEXITSTATUS(child_status) == 0,
          "inherited operational handles reject use after fork");
}

void output_preservation_and_sequence_exhaustion() {
  auto metadata = manifest();
  auto options = store_options();
  auto flow = context();
  auto cfg = config(options);
  sqdv::Session session;
  status(sqdv::Session::create(flow, metadata, cfg, {128, 2}, nullptr, nullptr,
                               session),
         sqdv::Status::ok, "output preservation session");
  auto input = batch(flow, metadata, options, 7);
  status(session.offer_live(input), sqdv::Status::ok, "offer output sentinel");
  sqdv::Delivery output;
  status(session.take(output), sqdv::Status::ok, "take output sentinel");
  const auto id = output.content_id();
  status(session.take(output), sqdv::Status::empty,
         "empty take preserves output");
  require(output.sequence() == 7 && output.content_id() == id &&
              output.has_payload(),
          "failed take preserves owning output");
  auto higher = batch(flow, metadata, options, 9);
  status(session.offer_live(higher), sqdv::Status::gap,
         "higher offer is explicit gap");
  auto changed = batch(flow, metadata, options, 7, 2);
  status(session.offer_live(changed), sqdv::Status::conflict,
         "same sequence changed content conflicts");
  status(session.offer_live(input), sqdv::Status::duplicate,
         "same last accepted identity is duplicate");
  require(stats(session).next_offer_sequence == 8,
          "rejected offers preserve next sequence");
  auto other_flow = context();
  auto foreign_context = batch(other_flow, metadata, options, 8);
  status(session.offer_live(foreign_context), sqdv::Status::invalid_argument,
         "foreign Context batch rejected before cursor mutation");
  require(stats(session).next_offer_sequence == 8,
          "Context mismatch preserves cursor");

  Root root;
  options.first_sequence = std::numeric_limits<std::uint64_t>::max();
  options.limits.max_batches = 1;
  sqdv::RetainedSource source;
  status(sqdv::RetainedSource::create(root.path, metadata, options, source),
         sqdv::Status::ok, "terminal source");
  auto terminal = batch(flow, metadata, options, options.first_sequence);
  sqdv::RetainedBatch proof;
  status(source.commit(flow, terminal, proof), sqdv::Status::ok,
         "terminal retained position");
  cfg = config(options, sqdv::Profile::retained_before_delivery);
  session.reset();
  status(sqdv::Session::create(flow, metadata, cfg, {128, 2}, &source, nullptr,
                               session),
         sqdv::Status::ok, "terminal session");
  status(session.offer_next(flow, &proof), sqdv::Status::ok, "terminal offer");
  status(session.take(output), sqdv::Status::ok, "terminal take");
  require(stats(session).offer_exhausted &&
              !stats(session).processed_exhausted &&
              output.sequence() == options.first_sequence,
          "offer exhaustion precedes processing exhaustion without wrapping");
  status(session.acknowledge_processed(output), sqdv::Status::ok,
         "terminal ACK");
  auto resume = checkpoint(session);
  require(resume.sequence_exhausted &&
              resume.next_sequence == options.first_sequence,
          "terminal checkpoint is explicit and does not wrap");
  status(session.offer_next(flow), sqdv::Status::limit,
         "terminal stream rejects further offers");
  session.reset();
  status(sqdv::Session::create(flow, metadata, cfg, {128, 2}, &source, &resume,
                               session),
         sqdv::Status::ok, "resume exhausted stream");
  require(stats(session).offer_exhausted && stats(session).processed_exhausted,
          "terminal flags survive resume");
}

void allocation_failure_preserves_delivery() {
  auto metadata = manifest();
  auto options = store_options();
  auto flow = context();
  auto cfg = config(options);
  bool created = false;
  std::size_t create_failures = 0;
  for (long failure = 0; failure != 256; ++failure) {
    sqdv::Session output;
    allocation_count = 0;
    failing_allocation = failure;
    const auto result = sqdv::Session::create(flow, metadata, cfg, {128, 2},
                                              nullptr, nullptr, output);
    failing_allocation = -1;
    if (result == sqdv::Status::ok) {
      created = true;
      break;
    }
    status(result, sqdv::Status::no_memory,
           "session creation allocation failure");
    require(!output, "failed creation publishes no partial session");
    ++create_failures;
  }
  require(created && create_failures > 0,
          "creation allocation sweep reaches success");
  auto input = batch(flow, metadata, options, 7);
  bool offered = false;
  std::size_t offer_failures = 0;
  for (long failure = 0; failure != 256; ++failure) {
    sqdv::Session session;
    status(sqdv::Session::create(flow, metadata, cfg, {128, 2}, nullptr,
                                 nullptr, session),
           sqdv::Status::ok, "allocation offer session");
    allocation_count = 0;
    failing_allocation = failure;
    const auto result = session.offer_live(input);
    failing_allocation = -1;
    if (result == sqdv::Status::ok) {
      offered = true;
      sqdv::Delivery output;
      allocation_count = 0;
      failing_allocation = 0;
      const auto take_result = session.take(output);
      failing_allocation = -1;
      status(take_result, sqdv::Status::ok, "take requires no new allocation");
      require(allocation_count == 0 && output.sequence() == 7,
              "preallocated delivery is transferred intact");
      sqdv::Checkpoint preserved{"unchanged checkpoint output", 1234, true};
      allocation_count = 0;
      failing_allocation = 0;
      const auto checkpoint_result = session.checkpoint(preserved);
      failing_allocation = -1;
      status(checkpoint_result, sqdv::Status::no_memory,
             "checkpoint allocation failure is explicit");
      require(preserved.view_reference == "unchanged checkpoint output" &&
                  preserved.next_sequence == 1234 &&
                  preserved.sequence_exhausted,
              "failed checkpoint preserves complete output");
      break;
    }
    status(result, sqdv::Status::no_memory, "offer allocation failure");
    auto observed = stats(session);
    require(observed.next_offer_sequence == 7 &&
                observed.next_processed_sequence == 7 &&
                observed.pending_deliveries == 0 &&
                observed.unacknowledged_batches == 0 &&
                observed.outstanding_bytes == 0,
            "failed offer rolls back dispatch, ACK ledger and byte credit");
    ++offer_failures;
  }
  require(offered && offer_failures > 0,
          "offer allocation sweep reaches success");
  std::printf("sqdv allocation rollback: %zu create and %zu offer failures\n",
              create_failures, offer_failures);

  Root root;
  sqdv::RetainedSource source;
  status(sqdv::RetainedSource::create(root.path, metadata, options, source),
         sqdv::Status::ok, "retained allocation source");
  sqdv::RetainedBatch proof;
  status(source.commit(flow, input, proof), sqdv::Status::ok,
         "actual proof for retained allocation sweep");
  cfg.profile = sqdv::Profile::retained_before_delivery;
  for (bool from_live_proof : {true, false}) {
    bool succeeded = false;
    std::size_t failures = 0;
    for (long failure = 0; failure != 256; ++failure) {
      sqdv::Session session;
      status(sqdv::Session::create(flow, metadata, cfg, {128, 2}, &source,
                                   nullptr, session),
             sqdv::Status::ok, "retained allocation session");
      sqfv::ContextStats before;
      require(flow.stats(before) == sqfv::Status::ok,
              "retained allocation baseline");
      allocation_count = 0;
      failing_allocation = failure;
      const auto result =
          session.offer_next(flow, from_live_proof ? &proof : nullptr);
      failing_allocation = -1;
      if (result == sqdv::Status::ok) {
        succeeded = true;
        sqdv::Delivery output;
        allocation_count = 0;
        failing_allocation = 0;
        const auto take_result = session.take(output);
        failing_allocation = -1;
        status(take_result, sqdv::Status::ok,
               "retained delivery take requires no allocation");
        require(allocation_count == 0 && output.sequence() == 7 &&
                    output.retention_receipt() &&
                    output.retention_receipt()->commit_sha256 ==
                        proof.receipt().commit_sha256,
                "retained offer publishes complete actual receipt");
        break;
      }
      status(result, sqdv::Status::no_memory,
             "retained offer allocation failure");
      const auto observed = stats(session);
      sqfv::ContextStats after;
      require(flow.stats(after) == sqfv::Status::ok,
              "retained allocation cleanup");
      require(
          observed.next_offer_sequence == 7 &&
              observed.next_processed_sequence == 7 &&
              observed.pending_deliveries == 0 &&
              observed.unacknowledged_batches == 0 &&
              observed.outstanding_bytes == 0 &&
              after.allocation_bytes == before.allocation_bytes,
          "failed retained offer releases every decoded batch and reservation");
      ++failures;
    }
    require(succeeded && failures > 0,
            "retained offer allocation sweep reaches success");
    std::printf("sqdv allocation rollback: %zu retained %s offer failures\n",
                failures, from_live_proof ? "live" : "read");
  }
}

class FileSizeLimit final {
public:
  FileSizeLimit() {
    require(::getrlimit(RLIMIT_FSIZE, &previous_limit_) == 0,
            "read file-size resource limit");
    struct sigaction ignored{};
    ignored.sa_handler = SIG_IGN;
    sigemptyset(&ignored.sa_mask);
    require(::sigaction(SIGXFSZ, &ignored, &previous_action_) == 0,
            "temporarily ignore file-size signal");
    auto limited = previous_limit_;
    limited.rlim_cur = 1;
    require(::setrlimit(RLIMIT_FSIZE, &limited) == 0,
            "limit private test write after admission");
  }
  ~FileSizeLimit() {
    require(::setrlimit(RLIMIT_FSIZE, &previous_limit_) == 0,
            "restore file-size resource limit");
    require(::sigaction(SIGXFSZ, &previous_action_, nullptr) == 0,
            "restore file-size signal action");
  }

private:
  struct rlimit previous_limit_{};
  struct sigaction previous_action_{};
};

void retention_failure_never_claims_delivery() {
  Root root;
  auto metadata = manifest();
  auto options = store_options();
  auto flow = context();
  auto cfg = config(options, sqdv::Profile::retained_before_delivery);
  sqdv::RetainedSource source;
  status(sqdv::RetainedSource::create(root.path, metadata, options, source),
         sqdv::Status::ok, "uncertainty source");
  sqdv::Session session;
  status(sqdv::Session::create(flow, metadata, cfg, {128, 2}, &source, nullptr,
                               session),
         sqdv::Status::ok, "uncertainty session");
  auto input = batch(flow, metadata, options, 7);
  sqdv::RetainedBatch output;
  status(source.commit(flow, input, output), sqdv::Status::ok,
         "commit known prefix before uncertainty");
  status(session.offer_next(flow, &output), sqdv::Status::ok,
         "offer known prefix before uncertainty");
  sqdv::Delivery delivered;
  status(session.take(delivered), sqdv::Status::ok,
         "take known prefix before uncertainty");
  const auto prior_commit = output.receipt().commit_sha256;
  input = batch(flow, metadata, options, 8);
  {
    FileSizeLimit file_size_limit;
    status(source.commit(flow, input, output), sqdv::Status::outcome_uncertain,
           "real partial staged write reports uncertain outcome");
  }
  require(
      output && output.receipt().batch_sequence == 7 &&
          output.receipt().commit_sha256 == prior_commit,
      "uncertain retention preserves existing output and mints no new proof");
  status(session.offer_next(flow), sqdv::Status::closed,
         "uncertain source blocks committed delivery");
  status(session.take(delivered), sqdv::Status::closed,
         "uncertain source closes further takes");
  require(delivered.sequence() == 7 && delivered.has_payload(),
          "uncertain retention preserves the previously issued delivery");
  status(session.acknowledge_processed(delivered), sqdv::Status::ok,
         "known issued processing remains acknowledgeable after source "
         "uncertainty");
  auto resume = checkpoint(session);
  require(resume.next_sequence == 8 &&
              stats(session).next_offer_sequence == 8 &&
              stats(session).outstanding_bytes == 64,
          "uncertainty preserves known processing and held payload accounting");
  session.reset();
  source.reset();
  status(sqdv::RetainedSource::open(root.path, metadata, options, source),
         sqdv::Status::ok, "reconcile partial write on source reopen");
  status(source.commit(flow, input, output), sqdv::Status::ok,
         "exact retry finishes staged retention");
  auto later_input = batch(flow, metadata, options, 9);
  sqdv::RetainedBatch later;
  status(source.commit(flow, later_input, later), sqdv::Status::ok,
         "newer committed live candidate");
  status(sqdv::Session::create(flow, metadata, cfg, {128, 2}, &source, &resume,
                               session),
         sqdv::Status::ok, "post-recovery session");
  corrupt_first_byte(sequence_path(root.path, "frame-", 8));
  status(session.offer_next(flow, &later), sqdv::Status::corrupt,
         "corrupt retained bytes cannot become committed delivery");
  require(stats(session).next_offer_sequence == 8 &&
              checkpoint(session).next_sequence == 8,
          "retained corruption cannot skip to a newer live candidate");
  status(session.take(delivered), sqdv::Status::empty,
         "failed retained read leaves no queued delivery");
  require(delivered.sequence() == 7 && delivered.has_payload(),
          "read failure and empty take preserve the complete old output");
}

} // namespace

int main() {
  retained_live_cutover_and_exact_resume();
  acknowledgement_and_lease_are_independent();
  independent_recipient_allowances();
  exact_identity_and_foreign_tickets();
  output_preservation_and_sequence_exhaustion();
  allocation_failure_preserves_delivery();
  retention_failure_never_claims_delivery();
  std::puts("sqdv native acceptance: 7 groups passed");
}
