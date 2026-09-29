import Darwin
import Foundation
import Testing
@testable import SSIAGMacOSKeychainSupport

@Suite(.serialized)
struct NativePeerTrustTests {
    @Test func acceptsKernelBoundSignedFixtureAndOwnsDescriptor() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        let session = try NativePeerSession.observe(descriptor: fixture.peer, policy: fixture.policy())
        #expect(session.identity.pid == fixture.process.processIdentifier)
        #expect(session.identity.uid == geteuid())
        #expect(session.identity.gid == getegid())
        fixture.closeAccepted()
        // Reuse the caller's descriptor number with an unrelated file.
        let unrelated = open("/dev/null", O_RDONLY)
        defer { if unrelated >= 0 { Darwin.close(unrelated) } }
        #expect(session.revalidate() == nil)
        session.close()
        session.close()
        #expect(session.revalidate() == .closed)
    }

    @Test func observationDoesNotConsumeQueuedData() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        let session = try NativePeerSession.observe(descriptor: fixture.peer, policy: fixture.policy())
        defer { session.close() }
        try fixture.send("P")
        var pending = pollfd(fd: fixture.peer, events: Int16(POLLIN), revents: 0)
        #expect(poll(&pending, 1, 2000) > 0)
        #expect(session.revalidate() == nil)
        var marker: UInt8 = 0
        #expect(read(fixture.peer, &marker, 1) == 1)
        #expect(marker == 77)
    }

    @Test func successfulAndFailedObservationsReleaseTheirDescriptors() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        let policy = try fixture.policy()
        let wrong = try NativePeerPolicy(uid: geteuid(), gid: getegid(), requirementText: "false")
        // Warm Security's lazy initialization before checking descriptor lifetime.
        let warm = try NativePeerSession.observe(descriptor: fixture.peer, policy: policy)
        warm.close()
        func firstFree() throws -> Int32 {
            let fd = fcntl(fixture.peer, F_DUPFD_CLOEXEC, 3)
            guard fd >= 0 else { throw NativePeerTestError.failed }
            Darwin.close(fd)
            return fd
        }
        let before = try firstFree()
        for _ in 0..<16 {
            let session = try NativePeerSession.observe(descriptor: fixture.peer, policy: policy)
            session.close()
            #expect(throws: NativePeerFailure.self) { try NativePeerSession.observe(descriptor: fixture.peer, policy: wrong) }
        }
        #expect(try firstFree() == before)
    }

    @Test func refusesWrongIdentityAndCodeRequirement() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        for policy in [
            try NativePeerPolicy(uid: geteuid() &+ 1, gid: getegid(), requirementText: NativePeerFixture.requirement),
            try NativePeerPolicy(uid: geteuid(), gid: getegid() &+ 1, requirementText: NativePeerFixture.requirement),
            try NativePeerPolicy(uid: geteuid(), gid: getegid(), requirementText: #"identifier "symphony.fixture.other""#),
            try NativePeerPolicy(uid: geteuid(), gid: getegid(), requirementText: "anchor apple"),
        ] {
            do { let s = try NativePeerSession.observe(descriptor: fixture.peer, policy: policy); s.close(); Issue.record("mismatched peer admitted") }
            catch { #expect(error is NativePeerFailure) }
        }
        // A failed observation never takes ownership of the supplied socket.
        let valid = try NativePeerSession.observe(descriptor: fixture.peer, policy: fixture.policy())
        #expect(valid.revalidate() == nil)
        valid.close()
    }

    @Test func refusesInvalidPolicies() {
        for text in ["", "identifier (", "true\0false", String(repeating: "a", count: 4097)] {
            #expect(throws: NativePeerFailure.self) { try NativePeerPolicy(uid: geteuid(), gid: getegid(), requirementText: text) }
        }
    }

    @Test func refusesNonConnectedNonStreamAndSelfSockets() throws {
        let policy = try NativePeerPolicy(uid: geteuid(), gid: getegid(), requirementText: "true")
        let file = open("/dev/null", O_RDONLY)
        let socketFD = socket(AF_UNIX, SOCK_STREAM, 0)
        let internet = socket(AF_INET, SOCK_STREAM, 0)
        var stream = [Int32](repeating: -1, count: 2)
        var datagram = [Int32](repeating: -1, count: 2)
        #expect(socketpair(AF_UNIX, SOCK_STREAM, 0, &stream) == 0)
        #expect(socketpair(AF_UNIX, SOCK_DGRAM, 0, &datagram) == 0)
        let descriptors = [-1, file, socketFD, internet] + stream + datagram
        defer { for fd in descriptors where fd >= 0 { Darwin.close(fd) } }
        for fd in descriptors {
            #expect(throws: NativePeerFailure.self) { try NativePeerSession.observe(descriptor: fd, policy: policy) }
        }
    }

    @Test func rejectsOriginalAuditTokenAfterSamePIDExec() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        let pid = fixture.process.processIdentifier
        let session = try NativePeerSession.observe(descriptor: fixture.peer, policy: fixture.policy())
        try fixture.send("E")
        var result: NativePeerFailure?
        for _ in 0..<200 {
            result = session.revalidate()
            if result != nil { break }
            usleep(10_000)
        }
        #expect(result != nil)
        #expect(fixture.process.isRunning)
        #expect(fixture.process.processIdentifier == pid)
        #expect(session.revalidate() == .closed)
    }

    @Test func rejectsClosedConnectionWhileProcessRemainsAlive() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        let session = try NativePeerSession.observe(descriptor: fixture.peer, policy: fixture.policy())
        try fixture.send("C")
        var result: NativePeerFailure?
        for _ in 0..<200 {
            result = session.revalidate()
            if result != nil { break }
            usleep(10_000)
        }
        #expect(result != nil)
        #expect(fixture.process.isRunning)
        #expect(session.revalidate() == .closed)
    }

    @Test func rejectsProcessExitAndRepeatedClose() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        let session = try NativePeerSession.observe(descriptor: fixture.peer, policy: fixture.policy())
        fixture.process.terminate()
        fixture.process.waitUntilExit()
        #expect(session.revalidate() != nil)
        session.close()
        #expect(session.revalidate() == .closed)
    }

    @Test func serializesConcurrentObservationAndClose() throws {
        let fixture = try NativePeerFixture()
        defer { fixture.cleanup() }
        let session = try NativePeerSession.observe(descriptor: fixture.peer, policy: fixture.policy())
        DispatchQueue.concurrentPerform(iterations: 64) { i in
            if i.isMultiple(of: 3) { session.close() }
            else { let result = session.revalidate(); #expect(result == nil || result == .closed) }
        }
        #expect(session.revalidate() == .closed)
    }
}

