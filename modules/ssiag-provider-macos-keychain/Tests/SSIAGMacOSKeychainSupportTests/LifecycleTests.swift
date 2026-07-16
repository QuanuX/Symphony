import Foundation
import Testing
@testable import SSIAGMacOSKeychainSupport

@Test func independentLifecycleIsDigestSafe() throws {
    let root = FileManager.default.temporaryDirectory.appending(path: "ssiag-provider-test-\(UUID().uuidString)", directoryHint: .isDirectory)
    defer { try? FileManager.default.removeItem(at: root) }
    try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
    let source = root.appending(path: "source")
    try Data("adapter".utf8).write(to: source)
    try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: source.path)
    let state = root.appending(path: "state", directoryHint: .isDirectory)
    let layout = InstallLayout(
        binary: root.appending(path: "bin/provider"),
        stateDirectory: state,
        manifest: state.appending(path: "install.json")
    )

    let installed = try ProviderLifecycle.install(source: source, scope: .user, force: false, layout: layout)
    #expect(FileManager.default.fileExists(atPath: installed.binary))
    #expect(installed.binarySHA256.count == 64)

    try Data("changed".utf8).write(to: layout.binary)
    #expect(throws: LifecycleError.self) {
        try ProviderLifecycle.uninstall(scope: .user, force: false, layout: layout)
    }
    _ = try ProviderLifecycle.uninstall(scope: .user, force: true, layout: layout)
    #expect(!FileManager.default.fileExists(atPath: layout.binary.path))
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
    let state = root.appending(path: "state", directoryHint: .isDirectory)
    let layout = InstallLayout(
        binary: link.appending(path: "provider"),
        stateDirectory: state,
        manifest: state.appending(path: "install.json")
    )
    #expect(throws: LifecycleError.self) {
        try ProviderLifecycle.install(source: source, scope: .user, force: false, layout: layout)
    }
}
