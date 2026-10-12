package steps

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cucumber/godog"

	"authgate/internal/authz"
	"authgate/internal/domain"
)

// InitializeScenario registers hooks and step definitions.
func InitializeScenario(sc *godog.ScenarioContext) {
	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		return context.WithValue(ctx, worldKey{}, newWorld()), nil
	})
	sc.After(func(ctx context.Context, s *godog.Scenario, err error) (context.Context, error) {
		Report.record(s, err, time.Since(from(ctx).start))
		return ctx, nil
	})

	sc.Step(`^the system is seeded with default data$`, seeded)
	sc.Step(`^card "([^"]*)" is active with balance "([^"]*)" ([A-Z]{3})$`, cardActiveWithBalance)
	sc.Step(`^card "([^"]*)" is frozen$`, cardFrozen)
	sc.Step(`^card "([^"]*)" has a per-transaction limit of "([^"]*)" ([A-Z]{3})$`, perTxnLimit)
	sc.Step(`^card "([^"]*)" has a daily limit of "([^"]*)" ([A-Z]{3})$`, dailyLimit)
	sc.Step(`^merchant "([^"]*)" has MCC "([^"]*)" in country "([^"]*)"$`, merchantHas)
	sc.Step(`^card "([^"]*)" on account "([^"]*)" does not block any MCC$`, newOpenCard)
	sc.Step(`^(\d+) approved transactions? on card "([^"]*)" in the last (\d+) seconds$`, approvedInLast)
	sc.Step(`^(\d+) approved transactions? on card "([^"]*)" (\d+) seconds ago$`, approvedAgo)
	sc.Step(`^I authorize "([^"]*)" ([A-Z]{3}) on card "([^"]*)" at merchant "([^"]*)"$`, authorize)
	registerAuthSteps(sc)
	sc.Step(`^the transaction is "(approved|declined)"$`, transactionIs)
	sc.Step(`^the decline reason is "([^"]*)"$`, declineReason)
	sc.Step(`^the account balance of "([^"]*)" becomes "([^"]*)" ([A-Z]{3})$`, balanceBecomes)
	sc.Step(`^the account balance of "([^"]*)" is unchanged$`, balanceUnchanged)
}

func seeded(ctx context.Context) {
	w := from(ctx)
	w.store.Reset()
	w.auth.Reset()
}

func cardActiveWithBalance(ctx context.Context, cardID, amount, cur string) error {
	w := from(ctx)
	cents, err := parseMoney(amount)
	if err != nil {
		return err
	}
	c, ok := w.store.Card(cardID)
	if !ok {
		return fmt.Errorf("card %s not in store", cardID)
	}
	w.store.MutateCard(cardID, func(c *domain.Card) { c.Status = "active" })
	w.store.MutateAccount(c.AccountID, func(a *domain.Account) {
		a.Status, a.Balance, a.Currency = "active", cents, cur
	})
	return nil
}

func cardFrozen(ctx context.Context, cardID string) error {
	if !from(ctx).store.MutateCard(cardID, func(c *domain.Card) { c.Status = "frozen" }) {
		return fmt.Errorf("card %s not in store", cardID)
	}
	return nil
}

func perTxnLimit(ctx context.Context, cardID, amount, _ string) error {
	cents, err := parseMoney(amount)
	if err != nil {
		return err
	}
	if !from(ctx).store.MutateCard(cardID, func(c *domain.Card) { c.PerTxnLimit = cents }) {
		return fmt.Errorf("card %s not in store", cardID)
	}
	return nil
}

func dailyLimit(ctx context.Context, cardID, amount, _ string) error {
	cents, err := parseMoney(amount)
	if err != nil {
		return err
	}
	if !from(ctx).store.MutateCard(cardID, func(c *domain.Card) { c.DailyLimit = cents }) {
		return fmt.Errorf("card %s not in store", cardID)
	}
	return nil
}

func merchantHas(ctx context.Context, id, mcc, country string) {
	from(ctx).store.PutMerchant(domain.Merchant{ID: id, Name: id, MCC: mcc, Country: country})
}

func newOpenCard(ctx context.Context, cardID, accountID string) error {
	w := from(ctx)
	if _, ok := w.store.Account(accountID); !ok {
		return fmt.Errorf("account %s not in store", accountID)
	}
	w.store.PutCard(domain.Card{ID: cardID, AccountID: accountID, Status: "active", VelocityMaxTxns: 5, VelocityWindowS: 60})
	return nil
}

func insertApproved(w *world, n int, cardID string, age time.Duration) error {
	c, ok := w.store.Card(cardID)
	if !ok {
		return fmt.Errorf("card %s not in store", cardID)
	}
	cur := "USD"
	if a, ok := w.store.Account(c.AccountID); ok {
		cur = a.Currency
	}
	for i := 0; i < n; i++ {
		w.store.InsertTransaction(domain.Transaction{
			CardID: cardID, MerchantID: "MERCH-COFFEE", Amount: 100, Currency: cur,
			CreatedAt: w.now.Add(-age), Status: "approved",
		})
	}
	return nil
}

// approvedInLast places the history at the midpoint of the given span.
func approvedInLast(ctx context.Context, n int, cardID string, secs int) error {
	return insertApproved(from(ctx), n, cardID, time.Duration(secs)*time.Second/2)
}

func approvedAgo(ctx context.Context, n int, cardID string, secs int) error {
	return insertApproved(from(ctx), n, cardID, time.Duration(secs)*time.Second)
}

func authorize(ctx context.Context, amount, cur, cardID, merchantID string) error {
	w := from(ctx)
	cents, err := parseMoney(amount)
	if err != nil {
		return err
	}
	if c, ok := w.store.Card(cardID); ok {
		if a, ok := w.store.Account(c.AccountID); ok {
			w.before[a.ID] = a.Balance
		}
	}
	w.last, err = authz.Authorize(w.store, authz.Request{CardID: cardID, MerchantID: merchantID, Amount: cents, Currency: cur})
	w.ran = true
	return err
}

func transactionIs(ctx context.Context, want string) error {
	w := from(ctx)
	if !w.ran {
		return errors.New("no authorization was performed")
	}
	got := "declined"
	if w.last.Approved {
		got = "approved"
	}
	if got != want {
		return fmt.Errorf("expected transaction to be %s but it was %s (reason %q)", want, got, w.last.Reason)
	}
	return nil
}

func declineReason(ctx context.Context, want string) error {
	if got := from(ctx).last.Reason; got != want {
		return fmt.Errorf("expected decline reason %q but got %q", want, got)
	}
	return nil
}

func balanceOf(w *world, accountID string) (int64, error) {
	a, ok := w.store.Account(accountID)
	if !ok {
		return 0, fmt.Errorf("account %s not in store", accountID)
	}
	return a.Balance, nil
}

func balanceBecomes(ctx context.Context, accountID, amount, _ string) error {
	want, err := parseMoney(amount)
	if err != nil {
		return err
	}
	got, err := balanceOf(from(ctx), accountID)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("expected balance of %s to be %d cents but it is %d", accountID, want, got)
	}
	return nil
}

func balanceUnchanged(ctx context.Context, accountID string) error {
	w := from(ctx)
	before, ok := w.before[accountID]
	if !ok {
		return fmt.Errorf("no authorization touched account %s", accountID)
	}
	got, err := balanceOf(w, accountID)
	if err != nil {
		return err
	}
	if got != before {
		return fmt.Errorf("expected balance of %s to stay %d cents but it is %d", accountID, before, got)
	}
	return nil
}
