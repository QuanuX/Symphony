#ifndef SYMPHONY_SBV_PROVIDER_V1_H
#define SYMPHONY_SBV_PROVIDER_V1_H
#include <stdint.h>
#if defined(_WIN32)
#define SBV_PROVIDER_EXPORT __declspec(dllexport)
#else
#define SBV_PROVIDER_EXPORT __attribute__((visibility("default")))
#endif
#ifdef __cplusplus
extern "C" {
#endif
#define SBV_PROVIDER_ABI_V1 UINT32_C(1)
#define SBV_PROVIDER_OK INT32_C(0)
#define SBV_PROVIDER_UNAVAILABLE INT32_C(1)
#define SBV_PROVIDER_CANCELLED INT32_C(2)
#define SBV_PROVIDER_INVALID_INPUT INT32_C(3)
#define SBV_PROVIDER_FAILED INT32_C(4)
#define SBV_PROVIDER_STRATEGY UINT64_C(1)
#define SBV_PROVIDER_MODEL UINT64_C(2)
#define SBV_CALL_RUNNING UINT32_C(0)
#define SBV_CALL_CANCEL_REQUESTED UINT32_C(1)
#define SBV_CALL_DEADLINE_EXCEEDED UINT32_C(2)

/* Native platform C layout, not a wire format. No exceptions may cross any
 * callback. All input views, readers and call contexts are borrowed ONLY for
 * their active synchronous invocation on its invoking thread. NEVER retain
 * or use host callbacks/tokens after return, even while an instance lives.
 * Host checks during live calls do not make dangling callback pointers safe. */
typedef struct sbv_bytes_v1 {
  uint32_t struct_size, abi_version;
  const uint8_t *data;
  uint64_t size;
} sbv_bytes_v1;
/* Size excludes optional NUL. A nonempty output owns immutable bytes until
 * release exactly once, including on host validation failure. Release may use
 * provider-owned state, NEVER an expired invocation context. The host releases
 * outputs before instance destruction and library unload. Empty: all payload
 * fields zero. No host allocator may free provider-owned data. */
typedef struct sbv_owned_bytes_v1 {
  uint32_t struct_size, abi_version;
  const uint8_t *data;
  uint64_t size;
  void *owner;
  void (*release)(void *owner, const uint8_t *data, uint64_t size);
} sbv_owned_bytes_v1;
typedef struct sbv_call_context_v1 {
  uint32_t struct_size, abi_version;
  uint32_t has_deadline;
  int64_t deadline_unix_ms;
  void *token;
  uint32_t (*cancel_state)(void *token);
} sbv_call_context_v1;
typedef struct sbv_mbo_event_v1 {
  uint32_t struct_size, abi_version;
  uint64_t source_ordinal;
  uint16_t publisher_id;
  uint32_t instrument_id;
  uint64_t ts_event, order_id;
  int64_t price_nanos;
  uint32_t size;
  uint8_t flags, channel_id, action, side;
  uint64_t ts_recv;
  int32_t ts_in_delta;
  uint32_t sequence, has_ts_out;
  uint64_t ts_out;
} sbv_mbo_event_v1;
typedef struct sbv_event_reader_v1 {
  uint32_t struct_size, abi_version;
  uint64_t first_ordinal, end_ordinal_exclusive;
  void *token;
  /* Provider initializes out prefix. Failure preserves all of out. */
  int32_t (*read)(void *token, uint64_t ordinal, sbv_mbo_event_v1 *out);
} sbv_event_reader_v1;
typedef struct sbv_strategy_event_v1 {
  uint32_t struct_size, abi_version;
  const sbv_mbo_event_v1 *event;
  const sbv_event_reader_v1 *prefix;
} sbv_strategy_event_v1;
typedef struct sbv_model_frame_v1 {
  uint32_t struct_size, abi_version;
  sbv_bytes_v1 signal_json;
  const sbv_event_reader_v1 *causal_events, *followup_events;
  uint64_t horizon_end_ns;
  sbv_bytes_v1 coverage_json;
} sbv_model_frame_v1;
typedef struct sbv_provider_api_v1 {
  uint32_t struct_size, abi_version;
  uint64_t role_bits, required_capability_bits;
  int32_t (*descriptor)(const sbv_call_context_v1 *, sbv_owned_bytes_v1 *,
                        sbv_owned_bytes_v1 *error);
  int32_t (*strategy_create)(const sbv_call_context_v1 *, const sbv_bytes_v1 *,
                             void **instance, sbv_owned_bytes_v1 *error);
  int32_t (*strategy_event)(void *, const sbv_call_context_v1 *,
                            const sbv_strategy_event_v1 *, sbv_owned_bytes_v1 *,
                            sbv_owned_bytes_v1 *error);
  int32_t (*strategy_finish)(void *, const sbv_call_context_v1 *,
                             sbv_owned_bytes_v1 *, sbv_owned_bytes_v1 *error);
  void (*strategy_destroy)(void *);
  int32_t (*model_create)(const sbv_call_context_v1 *, const sbv_bytes_v1 *,
                          void **instance, sbv_owned_bytes_v1 *error);
  int32_t (*model_evaluate)(void *, const sbv_call_context_v1 *,
                            const sbv_model_frame_v1 *, sbv_owned_bytes_v1 *,
                            sbv_owned_bytes_v1 *error);
  void (*model_destroy)(void *);
} sbv_provider_api_v1;
typedef int32_t (*sbv_provider_get_api_v1)(sbv_provider_api_v1 *);
/* Host sets struct_size to capacity and abi_version=1. Success supplies the
 * complete v1 table, never writes beyond capacity and reports supplied size.
 * A failed create must clean up and leave instance null. Destroy/release do not
 * throw or retain host work. No hard interruption/isolation is provided. */
SBV_PROVIDER_EXPORT int32_t
symphony_sbv_provider_api_v1(sbv_provider_api_v1 *inout_api);
#ifdef __cplusplus
}
#endif
#endif
