import Foundation
import ProviderCore
import Testing

@Test func formatsStatusLine() {
    let state = ProviderDashboardState(
        snapshot: ProviderSnapshot(
            nodeID: "node-1",
            selectedModel: "qwen3.5-35b-a3b",
            nodeState: .busy,
            hourlyRateUSDC: 1.25
        )
    )

    #expect(state.statusLine.contains("Busy"))
    #expect(state.statusLine.contains("qwen3.5-35b-a3b"))
    #expect(state.statusLine.contains("$1.25"))
}

@Test func updatesSnapshot() {
    let state = ProviderDashboardState.preview
    state.apply(
        ProviderSnapshot(
            nodeID: "node-2",
            selectedModel: "qwen3.5-122b-a10b",
            nodeState: .paused,
            hourlyRateUSDC: 2.00
        )
    )

    #expect(state.snapshot.nodeID == "node-2")
    #expect(state.snapshot.nodeState == .paused)
}
