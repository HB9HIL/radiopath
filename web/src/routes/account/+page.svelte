<script lang="ts">
	import { page } from '$app/state';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { api, call } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Field from '$lib/Field.svelte';
	import FormErrors from '$lib/FormErrors.svelte';
	import { invalid, notifyError } from '$lib/notify';
	import PageHead from '$lib/PageHead.svelte';

	const queryClient = useQueryClient();
	// The page needs a login: call() sends a guest there on 401.
	const me = createQuery(() => ({ queryKey: ['me', 'account'], queryFn: () => call(api.GET('/api/me')) }));
	let email = $state('');
	let emailPassword = $state('');
	let current = $state('');
	let password = $state('');
	let password2 = $state('');

	// The confirmation link of a new address leads here.
	onMount(() => {
		if (page.url.searchParams.get('msg') === 'verified') toast.success('Address confirmed.');
	});

	// The field shows the confirmed address until the user types another one.
	$effect(() => {
		if (me.data && email === '') email = me.data.email;
	});

	/** Asks for the new address to be confirmed by mail; the password guards against a stolen session. */
	async function changeEmail(e: SubmitEvent) {
		e.preventDefault();
		try {
			const r = await api.PUT('/api/me/email', { body: { email, password: emailPassword } });
			await call(Promise.resolve(r));
			emailPassword = '';
			// 204: the address did not change
			if (r.response.status === 202) toast.success('Confirmation link sent to the new address. It takes effect once you open it.');
			else toast.info('That is the address on file already.');
		} catch (err) {
			notifyError(err);
		}
	}

	/** Changes the password; every other session ends, this one gets a fresh cookie. */
	async function changePassword(e: SubmitEvent) {
		e.preventDefault();
		if (password !== password2) return notifyError(invalid('Passwords do not match'));
		try {
			await call(api.PUT('/api/me/password', { body: { current, password } }));
			current = password = password2 = '';
			toast.success('Password changed. Other devices have been logged out.');
			await queryClient.invalidateQueries({ queryKey: ['me'] });
		} catch (err) {
			notifyError(err);
		}
	}
</script>

<div class="mx-auto mt-10 w-full max-w-[26rem]">
	<PageHead title="Account" />
	<FormErrors error={me.error} />
	{#if me.data}
		<form class="grid gap-3 rounded-lg border bg-card p-4" onsubmit={changeEmail}>
			<Field label="User name"><Input value={me.data.name} disabled /></Field>
			<Field label="Email"><Input name="email" type="email" autocomplete="email" required bind:value={email} /></Field>
			<Field label="Password"
				><Input name="email_password" type="password" autocomplete="current-password" required bind:value={emailPassword} /></Field
			>
			<div><Button type="submit">Change address</Button></div>
		</form>
		<p class="mt-2 text-xs text-muted-foreground">
			{#if me.data.email}
				A new address takes effect once you open the confirmation link we send there; the old one gets a notice. Password reset mails go to the
				confirmed address.
			{:else}
				No address on file: without one there is no password reset.
			{/if}
		</p>
		<form class="mt-4 grid gap-3 rounded-lg border bg-card p-4" onsubmit={changePassword}>
			<Field label="Current password"><Input name="current" type="password" autocomplete="current-password" required bind:value={current} /></Field>
			<Field label="New password">
				<Input name="password" type="password" autocomplete="new-password" minlength={10} required bind:value={password} />
			</Field>
			<Field label="Repeat"
				><Input name="password2" type="password" autocomplete="new-password" minlength={10} required bind:value={password2} /></Field
			>
			<div><Button type="submit">Change password</Button></div>
		</form>
		<p class="mt-2 text-xs text-muted-foreground">At least 10 characters. Changing it logs out every other device.</p>
	{/if}
</div>
