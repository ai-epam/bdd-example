# BDD Demo Project — "AuthGate"

Below is a copy-paste-ready brief for Claude Code. It's opinionated so the agent doesn't wander.

---

## 0. Mission

Build **AuthGate**, a small but realistic **card transaction authorization engine** (fintech domain) that exists for one purpose: to demonstrate **Behavior-Driven Development** end-to-end.

The demo must make the BDD loop obvious and tangible:

1. A business rule is written in **Gherkin** (`.feature` file).
2. **godog** runs it, it fails (red).
3. The rule is implemented in Go.
4. The scenario goes green.
5. The same behavior is also observable in a **live web UI** where a human can click through the exact same scenario by hand.

No Docker. No external DB. In-memory KV store with deterministic seed data.

---

## 1. Why this domain

Card authorization is ideal for BDD because the business rules are *literally* what risk/compliance/product people argue about in meetings:

- "A frozen card must never authorize."
- "No more than 5 transactions per minute per card."
- "Gambling MCC 7995 is blocked."
- "A transaction over the card's per-txn limit must decline with `AMOUNT_EXCEEDS_PER_TXN_LIMIT`."

These translate one-to-one into Gherkin scenarios — no impedance mismatch, no "we'll write the tests later."

---

## 2. Tech stack (locked)

| Layer | Choice | Notes |
|---|---|---|
| Backend | **Go 1.22+** | stdlib `net/http` + `chi` router |
| BDD | **godog** (`github.com/cucumber/godog`) v0.14+ | Cucumber for Go |
| Storage | **In-memory thread-safe KV** | `map[string]any` guarded by `sync.RWMutex`. No DB. |
| Frontend | **Svelte 5 + Vite + TypeScript** | Smallest boilerplate, no router library needed (two tabs). |
| Dev orchestration | **Makefile** | `make dev`, `make bdd`, `make test` |

Do **not** add: Docker, Postgres, Redis, ORMs, GraphQL, auth, telemetry, k8s, monorepo tooling.

---

## 3. Project layout

```
authgate/
├── Makefile
├── README.md
├── backend/
│   ├── go.mod
│   ├── cmd/server/main.go                 # HTTP server bootstrap
│   ├── internal/
│   │   ├── domain/types.go                # Account, Card, Merchant, Transaction
│   │   ├── store/store.go                 # thread-safe in-mem KV
│   │   ├── store/seed.go                  # deterministic seed data
│   │   ├── authz/engine.go                # Authorize(...) (Result, error)
│   │   ├── authz/rules.go                 # individual rule funcs + reason codes
│   │   └── api/handlers.go                # HTTP handlers
│   ├── features/
│   │   ├── authorization.feature
│   │   ├── velocity.feature
│   │   ├── merchant_rules.feature
│   │   └── settlement.feature
│   ├── steps/
│   │   └── steps.go                       # godog step definitions
│   └── features_test.go                   # godog TestSuite entry point
└── frontend/
    ├── package.json
    ├── vite.config.ts                     # proxy /api -> :8080
    ├── index.html
    └── src/
        ├── main.ts
        ├── App.svelte
        ├── lib/api.ts
        └── routes/
            ├── Playground.svelte          # authorize a txn manually
            └── Scenarios.svelte           # render .feature files + last run report
```

---

## 4. Domain model

All money amounts are **int64 in minor units** (cents). Never floats.

```go
type Account struct {
    ID       string   // "ACC-001"
    Holder   string   // "Alice Nguyen"
    Balance  int64    // cents
    Currency string   // "USD"
    Status   string   // "active" | "frozen"
}

type Card struct {
    ID               string
    AccountID        string
    Status           string   // "active" | "frozen" | "expired"
    PerTxnLimit      int64    // cents, 0 = no limit
    DailyLimit       int64    // cents, 0 = no limit
    VelocityMaxTxns  int      // e.g. 5
    VelocityWindowS  int      // e.g. 60 (seconds)
    BlockedMCCs      []string // e.g. ["7995","6051"]
    AllowedCountries []string // empty = allow all
}

type Merchant struct {
    ID      string // "MERCH-COFFEE"
    Name    string // "Blue Bottle Coffee"
    MCC     string // "5812"
    Country string // "US"
}

type Transaction struct {
    ID         string
    CardID     string
    MerchantID string
    Amount     int64
    Currency   string
    CreatedAt  time.Time
    Status     string // "approved" | "declined"
    Reason     string // reason code, empty if approved
}
```

---

## 5. Authorization rules (evaluated in this order, first failure wins)

