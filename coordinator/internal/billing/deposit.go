package billing

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"math/big"
)

// GetOrCreateDepositAddress returns the consumer's unique Solana deposit address,
// generating a new keypair if one doesn't exist yet.
//
// The private key is stored (encrypted in production) so funds can be swept
// to the hot wallet later.
func (s *Service) GetOrCreateDepositAddress(accountID string) (string, error) {
	// Check if address already exists
	addr, err := s.store.GetDepositAddress(accountID, "solana")
	if err == nil && addr != "" {
		return addr, nil
	}

	// Generate a new ed25519 keypair (Solana uses raw ed25519)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate keypair: %w", err)
	}

	address := base58Encode(pub)
	privKeyEncoded := base58Encode(priv)

	// Store the address and private key
	if err := s.store.SetDepositAddress(accountID, "solana", address, privKeyEncoded); err != nil {
		return "", fmt.Errorf("store deposit address: %w", err)
	}

	s.logger.Info("billing: created deposit address",
		"account", accountID[:min(8, len(accountID))]+"...",
		"address", address,
	)

	return address, nil
}

// VerifyDepositOwnership checks that a deposit's destination address belongs
// to the authenticated consumer. Returns an error if the address doesn't match.
func (s *Service) VerifyDepositOwnership(accountID, depositToAddress string) error {
	owner, err := s.store.GetAccountByDepositAddress(depositToAddress, "solana")
	if err != nil {
		return fmt.Errorf("deposit address %s is not registered in our system", depositToAddress)
	}
	if owner != accountID {
		return fmt.Errorf("deposit address does not belong to your account")
	}
	return nil
}

// IsExternalIDProcessed checks the database (not in-memory) for whether a
// tx signature has already been credited. This survives coordinator restarts.
func (s *Service) IsExternalIDProcessed(externalID string) bool {
	return s.store.IsExternalIDProcessed(externalID)
}

// base58 alphabet used by Bitcoin and Solana
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// base58Encode encodes a byte slice to base58 (Solana address format).
func base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	// Convert to big.Int
	n := new(big.Int).SetBytes(input)
	zero := big.NewInt(0)
	mod := big.NewInt(58)

	var result []byte
	for n.Cmp(zero) > 0 {
		remainder := new(big.Int)
		n.DivMod(n, mod, remainder)
		result = append(result, base58Alphabet[remainder.Int64()])
	}

	// Add leading '1's for each leading zero byte
	for _, b := range input {
		if b != 0 {
			break
		}
		result = append(result, base58Alphabet[0])
	}

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}
