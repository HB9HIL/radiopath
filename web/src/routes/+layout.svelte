<script lang="ts">
	import '../app.css';
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	// Link analysis runs on every GET, so no refetch on window focus.
	const queryClient = new QueryClient({ defaultOptions: { queries: { refetchOnWindowFocus: false } } });
</script>

<svelte:head>
	<title>Radiopath</title>
</svelte:head>

<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>
