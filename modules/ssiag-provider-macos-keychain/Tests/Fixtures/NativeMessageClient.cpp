// C++26 disposable XPC client. Fixed nonsecret metadata only; no production target.
#include <xpc/xpc.h>
#include <dispatch/dispatch.h>
#include <unistd.h>
#include <cstdio>
#include <cstring>
#include <string>
#include <fstream>
#include <thread>
#include <chrono>

int main(int argc, char** argv) {
    if (argc != 3) return 2;
    alarm(8);
    const std::string scenario{argv[1]};
    auto connection = xpc_connection_create("symphony.fixture.message-service", nullptr);
    if (xpc_connection_set_peer_code_signing_requirement(connection, "identifier \"symphony.fixture.message-service\"") != 0) return 3;
    xpc_connection_set_event_handler(connection, ^(xpc_object_t) {});
    xpc_connection_activate(connection);
    auto request = xpc_dictionary_create_empty();
    xpc_dictionary_set_string(request, "format", "ssiag-internal-native-message-1");
    xpc_dictionary_set_string(request, "tops_id", "018f0c3a-7b2d-7e11-8c12-0242ac120002");
    xpc_dictionary_set_string(request, "request_id", "684921d8-a8b5-49da-872b-568eb6a6dc03");
    xpc_dictionary_set_string(request, "binding_digest", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    xpc_dictionary_set_string(request, "challenge", "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287");
    if (scenario == "wrong-request") xpc_dictionary_set_string(request, "request_id", "684921d8-a8b5-49da-872b-568eb6a6dc04");
    if (scenario == "wrong-tops") xpc_dictionary_set_string(request, "tops_id", "018f0c3a-7b2d-7e11-8c12-0242ac120003");
    if (scenario == "wrong-digest") xpc_dictionary_set_string(request, "binding_digest", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb");
    if (scenario == "wrong-challenge") xpc_dictionary_set_string(request, "challenge", "0190c7df-6df2-7f2b-9f4f-8e0c33f5f288");
    if (scenario == "extra") xpc_dictionary_set_bool(request, "unexpected", true);
    if (scenario == "type") xpc_dictionary_set_int64(request, "challenge", 1);
    if (scenario == "missing") xpc_dictionary_set_value(request, "challenge", nullptr);
    if (scenario == "oversize") { const std::string large(65536, 'x'); xpc_dictionary_set_string(request, "challenge", large.c_str()); }
    if (scenario == "data") { const unsigned char value[]{1,2,3}; xpc_dictionary_set_data(request, "challenge", value, sizeof(value)); }
    xpc_connection_send_message(connection, request);
    if (scenario == "duplicate") for (int i = 0; i < 31; ++i) xpc_connection_send_message(connection, request);
    for (int i = 0; i < 600; ++i) {
        std::ifstream output{argv[2]}; std::string value;
        if (output >> value) { std::printf("%s\n", value.c_str()); xpc_release(request); xpc_connection_cancel(connection); xpc_release(connection); return 0; }
        std::this_thread::sleep_for(std::chrono::milliseconds(10));
    }
    return 4;
}
