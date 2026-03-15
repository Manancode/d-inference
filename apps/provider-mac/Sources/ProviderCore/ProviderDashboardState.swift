import Foundation

public enum ProviderNodeState: String, Sendable {
    case ready
    case busy
    case paused
    case offline
}

public struct ProviderSnapshot: Equatable, Sendable {
    public var nodeID: String
    public var selectedModel: String
    public var nodeState: ProviderNodeState
    public var hourlyRateUSDC: Decimal

    public init(
        nodeID: String,
        selectedModel: String,
        nodeState: ProviderNodeState,
        hourlyRateUSDC: Decimal
    ) {
        self.nodeID = nodeID
        self.selectedModel = selectedModel
        self.nodeState = nodeState
        self.hourlyRateUSDC = hourlyRateUSDC
    }
}

public final class ProviderDashboardState: @unchecked Sendable {
    public private(set) var snapshot: ProviderSnapshot

    public init(snapshot: ProviderSnapshot) {
        self.snapshot = snapshot
    }

    public func apply(_ snapshot: ProviderSnapshot) {
        self.snapshot = snapshot
    }

    public var menuBarTitle: String {
        "DGInf"
    }

    public var statusLine: String {
        let rate = NumberFormatter.currencyFormatter.string(from: snapshot.hourlyRateUSDC as NSNumber) ?? "$0.00"
        return "\(snapshot.nodeState.rawValue.capitalized) | \(rate)/hr | \(snapshot.selectedModel)"
    }

    public static var preview: ProviderDashboardState {
        ProviderDashboardState(
            snapshot: ProviderSnapshot(
                nodeID: "dev-node",
                selectedModel: "qwen3.5-35b-a3b",
                nodeState: .ready,
                hourlyRateUSDC: 0.80
            )
        )
    }
}

private extension NumberFormatter {
    static let currencyFormatter: NumberFormatter = {
        let formatter = NumberFormatter()
        formatter.numberStyle = .currency
        formatter.currencyCode = "USD"
        formatter.maximumFractionDigits = 2
        formatter.minimumFractionDigits = 2
        return formatter
    }()
}

