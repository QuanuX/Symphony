import Foundation
import Testing
@testable import SSIAGMacOSKeychainSupport

@Test func readinessLayersCannotEnableProviderBehavior() throws {
    let executable = try #require(Bundle.main.executableURL)
    let observation = SignedBundleReadiness.observe(executableURL: executable, inspectRunningSelf: true)
    #expect(observation.protocolVersion == signedBundleReadinessProtocol)
    #expect(observation.metadataOnly)
    #expect(!observation.authorizationDecisionMade)
    #expect(!observation.operationalEligibility.evaluated)
    #expect(observation.operationalEligibility.state == "disabled")
    #expect(!observation.operationalAccessEnabled)
    #expect(!observation.providerOperationsEnabled)
    #expect(!observation.secretChannelEnabled)
    #expect(observation.reasonCodes == observation.reasonCodes.sorted())
}

@Test func bareExecutableHasNoBundleOrPolicyAuthority() {
    let observation = SignedBundleReadiness.observe(executableURL: URL(fileURLWithPath: "/bin/true"))
    #expect(!observation.appLikeBundleObserved)
    #expect(observation.provisioningProfileFileState == "absent")
    #expect(observation.policyMatch.state == "not_configured")
    #expect(!observation.policyMatch.evaluated)
    #expect(observation.dynamicSignatureState == "not_evaluated")
    #expect(!observation.operationalAccessEnabled)
}

