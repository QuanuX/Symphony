#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <signal.h>
#include <symphony/sqdv/checkpoint.hpp>
#include <sys/wait.h>
#include <unistd.h>
#include <vector>

#include <atomic>
#include <new>
namespace {
std::atomic<long> fail_at{-1}, allocations{0};
}
void *operator new(std::size_t n) {
  if (fail_at >= 0 && allocations.fetch_add(1) == fail_at)
    throw std::bad_alloc();
  if (auto p = std::malloc(n ? n : 1))
    return p;
  throw std::bad_alloc();
}
void *operator new[](std::size_t n) { return ::operator new(n); }
void operator delete(void *p) noexcept { std::free(p); }
void operator delete[](void *p) noexcept { std::free(p); }

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

sqdv::Config config(std::uint64_t first = 7) {
  sqdv::Config c{"checkpoint-view",
                 "recipient",
                 "interface",
                 "partition:a",
                 {},
                 first,
                 sqdv::Profile::disposable};
  c.producer_generation[0] = 1;
  return c;
}
sqdv::CheckpointOptions journal_options(std::uint32_t count = 8) {
  sqdv::CheckpointOptions o{{}, count, 1U << 20};
  o.generation[0] = 9;
  return o;
}
sqdv::Session session(sqfv::Context &flow, const sqmv::Manifest &m,
                      const sqdv::Config &c,
                      const sqdv::Checkpoint *resume = nullptr) {
  sqdv::Session result;
  require(sqdv::Session::create(flow, m, c, {65536, 4}, nullptr, resume,
                                result) == sqdv::Status::ok,
          "session");
  return result;
}
void process(sqdv::Session &s, sqfv::Context &flow, const sqmv::Manifest &m,
             std::uint64_t seq) {
  auto b = batch(flow, m, store_options(), seq);
  require(s.offer_live(b) == sqdv::Status::ok, "offer");
  sqdv::Delivery delivery;
  require(s.take(delivery) == sqdv::Status::ok, "take");
  require(s.acknowledge_processed(delivery) == sqdv::Status::ok, "acknowledge");
}
void durable_restart_and_bounds() {
  Root root;
  auto m = manifest();
  auto flow = context();
  auto s = session(flow, m, config());
  sqdv::Checkpoint initial;
  require(s.checkpoint(initial) == sqdv::Status::ok, "initial");
  sqdv::CheckpointJournal journal;
  auto options = journal_options(3);
  require(sqdv::CheckpointJournal::create(root.path, s, options, journal) ==
              sqdv::Status::ok,
          "journal create");
  sqdv::Checkpoint saved;
  require(journal.save(s, saved) == sqdv::Status::duplicate &&
              saved.next_sequence == 7,
          "baseline duplicate");
  process(s, flow, m, 7);
  require(journal.save(s, saved) == sqdv::Status::ok &&
              saved.next_sequence == 8,
          "save 8");
  process(s, flow, m, 8);
  require(journal.save(s, saved) == sqdv::Status::ok &&
              saved.next_sequence == 9,
          "save 9");
  process(s, flow, m, 9);
  require(journal.save(s, saved) == sqdv::Status::limit &&
              saved.next_sequence == 9,
          "finite journal");
  auto stale = session(flow, m, config());
  require(journal.save(stale, saved) == sqdv::Status::stale,
          "rollback refused");
  auto wrong = config();
  wrong.recipient_id = "other";
  auto other = session(flow, m, wrong);
  require(journal.save(other, saved) == sqdv::Status::binding_mismatch,
          "wrong view refused");
  sqdv::CheckpointJournal contender;
  require(sqdv::CheckpointJournal::open(root.path, initial, options,
                                        contender) == sqdv::Status::busy,
          "exclusive writer");
  journal.reset();
  require(sqdv::CheckpointJournal::open(root.path, initial, options, journal) ==
              sqdv::Status::ok,
          "reopen");
  require(journal.load(saved) == sqdv::Status::ok && saved.next_sequence == 9,
          "durable checkpoint");
  auto resumed = session(flow, m, config(), &saved);
  process(resumed, flow, m, 9);
  auto changed = initial;
  changed.next_sequence++;
  require(sqdv::CheckpointJournal::open(root.path, changed, options,
                                        contender) == sqdv::Status::busy,
          "busy precedes binding");
  journal.reset();
  require(
      sqdv::CheckpointJournal::open(root.path, changed, options, contender) ==
          sqdv::Status::binding_mismatch,
      "baseline binding");
}
void sequence_exhaustion() {
  Root root;
  auto m = manifest();
  auto flow = context();
  auto s = session(flow, m, config(UINT64_MAX));
  sqdv::Checkpoint initial, saved;
  require(s.checkpoint(initial) == sqdv::Status::ok, "max initial");
  sqdv::CheckpointJournal journal;
  auto options = journal_options();
  require(sqdv::CheckpointJournal::create(root.path, s, options, journal) ==
              sqdv::Status::ok,
          "max create");
  process(s, flow, m, UINT64_MAX);
  require(journal.save(s, saved) == sqdv::Status::ok &&
              saved.sequence_exhausted,
          "max saved");
  journal.reset();
  require(sqdv::CheckpointJournal::open(root.path, initial, options, journal) ==
              sqdv::Status::ok,
          "max reopen");
  require(journal.load(saved) == sqdv::Status::ok && saved.sequence_exhausted &&
              saved.next_sequence == UINT64_MAX,
          "max no wrap");
}
void process_crash_recovery() {
  for (bool persisted : {false, true}) {
    Root root;
    sqdv::Checkpoint initial;
    {
      auto m = manifest();
      auto flow = context();
      auto s = session(flow, m, config());
      sqdv::CheckpointJournal journal;
      require(s.checkpoint(initial) == sqdv::Status::ok, "crash initial");
      require(sqdv::CheckpointJournal::create(root.path, s, journal_options(),
                                              journal) == sqdv::Status::ok,
              "crash create");
    }
    int pipefd[2];
    require(::pipe(pipefd) == 0, "pipe");
    pid_t child = ::fork();
    require(child >= 0, "fork");
    if (child == 0) {
      ::close(pipefd[0]);
      auto m = manifest();
      auto flow = context();
      auto s = session(flow, m, config());
      sqdv::CheckpointJournal journal;
      require(sqdv::CheckpointJournal::open(root.path, initial,
                                            journal_options(),
                                            journal) == sqdv::Status::ok,
              "child open");
      process(s, flow, m, 7);
      sqdv::Checkpoint saved;
      if (persisted)
        require(journal.save(s, saved) == sqdv::Status::ok, "child save");
      const char ready = 'x';
      require(::write(pipefd[1], &ready, 1) == 1, "ready");
      for (;;)
        ::pause();
    }
    ::close(pipefd[1]);
    char ready = 0;
    require(::read(pipefd[0], &ready, 1) == 1, "read barrier");
    ::close(pipefd[0]);
    require(::kill(child, SIGKILL) == 0, "kill");
    int result = 0;
    require(::waitpid(child, &result, 0) == child && WIFSIGNALED(result),
            "wait");
    sqdv::CheckpointJournal journal;
    require(sqdv::CheckpointJournal::open(root.path, initial, journal_options(),
                                          journal) == sqdv::Status::ok,
            "recover");
    sqdv::Checkpoint saved;
    require(journal.load(saved) == sqdv::Status::ok &&
                saved.next_sequence == (persisted ? 8 : 7),
            "actual durable boundary");
  }
}
void checkpoint_allocation() {
  long failures = 0;
  bool reached = false;
  for (long n = 0; n < 128; ++n) {
    Root root;
    auto m = manifest();
    auto flow = context();
    auto s = session(flow, m, config());
    sqdv::Checkpoint initial;
    require(s.checkpoint(initial) == sqdv::Status::ok, "allocation baseline");
    sqdv::CheckpointJournal j;
    require(sqdv::CheckpointJournal::create(root.path, s, journal_options(),
                                            j) == sqdv::Status::ok,
            "allocation create");
    process(s, flow, m, 7);
    sqdv::Checkpoint out{"unchanged", 777, false};
    allocations = 0;
    fail_at = n;
    auto status = j.save(s, out);
    fail_at = -1;
    if (status == sqdv::Status::ok) {
      failures = n;
      reached = true;
      break;
    }
    require((status == sqdv::Status::no_memory ||
             status == sqdv::Status::outcome_uncertain) &&
                out.view_reference == "unchanged" && out.next_sequence == 777,
            "checkpoint output preserved");
    j.reset();
    require(sqdv::CheckpointJournal::open(root.path, initial, journal_options(),
                                          j) == sqdv::Status::ok,
            "allocation reopen");
    require(j.load(out) == sqdv::Status::ok && out.next_sequence >= 7 &&
                out.next_sequence <= 8,
            "actual checkpoint");
    if (status == sqdv::Status::no_memory)
      require(out.next_sequence == 7, "no memory before mutation");
  }
  require(reached && failures > 0, "checkpoint allocation sweep");
  std::printf("checkpoint allocation points: %ld\n", failures);
}

} // namespace
int main(int argc, char** argv) {
  const bool no_fork = argc == 2 && std::string_view(argv[1]) == "--no-fork";
  if (argc != 1 && !no_fork) return 2;
  durable_restart_and_bounds();
  sequence_exhaustion();
  if (!no_fork) process_crash_recovery();
  checkpoint_allocation();
  std::puts(no_fork ? "checkpoint: restart, bounds, identity and exhaustion passed; fork cases excluded"
                    : "checkpoint: restart, bounds, identity, exhaustion and two SIGKILL boundaries passed");
}
