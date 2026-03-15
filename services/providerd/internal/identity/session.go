package identity

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var ErrInvalidEnvelope = errors.New("invalid encrypted envelope")

type SessionKeyPair struct {
	privateKey *ecdh.PrivateKey
}

type Envelope struct {
	Version         int    `json:"version"`
	EphemeralPubkey string `json:"ephemeral_pubkey"`
	Nonce           string `json:"nonce"`
	Ciphertext      string `json:"ciphertext"`
}

type DecryptedEnvelope struct {
	Prompt          string `json:"prompt"`
	MaxOutputTokens int    `json:"max_output_tokens"`
}

func NewSessionKeyPair() (*SessionKeyPair, error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &SessionKeyPair{privateKey: key}, nil
}

func (s *SessionKeyPair) PublicKey() string {
	return base64.StdEncoding.EncodeToString(s.privateKey.PublicKey().Bytes())
}

func (s *SessionKeyPair) DecryptEnvelope(raw string) (DecryptedEnvelope, error) {
	var envelope Envelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return DecryptedEnvelope{}, ErrInvalidEnvelope
	}
	ephemeralRaw, err := base64.StdEncoding.DecodeString(envelope.EphemeralPubkey)
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	ephemeralKey, err := ecdh.X25519().NewPublicKey(ephemeralRaw)
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	shared, err := s.privateKey.ECDH(ephemeralKey)
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	key := sha256.Sum256(append(shared, []byte("dginf-envelope-v1")...))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return DecryptedEnvelope{}, err
	}
	var decrypted DecryptedEnvelope
	if err := json.Unmarshal(plaintext, &decrypted); err != nil {
		return DecryptedEnvelope{}, err
	}
	return decrypted, nil
}

func LooksEncryptedEnvelope(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, "\"ciphertext\"")
}
