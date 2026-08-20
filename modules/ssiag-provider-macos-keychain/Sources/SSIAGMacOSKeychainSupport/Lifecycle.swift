import Darwin
import Foundation

public enum InstallScope: String, Codable, Sendable {
    case user
    case system
}

public struct InstallLayout: Sendable {
    public let prefix: URL
    public let binary: URL
    public let bundle: URL
    public let bundleBinary: URL
    public let receipt: URL

    public init(prefix: URL, binary: URL, receipt: URL, bundle: URL? = nil, bundleBinary: URL? = nil) {
        self.prefix = prefix
        self.binary = binary
        self.bundle = bundle ?? prefix.appending(path: providerBundleRelativePath(version: providerVersion), directoryHint: .isDirectory)
        self.bundleBinary = bundleBinary ?? prefix.appending(path: providerBundleExecutableRelativePath(version: providerVersion))
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
    case unreceiptedBundleEntry(String)
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
        case let .unreceiptedBundleEntry(path): "immutable package contains unreceipted bundle entry \(path)"
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
        if let sourceBundle = sourceBundleRoot(for: source) {
            return try installBundle(sourceExecutable: source, sourceBundle: sourceBundle, scope: scope, layout: layout)
        }
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
            return record(layout, executable: layout.binary, scope: scope, evidence: installed, changed: false)
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
        return record(layout, executable: layout.binary, scope: scope, evidence: evidence, changed: true)
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
            if manager.fileExists(atPath: layout.binary.path) || manager.fileExists(atPath: layout.bundle.path) {
                throw LifecycleError.unreceiptedBinary
            }
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
        let (evidence, executable) = try validateInstalledReceipt(receiptData, scope: scope, layout: layout)
        let result = record(layout, executable: executable, scope: scope, evidence: evidence, changed: true)
        if executable == layout.bundleBinary {
            try rejectUnreceiptedBundleEntries(layout.bundle, evidence: evidence, prefix: layout.prefix)
        }
        // Remove exact receipt-owned files deepest-first. Missing owned files
        // are the only partial state a receipt-last interrupted uninstall may
        // leave, so retry completes deterministically.
        for file in evidence.files.sorted(by: { $0.path > $1.path }) {
            let target = layout.prefix.appending(path: file.path)
            guard manager.fileExists(atPath: target.path) else { continue }
            let current = try safeRegularFileEvidence(target)
            guard current.size == file.size, current.digest == file.digest else {
                throw LifecycleError.changedBinary
            }
            try manager.removeItem(at: target)
            try synchronizeDirectory(target.deletingLastPathComponent())
        }
        // The receipt is removed last. If the process stopped after removing
        // the binary, retrying reaches this same receipt-validated path and
        // completes the interrupted uninstall without treating the missing
        // receipt-owned binary as tampering.
        try manager.removeItem(at: layout.receipt)
        try synchronizeDirectory(layout.receipt.deletingLastPathComponent())
        if manager.fileExists(atPath: layout.bundle.path) {
            // Unknown files and links were rejected above; only now remove the
            // empty receipt-owned directory skeleton.
            try manager.removeItem(at: layout.bundle)
            try synchronizeDirectory(layout.bundle.deletingLastPathComponent())
        }
        removeEmptyDirectory(layout.binary.deletingLastPathComponent())
        removeEmptyDirectory(layout.receipt.deletingLastPathComponent())
        return result
    }

    private static func record(_ layout: InstallLayout, executable: URL, scope: InstallScope, evidence: ReceiptEvidence, changed: Bool) -> InstallRecord {
        InstallRecord(
            protocolVersion: "symphony.ssiag.provider-package-result.v1",
            scope: scope,
            prefix: layout.prefix.path,
            version: providerVersion,
            binary: executable.path,
            receipt: layout.receipt.path,
            binarySHA256: evidence.executableDigest,
            receiptDigest: evidence.receiptDigest,
            changed: changed
        )
    }

