#include "access.hpp"
#include "native_test.hpp"
#include <iostream>
using namespace native_test;
namespace a = cloudflare::access;
namespace n = a::n;
std::string b64(const std::string &b) {
  const char *abc =
      "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";
  std::string out;
  unsigned v = 0, bits = 0;
  for (unsigned char c : b) {
    v = (v << 8) | c;
    bits += 8;
    while (bits >= 6) {
      bits -= 6;
      out += abc[(v >> bits) & 63];
    }
  }
  if (bits)
    out += abc[(v << (6 - bits)) & 63];
  return out;
}
int test_access_native_sessions(int argc, char **argv) {
  try {
    NT_REQUIRE(argc == 2);
    n::user();
    TempDir temp;
    auto root = fs::canonical(temp.path);
    ::chmod(root.c_str(), 0700);
    auto child = fs::canonical(argv[1]);
    a::Runtime runtime{
        child, child, "sha256:" + std::string(64, 'a'), n::hash(n::read(child)),
        "",    ""};
    Json id{{"audience", std::string(64, 'b')},
            {"issuer", "https://test.cloudflareaccess.com"},
            {"subject", "fixture-user"}};
    auto payload = Json{{"aud", Json::array({id.at("audience")})},
                        {"iss", id.at("issuer")},
                        {"sub", id.at("subject")},
                        {"exp", n::now() / 1000 + 600}};
    auto token = [&](Json p) {
      return b64(Json{{"alg", "RS256"}}.dump()) + "." + b64(p.dump()) +
             ".fixture-signature";
    };
    auto jwt = token(payload);
    Json cfg{{"audience", id.at("audience")}, {"token", jwt}};
    n::atomic(root / "fixture.json", cfg);
    Json binding{{"application", "access.example.invalid"},
                 {"session_ref", "session:test"},
                 {"session_directory", root.string()},
                 {"profile_digest", "sha256:" + std::string(64, 'c')},
                 {"endpoint_ref", "sov:ssh:v2:sha256:" + std::string(64, 'd')}};
    auto request = [&](std::string op, Json intent = nullptr,
                       Json plan = nullptr) {
      return Json{
          {"protocol", "symphony.sov.transport-session-native-request.v1"},
          {"request_id", n::uuid()},
          {"operation", op},
          {"binding", binding},
          {"intent", intent},
          {"plan_digest", plan},
          {"deadline_unix_ms", std::to_string(n::now() + 60000)}};
    };
    struct Capture {
      n::FD out{::dup(STDOUT_FILENO)}, err{::dup(STDERR_FILENO)}, file;
      explicit Capture(const fs::path &p)
          : file(::open(p.c_str(), O_RDWR | O_CREAT | O_TRUNC | O_CLOEXEC,
                        0600)) {
        n::need(out.n >= 0 && err.n >= 0 && file.n >= 0);
        n::need(::dup2(file.n, STDOUT_FILENO) >= 0 &&
                ::dup2(file.n, STDERR_FILENO) >= 0);
      }
      ~Capture() {
        ::dup2(out.n, STDOUT_FILENO);
        ::dup2(err.n, STDERR_FILENO);
      }
    };
    auto call = [&](Json q) {
      Json r;
      {
        Capture capture(root / "captured");
        r = a::session(runtime, q);
      }
      NT_REQUIRE(n::read(root / "captured", 1024, true).empty());
      NT_REQUIRE(r.dump().find(jwt) == std::string::npos &&
                 r.dump().find("SECRET") == std::string::npos);
      return r;
    };
    auto plan = [&](std::string op) {
      return call(request("plan",
                          {{"action", op},
                           {"identity", op == "clear" ? Json(nullptr) : id}}))
          .at("plan_digest");
    };
    auto reject = [&](auto fn) {
      bool caught = false;
      try {
        fn();
      } catch (...) {
        caught = true;
      }
      NT_REQUIRE(caught);
    };
    NT_REQUIRE(call(request("inspect")).at("state") == "interaction_required");
    reject([&] {
      a::connect(runtime, "access.example.invalid", root, "session:test");
    });
    reject([&] { plan("renew"); });
    for (auto key : {"aud", "iss", "sub"}) {
      auto bad = payload;
      bad[key] = "wrong";
      reject([&] { a::claims(token(bad), id, n::now()); });
    }
    auto future = payload;
    future["nbf"] = std::uint64_t{18446744073709551615ULL};
    reject([&] { a::claims(token(future), id, n::now()); });
    auto expired = payload;
    expired["exp"] = n::now() / 1000 - 1;
    reject([&] { a::claims(token(expired), id, n::now()); });
    auto bad = request("inspect");
    bad["deadline_unix_ms"] = "1junk";
    reject([&] { call(bad); });
    bad = request("inspect");
    bad["request_id"] = std::string(36, '-');
    reject([&] { call(bad); });
    auto auth = plan("authenticate");
    auto result = call(request("authenticate", nullptr, auth));
    NT_REQUIRE(result.at("completed") == true &&
               result.at("state") == "locally_usable" &&
               result.at("provider_authorization") == "not_proven");
    reject([&] { call(request("authenticate", nullptr, auth)); });
    ::setenv("CF_ACCESS_TOKEN", "SECRET-AMBIENT", 1);
    ::setenv("HTTPS_PROXY", "SECRET-PROXY", 1);
    NT_REQUIRE(a::connect(runtime, "access.example.invalid", root,
                          "session:test") == 0);
    reject([&] {
      a::connect(runtime, "other.example.invalid", root, "session:test");
    });
    cfg["deny"] = true;
    n::atomic(root / "fixture.json", cfg);
    NT_REQUIRE(a::connect(runtime, "access.example.invalid", root,
                          "session:test") == 9);
    cfg.erase("deny");
    cfg["fail"] = true;
    n::atomic(root / "fixture.json", cfg);
    result = call(request("renew", nullptr, plan("renew")));
    NT_REQUIRE(result.at("code") == "session.authentication_failed" &&
               result.at("state") == "locally_usable");
    cfg.erase("fail");
    cfg["delay"] = true;
    n::atomic(root / "fixture.json", cfg);
    auto slow = request("renew", nullptr, plan("renew"));
    slow["deadline_unix_ms"] = std::to_string(n::now() + 150);
    auto start = n::now();
    NT_REQUIRE(call(slow).at("completed") == false && n::now() - start < 3000);
    cfg.erase("delay");
    payload["exp"] = n::now() / 1000 + 1200;
    cfg["token"] = token(payload);
    n::atomic(root / "fixture.json", cfg);
    NT_REQUIRE(call(request("renew", nullptr, plan("renew"))).at("completed") ==
               true);
    // Removal/expiry readiness never starts a child; private metadata does not
    // prove provider access.
    auto ready = n::parse(n::read(root / "readiness.json"));
    auto correct = ready;
    ready["expires_at"] = "invalid";
    n::atomic(root / "readiness.json", ready);
    NT_REQUIRE(call(request("inspect")).at("state") == "interaction_required");
    n::atomic(root / "readiness.json", correct);
    ::unlink((root / "readiness.json").c_str());
    NT_REQUIRE(call(request("inspect")).at("state") == "interaction_required");
    reject([&] {
      a::connect(runtime, "access.example.invalid", root, "session:test");
    });
    auto stale = plan("clear");
    auto state = n::parse(n::read(root / "state.json"));
    state["expires_unix_ms"] = "1";
    n::atomic(root / "state.json", state);
    reject([&] { call(request("clear", nullptr, stale)); });
    NT_REQUIRE(call(request("clear", nullptr, plan("clear"))).at("code") ==
               "session.cleared");
    NT_REQUIRE(!fs::exists(root / "token") && !fs::exists(root / "state.json"));
    for (auto &e : fs::directory_iterator(root))
      NT_REQUIRE(!e.path().filename().string().starts_with(".access-"));
    // A special-file token must not block local inspection.
    NT_REQUIRE(::mkfifo((root / "state.json").c_str(), 0600) == 0);
    reject([&] { call(request("inspect")); });
    ::unlink((root / "state.json").c_str());
    std::cout
        << "Access native fixtures passed; no live provider qualification\n";
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}

int main(int argc, char **argv) {
  return test_access_native_sessions(argc, argv);
}
