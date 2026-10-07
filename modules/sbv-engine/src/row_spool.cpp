#include "row_spool.hpp"
#include <array>
#include <limits>
#include <utility>

namespace symphony::sbv::result_store {
namespace {
constexpr std::array sections{
    "choices", "comparisons", "diagnostics", "distributions", "execution",
    "provenance", "replay", "resources", "search", "signals", "studies",
    "summary"};
constexpr std::string_view row_pointer = "/sections/signals/data";
Json unselected() {
  return {{"data", nullptr},
          {"reason", "private row spool; no calculation section selected"},
          {"status", "not_selected"}};
}
void stream_value(ResultWriter &writer, const Json &value) {
  if (value.is_object()) {
    writer.begin_object();
    for (auto it = value.begin(); it != value.end(); ++it) {
      writer.key(it.key());
      stream_value(writer, it.value());
    }
    writer.end();
  } else if (value.is_array()) {
    writer.begin_array();
    for (const auto &child : value)
      stream_value(writer, child);
    writer.end();
  } else {
    writer.value(value);
  }
}
} // namespace

struct RowSpool::Impl {
  enum class State { appending, closing, closed, failed };
  ResultWriter writer;
  ReadOptions read_options;
  Checkpoint checkpoint;
  std::uint64_t rows = 0;
  State state = State::appending;

  Impl(std::string path, WriteOptions write, ReadOptions read, Checkpoint check)
      : writer(std::move(path), write, check), read_options(read),
        checkpoint(std::move(check)) {
    try {
      writer.begin_object();
      writer.key("origin");
      writer.value("sbv-private-row-spool");
      writer.key("protocol");
      writer.value("symphony.sbv.result.v1");
      writer.key("sections");
      writer.begin_object();
      for (const auto *name : sections) {
        writer.key(name);
        if (std::string_view(name) == "signals") {
          writer.begin_object();
          writer.key("data");
          writer.begin_array();
          break;
        }
        writer.value(unselected());
      }
    } catch (...) {
      state = State::failed;
      throw;
    }
  }

  void require_appending() const {
    if (state != State::appending)
      throw StoreError("bundle.spool_state", "row spool is not appendable",
                       writer.recovery());
    if (rows == std::numeric_limits<std::uint64_t>::max())
      throw StoreError("bundle.spool_range", "row count exceeds uint64 range",
                       writer.recovery());
  }

  template <class Function> void append(Function &&function) {
    require_appending();
    try {
      if (checkpoint)
        checkpoint();
      function();
      ++rows;
    } catch (...) {
      // Even a partial row can have contributed bytes/pages. Do not close a
      // apparently successful prefix or silently resume after an exception.
      state = State::failed;
      throw;
    }
  }
};

RowSpool::RowSpool(std::string path, WriteOptions write, ReadOptions read,
                   Checkpoint checkpoint)
    : p_(std::make_unique<Impl>(std::move(path), write, read,
                                std::move(checkpoint))) {}
RowSpool::~RowSpool() = default;
void RowSpool::append(const Json &row) {
  p_->append([&] { stream_value(p_->writer, row); });
}
void RowSpool::append(const NodeHandle &row, ValueLimits limits) {
  p_->append([&] { (void)p_->writer.append_subtree(row, limits); });
}
ClosedRows RowSpool::close() {
  if (p_->state != Impl::State::appending)
    throw StoreError("bundle.spool_state", "row spool cannot be closed",
                     p_->writer.recovery());
  p_->state = Impl::State::closing;
  try {
    p_->writer.end(); // ordered array
    p_->writer.key("reason");
    p_->writer.value("");
    p_->writer.key("status");
    p_->writer.value("available");
    p_->writer.end(); // signals section
    bool after_signals = false;
    for (const auto *name : sections) {
      if (after_signals) {
        p_->writer.key(name);
        p_->writer.value(unselected());
      }
      after_signals |= std::string_view(name) == "signals";
    }
    p_->writer.end(); // sections
    p_->writer.key("status");
    p_->writer.value("completed");
    p_->writer.end(); // scratch envelope body
    auto receipt = p_->writer.finish();
    ResultReader reader(receipt.reference, p_->read_options, p_->checkpoint);
    auto rows = reader.select(row_pointer);
    const auto info = rows.describe();
    if (info.at("kind") != "array" ||
        info.at("children") != std::to_string(p_->rows))
      throw StoreError("bundle.spool_count", "closed row count contradiction",
                       p_->writer.recovery());
    auto verification = rows.hash();
    p_->state = Impl::State::closed;
    return {std::move(rows), std::move(receipt), std::move(verification), p_->rows};
  } catch (...) {
    p_->state = Impl::State::failed;
    throw;
  }
}
std::uint64_t RowSpool::size() const { return p_->rows; }
Json RowSpool::recovery() const {
  std::string state;
  switch (p_->state) {
  case Impl::State::appending: state = "appending"; break;
  case Impl::State::closing: state = "closing"; break;
  case Impl::State::closed: state = "closed"; break;
  case Impl::State::failed: state = "failed"; break;
  }
  return {{"purpose", "private_row_spool"},
          {"state", state},
          {"appended_rows", std::to_string(p_->rows)},
          {"row_pointer", row_pointer},
          {"scratch", p_->writer.recovery()},
          {"financial_result_published", false},
          {"automatic_cleanup", false}};
}
} // namespace symphony::sbv::result_store
