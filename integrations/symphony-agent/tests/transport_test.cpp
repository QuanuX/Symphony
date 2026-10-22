#include "symphony/transport.hpp"

#include <cerrno>
#include <chrono>
#include <csignal>
#include <cstdlib>
#include <cstring>
#include <fcntl.h>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <string>
#include <sys/stat.h>
#include <sys/wait.h>
#include <unistd.h>

namespace fs = std::filesystem;

namespace {
int checks = 0;
volatile sig_atomic_t terminated = 0;
void term_signal(int) { terminated = 1; }
void check(bool condition, const char* message) {
    ++checks;
    if (!condition) throw std::runtime_error(message);
}
template <typename Action> void rejects(Action action, const char* message) {
    bool rejected = false;
    try { action(); } catch (const std::exception&) { rejected = true; }
    check(rejected, message);
}

int child(int argc, char** argv) {
    const std::string mode = argv[1];
    if (mode == "--literal") {
        std::cout << argv[2];
        return 0;
    }
    if (mode == "--environment") {
        const char* allowed = std::getenv("SYMPHONY_TEST_ALLOWED");
        const char* inherited = std::getenv("SYMPHONY_TEST_AMBIENT");
        std::cout << (allowed ? allowed : "missing") << ':' << (inherited ? inherited : "absent");
        return 0;
    }
    if (mode == "--stdin") {
        char character;
        return ::read(STDIN_FILENO, &character, 1) == 0 ? 0 : 1;
    }
    if (mode == "--cwd") {
        std::cout << fs::current_path().string();
        return 0;
    }
    if (mode == "--exit") return 23;
    if (mode == "--sleep") { ::sleep(10); return 0; }
    if (mode == "--graceful") {
        ::signal(SIGTERM, term_signal);
        while (!terminated) ::pause();
        std::ofstream(argv[2]) << "owner cleanup completed";
        return 0;
    }
    if (mode == "--ignore-term") {
        ::signal(SIGTERM, SIG_IGN);
        ::sleep(10);
        return 0;
    }
    if (mode == "--flood") {
        const std::string block(8192, 'x');
        for (;;) {
            if (::write(STDOUT_FILENO, block.data(), block.size()) < 0) return 2;
            if (::write(STDERR_FILENO, block.data(), block.size()) < 0) return 3;
        }
    }
    if (mode == "--descendant") {
        const pid_t nested = ::fork();
        if (nested < 0) return 1;
        if (nested == 0) {
            ::usleep(700000);
            std::ofstream(argv[2]) << "escaped cleanup";
            ::_exit(0);
        }
        // Leave a descendant holding stdout/stderr after this direct child exits.
        return 0;
    }
    if (mode == "--closed-descendant") {
        const pid_t nested = ::fork();
        if (nested < 0) return 1;
        if (nested == 0) {
            ::close(0); ::close(1); ::close(2);
            ::usleep(700000);
            std::ofstream(argv[2]) << "escaped cleanup";
            ::_exit(0);
        }
        return 0;
    }
    if (mode == "--fd") {
        const int fd = std::stoi(argv[2]);
        struct stat status{};
        if (::fstat(fd, &status) < 0 && errno == EBADF) return 0;
        // Instrumented runtimes may open a different file at the recycled fd.
        return std::to_string(status.st_dev) == argv[3] &&
               std::to_string(status.st_ino) == argv[4] ? 1 : 0;
    }
    return argc < 0 ? 1 : 99;
}
} // namespace

