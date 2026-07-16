import Foundation

public let providerVersion = "0.1.0-draft"
public let providerProtocol = "symphony.ssiag.provider.v1"
public let maximumRequestBytes = 65_536

public struct ProviderDescriptor: Codable, Sendable {
    public let identifier: String
    public let version: String
    public let platform: String
    public let protocolVersion: String
    public let status: String
    public let capabilities: [String]
    public let operationalAccessEnabled: Bool
    public let transport: String

    enum CodingKeys: String, CodingKey {
        case identifier, version, platform, status, capabilities, transport
        case protocolVersion = "protocol_version"
        case operationalAccessEnabled = "operational_access_enabled"
    }

    public static let scaffold = ProviderDescriptor(
        identifier: "macos-keychain",
        version: providerVersion,
        platform: "macos",
        protocolVersion: providerProtocol,
        status: "declared_not_operational",
        capabilities: ["metadata", "capability-discovery"],
        operationalAccessEnabled: false,
        transport: "stdio-jsonl"
    )
}

public struct ProviderResponse: Codable, Sendable {
    public let schema: String
    public let requestID: String
    public let outcome: String
    public let reasonCode: String
    public let descriptor: ProviderDescriptor?

    enum CodingKeys: String, CodingKey {
        case schema
        case requestID = "request_id"
        case outcome
        case reasonCode = "reason_code"
        case descriptor
    }
}

public enum ProtocolError: Error, CustomStringConvertible {
    case invalidJSON
    case invalidShape
    case unknownField
    case invalidSchema
    case invalidRequestID
    case unsupportedOperation

    public var description: String {
        switch self {
        case .invalidJSON: "invalid JSON"
        case .invalidShape: "invalid request shape"
        case .unknownField: "unknown request field"
        case .invalidSchema: "unsupported request schema"
        case .invalidRequestID: "invalid request identifier"
        case .unsupportedOperation: "unsupported operation"
        }
    }
}

public func response(for data: Data) throws -> ProviderResponse {
    guard data.count <= maximumRequestBytes else { throw ProtocolError.invalidShape }
    let value: Any
    do {
        value = try JSONSerialization.jsonObject(with: data)
    } catch {
        throw ProtocolError.invalidJSON
    }
    guard let object = value as? [String: Any] else { throw ProtocolError.invalidShape }
    let allowed = Set(["schema", "request_id", "operation"])
    guard Set(object.keys).isSubset(of: allowed), object.count == allowed.count else {
        throw ProtocolError.unknownField
    }
    guard object["schema"] as? String == "symphony.ssiag.provider.request.v1" else {
        throw ProtocolError.invalidSchema
    }
    guard let requestID = object["request_id"] as? String, validIdentifier(requestID) else {
        throw ProtocolError.invalidRequestID
    }
    guard let operation = object["operation"] as? String,
          ["hello", "status", "capabilities"].contains(operation) else {
        throw ProtocolError.unsupportedOperation
    }
    return ProviderResponse(
        schema: "symphony.ssiag.provider.response.v1",
        requestID: requestID,
        outcome: "ok",
        reasonCode: "provider.metadata_available",
        descriptor: .scaffold
    )
}

public func encodedLine(_ response: ProviderResponse) throws -> Data {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.sortedKeys]
    var data = try encoder.encode(response)
    data.append(0x0a)
    return data
}

private func validIdentifier(_ value: String) -> Bool {
    guard !value.isEmpty, value.utf8.count <= 128 else { return false }
    return value.unicodeScalars.allSatisfy {
        CharacterSet.alphanumerics.contains($0) || $0 == "-" || $0 == "_" || $0 == "."
    }
}
