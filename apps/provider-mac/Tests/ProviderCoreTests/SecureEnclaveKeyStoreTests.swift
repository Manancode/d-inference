import Foundation
import ProviderCore
import Testing

@Test func derEncodingAddsSubjectPublicKeyInfoPrefix() {
    let raw = Data(repeating: 0x01, count: 65)
    let base64 = raw.derEncodedP256PublicKeyBase64
    let decoded = Data(base64Encoded: base64)!
    #expect(decoded.count == raw.count + 26)
}
