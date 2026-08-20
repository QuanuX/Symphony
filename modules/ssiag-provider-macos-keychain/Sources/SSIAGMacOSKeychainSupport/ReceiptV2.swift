import Darwin
import Foundation

let providerComponentID = "ssiag-macos-keychain-provider"
let providerEntryPointID = "ssiag.macos-keychain-provider"
let providerEntryPointProtocol = "symphony.ssiag.provider.control.v1"
let providerAdapterIdentifier = "adapter:symphony:ssiag.macos-keychain-provider.v1"
let providerExecutableName = "symphony-ssiag-provider-macos-keychain"
let providerBundleName = "SymphonySSIAGMacOSKeychainProvider.app"
let receiptProtocolV2 = "symphony.knowledge.install-receipt.v2"

struct ReceiptEvidence: Sendable {
    let receiptDigest: String
    let executableDigest: String
    let executableSize: UInt64
    let files: [ReceiptFileEvidence]
}

struct ReceiptFileEvidence: Equatable, Sendable {
    let path: String
    let kind: String
    let size: UInt64
    let digest: String
}

func providerBinaryRelativePath(version: String) -> String {
    "libexec/symphony/\(providerComponentID)/\(version)/\(providerExecutableName)"
}

func providerBundleRelativePath(version: String) -> String {
    "libexec/symphony/\(providerComponentID)/\(version)/\(providerBundleName)"
}

func providerBundleExecutableRelativePath(version: String) -> String {
    "\(providerBundleRelativePath(version: version))/Contents/MacOS/\(providerExecutableName)"
}

func providerReceiptRelativePath(version: String) -> String {
    "share/symphony/receipts/\(providerComponentID)/\(version)/install-receipt.json"
}

func makeProviderReceipt(
    scope: InstallScope,
    version: String,
    architecture: String,
    executableSize: UInt64,
    executableDigest: String
) throws -> ([String: Any], Data, String) {
    try makeProviderReceipt(
        version: version,
        architecture: architecture,
        files: [ReceiptFileEvidence(
            path: providerBinaryRelativePath(version: version),
            kind: "executable",
            size: executableSize,
            digest: executableDigest
        )],
        entryPointPath: providerBinaryRelativePath(version: version)
    )
}

func makeProviderBundleReceipt(
    scope _: InstallScope,
    version: String,
    architecture: String,
    files: [ReceiptFileEvidence]
) throws -> ([String: Any], Data, String) {
    try makeProviderReceipt(
        version: version,
        architecture: architecture,
        files: files,
        entryPointPath: providerBundleExecutableRelativePath(version: version)
    )
}

private func makeProviderReceipt(
    version: String,
    architecture: String,
    files: [ReceiptFileEvidence],
    entryPointPath: String
) throws -> ([String: Any], Data, String) {
    guard !files.isEmpty, files.count <= 4096,
          files.map(\.path) == files.map(\.path).sorted(),
          Set(files.map(\.path)).count == files.count,
          files.contains(where: { $0.path == entryPointPath && $0.kind == "executable" }) else {
        throw LifecycleError.invalidReceipt
    }
    var object: [String: Any] = [
        "protocol": receiptProtocolV2,
        "format_version": 2,
        "component_id": providerComponentID,
        "component_kind": "adapter",
        "module_id": providerComponentID,
        "vector_id": NSNull(),
        "engine_id": NSNull(),
        "package_id": providerComponentID,
        "version": version,
        "install_scope": "prefix",
        "prefix_mode": "installation_prefix",
        "files": files.map { file in
            [
                "path": file.path,
                "kind": file.kind,
                "size": NSNumber(value: file.size),
                "digest": file.digest,
            ] as [String: Any]
        },
        "entry_points": [[
            "entry_point_id": providerEntryPointID,
            "kind": "adapter",
            "path": entryPointPath,
            "protocols": [providerEntryPointProtocol],
        ]],
        "provides_capabilities": ["symphony.ssiag.provider.metadata.v1"],
        "requires_capabilities": [],
        "compatible_receptors": ["symphony.ssiag.provider-launcher.v1"],
        "platform_requirements": [[
            "os": "macos",
            "architecture": architecture,
            "kernel_abi": NSNull(),
            "critical": true,
        ]],
    ]
    let digest = try canonicalDigest(object, omitting: "receipt_digest")
    object["receipt_digest"] = digest
    var data = try canonicalJSONData(object)
    data.append(0x0a)
    return (object, data, digest)
}

