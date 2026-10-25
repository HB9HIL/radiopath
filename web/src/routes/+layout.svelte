<script lang="ts">
	import '../app.css';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import Footer from '$lib/Footer.svelte';
	import Header from '$lib/Header.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	// Link analysis runs on every GET, so no refetch on window focus.
	const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false, retry: false } } });
</script>

<svelte:head>
	<title>Radiopath</title>
</svelte:head>

<QueryClientProvider client={queryClient}>
	<div class="flex min-h-screen flex-col">
		<Header />
		<main class="mx-auto w-full max-w-[1400px] px-5 py-5">
			{@render children()}
		</main>
		<Footer />
	</div>
</QueryClientProvider>
