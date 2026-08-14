import Foundation

public let providerVersion = "0.1.0-draft"
public let providerProtocol = "symphony.ssiag.provider.v1"
public let maximumRequestBytes = 65_536
public let maximumResponseBytes = 65_536

public let providerOperationIDs = [
    "capabilities": "engop:symphony:ssiag.provider.metadata-capabilities",
    "handshake": "engop:symphony:ssiag.provider.metadata-handshake",
    "status": "engop:symphony:ssiag.provider.metadata-status",
]

public func providerOperationID(for operation: String) -> String? {
    providerOperationIDs[operation]
}

public enum ProtocolError: Error, CustomStringConvertible {
    case oversized
    case invalidJSON
    case invalidShape
    case unknownField
    case invalidProtocol
    case invalidIdentifier
    case invalidEvidence
    case invalidDigest
    case invalidTimestamp
    case expired
    case unsupportedOperation
    case responseOversized

    public var description: String {
        switch self {
        case .oversized: "bounded control message exceeded"
        case .invalidJSON: "invalid control JSON"
        case .invalidShape: "invalid control shape"
        case .unknownField: "unknown control field"
        case .invalidProtocol: "unsupported control protocol"
        case .invalidIdentifier: "invalid control identifier"
        case .invalidEvidence: "invalid foundation evidence"
        case .invalidDigest: "invalid control digest"
        case .invalidTimestamp: "invalid control timestamp"
        case .expired: "control deadline expired"
        case .unsupportedOperation: "unsupported control operation"
        case .responseOversized: "bounded response exceeded"
        }
    }
}

struct ProviderControlRequest: Sendable {
    let protocolVersion: String
    let requestID: String
    let correlationID: String
    let topsID: String
    let providerName: String
    let adapterIdentifier: String
    let foundationExecutablePath: String
    let foundationInstallationDigest: String
    let foundationExecutableDigest: String
    let foundationSigningIdentity: String
    let operation: String
    let requestedAt: String
    let deadlineAt: String
    let timeoutMilliseconds: Int
    let operationalAccessRequested: Bool
    let providerOperationRequested: Bool
    let secretChannelRequested: Bool
    let requestDigest: String
}

public struct FoundationTrust: Codable, Sendable {
    public let verified: Bool
    public let executablePath: String
    public let installationDigest: String
    public let executableDigest: String
    public let signingIdentity: String
    public let reasonCode: String

    enum CodingKeys: String, CodingKey {
        case verified
        case executablePath = "executable_path"
        case installationDigest = "installation_digest"
        case executableDigest = "executable_digest"
        case signingIdentity = "signing_identity"
        case reasonCode = "reason_code"
    }
}

public struct ProviderLimits: Codable, Sendable {
    public let maximumControlRequestBytes = 65_536
    public let maximumControlResponseBytes = 65_536
    public let defaultTimeoutMilliseconds = 5_000
    public let maximumTimeoutMilliseconds = 30_000
    public let maximumCapabilities = 128
    public let maximumChecks = 128
    public let requestsPerProcess = 1
    public let responsesPerProcess = 1

    enum CodingKeys: String, CodingKey {
        case maximumControlRequestBytes = "maximum_control_request_bytes"
        case maximumControlResponseBytes = "maximum_control_response_bytes"
        case defaultTimeoutMilliseconds = "default_timeout_milliseconds"
        case maximumTimeoutMilliseconds = "maximum_timeout_milliseconds"
        case maximumCapabilities = "maximum_capabilities"
        case maximumChecks = "maximum_checks"
        case requestsPerProcess = "requests_per_process"
        case responsesPerProcess = "responses_per_process"
    }
}

