<script lang="ts">
	import Plus from '@lucide/svelte/icons/plus';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { toast } from 'svelte-sonner';
	import { api, call, type Site } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Table from '$lib/components/ui/table';
	import { ask } from '$lib/confirm.svelte';
	import FormErrors from '$lib/FormErrors.svelte';
	import { notifyError } from '$lib/notify';
	import PageHead from '$lib/PageHead.svelte';
	import SitesMap from '$lib/SitesMap.svelte';

	const queryClient = useQueryClient();
	const sites = createQuery(() => ({ queryKey: ['sites'], queryFn: () => call(api.GET('/api/sites')) }));

	/** Deletes a site after confirmation; a site still used by a link or coverage stays and the reason is shown. */
	async function remove(s: Site) {
		if (!(await ask({ title: `Delete site ${s.name}?`, action: 'Delete', destructive: true }))) return;
		try {
			await call(api.DELETE('/api/sites/{id}', { params: { path: { id: s.id } } }));
			toast.success(`Site ${s.name} deleted`);
			await queryClient.invalidateQueries({ queryKey: ['sites'] });
		} catch (err) {
			notifyError(err);
		}
	}
</script>

<PageHead title="Sites">
	<Button href="/sites/new"><Plus />New site</Button>
</PageHead>
<FormErrors error={sites.error} />
{#if sites.data}
	{@const list = sites.data}
	<div class="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
		<div>
			<div class="rounded-lg border bg-card">
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head>Name</Table.Head>
							<Table.Head class="text-right">Latitude</Table.Head>
							<Table.Head class="text-right">Longitude</Table.Head>
							<Table.Head class="text-right">Antenna</Table.Head>
							<Table.Head></Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each list as s (s.id)}
							<Table.Row>
								<Table.Cell><a class="text-primary" href="/sites/{s.id}">{s.name}</a></Table.Cell>
								<Table.Cell class="text-right font-mono tabular-nums">{s.lat.toFixed(5)}</Table.Cell>
								<Table.Cell class="text-right font-mono tabular-nums">{s.lon.toFixed(5)}</Table.Cell>
								<Table.Cell class="text-right font-mono tabular-nums">{s.antenna_height_m.toFixed(1)} m</Table.Cell>
								<Table.Cell class="space-x-1 text-right whitespace-nowrap">
									<Button variant="outline" size="sm" href="/sites/{s.id}">Edit</Button>
									<Button variant="outline" size="sm" href="/sites/new?from={s.id}">Duplicate</Button>
									<Button variant="destructive" size="sm" onclick={() => remove(s)}>Delete</Button>
								</Table.Cell>
							</Table.Row>
						{:else}
							<Table.Row>
								<Table.Cell colspan={5} class="py-6 text-center text-muted-foreground">
									No sites yet. <a class="text-primary" href="/sites/new">Add the first one</a>.
								</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			</div>
			{#if list.length}<p class="table-note mt-2 text-xs text-muted-foreground">{list.length} site{list.length === 1 ? '' : 's'}</p>{/if}
		</div>
		<SitesMap sites={list} />
	</div>
{/if}
