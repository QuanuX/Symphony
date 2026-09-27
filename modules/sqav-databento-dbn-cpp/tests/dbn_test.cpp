#include <algorithm>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <new>
#include <source_location>
#include <symphony/sqav/databento/dbn.hpp>
#include <symphony/sqdv/delivery.hpp>
#include <unistd.h>
#include <vector>
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
namespace db = symphony::sqav::databento;
constexpr db::Limits limits{65536, 4096, 100};
void check(bool p, std::source_location loc = std::source_location::current()) {
  if (!p) {
    std::fprintf(stderr, "DBN test failure at line %u\n", loc.line());
    std::abort();
  }
}
std::vector<std::uint8_t> fixture() {
  // Public Databento C++ v0.68.0 test_data.mbo.v3.dbn, unchanged file bytes.
  // https://github.com/databento/databento-cpp/blob/v0.68.0/tests/data/test_data.mbo.v3.dbn
  constexpr std::string_view hex =
      "44424e0360010000474c42582e4d4450330000000000000000000020a0acdbe254160000"
      "8fc4df065516020000000000000001000047000000000000000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000100000045534831000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000010000004553483100000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000000000010000000c3f34010d3f3401353438320000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "000000000000000000000000000000000000000000000000000000000000000000000000"
      "0ea001006a15000007afa6acdbe254168945fed29600000080fb30c56203000001000000"
      "800043413cdeaaacdbe25416d1590000b0db11000ea001006a15000031b6a6acdbe25416"
      "3f45fed29600000000ae17d4620300000100000080004341b0faaaacdbe25416a54c0000"
      "b1db1100";
  std::vector<std::uint8_t> b;
  b.reserve(hex.size() / 2);
  auto n = [](char c) { return c <= '9' ? c - '0' : c - 'a' + 10; };
  for (std::size_t i = 0; i < hex.size(); i += 2)
    b.push_back(static_cast<std::uint8_t>((n(hex[i]) << 4) | n(hex[i + 1])));
  return b;
}