public struct ProviderHandshake: Codable, Sendable {
    public let protocolVersion: String
    public let providerProtocol: String
    public let providerName: String
    public let providerKind: String
    public let adapterIdentifier: String
    public let adapterVersion: String
    public let platform: String
    public let architecture: String
    public let transport: String
    public let controlRequestProtocol: String
    public let controlResponseProtocol: String
    public let oneShotChannelProtocol: String
    public let status: String
    public let reasonCode: String
    public let foundationTrust: FoundationTrust
    public let capabilities: [String]
    public let exportable: Bool
    public let interactive: Bool
    public let safeOperations: [String]
    public let limits: ProviderLimits
    public let operationalAccessEnabled: Bool
    public let providerOperationsEnabled: Bool
    public let secretChannelEnabled: Bool
    public let handshakeDigest: String

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case providerProtocol = "provider_protocol"
        case providerName = "provider_name"
        case providerKind = "provider_kind"
        case adapterIdentifier = "adapter_identifier"
        case adapterVersion = "adapter_version"
        case platform, architecture, transport
        case controlRequestProtocol = "control_request_protocol"
        case controlResponseProtocol = "control_response_protocol"
        case oneShotChannelProtocol = "one_shot_channel_protocol"
        case status
        case reasonCode = "reason_code"
        case foundationTrust = "foundation_trust"
        case capabilities
        case exportable, interactive
        case safeOperations = "safe_operations"
        case limits
        case operationalAccessEnabled = "operational_access_enabled"
        case providerOperationsEnabled = "provider_operations_enabled"
        case secretChannelEnabled = "secret_channel_enabled"
        case handshakeDigest = "handshake_digest"
    }
}

public struct SafeProviderError: Codable, Sendable {
    public let code: String
    public let category: String
    public let retryable: Bool
    public let nativeDetailIncluded = false
    public let secretMaterialIncluded = false

    enum CodingKeys: String, CodingKey {
        case code, category, retryable
        case nativeDetailIncluded = "native_detail_included"
        case secretMaterialIncluded = "secret_material_included"
    }
}

public struct ProviderControlResponse: Codable, Sendable {
    public let protocolVersion: String
    public let requestID: String
    public let correlationID: String
    public let topsID: String
    public let providerName: String
    public let adapterIdentifier: String
    public let operation: String
    public let deadlineAt: String
    public let outcome: String
    public let status: String
    public let reasonCode: String
    public let handshake: ProviderHandshake?
    public let capabilities: [String]
    public let error: SafeProviderError?
    public let operationalAccessEnabled: Bool
    public let providerOperationsEnabled: Bool
    public let secretChannelEnabled: Bool
    public let completedAt: String
    public let responseDigest: String

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case requestID = "request_id"
        case correlationID = "correlation_id"
        case topsID = "tops_id"
        case providerName = "provider_name"
        case adapterIdentifier = "adapter_identifier"
        case operation
        case deadlineAt = "deadline_at"
        case outcome, status
        case reasonCode = "reason_code"
        case handshake, capabilities, error
        case operationalAccessEnabled = "operational_access_enabled"
        case providerOperationsEnabled = "provider_operations_enabled"
        case secretChannelEnabled = "secret_channel_enabled"
        case completedAt = "completed_at"
        case responseDigest = "response_digest"
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(protocolVersion, forKey: .protocolVersion)
        try container.encode(requestID, forKey: .requestID)
        try container.encode(correlationID, forKey: .correlationID)
        try container.encode(topsID, forKey: .topsID)
        try container.encode(providerName, forKey: .providerName)
        try container.encode(adapterIdentifier, forKey: .adapterIdentifier)
        try container.encode(operation, forKey: .operation)
        try container.encode(deadlineAt, forKey: .deadlineAt)
        try container.encode(outcome, forKey: .outcome)
        try container.encode(status, forKey: .status)
        try container.encode(reasonCode, forKey: .reasonCode)
        if let handshake { try container.encode(handshake, forKey: .handshake) } else { try container.encodeNil(forKey: .handshake) }
        try container.encode(capabilities, forKey: .capabilities)
        if let error { try container.encode(error, forKey: .error) } else { try container.encodeNil(forKey: .error) }
        try container.encode(operationalAccessEnabled, forKey: .operationalAccessEnabled)
        try container.encode(providerOperationsEnabled, forKey: .providerOperationsEnabled)
        try container.encode(secretChannelEnabled, forKey: .secretChannelEnabled)
        try container.encode(completedAt, forKey: .completedAt)
        try container.encode(responseDigest, forKey: .responseDigest)
    }
}