    private static func installBundle(
        sourceExecutable: URL,
        sourceBundle: URL,
        scope: InstallScope,
        layout: InstallLayout
    ) throws -> InstallRecord {
        let sourceFiles = try bundleSourceEvidence(sourceBundle, executable: sourceExecutable)
        let manager = FileManager.default
        if manager.fileExists(atPath: layout.receipt.path) {
            let data = try protectedReceiptData(layout.receipt)
            let evidence = try validateProviderReceipt(
                data: data, scope: scope, version: providerVersion,
                architecture: runtimeArchitectureForReceipt(), expectedExecutable: layout.bundleBinary,
                checkInstalledBytes: true
            )
            guard evidence.files == sourceFiles.map(\.evidence) else { throw LifecycleError.immutableVersion }
            try rejectUnreceiptedBundleEntries(layout.bundle, evidence: evidence, prefix: layout.prefix)
            return record(layout, executable: layout.bundleBinary, scope: scope, evidence: evidence, changed: false)
        }
        if manager.fileExists(atPath: layout.binary.path) || manager.fileExists(atPath: layout.bundle.path) {
            throw LifecycleError.unreceiptedBinary
        }
        try ensureDirectory(layout.bundle)
        for sourceFile in sourceFiles {
            let destination = layout.prefix.appending(path: sourceFile.evidence.path)
            try ensureDirectory(destination.deletingLastPathComponent())
            try installExactBytes(try Data(contentsOf: sourceFile.url), to: destination, mode: sourceFile.mode)
            let installed = try safeRegularFileEvidence(destination)
            guard installed.size == sourceFile.evidence.size, installed.digest == sourceFile.evidence.digest else {
                throw LifecycleError.writeFailed
            }
        }
        try ensureDirectory(layout.receipt.deletingLastPathComponent())
        let (_, receiptData, receiptDigest) = try makeProviderBundleReceipt(
            scope: scope, version: providerVersion, architecture: runtimeArchitectureForReceipt(),
            files: sourceFiles.map(\.evidence)
        )
        try installExactBytes(receiptData, to: layout.receipt, mode: 0o600)
        let evidence = try validateProviderReceipt(
            data: protectedReceiptData(layout.receipt), scope: scope, version: providerVersion,
            architecture: runtimeArchitectureForReceipt(), expectedExecutable: layout.bundleBinary,
            checkInstalledBytes: true
        )
        guard evidence.receiptDigest == receiptDigest else { throw LifecycleError.invalidReceipt }
        return record(layout, executable: layout.bundleBinary, scope: scope, evidence: evidence, changed: true)
    }
}

private func validateLayout(_ layout: InstallLayout, scope: InstallScope) throws {
    let canonicalPrefix = layout.prefix.standardizedFileURL
    guard canonicalPrefix.path.hasPrefix("/"), canonicalPrefix.path != "/",
          layout.binary.standardizedFileURL.path == canonicalPrefix.appending(path: providerBinaryRelativePath(version: providerVersion)).path,
          layout.bundle.standardizedFileURL.path == canonicalPrefix.appending(path: providerBundleRelativePath(version: providerVersion)).path,
          layout.bundleBinary.standardizedFileURL.path == canonicalPrefix.appending(path: providerBundleExecutableRelativePath(version: providerVersion)).path,
          layout.receipt.standardizedFileURL.path == canonicalPrefix.appending(path: providerReceiptRelativePath(version: providerVersion)).path else {
        throw LifecycleError.unsafePath
    }
    _ = scope
}

private struct BundleSourceFile {
    let url: URL
    let mode: mode_t
    let evidence: ReceiptFileEvidence
}

private func sourceBundleRoot(for executable: URL) -> URL? {
    let macOS = executable.standardizedFileURL.deletingLastPathComponent()
    let contents = macOS.deletingLastPathComponent()
    let bundle = contents.deletingLastPathComponent()
    guard macOS.lastPathComponent == "MacOS", contents.lastPathComponent == "Contents",
          bundle.lastPathComponent.hasSuffix(".app") else { return nil }
    return bundle
}

