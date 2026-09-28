import Darwin
import Foundation
import Testing
import XPC
@testable import SSIAGMacOSKeychainSupport

@Suite(.serialized)
struct NativeMessageGateTests {
    @Test(arguments: ["matched", "duplicate", "wrong-code", "wrong-uid", "wrong-request", "wrong-tops", "wrong-digest", "wrong-challenge", "extra", "type", "missing", "oversize", "data", "expired"])
    func crossProcessMessageBoundary(scenario: String) throws {
        let result = try MessageFixture.run(scenario)
        switch scenario {
        case "matched", "duplicate": #expect(result == "matched")
        case "wrong-code": #expect(result == "expired" || result == "disconnected")
        case "expired": #expect(result == "expired")
        default: #expect(result == "rejected")
        }
    }

    @Test func concurrentCancellationCompletesIdleGateOnce() throws {
        let listener = xpc_connection_create(nil, nil)
        let endpoint = xpc_endpoint_create(listener)
        let connection = xpc_connection_create_from_endpoint(endpoint)
        let result = MessageOutcomeCollector()
        let binding = try NativeMessageBinding(topsID: "018f0c3a-7b2d-7e11-8c12-0242ac120002", requestID: "684921d8-a8b5-49da-872b-568eb6a6dc03", bindingDigest: "sha256:" + String(repeating: "a", count: 64), challenge: "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287")
        let policy = try NativeMessagePolicy(uid: geteuid(), gid: getegid(), requirementText: "true")
        let gate = try NativeMessageGate(connection: connection, policy: policy, binding: binding, timeout: 1) { result.append($0) }
        DispatchQueue.concurrentPerform(iterations: 64) { _ in gate.cancel() }
        #expect(result.done.wait(timeout: .now() + 2) == .success)
        #expect(result.values == [.cancelled])
        xpc_connection_set_event_handler(listener) { _ in }
        xpc_connection_activate(listener); xpc_connection_cancel(listener)
        withExtendedLifetime(gate) {}
    }

    @Test func malformedOwnerBindingAndPolicyRefuse() {
        #expect(throws: NativeMessageFailure.self) {
            try NativeMessageBinding(topsID: "bad", requestID: "bad", bindingDigest: "bad", challenge: "bad")
        }
        for text in ["", "identifier (", "true\0false", String(repeating: "x", count: 4097)] {
            #expect(throws: NativeMessageFailure.self) { try NativeMessagePolicy(uid: geteuid(), gid: getegid(), requirementText: text) }
        }
    }

    @Test func invalidLifetimeRefusesBeforeActivation() throws {
        let binding = try NativeMessageBinding(topsID: "018f0c3a-7b2d-7e11-8c12-0242ac120002", requestID: "684921d8-a8b5-49da-872b-568eb6a6dc03", bindingDigest: "sha256:" + String(repeating: "a", count: 64), challenge: "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287")
        let policy = try NativeMessagePolicy(uid: geteuid(), gid: getegid(), requirementText: "true")
        for timeout in [0.0, -1, 30.1, .infinity, .nan] {
            let connection = xpc_connection_create(nil, nil)
            #expect(throws: NativeMessageFailure.self) { try NativeMessageGate(connection: connection, policy: policy, binding: binding, timeout: timeout) { _ in } }
            xpc_connection_set_event_handler(connection) { _ in }
            xpc_connection_activate(connection); xpc_connection_cancel(connection)
        }
    }
}

private final class MessageOutcomeCollector: @unchecked Sendable {
    private let lock = NSLock()
    private var stored: [NativeMessageOutcome] = []
    let done = DispatchSemaphore(value: 0)
    var values: [NativeMessageOutcome] { lock.lock(); defer { lock.unlock() }; return stored }
    func append(_ value: NativeMessageOutcome) { lock.lock(); stored.append(value); lock.unlock(); done.signal() }
}

