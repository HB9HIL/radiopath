<script lang="ts">
	let { state, id }: { state: string; id?: string } = $props();
	const colors: Record<string, string> = {
		idle: 'border-border text-muted-foreground',
		queued: 'border-warn text-warn',
		running: 'border-primary text-primary',
		done: 'border-ok text-ok',
		failed: 'border-destructive text-destructive'
	};
</script>

<span {id} class={['inline-flex items-center gap-1.5 border px-1.5 py-px text-xs font-medium', colors[state]]}>
	<span class={['size-1.5 rounded-full bg-current', state === 'running' && 'animate-pulse']}></span>{state}
</span>