private func bundleSourceEvidence(_ bundle: URL, executable: URL) throws -> [BundleSourceFile] {
    guard executable.lastPathComponent == providerExecutableName else { throw LifecycleError.unsafePath }
    let allowedSuffixes: [String: String] = [
        "Contents/Info.plist": "regular",
        "Contents/MacOS/\(providerExecutableName)": "executable",
        "Contents/Resources/ssiag-signing-policy.json": "regular",
        "Contents/_CodeSignature/CodeResources": "regular",
        "Contents/embedded.provisionprofile": "regular",
    ]
    guard let enumerator = FileManager.default.enumerator(
        at: bundle, includingPropertiesForKeys: nil, options: [], errorHandler: { _, _ in false }
    ) else { throw LifecycleError.writeFailed }
    var result = [BundleSourceFile]()
    while let value = enumerator.nextObject() as? URL {
        let components = value.pathComponents
        guard let bundleIndex = components.lastIndex(where: { $0.hasSuffix(".app") }), bundleIndex + 1 < components.count else {
            throw LifecycleError.unsafePath
        }
        let relative = components[(bundleIndex + 1)...].joined(separator: "/")
        var metadata = stat()
        guard lstat(value.path, &metadata) == 0 else { throw LifecycleError.sourceNotRegular }
        let shape = metadata.st_mode & S_IFMT
        guard shape != S_IFLNK else { throw LifecycleError.destinationNotRegular }
        if shape == S_IFDIR { continue }
        guard shape == S_IFREG, let kind = allowedSuffixes[relative] else {
            throw LifecycleError.unreceiptedBundleEntry(relative)
        }
        let safe = try safeRegularFileEvidence(value)
        let mode: mode_t = kind == "executable" ? 0o500 : 0o400
        let path = "\(providerBundleRelativePath(version: providerVersion))/\(relative)"
        result.append(BundleSourceFile(
            url: value, mode: mode,
            evidence: ReceiptFileEvidence(path: path, kind: kind, size: safe.size, digest: safe.digest)
        ))
    }
    result.sort { $0.evidence.path < $1.evidence.path }
    let paths = Set(result.map(\.evidence.path))
    guard paths.contains("\(providerBundleRelativePath(version: providerVersion))/Contents/Info.plist"),
          paths.contains(providerBundleExecutableRelativePath(version: providerVersion)) else {
        throw LifecycleError.invalidReceipt
    }
    return result
}

private func validateInstalledReceipt(_ data: Data, scope: InstallScope, layout: InstallLayout) throws -> (ReceiptEvidence, URL) {
    if let bundle = try? validateProviderReceipt(
        data: data, scope: scope, version: providerVersion, architecture: runtimeArchitectureForReceipt(),
        expectedExecutable: layout.bundleBinary, checkInstalledBytes: false
    ) {
        return (bundle, layout.bundleBinary)
    }
    return (try validateProviderReceipt(
        data: data, scope: scope, version: providerVersion, architecture: runtimeArchitectureForReceipt(),
        expectedExecutable: layout.binary, checkInstalledBytes: false
    ), layout.binary)
}

private func rejectUnreceiptedBundleEntries(_ bundle: URL, evidence: ReceiptEvidence, prefix: URL) throws {
    guard FileManager.default.fileExists(atPath: bundle.path) else { return }
    let owned = Set(evidence.files.map(\.path))
    guard let enumerator = FileManager.default.enumerator(
        at: bundle, includingPropertiesForKeys: [.isRegularFileKey, .isDirectoryKey, .isSymbolicLinkKey],
        options: [], errorHandler: { _, _ in false }
    ) else { throw LifecycleError.destinationNotRegular }
    while let value = enumerator.nextObject() as? URL {
        let values = try value.resourceValues(forKeys: [.isRegularFileKey, .isDirectoryKey, .isSymbolicLinkKey])
        guard values.isSymbolicLink != true else { throw LifecycleError.unreceiptedBinary }
        if values.isDirectory == true { continue }
        let components = value.pathComponents
        guard let marker = components.lastIndex(of: "libexec"), marker < components.count else {
            throw LifecycleError.unsafePath
        }
        let relative = components[marker...].joined(separator: "/")
        guard values.isRegularFile == true, owned.contains(relative) else { throw LifecycleError.unreceiptedBinary }
    }
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
