<script lang="ts">
	import { page } from '$app/state';
</script>

<main class="mx-auto mt-10 w-full max-w-136 px-5">
	<h1 class="border-b pb-3 text-lg font-semibold">{page.status === 404 ? 'Not found' : 'Error'}</h1>
	<p class="mt-4">{page.status === 404 ? 'This page does not exist.' : page.error?.message}</p>
	<a class="mt-4 inline-block text-primary" href="/">Back to start</a>
</main>
