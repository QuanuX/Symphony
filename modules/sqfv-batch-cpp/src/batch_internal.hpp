#ifndef SYMPHONY_SQFV_BATCH_INTERNAL_HPP
#define SYMPHONY_SQFV_BATCH_INTERNAL_HPP

#include "symphony/sqfv/batch.hpp"

namespace symphony::sqfv::detail {

struct BindingView {
  std::string_view metadata_ref;
  std::string_view dataset_revision;
  std::string_view schema_version;
  std::string_view layout_version;
  std::string_view access_scope;
};

struct DescriptorView {
  BindingView binding;
  std::string_view partition;
  std::string_view source_binding;
  std::string_view source_position;
  Generation producer_generation{};
  std::uint64_t batch_sequence = 0;
  std::uint64_t record_count = 0;
};

[[nodiscard]] inline DescriptorView view_of(const Descriptor& descriptor) noexcept {
  return {{descriptor.binding.metadata_ref,
           descriptor.binding.dataset_revision,
           descriptor.binding.schema_version,
           descriptor.binding.layout_version,
           descriptor.binding.access_scope},
          descriptor.partition,
          descriptor.source_binding,
          descriptor.source_position,
          descriptor.producer_generation,
          descriptor.batch_sequence,
          descriptor.record_count};
}

// Canonical v1 descriptor: eight u16-BE lengths and raw byte strings,
// generation[16], sequence u64-BE, and record count u64-BE.
[[nodiscard]] bool compute_content_id(const DescriptorView&, ByteView payload,
                                      ContentId& out) noexcept;

// The codec decodes borrowed views and calls prepare_copy_views; the core
// validates/reserves before making any owning descriptor or payload copy.
struct FrameAccess {
  [[nodiscard]] static const Limits* limits(const Context&) noexcept;
  [[nodiscard]] static ByteView payload(const Batch&) noexcept;
  [[nodiscard]] static bool same_context(const Context&, const Batch&) noexcept;
  [[nodiscard]] static Status prepare_copy_views(Context&, const DescriptorView&,
                                                 ByteView, Batch&) noexcept;
};

} // namespace symphony::sqfv::detail

#endif
