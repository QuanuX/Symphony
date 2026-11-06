#pragma once
#include <nlohmann/json.hpp>
#include <functional>
#include <memory>
#include <string>

namespace symphony::srrapiv {
using Json = nlohmann::json;

// Construction is offline. runtime.initialize explicitly creates the SDK engine;
// login remains a separate operation. The emitter must enqueue without blocking.
class Vendor final {
public:
    using Emit = std::function<void(Json)>;
    Vendor(Json config, Emit emit);
    ~Vendor();
    Vendor(const Vendor&) = delete;
    Vendor& operator=(const Vendor&) = delete;
    Json invoke(const std::string& overload_id, const Json& params);
    Json describe() const;
    void shutdown();
private:
    struct Impl;
    std::unique_ptr<Impl> impl_;
};
}
