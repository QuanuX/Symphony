import Darwin
import Foundation

public enum InstallScope: String, Codable, Sendable {
    case user
    case system
}

public struct InstallLayout: Sendable {
    public let prefix: URL
    public let binary: URL
    public let receipt: URL

    public init(prefix: URL, binary: URL, receipt: URL) {
        self.prefix = prefix
        self.binary = binary
        self.receipt = receipt
    }

    public static func resolve(_ scope: InstallScope, prefix explicitPrefix: URL? = nil) throws -> InstallLayout {
        let prefix: URL
        if let explicitPrefix {
            prefix = explicitPrefix.standardizedFileURL
        } else {
            switch scope {
            case .user:
                prefix = FileManager.default.homeDirectoryForCurrentUser.appending(path: ".local", directoryHint: .isDirectory)
            case .system:
                prefix = URL(fileURLWithPath: "/usr/local", isDirectory: true)
            }
        }
        guard prefix.path.hasPrefix("/"), prefix.path != "/" else { throw LifecycleError.unsafePath }
        let binary = prefix.appending(path: providerBinaryRelativePath(version: providerVersion))
        let receipt = prefix.appending(path: providerReceiptRelativePath(version: providerVersion))
        return InstallLayout(prefix: prefix, binary: binary, receipt: receipt)
    }
}

public struct InstallRecord: Codable, Sendable {
    public let protocolVersion: String
    public let scope: InstallScope
    public let prefix: String
    public let version: String
    public let binary: String
    public let receipt: String
    public let binarySHA256: String
    public let receiptDigest: String
    public let changed: Bool

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case scope, prefix, version, binary, receipt, changed
        case binarySHA256 = "binary_sha256"
        case receiptDigest = "receipt_digest"
    }
}

public enum LifecycleError: Error, CustomStringConvertible {
    case unsafePath
    case sourceNotRegular
    case destinationNotRegular
    case changedBinary
    case invalidReceipt
    case unreceiptedBinary
    case immutableVersion
    case writeFailed

    public var description: String {
        switch self {
        case .unsafePath: "unsafe installation path"
        case .sourceNotRegular: "source executable is not a protected regular file"
        case .destinationNotRegular: "refusing non-regular installation path"
        case .changedBinary: "receipt-owned executable digest changed"
        case .invalidReceipt: "install receipt v2 is invalid"
        case .unreceiptedBinary: "immutable package path contains unreceipted bytes"
        case .immutableVersion: "immutable package version already contains different bytes"
        case .writeFailed: "atomic package write failed"
        }
    }
}

public enum ProviderLifecycle {
    public static func install(
        source: URL,
        scope: InstallScope,
        force _: Bool = false,
        layout explicitLayout: InstallLayout? = nil
    ) throws -> InstallRecord {
        let manager = FileManager.default
        let layout = try explicitLayout ?? InstallLayout.resolve(scope)
        try validateLayout(layout, scope: scope)
        let sourceEvidence: (size: UInt64, digest: String, metadata: SafeFileMetadata)
        do {
            sourceEvidence = try safeRegularFileEvidence(source)
        } catch {
            throw LifecycleError.sourceNotRegular
        }

        if manager.fileExists(atPath: layout.receipt.path) {
            let receiptData = try protectedReceiptData(layout.receipt)
            let installed = try validateProviderReceipt(
                data: receiptData,
                scope: scope,
                version: providerVersion,
                architecture: runtimeArchitectureForReceipt(),
                expectedExecutable: layout.binary,
                checkInstalledBytes: true
            )
            guard installed.executableDigest == sourceEvidence.digest,
                  installed.executableSize == sourceEvidence.size else {
                throw LifecycleError.immutableVersion
            }
            return record(layout, scope: scope, evidence: installed, changed: false)
        }

        try ensureDirectory(layout.binary.deletingLastPathComponent())
        try ensureDirectory(layout.receipt.deletingLastPathComponent())
        if manager.fileExists(atPath: layout.binary.path) {
            let installed: (size: UInt64, digest: String, metadata: SafeFileMetadata)
            do {
                installed = try safeRegularFileEvidence(layout.binary)
            } catch {
                throw LifecycleError.destinationNotRegular
            }
            guard installed.size == sourceEvidence.size, installed.digest == sourceEvidence.digest else {
                throw LifecycleError.unreceiptedBinary
            }
        } else {
            try installExactBytes(try Data(contentsOf: source, options: []), to: layout.binary, mode: 0o755)
            let installed = try safeRegularFileEvidence(layout.binary)
            guard installed.size == sourceEvidence.size, installed.digest == sourceEvidence.digest else {
                throw LifecycleError.writeFailed
            }
        }

        let (_, receiptData, receiptDigest) = try makeProviderReceipt(
            scope: scope,
            version: providerVersion,
            architecture: runtimeArchitectureForReceipt(),
            executableSize: sourceEvidence.size,
            executableDigest: sourceEvidence.digest
        )
        try installExactBytes(receiptData, to: layout.receipt, mode: 0o600)
        let evidence = try validateProviderReceipt(
            data: protectedReceiptData(layout.receipt),
            scope: scope,
            version: providerVersion,
            architecture: runtimeArchitectureForReceipt(),
            expectedExecutable: layout.binary,
            checkInstalledBytes: true
        )
        guard evidence.receiptDigest == receiptDigest else { throw LifecycleError.invalidReceipt }
        return record(layout, scope: scope, evidence: evidence, changed: true)
    }

