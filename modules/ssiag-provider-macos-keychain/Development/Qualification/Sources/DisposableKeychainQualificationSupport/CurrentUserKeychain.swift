import Darwin
import Foundation
import Security

public enum CurrentUserKeychainFailure: String, Error {
    case requiresUser, randomUnavailable, temporaryDirectory, noninteractivePolicy
    case userPreferences, preferencesChanged
    case create, insert, duplicateRefusal, read, mismatch, replace, missingRefusal, delete, cleanup
}

// A separate test of Apple's file-based Keychain. All item operations target a
// concrete disposable keychain or the item handle returned by adding to it.
public enum CurrentUserKeychain {
    public static func exercise() throws {
        guard getuid() != 0, geteuid() == getuid() else { throw CurrentUserKeychainFailure.requiresUser }
        guard SecKeychainSetUserInteractionAllowed(false) == errSecSuccess else {
            throw CurrentUserKeychainFailure.noninteractivePolicy
        }
        let preferences = try UserPreferences()
        var password = try randomValue().base64EncodedData()
        defer { password.resetBytes(in: password.indices) }
        var first = try randomValue()
        defer { first.resetBytes(in: first.indices) }
        var second = try randomValue()
        defer { second.resetBytes(in: second.indices) }
        guard first != second else { throw CurrentUserKeychainFailure.randomUnavailable }

        var template = Array("/private/tmp/symphony-user-keychain.XXXXXX".utf8CString)
        guard let path = mkdtemp(&template) else { throw CurrentUserKeychainFailure.temporaryDirectory }
        let directory = URL(fileURLWithPath: String(cString: path), isDirectory: true)
        let outcome = Result {
            try run(in: directory, password: password, first: first, second: second, preferences: preferences)
        }
        do { try FileManager.default.removeItem(at: directory) } catch { throw CurrentUserKeychainFailure.cleanup }
        try preferences.verify()
        try outcome.get()
    }

    // Compare only file paths for the default and ordered search list. No
    // item query is ever issued against a keychain in these preferences.
    private struct UserPreferences: Equatable {
        let defaultKeychain: Data?
        let searchList: [Data]

        init() throws {
            var defaultRef: SecKeychain?
            let status = SecKeychainCopyDomainDefault(.user, &defaultRef)
            if status == errSecNoDefaultKeychain {
                defaultKeychain = nil
            } else {
                guard status == errSecSuccess, let defaultRef else { throw CurrentUserKeychainFailure.userPreferences }
                defaultKeychain = try Self.path(of: defaultRef)
            }
            var refs: CFArray?
            guard SecKeychainCopyDomainSearchList(.user, &refs) == errSecSuccess,
                  let keychains = refs as? [SecKeychain] else { throw CurrentUserKeychainFailure.userPreferences }
            searchList = try keychains.map(Self.path)
        }

        private static func path(of keychain: SecKeychain) throws -> Data {
            var buffer = [CChar](repeating: 0, count: Int(PATH_MAX))
            var count = UInt32(buffer.count)
            guard SecKeychainGetPath(keychain, &count, &buffer) == errSecSuccess,
                  count > 0, count <= buffer.count else { throw CurrentUserKeychainFailure.userPreferences }
            return buffer.withUnsafeBytes { Data($0.prefix(Int(count))) }
        }

        func verify() throws {
            guard try UserPreferences() == self else { throw CurrentUserKeychainFailure.preferencesChanged }
        }
    }

    private static func randomValue() throws -> Data {
        var result = Data(count: 32)
        let status = result.withUnsafeMutableBytes { SecRandomCopyBytes(kSecRandomDefault, $0.count, $0.baseAddress!) }
        guard status == errSecSuccess else { throw CurrentUserKeychainFailure.randomUnavailable }
        return result
    }

    private static func run(in directory: URL, password: Data, first: Data, second: Data,
                            preferences: UserPreferences) throws {
        var ref: SecKeychain?
        let path = directory.appendingPathComponent("disposable.keychain").path
        let created = password.withUnsafeBytes {
            SecKeychainCreate(path, UInt32($0.count), $0.baseAddress, false, nil, &ref)
        }
        guard created == errSecSuccess, let keychain = ref else { throw CurrentUserKeychainFailure.create }
        let outcome = Result {
            try preferences.verify()
            try exerciseItem(in: keychain, first: first, second: second)
        }
        guard SecKeychainDelete(keychain) == errSecSuccess else { throw CurrentUserKeychainFailure.cleanup }
        try outcome.get()
    }

    private static func exerciseItem(in keychain: SecKeychain, first: Data, second: Data) throws {
        let service = "org.quanux.symphony.file-keychain-qualification"
        let account = UUID().uuidString
        var ref: SecKeychainItem?
        func add(_ data: Data, item: UnsafeMutablePointer<SecKeychainItem?>?) -> OSStatus {
            data.withUnsafeBytes {
                SecKeychainAddGenericPassword(keychain, UInt32(service.utf8.count), service,
                    UInt32(account.utf8.count), account, UInt32($0.count), $0.baseAddress!, item)
            }
        }
        func find(_ account: String, _ length: UnsafeMutablePointer<UInt32>? = nil,
                  _ bytes: UnsafeMutablePointer<UnsafeMutableRawPointer?>? = nil) -> OSStatus {
            SecKeychainFindGenericPassword(keychain, UInt32(service.utf8.count), service,
                                          UInt32(account.utf8.count), account, length, bytes, nil)
        }
        func expectValue(_ expected: Data) throws {
            var length: UInt32 = 0
            var bytes: UnsafeMutableRawPointer?
            guard find(account, &length, &bytes) == errSecSuccess, let bytes else { throw CurrentUserKeychainFailure.read }
            defer { SecKeychainItemFreeContent(nil, bytes) }
            guard length == expected.count,
                  expected.withUnsafeBytes({ memcmp(bytes, $0.baseAddress!, $0.count) == 0 }) else {
                throw CurrentUserKeychainFailure.mismatch
            }
        }

        guard add(first, item: &ref) == errSecSuccess, let item = ref else { throw CurrentUserKeychainFailure.insert }
        guard add(second, item: nil) == errSecDuplicateItem else { throw CurrentUserKeychainFailure.duplicateRefusal }
        guard find(UUID().uuidString) == errSecItemNotFound else { throw CurrentUserKeychainFailure.missingRefusal }
        try expectValue(first)
        let updated = second.withUnsafeBytes {
            SecKeychainItemModifyAttributesAndData(item, nil, UInt32($0.count), $0.baseAddress)
        }
        guard updated == errSecSuccess else { throw CurrentUserKeychainFailure.replace }
        try expectValue(second)
        guard SecKeychainItemDelete(item) == errSecSuccess else { throw CurrentUserKeychainFailure.delete }
        guard find(account) == errSecItemNotFound else { throw CurrentUserKeychainFailure.missingRefusal }
    }
}
