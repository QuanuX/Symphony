#pragma once

#include "result_store.hpp"
#include <cstdint>
#include <memory>
#include <string>

namespace symphony::sbv::result_store {

// Private internal row spool. The scratch bundle is a complete storage
// envelope, never a declaration that a backtest/census has completed.
struct ClosedRows {
  NodeHandle rows;
  Receipt scratch_receipt;
  Json array_verification;
  std::uint64_t row_count;
};

// One single-caller append stream. Writer pages use the selected existing
// physical layout. No aggregate row cap or default deadline is introduced.
// Scratch directories and incomplete pages are retained; the operation owner
// chooses cleanup after all descendants have copied their required closure.
class RowSpool {
public:
  RowSpool(std::string scratch_bundle_path, WriteOptions = {},
           ReadOptions = {}, Checkpoint = {});
  ~RowSpool();
  RowSpool(const RowSpool &) = delete;
  RowSpool &operator=(const RowSpool &) = delete;

  // Json is consumed synchronously and never retained. This recursively streams
  // supplied structured rows, so an entire row need not fit one physical page.
  void append(const Json &);
  // A source row stays owned by its handle and is verified/copied synchronously.
  void append(const NodeHandle &, ValueLimits = {});
  // Exactly once, after any provider completion/domain admission required by
  // the caller. Returns an owned handle, not a caller-editable page descriptor.
  ClosedRows close();
  std::uint64_t size() const;
  Json recovery() const;

private:
  struct Impl;
  std::unique_ptr<Impl> p_;
};

} // namespace symphony::sbv::result_store