public func response(for data: Data) throws -> ProviderControlResponse {
    try response(for: data, inspector: LiveFoundationInspector())
}

func response(for data: Data, inspector: FoundationInspecting) throws -> ProviderControlResponse {
    let request = try decodeRequest(data)
    let observation = inspector.inspect(request)
    let trust = FoundationTrust(
        verified: observation.verified,
        executablePath: observation.executablePath,
        installationDigest: observation.installationDigest,
        executableDigest: observation.executableDigest,
        signingIdentity: observation.signingIdentity,
        reasonCode: observation.reasonCode
    )
    let capabilities = ["capability-discovery", "metadata"]
    let reason = observation.verified
        ? "symphony.ssiag.provider.metadata_available"
        : "symphony.ssiag.provider.foundation_unverified"
    var handshake: ProviderHandshake?
    if request.operation == "handshake" {
        let provisional = ProviderHandshake(
            protocolVersion: "symphony.ssiag.provider-handshake.v1",
            providerProtocol: providerProtocol,
            providerName: request.providerName,
            providerKind: "macos-keychain",
            adapterIdentifier: providerAdapterIdentifier,
            adapterVersion: providerVersion,
            platform: "macos",
            architecture: runtimeArchitecture(),
            transport: "stdio-one-shot-json",
            controlRequestProtocol: "symphony.ssiag.provider-control-request.v1",
            controlResponseProtocol: "symphony.ssiag.provider-control-response.v1",
            oneShotChannelProtocol: "symphony.ssiag.provider-one-shot-channel.v1",
            status: "disabled",
            reasonCode: reason,
            foundationTrust: trust,
            capabilities: capabilities,
            exportable: false,
            interactive: true,
            safeOperations: ["capabilities", "handshake", "status"],
            limits: ProviderLimits(),
            operationalAccessEnabled: false,
            providerOperationsEnabled: false,
            secretChannelEnabled: false,
            handshakeDigest: String(repeating: "0", count: 71)
        )
        handshake = try replacingHandshakeDigest(provisional)
    } else {
        handshake = nil
    }
    let completedAt = formatTimestamp(Date())
    let provisional = ProviderControlResponse(
        protocolVersion: "symphony.ssiag.provider-control-response.v1",
        requestID: request.requestID,
        correlationID: request.correlationID,
        topsID: request.topsID,
        providerName: request.providerName,
        adapterIdentifier: request.adapterIdentifier,
        operation: request.operation,
        deadlineAt: request.deadlineAt,
        outcome: "succeeded",
        status: "disabled",
        reasonCode: reason,
        handshake: handshake,
        capabilities: capabilities,
        error: nil,
        operationalAccessEnabled: false,
        providerOperationsEnabled: false,
        secretChannelEnabled: false,
        completedAt: completedAt,
        responseDigest: String(repeating: "0", count: 71)
    )
    return try replacingResponseDigest(provisional)
}

public func encodedLine(_ response: ProviderControlResponse) throws -> Data {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.sortedKeys, .withoutEscapingSlashes]
    var data = try encoder.encode(response)
    guard data.count + 1 <= maximumResponseBytes else { throw ProtocolError.responseOversized }
    data.append(0x0a)
    return data
}

