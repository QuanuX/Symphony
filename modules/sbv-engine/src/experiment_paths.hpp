#pragma once
#include "detail.hpp"
#include <algorithm>
#include <atomic>
#include <cerrno>
#include <fcntl.h>
#include <filesystem>
#include <map>
#include <memory>
#include <symphony/knowledge/engine/path.hpp>
#include <sys/stat.h>
#include <unistd.h>

namespace symphony::sbv::detail::trial {
using DirectoryId = std::pair<dev_t, ino_t>;
inline DirectoryId directory_id(int fd) {
  struct stat st{};
  need(::fstat(fd, &st) == 0 && S_ISDIR(st.st_mode),
       "destination parent unavailable");
  return {st.st_dev, st.st_ino};
}
struct Probe {
  int parent = -1, fd = -1;
  std::string name;
  std::vector<std::string> leaves;
  explicit Probe(int source) : parent(::dup(source)) {
    need(parent >= 0, "destination parent duplication failed");
  }
  ~Probe() {
    if (fd >= 0) {
      for (const auto &leaf : leaves)
        ::unlinkat(fd, leaf.c_str(), AT_REMOVEDIR);
      ::close(fd);
      ::unlinkat(parent, name.c_str(), AT_REMOVEDIR);
    }
    if (parent >= 0)
      ::close(parent);
  }
  void create(std::int64_t end) {
    static std::atomic<std::uint64_t> sequence{};
    for (;;) {
      deadline(end);
      name = ".sbv-path-probe-" + dec(::getpid()) + "-" +
             dec(sequence.fetch_add(1));
      if (::mkdirat(parent, name.c_str(), 0700) == 0)
        break;
      need(errno == EEXIST, "destination alias probe could not be created");
    }
    fd = ::openat(parent, name.c_str(),
                  O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW);
    if (fd < 0) {
      ::unlinkat(parent, name.c_str(), AT_REMOVEDIR);
      need(false, "destination alias probe could not be opened");
    }
  }
  void add(const std::string &leaf) {
    // The filesystem itself resolves case/Unicode aliases in a private sibling
    // directory, including aliases whose real destinations do not exist yet.
    leaves.push_back(leaf);
    need(::mkdirat(fd, leaf.c_str(), 0700) == 0,
         "experiment destinations alias or cannot be represented on this "
         "filesystem");
  }
};
struct Slot {
  std::string path;
  bool partition_directory{};
};
inline void destinations(const std::string &journal,
                         const std::vector<Slot> &slots, std::int64_t end) {
  std::map<DirectoryId, std::unique_ptr<Probe>> probes;
  std::vector<std::vector<DirectoryId>> ancestors;
  std::vector<DirectoryId> reserved;
  struct stat journal_stat{};
  need(::lstat(journal.c_str(), &journal_stat) == 0 &&
           S_ISDIR(journal_stat.st_mode),
       "journal must be an existing directory");
  const DirectoryId journal_id{journal_stat.st_dev, journal_stat.st_ino};
  for (const auto &slot : slots) {
    deadline(end);
    need(slot.path.size() > 1 && slot.path.size() <= 4096 &&
             slot.path[0] == '/' &&
             e::is_safe_relative_path(slot.path.substr(1)),
         "safe absolute trial destination required");
    struct Fd {
      int value;
      ~Fd() {
        if (value >= 0)
          ::close(value);
      }
    } current{::open("/", O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW)};
    need(current.value >= 0, "destination filesystem root unavailable");
    std::vector<DirectoryId> chain{directory_id(current.value)};
    const auto path = std::filesystem::path(slot.path);
    for (const auto &part : path.parent_path().relative_path()) {
      int next = ::openat(current.value, part.c_str(),
                          O_RDONLY | O_DIRECTORY | O_CLOEXEC | O_NOFOLLOW);
      need(next >= 0, "trial destination parent must exist without symlinks");
      ::close(current.value);
      current.value = next;
      chain.push_back(directory_id(current.value));
    }
    if (slot.partition_directory)
      need(std::find(chain.begin(), chain.end(), journal_id) == chain.end(),
           "partitioned destinations must be outside the experiment journal");
    const auto parent = directory_id(current.value);
    if (!probes.contains(parent)) {
      auto probe = std::make_unique<Probe>(current.value);
      probe->create(end);
      probes.emplace(parent, std::move(probe));
    }
    const auto leaf = path.filename().string();
    probes.at(parent)->add(leaf);
    if (slot.partition_directory) {
      struct stat actual{};
      if (::fstatat(current.value, leaf.c_str(), &actual,
                    AT_SYMLINK_NOFOLLOW) == 0) {
        need(S_ISDIR(actual.st_mode),
             "existing partitioned destination must be a directory");
        reserved.emplace_back(actual.st_dev, actual.st_ino);
      } else
        need(errno == ENOENT, "partitioned destination status unavailable");
    }
    ancestors.push_back(std::move(chain));
  }
  for (const auto &id : reserved)
    for (const auto &chain : ancestors)
      need(std::find(chain.begin(), chain.end(), id) == chain.end(),
           "experiment output/workspace destinations overlap");
}
} // namespace symphony::sbv::detail::trial
