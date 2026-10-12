package store

import "authgate/internal/domain"

func seed(t *Tx) {
	for _, a := range []domain.Account{
		{ID: "ACC-001", Holder: "Alice Nguyen", Balance: 50000, Currency: "USD", Status: "active"},
		{ID: "ACC-002", Holder: "Bob Carter", Balance: 5000, Currency: "USD", Status: "active"},
		{ID: "ACC-003", Holder: "Carol Diaz", Balance: 20000, Currency: "USD", Status: "frozen"},
	} {
		t.PutAccount(a)
	}
	for _, c := range []domain.Card{
		{ID: "CARD-001", AccountID: "ACC-001", Status: "active", PerTxnLimit: 20000, DailyLimit: 40000, VelocityMaxTxns: 5, VelocityWindowS: 60, BlockedMCCs: []string{"7995", "6051"}},
		{ID: "CARD-002", AccountID: "ACC-001", Status: "frozen", PerTxnLimit: 20000, DailyLimit: 40000, VelocityMaxTxns: 5, VelocityWindowS: 60},
		{ID: "CARD-003", AccountID: "ACC-002", Status: "active", VelocityMaxTxns: 5, VelocityWindowS: 60, BlockedMCCs: []string{"7995"}},
		{ID: "CARD-004", AccountID: "ACC-002", Status: "active", PerTxnLimit: 2000, DailyLimit: 10000, VelocityMaxTxns: 3, VelocityWindowS: 60, AllowedCountries: []string{"US"}},
	} {
		t.PutCard(c)
	}
	for _, m := range []domain.Merchant{
		{ID: "MERCH-COFFEE", Name: "Blue Bottle Coffee", MCC: "5812", Country: "US"},
		{ID: "MERCH-GROCERY", Name: "Whole Foods", MCC: "5411", Country: "US"},
		{ID: "MERCH-GAMBLING", Name: "LuckyBet", MCC: "7995", Country: "US"},
		{ID: "MERCH-CRYPTO", Name: "CoinRush", MCC: "6051", Country: "US"},
		{ID: "MERCH-FOREIGN", Name: "Kimchi Mart", MCC: "5411", Country: "KR"},
	} {
		t.PutMerchant(m)
	}
}
