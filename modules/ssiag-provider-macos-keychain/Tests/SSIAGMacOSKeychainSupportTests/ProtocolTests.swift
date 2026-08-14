import Foundation
import Testing
@testable import SSIAGMacOSKeychainSupport

private struct StubInspector: FoundationInspecting {
    let verified: Bool

    func inspect(_ request: ProviderControlRequest) -> FoundationObservation {
        FoundationObservation(
            verified: verified,
            executablePath: request.foundationExecutablePath,
            installationDigest: request.foundationInstallationDigest,
            executableDigest: request.foundationExecutableDigest,
            signingIdentity: "not_applicable",
            reasonCode: verified
                ? "symphony.ssiag.provider.foundation_verified"
                : "symphony.ssiag.provider.foundation_receipt_invalid"
        )
    }
}

@Test func stableOperationIdentitiesAreExactAndImplementationOwned() {
    #expect(providerOperationIDs == [
        "capabilities": "engop:symphony:ssiag.provider.metadata-capabilities",
        "handshake": "engop:symphony:ssiag.provider.metadata-handshake",
        "status": "engop:symphony:ssiag.provider.metadata-status",
    ])
    #expect(providerOperationID(for: "credential") == nil)
}

@Test func metadataHandshakeIsBoundAndNonOperational() throws {
    let result = try response(for: requestData(operation: "handshake"), inspector: StubInspector(verified: true))
    #expect(result.outcome == "succeeded")
    #expect(result.status == "disabled")
    #expect(result.handshake?.foundationTrust.verified == true)
    #expect(result.handshake?.exportable == false)
    #expect(result.handshake?.interactive == true)
    #expect(result.handshake?.limits.requestsPerProcess == 1)
    #expect(result.handshake?.limits.responsesPerProcess == 1)
    #expect(result.operationalAccessEnabled == false)
    #expect(result.providerOperationsEnabled == false)
    #expect(result.secretChannelEnabled == false)
    #expect(result.capabilities == ["capability-discovery", "metadata"])
    #expect(validDigest(result.responseDigest))
    #expect(validDigest(result.handshake?.handshakeDigest ?? ""))
    let responseObject = try strictJSONObject(Data(try encodedLine(result).dropLast()))
    #expect(try canonicalDigest(responseObject, omitting: "response_digest") == result.responseDigest)
    let handshakeObject = try #require(responseObject["handshake"] as? [String: Any])
    #expect(try canonicalDigest(handshakeObject, omitting: "handshake_digest") == result.handshake?.handshakeDigest)
    #expect(Set(handshakeObject.keys) == Set([
        "protocol", "provider_protocol", "provider_name", "provider_kind", "adapter_identifier",
        "adapter_version", "platform", "architecture", "transport", "control_request_protocol",
        "control_response_protocol", "one_shot_channel_protocol", "status", "reason_code",
        "foundation_trust", "capabilities", "exportable", "interactive", "safe_operations", "limits",
        "operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled", "handshake_digest",
    ]))
}

@Test func untrustedInvokerFailsClosed() throws {
    let result = try response(for: requestData(operation: "handshake"))
    #expect(result.outcome == "succeeded")
    #expect(result.status == "disabled")
    #expect(result.handshake?.foundationTrust.verified == false)
    #expect(result.reasonCode == "symphony.ssiag.provider.foundation_unverified")
    #expect(result.operationalAccessEnabled == false)
}

@Test func realProcessOneShotHandshakeHonorsBoundary() throws {
    let executable = packageRoot()
        .appending(path: ".build/debug/symphony-ssiag-provider-macos-keychain")
    #expect(FileManager.default.isExecutableFile(atPath: executable.path))

    let valid = try requestData(operation: "handshake")
    let accepted = try runProviderProcess(executable: executable, input: valid)
    #expect(accepted.status == 0)
    #expect(accepted.standardOutput.count <= maximumResponseBytes)
    #expect(accepted.standardOutput.last == 0x0a)
    #expect(accepted.standardOutput.dropLast().contains(0x0a) == false)
    let object = try strictJSONObject(Data(accepted.standardOutput.dropLast()))
    #expect(object["protocol"] as? String == "symphony.ssiag.provider-control-response.v1")
    #expect(object["outcome"] as? String == "succeeded")
    #expect(object["operational_access_enabled"] as? Bool == false)
    let handshake = try #require(object["handshake"] as? [String: Any])
    let foundation = try #require(handshake["foundation_trust"] as? [String: Any])
    #expect(foundation["verified"] as? Bool == false)

    let rejected = try runProviderProcess(executable: executable, input: valid + valid)
    #expect(rejected.status != 0)
    #expect(rejected.standardOutput.isEmpty)
}

