package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/domain"
	"github.com/dginf/dginf/services/coordinator/internal/store"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
)

type Service struct {
	store        *store.Memory
	now          func() time.Time
	challengeTTL time.Duration
}

func NewService(memory *store.Memory, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{
		store:        memory,
		now:          now,
		challengeTTL: 5 * time.Minute,
	}
}

func (s *Service) IssueChallenge(wallet string, chainID int64) (domain.AuthChallengeResponse, error) {
	wallet = strings.ToLower(wallet)
	nonce, err := randomNonce(16)
	if err != nil {
		return domain.AuthChallengeResponse{}, err
	}
	issuedAt := s.now().UTC()
	message := fmt.Sprintf("dginf wants you to sign in with your Ethereum account:\n%s\n\nURI: https://dginf.local\nVersion: 1\nChain ID: %d\nNonce: %s\nIssued At: %s", wallet, chainID, nonce, issuedAt.Format(time.RFC3339))
	challenge := domain.AuthChallenge{
		Wallet:    wallet,
		Message:   message,
		Nonce:     nonce,
		ChainID:   chainID,
		ExpiresAt: issuedAt.Add(s.challengeTTL),
	}
	s.store.PutChallenge(challenge)
	return domain.AuthChallengeResponse{
		Nonce:     nonce,
		Message:   message,
		ExpiresAt: challenge.ExpiresAt,
	}, nil
}

func (s *Service) VerifyChallenge(wallet, message, signature string) (domain.AuthVerifyResponse, error) {
	nonce := parseField(message, "Nonce")
	if nonce == "" {
		return domain.AuthVerifyResponse{}, domain.ErrChallengeNotFound
	}
	challenge, ok := s.store.GetChallenge(nonce)
	if !ok {
		return domain.AuthVerifyResponse{}, domain.ErrChallengeNotFound
	}
	if s.now().After(challenge.ExpiresAt) {
		return domain.AuthVerifyResponse{}, domain.ErrChallengeExpired
	}
	if challenge.Message != message || challenge.Wallet != strings.ToLower(wallet) {
		return domain.AuthVerifyResponse{}, domain.ErrChallengeMismatch
	}
	if !verifyEthereumSignature(wallet, message, signature) {
		return domain.AuthVerifyResponse{}, domain.ErrInvalidSignature
	}
	s.store.DeleteChallenge(nonce)
	sessionToken := uuid.NewString()
	s.store.PutSession(sessionToken, wallet)
	return domain.AuthVerifyResponse{
		SessionToken: sessionToken,
		Wallet:       strings.ToLower(wallet),
	}, nil
}

func randomNonce(length int) (string, error) {
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func parseField(message, field string) string {
	prefix := field + ": "
	for _, line := range strings.Split(message, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}

func verifyEthereumSignature(wallet, message, signature string) bool {
	sig := strings.TrimPrefix(signature, "0x")
	raw, err := hex.DecodeString(sig)
	if err != nil || len(raw) != 65 {
		return false
	}
	if raw[64] >= 27 {
		raw[64] -= 27
	}
	pub, err := crypto.SigToPub(accounts.TextHash([]byte(message)), raw)
	if err != nil {
		return false
	}
	recovered := crypto.PubkeyToAddress(*pub)
	return strings.EqualFold(common.HexToAddress(wallet).Hex(), recovered.Hex())
}
