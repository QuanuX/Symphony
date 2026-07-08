#include "cli.hpp"
#include "paths.hpp"
#include "skvi_index.hpp"
#include "sclv_changelog.hpp"
#include "cross_reference.hpp"
#include "vocabulary.hpp"
#include "sclv_shape.hpp"
#include "artifacts.hpp"
#include "canonical_surfaces.hpp"
#include "validator_contracts.hpp"
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
            int pass_count = 0;
            int warning_count = 0;
            int violation_count = 0;
            int final_exit = 0;

            auto process_msg = [&](const std::string& msg) {
                std::cout << msg << "\n";
                if (msg.find("evidence pass") == 0) pass_count++;
                else if (msg.find("evidence warning") == 0) warning_count++;
                else if (msg.find("evidence violation") == 0) violation_count++;
            };

            auto process_messages = [&](const std::vector<std::string>& messages) {
                for (const auto& msg : messages) {
                    process_msg(msg);
                }
            };

            auto print_summary = [&]() {
                std::cout << "summary pass=" << pass_count << " warning=" << warning_count << " violation=" << violation_count << " exit=" << final_exit << "\n";
            };

            PathCheckResult result = check_repository_path(args[2]);
            if (!result.message.empty()) {
                std::cout << result.message << "\n";
            }
            if (!result.is_valid_directory) {
                final_exit = 2;
                print_summary();
                return final_exit;
            }
            CanonicalSurfaceCheckResult canonical_result = check_required_canonical_surfaces(args[2]);
            process_messages(canonical_result.messages);
            if (!canonical_result.success) {
                final_exit = 9;
                print_summary();
                return final_exit;
            }

            ValidatorContractShapeResult validator_contract_result = check_validator_contract_shapes(args[2]);
            process_messages(validator_contract_result.messages);
            if (!validator_contract_result.success) {
                final_exit = 10;
                print_summary();
                return final_exit;
            }

            std::string index_path = args[2] + "/knowledge/skvi/INDEX.md";
            SkviCheckResult skvi_result = check_skvi_index(index_path);
            process_messages(skvi_result.messages);
            if (!skvi_result.success) {
                final_exit = 3;
                print_summary();
                return final_exit;
            }

            std::string changelog_path = args[2] + "/knowledge/sclv/CHANGELOG.md";
            SclvCheckResult sclv_result = check_sclv_changelog(changelog_path);
            process_messages(sclv_result.messages);
            if (!sclv_result.success) {
                final_exit = 4;
                print_summary();
                return final_exit;
            }

            CrossReferenceResult cross_result = check_cross_references(args[2], skvi_result, sclv_result);
            process_messages(cross_result.messages);
            if (!cross_result.success) {
                final_exit = 5;
                print_summary();
                return final_exit;
            }

            VocabularyCheckResult vocab_result = check_vocabulary(skvi_result, sclv_result);
            process_messages(vocab_result.messages);
            if (!vocab_result.success) {
                final_exit = 6;
                print_summary();
                return final_exit;
            }

            SclvShapeCheckResult shape_result = check_sclv_shapes(sclv_result);
            process_messages(shape_result.messages);
            if (!shape_result.success) {
                final_exit = 7;
                print_summary();
                return final_exit;
            }

            ArtifactCheckResult artifact_result = check_unauthorized_artifacts(args[2]);
            process_messages(artifact_result.messages);
            if (!artifact_result.success) {
                final_exit = 8;
                print_summary();
                return final_exit;
            }

            final_exit = 0;
            print_summary();
            return final_exit;
        } else {
            std::cerr << "error: check requires --repo <path>\n";
            return 1;
        }
    }

    std::cerr << "error: unknown command or invalid arguments\n";
    return 1;
}
