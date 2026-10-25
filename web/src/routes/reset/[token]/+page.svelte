<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { ApiError, api, call } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Field from '$lib/Field.svelte';
	import FormErrors from '$lib/FormErrors.svelte';
	import { invalid, notifyError } from '$lib/notify';

	const token = page.params.token ?? '';
	let password = $state('');
	let password2 = $state('');
	// An expired or used link: the page has nothing left to offer but a new one.
	let stale: unknown = $state(null);
	let busy = $state(false);

	/** Shows an expired or used link on the page, any other error as a toast. */
	function check(err: unknown) {
		if (err instanceof ApiError && err.status === 410) stale = err;
		else notifyError(err);
	}

	onMount(() => {
		call(api.GET('/api/password/reset/{token}', { params: { path: { token } } })).catch(check);
	});

	/** Sets the new password, which logs out every session of the account. */
	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (password !== password2) return notifyError(invalid('Passwords do not match'));
		busy = true;
		try {
			await call(api.POST('/api/password/reset', { body: { token, password } }));
			await goto('/login?msg=reset');
		} catch (err) {
			check(err);
			busy = false;
		}
	}
</script>

<svelte:head><title>New password – Radiopath</title></svelte:head>

<div class="mx-auto mt-10 w-full max-w-[26rem]">
	<h1 class="mb-4 text-lg font-semibold">Choose a new password</h1>
	{#if stale}
		<FormErrors error={stale} />
		<p class="text-xs"><a class="text-primary" href="/forgot">Request a new link</a></p>
	{:else}
		<form class="grid gap-4 rounded-lg border bg-card p-5" onsubmit={submit}>
			<Field label="New password">
				<!-- svelte-ignore a11y_autofocus -->
				<Input name="password" type="password" autocomplete="new-password" minlength={10} autofocus required bind:value={password} />
			</Field>
			<Field label="Repeat"
				><Input name="password2" type="password" autocomplete="new-password" minlength={10} required bind:value={password2} /></Field
			>
			<div><Button type="submit" disabled={busy}>Set password</Button></div>
		</form>
		<p class="mt-3 text-xs text-muted-foreground">At least 10 characters. Every open session of the account is logged out.</p>
	{/if}
</div>
