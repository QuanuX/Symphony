import Darwin
import Foundation
import SSIAGMacOSKeychainSupport

func writeStandardError(_ message: String) {
    FileHandle.standardError.write(Data(("symphony-ssiag-provider-macos-keychain: \(message)\n").utf8))
}

func emit(_ value: some Encodable) throws {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
    var data = try encoder.encode(value)
    data.append(0x0a)
    FileHandle.standardOutput.write(data)
}

func lifecycleOptions(_ arguments: ArraySlice<String>) throws -> (InstallScope, Bool, URL?) {
    var scope = InstallScope.user
    var force = false
    var prefix: URL?
    var index = arguments.startIndex
    while index < arguments.endIndex {
        switch arguments[index] {
        case "--scope":
            index = arguments.index(after: index)
            guard index < arguments.endIndex, let parsed = InstallScope(rawValue: arguments[index]) else {
                throw LifecycleError.unsafePath
            }
            scope = parsed
        case "--force":
            // Retained for invocation compatibility only. Receipt-v2 package
            // bytes are immutable and the lifecycle never uses this flag to
            // replace or remove changed bytes.
            force = true
        case "--prefix":
            index = arguments.index(after: index)
            guard index < arguments.endIndex, arguments[index].hasPrefix("/"), arguments[index] != "/" else {
                throw LifecycleError.unsafePath
            }
            prefix = URL(fileURLWithPath: arguments[index], isDirectory: true)
        case "--version":
            index = arguments.index(after: index)
            guard index < arguments.endIndex, arguments[index] == providerVersion else {
                throw LifecycleError.immutableVersion
            }
        default:
            throw LifecycleError.unsafePath
        }
        index = arguments.index(after: index)
    }
    return (scope, force, prefix)
}

func serve() throws {
    let input = FileHandle.standardInput
    var buffer = Data()
    while let chunk = try input.read(upToCount: min(4096, maximumRequestBytes + 1 - buffer.count)), !chunk.isEmpty {
        buffer.append(chunk)
        guard buffer.count <= maximumRequestBytes else { throw ProtocolError.oversized }
    }
    guard !buffer.isEmpty else { throw ProtocolError.invalidShape }
    // The strict decoder consumes the entire document. A second request,
    // trailing value, or JSONL stream therefore fails before any output.
    FileHandle.standardOutput.write(try encodedLine(response(for: buffer)))
}

func run() throws {
    let arguments = CommandLine.arguments
    guard arguments.count > 1 else {
        throw ProtocolError.unsupportedOperation
    }
    switch arguments[1] {
    case "--version", "version":
        print("symphony-ssiag-provider-macos-keychain version \(providerVersion)")
    case "serve":
        try serve()
    case "install":
        let (scope, force, prefix) = try lifecycleOptions(arguments.dropFirst(2))
        guard let source = Bundle.main.executableURL?.standardizedFileURL else {
            throw LifecycleError.sourceNotRegular
        }
        let layout = try InstallLayout.resolve(scope, prefix: prefix)
        try emit(ProviderLifecycle.install(source: source, scope: scope, force: force, layout: layout))
    case "uninstall":
        let (scope, force, prefix) = try lifecycleOptions(arguments.dropFirst(2))
        let layout = try InstallLayout.resolve(scope, prefix: prefix)
        try emit(ProviderLifecycle.uninstall(scope: scope, force: force, layout: layout))
    default:
        throw ProtocolError.unsupportedOperation
    }
}

do {
    try run()
} catch {
    writeStandardError(String(describing: error))
    exit(1)
}
