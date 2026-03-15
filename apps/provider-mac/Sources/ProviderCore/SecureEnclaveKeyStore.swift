import CryptoKit
import Foundation
import Security

public struct KeyToolResponse: Codable, Equatable, Sendable {
    public var publicKey: String
    public var signature: String?
    public var note: String?

    public init(publicKey: String, signature: String? = nil, note: String? = nil) {
        self.publicKey = publicKey
        self.signature = signature
        self.note = note
    }
}

public enum SecureEnclaveKeyStoreError: Error, Equatable {
    case keyCreationFailed(OSStatus)
    case keyLookupFailed(OSStatus)
    case publicKeyUnavailable
    case exportFailed(OSStatus)
    case signFailed(OSStatus)
    case invalidInput
}

public protocol SecureEnclaveKeyStoreProtocol {
    func ensureSigningKey(tag: String) throws -> String
    func sign(tag: String, payload: Data) throws -> String
}

public final class SecureEnclaveKeyStore: SecureEnclaveKeyStoreProtocol {
    public init() {}

    public func ensureSigningKey(tag: String) throws -> String {
        let privateKey = try loadOrCreatePrivateKey(tag: tag)
        return try exportPublicKey(privateKey)
    }

    public func sign(tag: String, payload: Data) throws -> String {
        let privateKey = try loadOrCreatePrivateKey(tag: tag)
        var error: Unmanaged<CFError>?
        guard let signature = SecKeyCreateSignature(
            privateKey,
            .ecdsaSignatureDigestX962SHA256,
            Data(SHA256.hash(data: payload)) as CFData,
            &error
        ) as Data? else {
            let status = (error?.takeRetainedValue() as Error?) as NSError?
            let code = status.map { OSStatus($0.code) } ?? errSecInternalError
            throw SecureEnclaveKeyStoreError.signFailed(code)
        }
        return signature.rawECDSASignatureBase64
    }

    private func loadOrCreatePrivateKey(tag: String) throws -> SecKey {
        if let existing = try findPrivateKey(tag: tag) {
            return existing
        }
        return try createPrivateKey(tag: tag)
    }

    private func findPrivateKey(tag: String) throws -> SecKey? {
        let query: [CFString: Any] = [
            kSecClass: kSecClassKey,
            kSecAttrApplicationTag: tag.data(using: .utf8)!,
            kSecAttrKeyType: kSecAttrKeyTypeECSECPrimeRandom,
            kSecReturnRef: true,
        ]
        var item: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &item)
        switch status {
        case errSecSuccess:
            return (item as! SecKey)
        case errSecItemNotFound:
            return nil
        default:
            throw SecureEnclaveKeyStoreError.keyLookupFailed(status)
        }
    }

    private func createPrivateKey(tag: String) throws -> SecKey {
        var error: Unmanaged<CFError>?
        guard let access = SecAccessControlCreateWithFlags(
            nil,
            kSecAttrAccessibleWhenUnlockedThisDeviceOnly,
            [.privateKeyUsage],
            nil
        ) else {
            throw SecureEnclaveKeyStoreError.invalidInput
        }
        let attributes: [CFString: Any] = [
            kSecAttrKeyType: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits: 256,
            kSecAttrTokenID: kSecAttrTokenIDSecureEnclave,
            kSecPrivateKeyAttrs: [
                kSecAttrIsPermanent: true,
                kSecAttrApplicationTag: tag.data(using: .utf8)!,
                kSecAttrAccessControl: access,
            ],
        ]
        guard let key = SecKeyCreateRandomKey(attributes as CFDictionary, &error) else {
            let status = (error?.takeRetainedValue() as Error?) as NSError?
            let code = status.map { OSStatus($0.code) } ?? errSecInternalError
            throw SecureEnclaveKeyStoreError.keyCreationFailed(code)
        }
        return key
    }

    private func exportPublicKey(_ privateKey: SecKey) throws -> String {
        guard let publicKey = SecKeyCopyPublicKey(privateKey) else {
            throw SecureEnclaveKeyStoreError.publicKeyUnavailable
        }
        var error: Unmanaged<CFError>?
        guard let external = SecKeyCopyExternalRepresentation(publicKey, &error) as Data? else {
            let status = (error?.takeRetainedValue() as Error?) as NSError?
            let code = status.map { OSStatus($0.code) } ?? errSecInternalError
            throw SecureEnclaveKeyStoreError.exportFailed(code)
        }
        return external.derEncodedP256PublicKeyBase64
    }
}

public extension Data {
    var derEncodedP256PublicKeyBase64: String {
        let prefix: [UInt8] = [
            0x30, 0x59, 0x30, 0x13, 0x06, 0x07, 0x2A, 0x86,
            0x48, 0xCE, 0x3D, 0x02, 0x01, 0x06, 0x08, 0x2A,
            0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07, 0x03,
            0x42, 0x00,
        ]
        let combined = Data(prefix) + self
        return combined.base64EncodedString()
    }

    var rawECDSASignatureBase64: String {
        guard let asn1 = try? P256.Signing.ECDSASignature(derRepresentation: self) else {
            return self.base64EncodedString()
        }
        return asn1.rawRepresentation.base64EncodedString()
    }
}