func validateProviderReceipt(
    data: Data,
    scope: InstallScope,
    version: String,
    architecture: String,
    expectedExecutable: URL,
    checkInstalledBytes: Bool
) throws -> ReceiptEvidence {
    let object = try strictJSONObject(data)
    try requireExactKeys(object, [
        "protocol", "format_version", "component_id", "component_kind", "module_id",
        "vector_id", "engine_id", "package_id", "version", "install_scope", "prefix_mode",
        "files", "entry_points", "provides_capabilities", "requires_capabilities",
        "compatible_receptors", "platform_requirements", "receipt_digest",
    ])
    guard object["protocol"] as? String == receiptProtocolV2,
          integer(object["format_version"]) == 2,
          object["component_id"] as? String == providerComponentID,
          object["component_kind"] as? String == "adapter",
          object["module_id"] as? String == providerComponentID,
          object["vector_id"] is NSNull,
          object["engine_id"] is NSNull,
          object["package_id"] as? String == providerComponentID,
          object["version"] as? String == version,
          object["install_scope"] as? String == "prefix",
          object["prefix_mode"] as? String == "installation_prefix",
          strings(object["provides_capabilities"]) == ["symphony.ssiag.provider.metadata.v1"],
          strings(object["requires_capabilities"]) == [],
          strings(object["compatible_receptors"]) == ["symphony.ssiag.provider-launcher.v1"],
          let recordedDigest = object["receipt_digest"] as? String,
          validDigest(recordedDigest),
          try canonicalDigest(object, omitting: "receipt_digest") == recordedDigest else {
        throw LifecycleError.invalidReceipt
    }
    guard let fileValues = object["files"] as? [Any], !fileValues.isEmpty, fileValues.count <= 4096 else {
        throw LifecycleError.invalidReceipt
    }
    var files = [ReceiptFileEvidence]()
    var ownedPaths = Set<String>()
    for value in fileValues {
        guard let file = value as? [String: Any] else { throw LifecycleError.invalidReceipt }
        try requireExactKeys(file, ["path", "kind", "size", "digest"])
        guard let path = file["path"] as? String,
              validRelativeReceiptPath(path), ownedPaths.insert(path).inserted,
              let kind = file["kind"] as? String, ["regular", "executable"].contains(kind),
              let size = unsignedInteger(file["size"]), size > 0, size <= 64 * 1_048_576,
              let digest = file["digest"] as? String, validDigest(digest) else {
            throw LifecycleError.invalidReceipt
        }
        files.append(ReceiptFileEvidence(path: path, kind: kind, size: size, digest: digest))
    }
    guard files.map(\.path) == files.map(\.path).sorted() else { throw LifecycleError.invalidReceipt }

    let legacyPath = providerBinaryRelativePath(version: version)
    let bundlePath = providerBundleExecutableRelativePath(version: version)
    let expectedPath: String
    if expectedExecutable.path.hasSuffix("/\(legacyPath)") {
        expectedPath = legacyPath
        guard files.count == 1 else { throw LifecycleError.invalidReceipt }
    } else if expectedExecutable.path.hasSuffix("/\(bundlePath)") {
        expectedPath = bundlePath
        let allowed = providerAllowedBundleReceiptPaths(version: version)
        guard files.allSatisfy({ allowed.contains($0.path) }),
              files.contains(where: { $0.path.hasSuffix("/Contents/Info.plist") && $0.kind == "regular" }) else {
            throw LifecycleError.invalidReceipt
        }
    } else {
        throw LifecycleError.invalidReceipt
    }
    guard let executable = files.first(where: { $0.path == expectedPath }), executable.kind == "executable" else {
        throw LifecycleError.invalidReceipt
    }

    guard let entries = object["entry_points"] as? [Any], entries.count == 1,
          let entry = entries.first as? [String: Any] else { throw LifecycleError.invalidReceipt }
    try requireExactKeys(entry, ["entry_point_id", "kind", "path", "protocols"])
    guard entry["entry_point_id"] as? String == providerEntryPointID,
          entry["kind"] as? String == "adapter",
          entry["path"] as? String == expectedPath,
          strings(entry["protocols"]) == [providerEntryPointProtocol] else {
        throw LifecycleError.invalidReceipt
    }

    guard let platforms = object["platform_requirements"] as? [Any], platforms.count == 1,
          let platform = platforms.first as? [String: Any] else { throw LifecycleError.invalidReceipt }
    try requireExactKeys(platform, ["os", "architecture", "kernel_abi", "critical"])
    guard platform["os"] as? String == "macos",
          platform["architecture"] as? String == architecture,
          platform["kernel_abi"] is NSNull,
          strictBoolean(platform["critical"]) == true else { throw LifecycleError.invalidReceipt }

    if checkInstalledBytes {
        let prefix = try installationPrefix(for: expectedExecutable, relativePath: expectedPath)
        for file in files {
            let installedURL = prefix.appending(path: file.path)
            let evidence = try safeRegularFileEvidence(installedURL)
            guard evidence.size == file.size, evidence.digest == file.digest,
                  file.kind != "executable" || evidence.metadata.mode & 0o111 != 0 else {
                throw LifecycleError.changedBinary
            }
        }
    }
    return ReceiptEvidence(
        receiptDigest: recordedDigest,
        executableDigest: executable.digest,
        executableSize: executable.size,
        files: files
    )
}

