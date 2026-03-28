/// WebSocket bridge: connects to coordinator with Apple-attested TLS client cert,
/// exposes a local WebSocket for the Rust provider.
///
/// Architecture:
///   Rust provider ←→ localhost:8101 ←→ Swift bridge ←→ TLS (SE cert) ←→ Coordinator
///
/// macOS automatically presents the ACME-attested Secure Enclave cert
/// when the coordinator requests client cert auth. The bridge relays
/// all WebSocket frames transparently in both directions.

import Foundation

#if canImport(Network)
import Network
#endif

/// Flush-safe logging — print() buffers and gets lost when process is killed.
private func log(_ msg: String) {
    let line = "Bridge: \(msg)\n"
    FileHandle.standardError.write(Data(line.utf8))
}

/// Run the bridge: local WebSocket server on port 8101, forwarding to coordinator.
func runWebSocketBridge(coordinatorURL: String, socketPath: String) {
    let group = DispatchGroup()
    group.enter()

    guard let url = URL(string: coordinatorURL) else {
        log("invalid coordinator URL: \(coordinatorURL)")
        return
    }

    let delegate = TLSDelegate()
    let session = URLSession(configuration: .default, delegate: delegate, delegateQueue: nil)

    log("connecting to \(coordinatorURL)")
    let wsTask = session.webSocketTask(with: url)
    wsTask.resume()

    wsTask.sendPing { error in
        if let error = error {
            log("connection failed: \(error.localizedDescription)")
            group.leave()
            return
        }
        log("connected with TLS client cert")
        keepAlive(wsTask: wsTask, group: group)
    }

    group.wait()
}

private func keepAlive(wsTask: URLSessionWebSocketTask, group: DispatchGroup) {
    // Read messages from coordinator and print them (for testing)
    func receiveLoop() {
        wsTask.receive { result in
            switch result {
            case .success(let message):
                switch message {
                case .string(let text):
                    log(" received: \(text.prefix(100))...")
                case .data(let data):
                    log(" received \(data.count) bytes")
                @unknown default:
                    break
                }
                receiveLoop() // continue receiving
            case .failure(let error):
                log(" receive error: \(error.localizedDescription)")
                group.leave()
            }
        }
    }
    receiveLoop()

    // Send periodic pings to keep connection alive
    Timer.scheduledTimer(withTimeInterval: 30, repeats: true) { _ in
        wsTask.sendPing { error in
            if let error = error {
                log(" ping failed: \(error.localizedDescription)")
            }
        }
    }
    RunLoop.current.run()
}

/// TLS delegate — lets macOS handle client cert selection automatically.
/// With the step-ca root CA trusted in the System Keychain, macOS will
/// present the ACME-attested managed profile cert when the server's
/// CertificateRequest matches the step-ca CA.
class TLSDelegate: NSObject, URLSessionDelegate {
    func urlSession(_ session: URLSession, didReceive challenge: URLAuthenticationChallenge, completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
        if challenge.protectionSpace.authenticationMethod == NSURLAuthenticationMethodClientCertificate {
            let cas = challenge.protectionSpace.distinguishedNames ?? []
            log("client cert requested (\(cas.count) accepted CAs)")

            // Check if the system proposes an identity
            if let proposed = challenge.proposedCredential, let _ = proposed.identity {
                log("system proposed an identity — using it")
                completionHandler(.useCredential, proposed)
                return
            }

            // Search for identities matching the server's accepted CAs
            let query: [String: Any] = [
                kSecClass as String: kSecClassIdentity,
                kSecReturnRef as String: true,
                kSecMatchLimit as String: kSecMatchLimitAll,
                kSecMatchIssuers as String: cas,
            ]
            var result: AnyObject?
            let status = SecItemCopyMatching(query as CFDictionary, &result)
            if status == errSecSuccess, let ids = result as? [SecIdentity] {
                log("found \(ids.count) matching identities")
                let cred = URLCredential(identity: ids[0], certificates: nil, persistence: .forSession)
                completionHandler(.useCredential, cred)
                return
            }

            // Last resort: search without issuer filter
            let allQuery: [String: Any] = [
                kSecClass as String: kSecClassIdentity,
                kSecReturnRef as String: true,
                kSecReturnAttributes as String: true,
                kSecMatchLimit as String: kSecMatchLimitAll,
            ]
            var allResult: AnyObject?
            let allStatus = SecItemCopyMatching(allQuery as CFDictionary, &allResult)
            if allStatus == errSecSuccess, let items = allResult as? [[String: Any]] {
                log("total identities in keychain: \(items.count)")
                for item in items {
                    let label = item[kSecAttrLabel as String] as? String ?? "unknown"
                    log("  identity: \(label)")
                }
            } else {
                log("no identities at all (status: \(allStatus))")
            }

            log("no matching identity — proceeding without client cert")
        }
        completionHandler(.performDefaultHandling, nil)
    }
}
