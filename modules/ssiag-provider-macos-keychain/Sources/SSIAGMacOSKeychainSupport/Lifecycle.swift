import CryptoKit
import Foundation

public enum InstallScope: String, Codable, Sendable {
    case user
    case system
}

public struct InstallLayout: Sendable {
    public let binary: URL
    public let stateDirectory: URL
    public let manifest: URL

    public init(binary: URL, stateDirectory: URL, manifest: URL) {
        self.binary = binary
        self.stateDirectory = stateDirectory
        self.manifest = manifest
    }

    public static func resolve(_ scope: InstallScope) throws -> InstallLayout {
        switch scope {
        case .user:
            let home = FileManager.default.homeDirectoryForCurrentUser
            let stateBase: URL
            if let value = ProcessInfo.processInfo.environment["XDG_STATE_HOME"], !value.isEmpty {
                guard value.hasPrefix("/") else { throw LifecycleError.unsafePath }
                stateBase = URL(fileURLWithPath: value, isDirectory: true)
            } else {
                stateBase = home.appending(path: ".local/state", directoryHint: .isDirectory)
            }
            let state = stateBase.appending(path: "symphony/ssiag/providers/macos-keychain", directoryHint: .isDirectory)
            return InstallLayout(
                binary: home.appending(path: ".local/bin/symphony-ssiag-provider-macos-keychain"),
                stateDirectory: state,
                manifest: state.appending(path: "install.json")
            )
        case .system:
            let state = URL(fileURLWithPath: "/var/lib/symphony/ssiag/providers/macos-keychain", isDirectory: true)
            return InstallLayout(
                binary: URL(fileURLWithPath: "/usr/local/bin/symphony-ssiag-provider-macos-keychain"),
                stateDirectory: state,
                manifest: state.appending(path: "install.json")
            )
        }
    }
}

public struct InstallRecord: Codable, Sendable {
    public let schema: String
    public let scope: InstallScope
    public let version: String
    public let binary: String
    public let binarySHA256: String

    enum CodingKeys: String, CodingKey {
        case schema, scope, version, binary
        case binarySHA256 = "binary_sha256"
    }
}

public enum LifecycleError: Error, CustomStringConvertible {
    case unsafePath
    case sourceNotRegular
    case destinationNotRegular
    case changedBinary
    case invalidManifest

    public var description: String {
        switch self {
        case .unsafePath: "unsafe installation path"
        case .sourceNotRegular: "source executable is not a regular file"
        case .destinationNotRegular: "refusing non-regular installation path"
        case .changedBinary: "installed binary digest changed; use --force"
        case .invalidManifest: "installation manifest does not match the selected scope"
        }
    }
}

public enum ProviderLifecycle {
    public static func install(source: URL, scope: InstallScope, force: Bool, layout explicitLayout: InstallLayout? = nil) throws -> InstallRecord {
        let manager = FileManager.default
        let layout = try explicitLayout ?? InstallLayout.resolve(scope)
        guard try regularFile(source) else { throw LifecycleError.sourceNotRegular }
        try ensureDirectory(layout.binary.deletingLastPathComponent())
        try ensureDirectory(layout.stateDirectory)
        try requireAbsentOrRegular(layout.manifest)

        let sourceDigest = try digest(source)
        if manager.fileExists(atPath: layout.binary.path) {
            guard try regularFile(layout.binary) else { throw LifecycleError.destinationNotRegular }
            if try digest(layout.binary) != sourceDigest {
                guard force else { throw LifecycleError.changedBinary }
                try manager.removeItem(at: layout.binary)
                try copyAtomically(source: source, destination: layout.binary)
            }
        } else {
            try copyAtomically(source: source, destination: layout.binary)
        }

        let record = InstallRecord(
            schema: "symphony.ssiag.provider.macos-keychain.install.v1",
            scope: scope,
            version: providerVersion,
            binary: layout.binary.path,
            binarySHA256: sourceDigest
        )
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        var data = try encoder.encode(record)
        data.append(0x0a)
        try data.write(to: layout.manifest, options: .atomic)
        try manager.setAttributes([.posixPermissions: 0o600], ofItemAtPath: layout.manifest.path)
        return record
    }

