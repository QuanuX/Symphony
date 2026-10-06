#ifndef SYMPHONY_SBV_SDK_V1_H
#define SYMPHONY_SBV_SDK_V1_H
#include <stddef.h>
#include <stdint.h>
#if defined(_WIN32)
#if defined(SBV_SDK_BUILD)
#define SBV_SDK_API __declspec(dllexport)
#else
#define SBV_SDK_API __declspec(dllimport)
#endif
#else
#define SBV_SDK_API __attribute__((visibility("default")))
#endif
#ifdef __cplusplus
#define SBV_SDK_NOEXCEPT noexcept
extern "C" {
#else
#define SBV_SDK_NOEXCEPT
#endif
/* This ABI transports exact engine-process.v1 JSON; C++26 owns calculations.
 * Caller buffers must be valid for their declared lengths. Request bytes are
 * borrowed only until return. On return the response is a separate allocation,
 * NUL-terminated for convenience; size excludes NUL. Release it exactly once
 * with this library's release function. Never free borrowed version strings.
 * Concurrent calls are allowed; caller-selected output paths still cannot be
 * replaced. No interpreter, vendor runtime, global callback or network is used.
 * Return codes match the native process. 64 means invalid ABI arguments and
 * 70 allocation/internal boundary failure, with null response and size zero.
 * Other failures return a structured response when allocation is possible.
 * Cancellation is deadline-based; foreign pointers cannot be validated here. */
SBV_SDK_API uint32_t symphony_sbv_sdk_abi_v1(void) SBV_SDK_NOEXCEPT;
SBV_SDK_API const char *symphony_sbv_sdk_version_v1(void) SBV_SDK_NOEXCEPT;
SBV_SDK_API int
symphony_sbv_sdk_process_v1(const char *request, size_t request_size,
                            char **response,
                            size_t *response_size) SBV_SDK_NOEXCEPT;
SBV_SDK_API void symphony_sbv_sdk_release_v1(void *response) SBV_SDK_NOEXCEPT;
#ifdef __cplusplus
}
#endif
#endif