private func decodeRequest(_ data: Data) throws -> ProviderControlRequest {
    guard !data.isEmpty, data.count <= maximumRequestBytes else { throw ProtocolError.oversized }
    let object: [String: Any]
    do {
        object = try strictJSONObject(data)
    } catch StrictJSONError.duplicateKey {
        throw ProtocolError.invalidJSON
    } catch {
        throw ProtocolError.invalidJSON
    }
    let expected = Set([
        "protocol", "request_id", "correlation_id", "tops_id", "provider_name",
        "adapter_identifier", "foundation_executable_path", "foundation_installation_digest",
        "foundation_executable_digest", "foundation_signing_identity", "operation", "requested_at",
        "deadline_at", "timeout_milliseconds", "operational_access_requested",
        "provider_operation_requested", "secret_channel_requested", "request_digest",
    ])
    guard Set(object.keys) == expected else { throw ProtocolError.unknownField }
    guard object["protocol"] as? String == "symphony.ssiag.provider-control-request.v1" else {
        throw ProtocolError.invalidProtocol
    }
    guard let requestID = object["request_id"] as? String, validUUID(requestID),
          let correlationID = object["correlation_id"] as? String, validUUID(correlationID),
          let topsID = object["tops_id"] as? String, validUUID(topsID),
          let providerName = object["provider_name"] as? String, validToken(providerName),
          let adapterIdentifier = object["adapter_identifier"] as? String,
          adapterIdentifier == providerAdapterIdentifier else {
        throw ProtocolError.invalidIdentifier
    }
    guard let foundationPath = object["foundation_executable_path"] as? String,
          foundationPath.hasPrefix("/"), foundationPath.utf8.count <= 4096, !foundationPath.contains("\0"),
          let installationDigest = object["foundation_installation_digest"] as? String,
          installationDigest == "not_applicable" || validDigest(installationDigest),
          let executableDigest = object["foundation_executable_digest"] as? String, validDigest(executableDigest),
          let signingIdentity = object["foundation_signing_identity"] as? String,
          signingIdentity == "not_applicable" || validToken(signingIdentity) else {
        throw ProtocolError.invalidEvidence
    }
    guard let operation = object["operation"] as? String,
          providerOperationID(for: operation) != nil else {
        throw ProtocolError.unsupportedOperation
    }
    guard let requestedAt = object["requested_at"] as? String,
          let deadlineAt = object["deadline_at"] as? String,
          let requestedDate = parseTimestamp(requestedAt),
          let deadlineDate = parseTimestamp(deadlineAt),
          deadlineDate >= requestedDate,
          let timeout = exactInteger(object["timeout_milliseconds"]), (1...30_000).contains(timeout),
          deadlineDate.timeIntervalSince(requestedDate) * 1_000 <= Double(timeout + 999) else {
        throw ProtocolError.invalidTimestamp
    }
    guard Date() <= deadlineDate else { throw ProtocolError.expired }
    guard exactBoolean(object["operational_access_requested"]) == false,
          exactBoolean(object["provider_operation_requested"]) == false,
          exactBoolean(object["secret_channel_requested"]) == false else {
        throw ProtocolError.invalidEvidence
    }
    guard let requestDigest = object["request_digest"] as? String, validDigest(requestDigest),
          try canonicalDigest(object, omitting: "request_digest") == requestDigest else {
        throw ProtocolError.invalidDigest
    }
    return ProviderControlRequest(
        protocolVersion: "symphony.ssiag.provider-control-request.v1",
        requestID: requestID,
        correlationID: correlationID,
        topsID: topsID,
        providerName: providerName,
        adapterIdentifier: adapterIdentifier,
        foundationExecutablePath: foundationPath,
        foundationInstallationDigest: installationDigest,
        foundationExecutableDigest: executableDigest,
        foundationSigningIdentity: signingIdentity,
        operation: operation,
        requestedAt: requestedAt,
        deadlineAt: deadlineAt,
        timeoutMilliseconds: timeout,
        operationalAccessRequested: false,
        providerOperationRequested: false,
        secretChannelRequested: false,
        requestDigest: requestDigest
    )
}

