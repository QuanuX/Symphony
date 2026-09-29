#pragma once
#include "common.hpp"
#include <functional>
#include <symphony/sqmv/metadata.hpp>
namespace symphony::sqpv::inspection {
using namespace symphony::sqv_admin;
struct Snapshot {
  sqmv::Manifest metadata;
  std::string partition, metadata_hash, head_commit;
  sqfv::Generation producer_generation{}, store_generation{};
  std::uint64_t first_sequence = 0, next_sequence = 0, committed_batches = 0,
                physical_bytes = 0;
  std::uint64_t max_frame_bytes = 0, max_store_bytes = 0, max_batches = 0;
  bool sequence_exhausted = false, recovery_required = false;
};
using Visitor =
    std::function<void(const Snapshot &, const sqfv::Batch &, sqfv::ByteView)>;
// Does not construct a Store. Holds a shared nonblocking lock on the existing
// writer.lock. Only O_RDONLY opens; never creates, repairs, syncs or renames.
Snapshot inspect(const Json &selection, std::int64_t deadline_ms,
                 const Visitor & = {});
Json summary(const Snapshot &);
std::string_view take(std::string_view &, std::size_t);
std::uint64_t take64(std::string_view &);
std::string field(std::string_view &, std::size_t);
} // namespace symphony::sqpv::inspection