@Test func fabricatedBundleCannotBecomeStructuralOrPolicyAuthority() throws {
    let fixture = try ReadinessFixture(policy: #"identifier "com.example.not-this-bundle""#)
    defer { fixture.cleanup() }
    let observation = SignedBundleReadiness.observe(executableURL: fixture.executable)
    #expect(observation.appLikeBundleObserved)
    #expect(observation.provisioningProfileFileState == "regular_safe")
    #expect(observation.staticSignatureState != "valid")
    #expect(["mismatch", "unavailable"].contains(observation.policyMatch.state))
    #expect(observation.policyMatch.evaluated)
    #expect(!observation.operationalAccessEnabled)
}

@Test func invalidOwnedPolicyIsDistinctFromStructuralValidation() throws {
    let fixture = try ReadinessFixture(policyPayload: Data(#"{"protocol":"wrong","adapter_requirement":"anchor apple"}"#.utf8))
    defer { fixture.cleanup() }
    let observation = SignedBundleReadiness.observe(executableURL: fixture.executable)
    #expect(observation.policyMatch.state == "invalid")
    #expect(!observation.policyMatch.evaluated)
    #expect(observation.policyRequirementDigest == "not_applicable")
    #expect(!observation.authorizationDecisionMade)
}

@Test func unsafeProfileShapeIsReportedWithoutParsingProfilePayload() throws {
    let fixture = try ReadinessFixture(profileMode: 0o620)
    defer { fixture.cleanup() }
    let observation = SignedBundleReadiness.observe(executableURL: fixture.executable)
    #expect(observation.provisioningProfileFileState == "unsafe")
    #expect(!observation.operationalAccessEnabled)
}

@Test func readinessJSONContainsOnlySafeBoundedEvidence() throws {
    let observation = SignedBundleReadiness.observe(executableURL: URL(fileURLWithPath: "/bin/true"))
    let encoded = try JSONEncoder().encode(observation)
    let object = try #require(try JSONSerialization.jsonObject(with: encoded) as? [String: Any])
    for required in ["structural_validation", "policy_match", "operational_eligibility", "authorization_decision_made", "operational_access_enabled", "provider_operations_enabled", "secret_channel_enabled"] {
        #expect(object[required] != nil)
    }
    let text = String(decoding: encoded, as: UTF8.self)
    for forbidden in ["certificate", "profile_payload", "entitlements", "native_error", "requirement_text", "security_session_id", "secret_value", "provider_payload"] {
        #expect(!text.contains(forbidden))
    }
}

@Test func readinessSourcesContainNoKeychainItemOrKeyOperationCalls() throws {
    let tests = URL(fileURLWithPath: #filePath).deletingLastPathComponent()
    let sources = tests.deletingLastPathComponent().deletingLastPathComponent().appending(path: "Sources/SSIAGMacOSKeychainSupport")
    let files = try FileManager.default.contentsOfDirectory(at: sources, includingPropertiesForKeys: nil).filter { $0.pathExtension == "swift" }
    for file in files {
        let source = try String(contentsOf: file, encoding: .utf8)
        #expect(!source.contains("Sec" + "Item"))
        #expect(!source.contains("Sec" + "Key"))
    }
}

@Test func realProcessReadinessOperationHonorsDisabledBoundary() throws {
    let tests = URL(fileURLWithPath: #filePath).deletingLastPathComponent()
    let package = tests.deletingLastPathComponent().deletingLastPathComponent()
    let executable = package.appending(path: ".build/debug/\(providerExecutableName)")
    #expect(FileManager.default.isExecutableFile(atPath: executable.path))
    let process = Process()
    let output = Pipe()
    let errors = Pipe()
    process.executableURL = executable
    process.arguments = ["readiness"]
    process.standardOutput = output
    process.standardError = errors
    try process.run()
    process.waitUntilExit()
    let stdout = output.fileHandleForReading.readDataToEndOfFile()
    let stderr = errors.fileHandleForReading.readDataToEndOfFile()
    #expect(process.terminationStatus == 0)
    #expect(stderr.isEmpty)
    #expect(stdout.count <= maximumResponseBytes)
    #expect(stdout.last == 0x0a)
    let object = try #require(try JSONSerialization.jsonObject(with: Data(stdout.dropLast())) as? [String: Any])
    #expect(object["protocol"] as? String == signedBundleReadinessProtocol)
    #expect(signedBundleReadinessOperationID == "engop:symphony:ssiag.macos-keychain-provider.readiness.observe")
    #expect(object["metadata_only"] as? Bool == true)
    #expect(object["authorization_decision_made"] as? Bool == false)
    #expect(object["operational_access_enabled"] as? Bool == false)
    #expect(object["provider_operations_enabled"] as? Bool == false)
    #expect(object["secret_channel_enabled"] as? Bool == false)
    let eligibility = try #require(object["operational_eligibility"] as? [String: Any])
    #expect(eligibility["state"] as? String == "disabled")
    #expect(eligibility["evaluated"] as? Bool == false)
}

private final class ReadinessFixture {
    let root: URL
    let executable: URL

    convenience init(policy: String, profileMode: Int = 0o400) throws {
        let payload = try JSONSerialization.data(withJSONObject: [
            "protocol": "symphony.ssiag.macos-signing-policy.v1",
            "adapter_requirement": policy,
        ], options: [.sortedKeys])
        try self.init(policyPayload: payload, profileMode: profileMode)
    }

    init(policyPayload: Data? = nil, profileMode: Int = 0o400) throws {
        root = FileManager.default.temporaryDirectory.appending(path: "ssiag-readiness-\(UUID().uuidString)", directoryHint: .isDirectory)
        executable = root.appending(path: "Readiness.app/Contents/MacOS/\(providerExecutableName)")
        let info = root.appending(path: "Readiness.app/Contents/Info.plist")
        let profile = root.appending(path: "Readiness.app/Contents/embedded.provisionprofile")
        let policy = root.appending(path: "Readiness.app/Contents/Resources/ssiag-signing-policy.json")
        try FileManager.default.createDirectory(at: executable.deletingLastPathComponent(), withIntermediateDirectories: true)
        try FileManager.default.createDirectory(at: policy.deletingLastPathComponent(), withIntermediateDirectories: true)
        try Data("not executable".utf8).write(to: executable)
        try Data("plist".utf8).write(to: info)
        try Data("not a profile".utf8).write(to: profile)
        if let policyPayload { try policyPayload.write(to: policy) }
        try FileManager.default.setAttributes([.posixPermissions: 0o500], ofItemAtPath: executable.path)
        try FileManager.default.setAttributes([.posixPermissions: 0o400], ofItemAtPath: info.path)
        try FileManager.default.setAttributes([.posixPermissions: profileMode], ofItemAtPath: profile.path)
        if policyPayload != nil { try FileManager.default.setAttributes([.posixPermissions: 0o400], ofItemAtPath: policy.path) }
    }

    func cleanup() { try? FileManager.default.removeItem(at: root) }
}
