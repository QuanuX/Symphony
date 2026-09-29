// Isolated C++26 test process. No production target or credential handling.
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#include <cstring>

int main(int argc, char** argv) {
    if (argc != 2) return 2;
    if (std::strcmp(argv[1], "--hold") == 0) { sleep(10); return 0; }
    const int fd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (fd < 0) return 3;
    sockaddr_un address{};
    address.sun_family = AF_UNIX;
    if (std::strlen(argv[1]) >= sizeof(address.sun_path)) return 4;
    std::strcpy(address.sun_path, argv[1]);
    if (connect(fd, reinterpret_cast<sockaddr*>(&address), sizeof(address)) != 0) return 5;
    if (write(fd, "R", 1) != 1) return 6;
    char command{};
    if (read(fd, &command, 1) != 1) return 7;
    if (command == 'P') {
        if (write(fd, "M", 1) != 1 || read(fd, &command, 1) != 1) return 9;
    }
    if (command == 'E') {
        // Retain the connected descriptor across exec to exercise stale-token
        // rejection independently from EOF or PID changes.
        execl(argv[0], argv[0], "--hold", static_cast<char*>(nullptr));
        return 8;
    }
    if (command == 'C') {
        close(fd);
        sleep(10);
        return 0;
    }
    close(fd);
    return 0;
}
