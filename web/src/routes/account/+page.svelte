<script lang="ts">
	import { page } from '$app/state';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError, api, call } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Field from '$lib/Field.svelte';
	import FormErrors from '$lib/FormErrors.svelte';
	import Notice from '$lib/Notice.svelte';
	import PageHead from '$lib/PageHead.svelte';

	const queryClient = useQueryClient();
	// The page needs a login: call() sends a guest there on 401.
	const me = createQuery(() => ({ queryKey: ['me', 'account'], queryFn: () => call(api.GET('/api/me')) }));
	let email = $state('');
	let current = $state('');
	let password = $state('');
	let password2 = $state('');
	let error: unknown = $state(null);
	let notice = $state(page.url.searchParams.get('msg') === 'verified' ? 'Address confirmed.' : '');

	// The field shows the confirmed address until the user types another one.
	$effect(() => {
		if (me.data && email === '') email = me.data.email;
	});

	/** Asks for the new address to be confirmed by mail. */
	async function changeEmail(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		notice = '';
		try {
			const r = await api.PUT('/api/me/email', { body: { email } });
			await call(Promise.resolve(r));
			// 204: the address did not change
			if (r.response.status === 202) notice = 'Confirmation link sent to the new address. It takes effect once you open it.';
		} catch (err) {
			error = err;
		}
	}

	/** Changes the password; every other session ends, this one gets a fresh cookie. */
	async function changePassword(e: SubmitEvent) {
		e.preventDefault();
		error = password === password2 ? null : new ApiError(422, 'Invalid input', ['Passwords do not match']);
		notice = '';
		if (error) return;
		try {
			await call(api.PUT('/api/me/password', { body: { current, password } }));
			current = password = password2 = '';
			notice = 'Password changed. Other devices have been logged out.';
			await queryClient.invalidateQueries({ queryKey: ['me'] });
		} catch (err) {
			error = err;
		}
	}
</script>

<div class="mx-auto mt-10 w-full max-w-[26rem]">
	<PageHead title="Account" />
	{#if notice}<Notice>{notice}</Notice>{/if}
	<FormErrors error={error ?? me.error} />
	{#if me.data}
		<form class="grid gap-3 rounded-lg border bg-card p-4" onsubmit={changeEmail}>
			<Field label="User name"><Input value={me.data.name} disabled /></Field>
			<Field label="Email"><Input name="email" type="email" autocomplete="email" required bind:value={email} /></Field>
			<div><Button type="submit">Change address</Button></div>
		</form>
		<p class="mt-2 text-xs text-muted-foreground">
			{#if me.data.email}
				A new address takes effect once you open the confirmation link we send there. Password reset mails go to the confirmed address.
			{:else}
				No address on file: without one there is no password reset.
			{/if}
		</p>
		<form class="mt-4 grid gap-3 rounded-lg border bg-card p-4" onsubmit={changePassword}>
			<Field label="Current password"><Input name="current" type="password" autocomplete="current-password" required bind:value={current} /></Field>
			<Field label="New password">
				<Input name="password" type="password" autocomplete="new-password" minlength={10} required bind:value={password} />
			</Field>
			<Field label="Repeat"><Input name="password2" type="password" autocomplete="new-password" minlength={10} required bind:value={password2} /></Field>
			<div><Button type="submit">Change password</Button></div>
		</form>
		<p class="mt-2 text-xs text-muted-foreground">At least 10 characters. Changing it logs out every other device.</p>
	{/if}
</div>
