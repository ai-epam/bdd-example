<script lang="ts">
  import { api, type BddReport } from '../lib/api'

  let features = $state<string[]>([])
  let report = $state<BddReport | null>(null)
  let runner = $state(false)
  let running = $state(false)
  let error = $state('')

  // Scenario name -> report entry. Scenario names are unique across features.
  const results = $derived(
    new Map(report?.features.flatMap((f) => f.scenarios).map((s) => [s.name, s]) ?? []),
  )

  async function load() {
    try {
      ;[features, report, runner] = await Promise.all([
        api.features(),
        api.report(),
        api.config().then((c) => c.bddRunner),
      ])
    } catch (e) {
      error = String(e)
    }
  }
  $effect(() => {
    load()
  })

  async function run() {
    running = true
    error = ''
    try {
      report = await api.runBdd()
    } catch (e) {
      error = String(e)
    } finally {
      running = false
    }
  }

  const title = (src: string) => /^\s*Feature:\s*(.*)$/m.exec(src)?.[1] ?? 'Feature'
  const ESC = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;')

  interface Line { html: string; scenario?: string }

  /** Tiny hand-rolled Gherkin highlighter; returns one entry per source line. */
  function highlight(src: string): Line[] {
    return src.split('\n').map((raw) => {
      const m = /^(\s*)(Feature|Background|Scenario Outline|Scenario|Examples):(.*)$/.exec(raw)
      if (m) {
        const scenario = m[2].startsWith('Scenario') ? m[3].trim() : undefined
        return { html: `${m[1]}<span class="kw-block">${m[2]}:</span>${ESC(m[3])}`, scenario }
      }
      const s = /^(\s*)(Given|When|Then|And|But)(\s.*)$/.exec(raw)
      if (s) return { html: `${s[1]}<span class="kw-${s[2].toLowerCase()}">${s[2]}</span>${ESC(s[3])}` }
      if (/^\s*@/.test(raw)) return { html: `<span class="kw-tag">${ESC(raw)}</span>` }
      if (/^\s*#/.test(raw)) return { html: `<span class="kw-comment">${ESC(raw)}</span>` }
      return { html: ESC(raw) }
    })
  }
</script>

<div class="bar">
  {#if report}
    <span>Last run {new Date(report.generatedAt).toLocaleString()}:</span>
    <span class="pill passed">{report.passed} passed</span>
    <span class="pill failed">{report.failed} failed</span>
    {#if report.skipped}<span class="pill skipped">{report.skipped} skipped</span>{/if}
  {:else}
    <span>No report yet.</span>
  {/if}
  <span class="spacer"></span>
  {#if runner}
    <button class="primary" onclick={run} disabled={running}>{running ? 'Running…' : 'Run BDD suite'}</button>
  {:else}
    <span class="muted">Run <code>make bdd</code> in your terminal, then reload.</span>
  {/if}
</div>
{#if error}<p class="err">{error}</p>{/if}

{#each features as src}
  <details class="feature" open>
    <summary>{title(src)}</summary>
    <pre class="gherkin">{#each highlight(src) as line}<span class="line"
          >{@html line.html}{#if line.scenario}{@const r = results.get(line.scenario)}<span
              class="pill {r?.status ?? 'unknown'}"
              title={r?.error ?? ''}>{r?.status ?? 'not run'}</span
            >{/if}</span
        >
{/each}</pre>
  </details>
{/each}
