#include <algorithm>
#include <array>
#include <atomic>
#include <bit>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <limits>
#include <new>
#include <symphony/sqdv/delivery.hpp>
#include <symphony/sqtv/integer_conversion.hpp>
#include <unistd.h>
#include <utility>
#include <variant>
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
using Value = std::variant<std::int64_t, std::uint64_t>;
constexpr sqtv::Limits limits{4096, 32768, 32768};
void require(bool ok, const char *msg) {
  if (!ok) {
    std::fprintf(stderr, "sqtv: %s\n", msg);
    std::abort();
  }
}
std::vector<sqtv::Format> formats() {
  std::vector<sqtv::Format> all;
  for (auto bits : {8, 16, 32, 64})
    for (auto sign :
         {sqtv::Signedness::signed_integer, sqtv::Signedness::unsigned_integer})
      for (auto order : {sqtv::ByteOrder::little, sqtv::ByteOrder::big})
        all.push_back({static_cast<std::uint8_t>(bits), sign, order});
  return all;
}
bool fits(Value v, sqtv::Format f) {
  return std::visit(
      [&](auto n) {
        if (f.signedness == sqtv::Signedness::signed_integer) {
          switch (f.bits) {
          case 8:
            return std::in_range<std::int8_t>(n);
          case 16:
            return std::in_range<std::int16_t>(n);
          case 32:
            return std::in_range<std::int32_t>(n);
          default:
            return std::in_range<std::int64_t>(n);
          }
        }
        switch (f.bits) {
        case 8:
          return std::in_range<std::uint8_t>(n);
        case 16:
          return std::in_range<std::uint16_t>(n);
        case 32:
          return std::in_range<std::uint32_t>(n);
        default:
          return std::in_range<std::uint64_t>(n);
        }
      },
      v);
}
// The fixture oracle uses native integer object representation and byte
// reversal, independently of the runtime's shift-based reader and
// signed-magnitude checks.
std::vector<std::uint8_t> encode(Value v, sqtv::Format f) {
  auto bytes = std::visit(
      [](auto n) { return std::bit_cast<std::array<std::uint8_t, 8>>(n); }, v);
  if constexpr (std::endian::native == std::endian::big)
    std::reverse(bytes.begin(), bytes.end());
  std::vector<std::uint8_t> out(bytes.begin(), bytes.begin() + f.bits / 8);
  if (f.order == sqtv::ByteOrder::big)
    std::reverse(out.begin(), out.end());
  return out;
}
sqfv::Context context() {
  sqfv::Context c;
  require(sqfv::Context::create({65536, 73728, 4096, 8U << 20, 8}, c) ==
              sqfv::Status::ok,
          "context");
  return c;
}
sqmv::Manifest metadata(sqtv::Format f) {
  std::string layout;
  require(sqtv::layout_id(f, layout) == sqtv::Status::ok, "layout");
  sqmv::Description d{
      "dataset:integers",
      "revision:fixture",
      std::string(sqtv::integer_schema),
      layout,
      "private:fixture",
      "producer:fixture",
      {{sqmv::EvidenceRole::schema, "producer:fixture",
        "schema:integer-values"},
       {sqmv::EvidenceRole::layout, "producer:fixture", layout},
       {sqmv::EvidenceRole::access, "producer:fixture", "access:fixture"},
       {sqmv::EvidenceRole::source, "provider:fixture", "evidence:source"},
       {sqmv::EvidenceRole::units, "provider:fixture",
        "units:unscaled-integers"},
       {sqmv::EvidenceRole::time, "provider:fixture", "time:original-date"}}};
  sqmv::Manifest m;
  require(sqmv::Manifest::create(d, {65536, 4096, 128}, m) == sqmv::Status::ok,
          "metadata");
  return m;
}
sqtv::Position position() {
  sqtv::Position p{"partition:derived", {}, 100};
  p.producer_generation[0] = 2;
  return p;
}
sqfv::Batch input(sqfv::Context &c, const sqmv::Manifest &m, sqfv::ByteView raw,
                  std::uint64_t count = 1) {
  sqfv::Descriptor d;
  require(m.binding(d.binding) == sqmv::Status::ok, "binding");
  d.partition = "partition:original";
  d.producer_generation[0] = 1;
  d.batch_sequence = 7;
  d.record_count = count;
  d.source_binding = "source:fixture";
  d.source_position = "provider-position:91";
  sqfv::Batch b;
  require(c.prepare_copy(d, raw, b) == sqfv::Status::ok, "input");
  return b;
}
std::vector<std::uint8_t> payload(const sqfv::Batch &b) {
  sqfv::Lease l;
  require(b.acquire("private:fixture", l) == sqfv::Status::ok, "lease");
  return {l.payload().begin(), l.payload().end()};
}
void numeric_matrix_and_roundtrip() {
  auto flow = context();
  auto fs = formats();
  std::vector<Value> values{std::int64_t{INT64_MIN},
                            std::int64_t{-2147483649LL},
                            std::int64_t{-2147483648LL},
                            std::int64_t{-32769},
                            std::int64_t{-32768},
                            std::int64_t{-129},
                            std::int64_t{-128},
                            std::int64_t{-1},
                            std::int64_t{0},
                            std::int64_t{1},
                            std::int64_t{127},
                            std::int64_t{128},
                            std::int64_t{255},
                            std::int64_t{256},
                            std::int64_t{32767},
                            std::int64_t{32768},
                            std::int64_t{65535},
                            std::int64_t{65536},
                            std::int64_t{INT32_MAX},
                            std::uint64_t{2147483648ULL},
                            std::uint64_t{UINT32_MAX},
                            std::uint64_t{4294967296ULL},
                            std::int64_t{INT64_MAX},
                            std::uint64_t{9223372036854775808ULL},
                            std::uint64_t{UINT64_MAX}};
  unsigned accepted = 0, rejected = 0;
  for (auto from : fs) {
    auto m = metadata(from);
    for (auto v : values) {
      if (!fits(v, from))
        continue;
      auto bytes = encode(v, from);
      auto b = input(flow, m, bytes);
      const auto id = b.content_id();
      for (auto to : fs) {
        sqtv::Result out;
        const auto s =
            sqtv::Result::convert(flow, b, m, to, position(), limits, out);
        if (!fits(v, to)) {
          require(s == sqtv::Status::overflow && !out,
                  "overflow is exact failure");
          ++rejected;
          continue;
        }
        require(s == sqtv::Status::ok, "matrix convert");
        require(payload(out.batch()) == encode(v, to),
                "independent numeric oracle");
        sqtv::Result roundtrip;
        require(sqtv::Result::convert(flow, out.batch(), out.metadata(), from,
                                      position(), limits,
                                      roundtrip) == sqtv::Status::ok,
                "roundtrip convert");
        require(payload(roundtrip.batch()) == bytes,
                "roundtrip original bytes");
        require(b.content_id() == id, "input immutable");
        ++accepted;
      }
    }
  }
  std::printf("sqtv numeric matrix: %u accepted and %u overflow rejections "
              "across 256 format pairs\n",
              accepted, rejected);
}
void lineage_and_metadata() {
  auto c = context();
  auto f = formats()[4];
  auto m = metadata(f);
  auto bytes = encode(std::int64_t{-123}, f);
  auto b = input(c, m, bytes);
  sqtv::Result r;
  auto pos = position();
  require(sqtv::Result::convert(c, b, m, formats()[13], pos, limits, r) ==
              sqtv::Status::ok,
          "lineage conversion");
  const auto &d = r.metadata().description();
  require(d.dataset_id == m.description().dataset_id &&
              d.dataset_revision == m.description().dataset_revision &&
              d.access_scope == m.description().access_scope,
          "semantic scope preserved");
  require(d.producer_ref == sqtv::converter_identity, "derived producer");
  for (const auto &e : m.description().evidence) {
    if (e.role == sqmv::EvidenceRole::layout)
      continue;
    require(std::any_of(d.evidence.begin(), d.evidence.end(),
                        [&](const auto &x) {
                          return x.role == e.role &&
                                 x.producer_ref == e.producer_ref &&
                                 x.evidence_ref == e.evidence_ref;
                        }),
            "source units and time evidence preserved");
  }
  for (auto ref : {m.reference(), r.operation_reference(), r.input_reference()})
    require(std::any_of(d.evidence.begin(), d.evidence.end(),
                        [&](const auto &e) {
                          return e.role == sqmv::EvidenceRole::lineage &&
                                 e.producer_ref == sqtv::converter_identity &&
                                 e.evidence_ref == ref;
                        }),
            "exact lineage");
  require(r.batch().descriptor().source_binding == r.operation_reference() &&
              r.batch().descriptor().source_position == r.input_reference() &&
              r.batch().descriptor().batch_sequence == pos.batch_sequence,
          "derived descriptor provenance and position");
  auto id = r.batch().content_id();
  sqtv::Result retry;
  require(sqtv::Result::convert(c, b, m, formats()[13], pos, limits, retry) ==
                  sqtv::Status::ok &&
              retry.batch().content_id() == id,
          "deterministic retry");
  require(sqtv::Result::convert(c, r.batch(), r.metadata(), f, pos, limits,
                                r) == sqtv::Status::ok &&
              payload(r.batch()) == bytes,
          "aliased result reuse");
  auto separate = context();
  sqtv::Result cross;
  require(sqtv::Result::convert(separate, b, m, f, pos, limits, cross) ==
              sqtv::Status::ok,
          "selected output context independent");
}
void rejection_and_bounds() {
  auto c = context();
  const auto f = formats()[6];
  auto m = metadata(f);
  auto b = input(c, m, encode(std::uint64_t{65535}, f));
  sqtv::Result out;
  require(sqtv::Result::convert(c, b, m, f, position(), limits, out) ==
              sqtv::Status::ok,
          "baseline");
  const auto before = out.batch().content_id();
  require(sqtv::Result::convert(c, b, m, formats()[0], position(), limits,
                                out) == sqtv::Status::overflow,
          "narrow overflow");
  auto bad = input(c, m, std::vector<std::uint8_t>{1, 2, 3});
  require(sqtv::Result::convert(c, bad, m, f, position(), limits, out) ==
              sqtv::Status::malformed_input,
          "partial element");
  auto wrongcount = input(c, m, std::vector<std::uint8_t>{1, 2}, 2);
  require(sqtv::Result::convert(c, wrongcount, m, f, position(), limits, out) ==
              sqtv::Status::malformed_input,
          "record count mismatch");
  auto wrong = metadata(formats()[8]);
  require(sqtv::Result::convert(c, b, wrong, f, position(), limits, out) ==
              sqtv::Status::binding_mismatch,
          "metadata binding");
  auto desc = m.description();
  desc.schema_version = "provider-nullable-integers";
  sqmv::Manifest nullable;
  require(sqmv::Manifest::create(desc, {65536, 4096, 128}, nullable) ==
              sqmv::Status::ok,
          "nullable fixture");
  auto nb = input(c, nullable, encode(std::uint64_t{1}, f));
  require(sqtv::Result::convert(c, nb, nullable, f, position(), limits, out) ==
              sqtv::Status::unsupported_representation,
          "unknown nullable semantics reject");
  auto l = limits;
  l.max_output_bytes = 1;
  require(sqtv::Result::convert(c, b, m, f, position(), l, out) ==
              sqtv::Status::limit,
          "output bound");
  l = limits;
  l.max_elements = 0;
  require(sqtv::Result::convert(c, b, m, f, position(), l, out) ==
              sqtv::Status::invalid_argument,
          "mandatory limit");
  require(sqtv::Result::convert(
              c, b, m,
              {24, sqtv::Signedness::signed_integer, sqtv::ByteOrder::little},
              position(), limits,
              out) == sqtv::Status::unsupported_representation,
          "unsupported width");
  require(out.batch().content_id() == before, "all failures preserve result");
  // Overflow in the last element must not publish the converted prefix.
  auto late = input(c, m, std::vector<std::uint8_t>{1, 0, 255, 255}, 2);
  require(sqtv::Result::convert(c, late, m, formats()[0], position(), limits,
                                out) == sqtv::Status::overflow &&
              out.batch().content_id() == before,
          "no partial batch on late overflow");
  desc = m.description();
  for (int i = 0; desc.evidence.size() < 128; ++i)
    desc.evidence.push_back({sqmv::EvidenceRole::lineage, "prior:producer",
                             "prior:lineage-" + std::to_string(i)});
  sqmv::Manifest full;
  require(sqmv::Manifest::create(desc, {65536, 4096, 128}, full) ==
              sqmv::Status::ok,
          "full lineage metadata");
  auto fb = input(c, full, encode(std::uint64_t{1}, f));
  require(sqtv::Result::convert(c, fb, full, f, position(), limits, out) ==
              sqtv::Status::limit,
          "lineage saturation visible");
}
void converted_retention_delivery() {
  char temp[] = "/private/tmp/sqtv-retained-XXXXXX";
  auto path = ::mkdtemp(temp);
  require(path, "private root");
  {
    auto c = context();
    auto f = formats()[4];
    auto m = metadata(f);
    auto b = input(c, m, encode(std::int64_t{-32768}, f));
    sqtv::Result r;
    auto pos = position();
    require(sqtv::Result::convert(c, b, m, formats()[13], pos, limits, r) ==
                sqtv::Status::ok,
            "convert retained fixture");
    sqpv::Options options{pos.partition,
                          pos.producer_generation,
                          {},
                          pos.batch_sequence,
                          {73728, 1U << 20, 8}};
    options.store_generation[0] = 3;
    sqdv::RetainedSource source;
    require(sqdv::RetainedSource::create(path, r.metadata(), options, source) ==
                sqdv::Status::ok,
            "derived store");
    sqdv::RetainedBatch proof;
    require(source.commit(c, r.batch(), proof) == sqdv::Status::ok,
            "retained conversion");
    sqdv::Config cfg{"view:integers",
                     "recipient:fixture",
                     "interface:integer-values",
                     pos.partition,
                     pos.producer_generation,
                     pos.batch_sequence,
                     sqdv::Profile::retained_before_delivery};
    sqdv::Session s;
    require(sqdv::Session::create(c, r.metadata(), cfg, {65536, 2}, &source,
                                  nullptr, s) == sqdv::Status::ok,
            "derived session");
    require(s.offer_next(c) == sqdv::Status::ok, "retained read");
    sqdv::Delivery d;
    require(s.take(d) == sqdv::Status::ok, "take derived");
    require(std::ranges::equal(d.payload(),
                               encode(std::int64_t{-32768}, formats()[13])) &&
                d.descriptor().source_position == r.input_reference() &&
                d.descriptor().source_binding == r.operation_reference(),
            "retained bytes and lineage");
    require(s.acknowledge_processed(d) == sqdv::Status::ok,
            "derived processing");
  }
  std::filesystem::remove_all(path);
}
void allocation_rollback() {
  auto c = context();
  auto f = formats()[4];
  auto m = metadata(f);
  auto b = input(c, m, encode(std::int64_t{-1}, f));
  auto pos = position();
  sqtv::Result out;
  require(sqtv::Result::convert(c, b, m, f, pos, limits, out) ==
              sqtv::Status::ok,
          "baseline allocation");
  auto id = out.batch().content_id();
  sqfv::ContextStats before;
  require(c.stats(before) == sqfv::Status::ok, "baseline reservations");
  const auto target = formats()[13];
  long failures = 0;
  for (long n = 0; n < 512; ++n) {
    allocations = 0;
    fail_at = n;
    const auto result =
        sqtv::Result::convert(c, b, m, target, pos, limits, out);
    fail_at = -1;
    if (result == sqtv::Status::ok) {
      failures = n;
      break;
    }
    require(result == sqtv::Status::no_memory && out.batch().content_id() == id,
            "allocation failure atomic result");
    sqfv::ContextStats now;
    require(c.stats(now) == sqfv::Status::ok &&
                now.allocation_bytes == before.allocation_bytes,
            "temporary reservations returned");
  }
  require(failures > 20, "allocation path exercised");
  std::printf("sqtv allocation rollback: %ld failures\n", failures);
}
} // namespace
int main() {
  numeric_matrix_and_roundtrip();
  lineage_and_metadata();
  rejection_and_bounds();
  converted_retention_delivery();
  allocation_rollback();
  std::puts("sqtv native acceptance: 5 groups passed");
}
