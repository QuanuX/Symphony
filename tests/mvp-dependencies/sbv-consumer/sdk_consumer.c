/* Standalone C consumer of the installed C ABI; no C++ headers required. */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <symphony/sbv/sdk.h>
int main(int argc, char **argv) {
  if (symphony_sbv_sdk_abi_v1() != 1 ||
      strcmp(symphony_sbv_sdk_version_v1(), "0.33.0-dev") != 0)
    return 1;
  if (argc != 2)
    return 2;
  FILE *f = fopen(argv[1], "rb");
  if (!f)
    return 2;
  char *input = malloc(1048577);
  if (!input) {
    fclose(f);
    return 70;
  }
  size_t n = fread(input, 1, 1048577, f);
  fclose(f);
  char *output = NULL;
  size_t size = 0;
  int status = symphony_sbv_sdk_process_v1(input, n, &output, &size);
  free(input);
  if (output) {
    fwrite(output, 1, size, stdout);
    symphony_sbv_sdk_release_v1(output);
  }
  return status;
}
