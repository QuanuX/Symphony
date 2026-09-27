#include "hook.hpp"
#include <algorithm>
#include <array>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <new>
#include <signal.h>
#include <string>
#include <symphony/sqpv/local_store.hpp>
#include <sys/mount.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>
#include <vector>

namespace {
std::atomic<int> allocation_failure{-1};
std::atomic<int> allocation_calls{0};
} // namespace
void *operator new(std::size_t size) {
  if (allocation_failure.load() >= 0 &&
      allocation_calls.fetch_add(1) == allocation_failure.load())
    throw std::bad_alloc();
  if (auto *memory = std::malloc(size == 0 ? 1 : size))
    return memory;
  throw std::bad_alloc();
}
void *operator new[](std::size_t size) { return ::operator new(size); }
void operator delete(void *pointer) noexcept { std::free(pointer); }
void operator delete[](void *pointer) noexcept { std::free(pointer); }

namespace {
using namespace symphony;
void require(bool yes, const char *message) {
  if (!yes) {
    std::fprintf(stderr, "sqpv test: %s\n", message);
    std::abort();
  }
}
void status(sqpv::Status actual, sqpv::Status expected, const char *message) {
  if (actual != expected) {
    std::fprintf(stderr, "sqpv test: %s; status %d expected %d\n", message,
                 static_cast<int>(actual), static_cast<int>(expected));
    std::abort();
  }
}
struct Root {
  std::string path;
  Root() {
    char name[] = "/private/tmp/sqpv-native-XXXXXX";
    auto *p = ::mkdtemp(name);
    require(p != nullptr, "mkdtemp");
    path = p;
  }
  ~Root() {
    std::filesystem::remove_all(path);
    std::filesystem::remove_all(path + "-moved");
  }
};
sqmv::Manifest manifest() {
  sqmv::Description d{
      "dataset:test",
      "revision:1",
      "schema:1",
      "layout:1",
      "private:test",
      "producer:test",
      {{sqmv::EvidenceRole::schema, "producer:test", "evidence:schema"},
       {sqmv::EvidenceRole::layout, "producer:test", "evidence:layout"},
       {sqmv::EvidenceRole::access, "producer:test", "evidence:access"}}};
  sqmv::Manifest m;
  require(sqmv::Manifest::create(d, {65536, 4096, 128}, m) == sqmv::Status::ok,
          "metadata fixture");
  return m;
}
sqpv::Options options() {
  sqpv::Options o;
  o.partition = "partition:a";
  o.producer_generation[0] = 1;
  o.store_generation[0] = 2;
  o.first_sequence = 7;
  o.limits = {73728, 1048576, 8};
  return o;
}
sqfv::Context context() {
  sqfv::Context c;
  require(sqfv::Context::create({65536, 73728, 4096, 4U << 20, 2}, c) ==
              sqfv::Status::ok,
          "flow context");
  return c;
}
sqfv::Batch batch(sqfv::Context &c, const sqmv::Manifest &m,
                  const sqpv::Options &o, std::uint64_t seq,
                  std::uint8_t value = 1, std::size_t size = 1024) {
  sqfv::Descriptor d;
  require(m.binding(d.binding) == sqmv::Status::ok, "binding");
  d.partition = o.partition;
  d.producer_generation = o.producer_generation;
  d.batch_sequence = seq;
  d.record_count = 1;
  std::vector<std::uint8_t> bytes(size, value);
  sqfv::Batch b;
  require(c.prepare_copy(d, bytes, b) == sqfv::Status::ok, "batch");
  return b;
}
std::string name(const char *kind, std::uint64_t sequence) {
  char value[21];
  std::snprintf(value, sizeof(value), "%020llu",
                static_cast<unsigned long long>(sequence));
  return std::string(kind) + value;
}
void corrupt(const std::string &path) {
  std::fstream f(path, std::ios::in | std::ios::out | std::ios::binary);
  char c;
  f.read(&c, 1);
  c ^= 1;
  f.seekp(0);
  f.write(&c, 1);
  require(static_cast<bool>(f), "corrupt fixture");
}
void commit_retry_and_exact_stream() {
  Root root;
  auto m = manifest();
  auto o = options();
  auto c = context();
  sqpv::Store s;
  status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok, "create");
  sqpv::Receipt receipt;
  auto b = batch(c, m, o, 7);
  status(s.append(c, b, receipt), sqpv::Status::ok, "append");
  auto first = receipt;
  status(s.append(c, b, receipt), sqpv::Status::duplicate, "exact retry");
  require(first.commit_sha256 == receipt.commit_sha256, "same receipt");
  auto conflict = batch(c, m, o, 7, 2);
  status(s.append(c, conflict, receipt), sqpv::Status::conflict,
         "conflicting retry");
  auto gap = batch(c, m, o, 9);
  status(s.append(c, gap, receipt), sqpv::Status::gap, "sequence gap");
  auto stale = batch(c, m, o, 6);
  status(s.append(c, stale, receipt), sqpv::Status::stale, "old sequence");
  auto wrong = o;
  wrong.producer_generation[0] = 3;
  auto foreign = batch(c, m, wrong, 8);
  status(s.append(c, foreign, receipt), sqpv::Status::binding_mismatch,
         "generation binding");
  wrong = o;
  wrong.partition = "other";
  foreign = batch(c, m, wrong, 8);
  status(s.append(c, foreign, receipt), sqpv::Status::binding_mismatch,
         "partition binding");
  s.reset();
  status(sqpv::Store::open(root.path, m, o, s), sqpv::Status::ok, "reopen");
  sqfv::Batch read;
  sqpv::Receipt observed;
  status(s.read(7, c, read, observed), sqpv::Status::ok, "retained read");
  require(read.content_id() == b.content_id() &&
              observed.commit_sha256 == first.commit_sha256,
          "retained identity");
  status(s.read(8, c, read, observed), sqpv::Status::missing, "missing read");
  require(read.content_id() == b.content_id(), "failed read preserves batch");
  s.reset();
  wrong = o;
  wrong.limits.max_batches++;
  status(sqpv::Store::open(root.path, m, wrong, s),
         sqpv::Status::binding_mismatch, "exact options reopen");
}
void recovery_crash_boundaries() {
  for (int point = 1; point <= 9; ++point) {
    Root root;
    auto m = manifest();
    auto o = options();
    sqpv::Store s;
    status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
           "crash create");
    s.reset();
    pid_t pid = ::fork();
    require(pid >= 0, "fork");
    if (pid == 0) {
      auto c = context();
      auto b = batch(c, m, o, 7);
      sqpv::Store child;
      sqpv::Receipt r;
      if (sqpv::Store::open(root.path, m, o, child) != sqpv::Status::ok)
        ::_exit(22);
      sqpv::testing::crash_at(point);
      (void)child.append(c, b, r);
      ::_exit(23);
    }
    int result = 0;
    require(::waitpid(pid, &result, 0) == pid && WIFSIGNALED(result) &&
                WTERMSIG(result) == SIGKILL,
            "actual checkpoint SIGKILL");
    status(sqpv::Store::open(root.path, m, o, s), sqpv::Status::ok,
           "recover killed append");
    sqpv::Snapshot snapshot;
    status(s.snapshot(snapshot), sqpv::Status::ok, "recovered snapshot");
    require(snapshot.committed_batches == (point >= 5 ? 1U : 0U),
            "commit classification");
    require(snapshot.staged == (point <= 4), "staging classification");
    auto c = context();
    auto b = batch(c, m, o, 7);
    sqpv::Receipt r;
    if (point <= 4) {
      auto different = batch(c, m, o, 7, 2);
      status(s.append(c, different, r), sqpv::Status::conflict,
             "uncertain retry exactness");
    }
    status(s.append(c, b, r),
           point >= 5 ? sqpv::Status::duplicate : sqpv::Status::ok,
           "recover exact retry");
    require(r.batch_sequence == 7, "recovery receipt");
    sqfv::Batch retained;
    sqpv::Receipt retained_receipt;
    status(s.read(7, c, retained, retained_receipt), sqpv::Status::ok,
           "crash retained read");
    require(retained.content_id() == b.content_id() &&
                retained_receipt.frame_sha256 == r.frame_sha256 &&
                retained_receipt.commit_sha256 == r.commit_sha256,
            "crash read exact identity");
    sqfv::Lease lease;
    require(retained.acquire("private:test", lease) == sqfv::Status::ok,
            "crash read lease");
    require(lease.payload().size() == 1024 &&
                std::all_of(lease.payload().begin(), lease.payload().end(),
                            [](auto value) { return value == 1; }),
            "crash read payload");
  }
}
void initialization_crash_boundaries() {
  for (int point = 10; point <= 13; ++point) {
    Root root;
    auto m = manifest();
    auto o = options();
    pid_t pid = ::fork();
    require(pid >= 0, "init fork");
    if (pid == 0) {
      sqpv::Store s;
      sqpv::testing::crash_at(point);
      (void)sqpv::Store::create(root.path, m, o, s);
      ::_exit(23);
    }
    int result = 0;
    require(::waitpid(pid, &result, 0) == pid && WIFSIGNALED(result) &&
                WTERMSIG(result) == SIGKILL,
            "initialization SIGKILL");
    sqpv::Store s;
    status(point <= 11 ? sqpv::Store::create(root.path, m, o, s)
                       : sqpv::Store::open(root.path, m, o, s),
           sqpv::Status::ok, "recover initialization");
  }
}
void uncertain_failures_poison_handle() {
  for (int point = 1; point <= 9; ++point) {
    Root root;
    auto m = manifest();
    auto o = options();
    auto c = context();
    auto b = batch(c, m, o, 7);
    sqpv::Store s;
    sqpv::Receipt r;
    r.commit_sha256 = "unchanged";
    status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
           "failure create");
    sqpv::testing::fail_at(point);
    status(s.append(c, b, r), sqpv::Status::outcome_uncertain,
           "failure uncertainty");
    sqpv::testing::fail_at(0);
    require(r.commit_sha256 == "unchanged", "failure preserves receipt");
    sqpv::Snapshot snapshot;
    status(s.snapshot(snapshot), sqpv::Status::closed, "poisoned handle");
    s.reset();
    status(sqpv::Store::open(root.path, m, o, s), sqpv::Status::ok,
           "failure reopen");
    status(s.append(c, b, r),
           point >= 5 ? sqpv::Status::duplicate : sqpv::Status::ok,
           "failure retry");
  }
}
void partial_staging_retries() {
  for (int phase : {1, 4}) {
    Root root;
    auto m = manifest();
    auto o = options();
    auto c = context();
    auto b = batch(c, m, o, 7);
    sqpv::Store s;
    sqpv::Receipt r;
    status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
           "partial create");
    sqpv::testing::fail_at(phase);
    status(s.append(c, b, r), sqpv::Status::outcome_uncertain,
           "partial staged fault");
    sqpv::testing::fail_at(0);
    s.reset();
    for (const auto &entry : std::filesystem::directory_iterator(root.path))
      if (entry.path().filename().string().starts_with(
              phase == 1 ? "stage-frame-" : "stage-commit"))
        std::filesystem::resize_file(
            entry.path(), std::filesystem::file_size(entry.path()) / 2);
    status(sqpv::Store::open(root.path, m, o, s), sqpv::Status::ok,
           "partial open");
    status(s.append(c, b, r), sqpv::Status::ok, "partial exact retry");
  }
}
void allocation_failure_boundaries() {
  unsigned failed = 0;
  for (int selected = 0; selected < 1000; ++selected) {
    Root root;
    auto m = manifest();
    auto o = options();
    auto c = context();
    auto b = batch(c, m, o, 7);
    sqpv::Store s;
    sqpv::Receipt r;
    r.commit_sha256 = "unchanged";
    status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
           "allocation create");
    allocation_calls = 0;
    allocation_failure = selected;
    auto result = s.append(c, b, r);
    allocation_failure = -1;
    if (result == sqpv::Status::ok) {
      require(failed > 30, "allocation sweep exercised boundaries");
      std::printf("SQPV allocation failures checked: %u\n", failed);
      return;
    }
    require(result == sqpv::Status::no_memory ||
                result == sqpv::Status::outcome_uncertain,
            "allocation failure classification");
    require(r.commit_sha256 == "unchanged", "allocation output unchanged");
    ++failed;
    s.reset();
    status(sqpv::Store::open(root.path, m, o, s), sqpv::Status::ok,
           "allocation reopen");
    auto retried = s.append(c, b, r);
    require(retried == sqpv::Status::ok || retried == sqpv::Status::duplicate,
            "allocation exact recovery");
  }
  require(false, "allocation sweep bounded terminal success");
}

