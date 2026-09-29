#include <symphony/sqmv/metadata.hpp>
#include <symphony/knowledge/engine/digest.hpp>

#include <algorithm>
#include <array>
#include <cstring>
#include <new>
#include <stdexcept>
#include <utility>

namespace symphony::sqmv {
namespace detail {
struct ManifestState {
  Description description;
  std::vector<std::uint8_t> encoded;
  std::string reference;
};
}
namespace {
constexpr std::size_t prefix_bytes = 16;
constexpr std::size_t digest_bytes = 32;
constexpr std::size_t max_manifest_bytes = 65'536;
constexpr std::size_t max_field_bytes = 4'096;
constexpr std::size_t max_evidence_refs = 128;
constexpr char digest_domain[] = "symphony.sqmv.metadata-manifest.v1";
constexpr std::string_view reference_prefix = "sqmv1-sha256-";
constexpr char hex[] = "0123456789abcdef";

struct EvidenceView {
  EvidenceRole role;
  std::string_view producer;
  std::string_view reference;
};

bool valid_limits(const Limits& limits) noexcept {
  return limits.max_manifest_bytes > 0 &&
         limits.max_manifest_bytes <= max_manifest_bytes &&
         limits.max_field_bytes > 0 &&
         limits.max_field_bytes <= max_field_bytes &&
         limits.max_evidence_refs > 0 &&
         limits.max_evidence_refs <= max_evidence_refs;
}

bool known_role(EvidenceRole role) noexcept {
  const auto value = static_cast<std::uint8_t>(role);
  return value >= 1 && value <= 8;
}

int compare_bytes(std::string_view left, std::string_view right) noexcept {
  const auto common = std::min(left.size(), right.size());
  const auto compared = common == 0 ? 0 : std::memcmp(left.data(), right.data(), common);
  if (compared != 0) return compared;
  return left.size() < right.size() ? -1 : left.size() > right.size() ? 1 : 0;
}

int compare_evidence(EvidenceView left, EvidenceView right) noexcept {
  if (left.role != right.role)
    return static_cast<std::uint8_t>(left.role) < static_cast<std::uint8_t>(right.role) ? -1 : 1;
  if (const auto comparison = compare_bytes(left.producer, right.producer); comparison != 0)
    return comparison;
  return compare_bytes(left.reference, right.reference);
}

EvidenceView view(const EvidenceReference& value) noexcept {
  return {value.role, value.producer_ref, value.evidence_ref};
}

std::array<std::string_view, 6> fields(const Description& description) noexcept {
  return {description.dataset_id, description.dataset_revision,
          description.schema_version, description.layout_version,
          description.access_scope, description.producer_ref};
}

Status measure(const Description& description, const Limits& limits,
               std::size_t& encoded_size) noexcept {
  if (!valid_limits(limits)) return Status::invalid_argument;
  if (description.evidence.size() > limits.max_evidence_refs) return Status::limit;
  std::size_t size = prefix_bytes + digest_bytes + 2;
  for (const auto field : fields(description)) {
    if (field.empty()) return Status::invalid_argument;
    if (field.size() > limits.max_field_bytes) return Status::limit;
    size += 2 + field.size();
  }
  std::array<unsigned, 3> required{};
  for (std::size_t index = 0; index < description.evidence.size(); ++index) {
    const auto& evidence = description.evidence[index];
    if (!known_role(evidence.role) || evidence.producer_ref.empty() || evidence.evidence_ref.empty())
      return Status::invalid_argument;
    if (evidence.producer_ref.size() > limits.max_field_bytes ||
        evidence.evidence_ref.size() > limits.max_field_bytes) return Status::limit;
    const auto role = static_cast<std::uint8_t>(evidence.role);
    if (role <= 3) ++required[role - 1];
    for (std::size_t previous = 0; previous < index; ++previous)
      if (compare_evidence(view(evidence), view(description.evidence[previous])) == 0)
        return Status::invalid_argument;
    size += 5 + evidence.producer_ref.size() + evidence.evidence_ref.size();
  }
  if (std::any_of(required.begin(), required.end(), [](unsigned count) { return count != 1; }))
    return Status::invalid_argument;
  if (size > limits.max_manifest_bytes) return Status::limit;
  encoded_size = size;
  return Status::ok;
}

template <typename Integer>
void write_be(std::uint8_t*& output, Integer value) noexcept {
  for (std::size_t index = 0; index < sizeof(Integer); ++index)
    *output++ = static_cast<std::uint8_t>(value >> ((sizeof(Integer) - 1 - index) * 8));
}

void write_string(std::uint8_t*& output, std::string_view value) noexcept {
  write_be(output, static_cast<std::uint16_t>(value.size()));
  std::memcpy(output, value.data(), value.size());
  output += value.size();
}

template <typename Integer>
Integer read_be(const std::uint8_t*& input) noexcept {
  Integer result = 0;
  for (std::size_t index = 0; index < sizeof(Integer); ++index)
    result = static_cast<Integer>((result << 8) | *input++);
  return result;
}

std::string digest_hex(ByteView canonical) {
  std::string input(digest_domain, sizeof(digest_domain));
  input.append(reinterpret_cast<const char*>(canonical.data()), canonical.size());
  auto digest = knowledge::engine::sha256_hex(input);
  // Validate the helper's exact-length postcondition before indexing or
  // publishing its result. Allocation failures propagate from the helper.
  if (digest.size() != 64) throw std::runtime_error("incomplete SHA-256 digest");
  return digest;
}

unsigned hex_value(char value) noexcept {
  return value <= '9' ? static_cast<unsigned>(value - '0') : static_cast<unsigned>(value - 'a' + 10);
}

bool valid_reference(std::string_view reference) noexcept {
  if (reference.size() != reference_prefix.size() + 64 || !reference.starts_with(reference_prefix))
    return false;
  for (const auto character : reference.substr(reference_prefix.size()))
    if (!((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')))
      return false;
  return true;
}

Status read_string(const std::uint8_t*& input, const std::uint8_t* end,
                   const Limits& limits, std::string_view& out) noexcept {
  if (end - input < 2) return Status::corrupt_manifest;
  const auto size = read_be<std::uint16_t>(input);
  if (size == 0 || end - input < size) return Status::corrupt_manifest;
  if (size > limits.max_field_bytes) return Status::limit;
  out = {reinterpret_cast<const char*>(input), size};
  input += size;
  return Status::ok;
}
}

Manifest::Manifest() noexcept = default;
Manifest::~Manifest() noexcept = default;
Manifest::Manifest(Manifest&&) noexcept = default;
Manifest& Manifest::operator=(Manifest&&) noexcept = default;
Manifest::operator bool() const noexcept { return static_cast<bool>(state_); }

Status Manifest::create(const Description& description, const Limits& limits,
                        Manifest& out) noexcept {
  std::size_t size = 0;
  if (const auto status = measure(description, limits, size); status != Status::ok) return status;
  try {
    auto state = std::make_shared<detail::ManifestState>();
    state->description = description;
    std::sort(state->description.evidence.begin(), state->description.evidence.end(),
              [](const auto& left, const auto& right) { return compare_evidence(view(left), view(right)) < 0; });
    state->encoded.resize(size);
    auto* cursor = state->encoded.data();
    for (const char character : std::string_view("SQM1")) *cursor++ = static_cast<std::uint8_t>(character);
    write_be(cursor, std::uint16_t{1});
    write_be(cursor, std::uint16_t{0});
    write_be(cursor, std::uint32_t{0});
    write_be(cursor, static_cast<std::uint32_t>(size - prefix_bytes - digest_bytes));
    for (const auto field : fields(state->description)) write_string(cursor, field);
    write_be(cursor, static_cast<std::uint16_t>(state->description.evidence.size()));
    for (const auto& evidence : state->description.evidence) {
      *cursor++ = static_cast<std::uint8_t>(evidence.role);
      write_string(cursor, evidence.producer_ref);
      write_string(cursor, evidence.evidence_ref);
    }
    const auto digest = digest_hex({state->encoded.data(), size - digest_bytes});
    for (std::size_t index = 0; index < digest_bytes; ++index)
      *cursor++ = static_cast<std::uint8_t>((hex_value(digest[index * 2]) << 4) | hex_value(digest[index * 2 + 1]));
    state->reference = reference_prefix;
    state->reference += digest;
    Manifest result;
    result.state_ = std::move(state);
    out = std::move(result);
    return Status::ok;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}

Status Manifest::resolve(ByteView encoded, std::string_view expected_reference,
                         const Limits& limits, Manifest& out) noexcept {
  if (!valid_limits(limits) || !valid_reference(expected_reference)) return Status::invalid_argument;
  if (encoded.size() > limits.max_manifest_bytes) return Status::limit;
  if (encoded.size() < prefix_bytes + digest_bytes ||
      std::memcmp(encoded.data(), "SQM1", 4) != 0) return Status::corrupt_manifest;
  const auto* cursor = encoded.data() + 4;
  const auto major = read_be<std::uint16_t>(cursor);
  const auto minor = read_be<std::uint16_t>(cursor);
  const auto flags = read_be<std::uint32_t>(cursor);
  if (major != 1 || minor != 0 || flags != 0) return Status::unsupported_manifest;
  const auto body_size = read_be<std::uint32_t>(cursor);
  if (body_size != encoded.size() - prefix_bytes - digest_bytes) return Status::corrupt_manifest;
  const auto* end = encoded.data() + encoded.size() - digest_bytes;
  std::array<std::string_view, 6> parsed_fields;
  for (auto& field : parsed_fields)
    if (const auto status = read_string(cursor, end, limits, field); status != Status::ok) return status;
  if (end - cursor < 2) return Status::corrupt_manifest;
  const auto count = read_be<std::uint16_t>(cursor);
  if (count > limits.max_evidence_refs) return Status::limit;
  std::array<EvidenceView, max_evidence_refs> evidence{};
  std::array<unsigned, 3> required{};
  for (std::size_t index = 0; index < count; ++index) {
    if (cursor == end) return Status::corrupt_manifest;
    auto& item = evidence[index];
    item.role = static_cast<EvidenceRole>(*cursor++);
    if (!known_role(item.role)) return Status::unsupported_manifest;
    if (const auto status = read_string(cursor, end, limits, item.producer); status != Status::ok) return status;
    if (const auto status = read_string(cursor, end, limits, item.reference); status != Status::ok) return status;
    if (index != 0 && compare_evidence(evidence[index - 1], item) >= 0) return Status::corrupt_manifest;
    const auto role = static_cast<std::uint8_t>(item.role);
    if (role <= 3) ++required[role - 1];
  }
  if (cursor != end || std::any_of(required.begin(), required.end(), [](unsigned value) { return value != 1; }))
    return Status::corrupt_manifest;
  try {
    const auto digest = digest_hex(encoded.first(encoded.size() - digest_bytes));
    for (std::size_t index = 0; index < digest_bytes; ++index)
      if (hex[end[index] >> 4] != digest[index * 2] || hex[end[index] & 15] != digest[index * 2 + 1])
        return Status::corrupt_manifest;
    if (expected_reference.substr(reference_prefix.size()) != digest) return Status::reference_mismatch;
    auto state = std::make_shared<detail::ManifestState>();
    state->description.dataset_id = parsed_fields[0];
    state->description.dataset_revision = parsed_fields[1];
    state->description.schema_version = parsed_fields[2];
    state->description.layout_version = parsed_fields[3];
    state->description.access_scope = parsed_fields[4];
    state->description.producer_ref = parsed_fields[5];
    state->description.evidence.reserve(count);
    for (std::size_t index = 0; index < count; ++index)
      state->description.evidence.push_back({evidence[index].role,
          std::string(evidence[index].producer), std::string(evidence[index].reference)});
    state->encoded.assign(encoded.begin(), encoded.end());
    state->reference = expected_reference;
    Manifest result;
    result.state_ = std::move(state);
    out = std::move(result);
    return Status::ok;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}

Status Manifest::retain(Manifest& out) const noexcept {
  if (!state_) return Status::invalid_argument;
  Manifest result;
  result.state_ = state_;
  out = std::move(result);
  return Status::ok;
}

Status Manifest::binding(sqfv::Binding& out) const noexcept {
  if (!state_) return Status::invalid_argument;
  try {
    sqfv::Binding result{state_->reference, state_->description.dataset_revision,
        state_->description.schema_version, state_->description.layout_version,
        state_->description.access_scope};
    out = std::move(result);
    return Status::ok;
  } catch (const std::bad_alloc&) { return Status::no_memory; }
    catch (...) { return Status::internal_error; }
}

Status Manifest::verify_binding(const sqfv::Binding& binding) const noexcept {
  if (!state_) return Status::invalid_argument;
  const auto& description = state_->description;
  return binding.metadata_ref == state_->reference &&
      binding.dataset_revision == description.dataset_revision &&
      binding.schema_version == description.schema_version &&
      binding.layout_version == description.layout_version &&
      binding.access_scope == description.access_scope ? Status::ok : Status::binding_mismatch;
}

const Description& Manifest::description() const noexcept { return state_->description; }
std::string_view Manifest::reference() const noexcept { return state_ ? state_->reference : std::string_view{}; }
ByteView Manifest::encoded() const noexcept { return state_ ? ByteView(state_->encoded) : ByteView{}; }

} // namespace symphony::sqmv
