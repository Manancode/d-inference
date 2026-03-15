package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var ErrWalletRequired = errors.New("wallet is required")

type Challenge struct {
	Challenge string    `json:"challenge"`
	Nonce     string    `json:"nonce"`
	IssuedAt  time.Time `json:"issued_at"`
}

type ChallengeService struct {
	now func() time.Time
}

func NewChallengeService(now func() time.Time) *ChallengeService {
	if now == nil {
		now = time.Now
	}

	return &ChallengeService{now: now}
}

func (s *ChallengeService) New(wallet string) (Challenge, error) {
	if wallet == "" {
		return Challenge{}, ErrWalletRequired
	}

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return Challenge{}, fmt.Errorf("generate nonce: %w", err)
	}

	nonce := hex.EncodeToString(nonceBytes)
	issuedAt := s.now().UTC()

	return Challenge{
		Challenge: fmt.Sprintf("DGInf sign-in request for %s\nNonce: %s", wallet, nonce),
		Nonce:     nonce,
		IssuedAt:  issuedAt,
	}, nil
}
