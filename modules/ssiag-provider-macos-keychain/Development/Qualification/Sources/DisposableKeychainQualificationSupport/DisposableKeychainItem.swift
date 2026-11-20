import Foundation
import LocalAuthentication
import Security

public enum DisposableKeychainError: String, Error {
    case invalidInput, duplicate, missing, accessDenied, interactionRequired, unavailable, unexpected
}

// The qualification executable supplies a private group from its signed identity and
// fresh UUIDs/values generated in-process. No production credential belongs in this namespace.
public struct DisposableKeychainItem {
    private let group: String
    private let service: String
    private let account: String
    private let api: DisposableSecItemAPI

    public init(accessGroup: String, topsID: UUID, itemID: UUID) throws {
        try self.init(accessGroup: accessGroup, topsID: topsID, itemID: itemID, api: .system)
    }

    init(accessGroup: String, topsID: UUID, itemID: UUID, api: DisposableSecItemAPI) throws {
        let components = accessGroup.split(separator: ".", omittingEmptySubsequences: false)
        guard accessGroup.utf8.count <= 255, components.count >= 2,
              components.allSatisfy({ !$0.isEmpty && $0.utf8.allSatisfy {
                  (65...90).contains($0) || (97...122).contains($0) || (48...57).contains($0) || $0 == 45
              } }) else {
            throw DisposableKeychainError.invalidInput
        }
        self.group = accessGroup
        self.service = "symphony.ssiag.qualification.generic-password.v1." + topsID.uuidString.lowercased()
        self.account = itemID.uuidString.lowercased()
        self.api = api
    }

    // The same exact primary key scopes creation, lookup, replacement and cleanup.
    private func selector() throws -> [String: Any] {
        let context = api.makeContext()
        context.interactionNotAllowed = true
        guard context.interactionNotAllowed else { throw DisposableKeychainError.interactionRequired }
        return [
            kSecUseDataProtectionKeychain as String: true,
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccessGroup as String: group,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecAttrSynchronizable as String: false,
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
            kSecUseAuthenticationContext as String: context,
        ]
    }

    public func create(_ value: Data) throws {
        guard value.count == 32 else { throw DisposableKeychainError.invalidInput }
        var attributes = try selector()
        attributes[kSecValueData as String] = value
        try check(api.add(attributes as CFDictionary, nil))
    }

    public func matches(_ value: Data) throws -> Bool {
        guard value.count == 32 else { throw DisposableKeychainError.invalidInput }
        var query = try selector()
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        query[kSecReturnData as String] = true
        var result: CFTypeRef?
        try check(api.copyMatching(query as CFDictionary, &result))
        guard let bytes = result as? Data, bytes.count == 32 else { throw DisposableKeychainError.unexpected }
        return bytes == value
    }

    public func replace(with value: Data) throws {
        guard value.count == 32 else { throw DisposableKeychainError.invalidInput }
        let changes = [kSecValueData as String: value]
        try check(api.update(selector() as CFDictionary, changes as CFDictionary))
    }

    public func delete() throws {
        try check(api.delete(selector() as CFDictionary))
    }

    private func check(_ status: OSStatus) throws {
        switch status {
        case errSecSuccess: return
        case errSecDuplicateItem: throw DisposableKeychainError.duplicate
        case errSecItemNotFound: throw DisposableKeychainError.missing
        case errSecMissingEntitlement, errSecAuthFailed: throw DisposableKeychainError.accessDenied
        case errSecInteractionNotAllowed, errSecUserCanceled: throw DisposableKeychainError.interactionRequired
        case errSecNotAvailable: throw DisposableKeychainError.unavailable
        default: throw DisposableKeychainError.unexpected
        }
    }
}

// Tests capture the actual Security.framework dictionaries without opening a user's Keychain.
struct DisposableSecItemAPI {
    var makeContext: () -> LAContext = { LAContext() }
    var add: (CFDictionary, UnsafeMutablePointer<CFTypeRef?>?) -> OSStatus
    var copyMatching: (CFDictionary, UnsafeMutablePointer<CFTypeRef?>?) -> OSStatus
    var update: (CFDictionary, CFDictionary) -> OSStatus
    var delete: (CFDictionary) -> OSStatus

    static var system: Self {
        Self(add: SecItemAdd, copyMatching: SecItemCopyMatching, update: SecItemUpdate, delete: SecItemDelete)
    }
}
