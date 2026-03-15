import Foundation
import ProviderCore

@main
struct DGInfProviderKeyTool {
    static func main() throws {
        let arguments = Array(CommandLine.arguments.dropFirst())
        guard let command = arguments.first else {
            fputs("usage: DGInfProviderKeyTool <ensure-signing-key|sign> [options]\n", stderr)
            Foundation.exit(2)
        }

        let keyStore = SecureEnclaveKeyStore()
        switch command {
        case "ensure-signing-key":
            let tag = try requiredValue("--tag", in: arguments)
            let publicKey = try keyStore.ensureSigningKey(tag: tag)
            try emit(KeyToolResponse(publicKey: publicKey, note: "secure-enclave"))
        case "sign":
            let tag = try requiredValue("--tag", in: arguments)
            let payloadBase64 = try requiredValue("--payload-base64", in: arguments)
            guard let payload = Data(base64Encoded: payloadBase64) else {
                throw SecureEnclaveKeyStoreError.invalidInput
            }
            let publicKey = try keyStore.ensureSigningKey(tag: tag)
            let signature = try keyStore.sign(tag: tag, payload: payload)
            try emit(KeyToolResponse(publicKey: publicKey, signature: signature, note: "secure-enclave"))
        default:
            fputs("unknown command: \(command)\n", stderr)
            Foundation.exit(2)
        }
    }

    private static func requiredValue(_ flag: String, in args: [String]) throws -> String {
        guard let index = args.firstIndex(of: flag), args.indices.contains(index + 1) else {
            throw SecureEnclaveKeyStoreError.invalidInput
        }
        return args[index + 1]
    }

    private static func emit(_ response: KeyToolResponse) throws {
        let data = try JSONEncoder().encode(response)
        FileHandle.standardOutput.write(data)
        FileHandle.standardOutput.write(Data("\n".utf8))
    }
}
