#include <symphony/sqmv/metadata.hpp>
#include <symphony/knowledge/engine/digest.hpp>

#include <algorithm>
#include <atomic>
#include <cstdio>
#include <cstdlib>
#include <new>
#include <utility>

namespace {
std::atomic<int> fail_at{-1};
std::atomic<int> allocation_calls{0};
std::atomic<std::size_t> live_allocations{0};
}

void* operator new(std::size_t size) {
  const auto selected = fail_at.load();
  if (selected >= 0 && allocation_calls.fetch_add(1) == selected) throw std::bad_alloc();
  if (void* result = std::malloc(size == 0 ? 1 : size)) {
    ++live_allocations;
    return result;
  }
  throw std::bad_alloc();
}
void* operator new[](std::size_t size) { return ::operator new(size); }
void operator delete(void* pointer) noexcept {
  if (pointer) --live_allocations;
  std::free(pointer);
}
void operator delete[](void* pointer) noexcept { ::operator delete(pointer); }

namespace {
using namespace symphony::sqmv;

void require(bool value, const char* message) {
  if (value) return;
  std::fprintf(stderr, "sqmv metadata test: %s\n", message);
  std::abort();
}

Limits limits() { return {65'536, 4'096, 128}; }

Description description() {
  return {"dataset:observations", "revision:1", "schema:1", "layout:1",
          "private:test", "producer:test",
          {{EvidenceRole::schema, "producer:test", "evidence:schema:1"},
           {EvidenceRole::layout, "producer:test", "evidence:layout:1"},
           {EvidenceRole::access, "producer:test", "evidence:access:1"}}};
}

Manifest create(const Description& input) {
  Manifest result;
  require(Manifest::create(input, limits(), result) == Status::ok, "fixture creation");
  return result;
}

std::vector<std::uint8_t> bytes(const Manifest& value) {
  return {value.encoded().begin(), value.encoded().end()};
}

std::uint16_t u16(const std::vector<std::uint8_t>& value, std::size_t offset) {
  return static_cast<std::uint16_t>((value[offset] << 8) | value[offset + 1]);
}

std::size_t evidence_start(const std::vector<std::uint8_t>& value) {
  std::size_t cursor = 16;
  for (unsigned index = 0; index < 6; ++index) cursor += 2 + u16(value, cursor);
  return cursor + 2;
}

std::size_t evidence_end(const std::vector<std::uint8_t>& value, std::size_t offset) {
  auto cursor = offset + 1;
  cursor += 2 + u16(value, cursor);
  return cursor + 2 + u16(value, cursor);
}

// These mutations deliberately have internally valid SHA-256 digests. Their
// rejection must result from semantic/canonical parsing, not stale checksums.
std::string reseal(std::vector<std::uint8_t>& value) {
  constexpr char domain[] = "symphony.sqmv.metadata-manifest.v1";
  std::string input(domain, sizeof(domain));
  input.append(reinterpret_cast<const char*>(value.data()), value.size() - 32);
  const auto digest = symphony::knowledge::engine::sha256_hex(input);
  const auto nibble = [](char c) { return c <= '9' ? c - '0' : c - 'a' + 10; };
  for (unsigned index = 0; index < 32; ++index)
    value[value.size() - 32 + index] = static_cast<std::uint8_t>((nibble(digest[index * 2]) << 4) | nibble(digest[index * 2 + 1]));
  return "sqmv1-sha256-" + digest;
}

void canonical_round_trip() {
  const auto input = description();
  auto manifest = create(input);
  // Independently generated using Python hashlib and a separate wire encoder.
  require(manifest.reference() == "sqmv1-sha256-eb9a1cf0a64741137382a63471f13003c8c68f24f66e344b9c11d3e30a65001f", "independent golden digest");
  require(manifest.encoded().size() == 238, "independent golden length");
  Manifest decoded;
  require(Manifest::resolve(manifest.encoded(), manifest.reference(), limits(), decoded) == Status::ok, "round trip");
  require(decoded.reference() == manifest.reference() && bytes(decoded) == bytes(manifest), "exact canonical bytes");
  auto reordered = input;
  std::reverse(reordered.evidence.begin(), reordered.evidence.end());
  auto second = create(reordered);
  require(second.reference() == manifest.reference(), "evidence order canonicalized");
  auto binary = input;
  binary.dataset_id = std::string("a\0\xff", 3);
  binary.evidence.push_back({EvidenceRole::time, std::string("\x80\0", 2), std::string("\0\xff", 2)});
  auto binary_manifest = create(binary);
  require(Manifest::resolve(binary_manifest.encoded(), binary_manifest.reference(), limits(), decoded) == Status::ok, "binary round trip");
  require(decoded.description().dataset_id == binary.dataset_id && decoded.description().evidence.back().evidence_ref == binary.evidence.back().evidence_ref, "raw bytes preserved");
  for (std::size_t length : {1U, 55U, 56U, 63U, 64U, 65U, 127U, 128U}) {
    auto boundary = input;
    boundary.dataset_id.assign(length, 'x');
    auto selected = create(boundary);
    require(Manifest::resolve(selected.encoded(), selected.reference(), limits(), decoded) == Status::ok, "digest block boundary");
  }
}

void noncanonical_rejection() {
  auto manifest = create(description());
  auto output = create(description());
  const std::string original(output.reference());
  const auto expect = [&](std::vector<std::uint8_t> input, Status wanted) {
    const auto reference = reseal(input);
    require(Manifest::resolve(input, reference, limits(), output) == wanted, "digest-valid rejection status");
    require(output.reference() == original, "rejection preserves existing manifest");
  };
  auto reordered = bytes(manifest);
  const auto first = evidence_start(reordered);
  const auto second = evidence_end(reordered, first);
  const auto third = evidence_end(reordered, second);
  std::rotate(reordered.begin() + static_cast<std::ptrdiff_t>(first),
              reordered.begin() + static_cast<std::ptrdiff_t>(second),
              reordered.begin() + static_cast<std::ptrdiff_t>(third));
  expect(reordered, Status::corrupt_manifest);
  auto duplicated = bytes(manifest);
  std::copy(duplicated.begin() + static_cast<std::ptrdiff_t>(second),
            duplicated.begin() + static_cast<std::ptrdiff_t>(third),
            duplicated.begin() + static_cast<std::ptrdiff_t>(third));
  expect(duplicated, Status::corrupt_manifest);
  auto missing = bytes(manifest);
  missing[evidence_end(missing, evidence_end(missing, first))] = 4;
  expect(missing, Status::corrupt_manifest);
  auto unknown = bytes(manifest);
  unknown[first] = 9;
  expect(unknown, Status::unsupported_manifest);
  auto empty_field = bytes(manifest);
  empty_field[16] = empty_field[17] = 0;
  expect(empty_field, Status::corrupt_manifest);
  for (std::size_t length = 0; length < manifest.encoded().size(); ++length) {
    require(Manifest::resolve(manifest.encoded().first(length), manifest.reference(), limits(), output) != Status::ok, "all truncations reject");
    require(output.reference() == original, "truncation output unchanged");
  }
  auto trailing = bytes(manifest);
  trailing.push_back(0);
  require(Manifest::resolve(trailing, manifest.reference(), limits(), output) == Status::corrupt_manifest, "trailing bytes reject");
  auto corrupt = bytes(manifest);
  corrupt[corrupt.size() - 1] ^= 1;
  require(Manifest::resolve(corrupt, manifest.reference(), limits(), output) == Status::corrupt_manifest, "corrupt digest reject");
  for (const std::size_t offset : {4U, 6U, 8U}) {
    auto unsupported = bytes(manifest);
    unsupported[offset] = 1;
    require(Manifest::resolve(unsupported, manifest.reference(), limits(), output) == Status::unsupported_manifest, "unsupported version or flags reject");
  }
  auto duplicate = description();
  duplicate.evidence.push_back(duplicate.evidence[0]);
  require(Manifest::create(duplicate, limits(), output) == Status::invalid_argument, "duplicate evidence reject");
  duplicate.evidence.back().evidence_ref = "distinct-schema-evidence";
  require(Manifest::create(duplicate, limits(), output) == Status::invalid_argument, "multiple required roles reject");
}

void exact_reference_and_binding() {
  auto manifest = create(description());
  auto changed = description();
  changed.dataset_id += ":other";
  auto other = create(changed);
  Manifest output;
  require(Manifest::resolve(manifest.encoded(), other.reference(), limits(), output) == Status::reference_mismatch && !output, "expected digest is mandatory");
  require(Manifest::resolve(manifest.encoded(), "latest", limits(), output) == Status::invalid_argument, "latest is not a reference");
  symphony::sqfv::Binding binding;
  require(manifest.binding(binding) == Status::ok && manifest.verify_binding(binding) == Status::ok, "derived binding");
  require(other.verify_binding(binding) == Status::binding_mismatch, "dataset lineage bound by digest");
  for (unsigned field = 0; field < 5; ++field) {
    auto wrong = binding;
    switch (field) {
      case 0: wrong.metadata_ref += 'x'; break;
      case 1: wrong.dataset_revision += 'x'; break;
      case 2: wrong.schema_version += 'x'; break;
      case 3: wrong.layout_version += 'x'; break;
      case 4: wrong.access_scope += 'x'; break;
    }
    require(manifest.verify_binding(wrong) == Status::binding_mismatch, "every binding field checked");
  }
  for (unsigned field = 0; field < 8; ++field) {
    auto modified = description();
    switch (field) {
      case 0: modified.dataset_id += 'x'; break;
      case 1: modified.dataset_revision += 'x'; break;
      case 2: modified.schema_version += 'x'; break;
      case 3: modified.layout_version += 'x'; break;
      case 4: modified.access_scope += 'x'; break;
      case 5: modified.producer_ref += 'x'; break;
      case 6: modified.evidence[0].producer_ref += 'x'; break;
      case 7: modified.evidence[0].evidence_ref += 'x'; break;
    }
    require(create(modified).reference() != manifest.reference(), "every semantic field affects reference");
  }
}

void immutable_retention() {
  auto input = description();
  auto manifest = create(input);
  const std::string reference(manifest.reference());
  input.dataset_revision = "changed";
  input.evidence.clear();
  Manifest retained;
  require(manifest.retain(retained) == Status::ok, "retained manifest");
  manifest = Manifest{};
  require(retained.reference() == reference && retained.description().dataset_revision == "revision:1", "retained immutable bytes");
  require(retained.retain(retained) == Status::ok, "self retain");
  require(Manifest::resolve(retained.encoded(), retained.reference(), limits(), retained) == Status::ok, "aliased resolve output");
  require(Manifest::create(retained.description(), limits(), retained) == Status::ok, "aliased create output");
  require(retained.reference() == reference, "aliased outputs exact");
  Manifest empty;
  require(empty.encoded().empty() && empty.reference().empty(), "empty safe views");
  require(empty.retain(retained) == Status::invalid_argument && retained.reference() == reference, "empty retain preserves output");
  symphony::sqfv::Binding binding;
  require(empty.binding(binding) == Status::invalid_argument && empty.verify_binding(binding) == Status::invalid_argument, "empty binding rejected");
}

void finite_limits() {
  auto input = description();
  auto manifest = create(input);
  Manifest output;
  auto exact = limits();
  exact.max_manifest_bytes = static_cast<std::uint32_t>(manifest.encoded().size());
  require(Manifest::create(input, exact, output) == Status::ok, "exact encoded bound");
  --exact.max_manifest_bytes;
  require(Manifest::create(input, exact, output) == Status::limit, "encoded bound plus one");
  require(Manifest::resolve(manifest.encoded(), manifest.reference(), exact, output) == Status::limit, "decode encoded bound");
  for (unsigned field = 0; field < 3; ++field) {
    auto invalid = limits();
    if (field == 0) invalid.max_manifest_bytes = 0;
    if (field == 1) invalid.max_field_bytes = 0;
    if (field == 2) invalid.max_evidence_refs = 0;
    require(Manifest::create(input, invalid, output) == Status::invalid_argument, "zero configuration rejected");
  }
  input.dataset_id.assign(4096, 'x');
  require(Manifest::create(input, limits(), output) == Status::ok, "exact string ceiling");
  input.dataset_id.push_back('x');
  require(Manifest::create(input, limits(), output) == Status::limit, "string ceiling plus one");
  input = description();
  for (unsigned index = 3; index < 128; ++index)
    input.evidence.push_back({EvidenceRole::source, "producer", std::to_string(index)});
  require(Manifest::create(input, limits(), output) == Status::ok, "exact evidence ceiling");
  input.evidence.push_back({EvidenceRole::source, "producer", "extra"});
  require(Manifest::create(input, limits(), output) == Status::limit, "evidence ceiling plus one");
  input = description();
  for (unsigned index = 0; index < 8; ++index) {
    std::string producer(4096, 'p');
    producer.back() = static_cast<char>('a' + index);
    input.evidence.push_back({EvidenceRole::source, std::move(producer),
        std::string(index == 7 ? 3818 : 4096, 'e')});
  }
  require(Manifest::create(input, limits(), output) == Status::ok && output.encoded().size() == 65536, "exact technical encoded ceiling");
  auto smaller_fields = limits();
  smaller_fields.max_field_bytes = 4095;
  Manifest decoded;
  require(Manifest::resolve(output.encoded(), output.reference(), smaller_fields, decoded) == Status::limit, "decode exact string limit");
  require(Manifest::resolve(output.encoded(), output.reference(), limits(), decoded) == Status::ok, "decode exact technical ceiling");
  input.evidence.back().evidence_ref.push_back('e');
  require(Manifest::create(input, limits(), output) == Status::limit, "technical encoded ceiling plus one");
}

void allocation_rollback() {
  auto input = description();
  input.dataset_id.assign(80, 'd');
  input.dataset_revision.assign(80, 'r');
  input.schema_version.assign(80, 's');
  input.layout_version.assign(80, 'l');
  input.access_scope.assign(80, 'a');
  input.producer_ref.assign(80, 'p');
  for (auto& item : input.evidence) {
    item.producer_ref.assign(80, 'p');
    item.evidence_ref.assign(80, 'e');
  }
  const auto source = create(input);
  unsigned rejections = 0;
  for (unsigned operation = 0; operation < 3; ++operation) {
    bool success = false;
    for (int selected = 0; selected < 128; ++selected) {
      auto output = create(description());
      const std::string before(output.reference());
      symphony::sqfv::Binding binding{"old-metadata", "old-revision", "old-schema", "old-layout", "old-scope"};
      const auto live_before = live_allocations.load();
      allocation_calls = 0;
      fail_at = selected;
      const auto status = operation == 0 ? Manifest::create(input, limits(), output) :
          operation == 1 ? Manifest::resolve(source.encoded(), source.reference(), limits(), output) : source.binding(binding);
      fail_at = -1;
      if (status == Status::ok) { success = true; break; }
      ++rejections;
      if (status != Status::no_memory)
        std::fprintf(stderr, "operation %u allocation %d returned status %u\n",
                     operation, selected, static_cast<unsigned>(status));
      require(status == Status::no_memory, "allocation failure contained");
      require(live_allocations.load() == live_before, "partial allocations rolled back");
      require(output.reference() == before, "allocation failure preserves manifest");
      require(binding.metadata_ref == "old-metadata" && binding.dataset_revision == "old-revision" &&
          binding.schema_version == "old-schema" && binding.layout_version == "old-layout" &&
          binding.access_scope == "old-scope", "allocation failure preserves whole binding");
    }
    require(success, "all allocation failure points traversed");
  }
  Manifest retained;
  fail_at = 0;
  allocation_calls = 0;
  const auto retained_status = source.retain(retained);
  retained = Manifest{};
  fail_at = -1;
  require(retained_status == Status::ok && allocation_calls == 0, "retain and release allocate nothing");
  std::printf("sqmv allocation rollback: %u rejected allocation points\n", rejections);
}
}

int main() {
  const auto initial = live_allocations.load();
  canonical_round_trip();
  noncanonical_rejection();
  exact_reference_and_binding();
  immutable_retention();
  finite_limits();
  allocation_rollback();
  require(live_allocations.load() == initial, "all owned metadata allocations released");
  std::puts("sqmv metadata tests passed");
}
