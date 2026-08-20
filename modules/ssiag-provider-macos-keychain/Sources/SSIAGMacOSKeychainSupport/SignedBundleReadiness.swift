import Darwin
import Foundation
import Security

public let signedBundleReadinessProtocol = "symphony.ssiag.provider-readiness-observation.v1"
public let signedBundleReadinessOperationID = "engop:symphony:ssiag.macos-keychain-provider.readiness.observe"

public struct ReadinessLayer: Codable, Sendable {
    public let state: String
    public let evaluated: Bool
    public let reasonCode: String

    enum CodingKeys: String, CodingKey {
        case state, evaluated
        case reasonCode = "reason_code"
    }
}

/// Safe observations only. This envelope contains no certificate, profile,
/// entitlement, native-error, requirement, security-session, or provider
/// payload. It cannot authorize an operation or enable Keychain access.
public struct SignedBundleReadinessObservation: Codable, Sendable {
    public let protocolVersion: String
    public let metadataOnly: Bool
    public let structuralValidation: ReadinessLayer
    public let policyMatch: ReadinessLayer
    public let operationalEligibility: ReadinessLayer
    public let appLikeBundleObserved: Bool
    public let provisioningProfileFileState: String
    public let staticSignatureState: String
    public let dynamicSignatureState: String
    public let signingIdentifier: String
    public let designatedRequirementDigest: String
    public let policyRequirementDigest: String
    public let securitySessionObserved: Bool
    public let securitySessionRoot: Bool
    public let securitySessionGraphical: Bool
    public let securitySessionTTY: Bool
    public let securitySessionRemote: Bool
    public let authorizationDecisionMade: Bool
    public let operationalAccessEnabled: Bool
    public let providerOperationsEnabled: Bool
    public let secretChannelEnabled: Bool
    public let reasonCodes: [String]

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case metadataOnly = "metadata_only"
        case structuralValidation = "structural_validation"
        case policyMatch = "policy_match"
        case operationalEligibility = "operational_eligibility"
        case appLikeBundleObserved = "app_like_bundle_observed"
        case provisioningProfileFileState = "provisioning_profile_file_state"
        case staticSignatureState = "static_signature_state"
        case dynamicSignatureState = "dynamic_signature_state"
        case signingIdentifier = "signing_identifier"
        case designatedRequirementDigest = "designated_requirement_digest"
        case policyRequirementDigest = "policy_requirement_digest"
        case securitySessionObserved = "security_session_observed"
        case securitySessionRoot = "security_session_root"
        case securitySessionGraphical = "security_session_graphical"
        case securitySessionTTY = "security_session_tty"
        case securitySessionRemote = "security_session_remote"
        case authorizationDecisionMade = "authorization_decision_made"
        case operationalAccessEnabled = "operational_access_enabled"
        case providerOperationsEnabled = "provider_operations_enabled"
        case secretChannelEnabled = "secret_channel_enabled"
        case reasonCodes = "reason_codes"
    }
}

