import Darwin
import Foundation
import Security
import XPC

// Internal metadata transport profile. These values come from the trusted native
// owner after SSIAG admission, never from the incoming message as authority.
struct NativeMessageBinding: Sendable {
    let topsID: String
    let requestID: String
    let bindingDigest: String
    let challenge: String

    init(topsID: String, requestID: String, bindingDigest: String, challenge: String) throws {
        for value in [topsID, requestID, challenge] {
            guard UUID(uuidString: value)?.uuidString.lowercased() == value else { throw NativeMessageFailure.invalidBinding }
        }
        guard bindingDigest.utf8.count == 71, bindingDigest.hasPrefix("sha256:"),
              bindingDigest.dropFirst(7).utf8.allSatisfy({ (48...57).contains($0) || (97...102).contains($0) }) else {
            throw NativeMessageFailure.invalidBinding
        }
        self.topsID = topsID; self.requestID = requestID
        self.bindingDigest = bindingDigest; self.challenge = challenge
    }
}

struct NativeMessagePolicy {
    let uid: uid_t
    let gid: gid_t
    fileprivate let text: String
    fileprivate let requirement: SecRequirement

    init(uid: uid_t, gid: gid_t, requirementText: String) throws {
        guard !requirementText.isEmpty, requirementText.utf8.count <= 4096, !requirementText.contains("\0") else {
            throw NativeMessageFailure.invalidPolicy
        }
        var requirement: SecRequirement?
        guard SecRequirementCreateWithString(requirementText as CFString, SecCSFlags(rawValue: 0), &requirement) == errSecSuccess,
              let requirement else { throw NativeMessageFailure.invalidPolicy }
        self.uid = uid; self.gid = gid; self.text = requirementText; self.requirement = requirement
    }
}

enum NativeMessageFailure: Error { case invalidBinding, invalidPolicy, invalidTimeout, unavailable }
enum NativeMessageOutcome: String, Sendable { case matched, rejected, expired, cancelled, disconnected }

/// One-way, nonsecret, single-use message admission. Owns an INACTIVE accepted
/// XPC connection exclusively. The caller retains this object until completion.
/// Matching means only native code + exact metadata matched, not permission to
/// release credentials, authenticated FD delivery, or receiver acknowledgment.
final class NativeMessageGate: @unchecked Sendable {
    private let connection: xpc_connection_t
    private let queue = DispatchQueue(label: "symphony.ssiag.native-message")
    private let timer: DispatchSourceTimer
    private let policy: NativeMessagePolicy
    private let binding: NativeMessageBinding
    private let deadline: DispatchTime
    private let completion: @Sendable (NativeMessageOutcome) -> Void
    // Access only on queue. The installed handler and timer capture self weakly.
    private var finished = false

    init(connection: xpc_connection_t, policy: NativeMessagePolicy, binding: NativeMessageBinding,
         timeout: TimeInterval, completion: @escaping @Sendable (NativeMessageOutcome) -> Void) throws {
        guard timeout.isFinite, timeout > 0, timeout <= 30 else { throw NativeMessageFailure.invalidTimeout }
        guard xpc_get_type(connection) == XPC_TYPE_CONNECTION else { throw NativeMessageFailure.unavailable }
        self.connection = connection; self.policy = policy; self.binding = binding; self.completion = completion
        self.deadline = .now() + timeout
        self.timer = DispatchSource.makeTimerSource(queue: queue)
        // From here this object owns connection lifecycle, including setup error.
        xpc_connection_set_target_queue(connection, queue)
        let configured = policy.text.withCString { xpc_connection_set_peer_code_signing_requirement(connection, $0) }
        guard configured == 0 else {
            xpc_connection_set_event_handler(connection) { _ in }
            xpc_connection_activate(connection)
            xpc_connection_cancel(connection)
            timer.setEventHandler {}
            timer.activate()
            timer.cancel()
            throw NativeMessageFailure.unavailable
        }
        xpc_connection_set_event_handler(connection) { [weak self] message in self?.receive(message) }
        timer.setEventHandler { [weak self] in self?.finish(.expired) }
        timer.schedule(deadline: deadline)
        timer.activate()
        xpc_connection_activate(connection)
    }

    func cancel() { queue.async { [weak self] in self?.finish(.cancelled) } }

    deinit {
        timer.cancel()
        xpc_connection_cancel(connection)
    }

    private func receive(_ message: xpc_object_t) {
        guard !finished else { return }
        guard DispatchTime.now() < deadline else { finish(.expired); return }
        guard xpc_get_type(message) == XPC_TYPE_DICTIONARY else { finish(.disconnected); return }
        // Connection UID/GID are CONNECT-TIME credentials, not proof of the
        // sender's current UID/GID. Per-message code comes from the message's
        // audit token below. This profile does not claim OS-principal continuity
        // after connection transfer or credential changes.
        guard xpc_connection_get_euid(connection) == policy.uid,
              xpc_connection_get_egid(connection) == policy.gid else { finish(.rejected); return }
        var sender: SecCode?
        guard SecCodeCreateWithXPCMessage(message, SecCSFlags(rawValue: 0), &sender) == errSecSuccess,
              let sender, SecCodeCheckValidity(sender, SecCSFlags(rawValue: 0), policy.requirement) == errSecSuccess else {
            finish(.rejected); return
        }
        let expected = ["format": "ssiag-internal-native-message-1", "tops_id": binding.topsID,
                        "request_id": binding.requestID, "binding_digest": binding.bindingDigest, "challenge": binding.challenge]
        guard xpc_dictionary_get_count(message) == expected.count else { finish(.rejected); return }
        for (key, value) in expected {
            guard let field = xpc_dictionary_get_value(message, key), xpc_get_type(field) == XPC_TYPE_STRING,
                  xpc_string_get_length(field) == value.utf8.count,
                  let bytes = xpc_string_get_string_ptr(field), value.withCString({ strcmp(bytes, $0) == 0 }) else {
                finish(.rejected); return
            }
        }
        guard DispatchTime.now() < deadline else { finish(.expired); return }
        finish(.matched)
    }

    private func finish(_ outcome: NativeMessageOutcome) {
        guard !finished else { return }
        finished = true
        timer.cancel()
        xpc_connection_cancel(connection)
        completion(outcome)
    }
}
