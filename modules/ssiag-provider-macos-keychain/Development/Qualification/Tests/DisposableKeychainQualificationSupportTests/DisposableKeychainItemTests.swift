import Foundation
import LocalAuthentication
import Security
import XCTest
@testable import DisposableKeychainQualificationSupport

// LAContext can require system services just to accept a policy setting.
private final class TestAuthenticationContext: LAContext {
    var allowSettingPolicy = true
    private var noninteractive = false
    override var interactionNotAllowed: Bool {
        get { noninteractive }
        set { noninteractive = allowSettingPolicy && newValue }
    }
}

final class DisposableKeychainItemTests: XCTestCase {
    private let group = "EXAMPLETEAM1.org.example.qualification"
    private let topsID = UUID()
    private let itemID = UUID()
    private let first = Data(repeating: 1, count: 32)
    private let second = Data(repeating: 2, count: 32)

    func testEveryOperationUsesExactPrivateDeviceLocalItemWithoutInteraction() throws {
        var stored: Data?
        var queries = [[String: Any]]()
        let api = DisposableSecItemAPI(
            makeContext: { TestAuthenticationContext() },
            add: { attributes, output in
                let a = attributes as NSDictionary as! [String: Any]
                queries.append(a)
                XCTAssertNil(output)
                if stored != nil { return errSecDuplicateItem }
                stored = a[kSecValueData as String] as? Data
                return errSecSuccess
            },
            copyMatching: { query, output in
                let q = query as NSDictionary as! [String: Any]
                queries.append(q)
                XCTAssertEqual(q[kSecReturnData as String] as? Bool, true)
                XCTAssertEqual(q[kSecMatchLimit as String] as? String, kSecMatchLimitOne as String)
                guard let stored else { return errSecItemNotFound }
                output?.pointee = stored as CFData
                return errSecSuccess
            },
            update: { query, changes in
                queries.append(query as NSDictionary as! [String: Any])
                let attributes = changes as NSDictionary as! [String: Any]
                XCTAssertEqual(Set(attributes.keys), [kSecValueData as String])
                guard stored != nil else { return errSecItemNotFound }
                stored = attributes[kSecValueData as String] as? Data
                return errSecSuccess
            },
            delete: { query in
                queries.append(query as NSDictionary as! [String: Any])
                guard stored != nil else { return errSecItemNotFound }
                stored = nil
                return errSecSuccess
            }
        )
        let item = try DisposableKeychainItem(accessGroup: group, topsID: topsID, itemID: itemID, api: api)
        try item.create(first)
        XCTAssertThrowsError(try item.create(second)) { XCTAssertEqual($0 as? DisposableKeychainError, .duplicate) }
        XCTAssertTrue(try item.matches(first))
        try item.replace(with: second)
        XCTAssertFalse(try item.matches(first))
        XCTAssertTrue(try item.matches(second))
        try item.delete()
        XCTAssertThrowsError(try item.matches(second)) { XCTAssertEqual($0 as? DisposableKeychainError, .missing) }
        XCTAssertThrowsError(try item.replace(with: second)) { XCTAssertEqual($0 as? DisposableKeychainError, .missing) }
        XCTAssertEqual(queries.count, 9)
        for q in queries {
            XCTAssertEqual(q[kSecAttrAccessGroup as String] as? String, group)
            XCTAssertEqual(q[kSecAttrService as String] as? String,
                           "symphony.ssiag.qualification.generic-password.v1." + topsID.uuidString.lowercased())
            XCTAssertEqual(q[kSecAttrAccount as String] as? String, itemID.uuidString.lowercased())
            XCTAssertEqual(q[kSecClass as String] as? String, kSecClassGenericPassword as String)
            XCTAssertEqual(q[kSecUseDataProtectionKeychain as String] as? Bool, true)
            XCTAssertEqual(q[kSecAttrSynchronizable as String] as? Bool, false)
            XCTAssertEqual(q[kSecAttrAccessible as String] as? String, kSecAttrAccessibleWhenUnlockedThisDeviceOnly as String)
            XCTAssertEqual((q[kSecUseAuthenticationContext as String] as? LAContext)?.interactionNotAllowed, true)
        }
    }

    func testRefusesInvalidInputsBeforeCallingSecurity() throws {
        let api = DisposableSecItemAPI(
            makeContext: { TestAuthenticationContext() },
            add: { _, _ in XCTFail(); return errSecParam }, copyMatching: { _, _ in XCTFail(); return errSecParam },
            update: { _, _ in XCTFail(); return errSecParam }, delete: { _ in XCTFail(); return errSecParam }
        )
        for invalidGroup in ["", "single", "team.*", ".app", "team.", "a..b", "a.\0b", String(repeating: "x", count: 256) + ".app"] {
            XCTAssertThrowsError(try DisposableKeychainItem(accessGroup: invalidGroup, topsID: topsID, itemID: itemID, api: api))
        }
        let item = try DisposableKeychainItem(accessGroup: group, topsID: topsID, itemID: itemID, api: api)
        for size in [0, 31, 33] {
            let bytes = Data(repeating: 0, count: size)
            XCTAssertThrowsError(try item.create(bytes))
            XCTAssertThrowsError(try item.matches(bytes))
            XCTAssertThrowsError(try item.replace(with: bytes))
        }
    }

    func testUnavailableInteractionPolicyRefusesBeforeSecurityCall() throws {
        let api = DisposableSecItemAPI(
            makeContext: {
                let context = TestAuthenticationContext()
                context.allowSettingPolicy = false
                return context
            },
            add: { _, _ in XCTFail(); return errSecParam }, copyMatching: { _, _ in XCTFail(); return errSecParam },
            update: { _, _ in XCTFail(); return errSecParam }, delete: { _ in XCTFail(); return errSecParam }
        )
        let item = try DisposableKeychainItem(accessGroup: group, topsID: topsID, itemID: itemID, api: api)
        XCTAssertThrowsError(try item.create(first)) { XCTAssertEqual($0 as? DisposableKeychainError, .interactionRequired) }
    }

    func testRejectsMalformedResultsAndDoesNotRetryWithAnotherStore() throws {
        var status: OSStatus = errSecSuccess
        var response: CFTypeRef?
        var calls = 0
        let api = DisposableSecItemAPI(
            makeContext: { TestAuthenticationContext() },
            add: { _, _ in XCTFail(); return errSecParam },
            copyMatching: { _, output in calls += 1; output?.pointee = response; return status },
            update: { _, _ in XCTFail(); return errSecParam }, delete: { _ in XCTFail(); return errSecParam }
        )
        let item = try DisposableKeychainItem(accessGroup: group, topsID: topsID, itemID: itemID, api: api)
        for invalid: CFTypeRef? in [nil, "wrong" as CFString, Data(repeating: 0, count: 31) as CFData] {
            response = invalid
            XCTAssertThrowsError(try item.matches(first)) { XCTAssertEqual($0 as? DisposableKeychainError, .unexpected) }
        }
        let errors: [(OSStatus, DisposableKeychainError)] = [
            (errSecMissingEntitlement, .accessDenied), (errSecInteractionNotAllowed, .interactionRequired),
            (errSecNotAvailable, .unavailable), (errSecParam, .unexpected)
        ]
        for (osStatus, error) in errors {
            status = osStatus
            XCTAssertThrowsError(try item.matches(first)) { XCTAssertEqual($0 as? DisposableKeychainError, error) }
        }
        XCTAssertEqual(calls, 3 + errors.count)
    }
}
