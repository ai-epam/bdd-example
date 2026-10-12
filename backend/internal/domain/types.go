// Package domain holds the core types. All money is int64 minor units (cents).
package domain

import "time"

type Account struct {
	ID       string `json:"id"`
	Holder   string `json:"holder"`
	Balance  int64  `json:"balance"`
	Currency string `json:"currency"`
	Status   string `json:"status"` // active | frozen
}

type Card struct {
	ID               string   `json:"id"`
	AccountID        string   `json:"accountId"`
	Status           string   `json:"status"` // active | frozen | expired
	PerTxnLimit      int64    `json:"perTxnLimit"`
	DailyLimit       int64    `json:"dailyLimit"`
	VelocityMaxTxns  int      `json:"velocityMaxTxns"`
	VelocityWindowS  int      `json:"velocityWindowS"`
	BlockedMCCs      []string `json:"blockedMccs"`
	AllowedCountries []string `json:"allowedCountries"`
}

type Merchant struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	MCC     string `json:"mcc"`
	Country string `json:"country"`
}

type Transaction struct {
	ID         string    `json:"id"`
	CardID     string    `json:"cardId"`
	MerchantID string    `json:"merchantId"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"createdAt"`
	Status     string    `json:"status"` // approved | declined
	Reason     string    `json:"reason"`
}
