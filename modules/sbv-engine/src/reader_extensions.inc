// Private reader extension; included inside result_store namespace so it
// can reuse the unchanged physical reader. No installed ABI/wire is added.
struct NodeHandle::Impl {
  std::shared_ptr<Engine> engine;
  Reference reference;
  Located location;
};
NodeHandle::NodeHandle(std::shared_ptr<Impl> p) : p_(std::move(p)) {}
NodeHandle ResultReader::select(std::string_view pointer) {
  return NodeHandle(std::make_shared<NodeHandle::Impl>(NodeHandle::Impl{
      p_->engine, p_->reference, p_->engine->locate_pointer(pointer)}));
}
NodeHandle ResultReader::select_node(std::uint64_t id) {
  return NodeHandle(std::make_shared<NodeHandle::Impl>(NodeHandle::Impl{
      p_->engine, p_->reference, p_->engine->locate_id(id)}));
}
namespace {
Counts logical_counts(const Located &n, const Reference &r) {
  auto counts = node_counts(n.node, n.depth);
  if (n.root) {
    counts.values = add(counts.values, 1);
    counts.nodes = add(counts.nodes, 2);
    counts.bytes = add(counts.bytes,
        add(add(Json("content_sha256").dump().size(), 2), Json(r.content_sha256).dump().size()));
  }
  return counts;
}
Located located_child(const Located &parent, const Engine::Child &child) {
  const auto id = add(add(parent.id, parent.root ? 2 : 1), child.prefix_values);
  return {child.node, id, dec(parent.id),
          kind(parent.node) == "object" ? child.key : Json(dec(child.index)),
          add(parent.depth, 1), false};
}
Located virtual_digest(const Reference &r) {
  return {inline_node(r.content_sha256), 1, "0", "content_sha256", 1, false};
}
}
Json NodeHandle::describe() const {
  need(static_cast<bool>(p_), "moved node handle");
  const auto &n = p_->location;
  auto result = p_->engine->node_row(n);
  result.erase("value");
  const auto c = logical_counts(n, p_->reference);
  result["reference"] = ref_json(p_->reference);
  result["depth"] = dec(n.depth);
  result["values"] = dec(c.values);
  result["nodes"] = dec(c.nodes);
  result["bytes"] = dec(c.bytes);
  return result;
}
NodeHandle NodeHandle::child(std::string_view key) const {
  need(static_cast<bool>(p_), "moved node handle");
  const auto &n = p_->location;
  need(kind(n.node) == "object", "object member selection requires object");
  need(key.size() <= 65536, "member key exceeds logical string profile");
  check(p_->engine->checkpoint);
  auto selected = n.root && key == "content_sha256"
      ? virtual_digest(p_->reference)
      : located_child(n, p_->engine->by_key(n.node, std::string(key)));
  (void)logical_counts(selected, p_->reference);
  return NodeHandle(std::make_shared<Impl>(Impl{p_->engine, p_->reference, std::move(selected)}));
}
NodeHandle NodeHandle::child(std::uint64_t index) const {
  need(static_cast<bool>(p_), "moved node handle");
  const auto &n = p_->location;
  need(container(n.node), "child selection requires container");
  need(index < add(child_count(n.node), n.root ? 1 : 0), "child index out of range");
  check(p_->engine->checkpoint);
  auto selected = n.root && index == 0
      ? virtual_digest(p_->reference)
      : located_child(n, p_->engine->by_index(n.node, index - (n.root ? 1 : 0)));
  (void)logical_counts(selected, p_->reference);
  return NodeHandle(std::make_shared<Impl>(Impl{p_->engine, p_->reference, std::move(selected)}));
}

