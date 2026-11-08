#include "adapter.hpp"
#include <fstream>
#include <iostream>

using cloudflare::Json;
namespace fs = std::filesystem;
int main(int argc, char** argv) {
    try {
        if (argc < 2 || argc > 3) throw std::runtime_error("repository root [--check] required");
        auto root = fs::canonical(argv[1]); bool check = argc == 3 && std::string(argv[2]) == "--check";
        std::ifstream in(root / "knowledge/sov/cloudflare/owner.json");
        auto owner = cloudflare::parse(std::string(std::istreambuf_iterator<char>(in), {}));
        std::map<fs::path, std::string> output;
        auto schemas = root / "knowledge/sov/cloudflare/schemas";
        std::string operations = "// Code generated from knowledge/sov/cloudflare/owner.json; DO NOT EDIT.\npackage sovcloudflare\n\nvar Operations = []string{\n";
        for (const auto& op : owner.at("operations")) {
            auto name = op.at("operation").get<std::string>();
            for (const auto* kind : {"input_schema", "template"}) output[schemas / (name + "." + kind + ".json")] = op.at(kind).dump(2) + "\n";
            for (const std::string kind : {"request", "result"}) {
                auto schema = owner.at(kind + "_schema"); schema["properties"]["operation"] = Json{{"const", name}};
                if (kind == "request") schema["properties"]["input"] = op.at("input_schema");
                else schema["properties"]["data"] = op.at("implemented").get<bool>() ? Json{{"oneOf", Json::array({Json{{"type", "null"}}, op.at("data_schema")})}} : Json{{"type", "null"}};
                output[schemas / (name + "." + kind + "_schema.json")] = schema.dump(2) + "\n";
            }
            if (op.at("implemented").get<bool>()) operations += "\t\"" + name + "\",\n";
        }
        output[root / "tools/qxctl/internal/sovcloudflare/operations_generated.go"] = operations + "}\n";
        for (const auto& [path, text] : output) {
            if (check) { std::ifstream f(path); if (std::string(std::istreambuf_iterator<char>(f), {}) != text) throw std::runtime_error("stale projection: " + path.string()); }
            else { std::ofstream f(path); f << text; if (!f) throw std::runtime_error("projection write failed"); }
        }
        std::cout << "Cloudflare owner projections " << (check ? "verified" : "generated") << '\n'; return 0;
    } catch (const std::exception& e) { std::cerr << e.what() << '\n'; return 1; }
}
