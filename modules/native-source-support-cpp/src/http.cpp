#include <algorithm>
#include <array>
#include <charconv>
#include <chrono>
#include <curl/curl.h>
#include <limits>
#include <memory>
#include <mutex>
#include <symphony/source/support.hpp>
#include <unistd.h>
namespace symphony::source {
#ifdef SYMPHONY_SOURCE_HTTP_TESTING
namespace detail {
thread_local std::string endpoint_override;
}
#endif
namespace {
using Clock = std::chrono::steady_clock;
const auto original_pid = ::getpid();
struct Wipe {
  std::string value;
  ~Wipe() {
    volatile char *p = value.data();
    for (std::size_t i = 0; i < value.size(); ++i)
      p[i] = 0;
  }
};
bool endpoint_valid(std::string_view s) {
  if (!s.starts_with("https://") || s.size() > 2048)
    return false;
  s.remove_prefix(8);
  auto slash = s.find('/');
  if (slash == s.npos || slash == 0)
    return false;
  auto host = s.substr(0, slash);
  auto colon = host.find(':');
  if (colon != host.npos) {
    unsigned port = 0;
    auto v = host.substr(colon + 1);
    auto [end, ec] = std::from_chars(v.data(), v.data() + v.size(), port);
    if (ec != std::errc{} || end != v.data() + v.size() || port == 0 ||
        port > 65535)
      return false;
    host = host.substr(0, colon);
  }
  if (host.empty() || !std::ranges::all_of(host, [](unsigned char c) {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
               (c >= '0' && c <= '9') || c == '.' || c == '-';
      }))
    return false;
  return std::ranges::all_of(s.substr(slash), [](unsigned char c) {
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
           (c >= '0' && c <= '9') || c == '/' || c == '.' || c == '_' ||
           c == '-' || c == '%';
  });
}
bool request_valid(const HttpRequest &r) {
  if (!endpoint_valid(r.endpoint) || r.parameters.size() > 32768 ||
      static_cast<unsigned>(r.credential) > 2)
    return false;
  if (r.credential == HttpRequest::Credential::fred_query_key && r.post)
    return false;
  return std::ranges::all_of(r.parameters, [](unsigned char c) {
    return c >= 0x21 && c <= 0x7e && c != '#';
  });
}
struct Fetch final : SecretSink {
  const HttpRequest &r;
  HttpLimits limits;
  std::stop_token stop;
  Clock::time_point deadline;
  HttpResponse response;
  std::uint32_t header_bytes = 0;
  bool used = false, ready = false, encoded = false;
  Status actual = Status::not_authorized, failure = Status::ok;
  Fetch(const HttpRequest &request, HttpLimits l, std::stop_token s)
      : r(request), limits(l), stop(s),
        deadline(Clock::now() + std::chrono::milliseconds(l.timeout_ms)) {}
  static size_t head(char *p, size_t a, size_t b, void *ctx) noexcept {
    auto &s = *static_cast<Fetch *>(ctx);
    if (a && b > std::numeric_limits<size_t>::max() / a) {
      s.failure = Status::limit;
      return 0;
    }
    auto n = a * b;
    if (n > s.limits.max_header_bytes - s.header_bytes) {
      s.failure = Status::limit;
      return 0;
    }
    s.header_bytes += static_cast<std::uint32_t>(n);
    std::string_view line(p, n);
    if (!line.ends_with("\r\n")) {
      s.failure = Status::malformed;
      return 0;
    }
    line.remove_suffix(2);
    if (line.starts_with("HTTP/")) {
      if (s.ready) {
        s.failure = Status::malformed;
        return 0;
      }
      auto pos = line.find(' ');
      unsigned code = 0;
      if (pos == line.npos || line.size() < pos + 4) {
        s.failure = Status::malformed;
        return 0;
      }
      auto [end, ec] =
          std::from_chars(line.data() + pos + 1, line.data() + pos + 4, code);
      if (ec != std::errc{} || end != line.data() + pos + 4 || code < 100 ||
          code > 599 || (line.size() > pos + 4 && line[pos + 4] != ' ')) {
        s.failure = Status::malformed;
        return 0;
      }
      s.response.http_status = static_cast<std::uint16_t>(code);
      s.encoded = false;
      return n;
    }
    if (line.empty()) {
      if (s.response.http_status < 100) {
        s.failure = Status::malformed;
        return 0;
      }
      if (s.response.http_status >= 200)
        s.ready = true;
      return n;
    }
    if (s.ready) {
      s.failure = Status::malformed;
      return 0;
    }
    auto colon = line.find(':');
    if (colon == 0 || colon == line.npos) {
      s.failure = Status::malformed;
      return 0;
    }
    auto name = line.substr(0, colon);
    auto eq = [](std::string_view a, std::string_view b) {
      if (a.size() != b.size())
        return false;
      for (std::size_t i = 0; i < a.size(); ++i) {
        char c = a[i];
        if (c >= 'A' && c <= 'Z')
          c = char(c - 'A' + 'a');
        if (c != b[i])
          return false;
      }
      return true;
    };
    if (eq(name, "content-encoding")) {
      auto v = line.substr(colon + 1);
      while (!v.empty() && (v.front() == ' ' || v.front() == '\t'))
        v.remove_prefix(1);
      while (!v.empty() && (v.back() == ' ' || v.back() == '\t'))
        v.remove_suffix(1);
      if (s.encoded || !eq(v, "identity")) {
        s.failure = Status::unsupported;
        return 0;
      }
      s.encoded = true;
    }
    return n;
  }
  static size_t body(char *p, size_t a, size_t b, void *ctx) noexcept {
    auto &s = *static_cast<Fetch *>(ctx);
    if (a && b > std::numeric_limits<size_t>::max() / a) {
      s.failure = Status::limit;
      return 0;
    }
    auto n = a * b;
    if (!s.ready) {
      s.failure = Status::malformed;
      return 0;
    }
    if (n > s.limits.max_body_bytes - s.received) {
      s.failure = Status::limit;
      return 0;
    }
    s.received += static_cast<std::uint32_t>(n);
    if (s.response.http_status < 200 || s.response.http_status >= 300)
      return n;
    try {
      s.response.body.insert(s.response.body.end(),
                             reinterpret_cast<std::uint8_t *>(p),
                             reinterpret_cast<std::uint8_t *>(p) + n);
      return n;
    } catch (const std::bad_alloc &) {
      s.failure = Status::no_memory;
      return 0;
    } catch (...) {
      s.failure = Status::internal_error;
      return 0;
    }
  }
  std::uint32_t received = 0;
  static int progress(void *p, curl_off_t, curl_off_t, curl_off_t,
                      curl_off_t) noexcept {
    auto &s = *static_cast<Fetch *>(p);
    return s.stop.stop_requested() || Clock::now() >= s.deadline;
  }
  Status use(Bytes secret) override {
    if (used) {
      failure = Status::not_authorized;
      return failure;
    }
    used = true;
    try {
      actual = perform(secret);
    } catch (const std::bad_alloc &) {
      actual = Status::no_memory;
    } catch (...) {
      actual = Status::internal_error;
    }
    return actual;
  }
  Status perform(Bytes secret) {
    if (stop.stop_requested())
      return Status::cancelled;
    if (Clock::now() >= deadline)
      return Status::timeout;
    if (r.credential != HttpRequest::Credential::none) {
      if (secret.size() != 32)
        return Status::not_authorized;
      if (r.credential == HttpRequest::Credential::basic_username &&
          (secret[0] != 'd' || secret[1] != 'b' || secret[2] != '-'))
        return Status::not_authorized;
      for (std::size_t i =
               r.credential == HttpRequest::Credential::basic_username ? 3 : 0;
           i < secret.size(); ++i) {
        auto c = secret[i];
        if (!((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') ||
              (r.credential == HttpRequest::Credential::basic_username &&
               c >= 'A' && c <= 'Z')))
          return Status::not_authorized;
      }
    } else if (!secret.empty())
      return Status::not_authorized;
    static std::once_flag once;
    static CURLcode init = CURLE_FAILED_INIT;
    std::call_once(once, [] { init = curl_global_init(CURL_GLOBAL_DEFAULT); });
    if (init != CURLE_OK)
      return Status::internal_error;
    auto curl = std::unique_ptr<CURL, decltype(&curl_easy_cleanup)>(
        curl_easy_init(), curl_easy_cleanup);
    if (!curl)
      return Status::no_memory;
    Wipe key, url;
    if (!secret.empty())
      key.value.assign(reinterpret_cast<const char *>(secret.data()),
                       secret.size());
    url.value = r.endpoint;
    const char *protocols = "https";
#ifdef SYMPHONY_SOURCE_HTTP_TESTING
    if (!detail::endpoint_override.empty()) {
      url.value = detail::endpoint_override;
      protocols = "http";
    }
#endif
    if (!r.post && !r.parameters.empty())
      url.value += '?' + r.parameters;
    if (r.credential == HttpRequest::Credential::fred_query_key) {
      url.value += (r.parameters.empty() ? "?" : "&");
      url.value += "api_key=";
      url.value += key.value;
    }
    auto remaining = std::chrono::duration_cast<std::chrono::milliseconds>(
                         deadline - Clock::now())
                         .count();
    if (remaining <= 0)
      return Status::timeout;
    auto set = [&](auto option, auto value) {
      return curl_easy_setopt(curl.get(), option, value) == CURLE_OK;
    };
    if (!set(CURLOPT_URL, url.value.c_str()) ||
        !set(CURLOPT_PROTOCOLS_STR, protocols) ||
        !set(CURLOPT_FOLLOWLOCATION, 0L) || !set(CURLOPT_PROXY, "") ||
        !set(CURLOPT_NETRC, long(CURL_NETRC_IGNORED)) ||
        !set(CURLOPT_VERBOSE, 0L) || !set(CURLOPT_SSL_VERIFYPEER, 1L) ||
        !set(CURLOPT_SSL_VERIFYHOST, 2L) ||
        !set(CURLOPT_SSLVERSION, long(CURL_SSLVERSION_TLSv1_2)) ||
        !set(CURLOPT_NOSIGNAL, 1L) ||
        !set(CURLOPT_TIMEOUT_MS, long(remaining)) ||
        !set(CURLOPT_CONNECTTIMEOUT_MS,
             std::min(long(remaining), long(limits.connect_timeout_ms))) ||
        !set(CURLOPT_HEADERFUNCTION, &head) || !set(CURLOPT_HEADERDATA, this) ||
        !set(CURLOPT_WRITEFUNCTION, &body) || !set(CURLOPT_WRITEDATA, this) ||
        !set(CURLOPT_NOPROGRESS, 0L) ||
        !set(CURLOPT_XFERINFOFUNCTION, &progress) ||
        !set(CURLOPT_XFERINFODATA, this))
      return Status::internal_error;
    if (r.post && (!set(CURLOPT_POST, 1L) ||
                   !set(CURLOPT_POSTFIELDSIZE_LARGE,
                        static_cast<curl_off_t>(r.parameters.size())) ||
                   !set(CURLOPT_POSTFIELDS, r.parameters.data())))
      return Status::internal_error;
    if (r.credential == HttpRequest::Credential::basic_username &&
        (!set(CURLOPT_HTTPAUTH, long(CURLAUTH_BASIC)) ||
         !set(CURLOPT_USERNAME, key.value.c_str()) ||
         !set(CURLOPT_PASSWORD, "")))
      return Status::internal_error;
    auto code = curl_easy_perform(curl.get());
    if (stop.stop_requested())
      return Status::cancelled;
    if (code == CURLE_OPERATION_TIMEDOUT || Clock::now() >= deadline)
      return Status::timeout;
    if (failure != Status::ok)
      return failure;
    if (code != CURLE_OK || !ready)
      return Status::transport_error;
    response.complete = true;
    return response.http_status >= 200 && response.http_status < 300
               ? Status::ok
               : Status::http_error;
  }
};
} // namespace
Status fetch(const HttpRequest &r, CredentialUse *provider, const HttpLimits &l,
             std::stop_token stop, HttpResponse &out) noexcept {
  if (::getpid() != original_pid)
    return Status::stale;
  if (!request_valid(r) || l.timeout_ms == 0 || l.timeout_ms > 300000 ||
      l.connect_timeout_ms == 0 || l.connect_timeout_ms > l.timeout_ms ||
      l.max_header_bytes == 0 || l.max_header_bytes > 65536 ||
      l.max_body_bytes == 0 || l.max_body_bytes > (64U << 20))
    return Status::invalid_argument;
  if ((r.credential == HttpRequest::Credential::none) != (provider == nullptr))
    return Status::invalid_argument;
  if (stop.stop_requested())
    return Status::cancelled;
  try {
    Fetch sink(r, l, stop);
    Status s = Status::internal_error;
    try {
      s = provider
              ? provider->with_secret(r.reference(), l.timeout_ms, stop, sink)
              : sink.use({});
    } catch (const std::bad_alloc &) {
      s = Status::no_memory;
    } catch (...) {
      s = Status::internal_error;
    }
    if (s == Status::ok && (!sink.used || sink.actual != Status::ok))
      s = sink.actual;
    if (sink.failure != Status::ok)
      s = sink.failure;
    if (stop.stop_requested())
      s = Status::cancelled;
    else if (Clock::now() >= sink.deadline)
      s = Status::timeout;
    if (s != Status::ok && s != Status::http_error)
      sink.response.complete = false;
    out = std::move(sink.response);
    return s;
  } catch (const std::bad_alloc &) {
    return Status::no_memory;
  } catch (...) {
    return Status::internal_error;
  }
}
} // namespace symphony::source
