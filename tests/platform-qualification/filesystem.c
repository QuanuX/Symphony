#include <stdio.h>
#include <sys/mount.h>
int main(int argc, char **argv) {
    struct statfs observed;
    if (argc != 2 || statfs(argv[1], &observed) != 0) return 1;
    return printf("%s\n", observed.f_fstypename) < 0;
}
