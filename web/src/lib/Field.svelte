<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Label } from '$lib/components/ui/label';

	let { label, children }: { label: string; children: Snippet } = $props();
</script>

<Label class="grid gap-1.5 font-normal"><span class="text-muted-foreground">{label}</span>{@render children()}</Label>
