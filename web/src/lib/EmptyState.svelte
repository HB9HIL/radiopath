<script lang="ts">
	import type { Component, Snippet } from 'svelte';

	let { icon: Icon, children }: { icon: Component; children: Snippet } = $props();
</script>

<div class="flex flex-col items-center gap-3 rounded-lg border border-dashed bg-card px-6 py-10 text-center text-muted-foreground">
	<Icon class="size-8 opacity-60" />
	<p>{@render children()}</p>
</div>
