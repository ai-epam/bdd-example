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

const TOKEN_KEY = 'authgate.token'
const USER_KEY = 'authgate.user'

function storage(): Storage | null {
  try { return sessionStorage } catch { return null }
}

export const session = {
  token: () => storage()?.getItem(TOKEN_KEY) ?? '',
  user: () => storage()?.getItem(USER_KEY) ?? '',
  save(token: string, user: string) {
    storage()?.setItem(TOKEN_KEY, token)
    storage()?.setItem(USER_KEY, user)
  },
  clear() {
    storage()?.removeItem(TOKEN_KEY)
    storage()?.removeItem(USER_KEY)
  },
}

/** Called when the server rejects our token, so the app can show the login screen. */
let onUnauthorized: () => void = () => {}
export const setUnauthorizedHandler = (fn: () => void) => { onUnauthorized = fn }

export class ApiError extends Error {
  constructor(message: string, public status: number, public reason?: string) { super(message) }
}

async function call<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  const token = session.token()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const res = await fetch('/api' + path, { ...init, headers })
  const body = await res.json().catch(() => null)
  if (res.status === 401 && path !== '/login') onUnauthorized()
  if (!res.ok) throw new ApiError(body?.error ?? res.statusText, res.status, body?.reason)
  return body as T
}

export interface LoginResult { token: string; username: string; expiresAt: string }

export const LOGIN_ERRORS: Record<string, string> = {
  INVALID_CREDENTIALS: 'Wrong username or password.',
  MISSING_CREDENTIALS: 'Enter both a username and a password.',
  ACCOUNT_LOCKED: 'Too many failed attempts. This account is locked for 5 minutes.',
}

export const api = {
  login: (username: string, password: string) =>
    call<LoginResult>('/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  logout: () => call<{ ok: boolean }>('/logout', { method: 'POST' }),
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