@Test func realInstalledAdapterProcessProvidesGoIntegrationFixture() throws {
    let source = packageRoot().appending(path: ".build/debug/symphony-ssiag-provider-macos-keychain")
    let root = FileManager.default.temporaryDirectory
        .appending(path: "ssiag-installed-adapter-process-\(UUID().uuidString)", directoryHint: .isDirectory)
    defer { try? FileManager.default.removeItem(at: root) }
    let layout = try InstallLayout.resolve(.user, prefix: root.appending(path: "prefix", directoryHint: .isDirectory))
    let installed = try ProviderLifecycle.install(source: source, scope: .user, layout: layout)

    #expect(FileManager.default.isExecutableFile(atPath: installed.binary))
    #expect(FileManager.default.fileExists(atPath: installed.receipt))
    let exchange = try runProviderProcess(executable: URL(fileURLWithPath: installed.binary), input: requestData(operation: "handshake"))
    #expect(exchange.status == 0)
    let object = try strictJSONObject(Data(exchange.standardOutput.dropLast()))
    #expect(object["adapter_identifier"] as? String == providerAdapterIdentifier)
    #expect(object["operational_access_enabled"] as? Bool == false)
    let handshake = try #require(object["handshake"] as? [String: Any])
    let foundation = try #require(handshake["foundation_trust"] as? [String: Any])
    #expect(foundation["verified"] as? Bool == false)
}

@Test func statusAndCapabilitiesRemainMetadataOnly() throws {
    for operation in ["status", "capabilities"] {
        let result = try response(for: requestData(operation: operation), inspector: StubInspector(verified: true))
        #expect(result.operation == operation)
        #expect(result.handshake == nil)
        #expect(result.capabilities == ["capability-discovery", "metadata"])
        #expect(result.operationalAccessEnabled == false)
    }
}

@Test func unknownAndSecretShapedFieldsFailClosed() throws {
    for field in ["token", "secret", "credential", "password", "private_key", "assertion"] {
        var object = try requestObject(operation: "handshake")
        object[field] = "forbidden"
        let data = try finalizedRequest(object)
        #expect(throws: ProtocolError.self) {
            try response(for: data, inspector: StubInspector(verified: true))
        }
    }
}

@Test func credentialOperationsRemainDisabled() throws {
    for operation in ["read-secret", "write-secret", "sign", "decrypt", "export", "rotate"] {
        #expect(throws: ProtocolError.self) {
            try response(for: requestData(operation: operation), inspector: StubInspector(verified: true))
        }
    }
}

@Test func duplicateMalformedAndMultipleRequestsAreRejected() throws {
    let valid = try requestData(operation: "handshake")
    var duplicate = String(decoding: valid, as: UTF8.self)
    duplicate = duplicate.replacingOccurrences(of: "{", with: "{\"operation\":\"handshake\",", options: [], range: duplicate.startIndex..<duplicate.index(after: duplicate.startIndex))
    for input in [
        Data(duplicate.utf8),
        Data("{not-json}".utf8),
        valid + valid,
        valid + Data("\n{}".utf8),
    ] {
        #expect(throws: ProtocolError.self) {
            try response(for: input, inspector: StubInspector(verified: true))
        }
    }
}

@Test func requestAndResponseBoundsAreExact() throws {
    #expect(throws: ProtocolError.self) {
        try response(for: Data(repeating: 0x20, count: maximumRequestBytes + 1), inspector: StubInspector(verified: true))
    }
    let line = try encodedLine(response(for: requestData(operation: "handshake"), inspector: StubInspector(verified: true)))
    #expect(line.count <= maximumResponseBytes)
    #expect(line.last == 0x0a)
    #expect(line.dropLast().contains(0x0a) == false)
}

