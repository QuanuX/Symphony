#include "adapter.hpp"
#include "native_test.hpp"
#include <iostream>
using namespace native_test;

void test_cloudflare_owner_rejects_unknown_fields() {
    auto schema = cloudflare::owner().at("operations").at(0).at("input_schema");
    bool rejected = false;
    try { cloudflare::validate(Json{{"store", "/private/store"}, {"extra", true}}, schema); }
    catch (...) { rejected = true; }
    NT_REQUIRE(rejected);
    cloudflare::validate(Json{{"store", "/private/store"}}, schema);
}

void test_cloudflare_installed_process(const fs::path& prefix) {
    auto adapter = prefix / "libexec/symphony/sov-cloudflare-adapter/0.1.0-dev/symphony-sov-cloudflare-adapter";
    const auto receipt = read_json(prefix / "share/symphony/receipts/sov-cloudflare-adapter/0.1.0-dev/install-receipt.json");
    NT_REQUIRE(receipt.at("component_id") == "sov-cloudflare-adapter");
    for (const auto& f : receipt.at("files")) NT_REQUIRE(digest(read(prefix / f.at("path").get<std::string>())) == f.at("digest").get<std::string>());
    const auto descriptor = native_test::run({adapter.string(), "--describe"});
    NT_REQUIRE(descriptor.returncode == 0);
    const auto owner = Json::parse(descriptor.stdout_text);
    Json request{{"protocol", "symphony.sov.cloudflare-request.v1"}, {"request_id", "12345678-1234-4234-8234-123456789abc"}, {"correlation_id", "23456789-2345-4345-8345-23456789abcd"}, {"operation", "capability.list"}, {"descriptor_ref", cloudflare::digest(owner, "symphony.sov.cloudflare-owner.v1")}, {"installation_ref", {{"version", "0.1.0-dev"}, {"executable_digest", digest(read(adapter))}, {"receipt_digest", receipt.at("receipt_digest")}}}, {"context_ref", nullptr}, {"plan_ref", nullptr}, {"input", {{"store", nullptr}, {"toolchain_id", nullptr}, {"query", nullptr}}}};
    const auto response = native_test::run({adapter.string()}, request.dump());
    NT_REQUIRE(response.returncode == 0);
    auto value = Json::parse(response.stdout_text);
    NT_REQUIRE(value.at("data").at("owner") == owner);
    NT_REQUIRE(value.at("request_ref") == cloudflare::digest(request, "symphony.sov.cloudflare-request.v1"));
    request["input"]["unknown"] = true;
    const auto rejected = native_test::run({adapter.string()}, request.dump());
    NT_REQUIRE(rejected.returncode == 5);
    NT_REQUIRE(Json::parse(rejected.stdout_text).at("dispatch_state") == "not_dispatched");
}

int main(int argc, char** argv) {
    try {
        NT_REQUIRE(argc == 2);
        test_cloudflare_owner_rejects_unknown_fields();
        test_cloudflare_installed_process(fs::canonical(argv[1]));
        std::cout << "Cloudflare native producer and installed consumer passed\n";
        return 0;
    } catch (const std::exception& e) { std::cerr << e.what() << '\n'; return 1; }
}
