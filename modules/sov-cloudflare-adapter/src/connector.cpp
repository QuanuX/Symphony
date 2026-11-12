#include "connector.hpp"
#include "execution.hpp"
#include <signal.h>
#include <sys/wait.h>
namespace cloudflare {
namespace n = symphony::ssiag::native;
namespace {
const char *state_protocol = "symphony.sov.cloudflare-connector-state.v1";
const char *plan_protocol = "symphony.sov.cloudflare-connector-plan.v1";
Json initial() {
  return Json{{"protocol", state_protocol}, {"generation", "0"},
              {"selected", nullptr},        {"packages", Json::array()},
              {"configuration", nullptr},   {"run_id", nullptr}};
}
Json state(const fs::path &root) {
  auto file = root / "state.json";
  auto s =
      fs::exists(file) ? parse(n::read(file, frame_limit, true)) : initial();
  validate(s, owner().at("connector_state_schema"));
  return s;
}
fs::path binary(const fs::path &root, const std::string &id) {
  n::need(n::tagged(id));
  auto path = root / "packages" / (id.substr(7) + "-cloudflared");
  n::need(n::hash(n::read(path, 128 * 1024 * 1024, true)) == id);
  return path;
}
bool present(const Json &s, const Json &id) {
  return std::find(s.at("packages").begin(), s.at("packages").end(), id) !=
         s.at("packages").end();
}
Json runtime(const fs::path &root, const Json &id) {
  if (id.is_null())
    return nullptr;
  auto run = root / "runs" / id.get<std::string>();
  auto r = parse(n::read(run / "runtime.json", 32768, true));
  n::fields(r, {"instance_digest", "supervisor_digest", "pid", "state"});
  n::need(n::tagged(r.at("instance_digest")) &&
          n::tagged(r.at("supervisor_digest")));
  n::need(r.at("state") == "running" || r.at("state") == "stopped" ||
          r.at("state") == "failed");
  return r;
}
Json control(const fs::path &root, const Json &run_id, const std::string &op) {
  auto r = runtime(root, run_id);
  n::need(!r.is_null());
  if (r.at("state") != "running")
    return r;
  auto c = n::connect(root / (run_id.get<std::string>() + ".sock"), ::geteuid(),
                      r.at("supervisor_digest"));
  n::timeout(c.n, 10);
  n::frame(c.n,
           Json{{"operation", op}, {"instance_digest", r.at("instance_digest")}}
               .dump());
  auto reply = parse(n::frame(c.n));
  n::need(reply.at("instance_digest") == r.at("instance_digest") &&
          reply.at("supervisor_digest") == r.at("supervisor_digest") &&
          reply.at("pid") == r.at("pid"));
  n::fields(reply, {"instance_digest", "supervisor_digest", "pid", "state"});
  n::need(reply.at("state") == "running" || reply.at("state") == "stopped" ||
          reply.at("state") == "failed");
  return reply;
}
void stopped(const fs::path &root, const Json &s) {
  if (!s.at("run_id").is_null())
    n::need(control(root, s.at("run_id"), "status").at("state") != "running");
}
// Native fixture seam only; installed command frames cannot select it.
void qualified(const Json &id, bool fixture = false) {
  if (fixture)
    return;
  auto ids = owner().at("qualified_connector_binaries");
  n::need(std::find(ids.begin(), ids.end(), id) != ids.end());
}
Json load_journal(const fs::path &path, const std::string &d) {
  auto j = parse(n::read(path, 32768, true));
  n::fields(j, {"plan_digest", "stage", "effect_state"});
  n::need(j.at("plan_digest") == d);
  n::need(j.at("stage") == "prepared" || j.at("stage") == "dispatch_started" ||
          j.at("stage") == "settled");
  n::need(j.at("effect_state") == "not_applied" ||
          j.at("effect_state") == "applied_verified" ||
          j.at("effect_state") == "unresolved");
  return j;
}
} // namespace
void connector_barrier(const fs::path &store) {
  auto folder = store / "connector/attempts";
  if (!fs::exists(folder))
    return;
  for (auto f : fs::directory_iterator(folder)) {
    if (f.path().extension() != ".json")
      continue;
    auto j = load_journal(f.path(), "sha256:" + f.path().stem().string());
    n::need(j.at("stage") == "settled" && j.at("effect_state") != "unresolved");
  }
}
Json connector_admin(const Json &request, const fs::path &executable,
                     Json &result, bool &started, bool fixture_binary) {
  n::user();
  const auto &in = request.at("input");
  auto root = fs::path(in.at("store").get<std::string>()) / "connector";
  n::private_dir(root, true);
  for (auto sub : {"plans", "attempts", "packages", "runs"})
    n::private_dir(root / sub, true);
  auto s = state(root);
  auto op = request.at("operation").get<std::string>();
  const bool installation = op.starts_with("connector.installation.");
  auto status = [&] {
    return Json{{"state", s},
                {"state_digest", digest(s, state_protocol)},
                {"runtime", s.at("run_id").is_null()
                                ? Json(nullptr)
                                : control(root, s.at("run_id"), "status")},
                {"provider_readiness", "not_proven"}};
  };
  if (op.ends_with(".list") || op.ends_with(".inspect")) {
    for (auto id : s.at("packages"))
      static_cast<void>(binary(root, id));
    return status();
  }
  if (op.ends_with(".plan")) {
    connector_barrier(root.parent_path());
    n::need(in.at("expected_state") == digest(s, state_protocol));
    auto action = in.at("action").get<std::string>();
    auto after = s;
    auto generation = std::stoull(s.at("generation").get<std::string>());
    n::need(generation < 999999999999999999ULL);
    after["generation"] = std::to_string(generation + 1);
    if (installation) {
      auto id = in.at("package_digest");
      n::need(n::tagged(id));
      qualified(id, fixture_binary);
      if (action == "install" || action == "update") {
        n::need(!in.at("source").is_null() && !present(s, id));
        n::need(id == n::hash(n::read(in.at("source").get<std::string>(),
                                      128 * 1024 * 1024)));
        after["packages"].push_back(id);
      } else {
        n::need(in.at("source").is_null() && present(s, id));
        static_cast<void>(binary(root, id));
      }
      if (action == "select" || action == "update" || action == "rollback")
        after["selected"] = id;
      if (action == "remove") {
        n::need(s.at("selected") != id &&
                (s.at("configuration").is_null() ||
                 s.at("configuration").at("package_digest") != id));
        Json packages = Json::array();
        for (auto p : s.at("packages"))
          if (p != id)
            packages.push_back(p);
        after["packages"] = packages;
      }
    } else {
      if (action == "configure" || action == "update") {
        stopped(root, s);
        auto cfg = in.at("configuration");
        n::need(!cfg.is_null() && present(s, cfg.at("package_digest")) &&
                s.at("selected") == cfg.at("package_digest"));
        qualified(cfg.at("package_digest"), fixture_binary);
        static_cast<void>(binary(root, cfg.at("package_digest")));
        auto binding = execution_binding(cfg.at("execution"));
        n::need(binding.at("config").at("consumer_digest") ==
                    request.at("installation_ref").at("executable_digest") &&
                binding.at("config").at("toolchain_digest") ==
                    cfg.at("package_digest") &&
                cfg.at("execution").at("result_ref").is_null());
        if (action == "configure")
          n::need(s.at("configuration").is_null());
        else
          n::need(!s.at("configuration").is_null());
        after["configuration"] = cfg;
        after["run_id"] = nullptr;
      } else {
        n::need(in.at("configuration").is_null() &&
                !s.at("configuration").is_null());
        if (action == "start") {
          stopped(root, s);
          after["run_id"] = n::uuid();
          auto socket =
              root / (after.at("run_id").get<std::string>() + ".sock");
          n::need(socket.string().size() < sizeof(sockaddr_un::sun_path));
        } else if (action == "stop") {
          n::need(!s.at("run_id").is_null());
        } else if (action == "retire") {
          stopped(root, s);
          after["configuration"] = nullptr;
          after["run_id"] = nullptr;
        } else
          n::need(false);
      }
    }
    Json p{{"protocol", plan_protocol},
           {"uid", std::to_string(::geteuid())},
           {"request_id", request.at("request_id")},
           {"descriptor_ref", request.at("descriptor_ref")},
           {"installation_ref", request.at("installation_ref")},
           {"input", in},
           {"family", installation ? "installation" : "instance"},
           {"before", s},
           {"after", after},
           {"expires_unix_ms", std::to_string(n::now() + 3600000)}};
    auto d = digest(p, plan_protocol);
    auto path = root / "plans" / (d.substr(7) + ".json");
    if (!fs::exists(path))
      n::put(path, p.dump());
    return Json{
        {"plan", p}, {"plan_digest", d}, {"provider_readiness", "not_proven"}};
  }
  auto d = in.at("plan_digest").get<std::string>();
  n::need(request.at("plan_ref") == d);
  auto p = parse(
      n::read(root / "plans" / (d.substr(7) + ".json"), frame_limit, true));
  // Keep the declaration alive on compilers without range-for temporary extension.
  const auto declaration = owner();
  for (const auto &descriptor : declaration.at("operations"))
    if (descriptor.at("operation") ==
        std::string("connector.") +
            (installation ? "installation" : "instance") + ".plan")
      validate(p, descriptor.at("data_schema").at("properties").at("plan"));
  n::need(digest(p, plan_protocol) == d && p.at("protocol") == plan_protocol &&
          p.at("uid") == std::to_string(::geteuid()) &&
          p.at("input").at("store") == in.at("store") &&
          p.at("descriptor_ref") == request.at("descriptor_ref") &&
          p.at("installation_ref") == request.at("installation_ref") &&
          p.at("family") == (installation ? "installation" : "instance"));
  auto jp = root / "attempts" / (d.substr(7) + ".json");
  auto j = fs::exists(jp) ? load_journal(jp, d) : Json(nullptr);
  auto action = p.at("input").at("action").get<std::string>();
  if (op.ends_with(".status"))
    return Json{{"plan_digest", d}, {"attempt", j}, {"current", status()}};
  if (op.ends_with(".reconcile")) {
    n::need(!j.is_null());
    if (s == p.at("after")) {
      if (action == "start")
        n::need(control(root, s.at("run_id"), "status").at("state") ==
                "running");
      if (action == "stop")
        stopped(root, s);
      j["effect_state"] = "applied_verified";
    } else if (s == p.at("before") && j.at("stage") == "prepared")
      j["effect_state"] = "not_applied";
    else { // Never launch or repeat a provider action during reconciliation.
      n::need(s == p.at("before"));
      if (action == "start") {
        auto r = control(root, p.at("after").at("run_id"), "status");
        n::need(r.at("state") == "running");
      } else if (action == "stop")
        stopped(root, s);
      else if (installation && (action == "install" || action == "update"))
        static_cast<void>(binary(root, p.at("input").at("package_digest")));
      else if (action == "remove")
        n::need(!fs::exists(
            root / "packages" /
            (p.at("input").at("package_digest").get<std::string>().substr(7) +
             "-cloudflared")));
      else
        n::need(false);
      s = p.at("after");
      n::atomic(root / "state.json", s);
      j["effect_state"] = "applied_verified";
    }
    j["stage"] = "settled";
    started = true;
    n::atomic(jp, j);
  } else {
    connector_barrier(root.parent_path());
    resource_barrier(root.parent_path());
    n::need(j.is_null() && s == p.at("before") &&
            std::stoll(p.at("expires_unix_ms").get<std::string>()) > n::now());
    j = Json{{"plan_digest", d},
             {"stage", "prepared"},
             {"effect_state", "not_applied"}};
    n::atomic(jp, j);
    // Repeat exact file/config checks before durable dispatch.
    if (installation && (action == "install" || action == "update"))
      n::need(p.at("input").at("package_digest") ==
              n::hash(n::read(p.at("input").at("source").get<std::string>(),
                              128 * 1024 * 1024)));
    j["stage"] = "dispatch_started";
    j["effect_state"] = "unresolved";
    n::atomic(jp, j);
    started = true;
    if (installation) {
      auto id = p.at("input").at("package_digest").get<std::string>();
      auto path = root / "packages" / (id.substr(7) + "-cloudflared");
      if (action == "install" || action == "update") {
        n::put(path,
               n::read(p.at("input").at("source").get<std::string>(),
                       128 * 1024 * 1024),
               0700);
        static_cast<void>(binary(root, id));
      } else if (action == "remove") {
        static_cast<void>(binary(root, id));
        n::need(::unlink(path.c_str()) == 0);
        n::sync(path.parent_path());
      }
    } else if (action == "start") {
      auto cfg = p.at("after").at("configuration");
      static_cast<void>(binary(root, cfg.at("package_digest")));
      auto run = root / "runs" / p.at("after").at("run_id").get<std::string>();
      n::private_dir(run, true);
      auto supervisor = run / "supervisor";
      n::put(supervisor, n::read(executable, 512 * 1024 * 1024), 0700);
      n::need(request.at("installation_ref").at("executable_digest") ==
              n::hash(n::read(supervisor, 512 * 1024 * 1024, true)));
      int pair[2];
      n::need(::socketpair(AF_UNIX, SOCK_STREAM, 0, pair) == 0);
      n::FD parent(pair[0]), child(pair[1]);
      n::timeout(parent.n, 60);
      auto pid = ::fork();
      n::need(pid >= 0);
      if (pid == 0) {
        if (::setsid() < 0 || ::dup2(child.n, 3) < 0)
          _exit(126);
        n::close_others(3);
        int nil = ::open("/dev/null", O_RDWR);
        ::dup2(nil, 0);
        ::dup2(nil, 1);
        ::dup2(nil, 2);
        if (nil > 3)
          ::close(nil);
        char arg[] = "--connector-supervise", pathenv[] = "PATH=/usr/bin:/bin";
        char *av[]{const_cast<char *>(supervisor.c_str()), arg, nullptr};
        char *env[]{pathenv, nullptr};
        ::execve(av[0], av, env);
        _exit(127);
      }
      ::close(child.n);
      child.n = -1;
      n::frame(parent.n,
               Json{{"plan", p}, {"plan_digest", d}, {"root", root.string()}}
                   .dump());
      auto ack = parse(n::frame(parent.n));
      n::need(ack.at("state") == "running");
      result["authority_ref"] = ack.at("authority_ref");
      result["audit_ref"] = ack.at("audit_ref");
    } else if (action == "stop")
      n::need(control(root, s.at("run_id"), "stop").at("state") == "stopped");
    else
      stopped(root, s);
    s = p.at("after");
    n::atomic(root / "state.json", s);
    j["stage"] = "settled";
    j["effect_state"] = "applied_verified";
    n::atomic(jp, j);
  }
  result["effect_state"] = j.at("effect_state");
  return Json{{"plan_digest", d}, {"attempt", j}, {"current", status()}};
}
int connector_supervise(bool fixture_binary) {
  n::user();
  ::signal(SIGPIPE, SIG_IGN);
  auto peer = n::peer(3);
  n::need(peer.pid == ::getppid() && peer.uid == ::geteuid() &&
          peer.gid == ::getegid());
  auto self = n::hash(n::read(n::executable(::getpid()), 512 * 1024 * 1024));
  n::need(self == n::hash(n::read(n::executable(peer.pid), 512 * 1024 * 1024)));
  n::timeout(3, 60);
  auto message = parse(n::frame(3));
  n::fields(message, {"plan", "plan_digest", "root"});
  auto p = message.at("plan");
  auto d = message.at("plan_digest").get<std::string>();
  n::need(digest(p, plan_protocol) == d &&
          p.at("installation_ref").at("executable_digest") == self &&
          p.at("input").at("action") == "start" &&
          p.at("uid") == std::to_string(::geteuid()));
  auto root = fs::path(message.at("root").get<std::string>());
  n::need(root ==
          fs::path(p.at("input").at("store").get<std::string>()) / "connector");
  n::private_dir(root);
  auto cfg = p.at("after").at("configuration");
  qualified(cfg.at("package_digest"), fixture_binary);
  auto exe = binary(root, cfg.at("package_digest"));
  auto run = root / "runs" / p.at("after").at("run_id").get<std::string>();
  n::private_dir(run);
  auto binding = execution_binding(cfg.at("execution"));
  Json intent{{"protocol", "symphony.ssiag.cloudflare-use.v1"},
              {"request_id", n::uuid()},
              {"tops_id", binding.at("config").at("tops_id")},
              {"plan_digest", d},
              {"adapter_digest", self},
              {"toolchain_digest", cfg.at("package_digest")},
              {"credential_ref", cfg.at("credential_ref")},
              {"result_ref", nullptr},
              {"operation", "connector.start"},
              {"account_id", cfg.at("account_id")},
              {"zone_id", nullptr},
              {"resource_id", cfg.at("tunnel_id")},
              {"deadline_unix_ms", std::to_string(n::now() + 55000)}};
  CredentialSession credential(cfg.at("execution"), intent);
  // Empty private config prevents ambient YAML/certificate discovery; managed
  // Tunnel configuration remains provider-owned. Metrics binds loopback only.
  n::put(run / "config.yml", "{}\n");
  std::vector<std::string> args{exe.string(),
                                "tunnel",
                                "--config",
                                (run / "config.yml").string(),
                                "--no-autoupdate",
                                "--metrics",
                                "127.0.0.1:0",
                                "--loglevel",
                                "error",
                                "run",
                                cfg.at("tunnel_id").get<std::string>()};
  std::vector<std::string> environment{"PATH=/usr/bin:/bin",
                                       "HOME=" + run.string(),
                                       "XDG_CONFIG_HOME=" + run.string(),
                                       "TMPDIR=" + run.string(),
                                       "TUNNEL_TOKEN=" +
                                           credential.secret.bytes,
                                       "NO_AUTOUPDATE=true"};
  std::vector<char *> av, ev;
  for (auto &x : args)
    av.push_back(x.data());
  av.push_back(nullptr);
  for (auto &x : environment)
    ev.push_back(x.data());
  ev.push_back(nullptr);
  n::FD listener(::socket(AF_UNIX, SOCK_STREAM, 0));
  sockaddr_un a{};
  a.sun_family = AF_UNIX;
  auto socket =
      (root / (p.at("after").at("run_id").get<std::string>() + ".sock"))
          .string();
  n::need(socket.size() < sizeof(a.sun_path));
  std::copy(socket.begin(), socket.end(), a.sun_path);
  n::need(::bind(listener.n, reinterpret_cast<sockaddr *>(&a), sizeof(a)) ==
              0 &&
          ::chmod(socket.c_str(), 0600) == 0 && ::listen(listener.n, 4) == 0);
  n::need(
      n::now() <
      std::stoll(credential.grant.at("deadline_unix_ms").get<std::string>()));
  auto child = ::fork();
  n::need(child >= 0);
  if (child == 0) {
    ::setpgid(0, 0);
    n::close_others();
    if (::chdir(run.c_str()) != 0)
      _exit(126);
    ::execve(av[0], av.data(), ev.data());
    _exit(127);
  }
  ::setpgid(child, child);
  for (auto &x : environment)
    n::wipe(x);
  bool reaped = false;
  int status{};
  struct Cleanup {
    pid_t pid;
    bool &reaped;
    ~Cleanup() {
      if (!reaped) {
        ::kill(-pid, SIGKILL);
        int status{};
        while (::waitpid(pid, &status, 0) < 0 && errno == EINTR) {
        }
      }
    }
  } cleanup{child, reaped};
  auto until = n::now() + 250;
  while (n::now() < until) {
    if (::waitpid(child, &status, WNOHANG) == child) {
      reaped = true;
      break;
    }
    ::poll(nullptr, 0, 10);
  }
  n::need(!reaped);
  auto completion = credential.finish("completed");
  Json record{{"instance_digest", d},
              {"supervisor_digest", self},
              {"pid", std::to_string(::getpid())},
              {"state", "running"}};
  n::atomic(run / "runtime.json", record);
  n::frame(
      3,
      Json{{"state", "running"},
           {"authority_ref",
            "ssiag:" + credential.grant.at("policy_digest").get<std::string>()},
           {"audit_ref", completion.at("completion")
                             .at("receipt")
                             .at("commit")
                             .at("event_digest")}}
          .dump());
  ::close(3);
  while (!reaped) {
    if (::waitpid(child, &status, WNOHANG) == child) {
      reaped = true;
      record["state"] = "failed";
      n::atomic(run / "runtime.json", record);
      break;
    }
    pollfd poll{listener.n, POLLIN, 0};
    if (::poll(&poll, 1, 100) <= 0)
      continue;
    n::FD client(::accept(listener.n, nullptr, nullptr));
    try {
      auto caller = n::peer(client.n);
      n::need(caller.uid == ::geteuid() && caller.gid == ::getegid());
      n::timeout(client.n, 5);
      auto cmd = parse(n::frame(client.n));
      n::fields(cmd, {"operation", "instance_digest"});
      n::need(
          cmd.at("instance_digest") == d &&
          (cmd.at("operation") == "status" || cmd.at("operation") == "stop"));
      if (cmd.at("operation") == "stop") {
        ::kill(-child, SIGTERM);
        auto end = n::now() + 3000;
        while (n::now() < end) {
          if (::waitpid(child, &status, WNOHANG) == child) {
            reaped = true;
            break;
          }
          ::poll(nullptr, 0, 20);
        }
        if (!reaped) {
          ::kill(-child, SIGKILL);
          while (::waitpid(child, &status, 0) < 0 && errno == EINTR) {
          }
          reaped = true;
        }
        record["state"] = "stopped";
        n::atomic(run / "runtime.json", record);
      }
      n::frame(client.n, record.dump());
    } catch (...) {
    }
  }
  ::unlink(socket.c_str());
  return 0;
}
} // namespace cloudflare