@Test func requestDigestAndDeadlineAreEnforced() throws {
    var badDigest = try requestObject(operation: "handshake")
    badDigest["request_digest"] = "sha256:" + String(repeating: "0", count: 64)
    #expect(throws: ProtocolError.self) {
        try response(for: try canonicalJSONData(badDigest), inspector: StubInspector(verified: true))
    }

    var expired = try requestObject(operation: "handshake")
    expired["requested_at"] = timestamp(Date(timeIntervalSinceNow: -20))
    expired["deadline_at"] = timestamp(Date(timeIntervalSinceNow: -10))
    expired["timeout_milliseconds"] = 30_000
    #expect(throws: ProtocolError.self) {
        try response(for: try finalizedRequest(expired), inspector: StubInspector(verified: true))
    }

    var numericBoolean = try requestObject(operation: "handshake")
    numericBoolean["secret_channel_requested"] = 0
    #expect(throws: ProtocolError.self) {
        try response(for: try finalizedRequest(numericBoolean), inspector: StubInspector(verified: true))
    }

    // Preserve the JSON number lexeme. Foundation bridging normalizes an
    // integral Double to NSNumber(30000), which cannot exercise the wire-level
    // rejection of a fractional/exponent spelling.
    let validTimeout = try finalizedRequest(requestObject(operation: "handshake"))
    let floatingTimeout = Data(
        String(decoding: validTimeout, as: UTF8.self)
            .replacingOccurrences(of: "\"timeout_milliseconds\":30000", with: "\"timeout_milliseconds\":30000.0")
            .utf8
    )
    #expect(throws: ProtocolError.self) {
        try response(for: floatingTimeout, inspector: StubInspector(verified: true))
    }
}

private func requestData(operation: String) throws -> Data {
    try finalizedRequest(requestObject(operation: operation))
}

private func requestObject(operation: String) throws -> [String: Any] {
    [
        "protocol": "symphony.ssiag.provider-control-request.v1",
        "request_id": "018f0c3a-7b2d-4e11-8c12-0242ac120002",
        "correlation_id": "018f0c3a-7b2d-4e11-8c12-0242ac120003",
        "tops_id": "018f0c3a-7b2d-4e11-8c12-0242ac120004",
        "provider_name": "macos-keychain",
        "adapter_identifier": providerAdapterIdentifier,
        "foundation_executable_path": "/tmp/unreceipted-symphony-ssiag",
        "foundation_installation_digest": "sha256:" + String(repeating: "1", count: 64),
        "foundation_executable_digest": "sha256:" + String(repeating: "2", count: 64),
        "foundation_signing_identity": "not_applicable",
        "operation": operation,
        "requested_at": timestamp(Date()),
        "deadline_at": timestamp(Date(timeIntervalSinceNow: 20)),
        "timeout_milliseconds": 30_000,
        "operational_access_requested": false,
        "provider_operation_requested": false,
        "secret_channel_requested": false,
        "request_digest": "sha256:" + String(repeating: "0", count: 64),
    ]
}

private func finalizedRequest(_ input: [String: Any]) throws -> Data {
    var object = input
    object["request_digest"] = try canonicalDigest(object, omitting: "request_digest")
    return try canonicalJSONData(object)
}

private func timestamp(_ value: Date) -> String {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.calendar = Calendar(identifier: .gregorian)
    formatter.timeZone = TimeZone(secondsFromGMT: 0)
    formatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ss'Z'"
    return formatter.string(from: value)
}

private func packageRoot() -> URL {
    URL(fileURLWithPath: #filePath)
        .deletingLastPathComponent()
        .deletingLastPathComponent()
        .deletingLastPathComponent()
}

private func runProviderProcess(executable: URL, input: Data) throws -> (status: Int32, standardOutput: Data) {
    let process = Process()
    process.executableURL = executable
    process.arguments = ["serve"]
    let standardInput = Pipe()
    let standardOutput = Pipe()
    let standardError = Pipe()
    process.standardInput = standardInput
    process.standardOutput = standardOutput
    process.standardError = standardError
    try process.run()
    try standardInput.fileHandleForWriting.write(contentsOf: input)
    try standardInput.fileHandleForWriting.close()
    process.waitUntilExit()
    let output = standardOutput.fileHandleForReading.readDataToEndOfFile()
    _ = standardError.fileHandleForReading.readDataToEndOfFile()
    return (process.terminationStatus, output)
}
