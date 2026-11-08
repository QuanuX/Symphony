#include "adapter.hpp"
#include <symphony/knowledge/engine/protocol.hpp>
#include <iostream>

int main(int argc, char** argv) {
    try {
        if (argc == 2 && std::string(argv[1]) == "--describe") {
            std::cout << cloudflare::owner().dump() << '\n';
            return 0;
        }
        if (argc != 1) return 2;
        auto input = symphony::knowledge::engine::read_bounded(std::cin, cloudflare::frame_limit);
        auto result = cloudflare::process(cloudflare::parse(input), std::filesystem::canonical(argv[0]));
        auto output = result.dump();
        if (output.size() >= cloudflare::frame_limit) return 5;
        std::cout << output << '\n';
        return result.at("outcome") == "completed" ? 0 : 5;
    } catch (...) {
        // Neither submitted text nor child diagnostics are safe error channels.
        std::cerr << "Cloudflare control document rejected; no result available\n";
        return 2;
    }
}
