<script lang="ts">
	import RadioTower from '@lucide/svelte/icons/radio-tower';
	import type { Snippet } from 'svelte';

	// The frame of the login, registration and password pages: one centred card, links below it. Only the login, the
	// public entry, is meant for search engines.
	let { title, children, below, noindex = true }: { title: string; children: Snippet; below?: Snippet; noindex?: boolean } = $props();
</script>

<svelte:head>
	<title>{title} – Radiopath</title>
	{#if noindex}<meta name="robots" content="noindex" />{/if}
</svelte:head>

<div class="mx-auto mt-4 w-full max-w-sm sm:mt-12">
	<div class="mb-5 flex flex-col items-center gap-3 text-center">
		<span class="grid size-11 place-items-center rounded-full bg-primary/10 text-primary"><RadioTower class="size-5" /></span>
		<h1 class="text-lg font-semibold">{title}</h1>
	</div>
	<div class="grid gap-4 rounded-lg border bg-card p-5 shadow-sm">
		{@render children()}
	</div>
	{#if below}
		<div class="mt-4 flex flex-wrap justify-center gap-x-4 gap-y-1 text-center text-xs text-muted-foreground">{@render below()}</div>
	{/if}
</div>
