#pragma once
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/protocol.hpp"
#include <cerrno>
#include <fcntl.h>
#include <filesystem>
#include <random>
#include <string>
#include <sys/stat.h>
#include <unistd.h>

namespace symphony::srrapiv::private_io {
namespace engine = symphony::knowledge::engine;
using Json = engine::Json;
inline void require(bool ok, const char *code = "srrapiv.request.invalid") {
  if (!ok) throw engine::Error(code, "SRRAPIV request refused", 3);
}
struct Fd {
  int value = -1;
  explicit Fd(int v) : value(v) {}
  ~Fd() { if (value >= 0) ::close(value); }
  Fd(const Fd &) = delete;
  Fd &operator=(const Fd &) = delete;
  Fd(Fd &&other) noexcept : value(other.value) {other.value=-1;}
  Fd &operator=(Fd &&other) noexcept {if(this!=&other){if(value>=0)::close(value);value=other.value;other.value=-1;}return *this;}
};
inline std::string token(const Json &p, const char *key, std::size_t limit = 256) {
  require(p.contains(key) && p.at(key).is_string());
  auto s = p.at(key).get<std::string>();
  require(!s.empty() && s.size() <= limit && s.find('\0') == std::string::npos);
  return s;
}
inline void fields(const Json &p, std::initializer_list<const char *> allowed) {
  require(p.is_object());
  for (auto i = p.begin(); i != p.end(); ++i) {
    bool found = false;
    for (const auto *k : allowed) if (i.key() == k) found = true;
    require(found);
  }
}
// Paths are explicit absolute no-follow paths. Sticky shared temporary parents
// are permitted; the final private directory/file must belong to this uid.
inline void path(const std::string &s) {
  require(s.size() <= 4096 && !s.empty() && s[0] == '/' && s.find('\0') == std::string::npos);
  std::filesystem::path p(s), cursor("/");
  for (const auto &part : p.relative_path()) {
    require(part != "." && part != "..");
    cursor /= part;
    struct stat st{};
    if (::lstat(cursor.c_str(), &st) == 0) {
      require(!S_ISLNK(st.st_mode), "srrapiv.path.unsafe");
      if (cursor != p) require(S_ISDIR(st.st_mode) && (st.st_uid==0||st.st_uid==::getuid()) &&
        ((st.st_mode&0022)==0 || (st.st_uid==0&&(st.st_mode&S_ISVTX))), "srrapiv.path.unsafe");
    } else require(errno == ENOENT, "srrapiv.path.unsafe");
  }
}
inline Fd open_directory(const std::string &s,bool create=false) {
  path(s);std::filesystem::path p(s);Fd cursor(::open("/",O_RDONLY|O_DIRECTORY|O_CLOEXEC));require(cursor.value>=0);
  std::size_t index=0,total=0;for(const auto &part:p.relative_path()){(void)part;++total;}
  require(total>0,"srrapiv.path.unsafe");
  for(const auto &part:p.relative_path()) {
    ++index;auto name=part.string();
    if(create&&index==total&&::mkdirat(cursor.value,name.c_str(),0700)!=0)require(errno==EEXIST,"srrapiv.storage.failed");
    Fd next(::openat(cursor.value,name.c_str(),O_RDONLY|O_DIRECTORY|O_NOFOLLOW|O_CLOEXEC));struct stat st{};
    require(next.value>=0&&::fstat(next.value,&st)==0,"srrapiv.path.unsafe");
    if(index==total) require(st.st_uid==::getuid()&&(st.st_mode&0077)==0,"srrapiv.path.unsafe");
    else require((st.st_uid==0||st.st_uid==::getuid())&&((st.st_mode&0022)==0||(st.st_uid==0&&(st.st_mode&S_ISVTX))),"srrapiv.path.unsafe");
    cursor=std::move(next);
  }
  return cursor;
}
inline void directory(const std::string &s, bool create = false) {
  auto fd=open_directory(s,create);(void)fd;
}
inline std::string read(const std::string &s, std::size_t limit = 1048576) {
  path(s);
  auto p=std::filesystem::path(s);auto parent=open_directory(p.parent_path().string());
  Fd fd(::openat(parent.value,p.filename().c_str(), O_RDONLY | O_NOFOLLOW | O_CLOEXEC));
  struct stat st{};
  require(fd.value >= 0 && ::fstat(fd.value, &st) == 0 && S_ISREG(st.st_mode) &&
          st.st_uid == ::getuid() && (st.st_mode & 0077) == 0 && st.st_nlink == 1 &&
          st.st_size >= 0 && static_cast<std::uint64_t>(st.st_size) <= limit,
          "srrapiv.path.unsafe");
  std::string out; char b[8192];
  for (;;) {
    auto n = ::read(fd.value, b, sizeof b);
    if (n < 0 && errno == EINTR) continue;
    require(n >= 0, "srrapiv.storage.failed");
    if (n == 0) break;
    out.append(b, static_cast<std::size_t>(n));
    require(out.size() <= limit, "srrapiv.request.limit");
  }
  return out;
}
inline Json read_json(const std::string &s) { return engine::parse_bounded_json(read(s), 1048576,65536); }
inline void write(const std::string &s, const Json &j) {
  path(s);
  auto target=std::filesystem::path(s);auto parent=open_directory(target.parent_path().string());
  std::random_device random;std::string pattern=".write-"+std::to_string(::getpid())+"-"+std::to_string(random());
  Fd fd(::openat(parent.value,pattern.c_str(),O_CREAT|O_EXCL|O_WRONLY|O_NOFOLLOW|O_CLOEXEC,0600));
  require(fd.value >= 0, "srrapiv.storage.failed");
  try {
    require(::fchmod(fd.value, 0600) == 0, "srrapiv.storage.failed");
    const auto raw = j.dump() + "\n";
    std::size_t at = 0;
    while (at < raw.size()) {
      auto n = ::write(fd.value, raw.data() + at, raw.size() - at);
      if (n < 0 && errno == EINTR) continue;
      require(n > 0, "srrapiv.storage.failed"); at += static_cast<std::size_t>(n);
    }
    struct stat existing{};
    if(::fstatat(parent.value,target.filename().c_str(),&existing,AT_SYMLINK_NOFOLLOW)==0)
      require(S_ISREG(existing.st_mode)&&existing.st_uid==::getuid()&&(existing.st_mode&0077)==0&&existing.st_nlink==1,"srrapiv.path.unsafe");
    else require(errno==ENOENT,"srrapiv.storage.failed");
    require(::fsync(fd.value) == 0 && ::renameat(parent.value,pattern.c_str(),parent.value,target.filename().c_str()) == 0, "srrapiv.storage.failed");
    require(::fsync(parent.value) == 0, "srrapiv.storage.failed");
  } catch (...) { ::unlinkat(parent.value,pattern.c_str(),0); throw; }
}
inline std::uint64_t decimal(const Json &j) {
  require(j.is_string()); auto s = j.get<std::string>();
  require(!s.empty() && s.size() <= 20 && (s.size() == 1 || s.front() != '0'));
  std::uint64_t v = 0;
  for (char c : s) { require(c >= '0' && c <= '9'); auto d = unsigned(c - '0'); require(v <= (UINT64_MAX-d)/10); v=v*10+d; }
  return v;
}
}