func providerAllowedBundleReceiptPaths(version: String) -> Set<String> {
    let root = providerBundleRelativePath(version: version)
    return [
        "\(root)/Contents/Info.plist",
        "\(root)/Contents/MacOS/\(providerExecutableName)",
        "\(root)/Contents/Resources/ssiag-signing-policy.json",
        "\(root)/Contents/_CodeSignature/CodeResources",
        "\(root)/Contents/embedded.provisionprofile",
    ]
}

private func installationPrefix(for executable: URL, relativePath: String) throws -> URL {
    let components = relativePath.split(separator: "/")
    var prefix = executable.standardizedFileURL
    for _ in components { prefix.deleteLastPathComponent() }
    guard prefix.path.hasPrefix("/"), prefix.path != "/" else { throw LifecycleError.unsafePath }
    return prefix
}

func validateFoundationReceipt(parentExecutable: URL, requestedInstallationDigest: String) throws -> ReceiptEvidence {
    try validateProtectedAncestors(parentExecutable)
    let components = parentExecutable.standardizedFileURL.pathComponents
    guard components.count >= 7 else { throw FoundationTrustFailure.layout }
    let tail = Array(components.suffix(5))
    guard tail[0] == "libexec", tail[1] == "symphony",
          tail[2] == "secure-identity-access-governance", tail[4] == "symphony-ssiag",
          validVersion(tail[3]) else { throw FoundationTrustFailure.layout }
    let prefixComponents = Array(components.dropLast(5))
    let prefix = NSString.path(withComponents: prefixComponents)
    guard prefix.hasPrefix("/") && prefix != "/" else { throw FoundationTrustFailure.layout }
    let receipt = URL(fileURLWithPath: prefix, isDirectory: true)
        .appending(path: "share/symphony/receipts/secure-identity-access-governance/\(tail[3])/install-receipt.json")
    try validateProtectedAncestors(receipt)
    let receiptMetadata = try safeRegularFileMetadata(receipt)
    guard receiptMetadata.size <= 1_048_576 else { throw FoundationTrustFailure.receipt }
    let object = try strictJSONObject(Data(contentsOf: receipt, options: []))
    try requireExactKeys(object, [
        "protocol", "format_version", "component_id", "component_kind", "module_id",
        "vector_id", "engine_id", "package_id", "version", "install_scope", "prefix_mode",
        "files", "entry_points", "provides_capabilities", "requires_capabilities",
        "compatible_receptors", "platform_requirements", "receipt_digest",
    ])
    guard object["protocol"] as? String == receiptProtocolV2,
          integer(object["format_version"]) == 2,
          object["component_id"] as? String == "secure-identity-access-governance",
          object["component_kind"] as? String == "service",
          object["module_id"] as? String == "secure-identity-access-governance",
          object["vector_id"] is NSNull,
          object["engine_id"] is NSNull,
          object["package_id"] as? String == "secure-identity-access-governance",
          object["version"] as? String == tail[3],
          ["prefix", "user", "system", "tops"].contains(object["install_scope"] as? String ?? ""),
          object["prefix_mode"] as? String == "installation_prefix",
          let receiptDigest = object["receipt_digest"] as? String,
          receiptDigest == requestedInstallationDigest,
          try canonicalDigest(object, omitting: "receipt_digest") == receiptDigest else {
        throw FoundationTrustFailure.receipt
    }
    let relative = "libexec/symphony/secure-identity-access-governance/\(tail[3])/symphony-ssiag"
    guard validTokenArray(object["provides_capabilities"], maximum: 128),
          validTokenArray(object["requires_capabilities"], maximum: 128),
          validTokenArray(object["compatible_receptors"], maximum: 128),
          let files = object["files"] as? [Any], !files.isEmpty, files.count <= 4096 else {
        throw FoundationTrustFailure.receipt
    }
    var ownedPaths = Set<String>()
    var expectedSize: UInt64?
    var expectedDigest: String?
    for value in files {
        guard let file = value as? [String: Any] else { throw FoundationTrustFailure.receipt }
        try requireExactKeys(file, ["path", "kind", "size", "digest"])
        guard let path = file["path"] as? String, validRelativeReceiptPath(path), ownedPaths.insert(path).inserted,
              ["regular", "executable"].contains(file["kind"] as? String ?? ""),
              let size = unsignedInteger(file["size"]),
              let digest = file["digest"] as? String, validDigest(digest) else {
            throw FoundationTrustFailure.receipt
        }
        if path == relative {
            guard file["kind"] as? String == "executable" else { throw FoundationTrustFailure.receipt }
            expectedSize = size
            expectedDigest = digest
        }
    }
    guard let expectedSize, let expectedDigest else { throw FoundationTrustFailure.receipt }

    guard let entries = object["entry_points"] as? [Any], entries.count <= 128 else {
        throw FoundationTrustFailure.receipt
    }
    var entryIDs = Set<String>()
    for value in entries {
        guard let entry = value as? [String: Any] else { throw FoundationTrustFailure.receipt }
        try requireExactKeys(entry, ["entry_point_id", "kind", "path", "protocols"])
        guard let entryID = entry["entry_point_id"] as? String, validReceiptToken(entryID), entryIDs.insert(entryID).inserted,
              ["executable", "descriptor", "adapter"].contains(entry["kind"] as? String ?? ""),
              let entryPath = entry["path"] as? String, ownedPaths.contains(entryPath),
              validTokenArray(entry["protocols"], maximum: 64) else {
            throw FoundationTrustFailure.receipt
        }
    }
    guard let platforms = object["platform_requirements"] as? [Any], !platforms.isEmpty, platforms.count <= 128 else {
        throw FoundationTrustFailure.receipt
    }
    var currentPlatform = false
    for value in platforms {
        guard let platform = value as? [String: Any] else { throw FoundationTrustFailure.receipt }
        try requireExactKeys(platform, ["os", "architecture", "kernel_abi", "critical"])
        guard ["linux", "macos"].contains(platform["os"] as? String ?? ""),
              let architecture = platform["architecture"] as? String, validReceiptToken(architecture),
              platform["kernel_abi"] is NSNull || validReceiptToken(platform["kernel_abi"] as? String ?? ""),
              strictBoolean(platform["critical"]) != nil else { throw FoundationTrustFailure.receipt }
        if platform["os"] as? String == "macos" && architecture == runtimeReceiptArchitecture() && strictBoolean(platform["critical"]) == true {
            currentPlatform = true
        }
    }
    guard currentPlatform else { throw FoundationTrustFailure.receipt }
    let executableEvidence = try safeRegularFileEvidence(parentExecutable)
    guard receiptMetadata.uid == executableEvidence.metadata.uid,
          receiptMetadata.gid == executableEvidence.metadata.gid,
          executableEvidence.size == expectedSize, executableEvidence.digest == expectedDigest else {
        throw FoundationTrustFailure.executable
    }
    return ReceiptEvidence(
        receiptDigest: receiptDigest,
        executableDigest: expectedDigest,
        executableSize: expectedSize,
        files: [ReceiptFileEvidence(path: relative, kind: "executable", size: expectedSize, digest: expectedDigest)]
    )
}

