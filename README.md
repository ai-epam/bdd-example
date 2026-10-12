# AuthGate — a BDD demo

AuthGate is a small card-transaction authorization engine (Go) with a live web UI
(Svelte), built to show **Behavior-Driven Development** end to end. No Docker, no DB:
state lives in an in-memory store with deterministic seed data.

## What is BDD?

BDD describes software behavior in plain-language examples that business people,
testers and developers share. Those examples are written in Gherkin (`Given / When / Then`)
and are executable: a tool such as [godog](https://github.com/cucumber/godog) runs them
against the code. The spec *is* the test, so it cannot drift from reality.

## The loop

```
  write Gherkin ──► run (RED) ──► implement in Go ──► run (GREEN)
        ▲                                                  │
        └──────────── next rule / refactor ◄───────────────┘
```

## Quickstart

Requires Go 1.22+ and Node.

```bash
make dev          # backend :8080 + frontend :5173
make bdd          # full godog suite (the @wip scenario is red on purpose)
make test         # everything except @wip (CI-friendly)
make bdd TAGS=@smoke   # TAGS is passed through to godog (e.g. '~@wip')
```

Open <http://localhost:5173> and sign in with a demo user: `alice` / `wonderland1` or
`bob` / `builder123`. To get the **Run BDD suite** button in the UI, start the
backend with `ENABLE_BDD_RUNNER=1 make backend`.

`make bdd` writes `bdd-report.json` in the repo root; the Scenarios tab reads it.

## Demo script

1. Run `make bdd`, then open the **BDD Scenarios** tab. The `@wip` scenario
   *Refund reverses a prior debit* shows a **red** pill: its step
   `I refund the last transaction` is undefined.
2. Open `backend/features/regression.feature` and the TODO in
   `backend/internal/authz/rules.go`. Implement the refund: add the step in
   `backend/steps/steps.go` and `authz.Refund` in the engine. (Short on time?
   `git apply demo/refund.patch` applies a ready-made implementation.)
3. `make bdd` → **green**. Refresh the Scenarios tab.
4. Sign in, switch to **Playground**, pick `CARD-002`, authorize $12.50 at Blue Bottle Coffee:
   it declines with `CARD_FROZEN` — the same behavior the Gherkin asserts.
   Then pick `CARD-001` and watch the balance go $500.00 → $487.50.

## Adding a rule

1. Write the scenario in a `.feature` file under `backend/features/`.
2. `make bdd` — godog reports the undefined step and prints a snippet.
3. Add the step to `backend/steps/steps.go` (register it in `InitializeScenario`).
4. Run again: red, because the rule does not exist yet.
5. Add the rule function to the ordered list in `backend/internal/authz/rules.go`.
6. `make bdd` — green.

## Login

Every `/api` endpoint except `/api/login` and `/api/config` needs `Authorization: Bearer <token>`.
Behavior is specified in `backend/features/login.feature`: wrong credentials and unknown users
give the same `INVALID_CREDENTIALS`; three failures lock the account for 5 minutes
(`ACCOUNT_LOCKED`); sessions last 30 minutes and end on logout. Passwords are PBKDF2-hashed;
sessions live in memory and are lost on restart.

## Rules (first failure wins)

`CARD_NOT_FOUND`, `CARD_FROZEN`, `CARD_EXPIRED`, `ACCOUNT_NOT_FOUND`, `ACCOUNT_FROZEN`,
`MERCHANT_NOT_FOUND`, `MCC_BLOCKED`, `COUNTRY_BLOCKED`, `CURRENCY_MISMATCH`,
`AMOUNT_EXCEEDS_PER_TXN_LIMIT`, `VELOCITY_LIMIT_EXCEEDED`, `DAILY_LIMIT_EXCEEDED`,
`INSUFFICIENT_FUNDS`. Notes: velocity declines when the card already has the maximum
number of *approved* transactions inside the sliding window; the daily limit uses the
UTC day. Scenarios run on a frozen clock (2025-01-15 12:00 UTC).

## Project layout

```
Makefile
backend/
  cmd/server/main.go            HTTP server bootstrap
  internal/domain/types.go      Account, Card, Merchant, Transaction
  internal/store/               thread-safe in-memory KV, seed data, Reset()
  internal/auth/                login, sessions, lockout (in-memory, demo users)
  internal/authz/               engine.go (Authorize) + rules.go (rules, reason codes)
  internal/api/handlers.go      chi handlers (+ integration test)
  features/*.feature            the Gherkin specs
  steps/                        godog steps (steps.go, auth_steps.go) + JSON report collector
  features_test.go              godog entry point (writes ../bdd-report.json)
frontend/                       Svelte 5 + Vite + TS (Playground, BDD Scenarios)
demo/refund.patch               ready-made refund implementation for the demo
```

## What this demo deliberately omits

No real network calls, no persistence, no money movement, and only a demo-grade login (two seeded users, no registration, roles or password reset). The point
is the loop, not production readiness.
