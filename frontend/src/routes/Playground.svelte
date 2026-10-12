<script lang="ts">
  import { api, money, toCents, type Account, type AuthResult, type Seed, type Txn } from '../lib/api'

  const EXPLAIN: Record<string, string> = {
    CARD_NOT_FOUND: 'The card does not exist.',
    CARD_FROZEN: 'The card is frozen and can never authorize.',
    CARD_EXPIRED: 'The card has expired.',
    ACCOUNT_NOT_FOUND: 'The account behind this card does not exist.',
    ACCOUNT_FROZEN: 'The account is frozen.',
    MERCHANT_NOT_FOUND: 'The merchant is unknown.',
    MCC_BLOCKED: 'This merchant category is blocked on the card (e.g. gambling 7995).',
    COUNTRY_BLOCKED: 'The merchant is in a country the card is not allowed to spend in.',
    CURRENCY_MISMATCH: 'The transaction currency differs from the account currency.',
    AMOUNT_EXCEEDS_PER_TXN_LIMIT: "The amount is over the card's per-transaction limit.",
    VELOCITY_LIMIT_EXCEEDED: 'Too many approved transactions on this card in the velocity window.',
    DAILY_LIMIT_EXCEEDED: "Today's approved spend plus this amount exceeds the daily limit.",
    INSUFFICIENT_FUNDS: 'The account balance is too low.',
  }

  let seed = $state<Seed | null>(null)
  let cardId = $state('CARD-001')
  let merchantId = $state('MERCH-COFFEE')
  let amount = $state('12.50')
  let currency = $state('USD')
  let result = $state<AuthResult | null>(null)
  let account = $state<Account | null>(null)
  let txns = $state<Txn[]>([])
  let error = $state('')

  const card = $derived(seed?.cards.find((c) => c.id === cardId))
  const merchantName = (id: string) => seed?.merchants.find((m) => m.id === id)?.name ?? id

  async function refresh() {
    if (!card) return
    const [a, t] = await Promise.all([api.account(card.accountId), api.cardTxns(card.id)])
    account = a.account
    txns = t
    currency = a.account.currency
  }

  $effect(() => {
    api.seed().then((s) => (seed = s)).catch((e) => (error = String(e)))
  })

  // Reload balance + history whenever the selected card changes.
  $effect(() => {
    cardId
    if (seed) refresh().catch((e) => (error = String(e)))
  })

  async function authorize() {
    error = ''
    const amountCents = toCents(amount)
    if (!Number.isFinite(amountCents) || amountCents <= 0) {
      error = 'Enter a positive amount like 12.50'
      return
    }
    try {
      result = await api.authorize({ cardId, merchantId, amountCents, currency })
      await refresh()
    } catch (e) {
      error = String(e)
    }
  }

  async function reset() {
    error = ''
    try {
      await api.reset()
      result = null
      await refresh()
    } catch (e) {
      error = String(e)
    }
  }
</script>

{#if !seed}
  <p>{error || 'Loading… (is the backend running on :8080?)'}</p>
{:else}
  <div class="cols">
    <section class="panel">
      <h2>Authorize a transaction</h2>
      <label>Card
        <select bind:value={cardId}>
          {#each seed.cards as c}<option value={c.id}>{c.id} ({c.status})</option>{/each}
        </select>
      </label>
      <label>Merchant
        <select bind:value={merchantId}>
          {#each seed.merchants as m}<option value={m.id}>{m.name} · MCC {m.mcc} · {m.country}</option>{/each}
        </select>
      </label>
      <label>Amount
        <input bind:value={amount} inputmode="decimal" />
      </label>
      <label>Currency
        <select bind:value={currency}>
          <option>USD</option><option>EUR</option>
        </select>
      </label>
      <div class="row">
        <button class="primary" onclick={authorize}>Authorize</button>
        <button onclick={reset}>Reset data</button>
      </div>
      {#if error}<p class="err">{error}</p>{/if}

      {#if result}
        <div class="result">
          {#if result.approved}
            <span class="badge ok">APPROVED</span>
            <p>New balance <strong>{money(result.newBalance)}</strong></p>
            <p>Transaction <code class="chip">{result.transactionId}</code></p>
          {:else}
            <span class="badge bad">DECLINED</span>
            <p><code class="chip">{result.reason}</code></p>
            <p>{EXPLAIN[result.reason] ?? ''}</p>
          {/if}
        </div>
      {/if}
    </section>

    <section class="panel">
      <h2>Account</h2>
      {#if account}
        <p>{account.holder} · {account.id} · <em>{account.status}</em></p>
        <p class="balance" data-testid="balance">{money(account.balance, account.currency)}</p>
      {/if}
      <h2>Last transactions on {cardId}</h2>
      {#if txns.length === 0}
        <p class="muted">None yet.</p>
      {:else}
        <table>
          <thead><tr><th>ID</th><th>Merchant</th><th>Amount</th><th>Status</th></tr></thead>
          <tbody>
            {#each txns as t}
              <tr>
                <td>{t.id}</td><td>{merchantName(t.merchantId)}</td><td>{money(t.amount, t.currency)}</td>
                <td class={t.status}>{t.status}{t.reason ? ` · ${t.reason}` : ''}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </section>
  </div>
{/if}
