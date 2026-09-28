#include <cstdlib>
#include <string>
#include <symphony/sqav/news.hpp>
using namespace symphony;
void check(bool b) {
  if (!b)
    std::abort();
}
void installed_boundary() {
  std::string s =
      R"({"schema":"sqav-news-ingress-v1","provider":"fixture","publisher":"publisher","article":"article","revision":"v1","supersedes":"","published_at":"2026-09-28T12:00:00Z","updated_at":"2026-09-28T12:00:00Z","language":"en","rights_ref":"rights:fixture","source_uri":"https://example.test/a","headline":"Test","text":"Fixture only"})";
  sqav::news::Article a;
  auto bytes =
      source::Bytes{reinterpret_cast<const std::uint8_t *>(s.data()), s.size()};
  check(sqav::news::Article::admit(bytes, {4096, 128, 1024, 8}, nullptr, a) ==
        source::Status::ok);
  auto ref = std::string(a.reference());
  check(sqav::news::Article::admit(bytes.first(bytes.size() - 1),
                                   {4096, 128, 1024, 8}, nullptr,
                                   a) != source::Status::ok &&
        a.reference() == ref);
}
int main() { installed_boundary(); }