struct ChildCursor::Impl {
  NodeHandle parent;
  std::uint64_t first, next, stop, total, prefix_values = 0;
  bool initialized = false, failed = false;
  struct Frame { std::shared_ptr<const Json> page; std::size_t slot; };
  std::vector<Frame> frames;
  Json::const_iterator inline_next;
  explicit Impl(NodeHandle value, std::uint64_t begin, std::optional<std::uint64_t> count)
      : parent(std::move(value)), first(begin), next(begin), stop(0), total(0) {
    const auto &n = parent.p_->location;
    need(container(n.node), "child cursor requires a container");
    total = add(child_count(n.node), n.root ? 1 : 0);
    need(first <= total, "child range starts outside container");
    if (count) need(*count <= total - first, "selected child range extends outside container");
    stop = count ? add(first, *count) : total;
  }
  Engine &engine() { return *parent.p_->engine; }
  const Located &location() const { return parent.p_->location; }
  void initialize(std::uint64_t target) {
    auto &e = engine(); const auto &n = location();
    const auto count = child_count(n.node);
    const bool object = kind(n.node) == "object";
    if (n.node.at("kind") == "inline") {
      inline_next = n.node.at("value").begin();
      for (std::uint64_t i = 0; i < target; ++i, ++inline_next) {
        check(e.checkpoint);
        prefix_values = add(prefix_values, counts(inline_next.value()).values);
      }
      initialized = true;
      return;
    }
    const auto &index = n.node.at("index");
    validate_ref(index, object, count, num(e.manifest.at("page_bytes")));
    need(num(index.at("start")) == 0 && num(index.at("count")) == count &&
         num(index.at("values")) == node_counts(n.node).values - 1,
         "cursor container index metrics mismatch");
    Json selected = index;
    bool root = true;
    for (;;) {
      const auto owned = e.page(selected, object, count);
      const auto &page = *owned;
      const auto level = num(page.at("level"));
      const auto &entries = page.at("entries");
      if (root && level > 0) need(entries.size() >= 2, "cursor unary index root");
      root = false;
      std::uint64_t ordinal = num(page.at("start"));
      bool found = false;
      for (std::size_t slot = 0; slot < entries.size(); ++slot) {
        const auto &entry = entries.at(slot);
        const auto covered = level ? num(entry.at("count")) : 1;
        const auto values = level ? num(entry.at("values")) : node_counts(entry.at("node")).values;
        if (target >= ordinal && target - ordinal < covered) {
          frames.push_back({owned, slot});
          if (!level) { initialized = true; return; }
          selected = entry;
          found = true;
          break;
        }
        ordinal = add(ordinal, covered);
        prefix_values = add(prefix_values, values);
      }
      need(found, "cursor selected child absent from index");
    }
  }
  void advance_pages(std::uint64_t target) {
    // Called only when another row in the explicitly selected range is needed.
    // An absent page after the selected range is not touched speculatively.
    if (frames.back().slot < frames.back().page->at("entries").size()) return;
    frames.pop_back();
    while (!frames.empty()) {
      auto &parent_frame = frames.back();
      ++parent_frame.slot;
      if (parent_frame.slot == parent_frame.page->at("entries").size()) {
        frames.pop_back();
        continue;
      }
      Json ref = parent_frame.page->at("entries").at(parent_frame.slot);
      for (;;) {
        need(num(ref.at("start")) == target, "cursor sibling range discontinuity");
        const auto owned = engine().page(ref, kind(location().node) == "object", child_count(location().node));
        frames.push_back({owned, 0});
        if (num(owned->at("level")) == 0) return;
        ref = owned->at("entries").front();
      }
    }
    fail("bundle.contract", "cursor index ended before declared children");
  }
  std::optional<NodeHandle> take() {
    need(!failed, "child cursor failed; create an explicit new cursor to retry");
    check(engine().checkpoint);
    if (next == stop) return std::nullopt;
    const auto &n = location();
    if (n.root && next == 0) {
      ++next;
      return NodeHandle(std::make_shared<NodeHandle::Impl>(NodeHandle::Impl{
          parent.p_->engine, parent.p_->reference, virtual_digest(parent.p_->reference)}));
    }
    const auto target = next - (n.root ? 1 : 0);
    if (!initialized) initialize(target);
    Engine::Child child;
    if (n.node.at("kind") == "inline") {
      child = {inline_node(inline_next.value()),
               kind(n.node) == "object" ? Json(inline_next.key()) : Json(nullptr),
               target, prefix_values};
      ++inline_next;
    } else {
      advance_pages(target);
      const auto &frame = frames.back();
      const auto &entry = frame.page->at("entries").at(frame.slot);
      need(add(num(frame.page->at("start")), frame.slot) == target,
           "cursor leaf ordinal mismatch");
      child = {entry.at("node"), entry.at("key"), target, prefix_values};
      ++frames.back().slot;
    }
    auto selected = located_child(n, child);
    (void)logical_counts(selected, parent.p_->reference);
    prefix_values = add(prefix_values, node_counts(child.node).values);
    ++next;
    return NodeHandle(std::make_shared<NodeHandle::Impl>(NodeHandle::Impl{
        parent.p_->engine, parent.p_->reference, std::move(selected)}));
  }
};
ChildCursor::ChildCursor(std::unique_ptr<Impl> p) : p_(std::move(p)) {}
ChildCursor::~ChildCursor() = default;
ChildCursor::ChildCursor(ChildCursor &&) noexcept = default;
ChildCursor &ChildCursor::operator=(ChildCursor &&) noexcept = default;
ChildCursor NodeHandle::children(std::uint64_t first, std::optional<std::uint64_t> count) const {
  need(static_cast<bool>(p_), "moved node handle");
  return ChildCursor(std::make_unique<ChildCursor::Impl>(*this, first, count));
}
std::optional<NodeHandle> ChildCursor::next() {
  need(static_cast<bool>(p_), "moved child cursor");
  try { return p_->take(); }
  catch (...) { p_->failed = true; throw; }
}
Json ChildCursor::progress() const {
  need(static_cast<bool>(p_), "moved child cursor");
  return {{"reference", ref_json(p_->parent.p_->reference)},
          {"selected_node_id", dec(p_->location().id)}, {"first", dec(p_->first)},
          {"next", dec(p_->next)}, {"end", dec(p_->stop)}, {"total", dec(p_->total)},
          {"complete", p_->next == p_->stop && !p_->failed}, {"failed", p_->failed},
          {"verification_extent", "accessed_pages"}, {"io_stats", p_->engine().stats.json()}};
}

