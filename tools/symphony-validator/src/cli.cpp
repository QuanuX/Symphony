#include "cli.hpp"
#include "paths.hpp"
#include <iostream>

int run_cli(const std::vector<std::string>& args) {
    if (args.empty()) {
        std::cerr << "error: no arguments provided\n";
        return 1;
    }

    const std::string& command = args[0];

    if (command == "--help") {
        std::cout << "Usage: symphony-validator [command] [options]\n"
                  << "Commands:\n"
                  << "  --help                Show this help message\n"
                  << "  --version             Show version information\n"
                  << "  check --repo <path>   Check repository path validity\n";
        return 0;
    }

    if (command == "--version") {
        std::cout << "symphony-validator 0.1.0-seed\n";
        return 0;
    }

    if (command == "check") {
        if (args.size() == 3 && args[1] == "--repo") {
            PathCheckResult result = check_repository_path(args[2]);
            std::cout << result.message << "\n";
            if (result.is_valid_directory) {
                return 0;
            } else {
                return 2;
            }
        } else {
            std::cerr << "error: check requires --repo <path>\n";
            return 1;
        }
    }

    std::cerr << "error: unknown command or invalid arguments\n";
    return 1;
}
