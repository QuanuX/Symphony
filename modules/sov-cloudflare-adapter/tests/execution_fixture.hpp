#pragma once
#include "execution.hpp"
#include "native_test.hpp"
#include <sys/socket.h>
#include <sys/un.h>
#include <thread>
using namespace native_test;
namespace n = symphony::ssiag::native;
using cloudflare::RuntimeHooks;
// The native provider is real. This authority fixture isolates its native IPC
// boundary; the Go server suite separately exercises actual policy/STAV
// admission.
struct Authority {
  n::FD listener;
  std::thread worker;
  std::string error;
  fs::path socket;
  int expected = 0;
  Authority(const fs::path &path, int calls) : socket(path), expected(calls) {
    listener.n = ::socket(AF_UNIX, SOCK_STREAM, 0);
    NT_REQUIRE(listener.n >= 0);
    sockaddr_un a{};
    a.sun_family = AF_UNIX;
    auto s = path.string();
    NT_REQUIRE(s.size() < sizeof(a.sun_path));
    std::copy(s.begin(), s.end(), a.sun_path);
    NT_REQUIRE(
        ::bind(listener.n, reinterpret_cast<sockaddr *>(&a), sizeof(a)) == 0);
    ::chmod(path.c_str(), 0600);
    NT_REQUIRE(::listen(listener.n, 8) == 0);
    worker = std::thread([&] {
      try {
        for (int i = 0; i < expected; ++i) {
          pollfd p{listener.n, POLLIN, 0};
          NT_REQUIRE(::poll(&p, 1, 20000) > 0);
          n::FD c(::accept(listener.n, nullptr, nullptr));
          n::timeout(c.n, 10);
          std::string head;
          while (!head.ends_with("\r\n\r\n")) {
            char b;
            n::exact_read(c.n, &b, 1);
            head += b;
            NT_REQUIRE(head.size() < 4096);
          }
          auto at = head.find("Content-Length: ");
          NT_REQUIRE(at != std::string::npos);
          auto size = std::stoul(head.substr(at + 16));
          std::string body(size, '\0');
          n::exact_read(c.n, body.data(), body.size());
          auto intent = n::parse(body);
          auto d = n::seal(intent, "symphony.ssiag.cloudflare-use.v1");
          Json receipt{
              {"disposition", "committed"},
              {"commit", {{"event_digest", "sha256:" + std::string(64, 'c')}}}};
          n::write_all(c.n, "HTTP/1.1 101 Switching Protocols\r\n\r\n");
          n::frame(c.n,
                   Json{{"protocol", "symphony.ssiag.cloudflare-use.v1"},
                        {"request_digest", d},
                        {"uid", std::to_string(::geteuid())},
                        {"deadline_unix_ms", intent.at("deadline_unix_ms")},
                        {"policy_digest", "sha256:" + std::string(64, 'b')},
                        {"receipt", receipt}}
                       .dump());
          auto completion = n::parse(n::frame(c.n));
          n::frame(c.n, Json{{"request_digest", d},
                             {"outcome", completion.at("outcome")},
                             {"receipt", receipt}}
                            .dump());
        }
      } catch (const std::exception &e) {
        error = e.what();
      }
    });
  }
  ~Authority() {
    if (worker.joinable())
      worker.join();
    ::unlink(socket.c_str());
  }
  void check() {
    worker.join();
    NT_REQUIRE(error.empty());
  }
};
struct Harness {
  TempDir temp{"cfexec-"};
  fs::path prefix, root, vault;
  Json execution, context;
  std::string executable_digest;
  int serial = 0;
  Harness(const fs::path &provider, const fs::path &self) {
    prefix = temp.path / "prefix";
    root = temp.path / "store";
    vault = temp.path / "vault";
    for (auto p : {prefix, root, vault})
      n::private_dir(p, true);
    n::private_dir(root / "staging", true);
    auto rel = fs::path("libexec/symphony/ssiag-provider-private-store/"
                        "0.1.0-dev/symphony-ssiag-provider-private-store");
    write(prefix / rel, read(provider));
    ::chmod((prefix / rel).c_str(), 0700);
    auto bytes = read(prefix / rel);
    Json receipt{{"protocol", "symphony.knowledge.install-receipt.v2"},
                 {"component_id", "ssiag-provider-private-store"},
                 {"version", "0.1.0-dev"},
                 {"files", Json::array({Json{{"path", rel.string()},
                                             {"size", bytes.size()},
                                             {"digest", n::hash(bytes)}}})}};
    auto rd = n::hash(receipt.dump());
    receipt["receipt_digest"] = rd;
    write_json(prefix / "share/symphony/receipts/ssiag-provider-private-store/"
                        "0.1.0-dev/install-receipt.json",
               receipt);
    executable_digest = n::hash(read(self));
    auto socket = temp.path / "a.sock";
    Json config{{"protocol", "symphony.ssiag.private-store-config.v1"},
                {"tops_id", "11111111-1111-4111-8111-111111111111"},
                {"ssiag_socket", socket.string()},
                {"ssiag_uid", std::to_string(::geteuid())},
                {"ssiag_digest", executable_digest},
                {"consumer_digest", executable_digest},
                {"toolchain_digest", "sha256:" + std::string(64, 'a')}};
    write_json(vault / "config.json", config);
    ::chmod((vault / "config.json").c_str(), 0600);
    execution =
        Json{{"provider_prefix", prefix.string()},
             {"provider_version", "0.1.0-dev"},
             {"provider_receipt", rd},
             {"vault", vault.string()},
             {"config_digest",
              n::seal(config, "symphony.ssiag.private-store-config.v1")},
             {"credential_generation", "1"},
             {"result_ref", nullptr},
             {"max_pages", 2}};
    context = Json{{"host_ref", "host:fixture"},
                   {"principal_uid", std::to_string(::geteuid())},
                   {"account_id", std::string(32, '0')},
                   {"zone_id", std::string(32, '2')},
                   {"credential_ref", "ssiag-native:api"},
                   {"toolchain_id", config.at("toolchain_digest")}};
    n::private_dir(vault / "credentials", true);
    n::private_dir(vault / "credentials/api", true);
    n::put(vault / "credentials/api/value", "SECRET-SENTINEL");
    n::put(vault / "credentials/api/metadata.json",
           Json{{"reference", "ssiag-native:api"},
                {"purpose", "cloudflare.api-token"},
                {"account_id", context.at("account_id")},
                {"zone_id", nullptr},
                {"resource_id", nullptr},
                {"generation", "1"},
                {"expires_unix_ms", std::to_string(n::now() + 3600000)}}
               .dump());
  }
  Json input(const std::string &op) {
    // Keep the declaration alive on compilers without range-for temporary extension.
    const auto declaration = cloudflare::owner();
    for (auto p : declaration.at("resource_profiles"))
      if (p.at("operation") == op)
        return Json{
            {"store", root.string()},
            {"context", context},
            {"expected_context_ref",
             cloudflare::digest(context, "symphony.sov.cloudflare-context.v1")},
            {"expected_state", "sha256:" + std::string(64, 'd')},
            {"resource_operation", op},
            {"parameters", p.at("template")},
            {"expected_observation", nullptr},
            {"execution", execution}};
    throw std::runtime_error("profile");
  }
  Json run(Json in, RuntimeHooks &hook, const std::string &command = "",
           const Json &old = nullptr) {
    auto plan =
        old.is_null()
            ? Json{{"input", in},
                   {"expires_unix_ms", std::to_string(n::now() + 3600000)},
                   {"nonce", ++serial}}
            : old;
    auto d =
        cloudflare::digest(plan, "symphony.sov.cloudflare-resource-plan.v1");
    auto op = in.at("resource_operation").get<std::string>();
    std::string verb = op.ends_with(".list")
                           ? "list"
                           : (op.ends_with(".get") ? "inspect" : "apply");
    Json request{
        {"operation", command.empty() ? "resource." + verb : command},
        {"input", {{"plan_digest", d}}},
        {"installation_ref", {{"executable_digest", executable_digest}}}};
    Json result{{"outcome", "completed"},
                {"effect_state", "not_applicable"},
                {"error_code", nullptr}};
    bool started = false;
    auto data = cloudflare::execute_resource(request, plan, temp.path, result,
                                             started, &hook);
    NT_REQUIRE(data.dump().find("SENTINEL") == std::string::npos &&
               result.dump().find("SENTINEL") == std::string::npos);
    return Json{{"data", data}, {"result", result}, {"plan", plan}};
  }
};
