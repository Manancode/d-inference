package domain

import "errors"

var (
	ErrChallengeNotFound   = errors.New("auth challenge not found")
	ErrChallengeExpired    = errors.New("auth challenge expired")
	ErrChallengeMismatch   = errors.New("auth challenge mismatch")
	ErrInvalidSignature    = errors.New("invalid signature")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrModelUnavailable    = errors.New("model unavailable")
	ErrNoCapacity          = errors.New("no capacity")
	ErrQuoteNotFound       = errors.New("quote not found")
	ErrQuoteExpired        = errors.New("quote expired")
	ErrQuoteConsumed       = errors.New("quote already consumed")
	ErrJobNotFound         = errors.New("job not found")
	ErrJobNotCompletable   = errors.New("job not completable")
	ErrProviderUnreachable = errors.New("provider unreachable")
)
