<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { api, call, type Link } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import FormErrors from '$lib/FormErrors.svelte';
	import PageHead from '$lib/PageHead.svelte';

	const queryClient = useQueryClient();
	const links = createQuery(() => ({ queryKey: ['links'], queryFn: () => call(api.GET('/api/links')) }));
	const sites = createQuery(() => ({ queryKey: ['sites'], queryFn: () => call(api.GET('/api/sites')) }));
	const nSites = $derived(sites.data?.length ?? 0);
	let error: unknown = $state(null);

	/** Deletes a link after confirmation. */
	async function remove(l: Link) {
		if (!confirm(`Delete link ${l.name}?`)) return;
		error = null;
		try {
			await call(api.DELETE('/api/links/{id}', { params: { path: { id: l.id } } }));
			await queryClient.invalidateQueries({ queryKey: ['links'] });
		} catch (err) {
			error = err;
		}
	}
</script>

<PageHead title="Links">
	{#if nSites >= 2}
		<Button href="/links/new"><Plus />New link</Button>
	{:else}
		<Button disabled title="A link needs two sites"><Plus />New link</Button>
	{/if}
</PageHead>
<FormErrors error={error ?? links.error} />
{#if links.data && sites.data}
	{@const list = links.data}
	<div class="rounded-lg border bg-card">
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head>Name</Table.Head>
					<Table.Head>Site A (TX)</Table.Head>
					<Table.Head>Site B (RX)</Table.Head>
					<Table.Head class="text-right">Frequency</Table.Head>
					<Table.Head></Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each list as l (l.id)}
					<Table.Row>
						<Table.Cell><a class="text-primary" href="/links/{l.id}">{l.name}</a></Table.Cell>
						<Table.Cell>{l.site_a.name}</Table.Cell>
						<Table.Cell>{l.site_b.name}</Table.Cell>
						<Table.Cell class="text-right font-mono tabular-nums">{l.frequency_mhz.toFixed(3)} MHz</Table.Cell>
						<Table.Cell class="space-x-1 text-right whitespace-nowrap">
							<Button variant="outline" size="sm" href="/links/{l.id}/edit">Edit</Button>
							<Button variant="outline" size="sm" href="/links/new?from={l.id}">Duplicate</Button>
							<Button variant="destructive" size="sm" onclick={() => remove(l)}>Delete</Button>
						</Table.Cell>
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell colspan={5} class="py-6 text-center text-muted-foreground">
							{#if nSites >= 2}
								No links yet. <a class="text-primary" href="/links/new">Add the first one</a>.
							{:else if nSites}
								A link needs two sites. <a class="text-primary" href="/sites/new">Add the second one</a> first.
							{:else}
								A link connects two sites. <a class="text-primary" href="/sites/new">Create your sites</a> first.
							{/if}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</div>
	{#if list.length}<p class="table-note mt-2 text-xs text-muted-foreground">{list.length} link{list.length === 1 ? '' : 's'}</p>{/if}
{/if}