public enum SignedBundleReadiness {
    public static func observe(
        executableURL: URL,
        inspectRunningSelf: Bool = false
    ) -> SignedBundleReadinessObservation {
        let executable = executableURL.standardizedFileURL
        let bundle = appLikeBundleRoot(for: executable)
        let profileState = bundle.map(provisioningProfileFileState) ?? "absent"
        let structural = observeStaticSignature(at: bundle ?? executable)
        let dynamicState = inspectRunningSelf ? observeDynamicSelfSignature() : "not_evaluated"
        let policy = bundle.map { evaluateOwnedPolicy(bundle: $0, codeURL: $0) }
            ?? PolicyObservation(state: "not_configured", evaluated: false, reason: "symphony.ssiag.provider.readiness.policy_not_configured", digest: "not_applicable")
        let session = observeSecuritySession()

        let structuralLayer = ReadinessLayer(
            state: structural.state == "valid" ? "valid" : structural.state,
            evaluated: true,
            reasonCode: "symphony.ssiag.provider.readiness.structural_\(structural.state)"
        )
        let policyLayer = ReadinessLayer(state: policy.state, evaluated: policy.evaluated, reasonCode: policy.reason)
        let eligibilityLayer = ReadinessLayer(
            state: "disabled", evaluated: false,
            reasonCode: "symphony.ssiag.provider.readiness.phase_10b_operational_gate"
        )
        var reasons = [
            structuralLayer.reasonCode, policyLayer.reasonCode, eligibilityLayer.reasonCode,
            bundle == nil ? "symphony.ssiag.provider.readiness.app_bundle_absent" : "symphony.ssiag.provider.readiness.app_bundle_observed",
            "symphony.ssiag.provider.readiness.provisioning_profile_\(profileState)",
            session.observed ? "symphony.ssiag.provider.readiness.security_session_observed" : "symphony.ssiag.provider.readiness.security_session_unavailable",
        ]
        reasons.sort()

        return SignedBundleReadinessObservation(
            protocolVersion: signedBundleReadinessProtocol,
            metadataOnly: true,
            structuralValidation: structuralLayer,
            policyMatch: policyLayer,
            operationalEligibility: eligibilityLayer,
            appLikeBundleObserved: bundle != nil,
            provisioningProfileFileState: profileState,
            staticSignatureState: structural.state,
            dynamicSignatureState: dynamicState,
            signingIdentifier: structural.signingIdentifier,
            designatedRequirementDigest: structural.designatedRequirementDigest,
            policyRequirementDigest: policy.digest,
            securitySessionObserved: session.observed,
            securitySessionRoot: session.root,
            securitySessionGraphical: session.graphical,
            securitySessionTTY: session.tty,
            securitySessionRemote: session.remote,
            authorizationDecisionMade: false,
            operationalAccessEnabled: false,
            providerOperationsEnabled: false,
            secretChannelEnabled: false,
            reasonCodes: reasons
        )
    }
}

private struct StaticObservation {
    let state: String
    let signingIdentifier: String
    let designatedRequirementDigest: String
}

private struct PolicyObservation {
    let state: String
    let evaluated: Bool
    let reason: String
    let digest: String
}

private struct SessionObservation {
    let observed: Bool
    let root: Bool
    let graphical: Bool
    let tty: Bool
    let remote: Bool
}

private func appLikeBundleRoot(for executable: URL) -> URL? {
    let macOS = executable.deletingLastPathComponent()
    let contents = macOS.deletingLastPathComponent()
    let bundle = contents.deletingLastPathComponent()
    guard macOS.lastPathComponent == "MacOS", contents.lastPathComponent == "Contents",
          bundle.lastPathComponent.hasSuffix(".app") else { return nil }
    var metadata = stat()
    guard lstat(bundle.path, &metadata) == 0, (metadata.st_mode & S_IFMT) == S_IFDIR else { return nil }
    return bundle
}

private func provisioningProfileFileState(_ bundle: URL) -> String {
    let profile = bundle.appending(path: "Contents/embedded.provisionprofile")
    var metadata = stat()
    if lstat(profile.path, &metadata) != 0 { return errno == ENOENT ? "absent" : "unavailable" }
    guard (metadata.st_mode & S_IFMT) == S_IFREG, metadata.st_size > 0,
          metadata.st_size <= 16 * 1_048_576, (metadata.st_mode & 0o022) == 0 else { return "unsafe" }
    return "regular_safe"
}

private func observeStaticSignature(at url: URL) -> StaticObservation {
    let noFlags = SecCSFlags(rawValue: 0)
    var code: SecStaticCode?
    guard SecStaticCodeCreateWithPath(url as CFURL, noFlags, &code) == errSecSuccess, let code else {
        return emptyStaticObservation("unavailable")
    }
    let validateCompleteBundle = SecCSFlags(rawValue: (1 << 0) | (1 << 3) | (1 << 4))
    guard SecStaticCodeCheckValidity(code, validateCompleteBundle, nil) == errSecSuccess else {
        return emptyStaticObservation("invalid")
    }
    return StaticObservation(
        // Exact identity is evaluated by the native SecRequirement below.
        // We intentionally do not request the broad signing-information
        // dictionary merely to copy a display identifier.
        state: "valid", signingIdentifier: "not_observed",
        designatedRequirementDigest: requirementDigest(for: code)
    )
}