| # | Reason code | Condition |
|---|---|---|
| 1 | `CARD_NOT_FOUND` | Card ID not in store |
| 2 | `CARD_FROZEN` | `card.Status == "frozen"` |
| 3 | `CARD_EXPIRED` | `card.Status == "expired"` |
| 4 | `ACCOUNT_NOT_FOUND` | Account for card missing |
| 5 | `ACCOUNT_FROZEN` | `account.Status == "frozen"` |
| 6 | `MERCHANT_NOT_FOUND` | Merchant ID not in store |
| 7 | `MCC_BLOCKED` | Merchant MCC in `card.BlockedMCCs` |
| 8 | `COUNTRY_BLOCKED` | `len(card.AllowedCountries) > 0` and merchant country not in list |
| 9 | `CURRENCY_MISMATCH` | Tx currency ≠ account currency |
| 10 | `AMOUNT_EXCEEDS_PER_TXN_LIMIT` | `PerTxnLimit > 0 && amount > PerTxnLimit` |
| 11 | `VELOCITY_LIMIT_EXCEEDED` | More than `VelocityMaxTxns` approved txns on this card in last `VelocityWindowS` seconds |
| 12 | `DAILY_LIMIT_EXCEEDED` | Sum of today's approved txns + amount > `DailyLimit` |
| 13 | `INSUFFICIENT_FUNDS` | `amount > account.Balance` |

On approval: debit `account.Balance`, persist `Transaction{Status:"approved"}`, return balance.
On decline: persist `Transaction{Status:"declined", Reason: code}`, do **not** debit.

The rule engine must be **pure-ish**: takes a snapshot from the store, returns a `Result`. This makes it trivially testable from both godog steps and unit tests.

```go
type Result struct {
    Approved       bool
    Reason         string
    TransactionID  string
    NewBalance     int64
}
```

---

## 6. Seed data (deterministic, defined in `store/seed.go`)

**Accounts**
| ID | Holder | Balance | Currency | Status |
|---|---|---|---|---|
| ACC-001 | Alice Nguyen | 500.00 | USD | active |
| ACC-002 | Bob Carter | 50.00 | USD | active |
| ACC-003 | Carol Diaz | 200.00 | USD | frozen |

**Cards**
| ID | Account | Status | PerTxn | Daily | Velocity | BlockedMCCs | AllowedCountries |
|---|---|---|---|---|---|---|---|
| CARD-001 | ACC-001 | active | 200.00 | 400.00 | 5/60s | 7995, 6051 | (any) |
| CARD-002 | ACC-001 | frozen | 200.00 | 400.00 | 5/60s | — | (any) |
| CARD-003 | ACC-002 | active | 0 | 0 | 5/60s | 7995 | (any) |
| CARD-004 | ACC-002 | active | 20.00 | 100.00 | 3/60s | — | ["US"] |

**Merchants**
| ID | Name | MCC | Country |
|---|---|---|---|
| MERCH-COFFEE | Blue Bottle Coffee | 5812 | US |
| MERCH-GROCERY | Whole Foods | 5411 | US |
| MERCH-GAMBLING | LuckyBet | 7995 | US |
| MERCH-CRYPTO | CoinRush | 6051 | US |
| MERCH-FOREIGN | Kimchi Mart | 5411 | KR |

A `store.Reset()` function must restore this exact state — the BDD suite calls it in the `Background` step of every feature so scenarios are independent.

---

## 7. HTTP API

All endpoints JSON. Bind to `:8080`. CORS: allow `http://localhost:5173`.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/authorize` | Body: `{cardId, merchantId, amountCents, currency}` → `Result` |
| `GET`  | `/api/seed` | Returns all accounts, cards, merchants (for UI dropdowns) |
| `GET`  | `/api/accounts/{id}` | Account + recent txns |
| `GET`  | `/api/cards/{id}/transactions` | Last 20 txns for a card |
| `POST` | `/api/reset` | Restores seed state (used by UI "Reset" button) |
| `GET`  | `/api/features` | Returns `[]string` — raw contents of every `.feature` file |
| `GET`  | `/api/bdd-report` | Returns contents of `bdd-report.json` if present, else `null` |
| `POST` | `/api/bdd/run` | **Only if `ENABLE_BDD_RUNNER=1`.** Shells out to `go test ./... -args --format=json > bdd-report.json`, returns the report. Guarded so it's off by default. |

---

## 8. BDD / godog setup

### `backend/features_test.go`

```go
package main

import (
    "os"
    "testing"
    "github.com/cucumber/godog"
    "github.com/cucumber/godog/colors"
)

