#pragma once
// Native-only bounded IPC support. No secret bytes pass through Go or qxctl.
#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <cstring>
#include <dirent.h>
#include <fcntl.h>
#include <filesystem>
#include <poll.h>
#include <regex>
#include <set>
#include <stdexcept>
#include <string>
#include <symphony/knowledge/engine/digest.hpp>
#include <symphony/knowledge/engine/protocol.hpp>
#include <sys/file.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <sys/un.h>
#include <unistd.h>
#ifdef __APPLE__
#include <libproc.h>
#include <sys/ucred.h>
#endif
namespace symphony::ssiag::native {
namespace fs = std::filesystem;
using Json = symphony::knowledge::engine::Json;
inline void need(bool b) {
  if (!b)
    throw std::runtime_error("native boundary unavailable");
}
struct FD {
  int n = -1;
  explicit FD(int x = -1) : n(x) {}
  ~FD() {
    if (n >= 0)
      ::close(n);
  }
  FD(const FD &) = delete;
  FD &operator=(const FD &) = delete;
  FD(FD &&x) noexcept : n(x.n) { x.n = -1; }
};
inline void wipe(std::string &s) {
  volatile char *p = s.data();
  for (std::size_t i = 0; i < s.size(); ++i)
    p[i] = 0;
  s.clear();
}
struct Secret {
  std::string bytes;
  ~Secret() { wipe(bytes); }
  Secret() = default;
  Secret(const Secret &) = delete;
  Secret &operator=(const Secret &) = delete;
};
inline std::int64_t now() {
  return symphony::knowledge::engine::unix_time_ms();
}
inline std::string hash(const std::string &b) {
  return symphony::knowledge::engine::tagged_sha256(b);
}
inline std::string seal(const Json &j, const std::string &d) {
  return hash(d + '\0' + j.dump());
}
inline Json parse(const std::string &b, std::size_t limit = 8 * 1024 * 1024) {
  return symphony::knowledge::engine::parse_bounded_json(b, limit, 500000);
}
inline void fields(const Json &j, std::initializer_list<const char *> keys) {
  need(j.is_object() && j.size() == keys.size());
  for (auto k : keys)
    need(j.contains(k));
}
inline bool tagged(const Json &j) {
  return j.is_string() && std::regex_match(j.get<std::string>(),
                                           std::regex("sha256:[a-f0-9]{64}"));
}
inline void user() {
  need(::geteuid() != 0 && ::getuid() == ::geteuid() &&
       ::getgid() == ::getegid());
}
inline void dirs(const fs::path &p) {
  need(p.is_absolute() && p.lexically_normal() == p);
  fs::path walk;
  for (const auto &part : p) {
    walk /= part;
    struct stat s{};
    need(::lstat(walk.c_str(), &s) == 0 && S_ISDIR(s.st_mode));
    need((s.st_mode & 0022) == 0 || (s.st_uid == 0 && (s.st_mode & S_ISVTX)));
  }
}
inline void private_dir(const fs::path &p, bool create = false) {
  dirs(p.parent_path());
  if (create)
    need(::mkdir(p.c_str(), 0700) == 0 || errno == EEXIST);
  struct stat s{};
  need(::lstat(p.c_str(), &s) == 0 && S_ISDIR(s.st_mode) &&
       s.st_uid == ::geteuid() && (s.st_mode & 0077) == 0);
}
inline std::string read(const fs::path &p, std::size_t limit = 8 * 1024 * 1024,
                        bool secret = false) {
  dirs(p.parent_path());
  FD f(::open(p.c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC));
  struct stat s{};
  need(f.n >= 0 && ::fstat(f.n, &s) == 0 && S_ISREG(s.st_mode) &&
       s.st_nlink == 1 && s.st_size >= 0 && std::uint64_t(s.st_size) <= limit &&
       (s.st_mode & 0022) == 0);
  if (secret)
    need(s.st_uid == ::geteuid() && (s.st_mode & 0077) == 0);
  std::string b(std::size_t(s.st_size), '\0');
  std::size_t at = 0;
  while (at < b.size()) {
    auto n = ::read(f.n, b.data() + at, b.size() - at);
    if (n < 0 && errno == EINTR)
      continue;
    need(n > 0);
    at += std::size_t(n);
  }
  char c;
  need(::read(f.n, &c, 1) == 0);
  return b;
}
inline void sync(const fs::path &p) {
  FD f(::open(p.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
  need(f.n >= 0 && ::fsync(f.n) == 0);
}
inline void write_all(int fd, const std::string &b) {
  std::size_t at = 0;
  while (at < b.size()) {
    auto n = ::write(fd, b.data() + at, b.size() - at);
    if (n < 0 && errno == EINTR)
      continue;
    need(n > 0);
    at += std::size_t(n);
  }
}
inline void put(const fs::path &p, const std::string &b, mode_t mode = 0600) {
  dirs(p.parent_path());
  FD f(::open(p.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC,
              mode));
  need(f.n >= 0);
  write_all(f.n, b);
  need(::fsync(f.n) == 0);
  sync(p.parent_path());
}
inline void atomic(const fs::path &p, const Json &j) {
  auto q = p;
  q += ".new-" + std::to_string(::getpid());
  put(q, j.dump() + "\n");
  need(::rename(q.c_str(), p.c_str()) == 0);
  sync(p.parent_path());
}
inline void exact_read(int fd, char *p, std::size_t n) {
  while (n) {
    auto k = ::read(fd, p, n);
    if (k < 0 && errno == EINTR)
      continue;
    need(k > 0);
    p += k;
    n -= std::size_t(k);
  }
}
inline void frame(int fd, const std::string &b) {
  need(!b.empty() && b.size() <= 8 * 1024 * 1024);
  std::string h(4, '\0');
  auto n = std::uint32_t(b.size());
  for (int i = 3; i >= 0; --i) {
    h[std::size_t(i)] = char(n & 255);
    n >>= 8;
  }
  write_all(fd, h);
  write_all(fd, b);
}
inline std::string frame(int fd, std::size_t bound = 32768) {
  unsigned char h[4];
  exact_read(fd, reinterpret_cast<char *>(h), 4);
  std::uint32_t n = 0;
  for (auto c : h)
    n = (n << 8) | c;
  need(n > 0 && n <= bound);
  std::string b(n, '\0');
  exact_read(fd, b.data(), n);
  return b;
}
inline void timeout(int fd, int seconds) {
  timeval tv{seconds, 0};
  need(::setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &tv, sizeof(tv)) == 0 &&
       ::setsockopt(fd, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv)) == 0);
}
struct Peer {
  pid_t pid;
  uid_t uid;
  gid_t gid;
};
inline Peer peer(int fd) {
#ifdef __APPLE__
  uid_t uid{};
  gid_t gid{};
  pid_t pid{};
  socklen_t n = sizeof(pid);
  need(::getpeereid(fd, &uid, &gid) == 0 &&
       ::getsockopt(fd, SOL_LOCAL, LOCAL_PEERPID, &pid, &n) == 0 && pid > 0);
  return {pid, uid, gid};
#else
  struct ucred c{};
  socklen_t n = sizeof(c);
  need(::getsockopt(fd, SOL_SOCKET, SO_PEERCRED, &c, &n) == 0 && c.pid > 0);
  return {c.pid, c.uid, c.gid};
#endif
}
inline fs::path executable(pid_t pid) {
#ifdef __APPLE__
  std::array<char, PROC_PIDPATHINFO_MAXSIZE> p{};
  need(::proc_pidpath(pid, p.data(), std::uint32_t(p.size())) > 0);
  return fs::path(p.data());
#else
  std::array<char, 4096> p{};
  auto n = ::readlink(("/proc/" + std::to_string(pid) + "/exe").c_str(),
                      p.data(), p.size() - 1);
  need(n > 0 && std::size_t(n) < p.size() - 1);
  return fs::path(std::string(p.data(), std::size_t(n)));
#endif
}
inline void close_others(int keep = -1) {
  DIR *d = ::opendir("/dev/fd");
  if (!d)
    d = ::opendir("/proc/self/fd");
  need(d != nullptr);
  int table = ::dirfd(d);
  std::vector<int> fds;
  while (auto e = ::readdir(d)) {
    char *end = nullptr;
    long n = std::strtol(e->d_name, &end, 10);
    if (end && *end == '\0' && n > 2 && n != table && n != keep)
      fds.push_back(int(n));
  }
  ::closedir(d);
  for (int f : fds)
    ::close(f);
}
inline FD connect(const fs::path &path, uid_t expected,
                  const std::string &digest) {
  dirs(path.parent_path());
  struct stat s{};
  need(::lstat(path.c_str(), &s) == 0 && S_ISSOCK(s.st_mode) &&
       s.st_uid == expected && (s.st_mode & 0002) == 0);
  FD f(::socket(AF_UNIX, SOCK_STREAM, 0));
  need(f.n >= 0);
  sockaddr_un a{};
  a.sun_family = AF_UNIX;
  auto name = path.string();
  need(name.size() < sizeof(a.sun_path));
  std::memcpy(a.sun_path, name.c_str(), name.size() + 1);
  timeout(f.n, 60);
  need(::connect(f.n, reinterpret_cast<sockaddr *>(&a), sizeof(a)) == 0);
  auto p = peer(f.n);
  need(p.uid == expected &&
       hash(read(executable(p.pid), 512 * 1024 * 1024)) == digest);
  return f;
}
inline fs::path installation(const fs::path &prefix,
                             const std::string &component,
                             const std::string &version,
                             const std::string &expected) {
  dirs(prefix);
  auto p = prefix / "share/symphony/receipts" / component / version /
           "install-receipt.json";
  auto j = parse(read(p));
  need(j.at("protocol") == "symphony.knowledge.install-receipt.v2" &&
       j.at("component_id") == component && j.at("version") == version &&
       j.at("receipt_digest") == expected);
  auto unsigned_j = j;
  unsigned_j.erase("receipt_digest");
  need(hash(unsigned_j.dump()) == expected);
  std::set<std::string> seen;
  for (const auto &item : j.at("files")) {
    fs::path rel = item.at("path").get<std::string>();
    need(!rel.is_absolute() && rel.lexically_normal() == rel &&
         seen.insert(rel.string()).second);
    for (const auto &part : rel)
      need(part != ".." && part != ".");
    auto b = read(prefix / rel, 512 * 1024 * 1024);
    need(b.size() == item.at("size") && item.at("digest") == hash(b));
  }
  auto bin = fs::path("libexec/symphony") / component / version /
             ("symphony-" + component);
  need(seen.contains(bin.string()));
  return prefix / bin;
}
inline std::string uuid() {
  FD f(::open("/dev/urandom", O_RDONLY | O_CLOEXEC));
  need(f.n >= 0);
  unsigned char b[16];
  exact_read(f.n, reinterpret_cast<char *>(b), 16);
  b[6] = (b[6] & 15) | 64;
  b[8] = (b[8] & 63) | 128;
  static const char *hex = "0123456789abcdef";
  std::string s;
  for (int i = 0; i < 16; ++i) {
    if (i == 4 || i == 6 || i == 8 || i == 10)
      s += '-';
    s += hex[b[i] >> 4];
    s += hex[b[i] & 15];
  }
  return s;
}
} // namespace symphony::ssiag::native
