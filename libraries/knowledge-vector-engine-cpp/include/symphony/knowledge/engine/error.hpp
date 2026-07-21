#pragma once

#include <stdexcept>
#include <string>

namespace symphony::knowledge::engine {

class Error final : public std::runtime_error {
public:
    Error(std::string code, std::string message, int exit_status);

    [[nodiscard]] const std::string& code() const noexcept;
    [[nodiscard]] int exit_status() const noexcept;

private:
    std::string code_;
    int exit_status_;
};

}
