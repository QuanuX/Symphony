#include "installed_campaign.hpp"
using namespace scv_test;
int main(int argc, char **argv) {
  return nt::test_main([&] {
    nt::Arguments args(argc, argv);
    Campaign campaign(args);
    NT_REQUIRE(campaign.connector_version == "0.1.0-dev" ||
               campaign.connector_version == "0.2.0-dev");
    campaign.checked_run(
        [&] { campaign.test_installed_graph_index_recovery_and_retrieval(); });
    campaign.print();
  });
}
