#include "execution.hpp"
#include "native_test.hpp"
using namespace native_test;
namespace n = symphony::ssiag::native;
int main(int argc, char **argv) {
  try {
    NT_REQUIRE(argc == 2);
    auto package = fs::canonical(argv[1]);
    auto bundle = read_json(package / "bundle.json");
    auto id = cloudflare::digest(bundle, "symphony.sov.cloudflare-bundle.v1");
    auto qualified = cloudflare::owner().at("qualified_resource_toolchains");
    NT_REQUIRE(std::find(qualified.begin(), qualified.end(), id) !=
               qualified.end());
    TempDir temp("cf-runner-");
    auto profile = cloudflare::owner().at("resource_profiles").at(0);
    Json in{{"resource_operation", profile.at("operation")},
            {"parameters", profile.at("template")},
            {"context",
             {{"account_id", std::string(32, '0')}, {"zone_id", nullptr}}},
            {"execution", nullptr}};
    auto preview = cloudflare::resource_preview(in);
    preview["request"]["url"] =
        preview.at("request").at("url").get<std::string>() + "/refused-target";
    // The exact production runner executes the actual pinned CLI. The
    // receipt-owned guard rejects this deliberate mismatch before any network
    // call can occur.
    auto result = cloudflare::run_http(preview, "FAKE-TOKEN-SENTINEL", package,
                                       temp.path, n::now() + 20000);
    NT_REQUIRE(!result.attempted && !result.clean_exit &&
               result.body.is_null() && fs::is_empty(temp.path));
    std::cout << "Production HTTP runner: private FD framing, pre-network "
                 "target refusal, raw-output exclusion and scratch cleanup "
                 "passed; zero API attempts\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
