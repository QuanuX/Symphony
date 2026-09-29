// Built only by NativeMessageGateTests as a disposable app-bundled XPC service.
import Darwin
import Foundation
import XPC

private final class FixtureState: @unchecked Sendable {
    let lock = NSLock()
    var gate: NativeMessageGate?
    var connected = false
    var outcomes: [NativeMessageOutcome] = []
}
private let fixture = FixtureState()
private let requestID = "684921d8-a8b5-49da-872b-568eb6a6dc03"
private let topsID = "018f0c3a-7b2d-7e11-8c12-0242ac120002"
private let challenge = "0190c7df-6df2-7f2b-9f4f-8e0c33f5f287"
private let digest = "sha256:" + String(repeating: "a", count: 64)

func acceptFixture(_ connection: xpc_connection_t) {
    fixture.lock.lock()
    defer { fixture.lock.unlock() }
    guard !fixture.connected else {
        xpc_connection_set_event_handler(connection) { _ in }
        xpc_connection_activate(connection); xpc_connection_cancel(connection); return
    }
    fixture.connected = true
    let info = Bundle.main.infoDictionary ?? [:]
    guard let output = info["FixtureOutput"] as? String, let scenario = info["FixtureScenario"] as? String else { exit(2) }
    do {
        let requirement = scenario == "wrong-code" ? #"identifier "symphony.fixture.other""# : #"identifier "symphony.fixture.message-host""#
        let uid = scenario == "wrong-uid" ? geteuid() &+ 1 : geteuid()
        let policy = try NativeMessagePolicy(uid: uid, gid: getegid(), requirementText: requirement)
        let binding = try NativeMessageBinding(topsID: topsID, requestID: requestID, bindingDigest: digest, challenge: challenge)
        fixture.gate = try NativeMessageGate(connection: connection, policy: policy, binding: binding, timeout: scenario == "expired" ? 0.000000001 : 0.8) { result in
            fixture.lock.lock()
            fixture.outcomes.append(result)
            let first = fixture.outcomes.count == 1
            fixture.lock.unlock()
            guard first else { return }
            // Keep the gate alive past its original deadline so duplicate
            // callbacks or a resurrected timer become visible in the count.
            DispatchQueue.global().asyncAfter(deadline: .now() + 1.0) {
                fixture.lock.lock()
                let value = fixture.outcomes.count == 1 ? fixture.outcomes[0].rawValue : "multiple_completions"
                fixture.lock.unlock()
                try? value.write(toFile: output, atomically: true, encoding: .utf8)
                exit(0)
            }
        }
    } catch {
        try? "setup_failed".write(toFile: output, atomically: true, encoding: .utf8)
        exit(3)
    }
}
#if SQV19_TSAN
// Public sanitizer runtime API: capture reports from the actual service, whose
// launchd-managed stderr is not assumed to reach the test runner.
guard let reportPath = Bundle.main.infoDictionary?["FixtureSanitizerReport"] as? String,
      let symbol = dlsym(UnsafeMutableRawPointer(bitPattern: -2), "__sanitizer_set_report_path") else { exit(4) }
let reportSetter = unsafeBitCast(symbol, to: (@convention(c) (UnsafePointer<CChar>) -> Void).self)
reportPath.withCString { reportSetter($0) }
#endif
xpc_main(acceptFixture)
