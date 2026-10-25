<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api, call } from '$lib/api';
	import FormErrors from '$lib/FormErrors.svelte';

	// The mail link carries the token in the fragment, which never reaches a server or its logs.
	const token = location.hash.slice(1);
	let error: unknown = $state(null);

	// Confirmed on mount, not by the GET of the link, so mail scanners that prefetch it do not use up the token.
	onMount(async () => {
		// drop the token from the address bar and history; SvelteKit's replaceState throws before its router has started
		history.replaceState(history.state, '', location.pathname);
		try {
			await call(api.POST('/api/verify', { body: { token } }));
			const { response } = await api.GET('/api/me');
			if (response.ok) await goto('/account?msg=verified');
			else await goto('/login?msg=verified');
		} catch (err) {
			error = err;
		}
	});
</script>

<svelte:head><title>Confirm address – Radiopath</title></svelte:head>

<div class="mx-auto mt-10 w-full max-w-[26rem]">
	<h1 class="mb-4 text-lg font-semibold">Confirm address</h1>
	{#if error}
		<FormErrors {error} />
		<a class="text-primary" href="/login">Back to login</a>
	{:else}
		<p class="text-muted-foreground">Confirming …</p>
	{/if}
</div>