void create_allocation_failure_boundaries() {
  unsigned failed = 0, uncertain = 0;
  for (int selected = 0; selected < 1000; ++selected) {
    Root root;
    auto m = manifest();
    auto o = options();
    sqpv::Store s;
    allocation_calls = 0;
    allocation_failure = selected;
    auto result = sqpv::Store::create(root.path, m, o, s);
    allocation_failure = -1;
    if (result == sqpv::Status::ok) {
      require(failed > 30 && uncertain > 0,
              "create allocation sweep exercises both phases");
      std::printf(
          "SQPV create allocation failures checked: %u (%u uncertain)\n",
          failed, uncertain);
      return;
    }
    require(result == sqpv::Status::no_memory ||
                result == sqpv::Status::outcome_uncertain,
            "create allocation classification");
    require(!s, "failed create preserves empty output");
    ++failed;
    const bool empty = std::filesystem::is_empty(root.path);
    if (result == sqpv::Status::no_memory)
      require(empty, "no_memory must not hide persisted create state");
    else
      ++uncertain;
    const bool published = std::filesystem::exists(root.path + "/metadata");
    status(published ? sqpv::Store::open(root.path, m, o, s)
                     : sqpv::Store::create(root.path, m, o, s),
           sqpv::Status::ok, "exact failed-create reconciliation");
    sqpv::Snapshot snapshot;
    status(s.snapshot(snapshot), sqpv::Status::ok,
           "reconciled create snapshot");
    require(snapshot.committed_batches == 0 && !snapshot.staged,
            "reconciled initialization has no retained data");
  }
  require(false, "create allocation sweep bounded terminal success");
}

