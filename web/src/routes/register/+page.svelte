<script lang="ts">
	import { goto } from '$app/navigation';
	import { createQuery } from '@tanstack/svelte-query';
	import { api, call, configQuery } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Field from '$lib/Field.svelte';
	import FormErrors from '$lib/FormErrors.svelte';

	const config = createQuery(configQuery);
	let name = $state('');
	let email = $state('');
	let password = $state('');
	let password2 = $state('');
	let error: unknown = $state(null);
	let busy = $state(false);

	/** Creates the account; the confirmation mail is on its way when the login page shows. */
	async function submit(e: SubmitEvent) {
		e.preventDefault();
		error = password === password2 ? null : new Error('Passwords do not match');
		if (error) return;
		busy = true;
		try {
			await call(api.POST('/api/register', { body: { name, email, password } }));
			await goto('/login?msg=registered');
		} catch (err) {
			error = err;
			busy = false;
		}
	}
</script>

<svelte:head><title>Register – Radiopath</title></svelte:head>

<div class="mx-auto mt-10 w-full max-w-[26rem]">
	<h1 class="mb-4 text-lg font-semibold">Register</h1>
	{#if config.data && !config.data.registration}
		<p>Registration is closed. <a class="text-primary" href="/login">Back to login</a></p>
	{:else}
		<FormErrors {error} />
		<form class="grid gap-4 rounded-lg border bg-card p-5" onsubmit={submit}>
			<Field label="User name">
				<!-- svelte-ignore a11y_autofocus -->
				<Input name="name" autocomplete="username" autofocus required bind:value={name} />
			</Field>
			<Field label="Email"><Input name="email" type="email" autocomplete="email" required bind:value={email} /></Field>
			<Field label="Password"
				><Input name="password" type="password" autocomplete="new-password" minlength={10} required bind:value={password} /></Field
			>
			<Field label="Repeat password">
				<Input name="password2" type="password" autocomplete="new-password" minlength={10} required bind:value={password2} />
			</Field>
			<div class="flex flex-wrap items-center gap-4">
				<Button type="submit" disabled={busy}>Create account</Button>
				<a class="text-primary" href="/login">Back to login</a>
			</div>
		</form>
		<p class="mt-3 text-xs text-muted-foreground">
			3 to 32 characters for the name (letters, digits, . _ -), at least 10 for the password. We mail a confirmation link to the address; the account
			opens once you follow it.
		</p>
	{/if}
</div>
