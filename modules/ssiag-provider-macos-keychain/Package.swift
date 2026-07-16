// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "ssiag-provider-macos-keychain",
    platforms: [.macOS(.v13)],
    products: [
        .executable(
            name: "symphony-ssiag-provider-macos-keychain",
            targets: ["SymphonySSIAGMacOSKeychain"]
        )
    ],
    targets: [
        .target(name: "SSIAGMacOSKeychainSupport"),
        .executableTarget(
            name: "SymphonySSIAGMacOSKeychain",
            dependencies: ["SSIAGMacOSKeychainSupport"]
        ),
        .testTarget(
            name: "SSIAGMacOSKeychainSupportTests",
            dependencies: ["SSIAGMacOSKeychainSupport"]
        )
    ]
)