private enum MessageTestError: Error { case failed }
private enum MessageFixture {
    static let compiled: Result<URL, Error> = Result {
        let sanitized = ProcessInfo.processInfo.environment["SQV19_SANITIZE"] == "thread"
        let root = URL(fileURLWithPath: "/private/tmp/sqv19-compiled-\(sanitized ? "tsan-" : "")\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: root, withIntermediateDirectories: false, attributes: [.posixPermissions: 0o700])
        let module = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
        let fixture = module.appending(path: "Tests/Fixtures")
        // Swift permits top-level fixture startup only in a file named main.swift.
        let main = root.appending(path: "main.swift")
        try FileManager.default.copyItem(at: fixture.appending(path: "NativeMessageService.swift"), to: main)
        var serviceArguments = ["swiftc", "-swift-version", "6", "-module-cache-path", root.appending(path: "cache").path,
             module.appending(path: "Sources/SSIAGMacOSKeychainSupport/NativeMessageGate.swift").path, main.path, "-o", root.appending(path: "service").path]
        if sanitized { serviceArguments += ["-sanitize=thread", "-D", "SQV19_TSAN"] }
        try command("/usr/bin/xcrun", serviceArguments)
        try command("/usr/bin/xcrun", ["clang++", "-std=c++2c", "-fblocks", "-Wall", "-Wextra", "-Werror", fixture.appending(path: "NativeMessageClient.cpp").path, "-o", root.appending(path: "host").path])
        return root
    }

    static func run(_ scenario: String) throws -> String {
        let compiled = try compiled.get()
        let root = URL(fileURLWithPath: "/private/tmp/sqv19-case-\(UUID().uuidString)")
        defer { try? FileManager.default.removeItem(at: root) }
        let app = root.appending(path: "Fixture.app/Contents")
        let service = app.appending(path: "XPCServices/peer.xpc/Contents")
        for dir in [app, service] { try FileManager.default.createDirectory(at: dir.appending(path: "MacOS"), withIntermediateDirectories: true, attributes: [.posixPermissions: 0o700]) }
        let output = root.appending(path: "outcome")
        try FileManager.default.copyItem(at: compiled.appending(path: "host"), to: app.appending(path: "MacOS/host"))
        try FileManager.default.copyItem(at: compiled.appending(path: "service"), to: service.appending(path: "MacOS/service"))
        let hostInfo: [String: Any] = ["CFBundleIdentifier": "symphony.fixture.message-host", "CFBundleExecutable": "host", "CFBundlePackageType": "APPL", "CFBundleVersion": "1"]
        let serviceInfo: [String: Any] = ["CFBundleIdentifier": "symphony.fixture.message-service", "CFBundleExecutable": "service", "CFBundlePackageType": "XPC!", "CFBundleVersion": "1", "XPCService": ["ServiceType": "Application", "RunLoopType": "dispatch_main"], "FixtureOutput": output.path, "FixtureScenario": scenario, "FixtureSanitizerReport": root.appending(path: "tsan-report").path]
        try PropertyListSerialization.data(fromPropertyList: hostInfo, format: .xml, options: 0).write(to: app.appending(path: "Info.plist"))
        try PropertyListSerialization.data(fromPropertyList: serviceInfo, format: .xml, options: 0).write(to: service.appending(path: "Info.plist"))
        try command("/usr/bin/codesign", ["--force", "--sign", "-", service.deletingLastPathComponent().path])
        try command("/usr/bin/codesign", ["--force", "--sign", "-", app.deletingLastPathComponent().path])
        try command(app.appending(path: "MacOS/host").path, [scenario, output.path])
        let reports = try FileManager.default.contentsOfDirectory(at: root, includingPropertiesForKeys: nil).filter { $0.lastPathComponent.hasPrefix("tsan-report") }
        for report in reports { FileHandle.standardError.write(try Data(contentsOf: report)) }
        guard reports.isEmpty else { throw MessageTestError.failed }
        return try String(contentsOf: output, encoding: .utf8)
    }

    static func command(_ executable: String, _ arguments: [String]) throws {
        let process = Process(); process.executableURL = URL(fileURLWithPath: executable); process.arguments = arguments
        process.environment = ["PATH": "/usr/bin:/bin", "LANG": "C", "LC_ALL": "C"]
        process.standardOutput = FileHandle.nullDevice
        // Build errors contain no credentials and must remain diagnosable.
        process.standardError = FileHandle.standardError
        try process.run(); process.waitUntilExit()
        guard process.terminationStatus == 0 else { throw MessageTestError.failed }
    }
}
