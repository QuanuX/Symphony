#ifndef SYMPHONY_SQMV_METADATA_HPP
#define SYMPHONY_SQMV_METADATA_HPP

#include <cstddef>
#include <cstdint>
#include <memory>
#include <span>
#include <string>
#include <string_view>
#include <vector>

#include <symphony/sqfv/batch.hpp>

namespace symphony::sqmv {

using ByteView = std::span<const std::uint8_t>;

enum class Status : std::uint8_t {
  ok,
  invalid_argument,
  limit,
  no_memory,
  unsupported_manifest,
  corrupt_manifest,
  reference_mismatch,
  binding_mismatch,
  internal_error,
};

// Every limit is mandatory and positive. This release permits at most 65,536
// encoded bytes, 4,096 bytes per string, and 128 evidence references.
struct Limits {
  std::uint32_t max_manifest_bytes = 0;
  std::uint32_t max_field_bytes = 0;
  std::uint16_t max_evidence_refs = 0;
};

enum class EvidenceRole : std::uint8_t {
  schema = 1,
  layout = 2,
  access = 3,
  source = 4,
  time = 5,
  coverage = 6,
  lineage = 7,
  units = 8,
};

// References identify caller-resolved evidence. The library preserves the
// producer's attribution; it neither fetches evidence nor authenticates it.
struct EvidenceReference {
  EvidenceRole role = EvidenceRole::schema;
  std::string producer_ref;
  std::string evidence_ref;
};

// Strings are exact opaque byte strings, including embedded NUL. This profile
// requires exactly one schema, layout, and access reference. Other roles may
// have several distinct references. Evidence order has no semantic meaning.
struct Description {
  std::string dataset_id;
  std::string dataset_revision;
  std::string schema_version;
  std::string layout_version;
  std::string access_scope;
  std::string producer_ref;
  std::vector<EvidenceReference> evidence;
};

namespace detail { struct ManifestState; }

// Immutable, retained metadata for trusted callers in one address space.
// Fallible calls contain exceptions and change outputs only on success.
// Borrowed references remain valid while the retaining handle remains live;
// do not destroy or move that handle concurrently with its use.
class Manifest final {
 public:
  Manifest() noexcept;
  ~Manifest() noexcept;
  Manifest(Manifest&&) noexcept;
  Manifest& operator=(Manifest&&) noexcept;
  Manifest(const Manifest&) = delete;
  Manifest& operator=(const Manifest&) = delete;

  [[nodiscard]] explicit operator bool() const noexcept;
  [[nodiscard]] static Status create(const Description&, const Limits&,
                                     Manifest& out) noexcept;
  [[nodiscard]] static Status resolve(ByteView, std::string_view expected_reference,
                                      const Limits&, Manifest& out) noexcept;
  [[nodiscard]] Status retain(Manifest& out) const noexcept;
  [[nodiscard]] Status binding(sqfv::Binding& out) const noexcept;
  [[nodiscard]] Status verify_binding(const sqfv::Binding&) const noexcept;

  // Requires a nonempty handle (explicit operator bool() is true).
  [[nodiscard]] const Description& description() const noexcept;
  // Empty handles return empty views.
  [[nodiscard]] std::string_view reference() const noexcept;
  [[nodiscard]] ByteView encoded() const noexcept;

 private:
  std::shared_ptr<const detail::ManifestState> state_;
};

} // namespace symphony::sqmv

#endif
