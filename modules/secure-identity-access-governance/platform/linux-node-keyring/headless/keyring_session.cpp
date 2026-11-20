#include "process.hpp"

#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <endian.h>
#include <linux/posix_acl_xattr.h>
#include <sys/file.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/xattr.h>

using namespace node_keyring;
namespace {
volatile sig_atomic_t stopping{};
void stop(int) { stopping = 1; }

void verify_identity(uid_t uid) {
    require(getuid() == uid && geteuid() == uid && getgid() == getegid());
    const int count = getgroups(0, nullptr);
    require(count >= 0);
    std::vector<gid_t> groups(static_cast<size_t>(count));
    require(getgroups(count, groups.data()) == count);
    for (const auto group : groups) require(group == getgid());
}

void private_dir(const fs::path& path, uid_t uid, mode_t forbidden = 0077) {
    struct stat st{};
    require(lstat(path.c_str(), &st) == 0 && S_ISDIR(st.st_mode) && st.st_uid == uid &&
            (st.st_mode & forbidden) == 0);
}

void verify_vault(const fs::path& path, uid_t uid) {
    Descriptor file(open(path.c_str(), O_RDONLY | O_NOFOLLOW | O_NONBLOCK | O_CLOEXEC));
    auto st = file.metadata();
    require(S_ISREG(st.st_mode) && st.st_uid == uid && st.st_nlink == 1 && (st.st_mode & 0077) == 0);
    // Empty-password GNOME keyrings may be serialized as plaintext keyfiles.
    constexpr char magic[] = "GnomeKeyring\n\r\0\n";
    std::array<char, sizeof(magic) - 1> bytes{};
    require(pread(file.get(), bytes.data(), bytes.size(), 0) == static_cast<ssize_t>(bytes.size()) &&
            std::memcmp(bytes.data(), magic, bytes.size()) == 0);
}

fs::path configure_vault(const fs::path& home, uid_t uid, bool enroll) {
    private_dir(home, uid);
    auto dir = home;
    for (const auto* part : {".local", "share", "keyrings"}) {
        dir /= part;
        if (enroll) require(mkdir(dir.c_str(), 0700) == 0 || errno == EEXIST);
        private_dir(dir, uid, 0022);
    }
    private_dir(dir, uid);
    const auto login = dir / "login.keyring";
    if (enroll) require(fs::is_empty(dir));
    else verify_vault(login, uid);
    return login;
}

bool private_credential(int fd, const struct stat& s, uid_t uid) {
    if ((s.st_mode & 07777) == 0400) return s.st_uid == 0 || s.st_uid == uid;
    if ((s.st_mode & 07777) != 0440 || s.st_uid != 0) return false;
    // systemd commonly grants the service UID a named read ACL; the group
    // permission bits represent its mask. Refuse extra named users/groups.
    struct Acl { posix_acl_xattr_header header; posix_acl_xattr_entry entries[5]; };
    static_assert(sizeof(Acl) == 44);
    Acl acl{};
    if (fgetxattr(fd, "system.posix_acl_access", &acl, sizeof(acl)) != sizeof(acl) ||
        le32toh(acl.header.a_version) != POSIX_ACL_XATTR_VERSION) return false;
    constexpr uint32_t no_id = 0xffffffff;
    // ACL_USER_OBJ, ACL_USER, ACL_GROUP_OBJ, ACL_MASK, ACL_OTHER
    constexpr uint16_t tags[] = {1, 2, 4, 16, 32}, perms[] = {4, 4, 0, 4, 0};
    for (size_t i = 0; i < 5; ++i) {
        const auto& e = acl.entries[i];
        if (le16toh(e.e_tag) != tags[i] || le16toh(e.e_perm) != perms[i] ||
            le32toh(e.e_id) != (i == 1 ? uid : no_id)) return false;
    }
    return true;
}

Descriptor open_unlock(std::string_view mode) {
    const uid_t uid = geteuid();
    const std::string instance = std::to_string(uid) + ".service";
    const std::string enroll_unit = "symphony-node-keyring-enroll@" + instance;
    const std::string serve_unit = "symphony-node-keyring@" + instance;
    const char* location = getenv("CREDENTIALS_DIRECTORY");
    require(location != nullptr);
    const bool enroll = mode == "enroll" ||
                        (mode == "check-unlock" && location == "/run/credentials/" + enroll_unit);
    const auto& unit = enroll ? enroll_unit : serve_unit;
    // The manager's environment confirms the unit. Only the verified service
    // UID and the installed unit names select a filesystem entry.
    require(location == "/run/credentials/" + unit);
    constexpr int directory_flags = O_PATH | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC;
    Descriptor runtime(open("/run", directory_flags));
    auto root = runtime.metadata();
    require(root.st_uid == 0 && (root.st_mode & 0022) == 0);
    Descriptor credentials(openat(runtime.get(), "credentials", directory_flags));
    root = credentials.metadata();
    require(root.st_uid == 0 && (root.st_mode & 0022) == 0);
    Descriptor dir(openat(credentials.get(), unit.c_str(), directory_flags));
    const auto owner = dir.metadata();
    require((owner.st_uid == 0 || owner.st_uid == uid) && (owner.st_mode & 0022) == 0);
    Descriptor file(openat(dir.get(), "vault-unlock", O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC));
    const auto st = file.metadata();
    require(S_ISREG(st.st_mode) && st.st_nlink == 1 && st.st_size == 64 && private_credential(file.get(), st, uid));
    return file;
}

class SessionDirectory {
public:
    fs::path path;
    explicit SessionDirectory(const fs::path& runtime) {
        std::string name = (runtime / ".session-XXXXXX").string();
        require(mkdtemp(name.data()) != nullptr);
        path = name;
    }
    ~SessionDirectory() { std::error_code ec; fs::remove_all(path, ec); }
};

bool unlocked(const std::vector<std::string>& env) {
    Pipe out;
    Child query({"/usr/bin/gdbus", "call", "--session", "--timeout", "1", "--dest", "org.freedesktop.secrets",
                 "--object-path", "/org/freedesktop/secrets/collection/login", "--method",
                 "org.freedesktop.DBus.Properties.Get", "org.freedesktop.Secret.Collection", "Locked"},
                env, -1, out.write_end());
    out.finish_writing();
    if (!query.wait_until(Clock::now() + 2s)) return false;
    std::array<char, 32> reply{};
    const auto size = read(out.read_end(), reply.data(), reply.size());
    return size > 0 && std::string_view(reply.data(), static_cast<size_t>(size)) == "(<false>,)\n";
}

void notify_ready() {
    const char* dest = getenv("NOTIFY_SOCKET");
    if (!dest || !*dest) return;
    const size_t n = std::strlen(dest);
    sockaddr_un sa{};
    require(n < sizeof(sa.sun_path) && (dest[0] == '@' || dest[0] == '/'));
    sa.sun_family = AF_UNIX;
    std::memcpy(sa.sun_path, dest, n);
    socklen_t len = static_cast<socklen_t>(offsetof(sockaddr_un, sun_path) + n);
    if (dest[0] == '@') sa.sun_path[0] = 0;
    else ++len;
    Descriptor socket_fd(socket(AF_UNIX, SOCK_DGRAM | SOCK_CLOEXEC, 0));
    require(sendto(socket_fd.get(), "READY=1", 7, MSG_NOSIGNAL,
                   reinterpret_cast<const sockaddr*>(&sa), len) == 7);
}

void run(std::string_view mode, uid_t uid) {
    verify_identity(uid);
    protect_process();
    require(mode == "enroll" || mode == "serve" || mode == "check-unlock");
    auto unlock = open_unlock(mode);
    if (mode == "check-unlock") return;
    const auto runtime = fs::path("/run/symphony-node-keyring-" + std::to_string(uid));
    require(fs::canonical(runtime) == runtime);
    private_dir(runtime, uid);
    Descriptor lock(open((runtime / "session.lock").c_str(),
                         O_RDONLY | O_NONBLOCK | O_NOFOLLOW | O_CLOEXEC | O_CREAT, 0600));
    const auto lock_info = lock.metadata();
    require(S_ISREG(lock_info.st_mode) && lock_info.st_uid == uid && lock_info.st_nlink == 1 &&
            (lock_info.st_mode & 0077) == 0 && flock(lock.get(), LOCK_EX | LOCK_NB) == 0);
    const auto bus = runtime / "bus";
    require(fs::symlink_status(bus).type() == fs::file_type::not_found);
    const auto home = account_home(uid);
    const auto vault = configure_vault(home, uid, mode == "enroll");
    std::string address = "unix:path=" + bus.string();
    const std::vector<std::string> env{"HOME=" + home.string(), "PATH=/usr/bin:/bin",
          "XDG_DATA_HOME=" + (home / ".local/share").string(), "XDG_RUNTIME_DIR=" + runtime.string(),
          "DBUS_SESSION_BUS_ADDRESS=" + address};
    SessionDirectory temp(runtime);
    const auto config = temp.path / "bus.conf";
    {
        Descriptor file(open(config.c_str(), O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW | O_CLOEXEC, 0600));
        write_all(file.get(), "<busconfig><type>session</type><listen>" + address +
                 "</listen><auth>EXTERNAL</auth><policy context=\"default\"><allow send_destination=\"*\"/>"
                 "<allow receive_sender=\"*\"/><allow own=\"*\"/></policy></busconfig>");
    }
    Child daemon({"/usr/bin/dbus-daemon", "--nofork", "--config-file=" + config.string()}, env, -1, -1);
    auto cutoff = Clock::now() + 8s;
    while (!fs::is_socket(bus)) {
        require(!stopping && daemon.running() && Clock::now() < cutoff);
        std::this_thread::sleep_for(50ms);
    }
    Child keyring({"/usr/bin/gnome-keyring-daemon", "--foreground", "--unlock", "--components=secrets",
                   "--control-directory=" + temp.path.string()}, env, unlock.get(), -1);
    unlock.reset();
    while (!unlocked(env)) {
        require(!stopping && daemon.running() && keyring.running() && Clock::now() < cutoff);
        std::this_thread::sleep_for(80ms);
    }
    verify_vault(vault, uid);
    if (mode == "serve" && !stopping) {
        notify_ready();
        while (!stopping) {
            require(daemon.running() && keyring.running());
            std::this_thread::sleep_for(250ms);
        }
    }
}
}

int main(int argc, char** argv) {
    try {
        require(argc == 3);
        struct sigaction action{};
        action.sa_handler = stop;
        sigemptyset(&action.sa_mask);
        require(sigaction(SIGTERM, &action, nullptr) == 0);
        run(argv[1], parse_uid(argv[2]));
        return 0;
    } catch (...) {
        if (stopping) return 0;
        std::fputs("Keyring session refused or unavailable\n", stderr);
        return 1;
    }
}
