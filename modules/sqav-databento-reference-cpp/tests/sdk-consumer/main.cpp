#include <cstdlib>
#include <chrono>
#include <filesystem>
#include <string>
#include <symphony/sqav/databento/reference.hpp>
#include <unistd.h>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  namespace ref = sqav::databento::reference;
  ref::Plan p;
  check(ref::Plan::create({ref::Operation::security_master_range,
                           {"AAPL"},
                           "2026-09-24",
                           "2026-09-25"},
                          p) == source::Status::ok);
  auto before = std::string(p.reference());
  check(ref::Plan::create({ref::Operation::security_master_range,
                           {"AAPL"},
                           "2026-09-25",
                           "2026-09-24"},
                          p) == source::Status::invalid_argument &&
        p.reference() == before);
  ref::Page page;
  source::HttpResponse response{200, {1, 2, 3}, true};
  check(ref::Page::admit(p, response,
                         {{4096, 128, 1024, 8}, 4096, 4096, 10, 20},
                         page) != source::Status::ok &&
        !page);

  namespace db = sqav::databento;
  db::HistoricalPlan historical;
  check(db::HistoricalPlan::create({"GLBX.MDP3", {"ESZ6"}, 1000, 2000, 100},
                                  {{65536, 4096, 100}, 10000, 2, 2, 60},
                                  historical) == db::Status::ok);
  const auto now = static_cast<std::uint64_t>(
      std::chrono::duration_cast<std::chrono::milliseconds>(
          std::chrono::system_clock::now().time_since_epoch()).count());
  ref::CostQuote quote;
  check(ref::CostQuote::admit(historical, {200, {'0'}, true}, now, 60000, quote) ==
        source::Status::ok);
  db::AttemptQuote evidence;
  check(quote.for_attempt(historical, now, evidence) == source::Status::ok);
  auto root = (std::filesystem::canonical(std::filesystem::temp_directory_path()) /
               "sqav-installed-acquisition-XXXXXX").string();
  check(::mkdtemp(root.data()) != nullptr);
  struct Cleanup {
    std::string path;
    ~Cleanup() { std::filesystem::remove_all(path); }
  } cleanup{root};
  db::AttemptBudget budget{{}, 100, 0, 1, 1U << 20};
  budget.generation[0] = 4;
  db::AttemptLedger ledger;
  check(db::AttemptLedger::create(root, budget, ledger) == db::AttemptStatus::ok);
  struct Deny final : db::SsiagHistoricalUse {
    unsigned calls = 0;
    db::HttpStatus with_key(std::string_view, std::uint32_t, std::stop_token,
                            db::HistoricalKeySink &) noexcept override {
      ++calls;
      return db::HttpStatus::not_authorized;
    }
  } deny;
  ref::AcquisitionAuthorization approval{
      "fixture:installed-approval", std::string(historical.reference()),
      evidence.evidence_ref, "installed-attempt", 0, now, now + 60000};
  db::HistoricalResponse original;
  const auto acquired = ref::acquire_quoted_historical(
      historical, quote, approval, ledger, deny, {100, 50, 4096}, {}, original);
  check(acquired.admission == source::Status::ok &&
        acquired.reservation == db::AttemptStatus::ok && acquired.execution_invoked &&
        acquired.execution.status == db::HttpStatus::not_authorized &&
        acquired.execution.persistence == db::AttemptStatus::ok && deny.calls == 1);
  db::AttemptSnapshot snapshot;
  check(ledger.snapshot(snapshot) == db::AttemptStatus::ok &&
        snapshot.attempts == 1 && snapshot.charged_ceiling_nano_usd == 0);
}
int main() { installed_boundary(); }
