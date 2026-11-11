#include "access.hpp"
#include <chrono>
#include <csignal>
#include <ctime>
#include <iostream>
#include <limits>
#include <sys/wait.h>
#include <thread>
namespace cloudflare::access {
namespace {
constexpr auto request_protocol =
    "symphony.sov.transport-session-native-request.v1";
constexpr auto plan_protocol = "symphony.sov.transport-session-plan.v1";
std::string text(const Json &j) {
  n::need(j.is_string());
  return j.get<std::string>();
}
bool match(const std::string &s, const char *re) {
  return std::regex_match(s, std::regex(re));
}
void hostname(const std::string &s) {
  n::need(match(s, "[a-z0-9][a-z0-9.-]{0,251}[a-z0-9]") &&
          s.find("..") == std::string::npos);
}
void protected_path(const fs::path &path) {
  n::need(path.is_absolute() && path.lexically_normal() == path);
  for (auto p = path;; p = p.parent_path()) {
    struct stat s{};
    n::need(::lstat(p.c_str(), &s) == 0 && s.st_uid == 0 &&
            (s.st_mode & 0022) == 0 && !S_ISLNK(s.st_mode));
    n::need(p == path ? S_ISREG(s.st_mode) : S_ISDIR(s.st_mode));
    if (p == "/")
      break;
  }
}
std::string read_file(const fs::path &path, std::size_t limit,
                      bool secret = false) {
  n::dirs(path.parent_path());
  n::FD f(::open(path.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC | O_NONBLOCK));
  struct stat s{};
  n::need(f.n >= 0 && ::fstat(f.n, &s) == 0 && S_ISREG(s.st_mode) &&
          s.st_nlink == 1 && s.st_size >= 0 &&
          std::uint64_t(s.st_size) <= limit && !(s.st_mode & 0022));
  if (secret)
    n::need(s.st_uid == ::geteuid() && !(s.st_mode & 0077));
  std::string b(std::size_t(s.st_size), '\0');
  std::size_t at = 0;
  while (at < b.size()) {
    auto count = ::read(f.n, b.data() + at, b.size() - at);
    if (count < 0 && errno == EINTR)
      continue;
    n::need(count > 0);
    at += std::size_t(count);
  }
  char extra;
  n::need(::read(f.n, &extra, 1) == 0);
  return b;
}
void exact_binary(const Runtime &r) {
  n::need(n::hash(read_file(r.cloudflared, 512 * 1024 * 1024)) ==
          r.cloudflared_digest);
}
std::string decode64(const std::string &s) {
  n::need(s.size() <= 32768 && !s.empty());
  std::string out;
  unsigned v = 0, bits = 0;
  for (char c : s) {
    auto at =
        std::string(
            "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
            .find(c);
    n::need(at != std::string::npos);
    v = (v << 6) | static_cast<unsigned>(at);
    bits += 6;
    if (bits >= 8) {
      bits -= 8;
      out += char((v >> bits) & 255);
    }
  }
  n::need(bits != 6 && (v & ((1U << bits) - 1)) == 0);
  return out;
}
void identity(const Json &i) {
  n::fields(i, {"audience", "issuer", "subject"});
  n::need(match(text(i.at("audience")), "[a-f0-9]{64}"));
  n::need(match(text(i.at("issuer")),
                "https://[a-z0-9-]+\\.cloudflareaccess\\.com"));
  n::need(match(text(i.at("subject")), "[a-zA-Z0-9][a-zA-Z0-9._:@-]{0,255}"));
}
struct Locked {
  fs::path root;
  n::FD lock;
  explicit Locked(const fs::path &p) : root(p) {
    n::user();
    n::private_dir(root);
    lock.n = ::open((root / "session.lock").c_str(),
                    O_RDWR | O_CREAT | O_NOFOLLOW | O_CLOEXEC, 0600);
    struct stat st{};
    n::need(lock.n >= 0 && ::fstat(lock.n, &st) == 0 && S_ISREG(st.st_mode) &&
            st.st_nlink == 1 && st.st_uid == ::geteuid() &&
            !(st.st_mode & 0077));
    n::need(::flock(lock.n, LOCK_EX | LOCK_NB) == 0);
    n::private_dir(root / "plans", true);
    n::private_dir(root / "attempts", true);
  }
  Json state() const {
    auto p = root / "state.json";
    if (!fs::exists(p))
      return nullptr;
    return n::parse(read_file(p, 65536, true), 65536);
  }
};
void binding(const Json &b) {
  n::fields(b, {"application", "session_ref", "session_directory",
                "profile_digest", "endpoint_ref"});
  hostname(text(b.at("application")));
  n::need(match(text(b.at("session_ref")),
                "session:[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}"));
  n::need(n::tagged(b.at("profile_digest")) &&
          match(text(b.at("endpoint_ref")), "sov:ssh:v2:sha256:[a-f0-9]{64}"));
  auto p = fs::path(text(b.at("session_directory")));
  n::need(p.is_absolute() && p.lexically_normal() == p && p != "/");
}
void owns(const Json &state, const Json &b, const Runtime &r) {
  if (state.is_null())
    return;
  n::need(state.at("uid") == std::to_string(::geteuid()) &&
          state.at("application") == b.at("application") &&
          state.at("session_ref") == b.at("session_ref") &&
          state.at("helper_digest") == r.helper_digest);
}
std::string state_ref(const Json &state) {
  return n::seal(state, "symphony.sov.transport-session-state.v1");
}
void erase_tree(const fs::path &p) {
  // Only our fresh scratch tree; refuse symlinks, hard links, foreign owners.
  struct stat s{};
  if (::lstat(p.c_str(), &s) != 0) {
    n::need(errno == ENOENT);
    return;
  }
  n::need(s.st_uid == ::geteuid() && !S_ISLNK(s.st_mode));
  if (S_ISDIR(s.st_mode)) {
    n::need(::chmod(p.c_str(), 0700) == 0);
    for (auto &e : fs::directory_iterator(p))
      erase_tree(e.path());
    n::need(::rmdir(p.c_str()) == 0);
  } else {
    n::need(S_ISREG(s.st_mode) && s.st_nlink == 1);
    n::need(::unlink(p.c_str()) == 0);
  }
}
struct Scratch {
  fs::path path;
  n::FD lease;
  explicit Scratch(const fs::path &root) {
    auto pattern = (root / ".access-XXXXXX").string();
    std::vector<char> p(pattern.begin(), pattern.end());
    p.push_back(0);
    n::need(::mkdtemp(p.data()) != nullptr);
    path = p.data();
    n::private_dir(path / ".cloudflared", true);
    lease.n = ::open((path / "lease").c_str(),
                     O_RDWR | O_CREAT | O_EXCL | O_CLOEXEC | O_NOFOLLOW, 0600);
    n::need(lease.n >= 0 && ::flock(lease.n, LOCK_EX | LOCK_NB) == 0);
  }
  ~Scratch() {
    try {
      erase_tree(path);
    } catch (...) {
    }
  }
};
int child(const Runtime &r, const std::vector<std::string> &args,
          const fs::path &home, std::int64_t deadline, bool stream) {
  exact_binary(r);
  n::need(n::now() < deadline);
  std::vector<std::string> env{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C",
                               "HOME=" + home.string()};
  // Explicit interactive login may use the actual user's desktop session.
  // Connections never inherit these settings or any credential/proxy settings.
  if (!stream)
    for (auto key : {"DISPLAY", "WAYLAND_DISPLAY", "XDG_RUNTIME_DIR",
                     "DBUS_SESSION_BUS_ADDRESS"}) {
      auto p = ::getenv(key);
      if (p && std::strlen(p) < 4096 && std::strchr(p, '\n') == nullptr)
        env.push_back(std::string(key) + "=" + p);
    }
  std::vector<char *> argv{const_cast<char *>(r.cloudflared.c_str())}, envp;
  for (auto &a : args)
    argv.push_back(const_cast<char *>(a.c_str()));
  argv.push_back(nullptr);
  for (auto &e : env)
    envp.push_back(e.data());
  envp.push_back(nullptr);
  pid_t pid = ::fork();
  n::need(pid >= 0);
  if (pid == 0) {
    ::umask(0077);
    int dev = ::open("/dev/null", O_RDWR);
    if (dev < 0 || ::chdir(home.c_str()) != 0)
      _exit(126);
    if (!stream) {
      if (::dup2(dev, STDIN_FILENO) < 0 || ::dup2(dev, STDOUT_FILENO) < 0)
        _exit(126);
    }
    if (::dup2(dev, STDERR_FILENO) < 0)
      _exit(126);
    if (dev > 2)
      ::close(dev);
    n::close_others();
    ::execve(r.cloudflared.c_str(), argv.data(), envp.data());
    _exit(126);
  }
  int status = 0;
  for (;;) {
    auto waited = ::waitpid(pid, &status, WNOHANG);
    if (waited == pid)
      break;
    if (waited < 0) {
      if (errno == EINTR)
        continue;
      return 125;
    }
    if (n::now() >= deadline) {
      ::kill(pid, SIGKILL);
      while (::waitpid(pid, &status, 0) < 0 && errno == EINTR) {
      };
      return 124;
    }
    std::this_thread::sleep_for(std::chrono::milliseconds(20));
  }
  return WIFEXITED(status) ? WEXITSTATUS(status) : 125;
}
std::string timestamp(const Json &value) {
  auto seconds = std::stoll(text(value)) / 1000;
  std::time_t at = seconds;
  std::tm tm{};
  n::need(::gmtime_r(&at, &tm) != nullptr);
  char date[32];
  n::need(std::strftime(date, sizeof date, "%Y-%m-%dT%H:%M:%SZ", &tm) > 0);
  return date;
}
Json usable(const Runtime &r, Locked &l, const Json &b) {
  auto s = l.state();
  owns(s, b, r);
  if (s.is_null())
    return nullptr;
  auto ready =
      n::parse(read_file(l.root / "readiness.json", 65536, true), 65536);
  n::fields(ready, {"protocol", "application", "session_ref", "helper_digest",
                    "expires_at"});
  n::need(ready.at("protocol") == "symphony.sov.proxy-readiness.v1" &&
          ready.at("application") == s.at("application") &&
          ready.at("session_ref") == s.at("session_ref") &&
          ready.at("helper_digest") == r.helper_digest &&
          ready.at("expires_at") == timestamp(s.at("expires_unix_ms")));
  n::Secret token;
  token.bytes = read_file(l.root / "token", 32768, true);
  auto c = claims(token.bytes, s.at("identity"), n::now());
  n::need(c.at("expires_unix_ms") == s.at("expires_unix_ms"));
  return s;
}
} // namespace
Runtime runtime(const fs::path &executable) {
  n::user();
  protected_path(executable);
  auto config = executable.parent_path() / "access-runtime.json";
  protected_path(config);
  auto j = n::parse(read_file(config, 65536), 65536);
  n::fields(j, {"protocol", "cloudflared_version", "cloudflared_digest",
                "carrier_digest", "ssh_digest"});
  n::need(j.at("protocol") == "symphony.sov.cloudflare-access-runtime.v1" &&
          j.at("cloudflared_version") == "2026.10.7");
  for (auto key : {"cloudflared_digest", "carrier_digest", "ssh_digest"})
    n::need(n::tagged(j.at(key)));
  Runtime r{executable,
            executable.parent_path() / "cloudflared",
            n::hash(read_file(executable, 512 * 1024 * 1024)),
            text(j.at("cloudflared_digest")),
            text(j.at("carrier_digest")),
            text(j.at("ssh_digest"))};
  protected_path(r.cloudflared);
#if defined(__APPLE__) && defined(__x86_64__)
  n::need(r.cloudflared_digest ==
          "sha256:"
          "5c55e77da7012cebca19170357e196176c48c9647cc2ead404702fa125ddbd86");
#else
  n::need(false); // Runtime admission requires separately qualified exact
                  // artifacts.
#endif
  exact_binary(r);
  return r;
}
void parent(const Runtime &r, bool connection) {
  auto p = n::executable(::getppid());
  protected_path(p);
  n::need(n::hash(read_file(p, 512 * 1024 * 1024)) ==
          (connection ? r.ssh_digest : r.carrier_digest));
}
Json claims(const std::string &token, const Json &id, std::int64_t now) {
  identity(id);
  n::need(token.size() <= 32768);
  auto a = token.find('.'),
       b = token.find('.', a == std::string::npos ? 0 : a + 1);
  n::need(a != std::string::npos && b != std::string::npos &&
          b + 1 < token.size() && token.find('.', b + 1) == std::string::npos);
  auto header = n::parse(decode64(token.substr(0, a)), 4096);
  n::need(header.at("alg") == "RS256" || header.at("alg") == "ES256");
  auto c = n::parse(decode64(token.substr(a + 1, b - a - 1)), 32768);
  n::need(c.at("iss") == id.at("issuer") && c.at("sub") == id.at("subject"));
  bool aud = c.at("aud") == id.at("audience");
  if (c.at("aud").is_array())
    for (auto &v : c.at("aud"))
      aud = aud || v == id.at("audience");
  n::need(aud && c.at("exp").is_number_integer());
  auto expiry = c.at("exp").get<std::int64_t>();
  n::need(expiry > now / 1000 + 5 && expiry < 9007199254740LL);
  if (c.contains("nbf"))
    n::need(c.at("nbf").is_number_integer() &&
            (!c.at("nbf").is_number_unsigned() ||
             c.at("nbf").get<std::uint64_t>() <= 9007199254740991ULL) &&
            c.at("nbf").get<std::int64_t>() <= now / 1000);
  return Json{{"expires_unix_ms", std::to_string(expiry * 1000)}};
}
Json session(const Runtime &r, const Json &q) {
  n::user();
  n::fields(q, {"protocol", "request_id", "operation", "binding", "intent",
                "plan_digest", "deadline_unix_ms"});
  n::need(q.at("protocol") == request_protocol &&
          match(text(q.at("request_id")), "[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}"
                                          "-[89ab][a-f0-9]{3}-[a-f0-9]{12}"));
  auto op = text(q.at("operation"));
  n::need(op == "inspect" || op == "plan" || op == "authenticate" ||
          op == "renew" || op == "clear");
  auto b = q.at("binding");
  binding(b);
  auto decimal = text(q.at("deadline_unix_ms"));
  n::need(match(decimal, "[1-9][0-9]{0,15}"));
  auto deadline = std::stoll(decimal);
  n::need(deadline > n::now() && deadline <= n::now() + 300000);
  Locked l(text(b.at("session_directory")));
  auto before = l.state();
  owns(before, b, r);
  Json out{{"protocol", "symphony.sov.transport-session-native-result.v1"},
           {"request_id", q.at("request_id")},
           {"operation", op},
           {"binding", b},
           {"helper_digest", r.helper_digest},
           {"completed", true},
           {"code", "session.observed"},
           {"state", "interaction_required"},
           {"plan_digest", nullptr},
           {"expires_unix_ms", nullptr},
           {"provider_authorization", "not_proven"}};
  auto inspect = [&] {
    try {
      auto s = usable(r, l, b);
      if (!s.is_null()) {
        out["state"] = "locally_usable";
        out["expires_unix_ms"] = s.at("expires_unix_ms");
      }
    } catch (...) {
      out["state"] = "interaction_required";
    }
  };
  if (op == "inspect") {
    n::need(q.at("intent").is_null() && q.at("plan_digest").is_null());
    inspect();
    return out;
  }
  if (op == "plan") {
    n::need(q.at("plan_digest").is_null());
    auto i = q.at("intent");
    n::fields(i, {"action", "identity"});
    auto action = text(i.at("action"));
    n::need(action == "authenticate" || action == "renew" || action == "clear");
    if (action == "clear")
      n::need(i.at("identity").is_null());
    else
      identity(i.at("identity"));
    if (action == "renew")
      n::need(!before.is_null() && before.at("identity") == i.at("identity"));
    Json p{{"protocol", plan_protocol},
           {"request_id", q.at("request_id")},
           {"binding", b},
           {"uid", std::to_string(::geteuid())},
           {"helper_digest", r.helper_digest},
           {"cloudflared_digest", r.cloudflared_digest},
           {"intent", i},
           {"before", state_ref(before)},
           {"expires_unix_ms", std::to_string(deadline)}};
    auto d = n::seal(p, plan_protocol);
    auto path = l.root / "plans" / (d.substr(7) + ".json");
    if (!fs::exists(path))
      n::put(path, p.dump());
    out["plan_digest"] = d;
    out["code"] = "session.planned";
    inspect();
    return out;
  }
  n::need(q.at("intent").is_null() && n::tagged(q.at("plan_digest")));
  auto d = text(q.at("plan_digest"));
  auto p = n::parse(
      read_file(l.root / "plans" / (d.substr(7) + ".json"), 65536, true),
      65536);
  n::need(n::seal(p, plan_protocol) == d && p.at("binding") == b &&
          p.at("uid") == std::to_string(::geteuid()) &&
          p.at("helper_digest") == r.helper_digest &&
          p.at("cloudflared_digest") == r.cloudflared_digest &&
          p.at("intent").at("action") == op &&
          p.at("before") == state_ref(before) &&
          std::stoll(text(p.at("expires_unix_ms"))) > n::now());
  deadline = std::min<std::int64_t>(deadline,
                                    std::stoll(text(p.at("expires_unix_ms"))));
  auto journal = l.root / "attempts" / (d.substr(7) + ".json");
  n::need(!fs::exists(journal));
  n::put(journal,
         Json{{"plan_digest", d}, {"stage", "dispatch_intent"}}.dump());
  out["plan_digest"] = d;
  if (op == "clear") {
    for (auto name : {"readiness.json", "token", "token.new", "state.json"}) {
      auto path = l.root / name;
      if (fs::exists(path)) {
        (void)read_file(path, 65536, true);
        n::need(::unlink(path.c_str()) == 0);
      }
    }
    for (auto &e : fs::directory_iterator(l.root))
      if (e.path().filename().string().starts_with(".access-")) {
        n::private_dir(e.path());
        n::FD lease(::open((e.path() / "lease").c_str(),
                           O_RDONLY | O_CLOEXEC | O_NOFOLLOW | O_NONBLOCK));
        struct stat st{};
        n::need(lease.n >= 0 && ::fstat(lease.n, &st) == 0 &&
                S_ISREG(st.st_mode) && st.st_uid == ::geteuid() &&
                st.st_nlink == 1);
        if (::flock(lease.n, LOCK_EX | LOCK_NB) == 0)
          erase_tree(e.path());
        else
          n::need(errno == EWOULDBLOCK);
      }
    n::sync(l.root);
    out["code"] = "session.cleared";
  } else {
    Scratch scratch(l.root);
    auto status = child(
        r,
        {"access", "login", "--quiet", "https://" + text(b.at("application"))},
        scratch.path, deadline, false);
    if (status != 0) {
      out["completed"] = false;
      out["code"] = "session.authentication_failed";
      inspect();
    } else {
      auto id = p.at("intent").at("identity");
      auto file =
          text(b.at("application")) + "-" + text(id.at("audience")) + "-token";
      n::Secret token;
      token.bytes =
          read_file(scratch.path / ".cloudflared" / file, 32768, true);
      auto c = claims(token.bytes, id, n::now());
      Json state{{"uid", std::to_string(::geteuid())},
                 {"application", b.at("application")},
                 {"session_ref", b.at("session_ref")},
                 {"helper_digest", r.helper_digest},
                 {"identity", id},
                 {"expires_unix_ms", c.at("expires_unix_ms")}};
      // Invalidate public readiness first. Interrupted replacement is never
      // ready.
      ::unlink((l.root / "readiness.json").c_str());
      n::put(l.root / "token.new", token.bytes);
      n::need(::rename((l.root / "token.new").c_str(),
                       (l.root / "token").c_str()) == 0);
      n::atomic(l.root / "state.json", state);
      auto date = timestamp(c.at("expires_unix_ms"));
      n::atomic(l.root / "readiness.json",
                Json{{"protocol", "symphony.sov.proxy-readiness.v1"},
                     {"application", b.at("application")},
                     {"session_ref", b.at("session_ref")},
                     {"helper_digest", r.helper_digest},
                     {"expires_at", date}});
      out["code"] = "session.authenticated";
      inspect();
    }
  }
  n::atomic(journal, Json{{"plan_digest", d},
                          {"stage", "settled"},
                          {"completed", out.at("completed")},
                          {"code", out.at("code")}});
  return out;
}
int connect(const Runtime &r, const std::string &application,
            const fs::path &directory, const std::string &reference) {
  n::user();
  hostname(application);
  Locked l(directory);
  auto s = usable(
      r, l, Json{{"application", application}, {"session_ref", reference}});
  n::need(!s.is_null() && s.at("application") == application &&
          s.at("session_ref") == reference &&
          s.at("helper_digest") == r.helper_digest &&
          s.at("uid") == std::to_string(::geteuid()));
  (void)read_file(l.root / "readiness.json", 65536, true);
  n::Secret token;
  token.bytes = read_file(l.root / "token", 32768, true);
  (void)claims(token.bytes, s.at("identity"), n::now());
  Scratch scratch(l.root);
  auto cache = scratch.path / ".cloudflared";
  n::put(cache / (application + "-" + text(s.at("identity").at("audience")) +
                  "-token"),
         token.bytes, 0400);
  n::wipe(token.bytes);
  n::need(::chmod(cache.c_str(), 0500) == 0 &&
          ::chmod(scratch.path.c_str(), 0500) == 0);
  // No org token, writable cache or token lock: this pinned release cannot
  // refresh credentials or begin browser login on a connection attempt.
  n::need(::flock(l.lock.n, LOCK_UN) == 0);
  return child(
      r, {"access", "ssh", "--hostname", application, "--loglevel", "error"},
      scratch.path, std::numeric_limits<std::int64_t>::max(), true);
}
} // namespace cloudflare::access
