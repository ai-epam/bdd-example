package authz

import (
	"slices"
	"time"

	"authgate/internal/domain"
)

// Reason codes returned on decline.
const (
	CardNotFound       = "CARD_NOT_FOUND"
	CardFrozen         = "CARD_FROZEN"
	CardExpired        = "CARD_EXPIRED"
	AccountNotFound    = "ACCOUNT_NOT_FOUND"
	AccountFrozen      = "ACCOUNT_FROZEN"
	MerchantNotFound   = "MERCHANT_NOT_FOUND"
	MCCBlocked         = "MCC_BLOCKED"
	CountryBlocked     = "COUNTRY_BLOCKED"
	CurrencyMismatch   = "CURRENCY_MISMATCH"
	AmountExceedsLimit = "AMOUNT_EXCEEDS_PER_TXN_LIMIT"
	VelocityExceeded   = "VELOCITY_LIMIT_EXCEEDED"
	DailyLimitExceeded = "DAILY_LIMIT_EXCEEDED"
	InsufficientFunds  = "INSUFFICIENT_FUNDS"
)

// Request is an authorization attempt.
type Request struct {
	CardID     string
	MerchantID string
	Amount     int64
	Currency   string
}

// Snapshot is everything a rule may look at. Nil pointers mean "not found".
type Snapshot struct {
	Req      Request
	Now      time.Time
	Card     *domain.Card
	Account  *domain.Account
	Merchant *domain.Merchant
	History  []domain.Transaction // all transactions on the card
}

// rule returns a reason code, or "" when the transaction passes.
type rule func(Snapshot) string

// rules in evaluation order; the first failure wins.
// TODO(refund): refunds are not implemented yet. The @wip scenario in
// features/regression.feature drives that work: add a refund rule/operation here.
var rules = []rule{
	func(s Snapshot) string { return failIf(s.Card == nil, CardNotFound) },
	func(s Snapshot) string { return failIf(s.Card.Status == "frozen", CardFrozen) },
	func(s Snapshot) string { return failIf(s.Card.Status == "expired", CardExpired) },
	func(s Snapshot) string { return failIf(s.Account == nil, AccountNotFound) },
	func(s Snapshot) string { return failIf(s.Account.Status == "frozen", AccountFrozen) },
	func(s Snapshot) string { return failIf(s.Merchant == nil, MerchantNotFound) },
	func(s Snapshot) string {
		return failIf(slices.Contains(s.Card.BlockedMCCs, s.Merchant.MCC), MCCBlocked)
	},
	func(s Snapshot) string {
		return failIf(len(s.Card.AllowedCountries) > 0 && !slices.Contains(s.Card.AllowedCountries, s.Merchant.Country), CountryBlocked)
	},
	func(s Snapshot) string { return failIf(s.Req.Currency != s.Account.Currency, CurrencyMismatch) },
	func(s Snapshot) string {
		return failIf(s.Card.PerTxnLimit > 0 && s.Req.Amount > s.Card.PerTxnLimit, AmountExceedsLimit)
	},
	velocity,
	dailyLimit,
	func(s Snapshot) string { return failIf(s.Req.Amount > s.Account.Balance, InsufficientFunds) },
}

func failIf(cond bool, code string) string {
	if cond {
		return code
	}
	return ""
}

// velocity declines once the card already has VelocityMaxTxns approved
// transactions inside the sliding window (so the N+1th is declined).
func velocity(s Snapshot) string {
	if s.Card.VelocityMaxTxns <= 0 || s.Card.VelocityWindowS <= 0 {
		return ""
	}
	cutoff := s.Now.Add(-time.Duration(s.Card.VelocityWindowS) * time.Second)
	n := 0
	for _, t := range s.History {
		if t.Status == "approved" && t.CreatedAt.After(cutoff) {
			n++
		}
	}
	return failIf(n >= s.Card.VelocityMaxTxns, VelocityExceeded)
}

// dailyLimit sums today's (UTC) approved spend plus this amount.
func dailyLimit(s Snapshot) string {
	if s.Card.DailyLimit <= 0 {
		return ""
	}
	y, m, d := s.Now.UTC().Date()
	var sum int64
	for _, t := range s.History {
		ty, tm, td := t.CreatedAt.UTC().Date()
		if t.Status == "approved" && ty == y && tm == m && td == d {
			sum += t.Amount
		}
	}
	return failIf(sum+s.Req.Amount > s.Card.DailyLimit, DailyLimitExceeded)
}
