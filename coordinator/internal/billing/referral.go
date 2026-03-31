package billing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"unicode"

	"github.com/dginf/coordinator/internal/payments"
	"github.com/dginf/coordinator/internal/store"
)

// ReferralService manages the referral code system.
//
// Anyone can register as a referrer and receive a unique referral code.
// When a new user applies a referral code, a referral relationship is
// established. On each inference charge, the referrer earns a percentage
// of the platform fee.
//
// Fee split with referral:
//
//	Total cost  = 100%
//	Provider    = 95%
//	Platform    = 5% × (100% - referralSharePercent)
//	Referrer    = 5% × referralSharePercent
//
// Default referralSharePercent = 20, so:
//
//	Provider = 95%, Platform = 4%, Referrer = 1%
type ReferralService struct {
	store               store.Store
	ledger              *payments.Ledger
	logger              *slog.Logger
	referralSharePercent int64 // percentage of platform fee that goes to referrer
}

// NewReferralService creates a new referral service.
func NewReferralService(st store.Store, ledger *payments.Ledger, logger *slog.Logger, sharePercent int64) *ReferralService {
	if sharePercent <= 0 || sharePercent > 50 {
		sharePercent = 20
	}
	return &ReferralService{
		store:               st,
		ledger:              ledger,
		logger:              logger,
		referralSharePercent: sharePercent,
	}
}

// SharePercent returns the current referral share percentage.
func (r *ReferralService) SharePercent() int64 {
	return r.referralSharePercent
}

// Register creates a referral code for an account. If the account already
// has a code, it returns the existing one.
func (r *ReferralService) Register(accountID string) (*store.Referrer, error) {
	// Check if already registered
	existing, err := r.store.GetReferrerByAccount(accountID)
	if err == nil {
		return existing, nil
	}

	// Generate a unique referral code
	code, err := generateReferralCode()
	if err != nil {
		return nil, fmt.Errorf("referral: generate code: %w", err)
	}

	if err := r.store.CreateReferrer(accountID, code); err != nil {
		return nil, fmt.Errorf("referral: create referrer: %w", err)
	}

	r.logger.Info("referral: new referrer registered",
		"account", truncateID(accountID),
		"code", code,
	)

	return &store.Referrer{
		AccountID: accountID,
		Code:      code,
	}, nil
}

// Apply links an account to a referral code. The account must not already
// have a referrer, and the account cannot refer itself.
func (r *ReferralService) Apply(accountID, referralCode string) error {
	referralCode = strings.ToUpper(strings.TrimSpace(referralCode))
	if referralCode == "" {
		return fmt.Errorf("referral: code is required")
	}

	// Validate the referral code exists
	referrer, err := r.store.GetReferrerByCode(referralCode)
	if err != nil {
		return fmt.Errorf("referral: invalid code %q", referralCode)
	}

	// Prevent self-referral
	if referrer.AccountID == accountID {
		return fmt.Errorf("referral: cannot refer yourself")
	}

	// Check if account already has a referrer
	existing, err := r.store.GetReferrerForAccount(accountID)
	if err == nil && existing != "" {
		return fmt.Errorf("referral: account already has a referrer")
	}

	// Record the referral
	if err := r.store.RecordReferral(referralCode, accountID); err != nil {
		return fmt.Errorf("referral: record referral: %w", err)
	}

	r.logger.Info("referral: code applied",
		"account", truncateID(accountID),
		"referrer_code", referralCode,
	)

	return nil
}

// Stats returns referral statistics for the given account.
func (r *ReferralService) Stats(accountID string) (*ReferralStatsResponse, error) {
	referrer, err := r.store.GetReferrerByAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("referral: account is not a referrer")
	}

	stats, err := r.store.GetReferralStats(referrer.Code)
	if err != nil {
		return nil, fmt.Errorf("referral: get stats: %w", err)
	}

	balance := r.store.GetBalance(accountID)

	return &ReferralStatsResponse{
		Code:                 referrer.Code,
		SharePercent:         r.referralSharePercent,
		TotalReferred:        stats.TotalReferred,
		TotalRewardsMicroUSD: stats.TotalRewardsMicroUSD,
		TotalRewardsUSD:      fmt.Sprintf("%.6f", float64(stats.TotalRewardsMicroUSD)/1_000_000),
		BalanceMicroUSD:      balance,
		BalanceUSD:           fmt.Sprintf("%.6f", float64(balance)/1_000_000),
	}, nil
}

// DistributeReferralReward checks if a consumer has a referrer and distributes
// the referral share of the platform fee. Returns the adjusted platform fee
// (after deducting the referral reward).
//
// Call this during inference billing, after calculating the platform fee.
func (r *ReferralService) DistributeReferralReward(consumerKey string, platformFee int64, jobID string) int64 {
	// Check if consumer was referred
	referrerCode, err := r.store.GetReferrerForAccount(consumerKey)
	if err != nil || referrerCode == "" {
		return platformFee // no referrer, full platform fee
	}

	// Look up the referrer account
	referrer, err := r.store.GetReferrerByCode(referrerCode)
	if err != nil {
		return platformFee // referrer not found, full platform fee
	}

	// Calculate referral reward: X% of platform fee
	referralReward := platformFee * r.referralSharePercent / 100
	if referralReward <= 0 {
		return platformFee
	}

	// Credit the referrer
	if err := r.store.Credit(referrer.AccountID, referralReward, store.LedgerReferralReward, jobID); err != nil {
		r.logger.Error("referral: failed to credit reward",
			"referrer", truncateID(referrer.AccountID),
			"reward", referralReward,
			"error", err,
		)
		return platformFee
	}

	r.logger.Debug("referral: reward distributed",
		"referrer", truncateID(referrer.AccountID),
		"consumer", truncateID(consumerKey),
		"reward_micro_usd", referralReward,
		"job_id", jobID,
	)

	return platformFee - referralReward
}

// ReferralStatsResponse is the API response for referral statistics.
type ReferralStatsResponse struct {
	Code                 string `json:"code"`
	SharePercent         int64  `json:"share_percent"`
	TotalReferred        int    `json:"total_referred"`
	TotalRewardsMicroUSD int64  `json:"total_rewards_micro_usd"`
	TotalRewardsUSD      string `json:"total_rewards_usd"`
	BalanceMicroUSD      int64  `json:"balance_micro_usd"`
	BalanceUSD           string `json:"balance_usd"`
}

// generateReferralCode creates a random alphanumeric referral code.
// Format: DGINF-XXXXXX (uppercase alphanumeric, 6 random chars)
func generateReferralCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	raw := hex.EncodeToString(b)

	// Convert to uppercase and filter to alphanumeric
	var code strings.Builder
	code.WriteString("DGINF-")
	for _, ch := range strings.ToUpper(raw) {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			code.WriteRune(ch)
		}
	}

	return code.String(), nil
}

// truncateID shortens an ID for logging (shows first 8 chars).
func truncateID(id string) string {
	if len(id) <= 10 {
		return id
	}
	return id[:8] + "..."
}
