#include "symphony/knowledge/engine/error.hpp"

#include <utility>

namespace symphony::knowledge::engine {

Error::Error(std::string code, std::string message, int exit_status)
    : std::runtime_error(std::move(message)), code_(std::move(code)), exit_status_(exit_status) {}

const std::string& Error::code() const noexcept {
    return code_;
}

int Error::exit_status() const noexcept {
    return exit_status_;
}

}
