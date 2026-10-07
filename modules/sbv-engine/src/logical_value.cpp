#include "logical_value.hpp"
#include "streaming_sha256.hpp"
#include <limits>
#include <vector>

namespace symphony::sbv::logical {
namespace {
void need(bool b, const char *s) {
  if (!b)
    throw result_store::StoreError("logical.contract", s);
}
std::uint64_t add(std::uint64_t a, std::uint64_t b) {
  need(b <= UINT64_MAX - a, "logical counter overflow");
  return a + b;
}
std::uint64_t number(const Json &j) {
  return std::stoull(j.get<std::string>());
}
void check(const Checkpoint &c) {
  if (c)
    c();
}
std::string type(const Json &j) {
  if (j.is_object())
    return "object";
  if (j.is_array())
    return "array";
  if (j.is_string())
    return "string";
  if (j.is_boolean())
    return "boolean";
  if (j.is_null())
    return "null";
  throw result_store::StoreError("logical.contract",
                                 "portable numeric strings required");
}
} // namespace
struct Value::Impl {
  std::shared_ptr<const Json> owner;
  const Json *json = nullptr;
  std::optional<result_store::NodeHandle> node;
  std::optional<std::map<std::string, Value, std::less<>>> object;
  std::optional<std::vector<Value>> array;
};
Value::Value(std::shared_ptr<const Impl> p) : p_(std::move(p)) {}
Value::Value(Json j) {
  auto p = std::make_shared<Impl>();
  p->owner = std::make_shared<const Json>(std::move(j));
  p->json = p->owner.get();
  p_ = std::move(p);
}
Value::Value(result_store::NodeHandle h) {
  auto p = std::make_shared<Impl>();
  p->node.emplace(std::move(h));
  p_ = std::move(p);
}
Value Value::object(std::map<std::string, Value, std::less<>> o) {
  auto p = std::make_shared<Impl>();
  p->object.emplace(std::move(o));
  return Value(p);
}
Value Value::array(std::vector<Value> values) {
  auto p = std::make_shared<Impl>();
  p->array.emplace(std::move(values));
  return Value(p);
}
Value Value::independent_reader(result_store::ReadOptions options,
                                Checkpoint checkpoint) const {
  need(bool(p_), "moved logical value");
  if (p_->node) {
    const auto description = p_->node->describe(),
               reference = description.at("reference");
    result_store::ResultReader reader(
        {reference.at("manifest_path").get<std::string>(),
         reference.at("manifest_sha256").get<std::string>(),
         reference.at("content_sha256").get<std::string>()},
        options, std::move(checkpoint));
    return Value(reader.select_node(number(description.at("node_id"))));
  }
  if (p_->object) {
    std::map<std::string, Value, std::less<>> fields;
    for (const auto &[key, value] : *p_->object)
      fields.emplace(key, value.independent_reader(options, checkpoint));
    return object(std::move(fields));
  }
  if (p_->array) {
    std::vector<Value> rows;
    rows.reserve(p_->array->size());
    for (const auto &value : *p_->array)
      rows.push_back(value.independent_reader(options, checkpoint));
    return array(std::move(rows));
  }
  return *this;
}
std::string Value::kind() const {
  need(bool(p_), "moved logical value");
  return p_->node     ? p_->node->describe().at("kind").get<std::string>()
         : p_->object ? "object"
         : p_->array  ? "array"
         : p_->json->is_object() ? "object"
         : p_->json->is_array()  ? "array"
                                 : "scalar";
}
std::uint64_t Value::size() const {
  need(bool(p_), "moved logical value");
  if (p_->node)
    return number(p_->node->describe().at("children"));
  if (p_->object)
    return p_->object->size();
  if (p_->array)
    return p_->array->size();
  return p_->json->is_structured() ? p_->json->size() : 0;
}
Value Value::at(std::string_view key) const {
  need(kind() == "object", "logical member requires object");
  if (p_->node)
    return Value(p_->node->child(key));
  if (p_->object) {
    auto i = p_->object->find(key);
    need(i != p_->object->end(), "logical member absent");
    return i->second;
  }
  auto i = p_->json->find(std::string(key));
  need(i != p_->json->end(), "logical member absent");
  auto p = std::make_shared<Impl>();
  p->owner = p_->owner;
  p->json = &i.value();
  return Value(p);
}
Value Value::at(std::uint64_t index) const {
  need(kind() == "array" && index < size(),
       "logical array index outside range");
  if (p_->node)
    return Value(p_->node->child(index));
  if (p_->array)
    return p_->array->at(static_cast<std::size_t>(index));
  auto p = std::make_shared<Impl>();
  p->owner = p_->owner;
  p->json = &p_->json->at(static_cast<std::size_t>(index));
  return Value(p);
}
bool Value::contains(std::string_view key) const {
  need(kind() == "object", "logical contains requires object");
  if (p_->object)
    return p_->object->contains(key);
  if (p_->json)
    return p_->json->contains(std::string(key));
  auto c = children();
  while (auto v = c.next())
    if (v->edge == key)
      return true;
  return false;
}
Value Value::with(std::string key, Value replacement) const {
  need(kind() == "object", "logical graft requires object");
  std::map<std::string, Value, std::less<>> o;
  auto c = children();
  while (auto v = c.next())
    o.emplace(v->edge, std::move(v->value));
  o.insert_or_assign(std::move(key), std::move(replacement));
  return object(std::move(o));
}
Value Value::without(std::string_view key) const {
  need(kind() == "object", "logical removal requires object");
  std::map<std::string, Value, std::less<>> o;
  auto c = children();
  while (auto v = c.next())
    if (v->edge != key)
      o.emplace(v->edge, std::move(v->value));
  return object(std::move(o));
}
struct Cursor::Impl {
  Value parent;
  std::optional<result_store::ChildCursor> native;
  std::uint64_t next, end;
  Json::const_iterator json;
  std::map<std::string, Value, std::less<>>::const_iterator object;
  bool failed = false;
  Impl(Value v, std::uint64_t first, std::optional<std::uint64_t> count)
      : parent(std::move(v)), next(first), end(0) {
    need(parent.kind() == "object" || parent.kind() == "array",
         "logical cursor requires container");
    auto total = parent.size();
    need(first <= total, "logical cursor start out of range");
    need(!count || *count <= total - first,
         "logical cursor range out of bounds");
    end = count ? add(first, *count) : total;
    if (parent.p_->node)
      native.emplace(parent.p_->node->children(first, count));
    else if (parent.p_->object) {
      object = parent.p_->object->begin();
      std::advance(object, first);
    } else if (!parent.p_->array) {
      json = parent.p_->json->begin();
      std::advance(json, first);
    }
  }
};
Cursor::Cursor(std::unique_ptr<Impl> p) : p_(std::move(p)) {}
Cursor::~Cursor() = default;
Cursor::Cursor(Cursor &&) noexcept = default;
Cursor &Cursor::operator=(Cursor &&) noexcept = default;
Cursor Value::children(std::uint64_t first,
                       std::optional<std::uint64_t> count) const {
  return Cursor(std::make_unique<Cursor::Impl>(*this, first, count));
}
std::optional<Child> Cursor::next() {
  need(bool(p_) && !p_->failed, "moved or failed logical cursor");
  try {
    if (p_->next == p_->end)
      return {};
    if (p_->native) {
      auto n = p_->native->next();
      need(bool(n), "native cursor ended early");
      auto edge = n->describe().at("edge").get<std::string>();
      ++p_->next;
      return Child{std::move(edge), Value(std::move(*n))};
    }
    if (p_->parent.p_->object) {
      Child out{p_->object->first, p_->object->second};
      ++p_->object;
      ++p_->next;
      return out;
    }
    if (p_->parent.p_->array) {
      const auto index = p_->next++;
      return Child{std::to_string(index),
                   p_->parent.p_->array->at(static_cast<std::size_t>(index))};
    }
    const auto key = p_->parent.p_->json->is_object()
                         ? p_->json.key()
                         : std::to_string(p_->next);
    auto value = std::make_shared<Value::Impl>();
    value->owner = p_->parent.p_->owner;
    value->json = &p_->json.value();
    ++p_->json;
    ++p_->next;
    return Child{key, Value(value)};
  } catch (...) {
    p_->failed = true;
    throw;
  }
}
void Value::visit(const result_store::ValueVisitor &visitor,
                  Checkpoint checkpoint) const {
  check(checkpoint);
  if (p_->node) {
    auto wrapped = visitor;
    wrapped.begin_object = [&] {
      check(checkpoint);
      if (visitor.begin_object)
        visitor.begin_object();
    };
    wrapped.begin_array = [&] {
      check(checkpoint);
      if (visitor.begin_array)
        visitor.begin_array();
    };
    wrapped.end = [&] {
      check(checkpoint);
      if (visitor.end)
        visitor.end();
    };
    wrapped.key = [&](auto s) {
      check(checkpoint);
      if (visitor.key)
        visitor.key(s);
    };
    wrapped.scalar = [&](const Json &j) {
      check(checkpoint);
      if (visitor.scalar)
        visitor.scalar(j);
    };
    p_->node->visit(wrapped);
    return;
  }
  const auto k = kind();
  if (k == "object" || k == "array") {
    if (k == "object") {
      if (visitor.begin_object)
        visitor.begin_object();
    } else {
      if (visitor.begin_array)
        visitor.begin_array();
    }
    auto c = children();
    while (auto v = c.next()) {
      check(checkpoint);
      if (k == "object" && visitor.key)
        visitor.key(v->edge);
      v->value.visit(visitor, checkpoint);
    }
    if (visitor.end)
      visitor.end();
  } else if (visitor.scalar)
    visitor.scalar(*p_->json);
}
Json Value::export_json(const result_store::Sink &sink, Limits limits,
                        Checkpoint checkpoint) const {
  hash_detail::Sha256 hash;
  std::uint64_t bytes = 0, nodes = 0, values = 0;
  struct Frame {
    bool object, first = true, expecting_value = false;
    std::string prior;
  };
  std::vector<Frame> stack;
  bool root = false;
  std::optional<char> tail;
  auto emit = [&](std::string_view b) {
    bytes = add(bytes, b.size());
    need(!limits.max_canonical_bytes || bytes <= *limits.max_canonical_bytes,
         "logical byte budget exceeded");
    need(
        hash.update(reinterpret_cast<const std::uint8_t *>(b.data()), b.size()),
        "logical SHA256 length overflow");
    if (!b.empty()) {
      if (tail) {
        check(checkpoint);
        sink(std::string_view(&*tail, 1));
      }
      if (b.size() > 1) {
        check(checkpoint);
        sink(b.substr(0, b.size() - 1));
      }
      tail = b.back();
    }
  };
  auto count = [&](bool key) {
    nodes = add(nodes, 1);
    if (!key)
      values = add(values, 1);
    need(!limits.max_counted_nodes || nodes <= *limits.max_counted_nodes,
         "logical node budget exceeded");
  };
  auto prefix = [&] {
    count(false);
    need(stack.size() <= 64, "logical depth exceeds portable profile");
    if (stack.empty()) {
      need(!root, "multiple logical roots");
      root = true;
      return;
    }
    auto &f = stack.back();
    if (f.object) {
      need(f.expecting_value, "logical object value has no key");
      f.expecting_value = false;
    } else {
      if (!f.first)
        emit(",");
      f.first = false;
    }
  };
  result_store::ValueVisitor v;
  v.begin_object = [&] {
    prefix();
    emit("{");
    stack.push_back({true, true, false, {}});
  };
  v.begin_array = [&] {
    prefix();
    emit("[");
    stack.push_back({false, true, false, {}});
  };
  v.key = [&](std::string_view key) {
    need(!stack.empty() && stack.back().object, "logical key outside object");
    auto &f = stack.back();
    need(!f.expecting_value && (f.first || f.prior < key),
         "duplicate or unordered logical key");
    need(key.size() <= 65536, "logical key exceeds portable profile");
    count(true);
    if (!f.first)
      emit(",");
    emit(Json(key).dump());
    emit(":");
    f.first = false;
    f.expecting_value = true;
    f.prior = key;
  };
  v.scalar = [&](const Json &j) {
    need(!j.is_structured(), "logical scalar is structured");
    (void)type(j);
    need(!j.is_string() || j.get_ref<const std::string &>().size() <= 65536,
         "logical string exceeds portable profile");
    prefix();
    emit(j.dump());
  };
  v.end = [&] {
    need(!stack.empty() && !stack.back().expecting_value,
         "incomplete logical container");
    bool obj = stack.back().object;
    stack.pop_back();
    emit(obj ? "}" : "]");
  };
  visit(v, checkpoint);
  need(root && stack.empty(), "incomplete logical value");
  std::uint8_t digest[32];
  hash.finish(digest);
  std::string encoded;
  encoded.reserve(64);
  constexpr char hex[] = "0123456789abcdef";
  for (auto b : digest) {
    encoded.push_back(hex[b >> 4]);
    encoded.push_back(hex[b & 15]);
  }
  need(tail.has_value(), "empty logical encoding");
  check(checkpoint);
  sink(std::string_view(&*tail, 1));
  check(checkpoint);
  return {{"sha256", encoded},
          {"bytes", std::to_string(bytes)},
          {"nodes", std::to_string(nodes)},
          {"values", std::to_string(values)},
          {"verification_extent", "selected_value"}};
}
std::string Value::sha256(Limits limits, Checkpoint c) const {
  return export_json([](std::string_view) {}, limits, std::move(c))
      .at("sha256")
      .get<std::string>();
}
Json Value::materialize(Limits limits, Checkpoint c) const {
  std::string bytes;
  export_json([&](std::string_view b) { bytes.append(b); }, limits,
              std::move(c));
  return Json::parse(bytes);
}
void Value::write(result_store::ResultWriter &writer) const {
  if (p_->node) {
    writer.append_subtree(*p_->node);
    return;
  }
  if (kind() == "object" || kind() == "array") {
    const bool object = kind() == "object";
    if (object)
      writer.begin_object();
    else
      writer.begin_array();
    auto c = children();
    while (auto v = c.next()) {
      if (object)
        writer.key(v->edge);
      v->value.write(writer);
    }
    writer.end();
  } else
    writer.value(*p_->json);
}
} // namespace symphony::sbv::logical
