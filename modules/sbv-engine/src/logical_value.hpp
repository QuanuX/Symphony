#pragma once
#include "result_store.hpp"
#include <map>
#include <memory>
#include <optional>
#include <string>
#include <vector>

namespace symphony::sbv::logical {
using Json = result_store::Json;
using Limits = result_store::ValueLimits;
using Checkpoint = result_store::Checkpoint;
class Cursor;
// Owns either an immutable admitted JSON value, a lifetime-owning store node,
// or a composed object whose children may come from either representation.
// No aggregate array needs to become a Json. Store-backed values/cursors share
// a single-caller cache; workers use independent readers or bounded owned rows.
class Value {
public:
  explicit Value(Json);
  explicit Value(result_store::NodeHandle);
  static Value object(std::map<std::string, Value, std::less<>>);
  static Value array(std::vector<Value>);
  // Separate store/cache state for one worker. Immutable JSON storage may be
  // shared. This rebinds the same exact node; admission remains caller-owned.
  Value independent_reader(result_store::ReadOptions, Checkpoint = {}) const;
  std::string kind() const;   // object, array or scalar across both backends
  std::uint64_t size() const; // immediate children; scalar has zero
  bool contains(std::string_view) const;
  Value at(std::string_view) const;
  Value at(std::uint64_t) const;
  // Object transformations retain every unmodified child as an owned view.
  Value with(std::string, Value) const;
  Value without(std::string_view) const;
  Cursor children(std::uint64_t first = 0,
                  std::optional<std::uint64_t> count = {}) const;
  void visit(const result_store::ValueVisitor &, Checkpoint = {}) const;
  // Provisional bytes/callbacks must not be interpreted as complete until the
  // call returns. The final byte is withheld until verification succeeds. Hash
  // is SHA256 of the full selected value, including any embedded
  // content_sha256. Result-body hashing explicitly removes that key.
  Json export_json(const result_store::Sink &, Limits = {},
                   Checkpoint = {}) const;
  std::string sha256(Limits = {}, Checkpoint = {}) const;
  // Use for bounded metadata or a typed row; not an aggregate array fallback.
  Json materialize(Limits = {}, Checkpoint = {}) const;
  void write(result_store::ResultWriter &) const;

private:
  struct Impl;
  std::shared_ptr<const Impl> p_;
  explicit Value(std::shared_ptr<const Impl>);
  friend class Cursor;
};
struct Child {
  std::string edge;
  Value value;
};
class Cursor {
public:
  ~Cursor();
  Cursor(Cursor &&) noexcept;
  Cursor &operator=(Cursor &&) noexcept;
  Cursor(const Cursor &) = delete;
  Cursor &operator=(const Cursor &) = delete;
  std::optional<Child> next();

private:
  struct Impl;
  std::unique_ptr<Impl> p_;
  explicit Cursor(std::unique_ptr<Impl>);
  friend class Value;
};
} // namespace symphony::sbv::logical
