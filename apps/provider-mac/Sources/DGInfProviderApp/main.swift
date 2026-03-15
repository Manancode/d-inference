import ProviderCore
import SwiftUI

@main
struct DGInfProviderApp: App {
    @State private var state = ProviderDashboardState.preview

    var body: some Scene {
        MenuBarExtra(state.menuBarTitle, systemImage: "bolt.horizontal.circle") {
            VStack(alignment: .leading, spacing: 12) {
                Text(state.statusLine)
                    .font(.headline)
                Text("Node: \(state.snapshot.nodeID)")
                    .font(.subheadline)
                Text("Model: \(state.snapshot.selectedModel)")
                    .font(.subheadline)
                Divider()
                Text("Coordinator connectivity and provider controls will be wired through providerd.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
            .padding()
            .frame(minWidth: 320)
        }
    }
}

