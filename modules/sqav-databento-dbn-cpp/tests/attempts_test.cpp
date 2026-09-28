#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <signal.h>
#include <symphony/sqav/databento/attempts.hpp>
#include <sys/wait.h>
#include <unistd.h>

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

#include <thread>
namespace db = symphony::sqav::databento;
using S = db::AttemptStatus;
void check(bool b, const char *text) {
  if (!b) {
    std::fprintf(stderr, "attempts: %s\n", text);
    std::abort();
  }
}
struct Root {
  std::string path;
  Root() {
    char x[] = "/private/tmp/sqv-attempt-XXXXXX";
    auto p = ::mkdtemp(x);
    check(p, "root");
    path = p;
  }
  ~Root() { std::filesystem::remove_all(path); }
};
db::AttemptBudget budget() {
  db::AttemptBudget b{{}, 5'000'000'000, 15'646'000, 8, 1U << 20};
  b.generation[0] = 3;
  return b;
}
db::HistoricalPlan plan() {
  db::HistoricalPlan p;
  check(db::HistoricalPlan::create({"GLBX.MDP3", {"ESH1"}, 1, 100, 10},
                                   {{65536, 4096, 100}, 1000, 8, 3, 60},
                                   p) == db::Status::ok,
        "plan");
  return p;
}
db::AttemptQuote quote() {
  return {"fixture:quote", 1'000'000'000, 1000, 5000};
}
void accounting_and_restart() {
  Root root;
  db::AttemptLedger ledger;
  auto b = budget();
  auto p = plan();
  check(db::AttemptLedger::create(root.path, b, ledger) == S::ok, "create");
  db::AttemptTicket ticket;
  check(ledger.reserve(p, "attempt-1", quote(), 2000, ticket) == S::ok &&
            ticket.ordinal() == 1,
        "reserve");
  check(ledger.finish(ticket, db::AttemptOutcome::completed) ==
            S::invalid_argument,
        "unclaimed completion refused");
  check(ticket.claim(p, 2000) == S::ok && ticket.claim(p, 2000) == S::duplicate,
        "single claim");
  check(ledger.finish(ticket, db::AttemptOutcome::completed) == S::ok &&
            ledger.finish(ticket, db::AttemptOutcome::completed) ==
                S::duplicate,
        "finish");
  check(ledger.finish(ticket, db::AttemptOutcome::rejected) == S::conflict,
        "outcome conflict");
  check(ledger.reserve(p, "attempt-1", quote(), 2000, ticket) == S::conflict,
        "id reused");
  check(ledger.reserve(p, "attempt-2", quote(), 2000, ticket) == S::ok &&
            ticket.ordinal() == 2,
        "reserve pending");
  db::AttemptSnapshot snapshot;
  check(ledger.snapshot(snapshot) == S::ok &&
            snapshot.charged_ceiling_nano_usd == 2'015'646'000 &&
            snapshot.unresolved == 1,
        "accounting");
  ledger.reset();
  check(ticket.claim(p, 2000) == S::closed, "dead owner ticket");
  check(db::AttemptLedger::open(root.path, b, ledger) == S::ok, "reopen");
  db::AttemptOutcome outcome;
  check(ledger.lookup("attempt-2", outcome) == S::ok &&
            outcome == db::AttemptOutcome::indeterminate,
        "pending uncertainty");
  check(ledger.snapshot(snapshot) == S::ok &&
            snapshot.remaining_nano_usd == 2'984'354'000 &&
            snapshot.unresolved == 1,
        "charge retained");
  check(ledger.reserve(p, "attempt-2", quote(), 2000, ticket) == S::conflict,
        "no reissued ticket");
  check(ledger.reserve(p, "attempt-3", quote(), 2000, ticket) == S::ok &&
            ticket.ordinal() == 3,
        "new attempt");
  check(ledger.finish(ticket, db::AttemptOutcome::cancelled) == S::ok,
        "cancel reserved");
  check(ledger.reserve(p, "attempt-4", quote(), 2000, ticket) == S::limit,
        "plan retry limit");
}
void budgets_and_binding() {
  Root root;
  db::AttemptLedger ledger;
  auto b = budget();
  auto p = plan();
  db::AttemptTicket ticket;
  check(db::AttemptLedger::create(root.path, b, ledger) == S::ok,
        "create limits");
  auto q = quote();
  q.ceiling_nano_usd = UINT64_MAX;
  check(ledger.reserve(p, "overspend", q, 2000, ticket) == S::limit,
        "overflow refused");
  q = quote();
  q.expires_unix_ms = 2000;
  check(ledger.reserve(p, "expired", q, 2000, ticket) == S::invalid_argument,
        "expired quote");
  q = quote();
  q.quoted_unix_ms = 3000;
  check(ledger.reserve(p, "future", q, 2000, ticket) == S::invalid_argument,
        "future quote");
  q = quote();
  q.ceiling_nano_usd = b.ceiling_nano_usd - b.prior_charge_nano_usd;
  check(ledger.reserve(p, "exact", q, 2000, ticket) == S::ok,
        "exact budget boundary");
  check(ticket.claim(p, 5000) == S::stale, "expired ticket");
  db::AttemptTicket other;
  check(ledger.reserve(p, "extra", quote(), 2000, other) == S::limit,
        "finite total");
  check(ledger.finish(ticket, db::AttemptOutcome::cancelled) == S::ok,
        "cancel no refund");
  db::AttemptSnapshot snapshot;
  check(ledger.snapshot(snapshot) == S::ok && snapshot.remaining_nano_usd == 0,
        "no speculative refund");
  ledger.reset();
  b.ceiling_nano_usd++;
  check(db::AttemptLedger::open(root.path, b, ledger) == S::conflict,
        "budget binding");
}
void crash_reservation() {
  Root root;
  auto b = budget();
  {
    db::AttemptLedger l;
    check(db::AttemptLedger::create(root.path, b, l) == S::ok, "crash init");
  }
  int fd[2];
  check(::pipe(fd) == 0, "pipe");
  auto child = ::fork();
  check(child >= 0, "fork");
  if (child == 0) {
    ::close(fd[0]);
    db::AttemptLedger l;
    check(db::AttemptLedger::open(root.path, b, l) == S::ok, "child open");
    auto p = plan();
    db::AttemptTicket t;
    check(l.reserve(p, "crashed", quote(), 2000, t) == S::ok, "child reserve");
    const char c = 'x';
    check(::write(fd[1], &c, 1) == 1, "barrier");
    for (;;)
      ::pause();
  }
  ::close(fd[1]);
  char c = 0;
  check(::read(fd[0], &c, 1) == 1, "read");
  ::close(fd[0]);
  check(::kill(child, SIGKILL) == 0, "kill");
  int status = 0;
  check(::waitpid(child, &status, 0) == child && WIFSIGNALED(status), "wait");
  db::AttemptLedger l;
  check(db::AttemptLedger::open(root.path, b, l) == S::ok, "recover");
  db::AttemptSnapshot snapshot;
  check(l.snapshot(snapshot) == S::ok &&
            snapshot.charged_ceiling_nano_usd == 1'015'646'000 &&
            snapshot.unresolved == 1,
        "reserved charge survives crash");
  auto p = plan();
  db::AttemptTicket t;
  check(l.reserve(p, "crashed", quote(), 2000, t) == S::conflict,
        "crash no duplicate spend");
}

void one_shot_race_and_allocation() {
  {
    Root root;
    auto p = plan();
    db::AttemptLedger l;
    check(db::AttemptLedger::create(root.path, budget(), l) == S::ok,
          "race ledger");
    db::AttemptTicket t;
    check(l.reserve(p, "race", quote(), 2000, t) == S::ok, "race reserve");
    std::atomic<unsigned> accepted = 0;
    std::vector<std::thread> threads;
    for (unsigned i = 0; i < 16; ++i)
      threads.emplace_back([&] {
        auto s = t.claim(p, 2000);
        if (s == S::ok)
          ++accepted;
        else
          check(s == S::duplicate, "race duplicate");
      });
    for (auto &thread : threads)
      thread.join();
    check(accepted == 1, "exactly one claim");
  }
  long failures = 0;
  bool reached = false;
  for (long n = 0; n < 128; ++n) {
    Root root;
    auto p = plan();
    db::AttemptLedger l;
    check(db::AttemptLedger::create(root.path, budget(), l) == S::ok,
          "allocation ledger");
    db::AttemptTicket old;
    check(l.reserve(p, "old", quote(), 2000, old) == S::ok, "old ticket");
    auto q = quote();
    allocations = 0;
    fail_at = n;
    const auto status = l.reserve(p, "next", q, 2000, old);
    fail_at = -1;
    if (status == S::ok) {
      failures = n;
      reached = true;
      break;
    }
    check((status == S::no_memory || status == S::outcome_uncertain) &&
              old.ordinal() == 1,
          "reservation rollback");
    l.reset();
    check(db::AttemptLedger::open(root.path, budget(), l) == S::ok,
          "allocation recovery");
    db::AttemptSnapshot snapshot;
    check(l.snapshot(snapshot) == S::ok && snapshot.attempts >= 1 &&
              snapshot.attempts <= 2,
          "actual recovered entries");
    if (status == S::no_memory)
      check(snapshot.attempts == 1, "no memory did not reserve");
  }
  check(reached && failures > 0, "allocation sweep completed");
  std::printf("attempt reservation allocation points: %ld\n", failures);
}
int main(int argc, char** argv) {
  const bool no_fork = argc == 2 && std::string_view(argv[1]) == "--no-fork";
  if (argc != 1 && !no_fork) return 2;
  accounting_and_restart();
  budgets_and_binding();
  if (!no_fork) crash_reservation();
  one_shot_race_and_allocation();
  std::puts(no_fork ? "attempt ledger: accounting, restart, budget, identity and expiry passed; fork case excluded"
                    : "attempt ledger: accounting, restart, budget, identity, expiry and SIGKILL passed");
}
