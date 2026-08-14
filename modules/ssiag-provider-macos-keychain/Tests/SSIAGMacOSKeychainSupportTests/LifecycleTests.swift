import Foundation
import Testing
@testable import SSIAGMacOSKeychainSupport

@Test func immutableReceiptV2LifecycleIsDigestSafe() throws {
    let fixture = try LifecycleFixture()
    defer { fixture.cleanup() }

    let installed = try ProviderLifecycle.install(source: fixture.source, scope: .user, layout: fixture.layout)
    #expect(installed.protocolVersion == "symphony.ssiag.provider-package-result.v1")
    #expect(installed.changed == true)
    #expect(installed.binary.hasSuffix("/libexec/symphony/ssiag-macos-keychain-provider/\(providerVersion)/symphony-ssiag-provider-macos-keychain"))
    #expect(installed.receipt.hasSuffix("/share/symphony/receipts/ssiag-macos-keychain-provider/\(providerVersion)/install-receipt.json"))
    #expect(validDigest(installed.binarySHA256))
    #expect(validDigest(installed.receiptDigest))

    let repeated = try ProviderLifecycle.install(source: fixture.source, scope: .user, force: true, layout: fixture.layout)
    #expect(repeated.changed == false)
    #expect(repeated.receiptDigest == installed.receiptDigest)

    let removed = try ProviderLifecycle.uninstall(scope: .user, layout: fixture.layout)
    #expect(removed.receiptDigest == installed.receiptDigest)
    #expect(!FileManager.default.fileExists(atPath: fixture.layout.binary.path))
    #expect(!FileManager.default.fileExists(atPath: fixture.layout.receipt.path))
}

@Test func forceNeverReplacesOrRemovesChangedImmutableBytes() throws {
    let fixture = try LifecycleFixture()
    defer { fixture.cleanup() }
    _ = try ProviderLifecycle.install(source: fixture.source, scope: .user, layout: fixture.layout)

    let replacement = fixture.root.appending(path: "replacement")
    try Data("different-adapter".utf8).write(to: replacement)
    try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: replacement.path)
    #expect(throws: LifecycleError.self) {
        try ProviderLifecycle.install(source: replacement, scope: .user, force: true, layout: fixture.layout)
    }

    try Data("tampered".utf8).write(to: fixture.layout.binary)
    try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: fixture.layout.binary.path)
    #expect(throws: LifecycleError.self) {
        try ProviderLifecycle.uninstall(scope: .user, force: true, layout: fixture.layout)
    }
    #expect(FileManager.default.fileExists(atPath: fixture.layout.receipt.path))
}

@Test func interruptedUninstallSelfHealsFromReceiptLastEvidence() throws {
    let fixture = try LifecycleFixture()
    defer { fixture.cleanup() }
    let installed = try ProviderLifecycle.install(source: fixture.source, scope: .user, layout: fixture.layout)

    // This is the only partial state produced by receipt-last uninstall: the
    // exact binary is absent while its protected receipt still records what
    // was removed. Retrying must validate that receipt and finish cleanup.
    try FileManager.default.removeItem(at: fixture.layout.binary)
    let recovered = try ProviderLifecycle.uninstall(scope: .user, layout: fixture.layout)
    #expect(recovered.changed == true)
    #expect(recovered.receiptDigest == installed.receiptDigest)
    #expect(!FileManager.default.fileExists(atPath: fixture.layout.receipt.path))

    let replayed = try ProviderLifecycle.uninstall(scope: .user, layout: fixture.layout)
    #expect(replayed.changed == false)
    #expect(replayed.binarySHA256 == "not_applicable")
    #expect(replayed.receiptDigest == "not_applicable")
}

@Test func legacyCustomManifestIsEvidenceOnlyAndNeverSelected() throws {
    let fixture = try LifecycleFixture()
    defer { fixture.cleanup() }
    let legacyBinary = fixture.root.appending(path: "bin/symphony-ssiag-provider-macos-keychain")
    let legacyManifest = fixture.root.appending(path: "state/install.json")
    try FileManager.default.createDirectory(at: legacyBinary.deletingLastPathComponent(), withIntermediateDirectories: true)
    try FileManager.default.createDirectory(at: legacyManifest.deletingLastPathComponent(), withIntermediateDirectories: true)
    try Data("legacy".utf8).write(to: legacyBinary)
    try Data(#"{"schema":"symphony.ssiag.provider.macos-keychain.install.v1"}"#.utf8).write(to: legacyManifest)

    let installed = try ProviderLifecycle.install(source: fixture.source, scope: .user, layout: fixture.layout)
    #expect(installed.binary != legacyBinary.path)
    #expect(FileManager.default.fileExists(atPath: legacyBinary.path))
    #expect(FileManager.default.fileExists(atPath: legacyManifest.path))
}

@Test func symlinkedInstallationAncestorFailsClosed() throws {
    let root = FileManager.default.temporaryDirectory.appending(path: "ssiag-provider-link-test-\(UUID().uuidString)", directoryHint: .isDirectory)
    defer { try? FileManager.default.removeItem(at: root) }
    try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
    let source = root.appending(path: "source")
    try Data("adapter".utf8).write(to: source)
    try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: source.path)
    let external = root.appending(path: "external", directoryHint: .isDirectory)
    try FileManager.default.createDirectory(at: external, withIntermediateDirectories: true)
    let link = root.appending(path: "linked")
    try FileManager.default.createSymbolicLink(at: link, withDestinationURL: external)
    let layout = try InstallLayout.resolve(.user, prefix: link)
    #expect(throws: LifecycleError.self) {
        try ProviderLifecycle.install(source: source, scope: .user, layout: layout)
    }
}

private final class LifecycleFixture {
    let root: URL
    let source: URL
    let layout: InstallLayout

    init() throws {
        root = FileManager.default.temporaryDirectory.appending(path: "ssiag-provider-test-\(UUID().uuidString)", directoryHint: .isDirectory)
        try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
        source = root.appending(path: "source")
        try Data("adapter".utf8).write(to: source)
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: source.path)
        layout = try InstallLayout.resolve(.user, prefix: root.appending(path: "prefix", directoryHint: .isDirectory))
    }

    func cleanup() {
        try? FileManager.default.removeItem(at: root)
    }
}
