#include "native_process.hpp"

void test_installed_requests(const native_test::Arguments& args) {
    snv_native_test::test_installed_requests(args, "snrv");
}

int main(int argc, char** argv) {
    return native_test::test_main([&] { test_installed_requests(native_test::Arguments(argc, argv)); });
}
