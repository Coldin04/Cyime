import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

const deployTarget = (process.env.DEPLOY_TARGET ?? '').trim().toLowerCase();

async function createAdapter() {
	switch (deployTarget) {
		case 'cloudflare': {
			const { default: adapter } = await import('@sveltejs/adapter-cloudflare');
			return adapter();
		}
		case 'edgeone': {
			const { default: adapter } = await import('@edgeone/sveltekit');
			return adapter();
		}
		default: {
			const { default: adapter } = await import('@sveltejs/adapter-auto');
			return adapter();
		}
	}
}

/** @type {import('@sveltejs/kit').Config} */
const config = {
	// Consult https://svelte.dev/docs#compile-time-svelte-preprocess
	// for more information about preprocessors
	preprocess: vitePreprocess(),

	kit: {
		adapter: await createAdapter(),
		alias: {
			$paraglide: './src/paraglide'
		}
	}
};

export default config;
