import Foundation
import Testing
@testable import SSIAGMacOSKeychainSupport

@Test func metadataHandshakeIsSafeAndNonOperational() throws {
    let request = Data(#"{"schema":"symphony.ssiag.provider.request.v1","request_id":"req-1","operation":"hello"}"#.utf8)
    let result = try response(for: request)
    #expect(result.outcome == "ok")
    #expect(result.descriptor?.operationalAccessEnabled == false)
    #expect(result.descriptor?.capabilities.contains("metadata") == true)
}

@Test func unknownAndSecretShapedFieldsFailClosed() throws {
    let request = Data(#"{"schema":"symphony.ssiag.provider.request.v1","request_id":"req-1","operation":"hello","token":"forbidden"}"#.utf8)
    #expect(throws: ProtocolError.self) {
        try response(for: request)
    }
}

@Test func credentialOperationsRemainDisabled() throws {
    let request = Data(#"{"schema":"symphony.ssiag.provider.request.v1","request_id":"req-1","operation":"read-secret"}"#.utf8)
    #expect(throws: ProtocolError.self) {
        try response(for: request)
    }
}
