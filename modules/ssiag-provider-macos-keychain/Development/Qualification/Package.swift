// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "symphony-keychain-qualification",
    platforms: [.macOS(.v13)],
    products: [
        .executable(
            name: "symphony-keychain-qualification",
            targets: ["SymphonyKeychainQualification"]
        )
    ],
    targets: [
        .target(name: "DisposableKeychainQualificationSupport"),
        .executableTarget(
            name: "SymphonyKeychainQualification",
            dependencies: ["DisposableKeychainQualificationSupport"]
        ),
        .testTarget(
            name: "DisposableKeychainQualificationSupportTests",
            dependencies: ["DisposableKeychainQualificationSupport"]
        )
    ]
)
