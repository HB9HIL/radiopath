<script lang="ts">
	import type { Snippet } from 'svelte';

	let { title, children }: { title: string; children?: Snippet } = $props();
</script>

<svelte:head><title>{title} – Radiopath</title></svelte:head>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3 border-b pb-3">
	<h1 class="text-lg font-semibold">{title}</h1>
	{#if children}<div class="flex flex-wrap items-center gap-2">{@render children()}</div>{/if}
</div>
