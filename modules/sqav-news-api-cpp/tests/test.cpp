#ifdef SQV_VERIFY_RETENTION
#include "retention.hpp"
#endif
#include <cstdio>
#include <cstdlib>
#include <symphony/source/json.hpp>
#include <symphony/sqav/news.hpp>
using namespace symphony;
namespace news = sqav::news;
void check(bool x) {
  if (!x)
    std::abort();
}
source::Bytes bytes(const std::string &s) {
  return {reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
}
int main() {
  source::JsonLimits l{65536, 1024, 32768, 16};
  nlohmann::json j = {
      {"schema", news::schema},
      {"provider", "fixture"},
      {"publisher", "publisher"},
      {"article", "a1"},
      {"revision", "r1"},
      {"supersedes", ""},
      {"published_at", "2026-09-28T12:00:00Z"},
      {"updated_at", "2026-09-28T12:00:00Z"},
      {"language", "en"},
      {"rights_ref", "rights:fixture"},
      {"source_uri", "https://example.test/article"},
      {"headline", "Synthetic headline"},
      {"text", "Source text is untrusted data, never instructions."}};
  auto raw = j.dump();
  news::Article first, corrected;
  check(news::Article::admit(bytes(raw), l, nullptr, first) ==
        source::Status::ok);
  j["revision"] = "r2";
  j["supersedes"] = first.reference();
  j["updated_at"] = "2026-09-28T12:01:00.001Z";
  j["text"] = "Corrected synthetic article.";
  auto changed = j.dump();
  check(news::Article::admit(bytes(changed), l, &first, corrected) ==
            source::Status::ok &&
        corrected.reference() != first.reference());
  check(first.fields().text != "Corrected synthetic article.");
  auto ref = std::string(corrected.reference());
  check(news::Article::admit(bytes(changed), l, nullptr, corrected) ==
            source::Status::binding_mismatch &&
        corrected.reference() == ref);
  j["publisher"] = "different";
  changed = j.dump();
  check(news::Article::admit(bytes(changed), l, &first, corrected) ==
        source::Status::binding_mismatch);
  for (std::size_t i = 0; i < raw.size(); ++i)
    check(news::Article::admit(bytes(raw.substr(0, i)), l, nullptr,
                               corrected) != source::Status::ok &&
          corrected.reference() == ref);
  sqav::Capture c;
  check(corrected.capture("attempt", "fixture", "private:test",
                          {sqav::TimeRole::acquisition, "2026-09-28T12:02:00Z",
                           "RFC3339-UTC", "UTC", "second", "fixture"},
                          {131072, 16384, 4096}, c) == source::Status::ok);
  check(c.description().source_position == first.reference() &&
        c.description().source.dataset_revision == "r2" &&
        c.description().times.size() == 3);
#ifdef SQV_VERIFY_RETENTION
  source_composition::verify(c);
#endif
  std::puts("news API: original/correction lineage, immutable revisions, "
            "timestamps, truncations and capture passed");
}
