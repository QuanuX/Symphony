import Foundation

// The qualification runner builds outside the checkout. Local swift test keeps
// its existing debug-product default unless the caller selects an exact binary.
func providerTestExecutable() throws -> URL {
    let selected: URL
    if let path = ProcessInfo.processInfo.environment["SYMPHONY_SSIAG_TEST_EXECUTABLE"] {
        guard path.hasPrefix("/") else { throw ProviderTestExecutableError.invalidPath }
        selected = URL(fileURLWithPath: path)
    } else {
        selected = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
            .appending(path: ".build/debug/symphony-ssiag-provider-macos-keychain")
    }
    guard FileManager.default.isExecutableFile(atPath: selected.path) else {
        throw ProviderTestExecutableError.notExecutable(selected.path)
    }
    return selected
}

private enum ProviderTestExecutableError: Error {
    case invalidPath
    case notExecutable(String)
}