int main(int argc, char** argv) {
    if (argc > 1) return child(argc, argv);
    char temporary[] = "/private/tmp/symphony-transport-XXXXXX";
#if defined(__linux__)
    std::strcpy(temporary, "/tmp/symphony-transport-XXXXXX");
#endif
    char* root = ::mkdtemp(temporary);
    if (!root) { std::cerr << "mkdtemp failed\n"; return 1; }
    const std::string directory = fs::canonical(root).string();
    const std::string executable = fs::canonical(argv[0]).string();
    const std::string socket_path = directory + "/bridge.sock";
    try {
        auto run = [&](std::vector<std::string> args, int timeout = 3000, std::size_t limit = 65536) {
            return symphony::run_process(executable, args, directory, {}, timeout, limit);
        };
        const std::string literal = "$(touch should-not-exist); `echo unsafe` * spaces \"'";
        auto result = run({"--literal", literal});
        check(result.exit_code == 0 && result.out == literal, "argv must be literal");
        check(!fs::exists(directory + "/should-not-exist"), "shell syntax must not execute");
        ::setenv("SYMPHONY_TEST_AMBIENT", "must-not-inherit", 1);
        result = symphony::run_process(executable, {"--environment"}, directory,
                                       {{"SYMPHONY_TEST_ALLOWED", "exact=value"}}, 3000, 65536);
        check(result.out == "exact=value:absent", "environment must be explicitly selected");
        check(run({"--stdin"}).exit_code == 0, "child stdin must be EOF");
        check(run({"--cwd"}).out == directory, "child cwd must be selected");
        check(run({"--exit"}).exit_code == 23, "child exit code must be preserved");
        bool missing_failed = false;
        try { missing_failed = symphony::run_process(directory + "/missing", {}, directory, {}, 3000, 64).exit_code != 0; }
        catch (const std::exception&) { missing_failed = true; }
        check(missing_failed, "missing executable must fail without retry");

        int inherited = ::open((directory + "/inheritance-sentinel").c_str(), O_RDWR | O_CREAT | O_EXCL, 0600);
        check(inherited >= 3, "fixture descriptor open");
        struct stat inherited_stat{};
        check(::fstat(inherited, &inherited_stat) == 0, "fixture descriptor identity");
        check(run({"--fd", std::to_string(inherited), std::to_string(inherited_stat.st_dev),
                   std::to_string(inherited_stat.st_ino)}).exit_code == 0, "unrelated descriptors must not cross exec");
        ::close(inherited);
        const auto start = std::chrono::steady_clock::now();
        result = run({"--sleep"}, 150);
        check(result.timed_out && result.exit_code == 128 + SIGTERM, "timeout must request process termination");
        check(std::chrono::steady_clock::now() - start < std::chrono::seconds(3), "timeout must be bounded");
        const std::string grace_marker = directory + "/grace-marker";
        result = run({"--graceful", grace_marker}, 500);
        check(result.timed_out && result.exit_code == 0 && fs::exists(grace_marker), "SIGTERM grace must permit owner cleanup");
        const auto force_start = std::chrono::steady_clock::now();
        result = run({"--ignore-term"}, 500);
        check(result.timed_out && result.exit_code == 128 + SIGKILL, "uncooperative child must be forcibly terminated");
        check(std::chrono::steady_clock::now() - force_start < std::chrono::seconds(2), "SIGTERM grace must be bounded");
        result = run({"--flood"}, 2000, 4096);
        check(result.output_limited, "large output must report its limit");
        check(result.out.size() + result.err.size() == 4096, "stdout and stderr share one output bound");
        const std::string marker = directory + "/descendant-marker";
        result = run({"--descendant", marker}, 100);
        check(result.timed_out, "descendant-held output must honor the deadline");
        ::usleep(800000);
        check(!fs::exists(marker), "descendant must be killed even after the direct child exits");
        const std::string closed_marker = directory + "/closed-descendant-marker";
        result = run({"--closed-descendant", closed_marker});
        check(result.exit_code == 0 && !result.timed_out, "normally closed output must complete");
        ::usleep(800000);
        check(!fs::exists(closed_marker), "background descendants must be cleaned after normal completion");

        rejects([&] { symphony::run_process("relative", {}, directory, {}, 10, 1); }, "relative executable must reject");
        rejects([&] { symphony::run_process(executable, {}, ".", {}, 10, 1); }, "relative cwd must reject");
        rejects([&] { run({std::string("nul\0arg", 7)}); }, "NUL arguments must reject");
        rejects([&] { symphony::run_process(executable, {}, directory, {{"BAD=KEY", "v"}}, 10, 1); }, "invalid env must reject");
        rejects([&] { run({}, 0); }, "zero timeout must reject");
        rejects([&] { run({}, 10, 0); }, "zero output limit must reject");

        const auto token = symphony::random_token();
        check(token.size() == 64 && token.find_first_not_of("0123456789abcdef") == std::string::npos,
              "tokens must be 256-bit lowercase hex");
        check(token != symphony::random_token(), "independent token calls must differ");

        int listener = symphony::listen_local_socket(socket_path);
        struct stat st{};
        check(::lstat(socket_path.c_str(), &st) == 0 && S_ISSOCK(st.st_mode) && (st.st_mode & 0777) == 0600,
              "listener socket must have mode 0600");
        check((::fcntl(listener, F_GETFD) & FD_CLOEXEC) != 0, "listener must be close-on-exec");
        rejects([&] { symphony::listen_local_socket(socket_path); }, "existing socket must not be replaced");
        int client = symphony::connect_local_socket(socket_path);
        int server = symphony::accept_local_socket(listener);
        check((::fcntl(client, F_GETFD) & FD_CLOEXEC) != 0 && (::fcntl(server, F_GETFD) & FD_CLOEXEC) != 0,
              "both connected descriptors must be close-on-exec");
        check(::write(client, "hello", 5) == 5, "socket client send");
        char received[5]{};
        check(::read(server, received, sizeof(received)) == 5 && std::string(received, 5) == "hello", "socket transport roundtrip");
        ::close(server); ::close(client); ::close(listener);
        symphony::remove_local_socket(socket_path);
        check(!fs::exists(socket_path), "owned socket should be removed");

        listener = symphony::listen_local_socket(socket_path);
        ::close(listener);
        ::unlink(socket_path.c_str());
        std::ofstream(socket_path) << "replacement";
        symphony::remove_local_socket(socket_path);
        check(fs::is_regular_file(socket_path), "cleanup must preserve replacement files");
        rejects([&] { symphony::listen_local_socket(socket_path); }, "existing regular file must not be replaced");
        rejects([&] { symphony::connect_local_socket(socket_path); }, "regular file must not be a socket endpoint");
        fs::remove(socket_path);
        ::symlink("missing", socket_path.c_str());
        rejects([&] { symphony::listen_local_socket(socket_path); }, "dangling symlink must not be replaced");
        fs::remove(socket_path);
        ::chmod(directory.c_str(), 0755);
        rejects([&] { symphony::listen_local_socket(socket_path); }, "public parent directory must reject");
        ::chmod(directory.c_str(), 0700);
        const std::string directory_link = directory + "/linked";
        ::symlink(directory.c_str(), directory_link.c_str());
        rejects([&] { symphony::listen_local_socket(directory_link + "/socket"); }, "symlink parent must reject");
        rejects([&] { symphony::listen_local_socket("relative.sock"); }, "relative socket path must reject");
        rejects([&] { symphony::listen_local_socket(directory + "/" + std::string(200, 'x')); }, "overlong socket path must reject");
        fs::remove_all(directory);
        std::cout << checks << " transport checks passed\n";
        return 0;
    } catch (const std::exception& error) {
        symphony::remove_local_socket(socket_path);
        fs::remove_all(directory);
        std::cerr << "transport test failure: " << error.what() << '\n';
        return 1;
    }
}