    public static func uninstall(
        scope: InstallScope,
        force _: Bool = false,
        layout explicitLayout: InstallLayout? = nil
    ) throws -> InstallRecord {
        let manager = FileManager.default
        let layout = try explicitLayout ?? InstallLayout.resolve(scope)
        try validateLayout(layout, scope: scope)
        guard manager.fileExists(atPath: layout.receipt.path) else {
            if manager.fileExists(atPath: layout.binary.path) { throw LifecycleError.unreceiptedBinary }
            return InstallRecord(
                protocolVersion: "symphony.ssiag.provider-package-result.v1",
                scope: scope,
                prefix: layout.prefix.path,
                version: providerVersion,
                binary: layout.binary.path,
                receipt: layout.receipt.path,
                binarySHA256: "not_applicable",
                receiptDigest: "not_applicable",
                changed: false
            )
        }
        let receiptData = try protectedReceiptData(layout.receipt)
        let evidence = try validateProviderReceipt(
            data: receiptData,
            scope: scope,
            version: providerVersion,
            architecture: runtimeArchitectureForReceipt(),
            expectedExecutable: layout.binary,
            checkInstalledBytes: false
        )
        let result = record(layout, scope: scope, evidence: evidence, changed: true)
        if manager.fileExists(atPath: layout.binary.path) {
            _ = try validateProviderReceipt(
                data: receiptData,
                scope: scope,
                version: providerVersion,
                architecture: runtimeArchitectureForReceipt(),
                expectedExecutable: layout.binary,
                checkInstalledBytes: true
            )
            try manager.removeItem(at: layout.binary)
            try synchronizeDirectory(layout.binary.deletingLastPathComponent())
        }
        // The receipt is removed last. If the process stopped after removing
        // the binary, retrying reaches this same receipt-validated path and
        // completes the interrupted uninstall without treating the missing
        // receipt-owned binary as tampering.
        try manager.removeItem(at: layout.receipt)
        try synchronizeDirectory(layout.receipt.deletingLastPathComponent())
        removeEmptyDirectory(layout.binary.deletingLastPathComponent())
        removeEmptyDirectory(layout.receipt.deletingLastPathComponent())
        return result
    }

    private static func record(_ layout: InstallLayout, scope: InstallScope, evidence: ReceiptEvidence, changed: Bool) -> InstallRecord {
        InstallRecord(
            protocolVersion: "symphony.ssiag.provider-package-result.v1",
            scope: scope,
            prefix: layout.prefix.path,
            version: providerVersion,
            binary: layout.binary.path,
            receipt: layout.receipt.path,
            binarySHA256: evidence.executableDigest,
            receiptDigest: evidence.receiptDigest,
            changed: changed
        )
    }
}

