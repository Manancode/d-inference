package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"math/big"
)

type Signer interface {
	PublicKey() (string, error)
	Sign(payload []byte) (string, error)
}

type SoftwareSigner struct {
	key *ecdsa.PrivateKey
}

func NewSoftwareSigner() (*SoftwareSigner, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &SoftwareSigner{key: key}, nil
}

func (s *SoftwareSigner) PublicKey() (string, error) {
	encoded, err := x509.MarshalPKIXPublicKey(&s.key.PublicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encoded), nil
}

func (s *SoftwareSigner) Sign(payload []byte) (string, error) {
	hash := sha256.Sum256(payload)
	r, ss, err := ecdsa.Sign(rand.Reader, s.key, hash[:])
	if err != nil {
		return "", err
	}
	signature := append(pad32(r.Bytes()), pad32(ss.Bytes())...)
	return base64.StdEncoding.EncodeToString(signature), nil
}

func Verify(publicKey string, payload []byte, signature string) (bool, error) {
	rawKey, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return false, err
	}
	decoded, err := x509.ParsePKIXPublicKey(rawKey)
	if err != nil {
		return false, err
	}
	pubKey, ok := decoded.(*ecdsa.PublicKey)
	if !ok {
		return false, err
	}
	rawSig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}
	if len(rawSig) != 64 {
		return false, nil
	}
	hash := sha256.Sum256(payload)
	var r, s big.Int
	r.SetBytes(rawSig[:32])
	s.SetBytes(rawSig[32:])
	return ecdsa.Verify(pubKey, hash[:], &r, &s), nil
}

func pad32(input []byte) []byte {
	if len(input) >= 32 {
		return input
	}
	result := make([]byte, 32)
	copy(result[32-len(input):], input)
	return result
}
