<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { api, call } from '$lib/api';
	import FormErrors from '$lib/FormErrors.svelte';

	let error: unknown = $state(null);

	// Confirmed on mount, not by the GET of the link, so mail scanners that prefetch it do not use up the token.
	onMount(async () => {
		try {
			await call(api.POST('/api/verify', { body: { token: page.params.token ?? '' } }));
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
