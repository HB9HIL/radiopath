<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ApiError, api, call, parseNum, type Antenna, type ManualAntennaInput } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import * as Table from '$lib/components/ui/table';
	import Field from '$lib/Field.svelte';
	import FormErrors from '$lib/FormErrors.svelte';
	import ManualAntennaFields, { type ManualValues } from '$lib/ManualAntennaFields.svelte';
	import PageHead from '$lib/PageHead.svelte';

	const queryClient = useQueryClient();
	const antennas = createQuery(() => ({ queryKey: ['antennas'], queryFn: () => call(api.GET('/api/antennas')) }));
	const blank: ManualValues = { name: '', gain_dbi: '0', beamwidth_deg: '65', v_beamwidth_deg: '', front_back_db: '25', freq_mhz: '' };
	let error: unknown = $state(null);
	let file: FileList | undefined = $state();
	let importName = $state('');
	let manual = $state({ ...blank });
	let editing: Antenna | null = $state(null);
	let edit = $state({ ...blank });
	let editError: unknown = $state(null);

	/** Reloads the list after a change. */
	const refresh = () => queryClient.invalidateQueries({ queryKey: ['antennas'] });

	/** Turns the text fields into the API body; fields that are no number give one message each. */
	function manualBody(v: ManualValues): ManualAntennaInput {
		const labels = {
			gain_dbi: 'Gain',
			beamwidth_deg: 'Beamwidth',
			v_beamwidth_deg: 'Vertical beamwidth',
			front_back_db: 'Front-to-back',
			freq_mhz: 'Frequency'
		} as const;
		const optional = new Set(['v_beamwidth_deg', 'front_back_db', 'freq_mhz']);
		const n: Record<string, number | undefined> = {};
		const nan: string[] = [];
		for (const [k, label] of Object.entries(labels)) {
			const raw = v[k as keyof ManualValues];
			if (optional.has(k) && raw.trim() === '') continue;
			n[k] = parseNum(raw);
			if (isNaN(n[k]!)) nan.push(`${label}: not a number`);
		}
		if (nan.length) throw new ApiError(422, 'Invalid input', nan);
		return {
			name: v.name,
			gain_dbi: n.gain_dbi!,
			beamwidth_deg: n.beamwidth_deg!,
			front_back_db: n.front_back_db ?? 0,
			v_beamwidth_deg: n.v_beamwidth_deg,
			freq_mhz: n.freq_mhz
		};
	}

	/** Uploads an MSI Planet file. */
	async function importFile(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		try {
			await call(
				api.POST('/api/antennas/import', {
					body: { file: '', name: importName },
					bodySerializer: () => {
						const fd = new FormData();
						fd.append('file', file![0]);
						fd.append('name', importName);
						return fd;
					}
				})
			);
			(e.target as HTMLFormElement).reset();
			importName = '';
			await refresh();
		} catch (err) {
			error = err;
			await refresh();
		}
	}

	/** Creates a sector antenna from the manual form. */
	async function create(e: SubmitEvent) {
		e.preventDefault();
		error = null;
		try {
			await call(api.POST('/api/antennas', { body: manualBody(manual) }));
			manual = { ...blank };
			await refresh();
		} catch (err) {
			error = err;
		}
	}

	/** Opens the edit dialog with the antenna's parameters. */
	function open(a: Antenna) {
		const m = a.manual!;
		edit = {
			name: a.name,
			gain_dbi: String(a.gain_dbi),
			beamwidth_deg: String(m.beamwidth_deg),
			v_beamwidth_deg: m.v_beamwidth_deg == null ? '' : String(m.v_beamwidth_deg),
			front_back_db: String(m.front_back_db),
			freq_mhz: a.freq_mhz == null ? '' : String(a.freq_mhz)
		};
		editError = null;
		editing = a;
	}

	/** Saves the edited antenna and closes the dialog. */
	async function save(e: SubmitEvent) {
		e.preventDefault();
		editError = null;
		try {
			await call(api.PUT('/api/antennas/{id}', { params: { path: { id: editing!.id } }, body: manualBody(edit) }));
			editing = null;
			await refresh();
		} catch (err) {
			editError = err;
		}
	}

	/** Deletes an antenna after confirmation. */
	async function remove(a: Antenna) {
		if (!confirm(`Delete antenna ${a.name}? Links and coverages using it become omnidirectional.`)) return;
		error = null;
		try {
			await call(api.DELETE('/api/antennas/{id}', { params: { path: { id: a.id } } }));
			await refresh();
		} catch (err) {
			error = err;
		}
	}
</script>

