#pragma once
// Native process-level acceptance support. Assertions stay active in release
// builds.
#include <algorithm>
#include <cerrno>
#include <chrono>
#include <cstdlib>
#include <cstring>
#include <dirent.h>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <functional>
#include <iostream>
#include <limits>
#include <map>
#include <nlohmann/json.hpp>
#include <set>
#include <signal.h>
#include <sstream>
#include <stdexcept>
#include <string>
#include <symphony/knowledge/engine/digest.hpp>
#include <sys/stat.h>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>
#include <vector>
extern char **environ;
namespace native_test {
using Json = nlohmann::json;
namespace fs = std::filesystem;
inline void require(bool condition, const std::string &message) {
  if (!condition)
    throw std::runtime_error(message);
}
#define NT_REQUIRE(...)                                                        \
  ::native_test::require(static_cast<bool>((__VA_ARGS__)),                     \
                         std::string(__FILE__) + ":" +                         \
                             std::to_string(__LINE__) + ": " #__VA_ARGS__)
inline std::string canonical(const Json &value) {
  return value.dump(-1, ' ', false, Json::error_handler_t::strict);
}
inline std::string digest(std::string_view bytes) {
  return symphony::knowledge::engine::tagged_sha256(bytes);
}
inline Json seal(Json value, const std::string &field = "digest") {
  value.erase(field);
  value[field] = digest(canonical(value));
  return value;
}
inline std::string read(const fs::path &path) {
  std::ifstream in(path, std::ios::binary);
  require(bool(in), "Cannot read " + path.string());
  return {std::istreambuf_iterator<char>(in), {}};
}
inline void write(const fs::path &path, std::string_view bytes) {
  if (!path.parent_path().empty())
    fs::create_directories(path.parent_path());
  std::ofstream out(path, std::ios::binary | std::ios::trunc);
  require(bool(out), "Cannot write " + path.string());
  out.write(bytes.data(), static_cast<std::streamsize>(bytes.size()));
  require(bool(out), "Short write " + path.string());
}
inline Json read_json(const fs::path &path) { return Json::parse(read(path)); }
inline void write_json(const fs::path &path, const Json &value) {
  write(path, value.dump(2) + "\n");
}
inline long long now_ms() {
  return std::chrono::duration_cast<std::chrono::milliseconds>(
             std::chrono::system_clock::now().time_since_epoch())
      .count();
}
struct TempDir {
  fs::path path;
  explicit TempDir(std::string prefix = "symphony-native-") {
    std::string pattern =
        (fs::temp_directory_path() / (prefix + "XXXXXX")).string();
    std::vector<char> buf(pattern.begin(), pattern.end());
    buf.push_back('\0');
    auto p = ::mkdtemp(buf.data());
    require(p != nullptr, "mkdtemp failed");
    path = fs::canonical(p);
  }
  ~TempDir() {
    std::error_code ec;
    fs::remove_all(path, ec);
  }
  TempDir(const TempDir &) = delete;
  TempDir &operator=(const TempDir &) = delete;
};
struct ProcessResult {
  int returncode;
  std::string stdout_text;
  std::string stderr_text;
};
// Match subprocess's close_fds boundary. Inspect the child's own descriptor
// table after fork so descriptors opened concurrently in the parent cannot
// escape the snapshot. Both supported POSIX hosts expose this table.
inline bool close_process_descriptors() {
  DIR *table = ::opendir("/dev/fd");
  if (table == nullptr)
    table = ::opendir("/proc/self/fd");
  if (table == nullptr)
    return false;
  const int table_fd = ::dirfd(table);
  std::vector<int> descriptors;
  bool complete = true;
  for (;;) {
    errno = 0;
    auto *entry = ::readdir(table);
    if (entry == nullptr) {
      complete = errno == 0;
      break;
    }
    char *end = nullptr;
    const auto number = std::strtol(entry->d_name, &end, 10);
    if (entry->d_name == end || *end != '\0' || number <= STDERR_FILENO ||
        number > std::numeric_limits<int>::max())
      continue;
    const auto fd = static_cast<int>(number);
    if (fd != table_fd)
      descriptors.push_back(fd);
  }
  if (::closedir(table) != 0)
    complete = false;
  if (!complete)
    return false;
  for (const int fd : descriptors)
    if (::close(fd) != 0 && errno != EBADF)
      return false;
  return true;
}
inline void terminate_process_group(pid_t child, bool reaped) noexcept {
  ::kill(-child, SIGKILL);
  if (!reaped) {
    ::kill(child, SIGKILL);
    while (::waitpid(child, nullptr, 0) < 0 && errno == EINTR) {
    }
  }
}
inline ProcessResult
run(const std::vector<std::string> &command, const std::string &input = "",
    int timeout_seconds = 40,
    const std::map<std::string, std::string> &environment = {},
    const fs::path &cwd = {}, bool clear_environment = false) {
  require(!command.empty(), "Empty process command");
  std::map<std::string, std::string> child_environment;
  if (!clear_environment) {
    for (char **entry = ::environ; entry != nullptr && *entry != nullptr;
         ++entry) {
      const std::string value = *entry;
      const auto separator = value.find('=');
      if (separator != std::string::npos)
        child_environment[value.substr(0, separator)] =
            value.substr(separator + 1);
    }
  }
  for (const auto &[key, value] : environment) {
    require(!key.empty() && key.find('=') == std::string::npos &&
                key.find('\0') == std::string::npos &&
                value.find('\0') == std::string::npos,
            "Invalid child environment entry");
    child_environment[key] = value;
  }
  std::vector<std::string> environment_entries;
  for (const auto &[key, value] : child_environment)
    environment_entries.push_back(key + "=" + value);
  std::vector<char *> environment_pointers;
  for (auto &entry : environment_entries)
    environment_pointers.push_back(entry.data());
  environment_pointers.push_back(nullptr);
  TempDir files("native-process-");
  write(files.path / "stdin", input);
  pid_t child = ::fork();
  require(child >= 0, "fork failed");
  if (child == 0) {
    if (::setpgid(0, 0) != 0)
      _exit(126);
    int in = ::open((files.path / "stdin").c_str(), O_RDONLY);
    int out = ::open((files.path / "stdout").c_str(),
                     O_WRONLY | O_CREAT | O_TRUNC, 0600);
    int err = ::open((files.path / "stderr").c_str(),
                     O_WRONLY | O_CREAT | O_TRUNC, 0600);
    if (in < 0 || out < 0 || err < 0)
      _exit(126);
    if (::dup2(in, STDIN_FILENO) < 0 || ::dup2(out, STDOUT_FILENO) < 0 ||
        ::dup2(err, STDERR_FILENO) < 0 || !close_process_descriptors())
      _exit(126);
    if (!cwd.empty() && ::chdir(cwd.c_str()) != 0)
      _exit(126);
    // Build an explicit, null-terminated environment before fork. In
    // particular, macOS setenv cannot safely rebuild an environment set to
    // nullptr.
    ::environ = environment_pointers.data();
    std::vector<char *> argv;
    for (const auto &arg : command)
      argv.push_back(const_cast<char *>(arg.c_str()));
    argv.push_back(nullptr);
    ::execvp(argv[0], argv.data());
    _exit(127);
  }
  // Establish the same group from the parent as well, before a timeout can
  // arrive. EACCES means the child already exec'd; ESRCH means it already
  // exited.
  if (::setpgid(child, child) != 0 && errno != EACCES && errno != ESRCH) {
    terminate_process_group(child, false);
    throw std::runtime_error("Cannot establish process group");
  }
  int status = 0;
  const auto deadline =
      std::chrono::steady_clock::now() + std::chrono::seconds(timeout_seconds);
  for (;;) {
    auto waited = ::waitpid(child, &status, WNOHANG);
    if (waited == child)
      break;
    if (waited < 0 && errno != EINTR) {
      terminate_process_group(child, false);
      throw std::runtime_error("waitpid failed");
    }
    if (std::chrono::steady_clock::now() >= deadline) {
      terminate_process_group(child, false);
      throw std::runtime_error("Process timed out: " + command[0]);
    }
    std::this_thread::sleep_for(std::chrono::milliseconds(2));
  }
  // A regular capture file has no EOF notification from inherited writers.
  // After reaping the direct child, any remaining group member is a leaked
  // descendant; it may still append output or mutate the acceptance fixture.
  if (::kill(-child, 0) == 0 || errno == EPERM) {
    terminate_process_group(child, true);
    throw std::runtime_error("Process left running descendants: " + command[0]);
  }
  return {WIFEXITED(status) ? WEXITSTATUS(status) : 128 + WTERMSIG(status),
          read(files.path / "stdout"), read(files.path / "stderr")};
}
struct Arguments {
  std::map<std::string, std::string> values;
  Arguments(int argc, char **argv) {
    for (int i = 1; i < argc; ++i) {
      std::string key = argv[i];
      native_test::require(key.starts_with("--"),
                           "Unexpected argument: " + key);
      key = key.substr(2);
      native_test::require(!values.contains(key), "Duplicate argument: " + key);
      if (i + 1 < argc && !std::string(argv[i + 1]).starts_with("--"))
        values[key] = argv[++i];
      else
        values[key] = "true";
    }
  }
  bool has(const std::string &key) const { return values.contains(key); }
  std::string get(const std::string &key,
                  const std::string &fallback = "") const {
    auto it = values.find(key);
    return it == values.end() ? fallback : it->second;
  }
  std::string require(const std::string &key) const {
    native_test::require(has(key) && get(key) != "true", "Missing --" + key);
    return get(key);
  }
};
inline std::string replace(std::string value, const std::string &old,
                           const std::string &replacement,
                           std::size_t maximum = std::string::npos) {
  std::size_t offset = 0, count = 0;
  while (count < maximum &&
         (offset = value.find(old, offset)) != std::string::npos) {
    value.replace(offset, old.size(), replacement);
    offset += replacement.size();
    ++count;
  }
  return value;
}
inline Json updated(Json value, const Json &fields) {
  value.update(fields);
  return value;
}
struct CheckedFile {
  std::string bytes;
  struct stat info;
};
inline CheckedFile receipt_read(const fs::path &root, const fs::path &relative,
                                std::size_t maximum) {
  require(root.is_absolute() && !relative.is_absolute(),
          "Receipt path must be prefix-relative");
  std::vector<std::string> parts;
  for (const auto &p : root.relative_path())
    parts.push_back(p.string());
  for (const auto &p : relative)
    parts.push_back(p.string());
  require(!parts.empty(), "Empty receipt path");
  int directory = ::open("/", O_RDONLY | O_DIRECTORY | O_NOFOLLOW);
  require(directory >= 0, "Cannot open filesystem root");
  for (std::size_t i = 0; i + 1 < parts.size(); ++i) {
    if (parts[i].empty() || parts[i] == "." || parts[i] == "..") {
      ::close(directory);
      throw std::runtime_error("Noncanonical receipt path");
    }
    int next = ::openat(directory, parts[i].c_str(),
                        O_RDONLY | O_DIRECTORY | O_NOFOLLOW);
    ::close(directory);
    require(next >= 0, "Cannot follow receipt directory: " + parts[i]);
    directory = next;
  }
  const auto &leaf = parts.back();
  if (leaf.empty() || leaf == "." || leaf == "..") {
    ::close(directory);
    throw std::runtime_error("Invalid receipt leaf");
  }
  int fd = ::openat(directory, leaf.c_str(), O_RDONLY | O_NOFOLLOW);
  ::close(directory);
  require(fd >= 0, "Cannot open receipt-owned file " + relative.string());
  CheckedFile result;
  bool good = ::fstat(fd, &result.info) == 0 && S_ISREG(result.info.st_mode) &&
              result.info.st_size >= 0 &&
              static_cast<std::uintmax_t>(result.info.st_size) <= maximum;
  if (!good) {
    ::close(fd);
    throw std::runtime_error("Receipt file kind/size invalid");
  }
  char block[65536];
  for (;;) {
    auto n = ::read(fd, block, sizeof(block));
    if (n == 0)
      break;
    if (n < 0) {
      if (errno == EINTR)
        continue;
      ::close(fd);
      throw std::runtime_error("Receipt read failed");
    }
    result.bytes.append(block, static_cast<std::size_t>(n));
    if (result.bytes.size() > maximum) {
      ::close(fd);
      throw std::runtime_error("Receipt file exceeds bound");
    }
  }
  ::close(fd);
  return result;
}
inline std::string installed_engine(const fs::path &prefix,
                                    const std::string &module,
                                    const std::string &engine,
                                    const std::string &version,
                                    const std::string &kind = "vector_engine",
                                    const std::string &vector = "shv") {
  auto root = fs::absolute(prefix);
  const auto receipt =
      Json::parse(receipt_read(root,
                               "share/symphony/receipts/" + module + "/" +
                                   version + "/install-receipt.json",
                               1048576)
                      .bytes);
  NT_REQUIRE(receipt == seal(receipt, "receipt_digest"));
  Json expected = {{"protocol", "symphony.knowledge.install-receipt.v2"},
                   {"format_version", 2},
                   {"component_id", module},
                   {"module_id", module},
                   {"package_id", module},
                   {"engine_id", engine},
                   {"vector_id", vector},
                   {"version", version},
                   {"component_kind", kind}};
  for (const auto &[key, value] : expected.items())
    NT_REQUIRE(receipt.at(key) == value);
  std::string relative =
      "libexec/symphony/" + module + "/" + version + "/" + engine;
  Json entries = Json::array(), owned = Json::array();
  for (const auto &e : receipt.at("entry_points"))
    if (e.at("entry_point_id") == engine)
      entries.push_back(e);
  NT_REQUIRE(entries.size() == 1);
  NT_REQUIRE(entries[0].at("path") == relative &&
             entries[0].at("kind") == "executable" &&
             entries[0].at("protocols") ==
                 Json::array({"symphony.knowledge.engine-process.v1"}));
  for (const auto &f : receipt.at("files"))
    if (f.at("path") == relative)
      owned.push_back(f);
  NT_REQUIRE(owned.size() == 1 && owned[0].at("kind") == "executable");
  auto binary = receipt_read(root, relative, 4194304);
  NT_REQUIRE((binary.info.st_mode & 0111) != 0 &&
             binary.bytes.size() == owned[0].at("size") &&
             digest(binary.bytes) == owned[0].at("digest").get<std::string>());
  return (root / relative).string();
}
inline Json request(const std::string &engine, const std::string &operation,
                    const Json &payload, const std::string &id = "test") {
  return {{"protocol", "symphony.knowledge.engine-process.v1"},
          {"request_id", id},
          {"correlation_id", id},
          {"target_engine", engine},
          {"operation", operation},
          {"deadline_unix_ms", now_ms() + 30000},
          {"payload", payload}};
}
struct Engine {
  std::string executable, id, request_id = "test";
  int calls = 0, rejections = 0;
  bool seal_results = true;
  Json evidence = Json::array();
  Json call(const std::string &operation, const Json &payload, bool good = true,
            const fs::path &cwd = {}) {
    ++calls;
    if (!good)
      ++rejections;
    auto result = run({executable},
                      canonical(request(id, operation, payload, request_id)),
                      40, {}, cwd);
    Json envelope = Json::parse(result.stdout_text);
    require(result.stderr_text.empty(), result.stderr_text);
    require((result.returncode == 0) == good,
            operation + ": " + result.stdout_text);
    NT_REQUIRE(envelope.at("outcome") == (good ? "ok" : "error"));
    NT_REQUIRE(envelope == seal(envelope, "response_digest"));
    Json value = envelope.value("result", Json(nullptr));
    if (good && seal_results)
      NT_REQUIRE(
          value ==
          seal(value, operation == "inspect" ? "descriptor_digest" : "digest"));
    evidence.push_back({{"operation", operation},
                        {"input", payload},
                        {"good", good},
                        {"result", value},
                        {"exit_code", result.returncode}});
    return value;
  }
  void summary() const {
    std::cout << Json({{"status", "passed"},
                       {"calls", calls},
                       {"rejections", rejections},
                       {"failures", 0},
                       {"skips", 0}})
                     .dump()
              << '\n';
  }
};
inline int test_main(const std::function<void()> &body) {
  try {
    body();
    return 0;
  } catch (const std::exception &e) {
    std::cerr << e.what() << '\n';
    return 1;
  }
}
} // namespace native_test
