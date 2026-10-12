// Package store is a thread-safe in-memory key/value store with typed helpers.
package store

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"authgate/internal/domain"
)

type Store struct {
	mu  sync.RWMutex
	kv  map[string]any
	now func() time.Time
	seq int
}

// New returns a store populated with the seed data.
func New() *Store {
	s := &Store{now: time.Now}
	s.Reset()
	return s
}

// Tx is the lock-free view handed to Update/View callbacks.
type Tx struct{ s *Store }

// Update runs fn under the write lock, making multi-step changes atomic.
func (s *Store) Update(fn func(tx *Tx)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&Tx{s})
}

// View runs fn under the read lock.
func (s *Store) View(fn func(tx *Tx)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(&Tx{s})
}

// SetClock replaces the time source (tests); nil restores time.Now.
func (s *Store) SetClock(now func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now == nil {
		now = time.Now
	}
	s.now = now
}

// Reset restores the deterministic seed state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kv = map[string]any{}
	s.seq = 0
	seed(&Tx{s})
}

func (t *Tx) Now() time.Time { return t.s.now() }

func (t *Tx) NextTxnID() string {
	t.s.seq++
	return fmt.Sprintf("TXN-%06d", t.s.seq)
}

func (t *Tx) PutAccount(a domain.Account)   { t.s.kv["account:"+a.ID] = a }
func (t *Tx) PutMerchant(m domain.Merchant) { t.s.kv["merchant:"+m.ID] = m }

func (t *Tx) PutCard(c domain.Card) {
	c.BlockedMCCs = slices.Clone(c.BlockedMCCs)
	c.AllowedCountries = slices.Clone(c.AllowedCountries)
	t.s.kv["card:"+c.ID] = c
}

func (t *Tx) PutTransaction(x domain.Transaction) { t.s.kv["txn:"+x.ID] = x }

func (t *Tx) Account(id string) (domain.Account, bool) {
	a, ok := t.s.kv["account:"+id].(domain.Account)
	return a, ok
}

func (t *Tx) Merchant(id string) (domain.Merchant, bool) {
	m, ok := t.s.kv["merchant:"+id].(domain.Merchant)
	return m, ok
}

func (t *Tx) Card(id string) (domain.Card, bool) {
	c, ok := t.s.kv["card:"+id].(domain.Card)
	if ok {
		c.BlockedMCCs = slices.Clone(c.BlockedMCCs)
		c.AllowedCountries = slices.Clone(c.AllowedCountries)
	}
	return c, ok
}

func collect[T any](t *Tx, prefix string) []T {
	var out []T
	for k, v := range t.s.kv {
		if strings.HasPrefix(k, prefix) {
			out = append(out, v.(T))
		}
	}
	return out
}

func (t *Tx) Accounts() []domain.Account {
	out := collect[domain.Account](t, "account:")
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (t *Tx) Cards() []domain.Card {
	out := collect[domain.Card](t, "card:")
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (t *Tx) Merchants() []domain.Merchant {
	out := collect[domain.Merchant](t, "merchant:")
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortNewestFirst(out []domain.Transaction) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
}

// CardTransactions returns the card's transactions, newest first.
func (t *Tx) CardTransactions(cardID string) []domain.Transaction {
	var out []domain.Transaction
	for _, x := range collect[domain.Transaction](t, "txn:") {
		if x.CardID == cardID {
			out = append(out, x)
		}
	}
	sortNewestFirst(out)
	return out
}

// AccountTransactions returns transactions of all cards on the account, newest first.
func (t *Tx) AccountTransactions(accountID string) []domain.Transaction {
	ids := map[string]bool{}
	for _, c := range t.Cards() {
		if c.AccountID == accountID {
			ids[c.ID] = true
		}
	}
	var out []domain.Transaction
	for _, x := range collect[domain.Transaction](t, "txn:") {
		if ids[x.CardID] {
			out = append(out, x)
		}
	}
	sortNewestFirst(out)
	return out
}

// Convenience wrappers that take the lock themselves.

func (s *Store) Account(id string) (a domain.Account, ok bool) {
	s.View(func(t *Tx) { a, ok = t.Account(id) })
	return
}

func (s *Store) Card(id string) (c domain.Card, ok bool) {
	s.View(func(t *Tx) { c, ok = t.Card(id) })
	return
}

// MutateAccount applies fn to an existing account; it reports whether it existed.
func (s *Store) MutateAccount(id string, fn func(*domain.Account)) (ok bool) {
	s.Update(func(t *Tx) {
		var a domain.Account
		if a, ok = t.Account(id); ok {
			fn(&a)
			t.PutAccount(a)
		}
	})
	return
}

func (s *Store) MutateCard(id string, fn func(*domain.Card)) (ok bool) {
	s.Update(func(t *Tx) {
		var c domain.Card
		if c, ok = t.Card(id); ok {
			fn(&c)
			t.PutCard(c)
		}
	})
	return
}

func (s *Store) PutMerchant(m domain.Merchant) { s.Update(func(t *Tx) { t.PutMerchant(m) }) }
func (s *Store) PutCard(c domain.Card)         { s.Update(func(t *Tx) { t.PutCard(c) }) }

// InsertTransaction stores a transaction as-is (used to back-date history in tests).
func (s *Store) InsertTransaction(x domain.Transaction) string {
	var id string
	s.Update(func(t *Tx) {
		if x.ID == "" {
			x.ID = t.NextTxnID()
		}
		id = x.ID
		t.PutTransaction(x)
	})
	return id
}
