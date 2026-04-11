import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import { resolveApiUrl } from '$lib/config/api';
import { writable } from 'svelte/store';

function isEnvEnabled(value: string | undefined): boolean {
	if (!value || value.trim() === '') {
		return true;
	}
	return ['1', 'true', 'yes', 'y', 'on'].includes(value.trim().toLowerCase());
}

const frontendCollaborationEnabled = isEnvEnabled(env.PUBLIC_COLLABORATION_ENABLED);

interface RealtimeConfig {
	collaborationEnabled: boolean;
	realtimeWsUrl: string;
	documentImageMaxBytes: number;
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

			const config = (await response.json()) as RealtimeConfig;
			set({
				config: {
					...config,
					collaborationEnabled: frontendCollaborationEnabled && config.collaborationEnabled
				},
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