func TestFeatures(t *testing.T) {
    opts := godog.Options{
        Output: colors.Colored(os.Stdout),
        Format: "pretty", // and emit JSON via a custom formatter to bdd-report.json
        Paths:  []string{"features"},
        TestingT: t,
    }
    if st := godog.TestSuite{
        Name: "authgate",
        ScenarioInitializer: InitializeScenario,
        Options: &opts,
    }.Run(); st != 0 {
        t.Fatal("BDD suite failed")
    }
}
```

Also register a **custom formatter** that writes `../bdd-report.json` with a compact shape:

```json
{
  "generatedAt": "2025-01-01T00:00:00Z",
  "passed": 14, "failed": 1, "skipped": 0,
  "features": [
    {
      "name": "Card transaction authorization",
      "path": "features/authorization.feature",
      "scenarios": [
        {"name": "Approve a valid purchase", "status": "passed", "durationMs": 2}
      ]
    }
  ]
}
```

### `backend/steps/steps.go`

Step definitions must be **domain-level**, not HTTP-level. They call `authz.Authorize(...)` directly against a freshly reset store. (The HTTP layer has its own thin integration test.)

Suggested steps (implement all, plus any you need):

- `Given the system is seeded with default data` → `store.Reset()`
- `Given card {string} is active with balance {string} {string}`
- `Given card {string} is frozen`
- `Given card {string} has a per-transaction limit of {string} {string}`
- `Given merchant {string} has MCC {string} in country {string}`
- `Given {int} approved transactions on card {string} in the last {int} seconds`
- `When I authorize {string} {string} on card {string} at merchant {string}`
- `Then the transaction is {string}` (approved | declined)
- `And the decline reason is {string}`
- `And the account balance of {string} becomes {string} {string}`
- `And the account balance of {string} is unchanged`

Use `godog.ScenarioContext` with typed context storage; never use globals between steps.

---

## 9. Feature files (write all of these)

Write full Gherkin. Use `Background: Given the system is seeded with default data` in every feature.

### `features/authorization.feature`
Style reference — get this one exactly right:

```gherkin
Feature: Card transaction authorization
  As a payments risk manager
  I want every card transaction to be checked against our authorization policy
  So that only compliant spend is approved

  Background:
    Given the system is seeded with default data

  Scenario: Approve a valid coffee purchase within all limits
    Given card "CARD-001" is active with balance "500.00" USD
    And merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "12.50" USD on card "CARD-001" at merchant "MERCH-COFFEE"
    Then the transaction is "approved"
    And the account balance of "ACC-001" becomes "487.50" USD

  Scenario: Decline when the card is frozen
    Given card "CARD-002" is frozen
    And merchant "MERCH-COFFEE" has MCC "5812" in country "US"
    When I authorize "12.50" USD on card "CARD-002" at merchant "MERCH-COFFEE"
    Then the transaction is "declined"
    And the decline reason is "CARD_FROZEN"

  Scenario: Decline when the amount exceeds the per-transaction limit
    Given card "CARD-004" has a per-transaction limit of "20.00" USD
    And merchant "MERCH-GROCERY" has MCC "5411" in country "US"
    When I authorize "25.00" USD on card "CARD-004" at merchant "MERCH-GROCERY"
    Then the transaction is "declined"
    And the decline reason is "AMOUNT_EXCEEDS_PER_TXN_LIMIT"

  Scenario: Decline when the account has insufficient funds
    Given merchant "MERCH-GROCERY" has MCC "5411" in country "US"
    When I authorize "75.00" USD on card "CARD-003" at merchant "MERCH-GROCERY"
    Then the transaction is "declined"
    And the decline reason is "INSUFFICIENT_FUNDS"
```

### `features/velocity.feature`
Scenarios to cover:
- Approve when under the velocity window
- Decline with `VELOCITY_LIMIT_EXCEEDED` when the Nth+1 txn hits inside 60s
- Velocity window **slides**: a txn older than 60s no longer counts
- Declined txns do **not** count toward velocity

### `features/merchant_rules.feature`
- Decline MCC 7995 (gambling) with `MCC_BLOCKED` on CARD-001
- Approve MCC 7995 on a card that does **not** block it (create one inline)
- Decline a KR merchant on CARD-004 (allowed countries = ["US"]) with `COUNTRY_BLOCKED`
- Approve a US merchant on CARD-004

### `features/settlement.feature`
- Successful authorization debits the account exactly once
- A declined authorization leaves the balance untouched
- Two approved txns sum correctly
- Daily limit: three $15 txns on CARD-004 (daily 100.00) → third declines with `DAILY_LIMIT_EXCEEDED` if configured; adjust numbers so the scenario is unambiguous.

### `features/regression.feature` — include one `@wip` scenario
Add a scenario tagged `@wip` that intentionally **fails** (e.g., "Refund reverses a prior debit" — behavior not yet implemented). The README explains this is deliberate: run `make bdd` and you see red; implement refund and it goes green. This is the demo centerpiece.

---

## 10. Frontend (Svelte 5 + Vite + TS)

Two tabs. No router library — just a `let tab = $state('playground')`.

### Tab 1 — Playground
- Three dropdowns populated from `/api/seed`: Card, Merchant.
- Amount input (dollars, converts to cents).
- Currency select, defaults from the card's account.
- **Authorize** button → POST `/api/authorize`.
- Result card shows:
  - Big green **APPROVED** or red **DECLINED** badge.
  - If declined: the reason code as a monospace chip, plus a plain-English explanation (hard-coded map).
  - If approved: new balance + txn ID.
- Right-side panel: account balance (live, refetched after each call) and last 20 transactions for the selected card.
- **Reset data** button → POST `/api/reset`.

### Tab 2 — BDD Scenarios
- On mount: `GET /api/features` and `GET /api/bdd-report`.
- Left: list of feature files as accordions; Gherkin rendered with a tiny hand-rolled highlighter (color `Feature:` / `Scenario:` / `Given` / `When` / `Then` / `And` via CSS classes on spans — no external lib).
- Right: per-scenario status pill (green/red/gray) joined from `bdd-report.json` by scenario name.
- If `ENABLE_BDD_RUNNER=1` on the backend: a **Run BDD suite** button that POSTs `/api/bdd/run` and refreshes the report. Otherwise the button is hidden and a note says "run `make bdd` in your terminal."

Keep styling minimal — one CSS file, system font, no Tailwind, no component library.

---

## 11. Makefile

```makefile
.PHONY: dev backend frontend bdd test tidy clean

