<script lang="ts" module>
	export type Tile = { label: string; value: string; sub?: string; tone?: 'ok' | 'bad' };
</script>

<script lang="ts">
	// The key figures of a page, readable at a glance on a phone too.
	let { tiles }: { tiles: Tile[] } = $props();
</script>

<div class="mb-5 grid grid-cols-2 gap-3 md:grid-cols-4">
	{#each tiles as t (t.label)}
		<div class="rounded-lg border bg-card p-3">
			<p class="text-xs text-muted-foreground">{t.label}</p>
			<p class={['mt-1 font-mono text-lg font-semibold sm:text-xl', t.tone === 'ok' && 'text-ok', t.tone === 'bad' && 'text-destructive']}>
				{t.value}
			</p>
			{#if t.sub}<p class="mt-0.5 text-xs text-muted-foreground">{t.sub}</p>{/if}
		</div>
	{/each}
</div>
