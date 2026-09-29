#include <arpa/inet.h>
#include <array>
#include <chrono>
#include <cstdio>
#include <cstdlib>
#include <signal.h>
#include <symphony/source/support.hpp>
#include <sys/socket.h>
#include <sys/wait.h>
#include <thread>
#include <unistd.h>
namespace src = symphony::source;
namespace symphony::source::detail {
extern thread_local std::string endpoint_override;
}
void check(bool x, const char *why) {
  if (!x) {
    std::fprintf(stderr, "source HTTP: %s\n", why);
    std::abort();
  }
}
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
      if (request.find("GET /fixture") != 0 &&
          request.find("POST /fixture HTTP/") != 0)
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
struct Use : src::CredentialUse {
  bool fred = false, deny = false, swallow = false;
  unsigned calls = 0;
  src::Status with_secret(std::string_view ref, std::uint32_t ms,
                          std::stop_token, src::SecretSink &sink) override {
    check(!ref.empty() && ms > 0, "scope");
    ++calls;
    if (deny)
      return src::Status::not_authorized;
    std::array<std::uint8_t, 32> key;
    key.fill('x');
    if (!fred) {
      key[0] = 'd';
      key[1] = 'b';
      key[2] = '-';
    }
    auto s = sink.use(key);
    return swallow ? src::Status::ok : s;
  }
};
void run(unsigned mode) {
  std::string header = "HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\n";
  std::vector<std::uint8_t> body = {'{', '}'};
  if (mode == 1) {
    header = "HTTP/1.1 429 Rate limited\r\nContent-Length: 6\r\n\r\n";
    body = {'s', 'e', 'c', 'r', 'e', 't'};
  }
  if (mode == 2)
    body.resize(1);
  if (mode == 3)
    header =
        "HTTP/1.1 200 OK\r\nX-Large: " + std::string(2048, 'x') + "\r\n\r\n";
  if (mode == 4)
    header = "HTTP/1.1 200 OK\r\nContent-Encoding: gzip\r\n\r\n";
  if (mode == 7) {
    header = "HTTP/1.1 302 Found\r\nLocation: "
             "http://127.0.0.1:1/not-followed\r\nContent-Length: 0\r\n\r\n";
    body.clear();
  }
  if (mode == 8) {
    header = "HTTP/1.1 101 Switching Protocols\r\nConnection: "
             "Upgrade\r\nUpgrade: fixture\r\n\r\n";
    body.clear();
  }
  Server server(header, body, mode == 5 || mode == 6);
  src::detail::endpoint_override = server.endpoint;
  src::HttpRequest request{"https://example.test/fixture", "x=1", true,
                           src::HttpRequest::Credential::basic_username};
  Use use;
  use.swallow = mode == 2;
  if (mode == 9) {
    request.post = false;
    request.credential = src::HttpRequest::Credential::fred_query_key;
    use.fred = true;
  }
  src::HttpResponse response;
  std::stop_source stop;
  std::jthread cancel;
  if (mode == 6)
    cancel = std::jthread([&] {
      std::this_thread::sleep_for(std::chrono::milliseconds(20));
      stop.request_stop();
    });
  auto result =
      src::fetch(request, &use, {mode == 5 ? 50U : 1500U, 50, 1024, 1024},
                 stop.get_token(), response);
  check(use.calls == 1, "one key use");
  if (mode == 0 || mode == 9)
    check(result == src::Status::ok && response.complete &&
              response.body == body,
          "success");
  if (mode == 1)
    check(result == src::Status::http_error && response.complete &&
              response.http_status == 429 && response.body.empty(),
          "error body discarded");
  if (mode == 2)
    check(result == src::Status::transport_error && !response.complete,
          "swallowed truncation remains incomplete");
  if (mode == 3)
    check(result == src::Status::limit && !response.complete, "header bound");
  if (mode == 4)
    check(result == src::Status::unsupported && !response.complete,
          "encoding refused");
  if (mode == 5)
    check(result == src::Status::timeout && !response.complete, "deadline");
  if (mode == 6)
    check(result == src::Status::cancelled && !response.complete, "stop");
  if (mode == 7)
    check(result == src::Status::http_error && response.http_status == 302,
          "redirect refused");
  if (mode == 8)
    check(result == src::Status::transport_error && !response.complete,
          "upgrade refused");
}
int main() {
  ::signal(SIGPIPE, SIG_IGN);
  for (unsigned i = 0; i < 10; ++i)
    run(i);
  src::HttpRequest request{"https://example.test/fixture", "", false,
                           src::HttpRequest::Credential::fred_query_key};
  Use deny;
  deny.deny = true;
  src::HttpResponse out;
  check(src::fetch(request, &deny, {100, 50, 1024, 1024}, {}, out) ==
                src::Status::not_authorized &&
            !out.complete,
        "no credential no request");
  std::puts("source HTTP: ten loopback cases, FRED/basic credentials, bounds, "
            "timeout/cancel, denial, redirect and upgrade refusal passed");
}
