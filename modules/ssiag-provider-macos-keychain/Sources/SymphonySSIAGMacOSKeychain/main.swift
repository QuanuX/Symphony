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

func scopeAndForce(_ arguments: ArraySlice<String>) throws -> (InstallScope, Bool) {
    var scope = InstallScope.user
    var force = false
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
            force = true
        default:
            throw LifecycleError.unsafePath
        }
        index = arguments.index(after: index)
    }
    return (scope, force)
}

func serve() throws {
    let input = FileHandle.standardInput
    var buffer = Data()
    while let chunk = try input.read(upToCount: 4096), !chunk.isEmpty {
        for byte in chunk {
            if byte == 0x0a {
                if !buffer.isEmpty {
                    FileHandle.standardOutput.write(try encodedLine(response(for: buffer)))
                    buffer.removeAll(keepingCapacity: true)
                }
                continue
            }
            guard buffer.count < maximumRequestBytes else { throw ProtocolError.invalidShape }
            buffer.append(byte)
        }
    }
    if !buffer.isEmpty {
        FileHandle.standardOutput.write(try encodedLine(response(for: buffer)))
    }
}

func run() throws {
    let arguments = CommandLine.arguments
    guard arguments.count > 1 else {
        throw ProtocolError.unsupportedOperation
    }
    switch arguments[1] {
    case "--version", "version":
        print("symphony-ssiag-provider-macos-keychain version \(providerVersion)")
    case "status", "capabilities":
        try emit(ProviderDescriptor.scaffold)
    case "serve":
        try serve()
    case "install":
        let (scope, force) = try scopeAndForce(arguments.dropFirst(2))
        guard let source = Bundle.main.executableURL?.standardizedFileURL else {
            throw LifecycleError.sourceNotRegular
        }
        try emit(ProviderLifecycle.install(source: source, scope: scope, force: force))
    case "uninstall":
        let (scope, force) = try scopeAndForce(arguments.dropFirst(2))
        try emit(ProviderLifecycle.uninstall(scope: scope, force: force))
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
