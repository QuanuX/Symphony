import Darwin
import Foundation

enum FoundationTrustFailure: Error {
    case unavailable
    case layout
    case unsafeFile
    case executable
    case receipt
    case pathMismatch
    case digestMismatch
    case signingUnsupported
}

struct FoundationObservation: Sendable {
    let verified: Bool
    let executablePath: String
    let installationDigest: String
    let executableDigest: String
    let signingIdentity: String
    let reasonCode: String
}

protocol FoundationInspecting {
    func inspect(_ request: ProviderControlRequest) -> FoundationObservation
}

struct LiveFoundationInspector: FoundationInspecting {
    func inspect(_ request: ProviderControlRequest) -> FoundationObservation {
        var observedPath = request.foundationExecutablePath
        var observedDigest = request.foundationExecutableDigest
        do {
            let parentPath = try parentExecutablePath()
            observedPath = parentPath.path
            guard parentPath.standardizedFileURL.path == URL(fileURLWithPath: request.foundationExecutablePath).standardizedFileURL.path else {
                throw FoundationTrustFailure.pathMismatch
            }
            let metadata = try safeRegularFileMetadata(parentPath)
            guard metadata.uid == geteuid() || metadata.uid == 0 else {
                throw FoundationTrustFailure.unsafeFile
            }
            observedDigest = try sha256Digest(of: parentPath)
            guard observedDigest == request.foundationExecutableDigest else {
                throw FoundationTrustFailure.digestMismatch
            }
            guard request.foundationInstallationDigest != "not_applicable" else {
                throw FoundationTrustFailure.receipt
            }
            let receipt = try validateFoundationReceipt(
                parentExecutable: parentPath,
                requestedInstallationDigest: request.foundationInstallationDigest
            )
            guard receipt.executableDigest == observedDigest else {
                throw FoundationTrustFailure.digestMismatch
            }

            // Code-signing policy is intentionally not inferred. The cgo-free
            // foundation can declare not_applicable while receipt-v2 and exact
            // executable digests provide mutual trust. A future ratified
            // signing identity requires an independently verifiable adapter
            // policy and cannot be accepted from this request alone.
            guard request.foundationSigningIdentity == "not_applicable" else {
                throw FoundationTrustFailure.signingUnsupported
            }
            return FoundationObservation(
                verified: true,
                // The independently observed path has already been proven to
                // name the same protected file. Echo the request's exact
                // canonical spelling so macOS /var -> /private/var aliases do
                // not create a false cross-language identity mismatch.
                executablePath: request.foundationExecutablePath,
                installationDigest: receipt.receiptDigest,
                executableDigest: observedDigest,
                signingIdentity: "not_applicable",
                reasonCode: "symphony.ssiag.provider.foundation_verified"
            )
        } catch {
            return FoundationObservation(
                verified: false,
                executablePath: observedPath,
                installationDigest: request.foundationInstallationDigest,
                executableDigest: observedDigest,
                signingIdentity: "not_applicable",
                reasonCode: reasonCode(for: error)
            )
        }
    }

    private func parentExecutablePath() throws -> URL {
        let parent = getppid()
        guard parent > 1 else { throw FoundationTrustFailure.unavailable }
        var buffer = [CChar](repeating: 0, count: 4096)
        let count = proc_pidpath(parent, &buffer, UInt32(buffer.count))
        guard count > 0, count < buffer.count else { throw FoundationTrustFailure.unavailable }
        let pathBytes = buffer.prefix(Int(count)).prefix { $0 != 0 }.map { UInt8(bitPattern: $0) }
        let path = String(decoding: pathBytes, as: UTF8.self)
        guard path.hasPrefix("/"), !path.contains("\0") else { throw FoundationTrustFailure.unavailable }
        guard let resolvedPointer = realpath(path, nil) else {
            throw FoundationTrustFailure.unavailable
        }
        defer { free(resolvedPointer) }
        let resolved = String(cString: resolvedPointer)
        guard resolved.hasPrefix("/") else { throw FoundationTrustFailure.unavailable }
        return URL(fileURLWithPath: resolved).standardizedFileURL
    }

    private func reasonCode(for error: Error) -> String {
        switch error {
        case FoundationTrustFailure.pathMismatch:
            "symphony.ssiag.provider.foundation_path_mismatch"
        case FoundationTrustFailure.digestMismatch, FoundationTrustFailure.executable:
            "symphony.ssiag.provider.foundation_executable_mismatch"
        case FoundationTrustFailure.unsafeFile:
            "symphony.ssiag.provider.foundation_file_unsafe"
        case FoundationTrustFailure.receipt, FoundationTrustFailure.layout:
            "symphony.ssiag.provider.foundation_receipt_invalid"
        case FoundationTrustFailure.signingUnsupported:
            "symphony.ssiag.provider.foundation_signing_unsupported"
        default:
            "symphony.ssiag.provider.foundation_unavailable"
        }
    }
}
