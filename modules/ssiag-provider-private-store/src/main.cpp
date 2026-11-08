#include <iostream>
#include <signal.h>
#include <symphony/ssiag/private_channel.hpp>
#include <sys/resource.h>
using namespace symphony::ssiag::native;
namespace {
const char *protocol = "symphony.ssiag.cloudflare-use.v1";
std::string name(const Json &r) {
  need(r.is_string());
  auto s = r.get<std::string>();
  need(
      std::regex_match(s, std::regex("ssiag-native:[a-z0-9][a-z0-9_-]{0,63}")));
  return s.substr(13);
}
Json configuration(const fs::path &root, const std::string &expected) {
  private_dir(root);
  auto c = parse(read(root / "config.json", 32768, true));
  fields(c, {"protocol", "tops_id", "ssiag_socket", "ssiag_uid", "ssiag_digest",
             "consumer_digest", "toolchain_digest"});
  need(c.at("protocol") == "symphony.ssiag.private-store-config.v1" &&
       seal(c, "symphony.ssiag.private-store-config.v1") == expected);
  for (auto k : {"ssiag_digest", "consumer_digest", "toolchain_digest"})
    need(tagged(c.at(k)));
  return c;
}
void save_secret(const fs::path &root, const std::string &ref,
                 const std::string &bytes, const Json &metadata) {
  need(!bytes.empty() && bytes.size() <= 4 * 1024 * 1024);
  private_dir(root / "credentials", true);
  auto target = root / "credentials" / name(ref);
  if (fs::exists(target)) {
    private_dir(target);
    auto prior = parse(read(target / "metadata.json", 32768, true));
    need(metadata.contains("origin_plan_digest") &&
         prior.contains("origin_plan_digest") &&
         prior.at("origin_plan_digest") == metadata.at("origin_plan_digest") &&
         prior.at("purpose") == metadata.at("purpose") &&
         prior.at("account_id") == metadata.at("account_id"));
    return;
  }
  private_dir(target, true);
  put(target / "value", bytes);
  put(target / "metadata.json", metadata.dump() + "\n");
  sync(target);
  sync(root / "credentials");
}
FD lock_store(const fs::path &root, bool exclusive) {
  private_dir(root);
  FD f(::open((root / "lock").c_str(),
              O_RDWR | O_CREAT | O_NOFOLLOW | O_CLOEXEC, 0600));
  struct stat s{};
  need(f.n >= 0 && ::fstat(f.n, &s) == 0 && S_ISREG(s.st_mode) &&
       s.st_nlink == 1 && s.st_uid == ::geteuid() && (s.st_mode & 0077) == 0);
  need(::flock(f.n, (exclusive ? LOCK_EX : LOCK_SH) | LOCK_NB) == 0);
  return f;
}
void seed(const fs::path &root) {
  private_dir(root, true);
  auto lock = lock_store(root, true);
  private_dir(root / "credentials", true);
  auto input =
      parse(symphony::knowledge::engine::read_bounded(std::cin, 32768));
  fields(input, {"reference", "purpose", "account_id", "zone_id", "resource_id",
                 "generation", "expires_unix_ms"});
  need(input.at("purpose") == "cloudflare.api-token" ||
       input.at("purpose") == "cloudflare.tunnel-token");
  need(input.at("generation") == "1");
  need(std::regex_match(input.at("account_id").get<std::string>(),
                        std::regex("[a-f0-9]{32}")));
  for (auto k : {"zone_id", "resource_id"})
    need(input.at(k).is_null() ||
         std::regex_match(input.at(k).get<std::string>(),
                          std::regex("[a-f0-9-]{32,36}")));
  need(std::regex_match(input.at("expires_unix_ms").get<std::string>(),
                        std::regex("[1-9][0-9]{0,18}")) &&
       std::stoll(input.at("expires_unix_ms").get<std::string>()) > now());
  name(input.at("reference"));
  Secret secret;
  std::array<char, 4096> b{};
  for (;;) {
    auto n = ::read(3, b.data(), b.size());
    if (n < 0 && errno == EINTR)
      continue;
    need(n >= 0);
    if (!n)
      break;
    secret.bytes.append(b.data(), std::size_t(n));
    need(secret.bytes.size() <= 16384);
  }
  need(!secret.bytes.empty() &&
       std::all_of(secret.bytes.begin(), secret.bytes.end(),
                   [](unsigned char c) { return c >= 33 && c < 127; }));
  save_secret(root, input.at("reference"), secret.bytes, input);
  std::cout
      << Json{{"reference", input.at("reference")}, {"imported", true}}.dump()
      << '\n';
}
int session(const fs::path &root, const std::string &expected) {
  auto lock = lock_store(root, false);
  auto config = configuration(root, expected);
  auto parent = peer(3);
  need(parent.uid == ::geteuid() && parent.gid == ::getegid() &&
       parent.pid == ::getppid());
  need(config.at("consumer_digest") ==
       hash(read(executable(parent.pid), 512 * 1024 * 1024)));
  timeout(3, 60);
  auto in = parse(frame(3));
  fields(in, {"intent", "credential_generation"});
  auto intent = in.at("intent");
  need(intent.at("protocol") == protocol &&
       intent.at("tops_id") == config.at("tops_id") &&
       intent.at("adapter_digest") == config.at("consumer_digest") &&
       intent.at("toolchain_digest") == config.at("toolchain_digest"));
  auto deadline = std::stoll(intent.at("deadline_unix_ms").get<std::string>());
  need(deadline > now() && deadline - now() <= 60000);
  private_dir(root / "uses", true);
  auto use_id = intent.at("request_id").get<std::string>();
  need(std::regex_match(use_id, std::regex("[a-f0-9-]{36}")));
  auto attempt = root / "uses" / (use_id + ".json");
  need(!fs::exists(attempt));
  auto request_hash = seal(intent, protocol);
  put(attempt,
      Json{{"request_digest", request_hash}, {"stage", "prepared"}}.dump());
  auto src = root / "credentials" / name(intent.at("credential_ref"));
  private_dir(src);
  auto metadata = parse(read(src / "metadata.json", 32768, true));
  need(metadata.at("generation") == in.at("credential_generation") &&
       std::stoll(metadata.at("expires_unix_ms").get<std::string>()) >
           deadline &&
       metadata.at("account_id") == intent.at("account_id"));
  const bool connector = intent.at("operation") == "connector.start";
  need(metadata.at("purpose") ==
       (connector ? "cloudflare.tunnel-token" : "cloudflare.api-token"));
  need(metadata.at("zone_id").is_null() ||
       metadata.at("zone_id") == intent.at("zone_id"));
  need(metadata.at("resource_id").is_null() ||
       metadata.at("resource_id") == intent.at("resource_id"));
  if (!intent.at("result_ref").is_null()) {
    auto target = root / "credentials" / name(intent.at("result_ref"));
    if (fs::exists(target)) {
      auto prior = parse(read(target / "metadata.json", 32768, true));
      need(prior.contains("origin_plan_digest") &&
           prior.at("origin_plan_digest") == intent.at("plan_digest"));
    }
  }
  auto service =
      connect(config.at("ssiag_socket").get<std::string>(),
              uid_t(std::stoul(config.at("ssiag_uid").get<std::string>())),
              config.at("ssiag_digest"));
  auto body = intent.dump();
  write_all(service.n,
            "POST /v1/cloudflare/use HTTP/1.1\r\nHost: "
            "localhost\r\nContent-Type: application/json\r\nContent-Length: " +
                std::to_string(body.size()) +
                "\r\nConnection: Upgrade\r\nUpgrade: "
                "symphony-cloudflare-use-v1\r\n\r\n" +
                body);
  std::string header;
  while (!header.ends_with("\r\n\r\n")) {
    char c;
    exact_read(service.n, &c, 1);
    header += c;
    need(header.size() < 4096);
  }
  need(header.starts_with("HTTP/1.1 101 "));
  auto grant = parse(frame(service.n));
  need(grant.at("protocol") == protocol &&
       grant.at("request_digest") == request_hash &&
       grant.at("uid") == std::to_string(::geteuid()) &&
       grant.at("receipt").at("disposition") == "committed");
  deadline = std::stoll(grant.at("deadline_unix_ms").get<std::string>());
  need(deadline > now());
  atomic(attempt,
         Json{{"request_digest", request_hash}, {"stage", "admitted"}});
  Secret token;
  token.bytes = read(src / "value", 16384, true);
  need(!token.bytes.empty() &&
       std::all_of(token.bytes.begin(), token.bytes.end(),
                   [](unsigned char c) { return c >= 33 && c < 127; }));
  atomic(attempt,
         Json{{"request_digest", request_hash}, {"stage", "dispatch_started"}});
  frame(3, grant.dump());
  frame(3, token.bytes);
  wipe(token.bytes);
  auto finish = parse(frame(3));
  fields(finish, {"request_digest", "outcome", "sensitive_result"});
  need(finish.at("request_digest") == request_hash && now() < deadline);
  auto outcome = finish.at("outcome").get<std::string>();
  need(outcome == "completed" || outcome == "failed" || outcome == "unknown");
  Json result_ref = nullptr;
  if (finish.at("sensitive_result") == true) {
    need(outcome == "completed" && !intent.at("result_ref").is_null() &&
         !connector);
    Secret result;
    result.bytes = frame(3, 4 * 1024 * 1024);
    auto ref = intent.at("result_ref").get<std::string>();
    auto op = intent.at("operation").get<std::string>();
    need(op == "tunnels.token.get" || op == "tunnels.create" ||
         op == "tunnels.get");
    save_secret(root, ref, result.bytes,
                Json{{"reference", ref},
                     {"purpose", op == "tunnels.token.get"
                                     ? "cloudflare.tunnel-token"
                                     : "cloudflare.sensitive-result"},
                     {"account_id", intent.at("account_id")},
                     {"zone_id", intent.at("zone_id")},
                     {"resource_id", intent.at("resource_id")},
                     {"generation", "1"},
                     {"origin_plan_digest", intent.at("plan_digest")},
                     {"expires_unix_ms", metadata.at("expires_unix_ms")}});
    result_ref = ref;
  }
  atomic(attempt, Json{{"request_digest", request_hash},
                       {"stage", "observed"},
                       {"outcome", outcome},
                       {"result_ref", result_ref}});
  frame(service.n, Json{{"protocol", protocol},
                        {"request_digest", request_hash},
                        {"outcome", outcome}}
                       .dump());
  auto completion = parse(frame(service.n));
  need(completion.at("request_digest") == request_hash &&
       completion.at("outcome") == outcome &&
       completion.at("receipt").at("disposition") == "committed");
  atomic(attempt, Json{{"request_digest", request_hash},
                       {"stage", "settled"},
                       {"outcome", outcome},
                       {"result_ref", result_ref}});
  frame(3, Json{{"result_ref", result_ref}, {"completion", completion}}.dump());
  return 0;
}
} // namespace

