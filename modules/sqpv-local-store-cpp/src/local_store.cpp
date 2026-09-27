#include "symphony/sqpv/local_store.hpp"
#include <algorithm>
#include <array>
#include <cerrno>
#include <cstdio>
#include <cstring>
#include <dirent.h>
#include <fcntl.h>
#include <limits>
#include <mutex>
#include <new>
#include <string_view>
#include <symphony/knowledge/engine/digest.hpp>
#include <sys/file.h>
#include <sys/stat.h>
#ifdef __APPLE__
#include <sys/mount.h>
#endif
#include <unistd.h>
#include <utility>
#include <vector>
#ifdef SQPV_TESTING
#include <signal.h>
#endif

namespace symphony::sqpv {
namespace {
struct Failure {
  Status status;
};
[[noreturn]] void fail(Status s) { throw Failure{s}; }
void require(bool yes, Status s = Status::corrupt) {
  if (!yes)
    fail(s);
}
struct File {
  int fd = -1;
  explicit File(int n = -1) : fd(n) {}
  ~File() {
    if (fd >= 0)
      ::close(fd);
  }
  File(File &&other) noexcept : fd(std::exchange(other.fd, -1)) {}
  File &operator=(File &&other) noexcept {
    if (this != &other) {
      if (fd >= 0)
        ::close(fd);
      fd = std::exchange(other.fd, -1);
    }
    return *this;
  }
  File(const File &) = delete;
  File &operator=(const File &) = delete;
};
constexpr std::uint64_t max_frame = 64U << 20;
constexpr std::uint32_t max_batches = 65536;
constexpr std::size_t max_meta = 2U << 20;
constexpr std::size_t max_record = 1024;
std::string hash(std::string_view bytes) {
  return knowledge::engine::sha256_hex(bytes);
}
void u64(std::string &out, std::uint64_t value) {
  for (int n = 7; n >= 0; --n)
    out.push_back(static_cast<char>(value >> (n * 8)));
}
std::uint64_t take64(std::string_view &in) {
  require(in.size() >= 8);
  std::uint64_t n = 0;
  for (int i = 0; i < 8; ++i)
    n = (n << 8) | static_cast<unsigned char>(in[static_cast<std::size_t>(i)]);
  in.remove_prefix(8);
  return n;
}
std::string_view take(std::string_view &in, std::size_t count) {
  require(in.size() >= count);
  auto out = in.substr(0, count);
  in.remove_prefix(count);
  return out;
}
template <std::size_t N>
void bytes(std::string &out, const std::array<std::uint8_t, N> &value) {
  out.append(reinterpret_cast<const char *>(value.data()), N);
}
template <std::size_t N>
std::array<std::uint8_t, N> take_array(std::string_view &in) {
  std::array<std::uint8_t, N> out{};
  auto value = take(in, N);
  std::memcpy(out.data(), value.data(), N);
  return out;
}
void field(std::string &out, std::string_view text) {
  u64(out, text.size());
  out.append(text);
}
bool hex(std::string_view s) {
  return s.size() == 64 && std::all_of(s.begin(), s.end(), [](char c) {
           return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f');
         });
}
bool nonzero(const sqfv::Generation &g) {
  return std::any_of(g.begin(), g.end(), [](auto v) { return v != 0; });
}
std::string sealed(std::string body) {
  body += hash(body);
  return body;
}
std::string_view verified(const std::string &raw) {
  require(raw.size() >= 64);
  std::string_view body(raw.data(), raw.size() - 64);
  require(hash(body) == std::string_view(raw).substr(raw.size() - 64));
  return body;
}
std::string sequence_name(const char *kind, std::uint64_t seq) {
  char text[21]{};
  std::snprintf(text, sizeof(text), "%020llu",
                static_cast<unsigned long long>(seq));
  return std::string(kind) + text;
}
bool parse_sequence(std::string_view s, const char *prefix, std::uint64_t &n) {
  const std::string_view p(prefix);
  if (!s.starts_with(p) || s.size() != p.size() + 20)
    return false;
  s.remove_prefix(p.size());
  n = 0;
  for (char c : s) {
    if (c < '0' || c > '9' ||
        n > (std::numeric_limits<std::uint64_t>::max() -
             static_cast<unsigned>(c - '0')) /
                10)
      return false;
    n = n * 10 + static_cast<unsigned>(c - '0');
  }
  return true;
}
void sync_file(int fd) {
  while (::fsync(fd) != 0) {
    if (errno != EINTR)
      fail(Status::io_error);
  }
}
void private_file(int fd, bool directory) {
  struct stat st{};
  require(fd >= 0, Status::unsafe_path);
  require(::fstat(fd, &st) == 0, Status::io_error);
  require(st.st_uid == ::geteuid() &&
              (st.st_mode & 07777) == (directory ? 0700 : 0600) &&
              (directory ? S_ISDIR(st.st_mode)
                         : (S_ISREG(st.st_mode) && st.st_nlink == 1)),
          Status::unsafe_path);
}
std::vector<std::string> names(int fd, std::size_t limit) {
  File copy(::openat(fd, ".", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
  require(copy.fd >= 0, Status::io_error);
  DIR *dir = ::fdopendir(copy.fd);
  require(dir != nullptr, Status::io_error);
  copy.fd = -1;
  std::vector<std::string> result;
  try {
    errno = 0;
    while (auto *item = ::readdir(dir)) {
      std::string n(item->d_name);
      if (n == "." || n == "..")
        continue;
      require(result.size() < limit, Status::limit);
      result.push_back(std::move(n));
      errno = 0;
    }
    require(errno == 0, Status::io_error);
    ::closedir(dir);
  } catch (...) {
    ::closedir(dir);
    throw;
  }
  return result;
}
std::string read_file(int directory, const std::string &name,
                      std::uint64_t limit, bool optional = false) {
  File file(::openat(directory, name.c_str(),
                     O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
  if (file.fd < 0) {
    if (errno == ENOENT && optional)
      return {};
    fail(errno == ENOENT ? Status::missing : Status::unsafe_path);
  }
  private_file(file.fd, false);
  struct stat st{};
  require(::fstat(file.fd, &st) == 0, Status::io_error);
  require(st.st_size >= 0 && static_cast<std::uint64_t>(st.st_size) <= limit,
          Status::limit);
  std::string out(static_cast<std::size_t>(st.st_size), '\0');
  std::size_t at = 0;
  while (at < out.size()) {
    auto n = ::read(file.fd, out.data() + at, out.size() - at);
    if (n < 0 && errno == EINTR)
      continue;
    require(n > 0, Status::io_error);
    at += static_cast<std::size_t>(n);
  }
  char tail;
  ssize_t n;
  do {
    n = ::read(file.fd, &tail, 1);
  } while (n < 0 && errno == EINTR);
  require(n == 0, Status::corrupt);
  return out;
}
void write_file(int directory, const std::string &name, std::string_view data) {
  File file(::openat(directory, name.c_str(),
                     O_CREAT | O_RDWR | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC,
                     0600));
  require(file.fd >= 0, Status::io_error);
  private_file(file.fd, false);
  require(::ftruncate(file.fd, 0) == 0, Status::io_error);
  std::size_t at = 0;
  while (at < data.size()) {
    auto n = ::write(file.fd, data.data() + at, data.size() - at);
    if (n < 0 && errno == EINTR)
      continue;
    require(n > 0, Status::io_error);
    at += static_cast<std::size_t>(n);
  }
  sync_file(file.fd);
}
void exclusive_rename(int directory, const std::string &from,
                      const std::string &to) {
#ifdef __APPLE__
  require(::renameatx_np(directory, from.c_str(), directory, to.c_str(),
                         RENAME_EXCL) == 0,
          Status::io_error);
#else
  (void)directory;
  (void)from;
  (void)to;
  fail(Status::unsupported);
#endif
}
#ifdef SQPV_TESTING
int crash_point = 0;
int failure_point = 0;
int pause_point = 0;
int pause_notify = -1;
int pause_resume = -1;
void checkpoint(int point) {
  if (pause_point == point) {
    char signal = 1;
    require(::write(pause_notify, &signal, 1) == 1, Status::io_error);
    require(::read(pause_resume, &signal, 1) == 1, Status::io_error);
  }
  if (crash_point == point) {
    ::kill(::getpid(), SIGKILL);
    ::_exit(127);
  }
  if (failure_point == point)
    fail(Status::io_error);
}
#else
void checkpoint(int) {}
#endif
Status map_flow(sqfv::Status s) {
  if (s == sqfv::Status::ok)
    return Status::ok;
  if (s == sqfv::Status::invalid_argument)
    return Status::invalid_argument;
  if (s == sqfv::Status::no_memory)
    return Status::no_memory;
  if (s == sqfv::Status::limit)
    return Status::limit;
  return Status::corrupt;
}
} // namespace
namespace detail {
struct StoreHandle {
  File parent, root, lock;
  std::string leaf, metadata, metadata_hash, staged_name;
  Options options;
  sqfv::Binding binding;
  dev_t device{};
  ino_t inode{}, lock_inode{};
  pid_t pid = ::getpid();
  bool poisoned = false, mutated = false;
  std::mutex mutex;
  std::vector<Receipt> receipts;
  std::uint64_t total_bytes = 0;
  void identity() {
    require(!poisoned, Status::closed);
    require(pid == ::getpid(), Status::stale);
    private_file(root.fd, true);
    private_file(lock.fd, false);
    struct stat st{};
    require(::fstatat(parent.fd, leaf.c_str(), &st, AT_SYMLINK_NOFOLLOW) == 0 &&
                S_ISDIR(st.st_mode) && st.st_dev == device &&
                st.st_ino == inode,
            Status::stale);
    require(::fstatat(root.fd, "writer.lock", &st, AT_SYMLINK_NOFOLLOW) == 0 &&
                S_ISREG(st.st_mode) && st.st_ino == lock_inode &&
                st.st_dev == device,
            Status::stale);
  }
  std::uint64_t next() const {
    if (receipts.empty())
      return options.first_sequence;
    return receipts.back().batch_sequence == UINT64_MAX
               ? UINT64_MAX
               : receipts.back().batch_sequence + 1;
  }
  bool exhausted() const {
    return !receipts.empty() && receipts.back().batch_sequence == UINT64_MAX;
  }
  std::string head() const {
    std::string raw = "SQPH0001";
    raw += metadata_hash;
    u64(raw, receipts.size());
    raw +=
        receipts.empty() ? std::string(64, '0') : receipts.back().commit_sha256;
    return sealed(std::move(raw));
  }
  void publish_head() {
    auto raw = head();
    auto old = read_file(root.fd, "stage-head", max_record, true);
    require(old.size() <= raw.size() && raw.starts_with(old), Status::corrupt);
    mutated = true;
    write_file(root.fd, "stage-head", raw);
    require(read_file(root.fd, "stage-head", max_record) == raw);
    checkpoint(7);
    identity();
    require(::renameat(root.fd, "stage-head", root.fd, "head") == 0,
            Status::io_error);
    checkpoint(8);
    sync_file(root.fd);
    checkpoint(9);
  }
  void scan() {
    identity();
    total_bytes = 0;
    staged_name.clear();
    auto entries = names(
        root.fd, static_cast<std::size_t>(options.limits.max_batches) * 2 + 8);
    for (const auto &n : entries) {
      bool allowed = n == "metadata" || n == "head" || n == "writer.lock" ||
                     n == "stage-metadata" || n == "stage-commit" ||
                     n == "stage-head";
      std::uint64_t seq = 0;
      allowed = allowed || parse_sequence(n, "frame-", seq) ||
                parse_sequence(n, "commit-", seq);
      if (n.starts_with("stage-frame-")) {
        std::uint64_t pending = 0;
        require(n.size() == 12 + 20 + 1 + 64 &&
                    parse_sequence(std::string_view(n).substr(0, 32),
                                   "stage-frame-", pending) &&
                    n[32] == '-' && hex(std::string_view(n).substr(33)),
                Status::corrupt);
        require(staged_name.empty(), Status::corrupt);
        staged_name = n;
        allowed = true;
      }
      require(allowed, Status::corrupt);
      File f(::openat(root.fd, n.c_str(),
                      O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
      private_file(f.fd, false);
      struct stat st{};
      require(::fstat(f.fd, &st) == 0 && st.st_size >= 0, Status::io_error);
      auto size = static_cast<std::uint64_t>(st.st_size);
      require(size <= options.limits.max_store_bytes - total_bytes,
              Status::limit);
      total_bytes += size;
    }
  }
};
} // namespace detail
namespace {
using detail::StoreHandle;
std::string metadata(const sqmv::Manifest &manifest, const Options &options) {
  std::string raw = "SQPS0001";
  auto encoded = manifest.encoded();
  field(raw, std::string_view(reinterpret_cast<const char *>(encoded.data()),
                              encoded.size()));
  field(raw, options.partition);
  bytes(raw, options.producer_generation);
  bytes(raw, options.store_generation);
  u64(raw, options.first_sequence);
  u64(raw, options.limits.max_frame_bytes);
  u64(raw, options.limits.max_store_bytes);
  u64(raw, options.limits.max_batches);
  require(raw.size() <= max_meta - 64, Status::limit);
  return sealed(std::move(raw));
}
std::string commit_record(const StoreHandle &h, const Receipt &r,
                          std::string_view previous) {
  std::string out = "SQPC0001";
  out += h.metadata_hash;
  bytes(out, r.store_generation);
  bytes(out, r.producer_generation);
  u64(out, r.batch_sequence);
  u64(out, r.frame_bytes);
  bytes(out, r.content_id);
  out += r.frame_sha256;
  out += previous;
  return sealed(std::move(out));
}
Receipt decode_commit(const StoreHandle &h, const std::string &raw,
                      std::string_view previous) {
  auto body = verified(raw);
  require(take(body, 8) == "SQPC0001" && take(body, 64) == h.metadata_hash);
  Receipt out;
  out.store_generation = take_array<16>(body);
  out.producer_generation = take_array<16>(body);
  out.batch_sequence = take64(body);
  out.frame_bytes = take64(body);
  out.content_id = take_array<32>(body);
  out.frame_sha256 = take(body, 64);
  require(hex(out.frame_sha256) && take(body, 64) == previous && body.empty() &&
          out.store_generation == h.options.store_generation &&
          out.producer_generation == h.options.producer_generation);
  out.commit_sha256 = raw.substr(raw.size() - 64);
  return out;
}
void binding(const StoreHandle &h, const sqfv::Descriptor &d) {
  const auto &b = d.binding;
  const auto &x = h.binding;
  require(b.metadata_ref == x.metadata_ref &&
              b.dataset_revision == x.dataset_revision &&
              b.schema_version == x.schema_version &&
              b.layout_version == x.layout_version &&
              b.access_scope == x.access_scope &&
              d.partition == h.options.partition &&
              d.producer_generation == h.options.producer_generation,
          Status::binding_mismatch);
}
void inspect_frame(StoreHandle &h, const std::string &frame,
                   std::uint64_t sequence, const Receipt *receipt) {
  sqfv::Limits limits{h.options.limits.max_frame_bytes,
                      h.options.limits.max_frame_bytes, 65536,
                      4 * h.options.limits.max_frame_bytes + 1048576, 1};
  sqfv::Context context;
  auto s = sqfv::Context::create(limits, context);
  require(s == sqfv::Status::ok, map_flow(s));
  sqfv::Batch batch;
  s = sqfv::frame_decode(
      context,
      sqfv::ByteView(reinterpret_cast<const std::uint8_t *>(frame.data()),
                     frame.size()),
      batch);
  require(s == sqfv::Status::ok, map_flow(s));
  binding(h, batch.descriptor());
  require(batch.descriptor().batch_sequence == sequence);
  if (receipt)
    require(receipt->content_id == batch.content_id() &&
            receipt->frame_bytes == frame.size() &&
            receipt->frame_sha256 == hash(frame));
}
void prepare(const std::string &path, const sqmv::Manifest &manifest,
             const Options &options, bool create,
             std::unique_ptr<StoreHandle> &h) {
#ifndef __APPLE__
  (void)path;
  (void)manifest;
  (void)options;
  (void)create;
  fail(Status::unsupported);
#endif
  require(!path.empty() && path.front() == '/' && path.size() > 1 &&
              path.back() != '/' && path.find('\0') == std::string::npos,
          Status::invalid_argument);
  require(!options.partition.empty() && options.partition.size() <= 65535 &&
              nonzero(options.producer_generation) &&
              nonzero(options.store_generation) &&
              options.limits.max_frame_bytes > 0 &&
              options.limits.max_frame_bytes <= max_frame &&
              options.limits.max_batches > 0 &&
              options.limits.max_batches <= max_batches &&
              options.limits.max_store_bytes > 0 &&
              options.limits.max_store_bytes <= (1ULL << 40),
          Status::invalid_argument);
  h = std::make_unique<StoreHandle>();
  h->options = options;
  const auto metadata_status = manifest.binding(h->binding);
  require(metadata_status == sqmv::Status::ok,
          metadata_status == sqmv::Status::no_memory
              ? Status::no_memory
              : Status::invalid_argument);
  const auto descriptor_bytes =
      48 + h->binding.metadata_ref.size() + h->binding.dataset_revision.size() +
      h->binding.schema_version.size() + h->binding.layout_version.size() +
      h->binding.access_scope.size() + options.partition.size();
  require(descriptor_bytes <= 65536 &&
              24 + descriptor_bytes + 32 + 1 <= options.limits.max_frame_bytes,
          Status::limit);
  h->metadata = metadata(manifest, options);
  h->metadata_hash = hash(h->metadata);
  require(h->metadata.size() + 4096 <= options.limits.max_store_bytes,
          Status::limit);
  h->parent =
      File(::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
  require(h->parent.fd >= 0, Status::io_error);
  std::size_t start = 1;
  while (start < path.size()) {
    auto end = path.find('/', start);
    auto part = path.substr(start, end == std::string::npos ? std::string::npos
                                                            : end - start);
    require(!part.empty() && part != "." && part != "..",
            Status::invalid_argument);
    if (end == std::string::npos) {
      h->leaf = part;
      break;
    }
    File next(::openat(h->parent.fd, part.c_str(),
                       O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    require(next.fd >= 0, Status::unsafe_path);
    struct stat st{};
    require(::fstat(next.fd, &st) == 0, Status::io_error);
    require((st.st_uid == ::geteuid() || st.st_uid == 0) &&
                ((st.st_mode & 0022) == 0 ||
                 (st.st_uid == 0 && (st.st_mode & S_ISVTX) != 0)),
            Status::unsafe_path);
    h->parent = std::move(next);
    start = end + 1;
  }
  h->root = File(::openat(h->parent.fd, h->leaf.c_str(),
                          O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
  require(h->root.fd >= 0, Status::unsafe_path);
  private_file(h->root.fd, true);
#ifdef __APPLE__
  struct statfs volume{};
  require(::fstatfs(h->root.fd, &volume) == 0, Status::io_error);
  require(std::string_view(volume.f_fstypename) == "apfs" &&
              (volume.f_flags & MNT_LOCAL) != 0,
          Status::unsupported);
#endif
  struct stat st{};
  require(::fstat(h->root.fd, &st) == 0, Status::io_error);
  h->device = st.st_dev;
  h->inode = st.st_ino;
  auto initial = names(
      h->root.fd, static_cast<std::size_t>(options.limits.max_batches) * 2 + 8);
  const auto metadata_stage = "stage-metadata-" + h->metadata_hash;
  if (create)
    for (const auto &n : initial)
      require(n == "writer.lock" || n == metadata_stage, Status::conflict);
  if (create &&
      std::find(initial.begin(), initial.end(), "writer.lock") == initial.end())
    h->mutated = true;
  h->lock = File(::openat(h->root.fd, "writer.lock",
                          O_RDWR | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC |
                              (create ? O_CREAT : 0),
                          0600));
  private_file(h->lock.fd, false);
  if (::flock(h->lock.fd, LOCK_EX | LOCK_NB) != 0)
    fail((errno == EAGAIN || errno == EWOULDBLOCK) ? Status::busy
                                                   : Status::io_error);
  require(::fstat(h->lock.fd, &st) == 0, Status::io_error);
  h->lock_inode = st.st_ino;
  require(st.st_size == 0, Status::corrupt);
  if (create)
    checkpoint(10);
  if (create) {
    auto old = read_file(h->root.fd, metadata_stage, max_meta, true);
    require(old.size() <= h->metadata.size() && h->metadata.starts_with(old),
            Status::conflict);
    h->mutated = true;
    write_file(h->root.fd, metadata_stage, h->metadata);
    require(read_file(h->root.fd, metadata_stage, max_meta) == h->metadata);
    checkpoint(11);
    exclusive_rename(h->root.fd, metadata_stage, "metadata");
    checkpoint(12);
    sync_file(h->root.fd);
    checkpoint(13);
    h->publish_head();
    // Creation remains uncertain until the caller completes recovery
    // verification.
  }
  require(read_file(h->root.fd, "metadata", max_meta) == h->metadata,
          Status::binding_mismatch);
}
void recover(StoreHandle &h) {
  h.scan();
  auto entries =
      names(h.root.fd,
            static_cast<std::size_t>(h.options.limits.max_batches) * 2 + 8);
  std::vector<std::uint64_t> commits, frames;
  for (const auto &n : entries) {
    std::uint64_t seq = 0;
    if (parse_sequence(n, "commit-", seq))
      commits.push_back(seq);
    if (parse_sequence(n, "frame-", seq))
      frames.push_back(seq);
    require(n != "stage-metadata", Status::corrupt);
  }
  std::sort(commits.begin(), commits.end());
  std::sort(frames.begin(), frames.end());
  require(commits.size() <= h.options.limits.max_batches &&
              frames.size() <= commits.size() + 1,
          Status::corrupt);
  std::uint64_t sequence = h.options.first_sequence;
  std::string previous(64, '0');
  h.receipts.reserve(h.options.limits.max_batches);
  for (std::size_t i = 0; i < commits.size(); ++i) {
    require(commits[i] == sequence && i < frames.size() &&
            frames[i] == sequence);
    auto raw =
        read_file(h.root.fd, sequence_name("commit-", sequence), max_record);
    auto receipt = decode_commit(h, raw, previous);
    require(receipt.batch_sequence == sequence);
    auto frame = read_file(h.root.fd, sequence_name("frame-", sequence),
                           h.options.limits.max_frame_bytes);
    inspect_frame(h, frame, sequence, &receipt);
    previous = receipt.commit_sha256;
    h.receipts.push_back(std::move(receipt));
    if (sequence == UINT64_MAX)
      require(i + 1 == commits.size());
    else
      ++sequence;
  }
  if (frames.size() > commits.size()) {
    require(commits.size() < h.options.limits.max_batches);
    require(!h.exhausted() && frames.back() == h.next());
    auto frame = read_file(h.root.fd, sequence_name("frame-", frames.back()),
                           h.options.limits.max_frame_bytes);
    inspect_frame(h, frame, frames.back(), nullptr);
  }
  if (!h.staged_name.empty()) {
    require(commits.size() < h.options.limits.max_batches);
    std::uint64_t staged = 0;
    require(!h.exhausted() && frames.size() == commits.size() &&
            parse_sequence(std::string_view(h.staged_name).substr(0, 32),
                           "stage-frame-", staged) &&
            staged == h.next());
    auto raw =
        read_file(h.root.fd, h.staged_name, h.options.limits.max_frame_bytes);
    if (raw.size() > 0 &&
        hash(raw) == std::string_view(h.staged_name).substr(33))
      inspect_frame(h, raw, staged, nullptr);
  }
  const auto contains = [&](std::string_view name) {
    return std::find(entries.begin(), entries.end(), name) != entries.end();
  };
  auto head = read_file(h.root.fd, "head", max_record, true);
  require(!contains("head") || !head.empty(), Status::corrupt);
  std::size_t acknowledged = 0;
  if (!head.empty()) {
    auto body = verified(head);
    require(take(body, 8) == "SQPH0001" && take(body, 64) == h.metadata_hash);
    auto count = take64(body);
    require(count <= h.receipts.size());
    auto wanted =
        count == 0
            ? std::string(64, '0')
            : h.receipts[static_cast<std::size_t>(count - 1)].commit_sha256;
    require(take(body, 64) == wanted && body.empty());
    acknowledged = static_cast<std::size_t>(count);
  } else
    require(h.receipts.empty(), Status::missing);
  // A complete commit may have reached publication before the head/ack barrier.
  if (acknowledged < h.receipts.size() || head.empty()) {
    for (std::size_t i = acknowledged; i < h.receipts.size(); ++i) {
      File f(::openat(
          h.root.fd,
          sequence_name("frame-", h.receipts[i].batch_sequence).c_str(),
          O_RDONLY | O_NOFOLLOW | O_CLOEXEC));
      require(f.fd >= 0, Status::missing);
      sync_file(f.fd);
      File c(::openat(
          h.root.fd,
          sequence_name("commit-", h.receipts[i].batch_sequence).c_str(),
          O_RDONLY | O_NOFOLLOW | O_CLOEXEC));
      require(c.fd >= 0, Status::missing);
      sync_file(c.fd);
    }
    sync_file(h.root.fd);
    h.publish_head();
  }
  auto scratch = read_file(h.root.fd, "stage-commit", max_record, true);
  if (contains("stage-commit")) {
    require(frames.size() > commits.size(), Status::corrupt);
    auto frame = read_file(h.root.fd, sequence_name("frame-", h.next()),
                           h.options.limits.max_frame_bytes);
    (void)frame;
    require(scratch.size() <= max_record);
  }
  struct stat scratch_status{};
  require(::fstatat(h.root.fd, "stage-head", &scratch_status,
                    AT_SYMLINK_NOFOLLOW) != 0 &&
              errno == ENOENT,
          Status::corrupt);
  h.scan();
  h.mutated = false;
}
Status exception_status(StoreHandle *h) {
  if (h && h->poisoned)
    return Status::closed;
  try {
    throw;
  } catch (const Failure &f) {
    if (h && h->mutated) {
      h->poisoned = true;
      return Status::outcome_uncertain;
    }
    return f.status;
  } catch (const std::bad_alloc &) {
    if (h && h->mutated) {
      h->poisoned = true;
      return Status::outcome_uncertain;
    }
    return Status::no_memory;
  } catch (...) {
    if (h && h->mutated) {
      h->poisoned = true;
      return Status::outcome_uncertain;
    }
    return Status::io_error;
  }
}
} // namespace
#ifdef SQPV_TESTING
namespace testing {
void crash_at(int n) noexcept { crash_point = n; }
void fail_at(int n) noexcept { failure_point = n; }
void pause_at(int n, int notify, int resume) noexcept {
  pause_point = n;
  pause_notify = notify;
  pause_resume = resume;
}
} // namespace testing
#endif
Store::Store() noexcept = default;
Store::~Store() noexcept = default;
Store::Store(Store &&) noexcept = default;
Store &Store::operator=(Store &&) noexcept = default;
Store::operator bool() const noexcept { return impl_ != nullptr; }
void Store::reset() noexcept { impl_.reset(); }
Status Store::create(const std::string &path, const sqmv::Manifest &m,
                     const Options &o, Store &out) noexcept {
  std::unique_ptr<StoreHandle> h;
  try {
    prepare(path, m, o, true, h);
    recover(*h);
    Store ready;
    ready.impl_ = std::move(h);
    out = std::move(ready);
    return Status::ok;
  } catch (...) {
    return exception_status(h.get());
  }
}
Status Store::open(const std::string &path, const sqmv::Manifest &m,
                   const Options &o, Store &out) noexcept {
  std::unique_ptr<StoreHandle> h;
  try {
    prepare(path, m, o, false, h);
    recover(*h);
    Store ready;
    ready.impl_ = std::move(h);
    out = std::move(ready);
    return Status::ok;
  } catch (...) {
    return exception_status(h.get());
  }
}
Status Store::append(sqfv::Context &context, const sqfv::Batch &batch,
                     Receipt &out) noexcept {
  auto *h = impl_.get();
  if (!h)
    return Status::closed;
  if (h->pid != ::getpid())
    return Status::stale;
  std::unique_lock guard(h->mutex, std::defer_lock);
  try {
    guard.lock();
    h->identity();
    require(static_cast<bool>(batch), Status::invalid_argument);
    binding(*h, batch.descriptor());
    auto seq = batch.descriptor().batch_sequence;
    std::size_t size = 0;
    auto fs = sqfv::frame_measure(context, batch, size);
    require(fs == sqfv::Status::ok, map_flow(fs));
    require(size <= h->options.limits.max_frame_bytes, Status::limit);
    std::string frame(size, '\0');
    fs = sqfv::frame_encode(
        context, batch,
        sqfv::MutableBytes(reinterpret_cast<std::uint8_t *>(frame.data()),
                           frame.size()),
        size);
    require(fs == sqfv::Status::ok, map_flow(fs));
    auto frame_hash = hash(frame);
    if (seq >= h->options.first_sequence &&
        seq - h->options.first_sequence < h->receipts.size()) {
      const auto &old = h->receipts[static_cast<std::size_t>(
          seq - h->options.first_sequence)];
      require(old.content_id == batch.content_id() &&
                  old.frame_sha256 == frame_hash,
              Status::conflict);
      auto retained = read_file(h->root.fd, sequence_name("frame-", seq),
                                h->options.limits.max_frame_bytes);
      require(retained == frame);
      auto raw =
          read_file(h->root.fd, sequence_name("commit-", seq), max_record);
      require(hash(verified(raw)) == old.commit_sha256);
      Receipt ready = old;
      out = std::move(ready);
      return Status::duplicate;
    }
    require(!h->exhausted(), Status::limit);
    require(seq >= h->next(), Status::stale);
    require(seq == h->next(), Status::gap);
    require(h->receipts.size() < h->options.limits.max_batches, Status::limit);
    h->scan();
    std::string staged = sequence_name("stage-frame-", seq) + "-" + frame_hash;
    require(h->staged_name.empty() || h->staged_name == staged,
            Status::conflict);
    auto final_name = sequence_name("frame-", seq);
    auto existing = read_file(h->root.fd, final_name,
                              h->options.limits.max_frame_bytes, true);
    require(existing.empty() || existing == frame, Status::conflict);
    auto prior =
        read_file(h->root.fd, staged, h->options.limits.max_frame_bytes, true);
    require(prior.size() <= frame.size() && frame.starts_with(prior),
            Status::corrupt);
    Receipt result{h->options.store_generation,
                   h->options.producer_generation,
                   seq,
                   frame.size(),
                   batch.content_id(),
                   frame_hash,
                   {}};
    auto previous = h->receipts.empty() ? std::string(64, '0')
                                        : h->receipts.back().commit_sha256;
    auto commit = commit_record(*h, result, previous);
    result.commit_sha256 = commit.substr(commit.size() - 64);
    auto scratch = read_file(h->root.fd, "stage-commit", max_record, true);
    require(scratch.size() <= commit.size() && commit.starts_with(scratch),
            Status::conflict);
    // Reserve room for the largest possible state before publishing any bytes.
    const auto needed = (existing.empty() ? frame.size() - prior.size() : 0) +
                        commit.size() - scratch.size() + h->head().size();
    require(needed <= h->options.limits.max_store_bytes - h->total_bytes,
            Status::limit);
    Receipt output = result;
    h->receipts.reserve(h->options.limits.max_batches);
    h->mutated = true;
    if (existing.empty()) {
      write_file(h->root.fd, staged, frame);
      require(read_file(h->root.fd, staged,
                        h->options.limits.max_frame_bytes) == frame);
      checkpoint(1);
      h->identity();
      exclusive_rename(h->root.fd, staged, final_name);
      checkpoint(2);
      sync_file(h->root.fd);
      checkpoint(3);
    } else {
      File f(::openat(h->root.fd, final_name.c_str(),
                      O_RDONLY | O_NOFOLLOW | O_CLOEXEC));
      require(f.fd >= 0, Status::missing);
      sync_file(f.fd);
      sync_file(h->root.fd);
    }
    write_file(h->root.fd, "stage-commit", commit);
    require(read_file(h->root.fd, "stage-commit", max_record) == commit);
    checkpoint(4);
    h->identity();
    exclusive_rename(h->root.fd, "stage-commit", sequence_name("commit-", seq));
    checkpoint(5);
    sync_file(h->root.fd);
    checkpoint(6);
    h->receipts.push_back(std::move(result));
    h->publish_head();
    h->scan();
    h->mutated = false;
    out = std::move(output);
    return Status::ok;
  } catch (...) {
    return exception_status(guard.owns_lock() ? h : nullptr);
  }
}
Status Store::read(std::uint64_t seq, sqfv::Context &context, sqfv::Batch &out,
                   Receipt &receipt) noexcept {
  auto *h = impl_.get();
  if (!h)
    return Status::closed;
  if (h->pid != ::getpid())
    return Status::stale;
  std::unique_lock guard(h->mutex, std::defer_lock);
  try {
    guard.lock();
    h->identity();
    require(seq >= h->options.first_sequence &&
                seq - h->options.first_sequence < h->receipts.size(),
            Status::missing);
    auto r =
        h->receipts[static_cast<std::size_t>(seq - h->options.first_sequence)];
    auto raw = read_file(h->root.fd, sequence_name("commit-", seq), max_record);
    require(verified(raw).size() > 0 &&
            raw.substr(raw.size() - 64) == r.commit_sha256);
    auto frame = read_file(h->root.fd, sequence_name("frame-", seq),
                           h->options.limits.max_frame_bytes);
    require(frame.size() == r.frame_bytes && hash(frame) == r.frame_sha256);
    sqfv::Batch batch;
    auto s = sqfv::frame_decode(
        context,
        sqfv::ByteView(reinterpret_cast<const std::uint8_t *>(frame.data()),
                       frame.size()),
        batch);
    require(s == sqfv::Status::ok, map_flow(s));
    binding(*h, batch.descriptor());
    require(batch.content_id() == r.content_id &&
            batch.descriptor().batch_sequence == seq);
    out = std::move(batch);
    receipt = std::move(r);
    return Status::ok;
  } catch (...) {
    return exception_status(guard.owns_lock() ? h : nullptr);
  }
}
Status Store::snapshot(Snapshot &out) noexcept {
  auto *h = impl_.get();
  if (!h)
    return Status::closed;
  if (h->pid != ::getpid())
    return Status::stale;
  std::unique_lock guard(h->mutex, std::defer_lock);
  try {
    guard.lock();
    h->scan();
    auto pending = read_file(h->root.fd, sequence_name("frame-", h->next()),
                             h->options.limits.max_frame_bytes, true);
    out = {h->receipts.size(), h->next(), h->total_bytes, h->exhausted(),
           !h->staged_name.empty() || (!h->exhausted() && !pending.empty())};
    return Status::ok;
  } catch (...) {
    return exception_status(guard.owns_lock() ? h : nullptr);
  }
}
} // namespace symphony::sqpv