private func validateLayout(_ layout: InstallLayout, scope: InstallScope) throws {
    let canonicalPrefix = layout.prefix.standardizedFileURL
    guard canonicalPrefix.path.hasPrefix("/"), canonicalPrefix.path != "/",
          layout.binary.standardizedFileURL.path == canonicalPrefix.appending(path: providerBinaryRelativePath(version: providerVersion)).path,
          layout.receipt.standardizedFileURL.path == canonicalPrefix.appending(path: providerReceiptRelativePath(version: providerVersion)).path else {
        throw LifecycleError.unsafePath
    }
    _ = scope
}

private func protectedReceiptData(_ url: URL) throws -> Data {
    let metadata = try safeRegularFileMetadata(url)
    guard metadata.size <= 1_048_576 else { throw LifecycleError.invalidReceipt }
    return try Data(contentsOf: url, options: [])
}

private func ensureDirectory(_ url: URL) throws {
    let manager = FileManager.default
    let path = url.standardizedFileURL
    let parent = path.deletingLastPathComponent()
    if path.path != "/" && parent.path != path.path { try ensureDirectory(parent) }
    if symbolicLink(path) {
        guard permittedSystemAlias(path) else { throw LifecycleError.unsafePath }
        return
    }
    if manager.fileExists(atPath: path.path) {
        let values = try path.resourceValues(forKeys: [.isDirectoryKey, .isSymbolicLinkKey])
        guard values.isDirectory == true, values.isSymbolicLink != true else { throw LifecycleError.unsafePath }
        return
    }
    try manager.createDirectory(at: path, withIntermediateDirectories: false, attributes: [.posixPermissions: 0o700])
}

private func installExactBytes(_ data: Data, to destination: URL, mode: mode_t) throws {
    let temporary = destination.deletingLastPathComponent().appending(path: ".ssiag-provider-\(UUID().uuidString)")
    defer { unlink(temporary.path) }
    try createExclusive(data, at: temporary, mode: mode)
    guard link(temporary.path, destination.path) == 0 else {
        if errno == EEXIST { throw LifecycleError.immutableVersion }
        throw LifecycleError.writeFailed
    }
    try synchronizeDirectory(destination.deletingLastPathComponent())
}

private func createExclusive(_ data: Data, at url: URL, mode: mode_t) throws {
    let descriptor = open(url.path, O_WRONLY | O_CREAT | O_EXCL | O_NOFOLLOW, mode)
    guard descriptor >= 0 else {
        if errno == EEXIST { throw LifecycleError.immutableVersion }
        throw LifecycleError.writeFailed
    }
    var success = false
    defer {
        close(descriptor)
        if !success { unlink(url.path) }
    }
    try data.withUnsafeBytes { raw in
        var offset = 0
        while offset < raw.count {
            let count = Darwin.write(descriptor, raw.baseAddress!.advanced(by: offset), raw.count - offset)
            guard count > 0 else { throw LifecycleError.writeFailed }
            offset += count
        }
    }
    guard fsync(descriptor) == 0 else { throw LifecycleError.writeFailed }
    success = true
    try synchronizeDirectory(url.deletingLastPathComponent())
}

private func synchronizeDirectory(_ url: URL) throws {
    let descriptor = open(url.path, O_RDONLY | O_DIRECTORY | O_CLOEXEC)
    guard descriptor >= 0 else { throw LifecycleError.writeFailed }
    defer { close(descriptor) }
    guard fsync(descriptor) == 0 else { throw LifecycleError.writeFailed }
}

private func symbolicLink(_ url: URL) -> Bool {
    (try? FileManager.default.destinationOfSymbolicLink(atPath: url.path)) != nil
}

private func permittedSystemAlias(_ url: URL) -> Bool {
    let expected = ["/var": "private/var", "/tmp": "private/tmp", "/etc": "private/etc"]
    guard let wanted = expected[url.path],
          let destination = try? FileManager.default.destinationOfSymbolicLink(atPath: url.path) else { return false }
    return destination == wanted
}

private func removeEmptyDirectory(_ url: URL) {
    guard let contents = try? FileManager.default.contentsOfDirectory(atPath: url.path), contents.isEmpty else { return }
    try? FileManager.default.removeItem(at: url)
}

private func runtimeArchitectureForReceipt() -> String {
#if arch(arm64)
    "arm64"
#elseif arch(x86_64)
    "amd64"
#else
    "unsupported"
#endif
}
