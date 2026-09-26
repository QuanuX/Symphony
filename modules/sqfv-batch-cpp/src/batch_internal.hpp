#ifndef SYMPHONY_SQFV_BATCH_INTERNAL_HPP
#define SYMPHONY_SQFV_BATCH_INTERNAL_HPP

#include "symphony/sqfv/batch.h"

namespace sqfv_internal {

// The canonical descriptor byte sequence has eight u16-big-endian lengths and
// raw values in this order: metadata_ref, dataset_revision, schema_version,
// layout_version, access_scope, partition, source_binding, source_position.
// It then has generation[16], sequence u64 BE, and record_count u64 BE.
// This sequence is used unchanged by the v1 frame and content identity.
bool compute_content_id(const sqfv_descriptor &descriptor, sqfv_span payload,
                        uint8_t out[SQFV_BATCH_CONTENT_ID_BYTES]) noexcept;

// A read-only copy of configured finite limits. A live context handle is
// required; using a destroyed handle is a caller error.
sqfv_limits context_limits(const sqfv_context *context) noexcept;

// These are internal, borrowed views for the synchronous frame codec. The
// caller must keep both handles alive for the duration of the call.
sqfv_span batch_payload_view(const sqfv_batch *batch) noexcept;
bool batch_belongs_to_context(const sqfv_batch *batch,
                              const sqfv_context *context) noexcept;

} // namespace sqfv_internal

#endif
