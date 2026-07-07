#include "cli.hpp"
#include "paths.hpp"
#include "skvi_index.hpp"
#include "sclv_changelog.hpp"
#include "cross_reference.hpp"
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
            if (!result.is_valid_directory) {
                return 2;
            }

            std::string index_path = args[2] + "/knowledge/skvi/INDEX.md";
            SkviCheckResult skvi_result = check_skvi_index(index_path);
            for (const auto& msg : skvi_result.messages) {
                std::cout << msg << "\n";
            }

            if (!skvi_result.success) {
                return 3;
            }

            std::string changelog_path = args[2] + "/knowledge/sclv/CHANGELOG.md";
            SclvCheckResult sclv_result = check_sclv_changelog(changelog_path);
            for (const auto& msg : sclv_result.messages) {
                std::cout << msg << "\n";
            }

            if (!sclv_result.success) {
                return 4;
            }

            CrossReferenceResult cross_result = check_cross_references(args[2], skvi_result, sclv_result);
            for (const auto& msg : cross_result.messages) {
                std::cout << msg << "\n";
            }

            if (cross_result.success) {
                return 0;
            } else {
                return 5;
            }
        } else {
            std::cerr << "error: check requires --repo <path>\n";
            return 1;
        }
    }

    std::cerr << "error: unknown command or invalid arguments\n";
    return 1;
}
