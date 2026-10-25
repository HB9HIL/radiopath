<script lang="ts">
	import { page } from '$app/state';
	import { createQuery } from '@tanstack/svelte-query';
	import { api, meQuery } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import ThemeToggle from '$lib/ThemeToggle.svelte';

	const me = createQuery(meQuery);
	// The last element marks sections still served as template pages: links to them need a full page load.
	const nav = [
		['/sites', 'Sites', 'off'],
		['/links', 'Links', 'off'],
		['/coverages', 'Coverage', ''],
		['/antennas', 'Antennas', 'off']
	];

	/** Ends the session and goes to the login page. */
	async function logout() {
		await api.DELETE('/api/session');
		location.assign('/login');
	}
</script>

<header class="top-0 z-30 flex min-h-12 flex-wrap items-center gap-x-6 border-b bg-card px-5 sm:sticky">
	<a class="text-[13px] font-semibold text-foreground" href="/" data-sveltekit-reload>Radiopath</a>
	{#if me.data}
		<nav class="order-last flex basis-full items-center gap-5 sm:order-none sm:basis-auto">
			{#each nav as [href, label, reload] (href)}
				<a
					class={[
						'-mb-px flex h-10 items-center border-b-2 text-[13px] hover:text-foreground sm:h-12',
						page.url.pathname.startsWith(href) ? 'border-foreground text-foreground' : 'border-transparent text-muted-foreground'
					]}
					{href}
					data-sveltekit-reload={reload}>{label}</a
				>
			{/each}
		</nav>
	{/if}
	<div class="ml-auto flex items-center gap-3">
		<ThemeToggle />
		{#if me.data}
			<Button variant="outline" size="sm" href="/account" title="Account" data-sveltekit-reload>{me.data.name}</Button>
			<Button variant="outline" size="sm" onclick={logout}>Logout</Button>
		{/if}
	</div>
</header>
