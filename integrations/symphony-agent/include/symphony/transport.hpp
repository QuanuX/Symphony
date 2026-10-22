#pragma once

#include <cstddef>
#include <map>
#include <string>
#include <vector>

namespace symphony {

struct ProcessResult {
    int exit_code = -1;
    bool timed_out = false;
    bool output_limited = false;
    std::string out;
    std::string err;
};

// Does not invoke a shell, search PATH, or inherit the ambient environment.
// max_output_bytes bounds stdout and stderr together. A timeout or output limit
// sends SIGTERM, permits at most 500ms for owner cleanup, then forcibly cleans
// the child process group. The grace period is additional to timeout_ms.
// No automatic execution retry is made; termination does not prove rollback.
ProcessResult run_process(const std::string& executable,
                          const std::vector<std::string>& args,
                          const std::string& cwd,
                          const std::map<std::string, std::string>& env,
                          int timeout_ms, std::size_t max_output_bytes);

std::string random_token();

// Local transport is a same-effective-UID boundary, not a distinct-principal
// authorization service. Paths must be absolute; their immediate parent must
// be an existing, non-symlink directory owned by this UID with mode 0700.
// Returned descriptors are blocking and close-on-exec; callers close them.
// The listener refuses existing paths, including stale sockets and symlinks.
int listen_local_socket(const std::string& path);
int connect_local_socket(const std::string& path);
int accept_local_socket(int listening_fd);
// Removes only the exact socket inode created by this process at this path.
void remove_local_socket(const std::string& path);

} // namespace symphony
