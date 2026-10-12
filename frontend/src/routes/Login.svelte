<script lang="ts">
  import { api, ApiError, LOGIN_ERRORS, session } from '../lib/api'

  let { onlogin }: { onlogin: (user: string) => void } = $props()

  let username = $state('')
  let password = $state('')
  let error = $state('')
  let busy = $state(false)

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    error = ''
    busy = true
    try {
      const r = await api.login(username, password)
      session.save(r.token, r.username)
      password = ''
      onlogin(r.username)
    } catch (err) {
      error =
        err instanceof ApiError && err.reason
          ? (LOGIN_ERRORS[err.reason] ?? err.reason)
          : 'Could not reach the server. Is the backend running on :8080?'
    } finally {
      busy = false
    }
  }
</script>

<form class="panel login" onsubmit={submit}>
  <h2>Sign in to AuthGate</h2>
  <label>Username
    <input bind:value={username} autocomplete="username" />
  </label>
  <label>Password
    <input type="password" bind:value={password} autocomplete="current-password" />
  </label>
  <button class="primary" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
  {#if error}<p class="err" role="alert">{error}</p>{/if}
  <p class="muted hint">Demo users: <code>alice</code> / <code>wonderland1</code>, <code>bob</code> / <code>builder123</code></p>
</form>
