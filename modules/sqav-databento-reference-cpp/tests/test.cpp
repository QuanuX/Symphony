#ifdef SQV_VERIFY_RETENTION
#include "retention.hpp"
#endif
#include <algorithm>
#include <cstdio>
#include <cstdlib>
#include <fstream>
#include <iterator>
#include <source_location>
#include <symphony/sqav/databento/reference.hpp>
#include <zstd.h>
namespace ref = symphony::sqav::databento::reference;
namespace db = symphony::sqav::databento;
namespace src = symphony::source;
void check(bool b, std::source_location loc = std::source_location::current()) {
  if (!b) {
    std::fprintf(stderr, "reference assertion line %u\n", loc.line());
    std::abort();
  }
}
src::HttpResponse compressed(std::string_view raw) {
  std::vector<std::uint8_t> b(ZSTD_compressBound(raw.size()));
  auto n = ZSTD_compress(b.data(), b.size(), raw.data(), raw.size(), 1);
  check(!ZSTD_isError(n));
  b.resize(n);
  return {200, std::move(b), true};
}
src::HttpResponse response(std::string_view s) {
  return {200, {s.begin(), s.end()}, true};
}
ref::Limits limits{{65536, 16384, 32768, 32}, 65536, 131072, 100, 23};
ref::Plan plan(ref::Operation op) {
  ref::Plan p;
  check(ref::Plan::create(
            {op,
             {"TEST"},
             op == ref::Operation::security_master_last ? "" : "2020-01-01",
             op == ref::Operation::security_master_last ? "" : "2026-01-01"},
            p) == src::Status::ok);
  return p;
}
void public_file(const char *path, ref::Operation op) {
  std::ifstream f(path, std::ios::binary);
  check(bool(f));
  std::string raw((std::istreambuf_iterator<char>(f)), {});
  ref::Page page;
  check(ref::Page::admit(plan(op), compressed(raw), limits, page) ==
            src::Status::ok &&
        page.records() == 2);
  check(std::string_view(reinterpret_cast<const char *>(page.jsonl().data()),
                         page.jsonl().size()) == raw);
}
int main(int argc, char **argv) {
  if (argc == 4) {
    public_file(argv[1], ref::Operation::security_master_range);
    public_file(argv[2], ref::Operation::corporate_actions);
    public_file(argv[3], ref::Operation::adjustment_factors);
    std::puts("official v0.84.0 reference fixtures: all six rows preserved "
              "byte-for-byte");
    return 0;
  }
  check(argc == 1);
  auto p = plan(ref::Operation::corporate_actions);
  check(p.request().post && p.request().parameters.find(
                                "allocate_isins=false") != std::string::npos);
  std::string raw =
      R"({"security_id":"S-1","event_id":"E-1","ts_record":"2020-01-01T00:00:00Z","factor":0.1234567890123456789})";
  raw += '\n';
  raw +=
      R"({"security_id":"S-1","event_id":"E-1","ts_record":"2020-01-02T00:00:00Z","factor":0.1234567890123456788})";
  raw += '\n';
  auto bytes = compressed(raw);
  ref::Page page;
  check(ref::Page::admit(p, bytes, limits, page) == src::Status::ok &&
        page.records() == 2);
  auto saved = std::string(page.reference());
  check(std::string_view(reinterpret_cast<const char *>(page.jsonl().data()),
                         page.jsonl().size()) == raw);
  for (std::size_t n = 0; n < bytes.body.size(); ++n) {
    auto bad = bytes;
    bad.body.resize(n);
    check(ref::Page::admit(p, bad, limits, page) != src::Status::ok &&
          page.reference() == saved);
  }
  auto small = limits;
  small.decompressed_bytes = 10;
  check(ref::Page::admit(p, bytes, small, page) == src::Status::limit);
  small = limits;
  small.records = 1;
  check(ref::Page::admit(p, bytes, small, page) == src::Status::limit);
  auto bad = bytes;
  bad.body.push_back(0);
  check(ref::Page::admit(p, bad, limits, page) == src::Status::malformed);
  bad = compressed("{\"security_id\":1,\"event_id\":1,\"event_id\":2}\n");
  check(ref::Page::admit(p, bad, limits, page) == src::Status::malformed);
  symphony::sqav::Capture c;
  check(page.capture("attempt", "fixture", "private:fixture",
                     {symphony::sqav::TimeRole::acquisition, "2026-09-28",
                      "iso-date", "UTC", "day", "fixture"},
                     {262144, 16384, 4096}, c) == src::Status::ok &&
        std::ranges::equal(c.original(), bytes.body));
  db::HistoricalPlan h;
  check(db::HistoricalPlan::create({"GLBX.MDP3", {"ESZ6"}, 1000, 2000, 100},
                                   {{65536, 4096, 100}, 10000, 2, 2, 60},
                                   h) == db::Status::ok);
  ref::CostQuote quote;
  src::HttpRequest q;
  check(ref::CostQuote::request(h, q) == src::Status::ok &&
        q.endpoint.ends_with("metadata.get_cost") &&
        q.parameters.find("limit=100") != std::string::npos);
  for (auto [input, expected] : {std::pair{"2.587353944778", 2587353945ULL},
                                 {"0", 0ULL},
                                 {"1e-100", 1ULL},
                                 {"0.000000001", 1ULL},
                                 {"1e1", 10000000000ULL},
                                 {"18446744073.709551615", UINT64_MAX}}) {
    check(ref::CostQuote::admit(h, response(input), 100, 1000, quote) ==
              src::Status::ok &&
          quote.nano_usd() == expected);
  }
  for (auto input : {"-1", "NaN", "1e100", "1.0garbage", "01", "1.",
                     "18446744073.709551616"})
    check(ref::CostQuote::admit(h, response(input), 100, 1000, quote) !=
          src::Status::ok);
  db::AttemptQuote ticket;
  check(quote.for_attempt(h, 200, ticket) == src::Status::ok &&
        ticket.ceiling_nano_usd == UINT64_MAX);
  check(quote.for_attempt(h, 1100, ticket) == src::Status::stale);
  db::HistoricalPlan left, right;
  check(h.split(left, right) == db::Status::ok);
  check(quote.for_attempt(left, 200, ticket) == src::Status::binding_mismatch);
#ifdef SQV_VERIFY_RETENTION
  source_composition::verify(c);
  source_composition::Root journal_root;
  db::AttemptBudget budget{{}, 2'000'000'000, 0, 4, 1U << 20};
  budget.generation[0] = 1;
  db::AttemptLedger ledger;
  check(db::AttemptLedger::create(journal_root.path, budget, ledger) ==
        db::AttemptStatus::ok);
  check(ref::CostQuote::admit(h, response("1.25"), 100, 1000, quote) ==
        src::Status::ok);
  check(quote.for_attempt(h, 200, ticket) == src::Status::ok);
  db::AttemptTicket execution;
  check(ledger.reserve(h, "reference-quoted-attempt", ticket, 200, execution) ==
        db::AttemptStatus::ok);
  check(execution.claim(h, 201) == db::AttemptStatus::ok);
  check(ledger.finish(execution, db::AttemptOutcome::completed) ==
        db::AttemptStatus::ok);
  ledger.reset();
  check(db::AttemptLedger::open(journal_root.path, budget, ledger) ==
        db::AttemptStatus::ok);
  db::AttemptSnapshot recovered;
  check(ledger.snapshot(recovered) == db::AttemptStatus::ok &&
        recovered.charged_ceiling_nano_usd == 1'250'000'000 &&
        recovered.attempts == 1);
  db::AttemptTicket duplicate;
  check(ledger.reserve(h, "reference-quoted-attempt", ticket, 202, duplicate) ==
        db::AttemptStatus::limit);
#endif
  std::puts("Databento reference: bounded zstd/JSONL, PIT fidelity, "
            "malformed/truncated data, exact cost rounding and request-bound "
            "quote expiry passed");
}
