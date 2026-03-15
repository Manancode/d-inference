package auth

import (
	"testing"
	"time"

	"github.com/dginf/dginf/services/coordinator/internal/domain"
	"github.com/dginf/dginf/services/coordinator/internal/store"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestIssueAndVerifyChallenge(t *testing.T) {
	memory := store.NewMemory()
	svc := NewService(memory, func() time.Time { return time.Unix(1_700_000_000, 0) })
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	wallet := crypto.PubkeyToAddress(key.PublicKey).Hex()

	challenge, err := svc.IssueChallenge(wallet, 8453)
	if err != nil {
		t.Fatalf("issue challenge: %v", err)
	}
	sig, err := crypto.Sign(accounts.TextHash([]byte(challenge.Message)), key)
	if err != nil {
		t.Fatalf("sign challenge: %v", err)
	}
	response, err := svc.VerifyChallenge(wallet, challenge.Message, "0x"+commonToHex(sig))
	if err != nil {
		t.Fatalf("verify challenge: %v", err)
	}
	if response.Wallet == "" || response.SessionToken == "" {
		t.Fatalf("expected wallet and session token, got %#v", response)
	}
}

func TestRejectsReplayedChallenge(t *testing.T) {
	memory := store.NewMemory()
	svc := NewService(memory, time.Now)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	wallet := crypto.PubkeyToAddress(key.PublicKey).Hex()
	challenge, _ := svc.IssueChallenge(wallet, 8453)
	sig, _ := crypto.Sign(accounts.TextHash([]byte(challenge.Message)), key)

	if _, err := svc.VerifyChallenge(wallet, challenge.Message, "0x"+commonToHex(sig)); err != nil {
		t.Fatalf("first verify failed: %v", err)
	}
	if _, err := svc.VerifyChallenge(wallet, challenge.Message, "0x"+commonToHex(sig)); err != domain.ErrChallengeNotFound {
		t.Fatalf("expected challenge not found, got %v", err)
	}
}

func commonToHex(input []byte) string {
	const hextable = "0123456789abcdef"
	result := make([]byte, len(input)*2)
	for i, b := range input {
		result[i*2] = hextable[b>>4]
		result[i*2+1] = hextable[b&0x0f]
	}
	return string(result)
}
