package store

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// testPostgresStore returns a PostgresStore connected to the test database.
// It skips the test if DATABASE_URL is not set.
// Each test gets a clean slate by truncating all tables.
func testPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := NewPostgres(ctx, dbURL)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}

	// Clean tables for test isolation.
	for _, table := range []string{
		"usage",
		"payments",
		"api_keys",
		"balances",
		"ledger_entries",
		"billing_sessions",
		"users",
		"device_codes",
		"provider_tokens",
		"invite_redemptions",
		"invite_codes",
		"referrals",
		"referrers",
		"provider_earnings",
		"provider_payouts",
		"providers",
	} {
		if _, err := s.pool.Exec(ctx, "TRUNCATE "+table+" CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}

	t.Cleanup(func() { s.Close() })
	return s
}

func TestPostgresCreateKey(t *testing.T) {
	s := testPostgresStore(t)

	key, err := s.CreateKey()
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	if !strings.HasPrefix(key, "eigeninference-") {
		t.Errorf("key %q does not have eigeninference- prefix", key)
	}

	if !s.ValidateKey(key) {
		t.Error("created key should be valid")
	}

	if s.KeyCount() != 1 {
		t.Errorf("key count = %d, want 1", s.KeyCount())
	}
}

func TestPostgresCreateMultipleKeys(t *testing.T) {
	s := testPostgresStore(t)

	key1, _ := s.CreateKey()
	key2, _ := s.CreateKey()

	if key1 == key2 {
		t.Error("keys should be unique")
	}

	if s.KeyCount() != 2 {
		t.Errorf("key count = %d, want 2", s.KeyCount())
	}
}

func TestPostgresValidateKeyInvalid(t *testing.T) {
	s := testPostgresStore(t)

	if s.ValidateKey("wrong-key") {
		t.Error("wrong key should not be valid")
	}
	if s.ValidateKey("") {
		t.Error("empty key should not be valid")
	}
}

func TestPostgresRevokeKey(t *testing.T) {
	s := testPostgresStore(t)

	key, _ := s.CreateKey()
	if !s.ValidateKey(key) {
		t.Fatal("key should be valid before revoke")
	}

	if !s.RevokeKey(key) {
		t.Error("RevokeKey should return true for existing key")
	}
	if s.ValidateKey(key) {
		t.Error("key should be invalid after revoke")
	}
	if s.KeyCount() != 0 {
		t.Errorf("key count = %d, want 0 after revoke", s.KeyCount())
	}
}

func TestPostgresRevokeKeyNonexistent(t *testing.T) {
	s := testPostgresStore(t)

	if s.RevokeKey("nonexistent") {
		t.Error("RevokeKey should return false for nonexistent key")
	}
}

func TestPostgresSeedKey(t *testing.T) {
	s := testPostgresStore(t)

	err := s.SeedKey("my-admin-key")
	if err != nil {
		t.Fatalf("SeedKey: %v", err)
	}

	if !s.ValidateKey("my-admin-key") {
		t.Error("seeded key should be valid")
	}

	// Seeding the same key again should be a no-op.
	err = s.SeedKey("my-admin-key")
	if err != nil {
		t.Fatalf("SeedKey (duplicate): %v", err)
	}

	if s.KeyCount() != 1 {
		t.Errorf("key count = %d, want 1", s.KeyCount())
	}
}

func TestPostgresRecordUsage(t *testing.T) {
	s := testPostgresStore(t)

	s.RecordUsage("provider-1", "consumer-key", "qwen3.5-9b", 50, 100)
	s.RecordUsage("provider-2", "consumer-key", "llama-3", 30, 200)

	records := s.UsageRecords()
	if len(records) != 2 {
		t.Fatalf("usage records = %d, want 2", len(records))
	}

	r := records[0]
	if r.ProviderID != "provider-1" {
		t.Errorf("provider_id = %q", r.ProviderID)
	}
	if r.Model != "qwen3.5-9b" {
		t.Errorf("model = %q", r.Model)
	}
	if r.PromptTokens != 50 {
		t.Errorf("prompt_tokens = %d", r.PromptTokens)
	}
	if r.CompletionTokens != 100 {
		t.Errorf("completion_tokens = %d", r.CompletionTokens)
	}
	if r.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestPostgresUsageRecordsEmpty(t *testing.T) {
	s := testPostgresStore(t)

	records := s.UsageRecords()
	if len(records) != 0 {
		t.Errorf("usage records = %d, want 0", len(records))
	}
}

func TestPostgresRecordPayment(t *testing.T) {
	s := testPostgresStore(t)

	err := s.RecordPayment("0xabc123", "0xconsumer", "0xprovider", "0.05", "qwen3.5-9b", 50, 100, "test payment")
	if err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}
}

func TestPostgresRecordPaymentDuplicateTxHash(t *testing.T) {
	s := testPostgresStore(t)

	err := s.RecordPayment("0xabc123", "0xconsumer", "0xprovider", "0.05", "qwen3.5-9b", 50, 100, "")
	if err != nil {
		t.Fatalf("first RecordPayment: %v", err)
	}

	err = s.RecordPayment("0xabc123", "0xconsumer", "0xprovider", "0.05", "qwen3.5-9b", 50, 100, "")
	if err == nil {
		t.Error("expected error for duplicate tx_hash")
	}
}

func TestPostgresProviderPayoutsPersist(t *testing.T) {
	s := testPostgresStore(t)

	payout := &ProviderPayout{
		ProviderAddress: "0xprovider-wallet",
		AmountMicroUSD:  900_000,
		Model:           "qwen3.5-9b",
		JobID:           "job-123",
	}
	if err := s.RecordProviderPayout(payout); err != nil {
		t.Fatalf("RecordProviderPayout: %v", err)
	}

	payouts, err := s.ListProviderPayouts()
	if err != nil {
		t.Fatalf("ListProviderPayouts: %v", err)
	}
	if len(payouts) != 1 {
		t.Fatalf("provider payouts = %d, want 1", len(payouts))
	}
	if payouts[0].ProviderAddress != payout.ProviderAddress {
		t.Errorf("provider address = %q, want %q", payouts[0].ProviderAddress, payout.ProviderAddress)
	}
	if payouts[0].Settled {
		t.Fatal("provider payout should start unsettled")
	}

	if err := s.SettleProviderPayout(payouts[0].ID); err != nil {
		t.Fatalf("SettleProviderPayout: %v", err)
	}

	payouts, err = s.ListProviderPayouts()
	if err != nil {
		t.Fatalf("ListProviderPayouts after settle: %v", err)
	}
	if !payouts[0].Settled {
		t.Fatal("provider payout should be settled")
	}
}

func TestPostgresCreditProviderAccountAtomic(t *testing.T) {
	s := testPostgresStore(t)

	earning := &ProviderEarning{
		AccountID:        "acct-linked",
		ProviderID:       "provider-1",
		ProviderKey:      "key-1",
		JobID:            "job-atomic",
		Model:            "qwen3.5-9b",
		AmountMicroUSD:   123_000,
		PromptTokens:     10,
		CompletionTokens: 20,
	}
	if err := s.CreditProviderAccount(earning); err != nil {
		t.Fatalf("CreditProviderAccount: %v", err)
	}

	if bal := s.GetBalance("acct-linked"); bal != 123_000 {
		t.Fatalf("balance = %d, want 123000", bal)
	}

	history := s.LedgerHistory("acct-linked")
	if len(history) != 1 {
		t.Fatalf("ledger history = %d, want 1", len(history))
	}
	if history[0].Type != LedgerPayout {
		t.Fatalf("ledger entry type = %q, want payout", history[0].Type)
	}

	earnings, err := s.GetAccountEarnings("acct-linked", 10)
	if err != nil {
		t.Fatalf("GetAccountEarnings: %v", err)
	}
	if len(earnings) != 1 {
		t.Fatalf("earnings = %d, want 1", len(earnings))
	}
	if earnings[0].JobID != "job-atomic" {
		t.Fatalf("earning job_id = %q, want job-atomic", earnings[0].JobID)
	}
}

func TestPostgresCreditProviderWalletAtomic(t *testing.T) {
	s := testPostgresStore(t)

	payout := &ProviderPayout{
		ProviderAddress: "0xatomicwallet",
		AmountMicroUSD:  456_000,
		Model:           "llama-3",
		JobID:           "job-wallet",
	}
	if err := s.CreditProviderWallet(payout); err != nil {
		t.Fatalf("CreditProviderWallet: %v", err)
	}

	if bal := s.GetBalance("0xatomicwallet"); bal != 456_000 {
		t.Fatalf("wallet balance = %d, want 456000", bal)
	}

	history := s.LedgerHistory("0xatomicwallet")
	if len(history) != 1 {
		t.Fatalf("ledger history = %d, want 1", len(history))
	}
	if history[0].Type != LedgerPayout {
		t.Fatalf("ledger entry type = %q, want payout", history[0].Type)
	}

	payouts, err := s.ListProviderPayouts()
	if err != nil {
		t.Fatalf("ListProviderPayouts: %v", err)
	}
	if len(payouts) != 1 {
		t.Fatalf("provider payouts = %d, want 1", len(payouts))
	}
	if payouts[0].JobID != "job-wallet" {
		t.Fatalf("payout job_id = %q, want job-wallet", payouts[0].JobID)
	}
}

func TestPostgresStoreImplementsInterface(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set — skipping PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := NewPostgres(ctx, dbURL)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	defer s.Close()

	var _ Store = s
}

func TestPostgresProviderRecordStatsPersisted(t *testing.T) {
	s := testPostgresStore(t)

	rec := ProviderRecord{
		ID:                         "provider-1",
		Hardware:                   []byte(`{"chip":"M4 Max"}`),
		Models:                     []byte(`["model-a"]`),
		Backend:                    "vllm_mlx",
		TrustLevel:                 "hardware",
		Attested:                   true,
		SEPublicKey:                "se-key",
		SerialNumber:               "serial-1",
		LifetimeRequestsServed:     42,
		LifetimeTokensGenerated:    1234,
		LastSessionRequestsServed:  7,
		LastSessionTokensGenerated: 222,
		RegisteredAt:               time.Now(),
		LastSeen:                   time.Now(),
	}

	if err := s.UpsertProvider(context.Background(), rec); err != nil {
		t.Fatalf("UpsertProvider: %v", err)
	}

	got, err := s.GetProviderRecord(context.Background(), "provider-1")
	if err != nil {
		t.Fatalf("GetProviderRecord: %v", err)
	}

	if got.LifetimeRequestsServed != rec.LifetimeRequestsServed {
		t.Errorf("lifetime_requests_served = %d, want %d", got.LifetimeRequestsServed, rec.LifetimeRequestsServed)
	}
	if got.LifetimeTokensGenerated != rec.LifetimeTokensGenerated {
		t.Errorf("lifetime_tokens_generated = %d, want %d", got.LifetimeTokensGenerated, rec.LifetimeTokensGenerated)
	}
	if got.LastSessionRequestsServed != rec.LastSessionRequestsServed {
		t.Errorf("last_session_requests_served = %d, want %d", got.LastSessionRequestsServed, rec.LastSessionRequestsServed)
	}
	if got.LastSessionTokensGenerated != rec.LastSessionTokensGenerated {
		t.Errorf("last_session_tokens_generated = %d, want %d", got.LastSessionTokensGenerated, rec.LastSessionTokensGenerated)
	}
}