void corruption_and_missing() {
  for (int test = 0; test < 11; ++test) {
    Root root;
    auto m = manifest();
    auto o = options();
    auto c = context();
    sqpv::Store s;
    status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
           "corruption create");
    sqpv::Receipt r;
    for (std::uint64_t seq = 7; seq < 10; ++seq) {
      auto b = batch(c, m, o, seq);
      status(s.append(c, b, r), sqpv::Status::ok, "corruption append");
    }
    s.reset();
    switch (test) {
    case 0:
      corrupt(root.path + "/" + name("frame-", 7));
      break;
    case 1:
      std::filesystem::remove(root.path + "/" + name("frame-", 8));
      break;
    case 2:
      std::filesystem::remove(root.path + "/" + name("commit-", 9));
      break;
    case 3:
      corrupt(root.path + "/head");
      break;
    case 4:
      corrupt(root.path + "/" + name("commit-", 8));
      break;
    case 5:
      std::filesystem::remove(root.path + "/head");
      break;
    case 6:
      corrupt(root.path + "/metadata");
      break;
    case 7: {
      std::ofstream f(root.path + "/unexpected");
      f << "x";
    } break;
    case 9: {
      std::ofstream f(root.path + "/head", std::ios::trunc);
    } break;
    case 10: {
      std::ofstream f(root.path + "/stage-commit", std::ios::trunc);
      require(::chmod((root.path + "/stage-commit").c_str(), 0600) == 0,
              "stage mode");
    } break;
    case 8: {
      std::ofstream f(root.path + "/" + name("commit-", 9), std::ios::trunc);
      f << "x";
    } break;
    }
    auto result = sqpv::Store::open(root.path, m, o, s);
    require(result == sqpv::Status::corrupt ||
                result == sqpv::Status::missing ||
                result == sqpv::Status::binding_mismatch,
            "corruption/missing rejected");
    require(!s, "no handle on corrupt open");
  }
}
void capacity_and_output_preservation() {
  Root root;
  auto m = manifest();
  auto o = options();
  o.limits.max_batches = 1;
  auto c = context();
  sqpv::Store s;
  status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
         "capacity create");
  sqpv::Receipt receipt;
  auto b = batch(c, m, o, 7);
  status(s.append(c, b, receipt), sqpv::Status::ok, "capacity append");
  auto before = receipt;
  sqpv::Snapshot initial;
  status(s.snapshot(initial), sqpv::Status::ok, "before full");
  auto next = batch(c, m, o, 8);
  status(s.append(c, next, receipt), sqpv::Status::limit, "batch bound");
  require(receipt.commit_sha256 == before.commit_sha256,
          "full preserves receipt");
  sqpv::Snapshot after;
  status(s.snapshot(after), sqpv::Status::ok, "after full");
  require(after.store_bytes == initial.store_bytes &&
              after.committed_batches == initial.committed_batches,
          "full no mutation");
  Root small;
  auto limited = options();
  limited.limits.max_store_bytes = 8192;
  status(sqpv::Store::create(small.path, m, limited, s), sqpv::Status::ok,
         "small store create");
  auto large = batch(c, m, limited, 7, 1, 16384);
  status(s.append(c, large, receipt), sqpv::Status::limit, "byte bound");
  Root final;
  auto terminal = options();
  terminal.first_sequence = UINT64_MAX;
  status(sqpv::Store::create(final.path, m, terminal, s), sqpv::Status::ok,
         "terminal create");
  auto last = batch(c, m, terminal, UINT64_MAX);
  status(s.append(c, last, receipt), sqpv::Status::ok, "last sequence");
  status(s.snapshot(after), sqpv::Status::ok, "terminal snapshot");
  require(after.sequence_exhausted, "no sequence wrap");
  status(s.append(c, last, receipt), sqpv::Status::duplicate,
         "terminal duplicate");
  s.reset();
  status(sqpv::Store::open(final.path, m, terminal, s), sqpv::Status::ok,
         "terminal reopen");
  sqfv::Batch retained;
  sqpv::Receipt retained_receipt;
  status(s.read(UINT64_MAX, c, retained, retained_receipt), sqpv::Status::ok,
         "terminal retained read");
  require(retained.content_id() == last.content_id(),
          "terminal retained identity");
  status(s.append(c, last, receipt), sqpv::Status::duplicate,
         "terminal reopened retry");
  status(s.snapshot(after), sqpv::Status::ok, "terminal reopened snapshot");
  require(after.sequence_exhausted, "terminal recovery saturated");
}
void exclusive_ownership_and_paths() {
  Root root;
  auto m = manifest();
  auto o = options();
  sqpv::Store s, second;
  status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
         "lock create");
  status(sqpv::Store::open(root.path, m, o, second), sqpv::Status::busy,
         "exclusive writer");
  pid_t pid = ::fork();
  require(pid >= 0, "ownership fork");
  if (pid == 0) {
    sqpv::Snapshot view;
    ::_exit(s.snapshot(view) == sqpv::Status::stale ? 0 : 24);
  }
  int result = 0;
  require(::waitpid(pid, &result, 0) == pid && WIFEXITED(result) &&
              WEXITSTATUS(result) == 0,
          "inherited handle fenced");
  s.reset();
  Root link;
  std::filesystem::create_symlink(root.path, link.path + "/linked");
  status(sqpv::Store::open(link.path + "/linked", m, o, s),
         sqpv::Status::unsafe_path, "root symlink rejected");
  auto lock = root.path + "/writer.lock";
  std::filesystem::remove(lock);
  require(::mkfifo(lock.c_str(), 0600) == 0, "fifo fixture");
  status(sqpv::Store::open(root.path, m, o, s), sqpv::Status::unsafe_path,
         "special file rejection without block");
  Root moved;
  status(sqpv::Store::create(moved.path, m, o, s), sqpv::Status::ok,
         "replacement create");
  std::filesystem::rename(moved.path, moved.path + "-moved");
  require(::mkdir(moved.path.c_str(), 0700) == 0, "replacement root");
  sqpv::Snapshot snap;
  status(s.snapshot(snap), sqpv::Status::stale, "root identity replaced");
}
void fork_during_locked_operation() {
  Root root;
  auto m = manifest();
  auto o = options();
  auto c = context();
  auto b = batch(c, m, o, 7);
  sqpv::Store s;
  status(sqpv::Store::create(root.path, m, o, s), sqpv::Status::ok,
         "fork lock create");
  int notify[2], resume[2];
  require(::pipe(notify) == 0 && ::pipe(resume) == 0, "hook pipes");
  sqpv::testing::pause_at(1, notify[1], resume[0]);
  sqpv::Status appended = sqpv::Status::closed;
  std::thread writer([&] {
    sqpv::Receipt r;
    appended = s.append(c, b, r);
  });
  char signal;
  require(::read(notify[0], &signal, 1) == 1, "writer paused with mutex");
  pid_t pid = ::fork();
  require(pid >= 0, "locked fork");
  if (pid == 0) {
    ::alarm(3);
    sqpv::Snapshot snapshot;
    ::_exit(s.snapshot(snapshot) == sqpv::Status::stale ? 0 : 25);
  }
  int result = 0;
  require(::waitpid(pid, &result, 0) == pid && WIFEXITED(result) &&
              WEXITSTATUS(result) == 0,
          "fork rejected before inherited mutex");
  require(::write(resume[1], &signal, 1) == 1, "resume writer");
  writer.join();
  sqpv::testing::pause_at(0, -1, -1);
  status(appended, sqpv::Status::ok, "writer completes");
  for (int fd : {notify[0], notify[1], resume[0], resume[1]})
    ::close(fd);
}
} // namespace
int main() {
  commit_retry_and_exact_stream();
  recovery_crash_boundaries();
  initialization_crash_boundaries();
  uncertain_failures_poison_handle();
  partial_staging_retries();
  allocation_failure_boundaries();
  create_allocation_failure_boundaries();
  corruption_and_missing();
  capacity_and_output_preservation();
  exclusive_ownership_and_paths();
  fork_during_locked_operation();
  struct statfs volume{};
  require(::statfs("/private/tmp", &volume) == 0, "filesystem profile");
  std::printf("SQPV: 11 groups; 13 actual SIGKILL boundaries; 9 uncertain I/O "
              "boundaries; filesystem=%s; process-crash evidence only\n",
              volume.f_fstypename);
}
