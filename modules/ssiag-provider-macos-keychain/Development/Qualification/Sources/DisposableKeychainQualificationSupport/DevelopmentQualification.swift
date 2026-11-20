import Darwin
import Foundation
import Security
import SystemConfiguration

public enum QualificationFailure: String, Error {
    case invalidSelection, wrongBundle, signatureRejected, privateGroupUnavailable, loginRequired
    case randomUnavailable, mismatch, cleanupFailed
}

// Only accept an explicitly selected development team and a dedicated QA app ID.
// Validating ASCII here also makes interpolation into a native code requirement unambiguous.
public struct DevelopmentQualification {
    private let teamID: String
    private let bundleID: String

    public init(teamID: String, bundleID: String) throws {
        let teamPattern = "^[A-Z0-9]{10}$"
        guard teamID.utf8.count == 10, teamID.range(of: teamPattern, options: .regularExpression) != nil,
              bundleID.utf8.count <= 200, bundleID.hasSuffix(".symphony-keychain-check"),
              bundleID.split(separator: ".", omittingEmptySubsequences: false).allSatisfy({
                  !$0.isEmpty && $0.utf8.allSatisfy { (65...90).contains($0) || (97...122).contains($0) || (48...57).contains($0) || $0 == 45 }
              }) else { throw QualificationFailure.invalidSelection }
        self.teamID = teamID
        self.bundleID = bundleID
    }

    public func preflight() throws -> String {
        guard Bundle.main.bundleURL.pathExtension == "app", Bundle.main.bundleIdentifier == bundleID else {
            throw QualificationFailure.wrongBundle
        }
        let flags = SecCSFlags(rawValue: 0)
        let nativeRule = "anchor apple generic and certificate leaf[field.1.2.840.113635.100.6.1.12] exists and certificate leaf[subject.OU] = \"\(teamID)\" and identifier \"\(bundleID)\""
        var rule: SecRequirement?
        var code: SecCode?
        var staticCode: SecStaticCode?
        let completeBundle = SecCSFlags(rawValue: kSecCSCheckAllArchitectures | kSecCSCheckNestedCode | kSecCSStrictValidate)
        guard SecRequirementCreateWithString(nativeRule as CFString, flags, &rule) == errSecSuccess, let rule,
              SecCodeCopySelf(flags, &code) == errSecSuccess, let code,
              SecCodeCheckValidity(code, flags, rule) == errSecSuccess,
              SecCodeCopyStaticCode(code, flags, &staticCode) == errSecSuccess, let staticCode,
              SecStaticCodeCheckValidity(staticCode, completeBundle, rule) == errSecSuccess,
              let task = SecTaskCreateFromSelf(nil),
              SecTaskCopyValueForEntitlement(task, "com.apple.developer.team-identifier" as CFString, nil) as? String == teamID else {
            throw QualificationFailure.signatureRejected
        }
        let appID = SecTaskCopyValueForEntitlement(task, "com.apple.application-identifier" as CFString, nil) as? String
        let explicitGroups = SecTaskCopyValueForEntitlement(task, "keychain-access-groups" as CFString, nil)
        guard let group = Self.privateGroup(appID: appID, bundleID: bundleID, explicitGroups: explicitGroups) else {
            throw QualificationFailure.privateGroupUnavailable
        }
        var consoleUID: uid_t = 0
        var session: SecuritySessionId = 0
        var attributes = SessionAttributeBits()
        guard getuid() != 0, getuid() == geteuid(),
              SCDynamicStoreCopyConsoleUser(nil, &consoleUID, nil) != nil, getuid() == consoleUID,
              SessionGetInfo(callerSecuritySession, &session, &attributes) == errSessionSuccess, session != 0,
              attributes.contains(.sessionHasGraphicAccess),
              !attributes.contains(.sessionIsRemote), !attributes.contains(.sessionIsRoot) else {
            throw QualificationFailure.loginRequired
        }
        return group
    }

    static func privateGroup(appID: String?, bundleID: String, explicitGroups: Any? = nil) -> String? {
        guard let appID, let dot = appID.firstIndex(of: "."),
              appID[appID.index(after: dot)...] == bundleID,
              explicitGroups == nil || (explicitGroups as? [String]) == [appID] else { return nil }
        let prefix = appID[..<dot].utf8
        guard prefix.count == 10, prefix.allSatisfy({ (65...90).contains($0) || (48...57).contains($0) }) else { return nil }
        return appID
    }

    // Identifiers and test data are always freshly generated inside the process.
    public func exercise() throws {
        let group = try preflight()
        let item = try DisposableKeychainItem(accessGroup: group, topsID: UUID(), itemID: UUID())
        var first = try Self.makeValue()
        defer { first.resetBytes(in: first.startIndex..<first.endIndex) }
        var second = try Self.makeValue()
        defer { second.resetBytes(in: second.startIndex..<second.endIndex) }
        guard first != second else { throw QualificationFailure.randomUnavailable }
        try exerciseOwnedItem(item, first: first, second: second)
    }

    private static func makeValue() throws -> Data {
        var value = Data(count: 32)
        let status = value.withUnsafeMutableBytes { bytes in
            SecRandomCopyBytes(kSecRandomDefault, bytes.count, bytes.baseAddress!)
        }
        guard status == errSecSuccess else { throw QualificationFailure.randomUnavailable }
        return value
    }
}

protocol QualificationItem {
    func create(_ value: Data) throws
    func matches(_ value: Data) throws -> Bool
    func replace(with value: Data) throws
    func delete() throws
}
extension DisposableKeychainItem: QualificationItem {}

func exerciseOwnedItem(_ item: any QualificationItem, first: Data, second: Data) throws {
    try item.create(first)
    do {
        guard try item.matches(first), try !item.matches(second) else { throw QualificationFailure.mismatch }
        try item.replace(with: second)
        guard try item.matches(second), try !item.matches(first) else { throw QualificationFailure.mismatch }
    } catch {
        try deleteAndConfirmAbsence(item, value: second)
        throw error
    }
    try deleteAndConfirmAbsence(item, value: second)
}

private func deleteAndConfirmAbsence(_ item: any QualificationItem, value: Data) throws {
    do { try item.delete() } catch { throw QualificationFailure.cleanupFailed }
    do {
        _ = try item.matches(value)
    } catch DisposableKeychainError.missing {
        return
    } catch {
        throw QualificationFailure.cleanupFailed
    }
    throw QualificationFailure.cleanupFailed
}
