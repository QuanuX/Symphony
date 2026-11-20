#include "volume_policy.hpp"

#include <algorithm>
#include <cerrno>
#include <csignal>
#include <cstdlib>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <pwd.h>
#include <sys/stat.h>
#include <sys/sysmacros.h>
#include <sys/wait.h>
#include <unistd.h>

namespace fs = std::filesystem;
using volume::require;
namespace {
struct Fd {
    int value;
    ~Fd() { if (value >= 0) close(value); }
    explicit Fd(int fd) : value(fd) { require(fd >= 0, "local evidence unavailable"); }
    Fd(const Fd&) = delete;
};
std::string read_all(int fd, std::size_t limit) {
    std::string result;
    char block[4096];
    while (true) {
        const auto n = read(fd, block, sizeof block);
        if (n < 0 && errno == EINTR) continue;
        require(n >= 0, "read failed");
        if (!n) return result;
        require(result.size() + static_cast<std::size_t>(n) <= limit, "oversized input");
        result.append(block, static_cast<std::size_t>(n));
    }
}
struct stat path_stat(const fs::path& path, bool follow = false) {
    struct stat info{};
    require((follow ? stat(path.c_str(), &info) : lstat(path.c_str(), &info)) == 0, "local evidence unavailable");
    return info;
}
std::string read_line(const fs::path& path) {
    Fd file(open(path.c_str(), O_RDONLY | O_CLOEXEC));
    auto value = read_all(file.value, 512);
    if (value.ends_with('\n')) value.pop_back();
    return value;
}
std::string selected_uuid() {
    const fs::path config("/etc/symphony/node-keyring/luks-uuid");
    for (auto dir = config.parent_path(); ; dir = dir.parent_path()) {
        auto info = path_stat(dir);
        require(S_ISDIR(info.st_mode) && info.st_uid == 0 && !(info.st_mode & 0022), "untrusted configuration directory");
        if (dir == dir.root_path()) break;
    }
    Fd fd(open(config.c_str(), O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
    struct stat st{};
    require(fstat(fd.value, &st) == 0 && S_ISREG(st.st_mode) && st.st_uid == 0 && !(st.st_mode & 0022),
            "untrusted volume selection");
    auto uuid = read_all(fd.value, 37);
    if (uuid.ends_with('\n')) uuid.pop_back();
    require(uuid.size() == 36, "invalid UUID");
    for (std::size_t i = 0; i < uuid.size(); ++i) {
        const char c = uuid[i];
        const bool dash = i == 8 || i == 13 || i == 18 || i == 23;
        require(dash ? c == '-' : ((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')), "invalid UUID");
    }
    return uuid;
}
std::string dump_header(const fs::path& path) {
    int pipes[2];
    require(pipe2(pipes, O_CLOEXEC) == 0, "cannot inspect volume");
    Fd input(pipes[0]), output(pipes[1]);
    const pid_t child = fork();
    require(child >= 0, "cannot inspect volume");
    if (child == 0) {
        const int null = open("/dev/null", O_RDWR);
        if (null < 0 || dup2(null, STDIN_FILENO) < 0 || dup2(null, STDERR_FILENO) < 0 ||
            dup2(output.value, STDOUT_FILENO) < 0 || clearenv() || setenv("LC_ALL", "C", 1) ||
            setenv("PATH", "/usr/sbin:/usr/bin", 1)) _exit(126);
        execl("/usr/bin/timeout", "timeout", "-k", "1", "8", "/usr/sbin/cryptsetup", "luksDump",
              "--dump-json-metadata", path.c_str(), static_cast<char*>(nullptr));
        _exit(127);
    }
    close(output.value);
    output.value = -1;
    std::string header;
    try { header = read_all(input.value, 4U << 20); }
    catch (...) {
        close(input.value); input.value = -1;
        while (waitpid(child, nullptr, 0) < 0 && errno == EINTR) {}
        throw;
    }
    int code = 0;
    pid_t result;
    do { result = waitpid(child, &code, 0); } while (result < 0 && errno == EINTR);
    require(result == child && WIFEXITED(code) && WEXITSTATUS(code) == 0, "cannot inspect volume");
    return header;
}
void check(unsigned uid) {
    const fs::path home(volume::home);
    require(geteuid() == 0 && uid != 0 && uid < std::numeric_limits<uid_t>::max(), "dedicated UID required");
    const auto* account = getpwuid(uid);
    require(account && home == account->pw_dir, "unexpected account home");
    const std::string uuid = selected_uuid();
    std::ifstream mounts("/proc/self/mountinfo");
    const std::string device = volume::mounted_device(mounts);
    auto mapped = fs::path("/sys/dev/block") / device;
    const std::string mapper = "symphonynodevault";
    std::string compact = uuid;
    std::erase(compact, '-');
    require(read_line(mapped / "dm/uuid") == "CRYPT-LUKS2-" + compact + "-" + mapper &&
            read_line(mapped / "dm/name") == mapper, "wrong mapped volume");
    fs::directory_iterator slave(mapped / "slaves"), end;
    require(slave != end, "missing backing device");
    const auto raw = read_line(slave->path() / "dev");
    require(++slave == end, "unexpected mapper topology");
    volume::check_device(raw);
    const auto raw_path = fs::path("/dev/block") / raw;
    const auto block = path_stat(raw_path, true), by_uuid = path_stat(fs::path("/dev/disk/by-uuid") / uuid, true);
    require(S_ISBLK(block.st_mode) && S_ISBLK(by_uuid.st_mode) && block.st_rdev == by_uuid.st_rdev, "wrong backing volume");
    const auto colon = device.find(':');
    const auto mounted = makedev(volume::decimal(device.substr(0, colon)), volume::decimal(device.substr(colon + 1)));
    for (const auto& dir : {home, home / ".local", home / ".local/share", home / ".local/share/keyrings"}) {
        const auto s = path_stat(dir);
        const auto mask = dir == home || dir.filename() == "keyrings" ? 0077 : 0022;
        require(S_ISDIR(s.st_mode) && s.st_uid == uid && s.st_dev == mounted && !(s.st_mode & mask), "unexpected keyring directory");
    }
    volume::check_slots(dump_header(raw_path));
}
}
int main(int argc, char** argv) {
    try {
        require(argc == 2 && argv[1][0] != '0', "dedicated UID required");
        check(volume::decimal(argv[1]));
        std::cout << "Hardware volume preflight passed\n";
        return 0;
    } catch (...) {
        // Exception text can contain filesystem paths or metadata. Only emit the outcome.
        std::cerr << "Hardware volume preflight refused\n";
        return 1;
    }
}
