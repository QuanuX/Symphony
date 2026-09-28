import Darwin
import Foundation
import Security

// Internal prerequisite only. No control operation constructs these values.
// The owner must load the requirement from verified deployment policy, never
// from a peer's message. Compiling a requirement does not establish its authority
// or strength (tests deliberately use an isolated ad-hoc fixture requirement).
struct NativePeerPolicy {
    let uid: uid_t
    let gid: gid_t
    fileprivate let requirement: SecRequirement

    init(uid: uid_t, gid: gid_t, requirementText: String) throws {
        guard !requirementText.isEmpty, requirementText.utf8.count <= 4096,
              !requirementText.contains("\0") else { throw NativePeerFailure.invalidPolicy }
        var requirement: SecRequirement?
        guard SecRequirementCreateWithString(requirementText as CFString, SecCSFlags(rawValue: 0), &requirement) == errSecSuccess,
              let requirement else { throw NativePeerFailure.invalidPolicy }
        self.uid = uid
        self.gid = gid
        self.requirement = requirement
    }
}

enum NativePeerFailure: Error, Sendable {
    case invalidPolicy, invalidDescriptor, peerUnavailable, identityMismatch, signatureMismatch, closed
}

struct NativePeerIdentity: Equatable, Sendable {
    let uid: uid_t
    let gid: gid_t
    let pid: pid_t
}

/// Retains a private close-on-exec duplicate of one connected Unix stream and
/// its kernel audit token. It is a time-specific native code observation, NOT
/// a credential delivery pin. A process may exec or transfer descriptors after
/// any check; a future channel must authenticate the actual message boundary.
/// No descriptor, token, requirement, native error or credential is exported.
final class NativePeerSession: @unchecked Sendable {
    let identity: NativePeerIdentity
    private let lock = NSLock()
    private var descriptor: Int32
    private let token: Data
    private let policy: NativePeerPolicy

    private init(descriptor: Int32, token: Data, identity: NativePeerIdentity, policy: NativePeerPolicy) {
        self.descriptor = descriptor
        self.token = token
        self.identity = identity
        self.policy = policy
    }

    // Caller owns the supplied descriptor and must keep it stable during this
    // call. After duplication, caller closure/reuse cannot retarget this session.
    static func observe(descriptor: Int32, policy: NativePeerPolicy) throws -> NativePeerSession {
        guard descriptor >= 0 else { throw NativePeerFailure.invalidDescriptor }
        let owned = fcntl(descriptor, F_DUPFD_CLOEXEC, 3)
        guard owned >= 0 else { throw NativePeerFailure.invalidDescriptor }
        do {
            try validateSocket(owned)
            let (token, identity) = try kernelIdentity(owned)
            guard identity.pid > 1, identity.pid != getpid(), identity.uid == policy.uid, identity.gid == policy.gid else {
                throw NativePeerFailure.identityMismatch
            }
            try validateCode(token: token, requirement: policy.requirement)
            return NativePeerSession(descriptor: owned, token: token, identity: identity, policy: policy)
        } catch {
            Darwin.close(owned)
            throw error
        }
    }

    // Every observation resolves a fresh SecCode from the ORIGINAL kernel token.
    // Never fall back to PID, process path, newest code or a peer-supplied token.
    // A failure permanently closes this session; it cannot be refreshed to a new
    // identity. Thread safety does not make observation and later I/O atomic.
    func revalidate() -> NativePeerFailure? {
        lock.lock()
        defer { lock.unlock() }
        guard descriptor >= 0 else { return .closed }
        do {
            try Self.validateSocket(descriptor)
            let (current, observed) = try Self.kernelIdentity(descriptor)
            guard current == token, observed == identity else { throw NativePeerFailure.identityMismatch }
            try Self.validateCode(token: token, requirement: policy.requirement)
            return nil
        } catch {
            closeLocked()
            return error as? NativePeerFailure ?? .peerUnavailable
        }
    }

    func close() {
        lock.lock()
        defer { lock.unlock() }
        closeLocked()
    }

    deinit { if descriptor >= 0 { Darwin.close(descriptor) } }

    private func closeLocked() {
        if descriptor >= 0 { Darwin.close(descriptor); descriptor = -1 }
    }

    private static func validateSocket(_ fd: Int32) throws {
        var metadata = stat()
        var kind: Int32 = 0
        var length = socklen_t(MemoryLayout<Int32>.size)
        var address = sockaddr_storage()
        var addressLength = socklen_t(MemoryLayout<sockaddr_storage>.size)
        let connected = withUnsafeMutablePointer(to: &address) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) { getpeername(fd, $0, &addressLength) }
        }
        guard fstat(fd, &metadata) == 0, (metadata.st_mode & S_IFMT) == S_IFSOCK,
              getsockopt(fd, SOL_SOCKET, SO_TYPE, &kind, &length) == 0,
              length == MemoryLayout<Int32>.size, kind == SOCK_STREAM,
              connected == 0, address.ss_family == sa_family_t(AF_UNIX) else {
            throw NativePeerFailure.invalidDescriptor
        }
        var state = pollfd(fd: fd, events: Int16(POLLIN), revents: 0)
        guard poll(&state, 1, 0) >= 0, state.revents & Int16(POLLHUP | POLLERR | POLLNVAL) == 0 else {
            throw NativePeerFailure.peerUnavailable
        }
    }

    private static func kernelIdentity(_ fd: Int32) throws -> (Data, NativePeerIdentity) {
        var token = audit_token_t()
        var length = socklen_t(MemoryLayout<audit_token_t>.size)
        guard getsockopt(fd, SOL_LOCAL, LOCAL_PEERTOKEN, &token, &length) == 0,
              length == MemoryLayout<audit_token_t>.size else { throw NativePeerFailure.peerUnavailable }
        let bytes = withUnsafeBytes(of: token) { Data($0) }
        return (bytes, NativePeerIdentity(uid: audit_token_to_euid(token), gid: audit_token_to_egid(token), pid: audit_token_to_pid(token)))
    }

    private static func validateCode(token: Data, requirement: SecRequirement) throws {
        var code: SecCode?
        let attributes = [kSecGuestAttributeAudit as String: token] as CFDictionary
        guard SecCodeCopyGuestWithAttributes(nil, attributes, SecCSFlags(rawValue: 0), &code) == errSecSuccess,
              let code else { throw NativePeerFailure.peerUnavailable }
        guard SecCodeCheckValidity(code, SecCSFlags(rawValue: 0), requirement) == errSecSuccess else {
            throw NativePeerFailure.signatureMismatch
        }
    }
}
