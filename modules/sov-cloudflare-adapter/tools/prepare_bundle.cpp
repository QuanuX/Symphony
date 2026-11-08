#include "adapter.hpp"
#include <fstream>
#include <iostream>
#include <unistd.h>
#include <sys/stat.h>

using cloudflare::Json;
namespace fs = std::filesystem;
int main(int argc, char** argv) {
    try {
        if (argc != 7 || std::string(argv[1]) != "--node" || std::string(argv[3]) != "--cf" || std::string(argv[5]) != "--output" || ::geteuid() == 0) throw std::runtime_error("non-root --node DIR --cf DIR --output NEW_DIR required");
        auto node = fs::canonical(argv[2]), cf = fs::canonical(argv[4]); auto target = fs::absolute(argv[6]).lexically_normal();
        if (fs::exists(target)) throw std::runtime_error("output already exists");
        fs::create_directories(target / "node/bin");
        fs::copy_file(node / "bin/node", target / "node/bin/node"); fs::copy_file(node / "LICENSE", target / "node/LICENSE");
        fs::copy(cf, target / "cf", fs::copy_options::recursive | fs::copy_options::copy_symlinks);
        std::vector<fs::path> links;
        for (const auto& e : fs::recursive_directory_iterator(target / "cf")) if (e.path().filename() == ".bin" && fs::is_directory(e.symlink_status())) links.push_back(e.path());
        for (const auto& p : links) fs::remove_all(p); // Unused npm launch links, never selected entrypoints.
        std::vector<fs::path> files;
        for (const auto& e : fs::recursive_directory_iterator(target)) {
            if (fs::is_directory(e.symlink_status())) ::chmod(e.path().c_str(), 0700);
            else if (fs::is_regular_file(e.symlink_status())) files.push_back(e.path());
            else throw std::runtime_error("special file or symlink in dependency closure");
        }
        ::chmod(target.c_str(), 0700); std::sort(files.begin(), files.end());
        Json inventory = Json::array();
        for (const auto& p : files) {
            auto rel = p.lexically_relative(target).generic_string(); struct stat st{};
            if (::lstat(p.c_str(), &st) != 0 || st.st_nlink != 1) throw std::runtime_error("unsafe package file");
            const bool executable = rel == "node/bin/node" || (st.st_mode & 0111) != 0; ::chmod(p.c_str(), executable ? 0700 : 0600);
            inventory.push_back(Json{{"path", rel}, {"digest", cloudflare::file_digest(p)}, {"size", fs::file_size(p)}, {"executable", executable}});
        }
#ifdef __APPLE__
        const char* os = "macos";
#else
        const char* os = "linux";
#endif
#if defined(__arm64__) || defined(__aarch64__)
        const char* arch = "arm64";
#else
        const char* arch = "amd64";
#endif
        Json manifest{{"protocol", "symphony.sov.cloudflare-bundle.v1"}, {"cf_version", "1.0.0-beta.28"}, {"node_version", "26.12.0"}, {"cf_integrity", cloudflare::owner().at("bundle_schema").at("properties").at("cf_integrity").at("const")}, {"os", os}, {"architecture", arch}, {"entrypoint", "cf/node_modules/cf/bin/cf"}, {"node", "node/bin/node"}, {"scripts_executed", false}, {"files", inventory}};
        cloudflare::validate(manifest, cloudflare::owner().at("bundle_schema"));
        std::ofstream output(target / "bundle.json"); output << manifest.dump() << '\n'; output.close(); ::chmod((target / "bundle.json").c_str(), 0600);
        std::cout << cloudflare::digest(manifest, "symphony.sov.cloudflare-bundle.v1") << '\n'; return 0;
    } catch (const std::exception& e) { std::cerr << e.what() << '\n'; return 1; }
}
