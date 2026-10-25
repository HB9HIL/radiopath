import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

// The Go server; in make dev Vite runs in a container and reaches it on the host.
const backend = process.env.BACKEND ?? 'http://localhost:8080';

// Everything the Go server answers itself.
const goPaths =
	'^/(api|admin|tiles|healthz|readyz|robots\\.txt)(/|$)|^/(links|coverages|s)/.+\\.(png|pdf)(\\?|$)|^/coverages/\\d+/events';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: { [goPaths]: backend }
	}
});
