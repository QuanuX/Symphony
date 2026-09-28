#include <algorithm>
#include <symphony/source/json.hpp>
#include <symphony/sqav/news.hpp>
namespace symphony::sqav::news {
namespace detail {
struct ArticleState {
  Fields fields;
  std::vector<std::uint8_t> bytes;
  std::string reference;
};
} // namespace detail
namespace {
const Fields empty{};
bool utc(std::string_view s) {
  if (s.size() < 20 || s.size() > 30 || !source::date(s.substr(0, 10)) ||
      s[10] != 'T' || s[13] != ':' || s[16] != ':' || s.back() != 'Z')
    return false;
  auto pair = [&](std::size_t p, unsigned max) {
    return s[p] >= '0' && s[p] <= '9' && s[p + 1] >= '0' && s[p + 1] <= '9' &&
           unsigned(s[p] - '0') * 10 + unsigned(s[p + 1] - '0') <= max;
  };
  if (!pair(11, 23) || !pair(14, 59) || !pair(17, 59))
    return false;
  if (s.size() == 20)
    return true;
  if (s[19] != '.' || s.size() < 22)
    return false;
  return std::ranges::all_of(s.substr(20, s.size() - 21),
                             [](char c) { return c >= '0' && c <= '9'; });
}
std::string order(std::string_view s) {
  std::string o(s.substr(0, 19));
  o += '.';
  if (s.size() > 20)
    o += s.substr(20, s.size() - 21);
  o.resize(29, '0');
  return o;
}
} // namespace
Status Article::admit(source::Bytes bytes, const source::JsonLimits &limits,
                      const Article *prior, Article &out) noexcept {
  nlohmann::json j;
  auto s = source::parse_json(bytes, limits, j);
  if (s != Status::ok)
    return s;
  try {
    if (j.at("schema") != schema || !j.is_object() || j.size() != 13)
      return Status::unsupported;
    auto read = [&](const char *name) { return j.at(name).get<std::string>(); };
    auto state = std::make_shared<detail::ArticleState>();
    auto &f = state->fields;
    f = {read("provider"),   read("publisher"),  read("article"),
         read("revision"),   read("supersedes"), read("published_at"),
         read("updated_at"), read("language"),   read("rights_ref"),
         read("source_uri"), read("headline"),   read("text")};
    if (!source::token(f.provider) || !source::token(f.publisher) ||
        !source::token(f.article, 256) || !source::token(f.revision) ||
        !source::token(f.language, 64) || !source::token(f.rights_ref, 256) ||
        f.headline.empty() || f.headline.size() > 1024 || f.text.empty() ||
        f.source_uri.size() > 2048 || !f.source_uri.starts_with("https://") ||
        !utc(f.published_at) || !utc(f.updated_at) ||
        order(f.published_at) > order(f.updated_at))
      return Status::malformed;
    if (f.supersedes.empty()) {
      if (prior)
        return Status::binding_mismatch;
    } else {
      if (!prior || !*prior || f.supersedes != prior->reference())
        return Status::binding_mismatch;
      const auto &p = prior->fields();
      if (f.provider != p.provider || f.publisher != p.publisher ||
          f.article != p.article || f.revision == p.revision ||
          f.published_at != p.published_at ||
          order(f.updated_at) < order(p.updated_at))
        return Status::binding_mismatch;
    }
    state->bytes.assign(bytes.begin(), bytes.end());
    state->reference =
        "news-article-v1-" +
        source::digest(std::string_view(
            reinterpret_cast<const char *>(bytes.data()), bytes.size()));
    out.state_ = std::move(state);
    return Status::ok;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (const nlohmann::json::exception &) {
    return Status::malformed;
  } catch (...) {
    return Status::internal_error;
  }
}
const Fields &Article::fields() const noexcept {
  return state_ ? state_->fields : empty;
}
source::Bytes Article::original() const noexcept {
  return state_ ? source::Bytes(state_->bytes) : source::Bytes{};
}
std::string_view Article::reference() const noexcept {
  return state_ ? state_->reference : std::string_view{};
}
Status Article::capture(std::string_view attempt, std::string_view attribution,
                        std::string_view scope, const TimeEvidence &acq,
                        const sqav::Limits &limits,
                        Capture &out) const noexcept {
  if (!state_ || acq.role != TimeRole::acquisition)
    return Status::invalid_argument;
  try {
    const auto &f = state_->fields;
    std::string selection = f.provider + ":" + f.publisher + ":" + f.article;
    auto precision = [](const std::string &t) {
      return t.size() == 20 ? std::string("second")
                            : "decimal-second-" + std::to_string(t.size() - 21);
    };
    Description d{{f.provider, "symphony-news-api", "1", "article.ingress",
                   "sqav-news-api-cpp", "0.1.0-dev", selection, f.revision,
                   selection, std::string(schema), "utf8-json",
                   std::string(scope)},
                  std::string(attempt),
                  std::string(attribution),
                  f.supersedes,
                  Coverage::complete,
                  "one-article-revision",
                  state_->reference,
                  1,
                  {acq,
                   {TimeRole::publication, f.published_at, "RFC3339-UTC", "UTC",
                    precision(f.published_at), state_->reference},
                   {TimeRole::revision, f.updated_at, "RFC3339-UTC", "UTC",
                    precision(f.updated_at), state_->reference}}};
    auto s = Capture::create(d, state_->bytes, limits, out);
    return s == sqav::Status::ok          ? Status::ok
           : s == sqav::Status::no_memory ? Status::no_memory
           : s == sqav::Status::limit     ? Status::limit
                                          : Status::invalid_argument;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
} // namespace symphony::sqav::news