    public static func uninstall(scope: InstallScope, force: Bool, layout explicitLayout: InstallLayout? = nil) throws -> InstallRecord {
        let manager = FileManager.default
        let layout = try explicitLayout ?? InstallLayout.resolve(scope)
        guard try regularFile(layout.manifest) else { throw LifecycleError.invalidManifest }
        let record = try JSONDecoder().decode(InstallRecord.self, from: Data(contentsOf: layout.manifest))
        guard record.schema == "symphony.ssiag.provider.macos-keychain.install.v1",
              record.scope == scope,
              record.binary == layout.binary.path,
              validDigest(record.binarySHA256) else { throw LifecycleError.invalidManifest }
        if manager.fileExists(atPath: layout.binary.path) {
            guard try regularFile(layout.binary) else { throw LifecycleError.destinationNotRegular }
            if try digest(layout.binary) != record.binarySHA256 && !force { throw LifecycleError.changedBinary }
            try manager.removeItem(at: layout.binary)
        }
        try manager.removeItem(at: layout.manifest)
        try? manager.removeItem(at: layout.stateDirectory)
        return record
    }

    private static func digest(_ url: URL) throws -> String {
        // Avoid memory-mapped executable data: lifecycle tests replace the
        // destination after hashing, and an outstanding mapping can fault.
        let hash = SHA256.hash(data: try Data(contentsOf: url))
        return hash.map { String(format: "%02x", $0) }.joined()
    }

    private static func regularFile(_ url: URL) throws -> Bool {
        if symbolicLink(url) { return false }
        let values = try url.resourceValues(forKeys: [.isRegularFileKey, .isSymbolicLinkKey])
        return values.isRegularFile == true && values.isSymbolicLink != true
    }

    private static func requireAbsentOrRegular(_ url: URL) throws {
        if symbolicLink(url) { throw LifecycleError.destinationNotRegular }
        if FileManager.default.fileExists(atPath: url.path), try !regularFile(url) {
            throw LifecycleError.destinationNotRegular
        }
    }

    private static func ensureDirectory(_ url: URL) throws {
        let manager = FileManager.default
        let path = url.standardizedFileURL
        let parent = path.deletingLastPathComponent()
        if path.path != "/" && parent.path != path.path {
            try ensureDirectory(parent)
        }
        if symbolicLink(path) {
            guard permittedSystemAlias(path) else { throw LifecycleError.unsafePath }
            return
        }
        if manager.fileExists(atPath: path.path) {
            let values = try path.resourceValues(forKeys: [.isDirectoryKey, .isSymbolicLinkKey])
            guard values.isDirectory == true, values.isSymbolicLink != true else {
                throw LifecycleError.unsafePath
            }
            return
        }
        try manager.createDirectory(at: path, withIntermediateDirectories: false, attributes: [.posixPermissions: 0o700])
    }

    private static func symbolicLink(_ url: URL) -> Bool {
        (try? FileManager.default.destinationOfSymbolicLink(atPath: url.path)) != nil
    }

    private static func validDigest(_ value: String) -> Bool {
        value.count == 64 && value.allSatisfy { $0.isNumber || ("a"..."f").contains(String($0)) }
    }

    private static func permittedSystemAlias(_ url: URL) -> Bool {
        let expected = ["/var": "private/var", "/tmp": "private/tmp", "/etc": "private/etc"]
        guard let wanted = expected[url.path],
              let destination = try? FileManager.default.destinationOfSymbolicLink(atPath: url.path) else {
            return false
        }
        return destination == wanted
    }

    private static func copyAtomically(source: URL, destination: URL) throws {
        let manager = FileManager.default
        let temporary = destination.deletingLastPathComponent().appending(path: ".ssiag-provider-\(UUID().uuidString)")
        defer { try? manager.removeItem(at: temporary) }
        try manager.copyItem(at: source, to: temporary)
        try manager.setAttributes([.posixPermissions: 0o755], ofItemAtPath: temporary.path)
        try manager.moveItem(at: temporary, to: destination)
    }
}