private func validateProtectedAncestors(_ url: URL) throws {
    var current = url.standardizedFileURL.deletingLastPathComponent()
    while true {
        var info = stat()
        guard lstat(current.path, &info) == 0 else {
            throw FoundationTrustFailure.unsafeFile
        }
        let fixedAlias = permittedSystemAlias(current.path, info: info)
        guard fixedAlias || ((info.st_mode & S_IFMT) == S_IFDIR &&
              (info.st_mode & 0o022) == 0 &&
              (info.st_uid == geteuid() || info.st_uid == 0)) else {
            throw FoundationTrustFailure.unsafeFile
        }
        if current.path == "/" { return }
        let parent = current.deletingLastPathComponent()
        guard parent.path != current.path else { throw FoundationTrustFailure.unsafeFile }
        current = parent
    }
}

private func permittedSystemAlias(_ path: String, info: stat) -> Bool {
    let expected = ["/var": "/private/var", "/tmp": "/private/tmp", "/etc": "/private/etc"]
    guard (info.st_mode & S_IFMT) == S_IFLNK, let expectedTarget = expected[path],
          let resolvedPointer = realpath(path, nil) else { return false }
    defer { free(resolvedPointer) }
    return String(cString: resolvedPointer) == expectedTarget && info.st_uid == 0
}

