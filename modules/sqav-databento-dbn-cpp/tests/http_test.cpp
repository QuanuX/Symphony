#include "public_fixture.hpp"
#include <arpa/inet.h>
#include <array>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <filesystem>
#include <signal.h>
#include <symphony/sqav/databento/http.hpp>
#include <sys/socket.h>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>
namespace db = symphony::sqav::databento;
namespace symphony::sqav::databento::detail {
extern thread_local std::string test_endpoint;
}
void check(bool b, const char *why) {
  if (!b) {
    std::fprintf(stderr, "HTTP fixture: %s\n", why);
    std::abort();
  }
}
struct Root {
  std::string path;
  Root() {
    char name[] = "/private/tmp/sqv-http-XXXXXX";
    auto p = ::mkdtemp(name);
    check(p, "root");
    path = p;
  }
  ~Root() { std::filesystem::remove_all(path); }
};
struct FixtureUse final : db::SsiagHistoricalUse {
  unsigned calls = 0;
  bool refuse = false, swallow = false;
  std::string expected;
  db::HttpStatus with_key(std::string_view ref, std::uint32_t duration,
                          std::stop_token,
                          db::HistoricalKeySink &sink) noexcept override {
    ++calls;
    check(ref == expected && duration > 0, "exact broker binding");
    if (refuse)
      return db::HttpStatus::not_authorized;
    // Fabricated fixture bytes; never an actual provider credential.
    std::array<std::uint8_t, 32> key{};
    key.fill('x');
    key[0] = 'd';
    key[1] = 'b';
    key[2] = '-';
    const auto result = sink.use(key);
    return swallow ? db::HttpStatus::ok : result;
  }
};
struct Server {
  int socket = -1;
  pid_t pid = -1;
  std::string endpoint;
  Server(std::string headers, std::vector<std::uint8_t> body,
         bool stall = false) {
    socket = ::socket(AF_INET, SOCK_STREAM, 0);
    check(socket >= 0, "socket");
    sockaddr_in address{};
    address.sin_family = AF_INET;
    address.sin_addr.s_addr = htonl(INADDR_LOOPBACK);
    check(::bind(socket, reinterpret_cast<sockaddr *>(&address),
                 sizeof(address)) == 0 &&
              ::listen(socket, 1) == 0,
          "listen");
    socklen_t length = sizeof(address);
    check(::getsockname(socket, reinterpret_cast<sockaddr *>(&address),
                        &length) == 0,
          "port");
    endpoint = "http://127.0.0.1:" + std::to_string(ntohs(address.sin_port)) +
               "/fixture";
    pid = ::fork();
    check(pid >= 0, "fork");
    if (pid == 0) {
      int peer = ::accept(socket, nullptr, nullptr);
      if (peer < 0)
        _exit(2);
      std::string request;
      std::array<char, 4096> data{};
      while (request.find("\r\n\r\n") == std::string::npos &&
             request.size() < 65536) {
        auto n = ::read(peer, data.data(), data.size());
        if (n <= 0)
          _exit(3);
        request.append(data.data(), static_cast<std::size_t>(n));
      }
      if (request.find("POST /fixture HTTP/") != 0 ||
          request.find("Authorization: Basic ") == std::string::npos)
        _exit(4);
      if (stall) {
        std::this_thread::sleep_for(std::chrono::seconds(2));
        ::close(peer);
        _exit(0);
      }
      auto send = [&](const void *p, std::size_t n) {
        auto bytes = static_cast<const char *>(p);
        while (n) {
          auto k = ::write(peer, bytes, n);
          if (k <= 0)
            break;
          bytes += k;
          n -= static_cast<std::size_t>(k);
        }
      };
      send(headers.data(), headers.size());
      send(body.data(), body.size());
      ::close(peer);
      _exit(0);
    }
    ::close(socket);
    socket = -1;
  }
  ~Server() {
    if (pid > 0) {
      ::kill(pid, SIGKILL);
      int status;
      ::waitpid(pid, &status, 0);
    }
  }
};
db::HistoricalPlan plan() {
  auto b = public_fixture::fixture();
  db::FileView file;
  check(db::FileView::inspect(b, {65536, 4096, 100}, file) == db::Status::ok,
        "fixture");
  const auto &m = file.metadata();
  db::HistoricalPlan p;
  check(db::HistoricalPlan::create(
            {std::string(m.dataset), {"ESH1"}, m.start, m.end, m.limit},
            {{65536, 4096, 100}, 86'400'000'000'000ULL, 8, 3, 60},
            p) == db::Status::ok,
        "plan");
  return p;
}
void run_case(unsigned mode) {
  Root root;
  auto p = plan();
  db::AttemptBudget budget{{}, 10000, 0, 8, 1U << 20};
  budget.generation[0] = 1;
  db::AttemptLedger ledger;
  check(db::AttemptLedger::create(root.path, budget, ledger) ==
            db::AttemptStatus::ok,
        "ledger");
  db::AttemptTicket ticket;
  check(ledger.reserve(p, "fixture", {"fixture:quote", 100, 100, 1000}, 200,
                       ticket) == db::AttemptStatus::ok,
        "reserve");
  auto body = public_fixture::fixture();
  std::string headers =
      "HTTP/1.1 200 OK\r\nContent-Length: " + std::to_string(body.size()) +
      "\r\n\r\n";
  if (mode == 1) {
    headers =
        "HTTP/1.1 429 Limited\r\nRetry-After: 12\r\nContent-Length: 5\r\n\r\n";
    body = {'e', 'r', 'r', 'o', 'r'};
  }
  if (mode == 2)
    body.resize(body.size() - 56); // valid DBN prefix, incomplete HTTP body
  if (mode == 3)
    headers =
        "HTTP/1.1 200 OK\r\nX-Large: " + std::string(2048, 'x') + "\r\n\r\n";
  if (mode == 4)
    headers = "HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\n\r\n";
  if (mode == 7) {
    headers = "HTTP/1.1 429 Limited\r\nRetry-After: Wed, 21 Oct 2037 07:28:00 GMT\r\nContent-Length: 0\r\n\r\n";
    body.clear();
  }
  if (mode == 8) {
    headers = "HTTP/1.1 302 Found\r\nLocation: http://127.0.0.1:1/should-not-follow\r\nContent-Length: 0\r\n\r\n";
    body.clear();
  }
  if (mode == 9) {
    headers = "HTTP/1.1 101 Switching Protocols\r\nConnection: Upgrade\r\nUpgrade: fixture\r\n\r\n";
    body.clear();
  }
  Server server(headers, body, mode == 5 || mode == 6);
  db::detail::test_endpoint = server.endpoint;
  FixtureUse use;
  use.expected = p.reference();
  use.swallow = mode == 2;
  db::HistoricalResponse response;
  std::stop_source cancellation;
  std::jthread cancel;
  if (mode == 6)
    cancel = std::jthread([&] {
      std::this_thread::sleep_for(std::chrono::milliseconds(20));
      cancellation.request_stop();
    });
  const auto result = db::execute_historical(
      p, ledger, ticket, use, {mode == 5 ? 50U : 1500U, 50, 1024}, 200,
      cancellation.get_token(), response);
  check(use.calls == 1 && result.persistence == db::AttemptStatus::ok,
        "one callback and durable outcome");
  if (mode == 0)
    check(result.status == db::HttpStatus::ok &&
              result.response.coverage == symphony::sqav::Coverage::partial &&
              response.body().size() == body.size(),
          "HTTP completed cap");
  if (mode == 1)
    check(result.status == db::HttpStatus::ok &&
              result.response.http_status == 429 &&
              result.response.retry_after_seconds == 12 &&
              response.body().empty(),
          "HTTP failure body discarded");
  if (mode == 2)
    check(result.status == db::HttpStatus::transport_error &&
              result.response.coverage == symphony::sqav::Coverage::gap,
          "prefix not completion despite broker return");
  if (mode == 3)
    check(result.status == db::HttpStatus::response_limit, "header budget");
  if (mode == 4)
    check(result.status == db::HttpStatus::malformed_headers,
          "compression refused");
  if (mode == 5)
    check(result.status == db::HttpStatus::timeout, "deadline");
  if (mode == 6)
    check(result.status == db::HttpStatus::cancelled, "cancellation");
  if (mode == 7) check(result.status == db::HttpStatus::ok && result.retry_after_uninterpreted &&
      !result.response.retry_after_seconds && result.response.recovery == db::Recovery::review,
      "unparsed HTTP-date requires review without an invented delay");
  if (mode == 8) check(result.status == db::HttpStatus::ok && result.response.http_status == 302,
      "redirect not followed");
  if (mode == 9) check(result.status == db::HttpStatus::transport_error && response.body().empty(),
      "upgrade is not a final historical HTTP response");
  const auto repeated = db::execute_historical(
      p, ledger, ticket, use, {100, 50, 1024}, 200, {}, response);
  check(repeated.status == db::HttpStatus::attempt_refused && use.calls == 1,
        "no repeated network");
}
void denied_and_pre_cancelled() {
  Root root;
  auto p = plan();
  db::AttemptBudget b{{}, 1000, 0, 4, 1U << 20};
  b.generation[0] = 1;
  db::AttemptLedger l;
  check(db::AttemptLedger::create(root.path, b, l) == db::AttemptStatus::ok,
        "denied ledger");
  db::AttemptTicket t;
  check(l.reserve(p, "denied", {"fixture:quote", 100, 100, 1000}, 200, t) ==
            db::AttemptStatus::ok,
        "denied reserve");
  FixtureUse use;
  use.expected = p.reference();
  use.refuse = true;
  db::HistoricalResponse out;
  auto result =
      db::execute_historical(p, l, t, use, {100, 50, 1024}, 200, {}, out);
  check(result.status == db::HttpStatus::not_authorized &&
            result.persistence == db::AttemptStatus::ok && out.body().empty(),
        "refused before connection");
  std::stop_source stop;
  stop.request_stop();
  result = db::execute_historical(p, l, t, use, {100, 50, 1024}, 200,
                                  stop.get_token(), out);
  check(result.status == db::HttpStatus::cancelled && use.calls == 1,
        "pre-cancel no provider");
}
int main() {
  ::signal(SIGPIPE, SIG_IGN);
  denied_and_pre_cancelled();
  for (unsigned i = 0; i < 10; ++i)
    run_case(i);
  std::puts(
      "HTTP: loopback framing, status, truncation, headers, compression, "
      "timeout, cancellation, broker refusal and one-shot execution passed");
}
