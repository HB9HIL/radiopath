<script lang="ts">
	import { page } from '$app/state';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { api, call } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import FormErrors from '$lib/FormErrors.svelte';
	import LinkDetail from '$lib/LinkDetail.svelte';
	import ShareDialog from '$lib/ShareDialog.svelte';

	const id = Number(page.params.id);
	const path = { params: { path: { id } } };
	const queryClient = useQueryClient();
	const l = createQuery(() => ({ queryKey: ['links', id], queryFn: () => call(api.GET('/api/links/{id}', path)) }));
</script>

{#if l.data}
	<LinkDetail d={l.data} path="/links/{id}">
		{#snippet actions()}
			<Button variant="outline" href="/links">Back</Button>
			<Button variant="outline" href="/links/{id}/edit">Edit</Button>
			<Button variant="outline" href="/links/new?from={id}">Duplicate</Button>
			<ShareDialog
				what="link"
				share={l.data.share}
				create={() => api.PUT('/api/links/{id}/share', path)}
				revoke={() => api.DELETE('/api/links/{id}/share', path)}
				onchange={() => queryClient.invalidateQueries({ queryKey: ['links'] })}
			/>
		{/snippet}
	</LinkDetail>
{:else}
	<FormErrors error={l.error} />
{/if}
