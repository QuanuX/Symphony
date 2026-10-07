#pragma once

#include "logical_value.hpp"
#include "row_spool.hpp"
#include <cstdint>
#include <functional>
#include <memory>
#include <string>
#include <string_view>

namespace symphony::sbv::detail {
using PartitionedBody =
    std::function<logical::Value(class PartitionedOutput &)>;

// Operation-owned private workspace; caller selects paths and cleanup.
// Producer callbacks begin only after workspace durability and actual alias
// checks. Each spool is single-caller and owns its independent page writer.
class PartitionedOutput {
public:
  ~PartitionedOutput();
  PartitionedOutput(const PartitionedOutput &) = delete;
  PartitionedOutput &operator=(const PartitionedOutput &) = delete;
  const result_store::Json &output() const;
  const std::string &workspace_path() const;
  const std::string &bundle_path() const;
  result_store::WriteOptions write_options() const;
  result_store::ReadOptions read_options() const; // unlimited pages; cache 0
  result_store::Checkpoint checkpoint() const;
  // Private path component: [a-z][a-z0-9_-]{0,63}. This bounds internal path
  // encoding, not the number of spools or amount of work.
  std::unique_ptr<result_store::RowSpool> spool(std::string_view name);

private:
  struct Impl;
  std::unique_ptr<Impl> p_;
  explicit PartitionedOutput(const result_store::Json &, std::int64_t);
  friend result_store::Json partitioned_result(const result_store::Json &,
                                               const std::string &,
                                               std::int64_t,
                                               const PartitionedBody &);
};

// Request retains exact caller output selection. slug is the existing result
// protocol slug (generate-census, evaluate, economics, ...). The callback
// returns a four-field result-v1 BODY, without content_sha256. No automatic
// cleanup, resume, retry or publication of an incomplete financial result.
result_store::Json partitioned_result(const result_store::Json &request,
                                      const std::string &slug, std::int64_t end,
                                      const PartitionedBody &);
} // namespace symphony::sbv::detail