dev:            ## run backend + frontend concurrently
	@(trap 'kill 0' INT; $(MAKE) backend & $(MAKE) frontend & wait)

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm install --silent && npm run dev

bdd:            ## run the godog suite and write bdd-report.json
	cd backend && go test ./... -run TestFeatures -count=1

test:
	cd backend && go test ./... -count=1

tidy:
	cd backend && go mod tidy

clean:
	rm -f backend/bdd-report.json
```

`vite.config.ts` proxies `/api` → `http://localhost:8080`.

---

## 12. README.md (write this too)

Must contain:

1. **What is BDD?** — 3-sentence explanation.
2. **The loop** — a diagram/ASCII of `Gherkin → red → implement → green`.
3. **Quickstart** — `make dev`, open `http://localhost:5173`.
4. **Demo script** — a numbered walkthrough for a live demo:
   1. Open the **Scenarios** tab. Show the failing `@wip` refund scenario in red.
   2. Open `features/regression.feature`, write the missing step in the terminal (or point at the TODO in `rules.go`).
   3. `make bdd` → green.
   4. Switch to **Playground**, pick CARD-002, authorize $12.50, watch it decline with `CARD_FROZEN` — the *same* behavior the Gherkin asserts.
5. **Adding a rule** — a short recipe: write Gherkin → run `make bdd` → see the undefined step → add the step → implement the rule → green.
6. **Project layout** — the tree from §3.
7. **What this demo deliberately omits** — no real network calls, no persistence, no auth; the point is the loop, not production readiness.

---

## 13. Acceptance criteria

- [ ] `make bdd` runs every feature and every non-`@wip` scenario passes.
- [ ] The `@wip` scenario fails, and its failure message is legible (undefined step, not a panic).
- [ ] `make dev` brings up backend on `:8080` and frontend on `:5173` with no manual steps beyond `npm`/`go` being installed.
- [ ] `curl -X POST localhost:8080/api/authorize -d '{"cardId":"CARD-002","merchantId":"MERCH-COFFEE","amountCents":1250,"currency":"USD"}'` returns `{"approved":false,"reason":"CARD_FROZEN", ...}`.
- [ ] Playground tab: approving $12.50 on CARD-001 drops the balance from $500.00 → $487.50 live.
- [ ] Scenarios tab: the failing `@wip` scenario shows a red pill after `make bdd`.
- [ ] No Docker, no external DB, no cloud dependencies.
- [ ] `go vet ./...` and `npm run check` are clean.

---

## 14. Optional stretch (only if everything above is done)

- A `@wip2` scenario that demonstrates a **Scenario Outline** with an `Examples:` table of amounts → expected reason codes.
- A tag filter: `make bdd TAGS=@smoke` passes through to godog's `--tags`.
- A tiny "diff" view in the Scenarios tab that shows the git diff of the last change to `rules.go` alongside the newly-green scenario — reinforces the red/green story.

---

## 15. Implementation order (do it in this sequence)

1. Scaffold repo + `go.mod` + `package.json` + Makefile.
2. Domain types + KV store + seed + `Reset()`.
3. `features_test.go` + **one** feature file (`authorization.feature`) + step definitions for the four scenarios in it. Get `make bdd` green.
4. Add `velocity.feature`, `merchant_rules.feature`, `settlement.feature`. Green.
5. HTTP API layer.
6. Svelte frontend (Playground first, Scenarios second).
7. Add `regression.feature` with the failing `@wip` scenario.
8. README.

Commit after each numbered step so the demo has clean, narratable checkpoints.
