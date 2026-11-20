import Darwin
import Foundation
#if SWIFT_PACKAGE
import DisposableKeychainQualificationSupport
#endif

let args = Array(CommandLine.arguments.dropFirst())
if args == ["user-keychain"] {
    do {
        try CurrentUserKeychain.exercise()
        print("PASS disposable current-user file Keychain create, duplicate refusal, read, replace, delete, absence, and cleanup")
        exit(0)
    } catch let failure as CurrentUserKeychainFailure {
        print("FAILED file Keychain \(failure.rawValue)")
        exit(1)
    } catch {
        print("FAILED file Keychain unexpected")
        exit(1)
    }
}
guard args.count == 3, ["preflight", "exercise"].contains(args[0]) else {
    print("usage: symphony-keychain-qualification user-keychain | preflight|exercise EXPECTED_TEAM_ID DEDICATED_BUNDLE_ID")
    exit(64)
}

do {
    let check = try DevelopmentQualification(teamID: args[1], bundleID: args[2])
    if args[0] == "preflight" {
        _ = try check.preflight()
        print("PASS development signing identity, private group, and local login")
    } else {
        try check.exercise()
        print("PASS disposable Keychain create, compare, replace, delete, and absence")
    }
} catch let failure as QualificationFailure {
    print("REFUSED \(failure.rawValue)")
    exit(1)
} catch let failure as DisposableKeychainError {
    print("FAILED \(failure.rawValue)")
    exit(1)
} catch {
    print("FAILED unexpected")
    exit(1)
}
