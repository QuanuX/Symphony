#ifndef SYMPHONY_SQAV_NEWS_HPP
#define SYMPHONY_SQAV_NEWS_HPP
#include <memory>
#include <symphony/source/support.hpp>
#include <symphony/sqav/capture.hpp>
namespace symphony::sqav::news {
using Status = source::Status;
inline constexpr std::string_view schema = "sqav-news-ingress-v1";
namespace detail {
struct ArticleState;
}
struct Fields {
  std::string provider, publisher, article, revision, supersedes, published_at,
      updated_at, language, rights_ref, source_uri, headline, text;
};
// Independently callable API ingress. This is an explicit Symphony news schema,
// not a claim to implement an unselected vendor's wire protocol or HTTP server.
// A correction must supply the actual predecessor and its exact capture
// identity.
class Article {
  std::shared_ptr<const detail::ArticleState> state_;

public:
  [[nodiscard]] static Status admit(source::Bytes, const source::JsonLimits &,
                                    const Article *predecessor,
                                    Article &) noexcept;
  [[nodiscard]] explicit operator bool() const noexcept { return bool(state_); }
  [[nodiscard]] const Fields &fields() const noexcept;
  [[nodiscard]] source::Bytes original() const noexcept;
  [[nodiscard]] std::string_view reference() const noexcept;
  [[nodiscard]] Status capture(std::string_view attempt,
                               std::string_view attribution,
                               std::string_view access_scope,
                               const TimeEvidence &acquisition,
                               const sqav::Limits &, Capture &) const noexcept;
};
} // namespace symphony::sqav::news
#endif
