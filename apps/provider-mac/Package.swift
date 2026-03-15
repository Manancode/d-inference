// swift-tools-version: 6.2
import PackageDescription

let package = Package(
    name: "provider-mac",
    platforms: [
        .macOS(.v14),
    ],
    products: [
        .library(name: "ProviderCore", targets: ["ProviderCore"]),
        .executable(name: "DGInfProviderApp", targets: ["DGInfProviderApp"]),
        .executable(name: "DGInfProviderKeyTool", targets: ["DGInfProviderKeyTool"]),
    ],
    targets: [
        .target(
            name: "ProviderCore"
        ),
        .executableTarget(
            name: "DGInfProviderApp",
            dependencies: ["ProviderCore"]
        ),
        .executableTarget(
            name: "DGInfProviderKeyTool",
            dependencies: ["ProviderCore"]
        ),
        .testTarget(
            name: "ProviderCoreTests",
            dependencies: ["ProviderCore"]
        ),
    ]
)