Json NodeHandle::visit(const ValueVisitor &visitor, ValueLimits limits) const {
  need(static_cast<bool>(p_), "moved node handle");
  const auto declared = logical_counts(p_->location, p_->reference);
  if (limits.max_canonical_bytes) need(declared.bytes <= *limits.max_canonical_bytes,
      "selected subtree exceeds caller canonical-byte materialization budget");
  if (limits.max_counted_nodes) need(declared.nodes <= *limits.max_counted_nodes,
      "selected subtree exceeds caller counted-node materialization budget");
  check(p_->engine->checkpoint);
  if (p_->location.root) p_->engine->semantics();
  Hash canonical_hash, body_hash;
  Counts observed{0, 0, 0};
  auto count_node = [&](bool key) {
    observed.nodes = add(observed.nodes, 1);
    if (!key) observed.values = add(observed.values, 1);
    if (limits.max_counted_nodes) need(observed.nodes <= *limits.max_counted_nodes,
        "actual subtree exceeds caller counted-node materialization budget");
  };
  auto emit = [&](std::string_view bytes, bool include_body) {
    observed.bytes = add(observed.bytes, bytes.size());
    if (limits.max_canonical_bytes) need(observed.bytes <= *limits.max_canonical_bytes,
        "actual subtree exceeds caller canonical-byte materialization budget");
    canonical_hash.update(bytes);
    if (p_->location.root && include_body) body_hash.update(bytes);
  };
  std::function<void(const NodeHandle &, bool)> walk = [&](const NodeHandle &source, bool include_body) {
    check(p_->engine->checkpoint);
    const auto before = observed;
    const auto &n = source.p_->location;
    const auto wanted = logical_counts(n, source.p_->reference);
    count_node(false);
    if (!container(n.node)) {
      const auto &value = n.node.at("value");
      const auto bytes = value.dump(); emit(bytes, include_body);
      if (visitor.scalar) visitor.scalar(value);
    } else {
      const bool object = kind(n.node) == "object";
      emit(object ? "{" : "[", include_body);
      if (object) { if (visitor.begin_object) visitor.begin_object(); }
      else if (visitor.begin_array) visitor.begin_array();
      auto cursor = source.children();
      std::uint64_t index = 0;
      while (auto child = cursor.next()) {
        const bool virtual_child = n.root && index == 0;
        if (index) emit(",", include_body && !(n.root && index == 1));
        if (object) {
          count_node(true);
          const auto key = str(child->p_->location.edge);
          emit(Json(key).dump(), include_body && !virtual_child);
          emit(":", include_body && !virtual_child);
          if (visitor.key) visitor.key(key);
        }
        walk(*child, include_body && !virtual_child);
        ++index;
      }
      emit(object ? "}" : "]", include_body);
      if (visitor.end) visitor.end();
    }
    need(observed.values - before.values == wanted.values &&
         observed.nodes - before.nodes == wanted.nodes && observed.bytes - before.bytes == wanted.bytes,
         "selected subtree actual metrics contradict declared metrics");
  };
  walk(*this, true);
  const auto sha = canonical_hash.finish();
  if (p_->location.root) need(body_hash.finish() == p_->reference.content_sha256,
      "selected root logical content digest mismatch");
  check(p_->engine->checkpoint);
  return {{"reference", ref_json(p_->reference)}, {"node_id", dec(p_->location.id)},
          {"canonical_sha256", sha}, {"canonical_bytes", dec(observed.bytes)},
          {"values", dec(observed.values)}, {"nodes", dec(observed.nodes)},
          {"verification_extent", p_->location.root ? "full_logical_closure" : "selected_subtree"},
          {"io_stats", p_->engine->stats.json()}};
}
Json NodeHandle::hash(ValueLimits limits) const { return visit({}, limits); }
Json NodeHandle::read_value(ValueLimits limits) const {
  struct Frame { Json value; std::optional<std::string> key; };
  std::vector<Frame> stack;
  Json result;
  bool assigned = false;
  auto attach = [&](Json value) {
    if (stack.empty()) { need(!assigned, "multiple materialized roots"); result = std::move(value); assigned = true; }
    else if (stack.back().value.is_array()) stack.back().value.push_back(std::move(value));
    else { need(stack.back().key.has_value(), "materializer key missing");
      stack.back().value[*stack.back().key] = std::move(value); stack.back().key.reset(); }
  };
  ValueVisitor visitor;
  visitor.begin_object = [&] { stack.push_back({Json::object(), {}}); };
  visitor.begin_array = [&] { stack.push_back({Json::array(), {}}); };
  visitor.key = [&](std::string_view key) { need(!stack.empty(), "materializer key outside object"); stack.back().key = std::string(key); };
  visitor.scalar = [&](const Json &value) { attach(value); };
  visitor.end = [&] { need(!stack.empty() && !stack.back().key, "materializer container incomplete");
    auto value = std::move(stack.back().value); stack.pop_back(); attach(std::move(value)); };
  (void)visit(visitor, limits);
  need(assigned && stack.empty(), "materialized subtree incomplete");
  return result;
}
Json NodeHandle::export_json(const Sink &sink, ValueLimits limits) const {
  need(static_cast<bool>(sink), "subtree JSON sink required");
  // Keep the final scalar/key/container bytes until full subtree verification.
  // Scalars are bounded by the existing scalar profile. Structural suffix is
  // one byte; this is not an aggregate output buffer.
  struct Frame { bool object; bool first = true; bool value_after_key = false; };
  std::vector<Frame> stack;
  std::string held;
  auto emit = [&](std::string bytes) { if (!held.empty()) sink(held); held = std::move(bytes); };
  auto prefix = [&] {
    if (stack.empty()) return;
    auto &frame = stack.back();
    if (frame.object) { need(frame.value_after_key, "subtree export key missing"); frame.value_after_key = false; }
    else { if (!frame.first) emit(","); frame.first = false; }
  };
  ValueVisitor visitor;
  visitor.begin_object = [&] { prefix(); emit("{"); stack.push_back({true}); };
  visitor.begin_array = [&] { prefix(); emit("["); stack.push_back({false}); };
  visitor.key = [&](std::string_view key) { auto &frame = stack.back();
    if (!frame.first) emit(","); frame.first = false; emit(Json(std::string(key)).dump()); emit(":"); frame.value_after_key = true; };
  visitor.scalar = [&](const Json &value) { prefix(); emit(value.dump()); };
  visitor.end = [&] { const bool object = stack.back().object; stack.pop_back(); emit(object ? "}" : "]"); };
  auto receipt = visit(visitor, limits);
  check(p_->engine->checkpoint);
  if (!held.empty()) sink(held);
  return receipt;
}
Json ResultWriter::append_subtree(const NodeHandle &source, ValueLimits limits) {
  return p_->guarded([&] {
    ValueVisitor visitor;
    visitor.begin_object = [&] { begin_object(); };
    visitor.begin_array = [&] { begin_array(); };
    visitor.key = [&](std::string_view name) { key(std::string(name)); };
    visitor.scalar = [&](const Json &value) { this->value(value); };
    visitor.end = [&] { end(); };
    try { return source.visit(visitor, limits); }
    catch (StoreError &error) { error.recovery = p_->recovery(); throw; }
  });
}