private func replacingHandshakeDigest(_ value: ProviderHandshake) throws -> ProviderHandshake {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.sortedKeys, .withoutEscapingSlashes]
    let object = try strictJSONObject(encoder.encode(value))
    let digest = try canonicalDigest(object, omitting: "handshake_digest")
    return ProviderHandshake(
        protocolVersion: value.protocolVersion,
        providerProtocol: value.providerProtocol,
        providerName: value.providerName,
        providerKind: value.providerKind,
        adapterIdentifier: value.adapterIdentifier,
        adapterVersion: value.adapterVersion,
        platform: value.platform,
        architecture: value.architecture,
        transport: value.transport,
        controlRequestProtocol: value.controlRequestProtocol,
        controlResponseProtocol: value.controlResponseProtocol,
        oneShotChannelProtocol: value.oneShotChannelProtocol,
        status: value.status,
        reasonCode: value.reasonCode,
        foundationTrust: value.foundationTrust,
        capabilities: value.capabilities,
        exportable: value.exportable,
        interactive: value.interactive,
        safeOperations: value.safeOperations,
        limits: value.limits,
        operationalAccessEnabled: false,
        providerOperationsEnabled: false,
        secretChannelEnabled: false,
        handshakeDigest: digest
    )
}

private func replacingResponseDigest(_ value: ProviderControlResponse) throws -> ProviderControlResponse {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.sortedKeys, .withoutEscapingSlashes]
    let object = try strictJSONObject(encoder.encode(value))
    let digest = try canonicalDigest(object, omitting: "response_digest")
    return ProviderControlResponse(
        protocolVersion: value.protocolVersion,
        requestID: value.requestID,
        correlationID: value.correlationID,
        topsID: value.topsID,
        providerName: value.providerName,
        adapterIdentifier: value.adapterIdentifier,
        operation: value.operation,
        deadlineAt: value.deadlineAt,
        outcome: value.outcome,
        status: value.status,
        reasonCode: value.reasonCode,
        handshake: value.handshake,
        capabilities: value.capabilities,
        error: value.error,
        operationalAccessEnabled: false,
        providerOperationsEnabled: false,
        secretChannelEnabled: false,
        completedAt: value.completedAt,
        responseDigest: digest
    )
}

private func validUUID(_ value: String) -> Bool {
    value.range(of: #"^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"#, options: .regularExpression) != nil
}

private func validToken(_ value: String) -> Bool {
    !value.isEmpty && value.utf8.count <= 256 && value.range(of: #"^[A-Za-z0-9._:-]+$"#, options: .regularExpression) != nil
}

private func exactInteger(_ value: Any?) -> Int? {
    guard let number = value as? NSNumber,
          CFGetTypeID(number) != CFBooleanGetTypeID(),
          !["f", "d"].contains(String(cString: number.objCType)) else { return nil }
    let result = number.intValue
    return NSNumber(value: result) == number ? result : nil
}

private func exactBoolean(_ value: Any?) -> Bool? {
    guard let number = value as? NSNumber, CFGetTypeID(number) == CFBooleanGetTypeID() else { return nil }
    return number.boolValue
}

private func parseTimestamp(_ value: String) -> Date? {
    guard value.range(of: #"^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$"#, options: .regularExpression) != nil else {
        return nil
    }
    return timestampFormatter.date(from: value)
}

private func formatTimestamp(_ value: Date) -> String {
    timestampFormatter.string(from: value)
}

private let timestampFormatter: DateFormatter = {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "en_US_POSIX")
    formatter.calendar = Calendar(identifier: .gregorian)
    formatter.timeZone = TimeZone(secondsFromGMT: 0)
    formatter.dateFormat = "yyyy-MM-dd'T'HH:mm:ss'Z'"
    formatter.isLenient = false
    return formatter
}()

private func runtimeArchitecture() -> String {
#if arch(arm64)
    "arm64"
#elseif arch(x86_64)
    "amd64"
#else
    "unsupported"
#endif
}
