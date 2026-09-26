#ifndef SYMPHONY_SQFV_BATCH_H
#define SYMPHONY_SQFV_BATCH_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#define SQFV_BATCH_ABI_VERSION 1u
#define SQFV_BATCH_CONTENT_ID_BYTES 32u
#define SQFV_BATCH_GENERATION_BYTES 16u

/* Calls on distinct live handles may run concurrently. A caller must finish
 * every call using one opaque handle before destroying that exact handle; it
 * must also not use a released handle or a view after its retaining handle is
 * released. Status-returning calls contain C++ failures. A void release that
 * cannot complete its internal synchronization terminates rather than
 * falsely reporting that its obligation was discharged. */

/* A span is borrowed for the duration of a call unless a returned view says
 * otherwise. A nonzero size requires a nonnull data pointer. */
typedef struct sqfv_span {
  const uint8_t *data;
  uint64_t size;
} sqfv_span;

typedef enum sqfv_status {
  SQFV_OK = 0,
  SQFV_INVALID_ARGUMENT = 1,
  SQFV_UNSUPPORTED_ABI = 2,
  SQFV_LIMIT = 3,
  SQFV_NO_MEMORY = 4,
  SQFV_BLOCKED = 5,
  SQFV_DUPLICATE = 6,
  SQFV_CONFLICT = 7,
  SQFV_GAP = 8,
  SQFV_STALE = 9,
  SQFV_SCOPE_MISMATCH = 10,
  SQFV_BINDING_MISMATCH = 11,
  SQFV_CLOSED = 12,
  SQFV_EMPTY = 13,
  SQFV_CORRUPT_FRAME = 14,
  SQFV_UNSUPPORTED_FRAME = 15,
  SQFV_INTERNAL_ERROR = 16
} sqfv_status;

typedef struct sqfv_context sqfv_context;
typedef struct sqfv_batch sqfv_batch;
typedef struct sqfv_lease sqfv_lease;
typedef struct sqfv_port sqfv_port;

/* All limits are mandatory, finite and nonzero. The technical ceilings are
 * specified in the installed module SPEC. No default is inferred. */
typedef struct sqfv_limits {
  uint32_t struct_size;
  uint32_t abi_version;
  uint64_t max_payload_bytes;
  uint64_t max_frame_bytes;
  uint64_t max_descriptor_bytes;
  uint64_t global_allocation_bytes;
  uint32_t max_ports;
  uint32_t reserved;
} sqfv_limits;

/* All five binding fields are required opaque byte strings. SQFV compares
 * them exactly; their meaning and access authority belong elsewhere. */
typedef struct sqfv_binding {
  uint32_t struct_size;
  uint32_t abi_version;
  sqfv_span metadata_ref;
  sqfv_span dataset_revision;
  sqfv_span schema_version;
  sqfv_span layout_version;
  sqfv_span access_scope;
} sqfv_binding;

typedef struct sqfv_descriptor {
  uint32_t struct_size;
  uint32_t abi_version;
  sqfv_binding binding;
  sqfv_span partition;
  sqfv_span source_binding;
  sqfv_span source_position;
  uint8_t producer_generation[SQFV_BATCH_GENERATION_BYTES];
  uint64_t batch_sequence;
  uint64_t record_count;
} sqfv_descriptor;

typedef struct sqfv_port_config {
  uint32_t struct_size;
  uint32_t abi_version;
  sqfv_binding binding;
  sqfv_span partition;
  uint8_t producer_generation[SQFV_BATCH_GENERATION_BYTES];
  uint64_t next_sequence;
  uint64_t outstanding_byte_credit;
  uint32_t max_pending_entries;
  uint32_t reserved;
} sqfv_port_config;

typedef struct sqfv_context_stats {
  uint32_t struct_size;
  uint32_t abi_version;
  uint64_t allocation_bytes;
  uint64_t peak_allocation_bytes;
  uint64_t allocation_limit_bytes;
  uint32_t live_ports;
  uint32_t reserved;
} sqfv_context_stats;

