import { browser } from '$app/environment';
import { resolveApiUrl } from '$lib/config/api';
import { writable } from 'svelte/store';

interface RealtimeConfig {
	realtimeWsUrl: string;
}

interface RealtimeStore {
	config: RealtimeConfig | null;
	loading: boolean;
	error: string | null;
}

function createRealtimeStore() {
	const { subscribe, set, update } = writable<RealtimeStore>({
		config: null,
		loading: true,
		error: null
	});

	async function loadConfig() {
		if (!browser) {
			set({
				config: null,
				loading: false,
				error: null
			});
			return;
		}

		try {
			update((state) => ({ ...state, loading: true, error: null }));

			const response = await fetch(resolveApiUrl('/api/v1/config'), {
				credentials: 'include'
			});
			if (!response.ok) {
				throw new Error(`Failed to fetch realtime config: ${response.statusText}`);
			}

			const config: RealtimeConfig = await response.json();
			set({
				config,
				loading: false,
				error: null
			});
		} catch (err) {
			const errorMessage = err instanceof Error ? err.message : 'Unknown error';
			console.error('Failed to load realtime config:', errorMessage);
			set({
				config: null,
				loading: false,
				error: errorMessage
			});
		}
	}

	if (browser) {
		void loadConfig();
	} else {
		set({
			config: null,
			loading: false,
			error: null
		});
	}

	return {
		subscribe,
		reload: loadConfig
	};
}

export const realtimeConfig = createRealtimeStore();