private enum NativePeerTestError: Error { case failed }
private final class NativePeerFixture {
    static let requirement = #"identifier "symphony.fixture.native-peer""#
    private static let helper: Result<URL, Error> = Result {
        let root = URL(fileURLWithPath: "/private/tmp/sqv18-helper-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: root, withIntermediateDirectories: false, attributes: [.posixPermissions: 0o700])
        let source = URL(fileURLWithPath: #filePath).deletingLastPathComponent().deletingLastPathComponent().appending(path: "Fixtures/NativePeerClient.cpp")
        let target = root.appending(path: "peer")
        try run("/usr/bin/xcrun", ["clang++", "-std=c++2c", "-Wall", "-Wextra", "-Werror", source.path, "-o", target.path])
        try run("/usr/bin/codesign", ["--force", "--sign", "-", "--identifier", "symphony.fixture.native-peer", target.path])
        return target
    }
    let root: URL
    let process = Process()
    var peer: Int32 = -1
    private var listener: Int32 = -1

    init() throws {
        let helper = try Self.helper.get()
        root = URL(fileURLWithPath: "/private/tmp/sqv18-peer-\(UUID().uuidString)")
        do {
            try FileManager.default.createDirectory(at: root, withIntermediateDirectories: false, attributes: [.posixPermissions: 0o700])
            listener = socket(AF_UNIX, SOCK_STREAM, 0)
            guard listener >= 0 else { throw NativePeerTestError.failed }
            let path = root.appending(path: "s").path
            var address = sockaddr_un()
            address.sun_family = sa_family_t(AF_UNIX)
            address.sun_len = UInt8(MemoryLayout<sockaddr_un>.size)
            let bytes = Array(path.utf8) + [0]
            guard bytes.count <= MemoryLayout.size(ofValue: address.sun_path) else { throw NativePeerTestError.failed }
            withUnsafeMutableBytes(of: &address.sun_path) { $0.copyBytes(from: bytes) }
            let bound = withUnsafePointer(to: &address) { $0.withMemoryRebound(to: sockaddr.self, capacity: 1) { bind(listener, $0, socklen_t(MemoryLayout<sockaddr_un>.size)) } }
            guard bound == 0, listen(listener, 1) == 0 else { throw NativePeerTestError.failed }
            process.executableURL = helper
            process.arguments = [path]
            process.environment = ["PATH": "/usr/bin:/bin"]
            process.standardOutput = FileHandle.nullDevice
            process.standardError = FileHandle.nullDevice
            try process.run()
            var ready = pollfd(fd: listener, events: Int16(POLLIN), revents: 0)
            guard poll(&ready, 1, 5000) > 0 else { throw NativePeerTestError.failed }
            peer = accept(listener, nil, nil)
            guard peer >= 0 else { throw NativePeerTestError.failed }
            var timeout = timeval(tv_sec: 2, tv_usec: 0)
            guard setsockopt(peer, SOL_SOCKET, SO_RCVTIMEO, &timeout, socklen_t(MemoryLayout<timeval>.size)) == 0 else { throw NativePeerTestError.failed }
            var byte: UInt8 = 0
            guard read(peer, &byte, 1) == 1, byte == 82 else { throw NativePeerTestError.failed }
        } catch { cleanup(); throw error }
    }

    func policy() throws -> NativePeerPolicy {
        try NativePeerPolicy(uid: geteuid(), gid: getegid(), requirementText: Self.requirement)
    }
    func send(_ command: UInt8) throws { var value = command; guard write(peer, &value, 1) == 1 else { throw NativePeerTestError.failed } }
    func send(_ command: String) throws { try send(try #require(command.utf8.first)) }
    func closeAccepted() { if peer >= 0 { Darwin.close(peer); peer = -1 } }
    func cleanup() {
        closeAccepted()
        if listener >= 0 { Darwin.close(listener); listener = -1 }
        if process.isRunning { process.terminate(); process.waitUntilExit() }
        try? FileManager.default.removeItem(at: root)
    }
    private static func run(_ executable: String, _ arguments: [String]) throws {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = arguments
        process.standardOutput = FileHandle.nullDevice
        process.standardError = FileHandle.nullDevice
        try process.run()
        process.waitUntilExit()
        guard process.terminationStatus == 0 else { throw NativePeerTestError.failed }
    }
}
