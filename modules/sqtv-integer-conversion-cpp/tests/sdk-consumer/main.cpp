#include <algorithm>
#include <array>
#include <cstdio>
#include <cstdlib>
#include <symphony/sqtv/integer_conversion.hpp>
using namespace symphony;
void check(bool condition) {
  if (!condition)
    std::abort();
}
void installed_conversion_and_rejection() {
  const sqtv::Format from{16, sqtv::Signedness::signed_integer,
                          sqtv::ByteOrder::little};
  const sqtv::Format to{64, sqtv::Signedness::signed_integer,
                        sqtv::ByteOrder::big};
  std::string layout, operation;
  check(sqtv::layout_id(from, layout) == sqtv::Status::ok);
  check(sqtv::operation_reference(from, to, operation) == sqtv::Status::ok);
  // Independently encoded from SPEC with Python hashlib, not runtime output.
  check(operation ==
        "sqtv-int1-sha256-"
        "6091e5951692232fea3b8435618562a0714a9a21f5f9320cb14fd276f501b6d5");
  sqtv::Format parsed;
  check(sqtv::parse_layout(layout, parsed) == sqtv::Status::ok &&
        parsed == from);
  sqmv::Description description{
      "fixture",
      "v1",
      std::string(sqtv::integer_schema),
      layout,
      "private",
      "fixture",
      {{sqmv::EvidenceRole::schema, "fixture", "integers"},
       {sqmv::EvidenceRole::layout, "fixture", layout},
       {sqmv::EvidenceRole::access, "fixture", "private"},
       {sqmv::EvidenceRole::source, "provider", "original-source"}}};
  sqmv::Manifest metadata;
  check(sqmv::Manifest::create(description, {65536, 4096, 128}, metadata) ==
        sqmv::Status::ok);
  sqfv::Context flow;
  check(sqfv::Context::create({65536, 73728, 4096, 1U << 20, 4}, flow) ==
        sqfv::Status::ok);
  sqfv::Descriptor descriptor;
  check(metadata.binding(descriptor.binding) == sqmv::Status::ok);
  descriptor.partition = "input";
  descriptor.producer_generation[0] = 1;
  descriptor.batch_sequence = 1;
  descriptor.record_count = 2;
  descriptor.source_binding = "provider";
  descriptor.source_position = "99";
  const std::array<std::uint8_t, 4> original{0, 128, 255, 127}; // -32768, 32767
  sqfv::Batch batch;
  check(flow.prepare_copy(descriptor, original, batch) == sqfv::Status::ok);
  sqtv::Position position{"output", {}, 5};
  position.producer_generation[0] = 2;
  sqtv::Result result;
  check(sqtv::Result::convert(flow, batch, metadata, to, position,
                              {32, 256, 256}, result) == sqtv::Status::ok);
  sqfv::Lease lease;
  check(result.batch().acquire("private", lease) == sqfv::Status::ok);
  const std::array<std::uint8_t, 16> expected{
      255, 255, 255, 255, 255, 255, 128, 0, 0, 0, 0, 0, 0, 0, 127, 255};
  check(std::ranges::equal(lease.payload(), expected));
  check(result.operation_reference() == operation &&
        result.batch().descriptor().source_binding == operation);
  check(result.metadata().description().access_scope == "private");
  check(std::ranges::any_of(result.metadata().description().evidence,
                            [&](const auto &e) {
                              return e.role == sqmv::EvidenceRole::lineage &&
                                     e.evidence_ref == metadata.reference();
                            }));
  const auto id = result.batch().content_id();
  check(sqtv::Result::convert(
            flow, batch, metadata,
            {8, sqtv::Signedness::signed_integer, sqtv::ByteOrder::little},
            position, {32, 256, 256}, result) == sqtv::Status::overflow);
  check(result.batch().content_id() == id);
  std::puts("sqtv installed consumer: identity, conversion/lineage and atomic "
            "rejection passed");
}

int main() { installed_conversion_and_rejection(); }
