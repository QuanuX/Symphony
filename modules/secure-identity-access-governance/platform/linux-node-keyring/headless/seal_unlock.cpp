#include "process.hpp"

#include <cstdio>
#include <cstring>
#include <sys/random.h>

using namespace node_keyring;
namespace {
class Ciphertext {
    std::string name_;
public:
    Descriptor file;
    explicit Ciphertext(const fs::path& dir)
        : name_((dir / ".node-keyring-XXXXXX").string()), file(mkostemp(name_.data(), O_CLOEXEC)) {}
    ~Ciphertext() { unlink(name_.c_str()); }
    void publish(const fs::path& final, const Descriptor& directory) {
        require(file.metadata().st_size > 0 && fsync(file.get()) == 0);
        // A competing enrollment cannot replace an already published seal.
        require(link(name_.c_str(), final.c_str()) == 0 && fsync(directory.get()) == 0);
    }
};

struct Unlock {
    std::array<unsigned char, 32> random{};
    std::array<char, 64> encoded{};
    ~Unlock() {
        explicit_bzero(random.data(), random.size());
        explicit_bzero(encoded.data(), encoded.size());
    }
    void send_to(int fd) {
        size_t used = 0;
        while (used < random.size()) {
            const auto n = getrandom(random.data() + used, random.size() - used, 0);
            if (n < 0 && errno == EINTR) continue;
            require(n > 0);
            used += static_cast<size_t>(n);
        }
        constexpr char hex[] = "0123456789abcdef";
        for (size_t i = 0; i < random.size(); ++i) {
            encoded[2 * i] = hex[random[i] >> 4];
            encoded[2 * i + 1] = hex[random[i] & 15];
        }
        write_all(fd, std::string_view(encoded.data(), encoded.size()));
    }
};

void seal(uid_t uid, std::string_view key) {
    require(getuid() == 0 && geteuid() == 0 && (key == "host" || key == "tpm2"));
    protect_process();
    const auto home = account_home(uid);
    const auto vaults = home / ".local/share/keyrings";
    if (fs::symlink_status(vaults).type() != fs::file_type::not_found) {
        require(fs::canonical(vaults) == vaults && fs::is_directory(vaults) && fs::is_empty(vaults));
    }
    const fs::path dir("/etc/credstore.encrypted");
    require(mkdir(dir.c_str(), 0700) == 0 || errno == EEXIST);
    require(fs::canonical(dir) == dir);
    Descriptor directory(open(dir.c_str(), O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC));
    const auto st = directory.metadata();
    require(st.st_uid == 0 && (st.st_mode & 0022) == 0);
    const auto final = dir / ("symphony-node-keyring-" + std::to_string(uid) + ".cred");
    require(fs::symlink_status(final).type() == fs::file_type::not_found);
    std::vector<std::string> args{"/usr/bin/systemd-creds", "encrypt", "--with-key=" + std::string(key),
                                  "--name=vault-unlock"};
    if (key == "tpm2") {
        args.emplace_back("--tpm2-device=auto");
        args.emplace_back("--tpm2-pcrs=7");
    }
    args.insert(args.end(), {"-", "-"});
    Ciphertext ciphertext(dir);
    Pipe source;
    Child encrypt(std::move(args), {"PATH=/usr/bin:/bin"}, source.read_end(), ciphertext.file.get());
    {
        Unlock unlock;
        unlock.send_to(source.write_end());
    }
    source.finish_writing();
    require(encrypt.wait_until(Clock::now() + 40s));
    ciphertext.publish(final, directory);
}
}

int main(int argc, char** argv) {
    try {
        require(argc == 3);
        require(std::signal(SIGPIPE, SIG_IGN) != SIG_ERR);
        seal(parse_uid(argv[1]), argv[2]);
        std::puts("Sealed local vault unlock credential");
        return 0;
    } catch (...) {
        std::fputs("Vault seal refused or unavailable\n", stderr);
        return 1;
    }
}
