<script lang="ts">
  import Playground from './routes/Playground.svelte'
  import Scenarios from './routes/Scenarios.svelte'

  let tab = $state<'playground' | 'scenarios'>('playground')
</script>

<header>
  <h1>AuthGate</h1>
  <nav>
    <button class:active={tab === 'playground'} onclick={() => (tab = 'playground')}>Playground</button>
    <button class:active={tab === 'scenarios'} onclick={() => (tab = 'scenarios')}>BDD Scenarios</button>
  </nav>
</header>

<main>
  {#if tab === 'playground'}<Playground />{:else}<Scenarios />{/if}
</main>