private func observeDynamicSelfSignature() -> String {
    let flags = SecCSFlags(rawValue: 0)
    var code: SecCode?
    guard SecCodeCopySelf(flags, &code) == errSecSuccess, let code else { return "unavailable" }
    return SecCodeCheckValidity(code, flags, nil) == errSecSuccess ? "valid" : "invalid"
}

private func evaluateOwnedPolicy(bundle: URL, codeURL: URL) -> PolicyObservation {
    let policyURL = bundle.appending(path: "Contents/Resources/ssiag-signing-policy.json")
    var metadata = stat()
    if lstat(policyURL.path, &metadata) != 0 {
        return PolicyObservation(state: "not_configured", evaluated: false, reason: "symphony.ssiag.provider.readiness.policy_not_configured", digest: "not_applicable")
    }
    guard (metadata.st_mode & S_IFMT) == S_IFREG, metadata.st_size > 0, metadata.st_size <= 16_384,
          (metadata.st_mode & 0o022) == 0,
          let data = try? Data(contentsOf: policyURL),
          let object = try? strictJSONObject(data),
          Set(object.keys) == ["protocol", "adapter_requirement"],
          object["protocol"] as? String == "symphony.ssiag.macos-signing-policy.v1",
          let text = object["adapter_requirement"] as? String,
          !text.isEmpty, text.utf8.count <= 4096, !text.contains("\0") else {
        return PolicyObservation(state: "invalid", evaluated: false, reason: "symphony.ssiag.provider.readiness.policy_invalid", digest: "not_applicable")
    }
    let flags = SecCSFlags(rawValue: 0)
    var requirement: SecRequirement?
    guard SecRequirementCreateWithString(text as CFString, flags, &requirement) == errSecSuccess, let requirement else {
        return PolicyObservation(state: "invalid", evaluated: false, reason: "symphony.ssiag.provider.readiness.policy_invalid", digest: "not_applicable")
    }
    let digest = requirementDigest(requirement)
    var code: SecStaticCode?
    guard SecStaticCodeCreateWithPath(codeURL as CFURL, flags, &code) == errSecSuccess, let code else {
        return PolicyObservation(state: "unavailable", evaluated: true, reason: "symphony.ssiag.provider.readiness.policy_unavailable", digest: digest)
    }
    let result = SecStaticCodeCheckValidity(code, SecCSFlags(rawValue: (1 << 0) | (1 << 3) | (1 << 4)), requirement)
    return result == errSecSuccess
        ? PolicyObservation(state: "matched", evaluated: true, reason: "symphony.ssiag.provider.readiness.policy_matched", digest: digest)
        : PolicyObservation(state: "mismatch", evaluated: true, reason: "symphony.ssiag.provider.readiness.policy_mismatch", digest: digest)
}

private func requirementDigest(for code: SecStaticCode) -> String {
    var requirement: SecRequirement?
    let flags = SecCSFlags(rawValue: 0)
    guard SecCodeCopyDesignatedRequirement(code, flags, &requirement) == errSecSuccess, let requirement else {
        return "not_applicable"
    }
    return requirementDigest(requirement)
}

private func requirementDigest(_ requirement: SecRequirement) -> String {
    var bytes: CFData?
    guard SecRequirementCopyData(requirement, SecCSFlags(rawValue: 0), &bytes) == errSecSuccess, let bytes else {
        return "not_applicable"
    }
    return sha256Digest(bytes as Data)
}

private func emptyStaticObservation(_ state: String) -> StaticObservation {
    StaticObservation(state: state, signingIdentifier: "not_applicable", designatedRequirementDigest: "not_applicable")
}

private func observeSecuritySession() -> SessionObservation {
    var identifier: SecuritySessionId = 0
    var attributes = SessionAttributeBits()
    guard SessionGetInfo(callerSecuritySession, &identifier, &attributes) == errSessionSuccess, identifier != 0 else {
        return SessionObservation(observed: false, root: false, graphical: false, tty: false, remote: false)
    }
    let bits = attributes.rawValue
    return SessionObservation(
        observed: true,
        root: bits & 0x0001 != 0,
        graphical: bits & 0x0010 != 0,
        tty: bits & 0x0020 != 0,
        remote: bits & 0x1000 != 0
    )
}
