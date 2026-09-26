#include <symphony/sqfv/batch.h>

#include <stdint.h>
#include <stdio.h>
#include <string.h>

static sqfv_span bytes(const void *value, uint64_t size) {
    sqfv_span result = {(const uint8_t *)value, size};
    return result;
}

#define LITERAL(value) bytes((value), sizeof(value) - 1u)

static int expect(sqfv_status actual, sqfv_status wanted, const char *step) {
    if(actual == wanted) return 1;
    fprintf(stderr, "%s: wanted status %d, got %d\n", step, (int)wanted, (int)actual);
    return 0;
}

int main(void) {
    sqfv_limits limits = {
        .struct_size = sizeof(sqfv_limits), .abi_version = SQFV_BATCH_ABI_VERSION,
        .max_payload_bytes = 1024, .max_frame_bytes = 4096,
        .max_descriptor_bytes = 512, .global_allocation_bytes = 8192,
        .max_ports = 2
    };
    sqfv_context *context = NULL;
    if(!expect(sqfv_context_create(&limits, &context), SQFV_OK, "context create")) return 1;
    sqfv_context_stats baseline = {
        .struct_size = sizeof(sqfv_context_stats), .abi_version = SQFV_BATCH_ABI_VERSION
    };
    if(!expect(sqfv_context_get_stats(context, &baseline), SQFV_OK,
               "initial context stats")) return 1;

    sqfv_binding binding = {
        .struct_size = sizeof(sqfv_binding), .abi_version = SQFV_BATCH_ABI_VERSION,
        .metadata_ref = LITERAL("fixture:metadata:1"),
        .dataset_revision = LITERAL("revision-1"),
        .schema_version = LITERAL("schema-1"),
        .layout_version = LITERAL("row-v1"),
        .access_scope = LITERAL("fixture-scope")
    };
    sqfv_descriptor descriptor = {
        .struct_size = sizeof(sqfv_descriptor), .abi_version = SQFV_BATCH_ABI_VERSION,
        .binding = binding, .partition = LITERAL("partition-a"),
        .source_binding = LITERAL("fixture-source"),
        .source_position = LITERAL("opaque-7"),
        .producer_generation = {1}, .batch_sequence = 7, .record_count = 3
    };
    uint8_t caller_payload[16] = {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15};
    sqfv_batch *batch = NULL;
    if(!expect(sqfv_batch_prepare_copy(context, &descriptor,
        bytes(caller_payload, sizeof(caller_payload)), &batch), SQFV_OK, "prepare copy")) return 1;
    caller_payload[0] = 99;

    sqfv_lease *direct = NULL;
    if(!expect(sqfv_lease_acquire(batch, binding.access_scope, &direct), SQFV_OK,
               "read lease acquire")) return 1;
    sqfv_span view = {0};
    if(!expect(sqfv_lease_view(direct, &view), SQFV_OK, "read lease view") ||
       view.size != 16 || view.data[0] != 0 || view.data[15] != 15) {
        fputs("frozen payload mismatch\n", stderr);
        return 1;
    }

    sqfv_port_config port_config = {
        .struct_size = sizeof(sqfv_port_config), .abi_version = SQFV_BATCH_ABI_VERSION,
        .binding = binding, .partition = LITERAL("partition-a"),
        .producer_generation = {1}, .next_sequence = 7,
        .outstanding_byte_credit = 1024, .max_pending_entries = 2
    };
    sqfv_port *port = NULL;
    if(!expect(sqfv_port_add(context, &port_config, &port), SQFV_OK, "port add") ||
       !expect(sqfv_port_offer(port, batch), SQFV_OK, "port offer")) return 1;
    sqfv_lease *taken = NULL;
    if(!expect(sqfv_port_take(port, &taken), SQFV_OK, "port take")) return 1;
    sqfv_port_destroy(port);
    sqfv_batch_release(batch);
    if(!expect(sqfv_lease_view(taken, &view), SQFV_OK, "taken lease after owner release") ||
       view.size != 16 || view.data[0] != 0) {
        fputs("taken lease lost its retained payload\n", stderr);
        return 1;
    }
    sqfv_lease_release(taken);
    sqfv_lease_release(direct);

    sqfv_context_stats stats = {
        .struct_size = sizeof(sqfv_context_stats), .abi_version = SQFV_BATCH_ABI_VERSION
    };
    if(!expect(sqfv_context_get_stats(context, &stats), SQFV_OK, "context stats") ||
       stats.allocation_bytes != baseline.allocation_bytes ||
       stats.peak_allocation_bytes <= baseline.allocation_bytes) {
        fputs("unexpected allocation accounting\n", stderr);
        return 1;
    }
    sqfv_context_destroy(context);
    puts("installed C11 consumer: frozen copy, exact binding, offer/take, lease lifetime, accounting");
    return 0;
}
