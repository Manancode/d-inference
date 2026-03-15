package identity

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"

	"crypto/aes"
	"crypto/cipher"
)

func TestSessionKeyPairDecryptEnvelope(t *testing.T) {
	keys, err := NewSessionKeyPair()
	if err != nil {
		t.Fatalf("new session key pair: %v", err)
	}
	recipientPubRaw, err := base64.StdEncoding.DecodeString(keys.PublicKey())
	if err != nil {
		t.Fatalf("decode public key: %v", err)
	}
	recipientPub, err := ecdh.X25519().NewPublicKey(recipientPubRaw)
	if err != nil {
		t.Fatalf("new recipient public key: %v", err)
	}
	ephemeral, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ephemeral: %v", err)
	}
	shared, err := ephemeral.ECDH(recipientPub)
	if err != nil {
		t.Fatalf("derive shared secret: %v", err)
	}
	key := sha256.Sum256(append(shared, []byte("dginf-envelope-v1")...))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("new gcm: %v", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("rand nonce: %v", err)
	}
	plaintext, err := json.Marshal(DecryptedEnvelope{
		Prompt:          "hello world",
		MaxOutputTokens: 12,
	})
	if err != nil {
		t.Fatalf("marshal plaintext: %v", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	envelopeBytes, err := json.Marshal(Envelope{
		Version:         1,
		EphemeralPubkey: base64.StdEncoding.EncodeToString(ephemeral.PublicKey().Bytes()),
		Nonce:           base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:      base64.StdEncoding.EncodeToString(ciphertext),
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	result, err := keys.DecryptEnvelope(string(envelopeBytes))
	if err != nil {
		t.Fatalf("decrypt envelope: %v", err)
	}
	if result.Prompt != "hello world" || result.MaxOutputTokens != 12 {
		t.Fatalf("unexpected decrypted result: %#v", result)
	}
}
