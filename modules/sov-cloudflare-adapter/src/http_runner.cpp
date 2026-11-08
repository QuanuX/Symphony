#include "execution.hpp"
#include "request_guard.hpp"
#include <signal.h>
#include <sys/resource.h>
#include <sys/wait.h>
namespace cloudflare {
namespace n = symphony::ssiag::native;
HTTPResult run_http(const Json &preview, const std::string &token,
                    const fs::path &package, const fs::path &root,
                    std::int64_t deadline) {
  n::user();
  deadline = std::min(deadline, n::now() + 20000);
  n::need(n::now() < deadline);
  auto scratch = root / ("request-" + n::uuid());
  n::private_dir(scratch, true);
  struct Cleanup {
    fs::path p;
    ~Cleanup() {
      std::error_code e;
      fs::remove_all(p, e);
    }
  } cleanup{scratch};
  auto expected = preview.at("request");
  expected["deadline_unix_ms"] = std::to_string(deadline);
  if (expected.contains("query")) {
    std::string query;
    for (const auto &[k, v] : expected.at("query").items()) {
      n::need(v.is_number_integer());
      query += (query.empty() ? "?" : "&") + k + "=" + v.dump();
    }
    expected["url"] = expected.at("url").get<std::string>() + query;
  }
  n::put(scratch / "request.json", expected.dump());
  n::put(scratch / "guard.mjs", reinterpret_cast<const char *>(request_guard));
  auto args = preview.at("preview_argv").get<std::vector<std::string>>();
  n::need(args.back() == "--dry-run");
  args.pop_back();
  if (expected.at("method") == "DELETE")
    args.push_back("--force");
  std::vector<std::string> argv{
      (package / "node/bin/node").string(),
      "--permission",
      "--allow-net",
      "--allow-fs-read=" + (package / "cf").string(),
      "--allow-fs-read=" + scratch.string(),
      "--import=" + (scratch / "guard.mjs").string(),
      (package / "cf/node_modules/cf/bin/cf").string()};
  argv.insert(argv.end(), args.begin(), args.end());
  std::vector<std::string> env{"PATH=/usr/bin:/bin",
                               "HOME=" + scratch.string(),
                               "XDG_CONFIG_HOME=" + scratch.string(),
                               "XDG_CACHE_HOME=" + scratch.string(),
                               "TMPDIR=" + scratch.string(),
                               "CI=1",
                               "CF_SEND_TELEMETRY=false",
                               "CF_DELEGATION=1",
                               "DO_NOT_TRACK=1",
                               "NO_COLOR=1",
                               "LANG=C",
                               "LC_ALL=C",
                               "CLOUDFLARE_ACCOUNT_ID=" +
                                   preview.at("nonsecret_environment")
                                       .at("CLOUDFLARE_ACCOUNT_ID")
                                       .get<std::string>(),
                               "CLOUDFLARE_API_TOKEN=" + token};
  struct Clear {
    std::vector<std::string> &e;
    ~Clear() {
      for (auto &s : e)
        n::wipe(s);
    }
  } clear{env};
  std::vector<char *> av, ev;
  for (auto &x : argv)
    av.push_back(x.data());
  av.push_back(nullptr);
  for (auto &x : env)
    ev.push_back(x.data());
  ev.push_back(nullptr);
  int p[2];
  n::need(::pipe(p) == 0);
  n::FD reader(p[0]), writer(p[1]);
  auto pid = ::fork();
  n::need(pid >= 0);
  if (pid == 0) {
    rlimit core{0, 0};
    ::setrlimit(RLIMIT_CORE, &core);
    if (::dup2(writer.n, 3) < 0)
      _exit(126);
    n::close_others(3);
    int nil = ::open("/dev/null", O_RDWR);
    ::dup2(nil, 0);
    ::dup2(nil, 1);
    ::dup2(nil, 2);
    if (nil > 3)
      ::close(nil);
    if (::chdir(scratch.c_str()) != 0)
      _exit(126);
    ::execve(av[0], av.data(), ev.data());
    _exit(127);
  }
  ::close(writer.n);
  writer.n = -1;
  n::need(::fcntl(reader.n, F_SETFL, O_NONBLOCK) == 0);
  std::string wire;
  struct Erase {
    std::string &s;
    ~Erase() { n::wipe(s); }
  } erase{wire};
  bool eof = false, done = false;
  int status{};
  deadline = std::min(deadline, n::now() + 20000);
  struct Reap {
    pid_t p;
    bool &done;
    ~Reap() {
      if (!done) {
        ::kill(p, SIGKILL);
        int s{};
        while (::waitpid(p, &s, 0) < 0 && errno == EINTR) {
        }
      }
    }
  } reap{pid, done};
  while (!eof || !done) {
    n::need(wire.size() < 4 * 1024 * 1024 && n::now() < deadline);
    pollfd f{reader.n, POLLIN, 0};
    ::poll(&f, 1, 10);
    char b[16384];
    auto count = ::read(reader.n, b, sizeof(b));
    if (count > 0)
      wire.append(b, std::size_t(count));
    else if (count == 0)
      eof = true;
    else
      n::need(errno == EAGAIN || errno == EINTR);
    if (!done && ::waitpid(pid, &status, WNOHANG) == pid)
      done = true;
  }
  HTTPResult out;
  std::size_t at = 0;
  bool exited = false, received = false;
  while (at < wire.size()) {
    n::need(at + 4 <= wire.size());
    std::uint32_t size = 0;
    for (int i = 0; i < 4; ++i)
      size = (size << 8) | static_cast<unsigned char>(wire[at++]);
    n::need(size > 0 && size <= 3 * 1024 * 1024 && at + size <= wire.size());
    auto event = n::parse(wire.substr(at, size));
    at += size;
    auto stage = event.at("stage");
    if (stage == "dispatch_started") {
      n::need(!out.attempted && event.at("attempts") == 1);
      out.attempted = true;
    } else if (stage == "response_received") {
      n::need(out.attempted && !received);
      received = true;
      out.status = event.at("status");
      if (out.status >= 200 && out.status < 300 &&
          !event.at("body").is_null()) {
        auto body = event.at("body").get<std::string>();
        if (!body.empty())
          out.body = n::parse(body);
        n::wipe(body);
      }
    } else if (stage == "exited") {
      n::need(!exited && event.at("attempts") == int(out.attempted));
      exited = true;
    } else
      n::need(false);
  }
  out.clean_exit = exited && WIFEXITED(status) && WEXITSTATUS(status) == 0;
  n::need(exited);
  return out;
}
} // namespace cloudflare
