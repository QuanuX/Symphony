#include "adapter.hpp"
#include "connector.hpp"
#include <symphony/knowledge/engine/protocol.hpp>
#include <iostream>
#include <sys/resource.h>

int main(int argc, char** argv) {
    rlimit core{0,0}; if (::setrlimit(RLIMIT_CORE, &core) != 0) return 2;
    try {
        if(argc==2 && std::string(argv[1])=="--connector-supervise") return cloudflare::connector_supervise();
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
