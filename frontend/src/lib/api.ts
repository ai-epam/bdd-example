export interface Account { id: string; holder: string; balance: number; currency: string; status: string }
export interface Card {
  id: string; accountId: string; status: string; perTxnLimit: number; dailyLimit: number
  velocityMaxTxns: number; velocityWindowS: number; blockedMccs: string[]; allowedCountries: string[]
}
export interface Merchant { id: string; name: string; mcc: string; country: string }
export interface Seed { accounts: Account[]; cards: Card[]; merchants: Merchant[] }
export interface Txn {
  id: string; cardId: string; merchantId: string; amount: number; currency: string
  createdAt: string; status: 'approved' | 'declined'; reason: string
}
export interface AuthResult { approved: boolean; reason: string; transactionId: string; newBalance: number }
export interface ScenarioReport { name: string; status: 'passed' | 'failed' | 'skipped'; durationMs: number; error?: string }
export interface FeatureReport { name: string; path: string; scenarios: ScenarioReport[] }
export interface BddReport { generatedAt: string; passed: number; failed: number; skipped: number; features: FeatureReport[] }

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch('/api' + path, init)
  const body = await res.json()
  if (!res.ok) throw new Error(body?.error ?? res.statusText)
  return body as T
}

export const api = {
  seed: () => call<Seed>('/seed'),
  account: (id: string) => call<{ account: Account; transactions: Txn[] }>(`/accounts/${id}`),
  cardTxns: (id: string) => call<Txn[]>(`/cards/${id}/transactions`),
  authorize: (b: { cardId: string; merchantId: string; amountCents: number; currency: string }) =>
    call<AuthResult>('/authorize', { method: 'POST', body: JSON.stringify(b) }),
  reset: () => call<{ ok: boolean }>('/reset', { method: 'POST' }),
  features: () => call<string[]>('/features'),
  report: () => call<BddReport | null>('/bdd-report'),
  config: () => call<{ bddRunner: boolean }>('/config'),
  runBdd: () => call<BddReport>('/bdd/run', { method: 'POST' }),
}

export const money = (cents: number, cur = 'USD') =>
  new Intl.NumberFormat('en-US', { style: 'currency', currency: cur }).format(cents / 100)

/** "12.50" -> 1250 without floating point. Returns NaN when invalid. */
export function toCents(input: string): number {
  const m = /^\s*(\d+)(?:\.(\d{1,2}))?\s*$/.exec(input)
  if (!m) return NaN
  return parseInt(m[1], 10) * 100 + parseInt((m[2] ?? '').padEnd(2, '0') || '0', 10)
}