<PageHead title="Antennas" />
<FormErrors error={error ?? antennas.error} />
<div class="grid grid-cols-1 items-start gap-5 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
	<div>
		<div class="rounded-lg border bg-card">
			<Table.Root>
				<Table.Header>
					<Table.Row>
						<Table.Head>Azimuth</Table.Head>
						<Table.Head>Elevation</Table.Head>
						<Table.Head>Name</Table.Head>
						<Table.Head class="text-right">Gain</Table.Head>
						<Table.Head class="text-right">Front/back</Table.Head>
						<Table.Head class="text-right">Frequency</Table.Head>
						<Table.Head></Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each antennas.data ?? [] as a (a.id)}
						<Table.Row>
							<Table.Cell class="w-28">{@html a.pattern_svg}</Table.Cell>
							<Table.Cell class="w-28">
								{#if a.elevation_svg}{@html a.elevation_svg}{:else}<span class="text-muted-foreground" title="mirrors the azimuth cut">&mdash;</span
									>{/if}
							</Table.Cell>
							<Table.Cell>
								{a.name}{#if a.source}<br /><span class="font-mono text-xs text-muted-foreground">{a.source}</span>{/if}
							</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">{a.gain_dbi.toFixed(1)} dBi</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">{a.front_db.toFixed(1)} dB</Table.Cell>
							<Table.Cell class="text-right font-mono tabular-nums">
								{#if a.freq_mhz}{a.freq_mhz.toFixed(0)} MHz{:else}<span class="text-muted-foreground">&mdash;</span>{/if}
							</Table.Cell>
							<Table.Cell class="space-x-1 text-right whitespace-nowrap">
								{#if a.manual}<Button variant="outline" size="sm" onclick={() => open(a)}>Edit</Button>{/if}
								<Button variant="destructive" size="sm" onclick={() => remove(a)}>Delete</Button>
							</Table.Cell>
						</Table.Row>
					{:else}
						<Table.Row>
							<Table.Cell colspan={7} class="py-6 text-center text-muted-foreground"
								>No antennas yet. Upload an MSI Planet file to get started.</Table.Cell
							>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		</div>
		{#if antennas.data?.length}
			<p class="table-note mt-2 text-xs text-muted-foreground">{antennas.data.length} antenna{antennas.data.length === 1 ? '' : 's'}</p>
		{/if}
	</div>
	<div class="grid gap-5">
		<form class="grid gap-3 rounded-lg border bg-card p-4" aria-label="Import pattern" onsubmit={importFile}>
			<h2 class="border-b pb-1 text-xs font-medium text-muted-foreground">Import pattern</h2>
			<Field label="File"><Input type="file" name="file" accept=".msi,.pln,.ant,.txt" required bind:files={file} /></Field>
			<Field label="Name"><Input name="name" placeholder="from the file" bind:value={importName} /></Field>
			<div><Button type="submit">Import</Button></div>
			<p class="text-xs text-muted-foreground">
				MSI Planet format (<span class="font-mono">.msi</span>, <span class="font-mono">.pln</span>), as shipped by antenna manufacturers. The
				horizontal and, when present, the vertical pattern are used: 360 attenuation values in dB below the main lobe each. Without a vertical block
				the elevation cut mirrors the azimuth cut. A gain without a unit is read as dBd.
			</p>
		</form>
		<form class="grid gap-3 rounded-lg border bg-card p-4" aria-label="Describe an antenna" onsubmit={create}>
			<h2 class="border-b pb-1 text-xs font-medium text-muted-foreground">Or describe one</h2>
			<ManualAntennaFields bind:v={manual} />
			<div><Button type="submit">Create</Button></div>
			<p class="text-xs text-muted-foreground">
				Builds the usual sector shape: 12 (angle / beamwidth)&sup2; dB, capped at the front-to-back ratio. The beamwidth is the 3 dB width, 360 deg
				gives an omni; the vertical beamwidth shapes the elevation cut the same way. The front-to-back ratio is also the floor: nothing outside the
				main lobe, in azimuth or elevation, is attenuated more than that. Good enough for planning, not a substitute for the manufacturer's file.
			</p>
		</form>
	</div>
</div>

<Dialog.Root open={editing !== null} onOpenChange={(o) => !o && (editing = null)}>
	<Dialog.Content showCloseButton={false}>
		<Dialog.Header><Dialog.Title>Edit antenna</Dialog.Title></Dialog.Header>
		<FormErrors error={editError} />
		<form class="grid gap-3" onsubmit={save}>
			<ManualAntennaFields bind:v={edit} />
			<p class="text-xs text-muted-foreground">
				Links and coverages using this antenna take over the new gain and pattern. A computed coverage keeps its result until you recompute it.
			</p>
			<div class="flex gap-3">
				<Button type="submit">Save</Button>
				<Button variant="outline" onclick={() => (editing = null)}>Close</Button>
			</div>
		</form>
	</Dialog.Content>
</Dialog.Root>