sqav::Description description() {
  return {{"databento", "public-sdk-fixture", "0.68.0", "offline-file-inspect",
           "caller", "caller", "GLBX.MDP3", "fixture-v0.68.0", "fixture:ESH1",
           std::string(db::native_schema), std::string(db::native_encoding),
           "private:fixture"},
          "attempt-1",
          "fixture-observer",
          "",
          sqav::Coverage::unknown,
          "public-fixture-only",
          "",
          std::nullopt,
          {{sqav::TimeRole::acquisition, "2026-09-27", "iso-date",
            "UTC-calendar", "day", "local-read"}}};
}
void public_fixture_fidelity() {
  auto b = fixture();
  db::FileView view;
  check(db::FileView::inspect(b, limits, view) == db::Status::ok);
  const auto &m = view.metadata();
  check(m.dataset == "GLBX.MDP3" && m.start == 1609160400000000000ULL &&
        m.end == 1609200000000000000ULL && m.limit == 2 &&
        m.record_count == 2 && m.symbols == 1 && m.mappings == 1 &&
        m.partial == 0 && m.not_found == 0 && m.symbol_cstr_len == 71 &&
        !m.ts_out);
  db::Mbo a, z;
  check(view.record(0, a) == db::Status::ok &&
        view.record(1, z) == db::Status::ok);
  check(a.publisher_id == 1 && a.instrument_id == 5482 &&
        a.ts_event == 1609160400000429831ULL && a.order_id == 647784973705ULL &&
        a.price == 3722750000000LL && a.size == 1 && a.flags == 128 &&
        a.channel_id == 0 && a.action == 'C' && a.side == 'A' &&
        a.ts_recv == 1609160400000704060ULL && a.ts_in_delta == 22993 &&
        a.sequence == 1170352 && !a.ts_out);
  check(z.ts_event == 1609160400000431665ULL && z.order_id == 647784973631ULL &&
        z.price == 3723000000000LL && z.ts_recv == 1609160400000711344ULL &&
        z.ts_in_delta == 19621 && z.sequence == 1170353);
  check(view.record(2, a) == db::Status::invalid_argument);
  // Unaligned transport bytes do not permit direct reinterpret_cast to provider
  // structs.
  std::vector<std::uint8_t> unaligned{99};
  unaligned.insert(unaligned.end(), b.begin(), b.end());
  check(db::FileView::inspect(sqav::ByteView(unaligned).subspan(1), limits,
                              view) == db::Status::ok);
  check(view.record(0, z) == db::Status::ok && z == a);
}
void raw_fields_and_gateway_timestamp() {
  auto b = fixture();
  std::vector<std::uint8_t> record(b.begin() + 360, b.begin() + 416);
  record[24] = 0;
  for (std::size_t i = 25; i < 31; ++i)
    record[i] = 0;
  record[31] = 128;
  for (std::size_t i = 8; i < 24; ++i)
    record[i] = 255;
  for (std::size_t i = 40; i < 52; ++i)
    record[i] = 255;
  record[36] = 255;
  record[38] = 255;
  record[39] = 255;
  db::Mbo m;
  check(db::decode_mbo(record, false, m) == db::Status::ok &&
        m.price == INT64_MIN && m.order_id == UINT64_MAX &&
        m.ts_event == UINT64_MAX && m.ts_recv == UINT64_MAX &&
        m.ts_in_delta == -1 && m.flags == 255 && m.action == 255 &&
        m.side == 255);
  record[0] = 16;
  record.resize(64, 255);
  check(db::decode_mbo(record, true, m) == db::Status::ok &&
        m.ts_out == UINT64_MAX);
  b.resize(360);
  b[52] = 1;
  b.insert(b.end(), record.begin(), record.end());
  db::FileView v;
  check(db::FileView::inspect(b, limits, v) == db::Status::ok &&
        v.metadata().record_count == 1 && v.record(0, m) == db::Status::ok &&
        m.ts_out == UINT64_MAX);
}
void malformed_and_limits() {
  auto b = fixture();
  db::FileView original;
  check(db::FileView::inspect(b, limits, original) == db::Status::ok);
  std::size_t rejected = 0;
  for (std::size_t n = 0; n < b.size(); ++n) {
    db::FileView out = original;
    auto s = db::FileView::inspect(sqav::ByteView(b).first(n), limits, out);
    if (n == 360 || n == 416)
      check(s == db::Status::ok);
    else {
      check(s != db::Status::ok && out.original().size() == b.size());
      ++rejected;
    }
  }
  std::printf("DBN truncations rejected: %zu; two valid record-boundary "
              "prefixes accepted\n",
              rejected);
  auto bad = b;
  bad[3] = 1;
  check(db::FileView::inspect(bad, limits, original) ==
        db::Status::unsupported);
  bad = b;
  bad[24] = 255;
  bad[25] = 255;
  check(db::FileView::inspect(bad, limits, original) ==
        db::Status::unsupported);
  bad = b;
  bad[52] = 2;
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  std::fill(bad.begin() + 112, bad.begin() + 116, 255);
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  std::fill(bad.begin() + 116, bad.begin() + 187, 65);
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  bad[360] = 0;
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  bad = b;
  bad[417] = 1;
  check(db::FileView::inspect(bad, limits, original) ==
        db::Status::unsupported);
  bad = b;
  bad.push_back(0);
  check(db::FileView::inspect(bad, limits, original) == db::Status::malformed);
  auto l = limits;
  l.max_records = 1;
  check(db::FileView::inspect(b, l, original) == db::Status::limit);
  l = limits;
  l.max_metadata_bytes = 128;
  check(db::FileView::inspect(b, l, original) == db::Status::limit);
  l = limits;
  l.max_file_bytes = 471;
  check(db::FileView::inspect(b, l, original) == db::Status::limit);
  // Parser and field access allocate no heap state, even with first-allocation
  // failure armed.
  allocations = 0;
  fail_at = 0;
  db::FileView view;
  auto status = db::FileView::inspect(b, limits, view);
  db::Mbo record;
  auto rs = view.record(0, record);
  fail_at = -1;
  check(status == db::Status::ok && rs == db::Status::ok && allocations == 0);
}
void capture_and_retained_delivery() {
  auto b = fixture();
  auto d = description();
  sqav::Capture capture;
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, capture) ==
        db::Status::ok);
  check(std::ranges::equal(capture.original(), b) &&
        capture.description().source_record_count == 2 &&
        capture.description().coverage == sqav::Coverage::unknown);
  db::FileView view;
  check(db::inspect_capture(capture, limits, view) == db::Status::ok);
  const auto before = std::string(capture.reference());
  d.source.dataset_id = "WRONG";
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, capture) ==
            db::Status::binding_mismatch &&
        capture.reference() == before);
  d = description();
  d.source_record_count = 3;
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, capture) ==
        db::Status::binding_mismatch);
  d = description();
  d.attempt_id.assign(4097, 'x');
  allocations = 0;
  fail_at = 0;
  const auto bounded =
      db::capture_file(d, b, limits, {65536, 16384, 4096}, capture);
  fail_at = -1;
  check(bounded == db::Status::limit && allocations == 0 &&
        capture.reference() == before);
  // Explicit partial-symbol evidence contradicts a complete coverage claim.
  auto partial = b;
  partial[187] = 1;
  partial.insert(partial.begin() + 191, 71, 0);
  partial.erase(partial.begin() + 424, partial.begin() + 431);
  partial[4] = 160;
  partial[5] = 1; // 416 metadata bytes after prefix.
  d = description();
  d.coverage = sqav::Coverage::complete;
  d.coverage_evidence_ref = "assertion";
  check(db::capture_file(d, partial, limits, {65536, 16384, 4096}, capture) ==
        db::Status::binding_mismatch);
  d.coverage = sqav::Coverage::partial;
  sqav::Capture partial_capture;
  check(db::capture_file(d, partial, limits, {65536, 16384, 4096},
                         partial_capture) == db::Status::ok);
  sqmv::Description md{
      "GLBX.MDP3",
      "fixture-v0.68.0",
      std::string(sqav::capture_schema),
      std::string(sqav::capture_layout),
      "private:fixture",
      "fixture-observer",
      {{sqmv::EvidenceRole::schema, "fixture", "capture-schema"},
       {sqmv::EvidenceRole::layout, "fixture", "capture-layout"},
       {sqmv::EvidenceRole::access, "fixture", "private"},
       {sqmv::EvidenceRole::source, "fixture-observer",
        std::string(capture.source_reference())}}};
  sqmv::Manifest manifest;
  check(sqmv::Manifest::create(md, {65536, 4096, 128}, manifest) ==
        sqmv::Status::ok);
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 1U << 20, 4}, flow) ==
        sqfv::Status::ok);
  sqav::Position pos{"partition", {}, 1};
  pos.producer_generation[0] = 1;
  sqfv::Batch batch;
  check(capture.prepare(flow, manifest, pos, batch) == sqav::Status::ok);
  char temp[] = "/private/tmp/sqv12-retention-XXXXXX";
  auto path = ::mkdtemp(temp);
  check(path);
  {
    sqpv::Options o{
        pos.partition, pos.producer_generation, {}, 1, {73728, 1U << 20, 4}};
    o.store_generation[0] = 1;
    sqdv::RetainedSource source;
    check(sqdv::RetainedSource::create(path, manifest, o, source) ==
          sqdv::Status::ok);
    sqdv::RetainedBatch proof;
    check(source.commit(flow, batch, proof) == sqdv::Status::ok);
    sqdv::Config cfg{"view",
                     "recipient",
                     "capture",
                     pos.partition,
                     pos.producer_generation,
                     1,
                     sqdv::Profile::retained_before_delivery};
    sqdv::Session session;
    check(sqdv::Session::create(flow, manifest, cfg, {65536, 2}, &source,
                                nullptr, session) == sqdv::Status::ok &&
          session.offer_next(flow) == sqdv::Status::ok);
    sqdv::Delivery delivery;
    check(session.take(delivery) == sqdv::Status::ok);
    sqav::Capture replay;
    check(sqav::Capture::from_delivery(
              delivery.payload(), delivery.descriptor(), manifest,
              {65536, 16384, 4096}, replay) == sqav::Status::ok);
    check(std::ranges::equal(replay.original(), b) &&
          db::inspect_capture(replay, limits, view) == db::Status::ok);
    db::Mbo record;
    check(view.record(1, record) == db::Status::ok &&
          record.sequence == 1170353);
    check(session.acknowledge_processed(delivery) == sqdv::Status::ok);
  }
  std::filesystem::remove_all(path);
}
void allocation_rollback() {
  auto b = fixture();
  auto d = description();
  sqav::Capture out;
  check(db::capture_file(d, b, limits, {65536, 16384, 4096}, out) ==
        db::Status::ok);
  auto ref = std::string(out.reference());
  long failures = 0;
  for (long n = 0; n < 256; ++n) {
    allocations = 0;
    fail_at = n;
    auto status = db::capture_file(d, b, limits, {65536, 16384, 4096}, out);
    fail_at = -1;
    if (status == db::Status::ok) {
      failures = n;
      break;
    }
    check(status == db::Status::no_memory && out.reference() == ref);
  }
  check(failures > 0);
  std::printf("DBN capture allocation failures: %ld\n", failures);
}
} // namespace
int main() {
  public_fixture_fidelity();
  raw_fields_and_gateway_timestamp();
  malformed_and_limits();
  capture_and_retained_delivery();
  allocation_rollback();
  std::puts("DBN native acceptance: 5 groups passed");
}
