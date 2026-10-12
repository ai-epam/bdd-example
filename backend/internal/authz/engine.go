// Package authz decides whether a card transaction is authorized.
package authz

import (
	"authgate/internal/domain"
	"authgate/internal/store"
)

// Result is the outcome of an authorization.
type Result struct {
	Approved      bool   `json:"approved"`
	Reason        string `json:"reason"`
	TransactionID string `json:"transactionId"`
	NewBalance    int64  `json:"newBalance"`
}

// Evaluate runs the rules against a snapshot and returns the first failure
// reason, or "" if approved. It has no side effects.
func Evaluate(s Snapshot) string {
	for _, r := range rules {
		if reason := r(s); reason != "" {
			return reason
		}
	}
	return ""
}

// Authorize evaluates req and records the transaction. Evaluation, persistence
// and the debit happen under a single store lock, so concurrent calls are safe.
func Authorize(st *store.Store, req Request) (Result, error) {
	var res Result
	st.Update(func(tx *store.Tx) {
		snap := Snapshot{Req: req, Now: tx.Now()}
		if c, ok := tx.Card(req.CardID); ok {
			snap.Card = &c
			snap.History = tx.CardTransactions(c.ID)
			if a, ok := tx.Account(c.AccountID); ok {
				snap.Account = &a
			}
		}
		if m, ok := tx.Merchant(req.MerchantID); ok {
			snap.Merchant = &m
		}

		reason := Evaluate(snap)
		txn := domain.Transaction{
			ID: tx.NextTxnID(), CardID: req.CardID, MerchantID: req.MerchantID,
			Amount: req.Amount, Currency: req.Currency, CreatedAt: snap.Now,
			Status: "declined", Reason: reason,
		}
		res = Result{Reason: reason, TransactionID: txn.ID}
		if snap.Account != nil {
			res.NewBalance = snap.Account.Balance
		}
		if reason == "" {
			txn.Status = "approved"
			acc := *snap.Account
			acc.Balance -= req.Amount
			tx.PutAccount(acc)
			res.Approved = true
			res.NewBalance = acc.Balance
		}
		// Unknown cards have no history to attach to, but we still record them.
		tx.PutTransaction(txn)
	})
	return res, nil
}
