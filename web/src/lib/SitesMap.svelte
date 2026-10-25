<script lang="ts">
	import type { Site } from '$lib/api';
	import { sitesMap } from '$lib/map';

	let { sites }: { sites: Site[] } = $props();
	let el: HTMLDivElement;

	$effect(() => {
		const m = sitesMap(el, sites);
		return () => m.remove();
	});
</script>

<div class="map" bind:this={el}></div>