typedef struct sqfv_port_stats {
  uint32_t struct_size;
  uint32_t abi_version;
  uint64_t next_sequence;
  uint64_t outstanding_bytes;
  uint64_t outstanding_byte_credit;
  uint32_t pending_entries;
  uint32_t max_pending_entries;
  uint32_t sequence_exhausted;
  uint32_t reserved;
} sqfv_port_stats;

sqfv_status sqfv_context_create(const sqfv_limits *limits,
                                sqfv_context **out_context);
void sqfv_context_destroy(sqfv_context *context);
sqfv_status sqfv_context_get_stats(const sqfv_context *context,
                                   sqfv_context_stats *out_stats);

sqfv_status sqfv_batch_prepare_copy(sqfv_context *context,
                                    const sqfv_descriptor *descriptor,
                                    sqfv_span payload, sqfv_batch **out_batch);
sqfv_status sqfv_batch_retain(const sqfv_batch *batch, sqfv_batch **out_batch);
void sqfv_batch_release(sqfv_batch *batch);
/* Set output struct_size and abi_version to exact v1 values. Returned spans
 * remain valid only while this exact batch handle is retained. */
sqfv_status sqfv_batch_descriptor_view(const sqfv_batch *batch,
                                       sqfv_descriptor *out_descriptor);
sqfv_status sqfv_batch_content_id(const sqfv_batch *batch,
                                  uint8_t out_id[SQFV_BATCH_CONTENT_ID_BYTES]);

sqfv_status sqfv_lease_acquire(const sqfv_batch *batch, sqfv_span access_scope,
                               sqfv_lease **out_lease);
/* The payload view remains valid only until this exact lease is released. */
sqfv_status sqfv_lease_view(const sqfv_lease *lease, sqfv_span *out_payload);
/* Set output struct_size and abi_version to exact v1 values. */
sqfv_status sqfv_lease_descriptor_view(const sqfv_lease *lease,
                                       sqfv_descriptor *out_descriptor);
sqfv_status sqfv_lease_content_id(const sqfv_lease *lease,
                                  uint8_t out_id[SQFV_BATCH_CONTENT_ID_BYTES]);
void sqfv_lease_release(sqfv_lease *lease);

/* A port binds one exact metadata binding, partition and producer generation.
 * Its first accepted sequence is next_sequence in the port configuration. */
sqfv_status sqfv_port_add(sqfv_context *context, const sqfv_port_config *config,
                          sqfv_port **out_port);
/* Destruction cancels queued deliveries. Already taken leases stay valid. */
void sqfv_port_destroy(sqfv_port *port);
sqfv_status sqfv_port_offer(sqfv_port *port, const sqfv_batch *batch);
/* Taking a delivery needs no new allocation or data credit. */
sqfv_status sqfv_port_take(sqfv_port *port, sqfv_lease **out_lease);
/* Cancels one still-pending delivery; accepted cursor is not rolled back. */
sqfv_status sqfv_port_cancel(sqfv_port *port, uint64_t batch_sequence);
sqfv_status sqfv_port_get_stats(const sqfv_port *port,
                                sqfv_port_stats *out_stats);

/* Caller owns the output buffer. The frame is a local integrity-checked
 * serialization, not an authenticated or cross-process transport. */
sqfv_status sqfv_frame_measure(const sqfv_context *context,
                               const sqfv_batch *batch, uint64_t *out_size);
sqfv_status sqfv_frame_encode(const sqfv_context *context,
                              const sqfv_batch *batch, uint8_t *out,
                              uint64_t capacity, uint64_t *out_size);
sqfv_status sqfv_frame_decode(sqfv_context *context, sqfv_span frame,
                              sqfv_batch **out_batch);

#ifdef __cplusplus
}
#endif

#endif
