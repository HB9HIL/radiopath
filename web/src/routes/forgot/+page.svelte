<script lang="ts">
	import { toast } from 'svelte-sonner';
	import { api, call } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Input } from '$lib/components/ui/input';
	import Field from '$lib/Field.svelte';
	import { notifyError } from '$lib/notify';

	let email = $state('');
	let busy = $state(false);

	/** Requests the reset link; the answer is the same whether the address has an account or not. */
	async function submit(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			await call(api.POST('/api/password/forgot', { body: { email } }));
			toast.success('If that address belongs to an account, a reset link is on its way.');
		} catch (err) {
			notifyError(err);
		}
		busy = false;
	}
</script>

<svelte:head><title>Forgot password – Radiopath</title></svelte:head>

<div class="mx-auto mt-10 w-full max-w-[26rem]">
	<h1 class="mb-4 text-lg font-semibold">Forgot password</h1>
	<form class="grid gap-4 rounded-lg border bg-card p-5" onsubmit={submit}>
		<Field label="Email">
			<!-- svelte-ignore a11y_autofocus -->
			<Input name="email" type="email" autocomplete="email" autofocus required bind:value={email} />
		</Field>
		<div class="flex flex-wrap items-center gap-4">
			<Button type="submit" disabled={busy}>Send reset link</Button>
			<a class="text-primary" href="/login">Back to login</a>
		</div>
	</form>
	<p class="mt-3 text-xs text-muted-foreground">We mail a link to the address of your account. It is valid for one hour.</p>
</div>
