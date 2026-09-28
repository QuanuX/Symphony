#ifndef SYMPHONY_SOURCE_JSON_HPP
#define SYMPHONY_SOURCE_JSON_HPP
#include <nlohmann/json.hpp>
#include <symphony/source/support.hpp>
namespace symphony::source {
// Parser for provider responses, not the knowledge process protocol. Numeric
// JSON projections are inspection aids; retain original bytes for exact decimal
// evidence. Public provider adapters must not reserialize financial values from
// this DOM. Duplicate keys, excessive depth/events/strings and invalid UTF-8
// fail.
[[nodiscard]] Status parse_json(Bytes, const JsonLimits &,
                                nlohmann::json &) noexcept;
} // namespace symphony::source
#endif
