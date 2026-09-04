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

const frontendSharingEnabled = isEnvEnabled(env.PUBLIC_COLLABORATION_ENABLED);

interface ClientConfig {
	sharingEnabled: boolean;
	documentImageMaxBytes: number;
}

interface ClientConfigResponse {
	collaborationEnabled: boolean;
	documentImageMaxBytes: number;
}

interface ClientConfigStore {
	config: ClientConfig | null;
	loading: boolean;
	error: string | null;
}

function createClientConfigStore() {
	const { subscribe, set, update } = writable<ClientConfigStore>({
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
				throw new Error(`Failed to fetch client config: ${response.statusText}`);
			}

			const responseConfig = (await response.json()) as ClientConfigResponse;
			set({
				config: {
					sharingEnabled: frontendSharingEnabled && responseConfig.collaborationEnabled,
					documentImageMaxBytes: responseConfig.documentImageMaxBytes
				},
				loading: false,
				error: null
			});
		} catch (err) {
			const errorMessage = err instanceof Error ? err.message : 'Unknown error';
			console.error('Failed to load client config:', errorMessage);
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

export const clientConfig = createClientConfigStore();
