#include <stddef.h>
#include <stdio.h>
#include <symphony/sbv/provider.h>
_Static_assert(offsetof(sbv_bytes_v1, struct_size) == 0, "prefix");
_Static_assert(offsetof(sbv_bytes_v1, abi_version) == sizeof(uint32_t),
               "ABI prefix");
_Static_assert(sizeof(((sbv_bytes_v1 *)0)->size) == 8, "full length");
_Static_assert(sizeof(((sbv_mbo_event_v1 *)0)->ts_recv) == 8, "exact time");
_Static_assert(sizeof(((sbv_mbo_event_v1 *)0)->price_nanos) == 8,
               "exact price");
_Static_assert(sizeof(((sbv_mbo_event_v1 *)0)->ts_in_delta) == 4,
               "signed delta");
int main(void) {
  sbv_provider_api_v1 api = {0};
  api.struct_size = sizeof(api);
  api.abi_version = SBV_PROVIDER_ABI_V1;
  if (api.role_bits || api.descriptor || api.strategy_create ||
      api.model_create)
    return 1;
  puts("native provider public header: standalone C11 layout checks passed");
  return 0;
}
