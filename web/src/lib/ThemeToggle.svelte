<script lang="ts">
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import { Button } from '$lib/components/ui/button';

	const root = document.documentElement;
	let dark = $state(root.dataset.theme === 'dark');

	/** Switches between light and dark and remembers the choice. */
	function toggle() {
		dark = !dark;
		root.dataset.theme = dark ? 'dark' : 'light';
		try {
			localStorage.setItem('cp-theme', root.dataset.theme);
		} catch {}
	}

	// Without a stored choice the theme follows the system.
	$effect(() => {
		const mq = matchMedia('(prefers-color-scheme: dark)');
		/** Applies a system theme change unless the user picked one. */
		const follow = (e: MediaQueryListEvent) => {
			try {
				if (localStorage.getItem('cp-theme')) return;
			} catch {
				return;
			}
			dark = e.matches;
			root.dataset.theme = dark ? 'dark' : 'light';
		};
		mq.addEventListener('change', follow);
		return () => mq.removeEventListener('change', follow);
	});
</script>

<Button variant="ghost" size="icon-sm" title="Light / dark" aria-label="Light / dark" aria-pressed={dark} onclick={toggle}>
	{#if dark}<Sun />{:else}<Moon />{/if}
</Button>
