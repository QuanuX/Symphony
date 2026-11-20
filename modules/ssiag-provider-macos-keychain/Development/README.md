# Test the macOS Keychain

The qualification tool has two native exercises. The current-user exercise creates a temporary file-based keychain. The data-protection exercise checks a signed app's private access group. Both use generated disposable items. Apple describes the storage and access-control differences in [TN3137](https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains).

## Run as the logged-in Mac user

From this directory, run the following command in the intended user's local session.

```sh
swift run --package-path Qualification symphony-keychain-qualification user-keychain
```

This exercise uses a temporary keychain dedicated to the test. It requires no Apple Developer account, team selection or distribution certificate. A pass qualifies the file-based Keychain path for the current user.

## Test the implementation locally

From this directory, run `swift test --package-path Qualification`. These tests use an injected Keychain implementation to verify query scope, refusal handling and cleanup without touching any user's Keychain. The Xcode project compiles the same Swift files.

## Run against a signed app's data-protection Keychain

Use the local graphical login of the intended user. Choose the Personal Team or development team that owns this test. Xcode manages the development certificate and profile. A unique test bundle ID keeps the item in a dedicated private access group. Apple documents the [free Personal Team](https://developer.apple.com/help/account/basics/about-your-developer-account/), [macOS capabilities](https://developer.apple.com/help/account/reference/supported-capabilities-macos/) and [data-protection Keychain](https://developer.apple.com/documentation/technotes/tn3137-on-mac-keychains).

1. Open `KeychainQualification.xcodeproj` in Xcode. Under **Signing & Capabilities**, select the intended team and set a bundle ID you control ending in `.symphony-keychain-check`. Use automatic signing. The project requests only the app's own Keychain group.
2. Build the Debug app for **My Mac**. Find the built `KeychainQualification.app` in Products. In a Terminal window in the same login, set the following values to the built app path, the selected team ID and the full bundle ID you configured.

   ```sh
   app="/absolute/path/to/KeychainQualification.app"
   team="YOUR_TEAM_ID"
   bundle="your.unique.symphony-keychain-check"
   "$app/Contents/MacOS/KeychainQualification" preflight "$team" "$bundle"
   ```

3. After preflight passes, run the test.

   ```sh
   "$app/Contents/MacOS/KeychainQualification" exercise "$team" "$bundle"
   ```

The signed exercise generates two random values and a fresh item identifier, creates the item, verifies the first value, replaces it, verifies the second value, deletes the item and checks that it is absent. The item uses device-only accessibility while unlocked and disables synchronization. The executable reports only status. A failed create never triggers deletion. If the process is killed after creation, a synthetic item may remain in this test app's private group.

Retain the commit, macOS version, architecture, Xcode version and exit status as development evidence. Production uses the [separately signed and notarized provider bundle](../SPEC.md#phase-10b-production-readiness-profile). Operational credential use also requires the [SSIAG dispatch and lease integration](../../secure-identity-access-governance/CREDENTIAL-QUALIFICATION.md).
