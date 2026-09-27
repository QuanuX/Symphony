#include <algorithm>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <limits>
#include <new>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/sqav/capture.hpp>
#include <symphony/sqdv/delivery.hpp>
#include <unistd.h>

namespace {
std::atomic<long> fail_at{-1}, allocations{0};
}
void *operator new(std::size_t n) {
  if (fail_at.load() >= 0 && allocations.fetch_add(1) == fail_at.load())
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
constexpr sqav::Limits limits{1U << 20, 65536, 4096};
void require(bool p, const char *msg) {
  if (!p) {
    std::fprintf(stderr, "sqav: %s\n", msg);
    std::abort();
  }
}
void ok(sqav::Status s, const char *msg) {
  require(s == sqav::Status::ok, msg);
}
sqav::Description description() {
  return {
      {"provider:offline-fixture", "interface:original-byte-input",
       "interface-version:exact-1", "operation:offline-capture",
       "adapter:fixture-capture", "adapter-version:exact-1",
       "dataset:provider-native-fixture", "revision:2026-09-26",
       "selection:record-block-7", "schema:provider-native-test-v1",
       "encoding:opaque-original-bytes", "access:private-test"},
      "attempt:fixture-0001",
      "observer:fixture-producer",
      "provider-position:900",
      sqav::Coverage::partial,
      "coverage:requested-block-7",
      "evidence:partial-response",
      std::nullopt,
      {{sqav::TimeRole::publication, "2026-09-26", "format:iso-date",
        "calendar:provider-dates", "precision:day",
        "evidence:publication-date"},
       {sqav::TimeRole::acquisition, "2026-09-26T12:00:00Z", "format:iso-8601",
        "clock:fixture-utc", "precision:second", "evidence:fixture-clock"}}};
}
std::vector<std::uint8_t> raw() {
  return {0, 255, 128, '{', '"', 'x', '"', ':', '1', '}', 0, '\n'};
}
sqav::Capture capture() {
  sqav::Capture c;
  ok(sqav::Capture::create(description(), raw(), limits, c), "create fixture");
  return c;
}
sqmv::Manifest metadata(const sqav::Capture &c) {
  const auto &d = c.description();
  sqmv::Description md{d.source.dataset_id,
                       d.source.dataset_revision,
                       std::string(sqav::capture_schema),
                       std::string(sqav::capture_layout),
                       d.source.access_scope,
                       d.attribution_ref,
                       {{sqmv::EvidenceRole::schema, "producer:format",
                         "evidence:capture-schema"},
                        {sqmv::EvidenceRole::layout, "producer:format",
                         "evidence:capture-layout"},
                        {sqmv::EvidenceRole::access, d.attribution_ref,
                         "evidence:private-access"},
                        {sqmv::EvidenceRole::source, d.attribution_ref,
                         std::string(c.source_reference())}}};
  sqmv::Manifest m;
  require(sqmv::Manifest::create(md, {65536, 4096, 128}, m) == sqmv::Status::ok,
          "manifest");
  return m;
}
sqfv::Context context() {
  sqfv::Context c;
  require(sqfv::Context::create({1U << 20, 2U << 20, 4096, 16U << 20, 8}, c) ==
              sqfv::Status::ok,
          "context");
  return c;
}
sqav::Position position(std::uint64_t n = 7) {
  sqav::Position p{"partition:fixture", {}, n};
  p.producer_generation[0] = 1;
  return p;
}
void bytes_equal(sqav::ByteView a, sqav::ByteView b, const char *msg) {
  require(std::ranges::equal(a, b), msg);
}
void exact_capture_roundtrip() {
  auto d = description();
  auto data = raw();
  sqav::Capture c;
  ok(sqav::Capture::create(d, data, limits, c), "create");
  require(c.description().times[0].role == sqav::TimeRole::acquisition,
          "canonical role order");
  require(c.description().times[1].value == "2026-09-26" &&
              c.description().times[1].precision_ref == "precision:day",
          "no invented time precision");
  require(!c.description().source_record_count,
          "unknown count distinct from zero");
  auto ref = std::string(c.reference());
  std::reverse(d.times.begin(), d.times.end());
  sqav::Capture same;
  ok(sqav::Capture::create(d, data, limits, same), "permuted times");
  require(same.reference() == ref, "canonical identity");
  d.source_record_count = 0;
  sqav::Capture zero;
  ok(sqav::Capture::create(d, data, limits, zero), "known zero");
  require(zero.reference() != ref, "known zero differs");
  d.attempt_id = "attempt:fixture-0002";
  sqav::Capture retry;
  ok(sqav::Capture::create(d, data, limits, retry), "attempt identity");
  require(retry.source_reference() == c.source_reference() &&
              retry.reference() != ref,
          "source stable attempt distinct");
  data[0] = 42;
  d.source.provider_ref = "changed";
  bytes_equal(c.original(), raw(), "input copy isolated");
  sqav::Capture decoded;
  ok(sqav::Capture::resolve(c.encoded(), c.reference(), limits, decoded),
     "decode");
  bytes_equal(c.encoded(), decoded.encoded(), "entire capture exact");
  bytes_equal(decoded.original(), raw(), "original bytes exact");
  ok(decoded.retain(decoded), "retain self alias");
  ok(sqav::Capture::resolve(decoded.encoded(), decoded.reference(), limits,
                            decoded),
     "resolve self alias");
  for (unsigned n = 0; n < 32; ++n) {
    std::vector<std::uint8_t> b(n);
    for (unsigned j = 0; j < n; ++j)
      b[j] = static_cast<std::uint8_t>(j * 73 + n);
    auto x = description();
    x.source_position = std::string("\0\xff", 2);
    x.coverage = sqav::Coverage::unknown;
    x.coverage_evidence_ref.clear();
    sqav::Capture a, z;
    ok(sqav::Capture::create(x, b, limits, a), "bounded property create");
    ok(sqav::Capture::resolve(a.encoded(), a.reference(), limits, z),
       "bounded property resolve");
    bytes_equal(z.original(), b, "binary property");
  }
}
void malformed_and_limits() {
  auto c = capture();
  auto out = capture();
  const auto before = std::string(out.reference());
  const std::vector<std::uint8_t> original(c.encoded().begin(),
                                           c.encoded().end());
  for (std::size_t n = 0; n < original.size(); ++n)
    require(sqav::Capture::resolve({original.data(), n}, c.reference(), limits,
                                   out) != sqav::Status::ok,
            "all truncations reject");
  for (std::size_t i = 0; i < original.size(); ++i) {
    auto b = original;
    b[i] ^= 1;
    require(sqav::Capture::resolve(b, c.reference(), limits, out) !=
                sqav::Status::ok,
            "all single byte mutations reject");
  }
  auto b = original;
  b.push_back(0);
  require(sqav::Capture::resolve(b, c.reference(), limits, out) ==
              sqav::Status::corrupt_capture,
          "trailing bytes");
  b = original;
  b[5] = 2;
  require(sqav::Capture::resolve(b, c.reference(), limits, out) ==
              sqav::Status::unsupported_capture,
          "future version");
  b = original;
  b[8] = 1;
  require(sqav::Capture::resolve(b, c.reference(), limits, out) ==
              sqav::Status::unsupported_capture,
          "flags unsupported");
  auto wrong = std::string(c.reference());
  wrong.back() = wrong.back() == '0' ? '1' : '0';
  require(sqav::Capture::resolve(c.encoded(), wrong, limits, out) ==
              sqav::Status::reference_mismatch,
          "expected ref mismatch");
  auto l = limits;
  l.max_capture_bytes = original.size() - 1;
  require(sqav::Capture::resolve(c.encoded(), c.reference(), l, out) ==
              sqav::Status::limit,
          "encoded cap");
  auto d = description();
  d.times.push_back(d.times[0]);
  require(sqav::Capture::create(d, raw(), limits, out) ==
              sqav::Status::invalid_argument,
          "duplicate role");
  d = description();
  d.times.erase(d.times.begin() + 1);
  require(sqav::Capture::create(d, raw(), limits, out) ==
              sqav::Status::invalid_argument,
          "acquisition required");
  d = description();
  d.coverage_evidence_ref.clear();
  require(sqav::Capture::create(d, raw(), limits, out) ==
              sqav::Status::invalid_argument,
          "partial requires evidence");
  d = description();
  d.source.interface_version.clear();
  require(sqav::Capture::create(d, raw(), limits, out) ==
              sqav::Status::invalid_argument,
          "exact interface required");
  d = description();
  d.source.native_schema_ref.assign(4097, 'a');
  require(sqav::Capture::create(d, raw(), limits, out) == sqav::Status::limit,
          "field cap");
  d = description();
  l = limits;
  l.max_metadata_bytes = 10;
  require(sqav::Capture::create(d, raw(), l, out) == sqav::Status::limit,
          "metadata cap");
  require(out.reference() == before, "failure output preservation");
  // Recompute integrity for malformed structure: a valid digest cannot admit
  // duplicate time roles.
  b = original;
  std::size_t p = 24;
  auto skip = [&] {
    const auto n = (b[p] << 8) | b[p + 1];
    p += 2 + static_cast<std::size_t>(n);
  };
  for (int i = 0; i < 15; ++i)
    skip();
  ++p;
  skip();
  skip();
  p += 10;
  const auto first = p;
  ++p;
  for (int i = 0; i < 5; ++i)
    skip();
  b[p] = b[first];
  auto h =
      knowledge::engine::sha256_hex(sqav::ByteView(b).first(b.size() - 32));
  for (std::size_t i = 0; i < 32; ++i)
    b[b.size() - 32 + i] =
        static_cast<std::uint8_t>(std::stoul(h.substr(i * 2, 2), nullptr, 16));
  require(sqav::Capture::resolve(b, "sqac1-sha256-" + h, limits, out) ==
              sqav::Status::corrupt_capture,
          "canonical role order validated independent of digest");
}
void binding_and_original_identity() {
  auto c = capture();
  auto m = metadata(c);
  auto flow = context();
  sqfv::Batch batch;
  ok(c.prepare(flow, m, position(), batch), "prepare capture");
  sqfv::Lease lease;
  require(batch.acquire(c.description().source.access_scope, lease) ==
              sqfv::Status::ok,
          "lease");
  require(lease.descriptor().record_count == 1 &&
              lease.descriptor().batch_sequence == 7,
          "envelope count and transfer cursor");
  sqav::Capture decoded;
  ok(sqav::Capture::from_delivery(lease.payload(), lease.descriptor(), m,
                                  limits, decoded),
     "resolve public delivery");
  require(decoded.description().source_position == "provider-position:900",
          "provider position remains distinct");
  auto bad = lease.descriptor();
  bad.source_binding = "forged-source";
  require(
      sqav::Capture::from_delivery(lease.payload(), bad, m, limits, decoded) ==
          sqav::Status::binding_mismatch,
      "source binding reject");
  bad = lease.descriptor();
  bad.record_count = 2;
  require(
      sqav::Capture::from_delivery(lease.payload(), bad, m, limits, decoded) ==
          sqav::Status::binding_mismatch,
      "wrong envelope count");
  auto md = m.description();
  md.evidence.back().evidence_ref = "different-source";
  sqmv::Manifest wrong;
  require(sqmv::Manifest::create(md, {65536, 4096, 128}, wrong) ==
              sqmv::Status::ok,
          "wrong source manifest");
  const auto id = batch.content_id();
  require(c.prepare(flow, wrong, position(), batch) ==
              sqav::Status::binding_mismatch,
          "source evidence required");
  require(batch.content_id() == id, "prepare failure preserves batch");
  for (auto member :
       {&sqav::Source::operation, &sqav::Source::interface_version,
        &sqav::Source::adapter_version, &sqav::Source::dataset_revision,
        &sqav::Source::native_schema_ref, &sqav::Source::access_scope}) {
    auto d = description();
    d.source.*member += "-changed";
    sqav::Capture changed;
    ok(sqav::Capture::create(d, raw(), limits, changed), "changed source");
    require(changed.source_reference() != c.source_reference(),
            "source fields independently bound");
    require(changed.prepare(flow, m, position(), batch) ==
                sqav::Status::binding_mismatch,
            "old manifest cannot launder changed source");
  }
}
void retained_delivery_composition() {
  char temp[] = "/private/tmp/sqav-composition-XXXXXX";
  auto path = ::mkdtemp(temp);
  require(path, "private fixture root");
  {
    auto c = capture();
    auto m = metadata(c);
    auto flow = context();
    auto pos = position();
    sqpv::Options options{
        pos.partition, pos.producer_generation, {}, 7, {2U << 20, 8U << 20, 8}};
    options.store_generation[0] = 2;
    sqdv::RetainedSource source;
    require(sqdv::RetainedSource::create(path, m, options, source) ==
                sqdv::Status::ok,
            "retained source");
    sqfv::Batch b;
    ok(c.prepare(flow, m, pos, b), "capture flow");
    sqdv::RetainedBatch proof;
    require(source.commit(flow, b, proof) == sqdv::Status::ok,
            "capture retained");
    sqdv::Config cfg{"view:capture",
                     "recipient:offline",
                     "interface:original-capture",
                     pos.partition,
                     pos.producer_generation,
                     7,
                     sqdv::Profile::retained_before_delivery};
    sqdv::Session s;
    require(sqdv::Session::create(flow, m, cfg, {65536, 2}, &source, nullptr,
                                  s) == sqdv::Status::ok,
            "delivery session");
    require(s.offer_next(flow) == sqdv::Status::ok, "read retained capture");
    sqdv::Delivery delivered;
    require(s.take(delivered) == sqdv::Status::ok, "delivery take");
    sqav::Capture recovered;
    ok(sqav::Capture::from_delivery(delivered.payload(), delivered.descriptor(),
                                    m, limits, recovered),
       "capture evidence recovered");
    bytes_equal(recovered.original(), raw(), "five-owner original bytes");
    require(recovered.reference() == c.reference(),
            "five-owner capture identity");
    sqdv::Checkpoint checkpoint;
    require(s.checkpoint(checkpoint) == sqdv::Status::ok &&
                checkpoint.next_sequence == 7,
            "unacknowledged capture replayable");
    s.reset();
    source.reset();
    require(sqdv::RetainedSource::open(path, m, options, source) ==
                sqdv::Status::ok,
            "reopen source");
    require(sqdv::Session::create(flow, m, cfg, {65536, 2}, &source,
                                  &checkpoint, s) == sqdv::Status::ok,
            "resume");
    require(s.offer_next(flow) == sqdv::Status::ok, "replay");
    require(s.take(delivered) == sqdv::Status::ok, "replace held delivery");
    ok(sqav::Capture::from_delivery(delivered.payload(), delivered.descriptor(),
                                    m, limits, recovered),
       "replayed source meaning");
    require(s.acknowledge_processed(delivered) == sqdv::Status::ok,
            "ack processing");
  }
  std::filesystem::remove_all(path);
}
template <class F> long sweep(F action, const char *label) {
  for (long n = 0; n < 512; ++n) {
    allocations = 0;
    fail_at = n;
    const auto result = action();
    fail_at = -1;
    if (result == sqav::Status::ok) {
      std::printf("sqav allocation rollback: %s %ld failures\n", label, n);
      return n;
    }
    require(result == sqav::Status::no_memory,
            "injected failure classification");
  }
  require(false, "allocation sweep finite");
  return 0;
}
void allocation_rollback() {
  auto d = description();
  auto bytes = raw();
  auto c = capture();
  auto out = capture();
  const auto before = std::string(out.reference());
  auto total = sweep(
      [&] {
        auto s = sqav::Capture::create(d, bytes, limits, out);
        if (s != sqav::Status::ok)
          require(out.reference() == before, "create output");
        return s;
      },
      "create");
  total += sweep(
      [&] {
        auto s =
            sqav::Capture::resolve(c.encoded(), c.reference(), limits, out);
        if (s != sqav::Status::ok)
          require(out.reference() == before, "resolve output");
        return s;
      },
      "resolve");
  auto m = metadata(c);
  auto flow = context();
  auto p = position();
  sqfv::Batch b;
  ok(c.prepare(flow, m, p, b), "baseline batch");
  const auto id = b.content_id();
  sqfv::ContextStats old_stats;
  require(flow.stats(old_stats) == sqfv::Status::ok, "baseline stats");
  total += sweep(
      [&] {
        auto s = c.prepare(flow, m, p, b);
        if (s != sqav::Status::ok) {
          require(b.content_id() == id, "prepare output");
          sqfv::ContextStats now;
          require(flow.stats(now) == sqfv::Status::ok &&
                      now.allocation_bytes == old_stats.allocation_bytes,
                  "flow reservations rolled back");
        }
        return s;
      },
      "prepare");
  sqfv::Lease lease;
  require(b.acquire(d.source.access_scope, lease) == sqfv::Status::ok, "lease");
  total += sweep(
      [&] {
        auto s = sqav::Capture::from_delivery(
            lease.payload(), lease.descriptor(), m, limits, out);
        if (s != sqav::Status::ok)
          require(out.reference() == before, "delivery output");
        return s;
      },
      "from-delivery");
  allocations = 0;
  fail_at = 0;
  auto s = c.retain(out);
  fail_at = -1;
  ok(s, "retain allocation-free");
  require(total > 40, "meaningful allocation sweep");
}
} // namespace
int main() {
  exact_capture_roundtrip();
  malformed_and_limits();
  binding_and_original_identity();
  retained_delivery_composition();
  allocation_rollback();
  std::puts("sqav native acceptance: 5 groups passed");
}
