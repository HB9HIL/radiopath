<script lang="ts">
	import { call, type Share } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import * as Dialog from '$lib/components/ui/dialog';
	import { Input } from '$lib/components/ui/input';
	import { day } from '$lib/format';
	import FormErrors from '$lib/FormErrors.svelte';

	// create and revoke call the share API of the object; onchange reloads it.
	let {
		what,
		hint = '',
		share,
		create,
		revoke,
		onchange
	}: {
		what: string;
		hint?: string;
		share?: Share;
		create: () => Promise<{ response: Response }>;
		revoke: () => Promise<{ response: Response }>;
		onchange: () => Promise<unknown>;
	} = $props();
	let open = $state(false);
	let copied = $state(false);
	let error: unknown = $state(null);

	/** Runs a share call and reloads the page data; errors stay in the dialog. */
	async function run(req: () => Promise<{ response: Response }>) {
		error = null;
		try {
			await call(req());
			await onchange();
		} catch (err) {
			error = err;
		}
	}

	/** Copies the public URL to the clipboard. */
	async function copy() {
		await navigator.clipboard?.writeText(share!.url);
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}
</script>

<Button variant="outline" class={share ? 'btn-shared' : ''} title={share ? 'Public link active' : undefined} onclick={() => (open = true)}>
	{#if share}<span class="size-1.5 rounded-full bg-ok"></span>{/if}Share
</Button>

<Dialog.Root bind:open>
	<Dialog.Content showCloseButton={false} class="sm:max-w-xl">
		<Dialog.Header><Dialog.Title>Share this {what}</Dialog.Title></Dialog.Header>
		<FormErrors {error} />
		{#if share}
			<div>
				<p class="mb-1 text-xs text-muted-foreground">Public link</p>
				<div class="flex flex-wrap items-center gap-3">
					<Input class="max-w-120 font-mono text-xs" readonly value={share.url} onfocus={(e) => e.currentTarget.select()} />
					<Button variant="outline" onclick={copy}>{copied ? 'Copied' : 'Copy'}</Button>
				</div>
				<p class="mt-2 text-xs text-muted-foreground">
					Anyone with this link sees this page without an account, until {day(share.expires_at)}.{hint}
				</p>
			</div>
			<div class="flex flex-wrap gap-3">
				<Button variant="outline" onclick={() => run(create)}>Extend by 30 days</Button>
				<Button variant="destructive" onclick={() => run(revoke)}>Revoke</Button>
				<Button variant="outline" onclick={() => (open = false)}>Close</Button>
			</div>
		{:else}
			<p class="text-xs text-muted-foreground">Creates a link that shows this page to anyone without an account, for 30 days.{hint}</p>
			<div class="flex flex-wrap gap-3">
				<Button onclick={() => run(create)}>Create public link</Button>
				<Button variant="outline" onclick={() => (open = false)}>Close</Button>
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>
