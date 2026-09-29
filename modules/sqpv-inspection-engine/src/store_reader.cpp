#include "store_reader.hpp"
#include <algorithm>
#include <cerrno>
#include <cstdio>
#include <dirent.h>
#include <fcntl.h>
#include <map>
#include <sys/file.h>
#include <sys/stat.h>
#ifdef __APPLE__
#include <sys/mount.h>
#endif
#include <unistd.h>
#include <utility>
namespace symphony::sqpv::inspection {
namespace {
[[noreturn]] void corrupt() {
  refuse("sqv.store.corrupt", "Retained store integrity refused", 3);
}
void check(bool b) {
  if (!b)
    corrupt();
}
void io(bool b) {
  if (!b)
    refuse("sqv.store.unsafe_or_missing",
           "Retained store path unavailable or unsafe", 3);
}
struct File {
  int fd = -1;
  explicit File(int n = -1) : fd(n) {}
  ~File() {
    if (fd >= 0)
      ::close(fd);
  }
  File(File &&o) noexcept : fd(std::exchange(o.fd, -1)) {}
  File &operator=(File &&o) noexcept {
    if (this != &o) {
      if (fd >= 0)
        ::close(fd);
      fd = std::exchange(o.fd, -1);
    }
    return *this;
  }
  File(const File &) = delete;
  File &operator=(const File &) = delete;
};
struct Stamp {
  dev_t device;
  ino_t inode;
  off_t size;
  mode_t mode;
  uid_t uid;
  nlink_t links;
  std::int64_t msec, mnano, csec, cnano;
  bool operator==(const Stamp &) const = default;
};
Stamp stamp(const struct stat &s) {
#ifdef __APPLE__
  return {s.st_dev,
          s.st_ino,
          s.st_size,
          s.st_mode,
          s.st_uid,
          s.st_nlink,
          s.st_mtimespec.tv_sec,
          s.st_mtimespec.tv_nsec,
          s.st_ctimespec.tv_sec,
          s.st_ctimespec.tv_nsec};
#else
  return {s.st_dev,         s.st_ino,          s.st_size,
          s.st_mode,        s.st_uid,          s.st_nlink,
          s.st_mtim.tv_sec, s.st_mtim.tv_nsec, s.st_ctim.tv_sec,
          s.st_ctim.tv_nsec};
#endif
}
Stamp status(int fd, bool directory) {
  struct stat s{};
  io(fd >= 0 && ::fstat(fd, &s) == 0);
  io(s.st_uid == ::geteuid() &&
     (s.st_mode & 07777) == (directory ? 0700 : 0600) &&
     (directory ? S_ISDIR(s.st_mode)
                : (S_ISREG(s.st_mode) && s.st_nlink == 1)) &&
     s.st_size >= 0);
  return stamp(s);
}
Stamp linked(int root, const std::string &name) {
  struct stat s{};
  io(::fstatat(root, name.c_str(), &s, AT_SYMLINK_NOFOLLOW) == 0);
  return stamp(s);
}
void unchanged(const Stamp &a, const Stamp &b) {
  if (a != b)
    refuse("sqv.store.changed", "Retained store changed during inspection", 3);
}
using Entries = std::map<std::string, Stamp>;
Entries entries(int root, std::size_t bound, std::uint64_t bytes,
                std::int64_t end) {
  File copy(
      ::openat(root, ".", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
  io(copy.fd >= 0);
  DIR *dir = ::fdopendir(copy.fd);
  io(dir != nullptr);
  copy.fd = -1;
  Entries result;
  std::uint64_t total = 0;
  try {
    for (;;) {
      deadline(end);
      errno = 0;
      auto *item = ::readdir(dir);
      if (!item) {
        io(errno == 0);
        break;
      }
      std::string n = item->d_name;
      if (n == "." || n == "..")
        continue;
      if (result.size() >= bound)
        refuse("sqv.store.limit", "Retained store inspection bound exceeded",
               3);
      File f(::openat(root, n.c_str(),
                      O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
      auto st = status(f.fd, false);
      unchanged(st, linked(root, n));
      auto size = static_cast<std::uint64_t>(st.size);
      if (size > bytes - total)
        refuse("sqv.store.limit",
               "Retained store inspection byte bound exceeded", 3);
      total += size;
      check(result.emplace(n, st).second);
    }
    ::closedir(dir);
  } catch (...) {
    ::closedir(dir);
    throw;
  }
  return result;
}
std::string read(int root, const Entries &names, const std::string &name,
                 std::uint64_t bound, std::uint64_t &remaining,
                 std::int64_t end) {
  auto i = names.find(name);
  check(i != names.end());
  File f(::openat(root, name.c_str(),
                  O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
  auto st = status(f.fd, false);
  unchanged(i->second, st);
  auto size = static_cast<std::uint64_t>(st.size);
  if (size > bound || size > remaining)
    refuse("sqv.store.limit", "Retained store inspection byte bound exceeded",
           3);
  remaining -= size;
  std::string out(static_cast<std::size_t>(size), '\0');
  std::size_t at = 0;
  while (at < out.size()) {
    deadline(end);
    auto n = ::read(f.fd, out.data() + at,
                    std::min<std::size_t>(out.size() - at, 1U << 20));
    if (n < 0 && errno == EINTR)
      continue;
    io(n > 0);
    at += static_cast<std::size_t>(n);
  }
  char extra = 0;
  ssize_t n;
  do {
    n = ::read(f.fd, &extra, 1);
  } while (n < 0 && errno == EINTR);
  io(n == 0);
  unchanged(st, status(f.fd, false));
  unchanged(st, linked(root, name));
  return out;
}
std::string_view verified(const std::string &raw) {
  check(raw.size() >= 64);
  std::string_view body(raw.data(), raw.size() - 64);
  check(engine::sha256_hex(body) ==
        std::string_view(raw).substr(raw.size() - 64));
  return body;
}
template <std::size_t N> auto array(std::string_view &body) {
  std::array<std::uint8_t, N> out{};
  auto bytes = take(body, N);
  std::copy(bytes.begin(), bytes.end(), out.begin());
  return out;
}
bool sequence(std::string_view n, std::string_view prefix, std::uint64_t &out) {
  if (!n.starts_with(prefix) || n.size() != prefix.size() + 20)
    return false;
  n.remove_prefix(prefix.size());
  out = 0;
  for (char c : n) {
    if (c < '0' || c > '9' ||
        out > (UINT64_MAX - static_cast<unsigned>(c - '0')) / 10)
      return false;
    out = out * 10 + static_cast<unsigned>(c - '0');
  }
  return true;
}
std::string name(const char *prefix, std::uint64_t seq) {
  char buf[21]{};
  std::snprintf(buf, sizeof(buf), "%020llu",
                static_cast<unsigned long long>(seq));
  return std::string(prefix) + buf;
}
std::pair<File, std::string> open_parent(const std::string &path) {
  File parent(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
  io(parent.fd >= 0);
  std::string leaf;
  for (std::size_t start = 1; start < path.size();) {
    auto endpart = path.find('/', start);
    auto part =
        path.substr(start, endpart == std::string::npos ? std::string::npos
                                                        : endpart - start);
    require(!part.empty() && part != "." && part != "..");
    if (endpart == std::string::npos) {
      leaf = part;
      break;
    }
    File next(::openat(parent.fd, part.c_str(),
                       O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    struct stat st{};
    io(next.fd >= 0 && ::fstat(next.fd, &st) == 0);
    io((st.st_uid == ::geteuid() || st.st_uid == 0) &&
       ((st.st_mode & 0022) == 0 ||
        (st.st_uid == 0 && (st.st_mode & S_ISVTX) != 0)));
    parent = std::move(next);
    start = endpart + 1;
  }
  return {std::move(parent), std::move(leaf)};
}
bool nonzero(const sqfv::Generation &a) {
  return std::any_of(a.begin(), a.end(), [](auto b) { return b != 0; });
}
} // namespace
std::string_view take(std::string_view &in, std::size_t size) {
  check(in.size() >= size);
  auto out = in.substr(0, size);
  in.remove_prefix(size);
  return out;
}
std::uint64_t take64(std::string_view &in) {
  auto b = take(in, 8);
  std::uint64_t n = 0;
  for (unsigned char c : b)
    n = (n << 8) | c;
  return n;
}
std::string field(std::string_view &in, std::size_t max) {
  auto n = take64(in);
  check(n <= max);
  return std::string(take(in, static_cast<std::size_t>(n)));
}
Snapshot inspect(const Json &p, std::int64_t end, const Visitor &visit) {
#ifndef __APPLE__
  (void)p;
  (void)end;
  (void)visit;
  refuse("sqv.store.unsupported",
         "Retained store inspection requires local APFS", 3);
#endif
  fields(p, {"root", "expected_metadata_reference", "expected_store_generation",
             "max_read_bytes", "max_batches"});
  auto path = text(p, "root", 4096);
  auto expected = text(p, "expected_metadata_reference", 128);
  auto generation = fixed<16>(p, "expected_store_generation");
  const auto bytes = u64(p, "max_read_bytes"), bound = u64(p, "max_batches");
  require(bytes > 0 && bytes <= (512ULL << 20) && bound > 0 && bound <= 65536);
  require(path.size() > 1 && path.front() == '/' && path.back() != '/' &&
          path.find('\0') == std::string::npos);
  auto [parent, leaf] = open_parent(path);
  File root(::openat(parent.fd, leaf.c_str(),
                     O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
  auto root_stamp = status(root.fd, true);
  unchanged(root_stamp, linked(parent.fd, leaf));
#ifdef __APPLE__
  struct statfs volume{};
  io(::fstatfs(root.fd, &volume) == 0);
  if (std::string_view(volume.f_fstypename) != "apfs" ||
      (volume.f_flags & MNT_LOCAL) == 0)
    refuse("sqv.store.unsupported",
           "Retained store inspection requires local APFS", 3);
#endif
  File lock(::openat(root.fd, "writer.lock",
                     O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
  auto lock_stamp = status(lock.fd, false);
  check(lock_stamp.size == 0);
  if (::flock(lock.fd, LOCK_SH | LOCK_NB) != 0)
    refuse("sqv.store.busy", "Retained store is owned by an active writer", 3);
  unchanged(lock_stamp, linked(root.fd, "writer.lock"));
  auto initial =
      entries(root.fd, static_cast<std::size_t>(bound) * 2 + 8, bytes, end);
  Snapshot out;
  auto remaining = bytes;
  const auto raw = read(root.fd, initial, "metadata", 2U << 20, remaining, end);
  auto body = verified(raw);
  check(take(body, 8) == "SQPS0001");
  auto encoded = field(body, 65536);
  accepted(sqmv::Manifest::resolve(
      sqmv::ByteView(reinterpret_cast<const std::uint8_t *>(encoded.data()),
                     encoded.size()),
      expected, {65536, 4096, 128}, out.metadata));
  out.partition = field(body, 65535);
  out.producer_generation = array<16>(body);
  out.store_generation = array<16>(body);
  out.first_sequence = take64(body);
  out.max_frame_bytes = take64(body);
  out.max_store_bytes = take64(body);
  out.max_batches = take64(body);
  check(body.empty() && !out.partition.empty() &&
        nonzero(out.producer_generation) && nonzero(out.store_generation) &&
        out.store_generation == generation && out.max_frame_bytes > 0 &&
        out.max_frame_bytes <= (64U << 20) && out.max_store_bytes > 0 &&
        out.max_store_bytes <= (1ULL << 40) && out.max_batches > 0 &&
        out.max_batches <= 65536);
  sqfv::Binding bound_metadata;
  accepted(out.metadata.binding(bound_metadata));
  const auto descriptor_bytes = 48 + bound_metadata.metadata_ref.size() +
                                bound_metadata.dataset_revision.size() +
                                bound_metadata.schema_version.size() +
                                bound_metadata.layout_version.size() +
                                bound_metadata.access_scope.size() +
                                out.partition.size();
  check(descriptor_bytes <= 65536 &&
        24 + descriptor_bytes + 32 + 1 <= out.max_frame_bytes &&
        raw.size() + 4096 <= out.max_store_bytes);
  out.metadata_hash = engine::sha256_hex(raw);
  out.head_commit = std::string(64, '0');
  out.next_sequence = out.first_sequence;
  std::vector<std::uint64_t> commits, frames;
  for (const auto &[n, st] : initial) {
    out.physical_bytes += static_cast<std::uint64_t>(st.size);
    std::uint64_t seq = 0;
    if (n == "metadata" || n == "head" || n == "writer.lock")
      continue;
    if (sequence(n, "commit-", seq))
      commits.push_back(seq);
    else if (sequence(n, "frame-", seq))
      frames.push_back(seq);
    else if (n == "stage-head" || n == "stage-commit" || n == "stage-metadata")
      out.recovery_required = true;
    else if (n.size() == 97 &&
             sequence(std::string_view(n).substr(0, 32), "stage-frame-", seq) &&
             n[32] == '-') {
      (void)unhex(std::string_view(n).substr(33), 32);
      out.recovery_required = true;
    } else
      corrupt();
  }
  check(out.physical_bytes <= out.max_store_bytes &&
        commits.size() <= out.max_batches && frames.size() <= out.max_batches);
  if (commits.size() > bound || frames.size() > bound)
    refuse("sqv.store.limit", "Retained store inspection batch bound exceeded",
           3);
  std::sort(commits.begin(), commits.end());
  std::sort(frames.begin(), frames.end());
  if (initial.contains("head")) {
    const auto head = read(root.fd, initial, "head", 1024, remaining, end);
    body = verified(head);
    check(take(body, 8) == "SQPH0001" && take(body, 64) == out.metadata_hash);
    out.committed_batches = take64(body);
    out.head_commit = take(body, 64);
    check(body.empty() && out.committed_batches <= commits.size() &&
          out.committed_batches <= frames.size());
  } else
    out.recovery_required = true;
  if (commits.size() != out.committed_batches ||
      frames.size() != out.committed_batches)
    out.recovery_required = true;
  sqfv::Context context;
  accepted(
      sqfv::Context::create({out.max_frame_bytes, out.max_frame_bytes, 65536,
                             4 * out.max_frame_bytes + (1U << 20), 1},
                            context));
  std::string previous(64, '0');
  for (std::uint64_t i = 0; i < out.committed_batches; ++i) {
    deadline(end);
    check(!out.sequence_exhausted &&
          commits[static_cast<std::size_t>(i)] == out.next_sequence &&
          frames[static_cast<std::size_t>(i)] == out.next_sequence);
    const auto commit =
        read(root.fd, initial, name("commit-", out.next_sequence), 1024,
             remaining, end);
    body = verified(commit);
    check(take(body, 8) == "SQPC0001" && take(body, 64) == out.metadata_hash &&
          array<16>(body) == out.store_generation &&
          array<16>(body) == out.producer_generation &&
          take64(body) == out.next_sequence);
    auto frame_bytes = take64(body);
    auto content = array<32>(body);
    auto frame_hash = take(body, 64);
    check(take(body, 64) == previous && body.empty());
    previous = commit.substr(commit.size() - 64);
    const auto frame = read(root.fd, initial, name("frame-", out.next_sequence),
                            out.max_frame_bytes, remaining, end);
    check(frame.size() == frame_bytes &&
          engine::sha256_hex(frame) == frame_hash);
    sqfv::Batch batch;
    accepted(sqfv::frame_decode(
        context,
        sqfv::ByteView(reinterpret_cast<const std::uint8_t *>(frame.data()),
                       frame.size()),
        batch));
    check(batch.content_id() == content);
    const auto &d = batch.descriptor();
    accepted(out.metadata.verify_binding(d.binding));
    check(d.partition == out.partition &&
          d.producer_generation == out.producer_generation &&
          d.batch_sequence == out.next_sequence);
    if (visit) {
      sqfv::Lease lease;
      accepted(batch.acquire(d.binding.access_scope, lease));
      visit(out, batch, lease.payload());
    }
    if (out.next_sequence == UINT64_MAX)
      out.sequence_exhausted = true;
    else
      ++out.next_sequence;
  }
  check(previous == out.head_commit);
  auto final =
      entries(root.fd, static_cast<std::size_t>(bound) * 2 + 8, bytes, end);
  if (final != initial)
    refuse("sqv.store.changed", "Retained store changed during inspection", 3);
  auto [path_parent, path_leaf] = open_parent(path);
  unchanged(root_stamp, linked(path_parent.fd, path_leaf));
  unchanged(root_stamp, status(root.fd, true));
  unchanged(root_stamp, linked(parent.fd, leaf));
  unchanged(lock_stamp, status(lock.fd, false));
  unchanged(lock_stamp, linked(root.fd, "writer.lock"));
  deadline(end);
  return out;
}
Json summary(const Snapshot &s) {
  return {{"state", s.recovery_required ? "recovery_required" : "clean"},
          {"integrity_scope", s.recovery_required ? "published_head_prefix_only"
                                                  : "all_retained_frames"},
          {"metadata_reference", s.metadata.reference()},
          {"store_generation", hex(s.store_generation)},
          {"producer_generation", hex(s.producer_generation)},
          {"partition_hex_chunks", hex_chunks(s.partition)},
          {"committed_batches", std::to_string(s.committed_batches)},
          {"next_sequence", std::to_string(s.next_sequence)},
          {"sequence_exhausted", s.sequence_exhausted},
          {"physical_bytes", std::to_string(s.physical_bytes)},
          {"head_commit_sha256", s.head_commit},
          {"tail_validated", !s.recovery_required}};
}
} // namespace symphony::sqpv::inspection