private func validRelativeReceiptPath(_ value: String) -> Bool {
    guard !value.isEmpty, value.utf8.count <= 4096, !value.hasPrefix("/"), !value.hasPrefix(".symphony-"),
          !value.hasPrefix("share/symphony/receipts"), !value.contains("\\"), !value.contains("//") else { return false }
    return !value.split(separator: "/", omittingEmptySubsequences: false).contains { $0 == "." || $0 == ".." || $0.isEmpty }
}

private func validTokenArray(_ value: Any?, maximum: Int) -> Bool {
    guard let array = value as? [Any], array.count <= maximum else { return false }
    let strings = array.compactMap { $0 as? String }
    return strings.count == array.count && strings.allSatisfy(validReceiptToken) && Set(strings).count == strings.count
}

private func validReceiptToken(_ value: String) -> Bool {
    !value.isEmpty && value.utf8.count <= 256 && value.range(of: #"^[A-Za-z0-9._:-]+$"#, options: .regularExpression) != nil
}

private func runtimeReceiptArchitecture() -> String {
#if arch(arm64)
    "arm64"
#elseif arch(x86_64)
    "amd64"
#else
    "unsupported"
#endif
}

struct SafeFileMetadata {
    let uid: uid_t
    let gid: gid_t
    let mode: mode_t
    let size: UInt64
}

func safeRegularFileMetadata(_ url: URL) throws -> SafeFileMetadata {
    var info = stat()
    guard lstat(url.path, &info) == 0,
          (info.st_mode & S_IFMT) == S_IFREG,
          (info.st_mode & 0o022) == 0,
          info.st_uid == geteuid() || info.st_uid == 0 else {
        throw FoundationTrustFailure.unsafeFile
    }
    return SafeFileMetadata(uid: info.st_uid, gid: info.st_gid, mode: info.st_mode & 0o7777, size: UInt64(info.st_size))
}

func safeRegularFileEvidence(_ url: URL) throws -> (size: UInt64, digest: String, metadata: SafeFileMetadata) {
    let before = try safeRegularFileMetadata(url)
    let digest = try sha256Digest(of: url)
    let after = try safeRegularFileMetadata(url)
    guard before.uid == after.uid, before.gid == after.gid, before.mode == after.mode, before.size == after.size else {
        throw FoundationTrustFailure.executable
    }
    return (before.size, digest, before)
}

func requireExactKeys(_ object: [String: Any], _ expected: Set<String>) throws {
    guard Set(object.keys) == expected else { throw StrictJSONError.malformed }
}

func validDigest(_ value: String) -> Bool {
    value.range(of: #"^sha256:[0-9a-f]{64}$"#, options: .regularExpression) != nil
}

func validVersion(_ value: String) -> Bool {
    !value.isEmpty && value.utf8.count <= 64 && value.range(of: #"^[0-9A-Za-z.+-]+$"#, options: .regularExpression) != nil
}

private func integer(_ value: Any?) -> Int? {
    guard let number = value as? NSNumber, isIntegerNumber(number) else { return nil }
    return number.intValue
}

private func unsignedInteger(_ value: Any?) -> UInt64? {
    guard let number = value as? NSNumber, isIntegerNumber(number), number.int64Value >= 0 else { return nil }
    return number.uint64Value
}

private func strings(_ value: Any?) -> [String]? {
    guard let array = value as? [Any] else { return nil }
    let values = array.compactMap { $0 as? String }
    return values.count == array.count ? values : nil
}

private func strictBoolean(_ value: Any?) -> Bool? {
    guard let number = value as? NSNumber, CFGetTypeID(number) == CFBooleanGetTypeID() else { return nil }
    return number.boolValue
}

private func isIntegerNumber(_ value: NSNumber) -> Bool {
    guard CFGetTypeID(value) != CFBooleanGetTypeID() else { return false }
    return !["f", "d"].contains(String(cString: value.objCType))
}
