import Foundation
import Testing
@testable import DisposableKeychainQualificationSupport

private let bundle = "org.example.unique.symphony-keychain-check"
private let first = Data(repeating: 1, count: 32)
private let second = Data(repeating: 2, count: 32)

@Test func requiresDedicatedBundleAndExactTeam() throws {
    _ = try DevelopmentQualification(teamID: "AB1CD2EF34", bundleID: bundle)
    for team in ["", "ab1cd2ef34", "AB1CD2 EF3", "ZZZZZZZZZ\n"] {
        #expect(throws: QualificationFailure.invalidSelection) {
            try DevelopmentQualification(teamID: team, bundleID: bundle)
        }
    }
    for bundle in ["org.example.app", "org..symphony-keychain-check", "org.*.symphony-keychain-check", "\").symphony-keychain-check"] {
        #expect(throws: QualificationFailure.invalidSelection) {
            try DevelopmentQualification(teamID: "AB1CD2EF34", bundleID: bundle)
        }
    }
}

@Test func privateGroupRequiresExactEntitledAppID() {
    let appID = "AB12CD34EF." + bundle
    #expect(DevelopmentQualification.privateGroup(appID: appID, bundleID: bundle) == appID)
    #expect(DevelopmentQualification.privateGroup(appID: appID, bundleID: bundle, explicitGroups: [appID]) == appID)
    #expect(DevelopmentQualification.privateGroup(appID: appID, bundleID: bundle, explicitGroups: [appID, "other"]) == nil)
    #expect(DevelopmentQualification.privateGroup(appID: appID, bundleID: bundle, explicitGroups: "wrong type") == nil)
    for id in [nil, "", "*." + bundle, "AB12CD34EF.*", "AB12CD34EF.org.wrong", "ab12cd34ef." + bundle] {
        #expect(DevelopmentQualification.privateGroup(appID: id, bundleID: bundle) == nil)
    }
}

private final class MemoryItem: QualificationItem {
    var value: Data?
    var refuseReplacement = false
    var failRemoval = false
    var ignoreRemoval = false
    var deletes = 0

    func create(_ value: Data) throws {
        guard self.value == nil else { throw DisposableKeychainError.duplicate }
        self.value = value
    }

    func matches(_ value: Data) throws -> Bool {
        guard let stored = self.value else { throw DisposableKeychainError.missing }
        return value == stored
    }

    func replace(with value: Data) throws {
        if refuseReplacement { throw DisposableKeychainError.accessDenied }
        self.value = value
    }

    func delete() throws {
        deletes += 1
        if failRemoval { throw DisposableKeychainError.accessDenied }
        if !ignoreRemoval { value = nil }
    }
}

@Test func provesReplacementAndAbsence() throws {
    let item = MemoryItem()
    try exerciseOwnedItem(item, first: first, second: second)
    #expect(item.deletes == 1)
    #expect(item.value == nil)
}

@Test func duplicateIsNeverOverwrittenOrDeleted() {
    let item = MemoryItem()
    item.value = second
    #expect(throws: DisposableKeychainError.duplicate) {
        try exerciseOwnedItem(item, first: first, second: second)
    }
    #expect(item.value == second)
    #expect(item.deletes == 0)
}

@Test func cleansUpAfterRefusedReplacement() {
    let item = MemoryItem()
    item.refuseReplacement = true
    #expect(throws: DisposableKeychainError.accessDenied) {
        try exerciseOwnedItem(item, first: first, second: second)
    }
    #expect(item.value == nil)
    #expect(item.deletes == 1)
}

@Test func cleanupFailureTakesPrecedence() {
    let item = MemoryItem()
    item.refuseReplacement = true
    item.failRemoval = true
    #expect(throws: QualificationFailure.cleanupFailed) {
        try exerciseOwnedItem(item, first: first, second: second)
    }
    #expect(item.deletes == 1)
}

@Test func requiresConfirmedAbsenceForSuccess() {
    let item = MemoryItem()
    item.ignoreRemoval = true
    #expect(throws: QualificationFailure.cleanupFailed) {
        try exerciseOwnedItem(item, first: first, second: second)
    }
}

@Test func confirmsAbsenceAfterRefusedReplacement() {
    let item = MemoryItem()
    item.refuseReplacement = true
    item.ignoreRemoval = true
    #expect(throws: QualificationFailure.cleanupFailed) {
        try exerciseOwnedItem(item, first: first, second: second)
    }
}
