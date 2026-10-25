<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import { api, call, type Coverage } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import { ask } from '$lib/confirm.svelte';
	import FormErrors from '$lib/FormErrors.svelte';
	import { notifyError } from '$lib/notify';
	import JobBadge from '$lib/JobBadge.svelte';
	import PageHead from '$lib/PageHead.svelte';

	const queryClient = useQueryClient();
	const coverages = createQuery(() => ({ queryKey: ['coverages'], queryFn: () => call(api.GET('/api/coverages')) }));
	const sites = createQuery(() => ({ queryKey: ['sites'], queryFn: () => call(api.GET('/api/sites')) }));
	const nSites = $derived(sites.data?.length ?? 0);

	/** Deletes a coverage after confirmation. */
	async function remove(c: Coverage) {
		if (!(await ask({ title: `Delete coverage ${c.name}?`, action: 'Delete', destructive: true }))) return;
		try {
			await call(api.DELETE('/api/coverages/{id}', { params: { path: { id: c.id } } }));
			toast.success(`Coverage ${c.name} deleted`);
			await queryClient.invalidateQueries({ queryKey: ['coverages'] });
		} catch (err) {
			notifyError(err);
		}
	}
</script>

<PageHead title="Coverage">
	{#if nSites}
		<Button href="/coverages/new"><Plus />New coverage</Button>
	{:else}
		<Button disabled title="A coverage needs a site"><Plus />New coverage</Button>
	{/if}
</PageHead>
<FormErrors error={coverages.error} />
{#if coverages.data && sites.data}
	{@const list = coverages.data}
	<div class="rounded-lg border bg-card">
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head>Name</Table.Head>
					<Table.Head>Site (TX)</Table.Head>
					<Table.Head class="text-right">Frequency</Table.Head>
					<Table.Head class="text-right">Range</Table.Head>
					<Table.Head>Computed</Table.Head>
					<Table.Head></Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each list as c (c.id)}
					<Table.Row>
						<Table.Cell><a class="text-primary" href="/coverages/{c.id}">{c.name}</a></Table.Cell>
						<Table.Cell>{c.site.name}</Table.Cell>
						<Table.Cell class="text-right font-mono tabular-nums">{c.frequency_mhz.toFixed(3)} MHz</Table.Cell>
						<Table.Cell class="text-right font-mono tabular-nums">{c.range_km.toFixed(1)} km</Table.Cell>
						<Table.Cell>
							{#if c.job.state === 'queued' || c.job.state === 'running' || c.job.state === 'failed'}
								<JobBadge state={c.job.state} />
							{:else if c.computed_at}
								<span class="font-mono text-xs">{new Date(c.computed_at).toLocaleString('sv-SE').slice(0, 16)}</span>
							{:else}
								<span class="text-muted-foreground">not yet</span>
							{/if}
						</Table.Cell>
						<Table.Cell class="space-x-1 text-right whitespace-nowrap">
							<Button variant="outline" size="sm" href="/coverages/{c.id}/edit">Edit</Button>
							<Button variant="outline" size="sm" href="/coverages/new?from={c.id}">Duplicate</Button>
							<Button variant="destructive" size="sm" onclick={() => remove(c)}>Delete</Button>
						</Table.Cell>
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell colspan={6} class="py-6 text-center text-muted-foreground">
							{#if nSites}
								No coverages yet. <a class="text-primary" href="/coverages/new">Add the first one</a>.
							{:else}
								A coverage starts at a site. <a class="text-primary" href="/sites/new">Create one</a> first.
							{/if}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	</div>
	{#if list.length}<p class="table-note mt-2 text-xs text-muted-foreground">{list.length} coverage{list.length === 1 ? '' : 's'}</p>{/if}
{/if}
