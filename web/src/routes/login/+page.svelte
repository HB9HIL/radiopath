<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { ApiError, api, call, configQuery, meQuery, safeNext } from '$lib/api';
	import * as Alert from '$lib/components/ui/alert';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Field from '$lib/Field.svelte';
	import Notice from '$lib/Notice.svelte';
	import { notifyError } from '$lib/notify';

	const notices: Record<string, string> = {
		registered: 'Almost done: check your mail for the next step.',
		verified: 'Address confirmed.',
		reset: 'Password changed. Log in with the new one.'
	};
	const config = createQuery(configQuery);
	const me = createQuery(meQuery);
	const next = safeNext(page.url.searchParams.get('next'));
	let name = $state('');
	let password = $state('');
	// Why the account cannot log in yet, shown next to the resend button.
	let unverified = $state('');
	let busy = $state(false);

	// The page a mail link or the registration led here from says what happened.
	onMount(() => {
		const msg = notices[page.url.searchParams.get('msg') ?? ''];
		if (msg) toast.success(msg);
	});

	// Logged in already: nothing to do here.
	$effect(() => {
		if (me.data) location.replace(next);
	});

	/** Logs in and loads the page the user came from; an unconfirmed account gets the resend button instead. */
	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			await call(api.POST('/api/session', { body: { name, password } }));
			location.assign(next);
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) unverified = err.message;
			else notifyError(err);
			busy = false;
		}
	}

	/** Asks for a new confirmation mail for the name just entered. */
	async function resend() {
		try {
			await call(api.POST('/api/verify/resend', { body: { name } }));
			unverified = '';
			toast.success('If that account is waiting for confirmation, a new link is on its way.');
		} catch (err) {
			notifyError(err);
		}
	}
</script>

<svelte:head>
	<title>Radiopath – radio link and coverage planning</title>
	<meta
		name="description"
		content="Radiopath plans radio links and area coverage: terrain profiles from Copernicus GLO-30, Longley-Rice/ITM path loss, Fresnel zone, link budget with real antenna patterns, and PDF reports."
	/>
	<link rel="canonical" href="/login" />
</svelte:head>

<div class="mx-auto mt-10 grid w-full max-w-4xl gap-8 md:grid-cols-[26rem_minmax(0,1fr)]">
	<div>
		<h1 class="mb-4 text-lg font-semibold">Log in</h1>
		{#if config.data?.maintenance}
			<Alert.Root class="border-warn/50 text-warn">
				<b>Maintenance</b><br />Radiopath is closed for maintenance. Logins are disabled, please try again later.
			</Alert.Root>
		{:else}
			{#if unverified}
				<Notice>{unverified}</Notice>
				<div class="flex flex-wrap items-center gap-4">
					<Button onclick={resend}>Resend confirmation mail</Button>
					<button class="text-primary" onclick={() => (unverified = '')}>Back to login</button>
				</div>
			{:else}
				<form class="grid gap-4 rounded-lg border bg-card p-5" onsubmit={submit}>
					<Field label="User name">
						<!-- svelte-ignore a11y_autofocus -->
						<Input name="name" autocomplete="username" autofocus required bind:value={name} />
					</Field>
					<Field label="Password"><Input name="password" type="password" autocomplete="current-password" required bind:value={password} /></Field>
					<div class="flex flex-wrap items-center gap-4">
						<Button type="submit" disabled={busy}>Log in</Button>
						{#if config.data?.registration}<a class="text-primary" href="/register">Create an account</a>{/if}
						{#if config.data?.password_reset}<a class="text-primary" href="/forgot">Forgot password?</a>{/if}
					</div>
				</form>
			{/if}
			{#if config.data && !config.data.registration}
				<p class="mt-3 text-xs text-muted-foreground">Registration is closed while the service is under development.</p>
			{/if}
		{/if}
	</div>
	<div class="text-[13px] leading-relaxed text-muted-foreground md:mt-11">
		<h2 class="text-xs font-medium text-foreground">What is Radiopath?</h2>
		<p class="mt-2">
			Radiopath is an independent clone of <i>Radio Mobile Online</i> (Roger Coudé, VE2DBE): radio link and coverage planning for amateur radio, rf links
			and similar point-to-point or point-to-area systems.
		</p>
		<ul class="mt-3 list-disc space-y-1 pl-4">
			<li>
				<b class="text-foreground">Point-to-point links</b>: terrain profile from Copernicus GLO-30 elevation data (30 m, worldwide), path loss with
				the Longley-Rice / ITM model, first Fresnel zone, link budget with real antenna patterns, and vegetation loss where a link ends in woodland
				(ITU-R P.833).
			</li>
			<li><b class="text-foreground">Area coverage</b>: the link margin around a transmitter as a raster on the map, computed in the background.</li>
			<li><b class="text-foreground">Reports</b>: every calculation as a one-page PDF.</li>
			<li><b class="text-foreground">Sharing</b>: publish a calculation as a read-only link that works without an account, for 30 days.</li>
		</ul>
		<p class="mt-4 rounded-lg border px-4 py-3">
			<b class="text-foreground">Under development.</b> The models are still being checked against measured links, so results can be off and may change
			between versions. Accounts and stored projects can be reset without notice.
		</p>
	</div>
</div>
