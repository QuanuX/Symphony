#include "local_git.hpp"

#include "provider.hpp"
#include "symphony/knowledge/engine/digest.hpp"
#include "symphony/knowledge/engine/error.hpp"
#include "symphony/knowledge/engine/limits.hpp"

#include <algorithm>
#include <array>
#include <cerrno>
#include <chrono>
#include <csignal>
#include <fcntl.h>
#include <poll.h>
#include <string>
#include <sys/wait.h>
#include <unistd.h>
#include <vector>

namespace symphony::knowledge::sclv::provider {
namespace {

std::int64_t now_ms() {
    const auto now = std::chrono::system_clock::now().time_since_epoch();
    return std::chrono::duration_cast<std::chrono::milliseconds>(now).count();
}

bool lower_hex(const std::string& value, std::size_t length) {
    return value.size() == length && std::all_of(value.begin(), value.end(), [](const unsigned char character) {
        return (character >= '0' && character <= '9') || (character >= 'a' && character <= 'f');
    });
}

std::string required_string(const engine::Json& object, const char* field, std::size_t maximum) {
    const auto& value = object.at(field);
    if (!value.is_string() || !bounded_text(value.get_ref<const std::string&>(), maximum)) {
        throw engine::Error("local_git.invalid_field", std::string(field) + " is invalid", 4);
    }
    return value.get<std::string>();
}

std::string run_git(
    const std::vector<std::string>& arguments,
    std::size_t maximum_output,
    std::int64_t deadline_unix_ms) {
    int descriptors[2];
    if (::pipe(descriptors) != 0) {
        throw engine::Error("local_git.pipe_failed", "could not create bounded Git output pipe", 5);
    }
    const auto pid = ::fork();
    if (pid < 0) {
        ::close(descriptors[0]);
        ::close(descriptors[1]);
        throw engine::Error("local_git.fork_failed", "could not start local Git", 5);
    }
    if (pid == 0) {
        ::close(descriptors[0]);
        if (::dup2(descriptors[1], STDOUT_FILENO) < 0) _exit(126);
        const int null_fd = ::open("/dev/null", O_WRONLY | O_CLOEXEC);
        if (null_fd < 0 || ::dup2(null_fd, STDERR_FILENO) < 0) _exit(126);
        ::close(descriptors[1]);
        if (null_fd != STDERR_FILENO) ::close(null_fd);

        std::vector<char*> argv;
        argv.reserve(arguments.size() + 2U);
        argv.push_back(const_cast<char*>("/usr/bin/git"));
        for (const auto& argument : arguments) argv.push_back(const_cast<char*>(argument.c_str()));
        argv.push_back(nullptr);
        std::array<char*, 7> environment = {
            const_cast<char*>("PATH=/usr/bin:/bin"),
            const_cast<char*>("LC_ALL=C"),
            const_cast<char*>("GIT_CONFIG_NOSYSTEM=1"),
            const_cast<char*>("GIT_CONFIG_GLOBAL=/dev/null"),
            const_cast<char*>("GIT_OPTIONAL_LOCKS=0"),
            const_cast<char*>("GIT_NO_REPLACE_OBJECTS=1"),
            nullptr,
        };
        ::execve("/usr/bin/git", argv.data(), environment.data());
        _exit(127);
    }

    ::close(descriptors[1]);
    const int flags = ::fcntl(descriptors[0], F_GETFL, 0);
    if (flags < 0 || ::fcntl(descriptors[0], F_SETFL, flags | O_NONBLOCK) != 0) {
        ::kill(pid, SIGKILL);
        ::close(descriptors[0]);
        static_cast<void>(::waitpid(pid, nullptr, 0));
        throw engine::Error("local_git.pipe_failed", "could not bound local Git output", 5);
    }

    std::string output;
    std::array<char, 16384> buffer{};
    bool eof = false;
    while (!eof) {
        if (now_ms() >= deadline_unix_ms) {
            ::kill(pid, SIGKILL);
            ::close(descriptors[0]);
            static_cast<void>(::waitpid(pid, nullptr, 0));
            throw engine::Error("request.deadline_expired", "local Git exceeded request deadline", 3);
        }
        pollfd descriptor{descriptors[0], POLLIN | POLLHUP, 0};
        const auto wait = ::poll(&descriptor, 1, 50);
        if (wait < 0 && errno != EINTR) {
            ::kill(pid, SIGKILL);
            ::close(descriptors[0]);
            static_cast<void>(::waitpid(pid, nullptr, 0));
            throw engine::Error("local_git.read_failed", "could not poll local Git output", 5);
        }
        if (wait <= 0) continue;
        for (;;) {
            const auto count = ::read(descriptors[0], buffer.data(), buffer.size());
            if (count > 0) {
                if (output.size() + static_cast<std::size_t>(count) > maximum_output) {
                    ::kill(pid, SIGKILL);
                    ::close(descriptors[0]);
                    static_cast<void>(::waitpid(pid, nullptr, 0));
                    throw engine::Error("local_git.output_limit", "local Git output exceeds the bounded limit", 5);
                }
                output.append(buffer.data(), static_cast<std::size_t>(count));
                continue;
            }
            if (count == 0) eof = true;
            if (count < 0 && errno != EAGAIN && errno != EWOULDBLOCK && errno != EINTR) {
                ::kill(pid, SIGKILL);
                ::close(descriptors[0]);
                static_cast<void>(::waitpid(pid, nullptr, 0));
                throw engine::Error("local_git.read_failed", "could not read local Git output", 5);
            }
            break;
        }
    }
    ::close(descriptors[0]);
    int status = 0;
    for (;;) {
        const auto waited = ::waitpid(pid, &status, WNOHANG);
        if (waited == pid) break;
        if (waited < 0) {
            throw engine::Error("local_git.command_failed", "could not collect local Git status", 5);
        }
        if (now_ms() >= deadline_unix_ms) {
            ::kill(pid, SIGKILL);
            static_cast<void>(::waitpid(pid, nullptr, 0));
            throw engine::Error("request.deadline_expired", "local Git exceeded request deadline", 3);
        }
        static_cast<void>(::poll(nullptr, 0, 10));
    }
    if (!WIFEXITED(status) || WEXITSTATUS(status) != 0) {
        throw engine::Error("local_git.command_failed", "fixed local Git evidence command failed", 4);
    }
    return output;
}

}

engine::Json normalize_local_git(const engine::Json& payload, std::int64_t deadline_unix_ms) {
    require_exact_fields(payload, {"observed_at", "source_reference", "revision_scheme", "revision_value"});
    const auto observed_at = required_string(payload, "observed_at", 20U);
    const auto source_reference = required_string(payload, "source_reference", 4096U);
    const auto scheme = required_string(payload, "revision_scheme", 128U);
    const auto revision = required_string(payload, "revision_value", 64U);
    if (!strict_utc(observed_at) ||
        (scheme != "git-sha1" && scheme != "git-sha256") ||
        (scheme == "git-sha1" && !lower_hex(revision, 40U)) ||
        (scheme == "git-sha256" && !lower_hex(revision, 64U))) {
        throw engine::Error("local_git.revision_invalid", "local Git revision input is invalid", 4);
    }
    static_cast<void>(run_git({"--no-replace-objects", "cat-file", "-e", revision + "^{commit}"}, 4096U, deadline_unix_ms));
    const auto tree = run_git(
        {"--no-replace-objects", "ls-tree", "-r", "-z", "--full-tree", revision},
        engine::Limits::max_snapshot_file_bytes,
        deadline_unix_ms);
    engine::Json result{
        {"protocol", evidence_protocol},
        {"adapter_id", "symphony-sclv-evidence-local-git"},
        {"adapter_version", adapter_version},
        {"provider_namespace", "local-git"},
        {"evidence_kind", "revision"},
        {"observed_at", observed_at},
        {"source_reference", source_reference},
        {"repository", engine::Json{
            {"revision_scheme", scheme},
            {"revision_value", revision},
            {"tree_digest", engine::tagged_sha256(tree)},
        }},
        {"change_request", engine::Json{
            {"state", "not_applicable"},
            {"provider", "not_applicable"},
            {"id", "not_applicable"},
            {"reference", "not_applicable"},
            {"absence_reason", "local Git revision evidence contains no forge change request"},
        }},
        {"ratification", engine::Json{
            {"state", "not_asserted"},
            {"subject", "not_applicable"},
            {"effective_permission", "not_applicable"},
            {"method", "not_applicable"},
            {"evidence_reference", "not_applicable"},
            {"evidence_digest", "not_applicable"},
            {"absence_reason", "local Git revision evidence does not assert ratification"},
        }},
    };
    result["evidence_digest"] = engine::tagged_sha256(result.dump());
    validate_evidence(result);
    return result;
}

}
