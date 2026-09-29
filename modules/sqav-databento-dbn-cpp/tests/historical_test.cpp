#include "public_fixture.hpp"
#include <algorithm>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <new>
#include <source_location>
#include <symphony/sqav/databento/historical.hpp>
#include <symphony/sqdv/delivery.hpp>
#include <unistd.h>
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
namespace db = sqav::databento;
using S = db::Status;
constexpr db::HistoricalLimits limits{
    {65536, 4096, 100}, 86'400'000'000'000ULL, 8, 3, 60};
constexpr sqav::Limits capture_limits{65536, 16384, 4096};
void check(bool ok,
           std::source_location where = std::source_location::current()) {
  if (!ok) {
    std::fprintf(stderr, "historical failure at %u\n", where.line());
    std::abort();
  }
}
void put(std::vector<std::uint8_t> &b, std::size_t at, std::uint64_t n,
         unsigned width = 8) {
  for (unsigned i = 0; i < width; ++i)
    b[at + i] = static_cast<std::uint8_t>(n >> (8 * i));
}
db::HistoricalSelection selection(const std::vector<std::uint8_t> &b) {
  db::FileView f;
  check(db::FileView::inspect(b, limits.dbn, f) == S::ok);
  return {std::string(f.metadata().dataset),
          {"ESH1"},
          f.metadata().start,
          f.metadata().end,
          f.metadata().limit};
}
db::HistoricalAttribution attribution() {
  return {"offline-attempt",
          "fixture-observer",
          "public-fixture-v0.68.0",
          "private:fixture",
          {sqav::TimeRole::acquisition, "2026-09-28", "iso-date",
           "UTC-calendar", "day", "offline-fixture"}};
}
db::HistoricalResponse
response(const std::vector<std::uint8_t> &b, const db::HistoricalSelection &s,
         db::HistoricalReport &report,
         db::TransportEnd end = db::TransportEnd::complete) {
  db::HistoricalPlan p;
  check(db::HistoricalPlan::create(s, limits, p) == S::ok);
  db::HistoricalResponse r;
  check(db::HistoricalResponse::begin(p, 1, 200, {}, r) == S::ok);
  // Every byte is a possible transport-chunk boundary, independent of framing.
  for (const auto &byte : b)
    check(r.append({&byte, 1}) == S::ok);
  allocations = 0;
  fail_at = 0;
  auto status = r.finish(end, report);
  fail_at = -1;
  check(status == S::ok && allocations == 0 &&
        report.request.reference() == p.reference());
  return r;
}
void request_planning() {
  auto s = selection(public_fixture::fixture());
  db::HistoricalPlan p;
  check(db::HistoricalPlan::create(s, limits, p) == S::ok);
  const auto ref = std::string(p.reference());
  check(p.parameters().find(
            "schema=mbo&stype_in=raw_symbol&stype_out=instrument_id") !=
        std::string_view::npos);
  auto revised = s;
  revised.end = s.start;
  check(db::HistoricalPlan::create(revised, limits, p) == S::invalid_argument &&
        p.reference() == ref);
  revised = s;
  revised.record_limit = 101;
  check(db::HistoricalPlan::create(revised, limits, p) == S::limit);
  revised = s;
  revised.symbols = {"ALL_SYMBOLS"};
  check(db::HistoricalPlan::create(revised, limits, p) == S::invalid_argument);
  revised.symbols = {"ES,H1"};
  check(db::HistoricalPlan::create(revised, limits, p) == S::invalid_argument);
  revised.symbols = {"ESH1", "ESH1"};
  check(db::HistoricalPlan::create(revised, limits, p) == S::invalid_argument);
  revised.symbols = {"A+B&x=y", "ESH1"};
  check(db::HistoricalPlan::create(revised, limits, p) == S::ok);
  check(p.parameters().find("symbols=A%2BB%26x%3Dy%2CESH1&") !=
        std::string_view::npos);
  const auto ordered = std::string(p.reference());
  std::reverse(revised.symbols.begin(), revised.symbols.end());
  check(db::HistoricalPlan::create(revised, limits, p) == S::ok &&
        p.reference() == ordered);
  db::HistoricalPlan a, b;
  check(p.split(a, b) == S::ok);
  check(a.selection()->start == s.start &&
        a.selection()->end == b.selection()->start &&
        b.selection()->end == s.end);
  check(a.reference() != b.reference() && p.split(a, a) == S::invalid_argument);
  revised = s;
  revised.end = revised.start + 1;
  check(db::HistoricalPlan::create(revised, limits, p) == S::ok);
  const auto saved = std::string(a.reference());
  check(p.split(a, b) == S::limit && a.reference() == saved);
  // Output aliases the source only after both new plans have been prepared.
  check(db::HistoricalPlan::create(s, limits, p) == S::ok &&
        p.split(p, b) == S::ok && p.selection()->end == b.selection()->start);
}
void response_binding() {
  for (auto b : {public_fixture::fixture(), public_fixture::fixture_v1()}) {
    auto s = selection(b);
    db::HistoricalReport report;
    auto r = response(b, s, report);
    check(report.outcome == db::HistoricalOutcome::partial_window &&
          report.record_limit_reached &&
          report.recovery == db::Recovery::split_window &&
          report.records == 2 && report.coverage == sqav::Coverage::partial);
    sqav::Capture capture;
    check(r.capture(attribution(), capture_limits, capture) == S::ok &&
          std::ranges::equal(capture.original(), b) &&
          capture.description().coverage == sqav::Coverage::partial &&
          capture.description().source_position.find("capped=1") !=
              std::string::npos);
    // Synthetic complete-window fixture: increase declared/requested cap,
    // retain records.
    put(b, 42, 3);
    s.record_limit = 3;
    r = response(b, s, report);
    check(report.outcome == db::HistoricalOutcome::complete_window &&
          report.recovery == db::Recovery::none);
    check(r.capture(attribution(), capture_limits, capture) == S::ok &&
          capture.description().coverage == sqav::Coverage::complete);
    const auto ref = std::string(capture.reference());
    auto a = attribution();
    a.acquisition.role = sqav::TimeRole::event;
    check(r.capture(a, capture_limits, capture) == S::invalid_argument &&
          capture.reference() == ref);
    auto interrupted = response(b, s, report, db::TransportEnd::interrupted);
    check(report.outcome == db::HistoricalOutcome::interrupted &&
          report.recovery == db::Recovery::repeat_window &&
          report.coverage == sqav::Coverage::gap);
    check(interrupted.capture(attribution(), capture_limits, capture) ==
              S::binding_mismatch &&
          capture.reference() == ref);
    auto bad = s;
    bad.symbols = {"AAPL"};
    r = response(b, bad, report);
    check(report.outcome == db::HistoricalOutcome::binding_mismatch);
    bad = s;
    bad.dataset = "XNAS.ITCH";
    r = response(b, bad, report);
    check(report.outcome == db::HistoricalOutcome::binding_mismatch);
    bad = s;
    bad.start++;
    r = response(b, bad, report);
    check(report.outcome == db::HistoricalOutcome::binding_mismatch);
    bad = s;
    bad.record_limit = 4;
    r = response(b, bad, report);
    check(report.outcome == db::HistoricalOutcome::binding_mismatch);
    const auto offset = b[3] == 1 ? 206U : 360U;
    auto corrupt = b;
    put(corrupt, offset + 40, s.end);
    r = response(corrupt, s, report);
    check(report.outcome == db::HistoricalOutcome::binding_mismatch &&
          !report.first_ts_recv);
    corrupt = b;
    std::copy_n(b.begin() + offset + 40, 8, corrupt.begin() + offset + 96);
    r = response(corrupt, s, report);
    check(report.outcome == db::HistoricalOutcome::complete_window &&
          report.first_ts_recv == report.last_ts_recv);
    corrupt = b;
    put(corrupt, offset + 96, s.start);
    r = response(corrupt, s, report);
    check(report.outcome == db::HistoricalOutcome::binding_mismatch);
    corrupt = b;
    corrupt.pop_back();
    r = response(corrupt, s, report);
    check(report.outcome == db::HistoricalOutcome::invalid_dbn);
    corrupt = b;
    corrupt.resize(offset);
    r = response(corrupt, s, report);
    check(report.outcome == db::HistoricalOutcome::complete_window &&
          report.records == 0 && !report.first_ts_recv);
    // A partial-symbol list is structurally valid, but prevents complete
    // coverage.
    corrupt = b;
    const auto width = b[3] == 1 ? 22U : 71U;
    const auto partial = 116 + width;
    const unsigned old_padding = b[3] == 3 ? 7 : 0;
    corrupt.erase(corrupt.begin() + offset - old_padding,
                  corrupt.begin() + offset);
    put(corrupt, partial, 1, 4);
    corrupt.insert(corrupt.begin() + partial + 4, width, 0);
    std::copy_n(b.begin() + 116, width, corrupt.begin() + partial + 4);
    auto metadata = offset - old_padding + width;
    if (b[3] == 3) {
      const auto padding = (8 - metadata % 8) % 8;
      corrupt.insert(corrupt.begin() + metadata, padding, 0);
      metadata += padding;
    }
    put(corrupt, 4, metadata - 8, 4);
    r = response(corrupt, s, report);
    check(report.outcome == db::HistoricalOutcome::partial_window &&
          report.unresolved_symbols && report.recovery == db::Recovery::review);
  }
}
void multi_symbol_binding() {
  const auto original = public_fixture::fixture();
  auto selected = selection(original);
  selected.symbols = {"AAPL", "ESH1"};
  selected.record_limit = 3;
  auto bytes =
      std::vector<std::uint8_t>(original.begin(), original.begin() + 108);
  auto count = [&](std::uint32_t n) {
    const auto at = bytes.size();
    bytes.resize(at + 4);
    put(bytes, at, n, 4);
  };
  auto symbol = [&](std::string_view text) {
    const auto at = bytes.size();
    bytes.resize(at + 71);
    std::copy(text.begin(), text.end(), bytes.begin() + at);
  };
  count(0);
  count(2);
  symbol("ESH1");
  symbol("AAPL");
  count(0);
  count(0);
  count(0);
  bytes.resize((bytes.size() + 7) / 8 * 8);
  put(bytes, 4, bytes.size() - 8, 4);
  put(bytes, 42, 3);
  bytes.insert(bytes.end(), original.begin() + 360, original.end());
  db::HistoricalReport report;
  auto good = response(bytes, selected, report);
  check(report.outcome == db::HistoricalOutcome::complete_window);
  db::FileView view;
  check(db::FileView::inspect(bytes, limits.dbn, view) == S::ok);
  std::string_view name = "unchanged";
  check(view.symbol(2, name) == S::invalid_argument && name == "unchanged");
  check(view.symbol(0, name) == S::ok && name == "ESH1");
  std::copy_n(bytes.begin() + 116, 71, bytes.begin() + 187);
  auto bad = response(bytes, selected, report);
  check(report.outcome == db::HistoricalOutcome::binding_mismatch);
  // A report retains its exact plan even after the accumulator is released.
  bad = {};
  check(report.request &&
        report.request.selection()->symbols == selected.symbols);
}
void transport_and_budgets() {
  auto b = public_fixture::fixture();
  auto s = selection(b);
  db::HistoricalPlan p;
  check(db::HistoricalPlan::create(s, limits, p) == S::ok);
  for (const auto status :
       {0, 200, 400, 401, 403, 404, 429, 500, 502, 503, 504}) {
    db::HistoricalResponse r;
    db::HistoricalReport report;
    check(db::HistoricalResponse::begin(
              p, 1, static_cast<std::uint16_t>(status), 12, r) == S::ok);
    check(r.append(b) == S::ok);
    if (status != 200)
      check(r.body().empty());
    check(r.finish(db::TransportEnd::interrupted, report) == S::ok);
    const bool retry = status == 0 || status == 200 || status == 429 ||
                       status == 502 || status == 503 || status == 504;
    check(report.recovery ==
          (retry ? db::Recovery::repeat_window : db::Recovery::review));
    check(r.finish(db::TransportEnd::complete, report) == S::invalid_argument &&
          r.append({}) == S::invalid_argument);
  }
  for (auto attempt : {1, 3})
    for (auto wait : {12, 61}) {
      db::HistoricalResponse r;
      db::HistoricalReport report;
      check(db::HistoricalResponse::begin(p, static_cast<std::uint8_t>(attempt),
                                          429, wait, r) == S::ok);
      check(r.finish(db::TransportEnd::complete, report) == S::ok &&
            report.outcome == db::HistoricalOutcome::http_error);
      check(report.recovery == ((attempt == 3 || wait == 61)
                                    ? db::Recovery::review
                                    : db::Recovery::repeat_window));
      check(report.retry_after_seconds == wait);
    }
  auto small = limits;
  small.dbn.max_file_bytes = small.dbn.max_metadata_bytes = 400;
  check(db::HistoricalPlan::create(s, small, p) == S::ok);
  db::HistoricalResponse r;
  check(db::HistoricalResponse::begin(p, 1, 200, {}, r) == S::ok);
  check(r.append(sqav::ByteView(b).first(100)) == S::ok);
  check(r.append(r.body()) == S::invalid_argument && r.body().size() == 100);
  check(r.append(b) == S::limit && r.body().size() == 100 &&
        r.append({}) == S::limit);
  db::HistoricalReport report;
  check(r.finish(db::TransportEnd::complete, report) == S::ok &&
        report.outcome == db::HistoricalOutcome::byte_limit);
  check(db::HistoricalResponse::begin(p, 4, 200, {}, r) == S::invalid_argument);
  check(db::HistoricalResponse::begin(p, 1, 200, {}, r) == S::ok &&
        r.finish(db::TransportEnd::cancelled, report) == S::ok &&
        report.recovery == db::Recovery::none);
}
void allocation_rollback() {
  auto b = public_fixture::fixture();
  auto s = selection(b);
  db::HistoricalPlan p;
  check(db::HistoricalPlan::create(s, limits, p) == S::ok);
  const auto ref = std::string(p.reference());
  long failures = 0;
  for (long n = 0; n < 128; ++n) {
    allocations = 0;
    fail_at = n;
    auto status = db::HistoricalPlan::create(s, limits, p);
    fail_at = -1;
    if (status == S::ok) {
      failures = n;
      break;
    }
    check(status == S::no_memory && p.reference() == ref);
  }
  check(failures > 0);
  db::HistoricalPlan a = p, z = p;
  for (long n = 0; n < 256; ++n) {
    allocations = 0;
    fail_at = n;
    auto status = p.split(a, z);
    fail_at = -1;
    if (status == S::ok) {
      failures += n;
      break;
    }
    check(status == S::no_memory && a.reference() == ref &&
          z.reference() == ref);
  }
  db::HistoricalResponse r;
  check(db::HistoricalResponse::begin(p, 1, 200, {}, r) == S::ok);
  allocations = 0;
  fail_at = 0;
  auto status = r.append(b);
  fail_at = -1;
  check(status == S::no_memory && r.body().empty());
  check(r.append(b) == S::ok);
  allocations = 0;
  fail_at = 0;
  status = db::HistoricalResponse::begin(p, 2, 200, {}, r);
  fail_at = -1;
  check(status == S::no_memory && std::ranges::equal(r.body(), b));
  db::HistoricalReport report;
  check(r.finish(db::TransportEnd::complete, report) == S::ok);
  sqav::Capture c;
  check(r.capture(attribution(), capture_limits, c) == S::ok);
  const auto capture_ref = std::string(c.reference());
  const auto at = attribution();
  for (long n = 0; n < 256; ++n) {
    allocations = 0;
    fail_at = n;
    status = r.capture(at, capture_limits, c);
    fail_at = -1;
    if (status == S::ok) {
      failures += n;
      break;
    }
    check(status == S::no_memory && c.reference() == capture_ref);
  }
  std::printf("historical allocation rollback points: %ld\n", failures);
}
void historical_retained_replay() {
  auto b = public_fixture::fixture();
  db::HistoricalReport report;
  auto response_value = response(b, selection(b), report);
  sqav::Capture capture;
  check(response_value.capture(attribution(), capture_limits, capture) ==
        S::ok);
  sqmv::Manifest manifest;
  check(capture.metadata(
            false, {sqmv::EvidenceRole::access, "fixture", "private:fixture"},
            {65536, 4096, 128}, manifest) == sqav::Status::ok);
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 1U << 20, 8}, flow) ==
        sqfv::Status::ok);
  sqav::Position pos{"historical", {}, 1};
  pos.producer_generation[0] = 1;
  sqfv::Batch batch;
  check(capture.prepare(flow, manifest, pos, batch) == sqav::Status::ok);
  char temp[] = "/private/tmp/sqv21-history-XXXXXX";
  auto root = ::mkdtemp(temp);
  check(root);
  {
    sqpv::Options options{
        pos.partition, pos.producer_generation, {}, 1, {73728, 1U << 20, 8}};
    options.store_generation[0] = 1;
    sqdv::RetainedSource source;
    check(sqdv::RetainedSource::create_async(root, manifest, options, flow,
                                             {2, 147456},
                                             source) == sqdv::Status::ok);
    sqdv::QueuedBatch queued;
    check(source.enqueue(batch, queued) == sqdv::Status::ok);
    sqdv::Config config{"session",
                        "reader",
                        "capture",
                        pos.partition,
                        pos.producer_generation,
                        1,
                        sqdv::Profile::asynchronous_retention};
    sqdv::Session session;
    check(sqdv::Session::create(flow, manifest, config, {65536, 2}, &source,
                                nullptr, session) == sqdv::Status::ok);
    check(session.offer_preview(queued) == sqdv::Status::ok);
    sqdv::Delivery preview;
    check(session.take(preview) == sqdv::Status::ok);
    check(session.acknowledge_processed(preview) == sqdv::Status::ok);
    check(source.finish_retention() == sqdv::Status::ok);
  }
  {
    sqpv::Options options{
        pos.partition, pos.producer_generation, {}, 1, {73728, 1U << 20, 8}};
    options.store_generation[0] = 1;
    sqdv::RetainedSource source;
    check(sqdv::RetainedSource::open(root, manifest, options, source) ==
          sqdv::Status::ok);
    sqdv::Config config{"replay",
                        "reader",
                        "capture",
                        pos.partition,
                        pos.producer_generation,
                        1,
                        sqdv::Profile::retained_before_delivery};
    sqdv::Session session;
    check(sqdv::Session::create(flow, manifest, config, {65536, 2}, &source,
                                nullptr, session) == sqdv::Status::ok);
    check(session.offer_next(flow) == sqdv::Status::ok);
    sqdv::Delivery delivery;
    check(session.take(delivery) == sqdv::Status::ok);
    sqav::Capture replay;
    check(sqav::Capture::from_delivery(
              delivery.payload(), delivery.descriptor(), manifest,
              capture_limits, replay) == sqav::Status::ok);
    check(replay.reference() == capture.reference() &&
          std::ranges::equal(replay.original(), b) &&
          replay.description().coverage == sqav::Coverage::partial &&
          replay.description().coverage_scope ==
              capture.description().coverage_scope &&
          replay.description().source_position ==
              capture.description().source_position);
  }
  std::filesystem::remove_all(root);
}
} // namespace
int main() {
  request_planning();
  response_binding();
  multi_symbol_binding();
  transport_and_budgets();
  allocation_rollback();
  historical_retained_replay();
  std::puts("historical native acceptance: 6 groups passed");
}