int main(int argc, char **argv) {
  ::signal(SIGPIPE, SIG_IGN);
  try {
    user();
    rlimit core{0, 0};
    need(::setrlimit(RLIMIT_CORE, &core) == 0);
    if (argc == 3 && std::string(argv[1]) == "--configure") {
      fs::path root = argv[2];
      private_dir(root, true);
      auto lock = lock_store(root, true);
      auto input =
          parse(symphony::knowledge::engine::read_bounded(std::cin, 32768));
      fields(input, {"expected_digest", "configuration"});
      auto path = root / "config.json";
      need(fs::exists(path) ? input.at("expected_digest") ==
                                  seal(parse(read(path, 32768, true)),
                                       "symphony.ssiag.private-store-config.v1")
                            : input.at("expected_digest").is_null());
      auto cfg = input.at("configuration");
      fields(cfg, {"protocol", "tops_id", "ssiag_socket", "ssiag_uid",
                   "ssiag_digest", "consumer_digest", "toolchain_digest"});
      need(cfg.at("protocol") == "symphony.ssiag.private-store-config.v1");
      for (auto key : {"ssiag_digest", "consumer_digest", "toolchain_digest"})
        need(tagged(cfg.at(key)));
      atomic(path, cfg);
      std::cout << Json{{"config_digest",
                         seal(cfg, "symphony.ssiag.private-store-config.v1")}}
                       .dump()
                << '\n';
      return 0;
    }
    if (argc == 3 && (std::string(argv[1]) == "--inspect" ||
                      std::string(argv[1]) == "--revoke")) {
      fs::path root = argv[2];
      bool revoke = std::string(argv[1]) == "--revoke";
      auto lock = lock_store(root, revoke);
      auto input =
          parse(symphony::knowledge::engine::read_bounded(std::cin, 32768));
      fields(input, {"reference", "generation"});
      auto path =
          root / "credentials" / name(input.at("reference")) / "metadata.json";
      auto metadata = parse(read(path, 32768, true));
      need(metadata.at("generation") == input.at("generation"));
      if (revoke) {
        auto n = std::stoull(metadata.at("generation").get<std::string>());
        need(n < 18446744073709551615ULL);
        metadata["generation"] = std::to_string(n + 1);
        metadata["expires_unix_ms"] = "0";
        atomic(path, metadata);
      }
      std::cout << metadata.dump() << '\n';
      return 0;
    }
    if (argc == 3 && std::string(argv[1]) == "--seed") {
      seed(fs::path(argv[2]));
      return 0;
    }
    need(argc == 4 && std::string(argv[1]) == "--session");
    return session(fs::path(argv[2]), argv[3]);
  } catch (...) {
    std::cerr << "Native credential operation unavailable\n";
    return 5;
  }
}
