#include <symphony/ssiag/private_channel.hpp>
int main() {
    using namespace symphony::ssiag::native;
    if (hash("abc") != "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad") return 1;
    if (parse("{\"ready\":true}").at("ready") != true) return 2;
    bool refused = false;
    try { (void)parse("{\"ready\":true,\"ready\":false}"); } catch (...) { refused = true; }
    if (!refused) return 3;
    Secret secret;
    secret.bytes = "fixture";
    wipe(secret.bytes);
    return secret.bytes.empty() ? 0 : 4;
}
