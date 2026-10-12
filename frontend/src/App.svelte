<script lang="ts">
  import { api, session, setUnauthorizedHandler } from './lib/api'
  import Login from './routes/Login.svelte'
  import Playground from './routes/Playground.svelte'
  import Scenarios from './routes/Scenarios.svelte'

  let tab = $state<'playground' | 'scenarios'>('playground')
  let user = $state(session.token() ? session.user() : '')

  function signedOut() {
    session.clear()
    user = ''
  }
  setUnauthorizedHandler(signedOut)

  async function logout() {
    try {
      await api.logout()
    } catch {
      // the token is dropped locally either way
    }
    signedOut()
  }
</script>

<header>
  <h1>AuthGate</h1>
  {#if user}
    <nav>
      <button class:active={tab === 'playground'} onclick={() => (tab = 'playground')}>Playground</button>
      <button class:active={tab === 'scenarios'} onclick={() => (tab = 'scenarios')}>BDD Scenarios</button>
    </nav>
    <span class="spacer"></span>
    <span class="who">{user}</span>
    <button class="logout" onclick={logout}>Log out</button>
  {/if}
</header>

<main>
  {#if !user}
    <Login onlogin={(u) => (user = u)} />
  {:else if tab === 'playground'}
    <Playground />
  {:else}
    <Scenarios />
  {/if}
</main>
