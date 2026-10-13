#ifndef SYMPHONY_SBV_FEEDBACK_V1_H
#define SYMPHONY_SBV_FEEDBACK_V1_H
#include <symphony/sbv/provider.h>
#ifdef __cplusplus
extern "C" {
#endif
/* Independent optional ABI: existing provider v1 tables are unchanged.
 * Input is account-feedback-call.v1 JSON. Output is feedback-decision.v1 JSON.
 * All views and cancellation contexts are borrowed only during the synchronous
 * callback. The provider must return all continuation state in explicit memory;
 * hidden process state cannot establish portable branch equivalence. Owned
 * output buffers follow provider.h release/lifetime rules. No exceptions may
 * cross ABI. */
typedef struct sbv_feedback_api_v1 {
  uint32_t struct_size, abi_version;
  uint64_t required_capability_bits;
  int32_t (*descriptor)(const sbv_call_context_v1 *, sbv_owned_bytes_v1 *,
                        sbv_owned_bytes_v1 *error);
  int32_t (*create)(const sbv_call_context_v1 *, const sbv_bytes_v1 *,
                    void **instance, sbv_owned_bytes_v1 *error);
  int32_t (*decide)(void *, const sbv_call_context_v1 *, const sbv_bytes_v1 *,
                    sbv_owned_bytes_v1 *, sbv_owned_bytes_v1 *error);
  void (*destroy)(void *);
} sbv_feedback_api_v1;
typedef int32_t (*sbv_feedback_get_api_v1)(sbv_feedback_api_v1 *);
SBV_PROVIDER_EXPORT int32_t
symphony_sbv_feedback_api_v1(sbv_feedback_api_v1 *inout_api);
#ifdef __cplusplus
}
#endif
#endif
