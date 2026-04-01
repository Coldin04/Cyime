import * as Y from 'yjs';
import { HocuspocusProvider, HocuspocusProviderWebsocket } from '@hocuspocus/provider';
import type { Awareness } from 'y-protocols/awareness';
import type {
	onAuthenticationFailedParameters,
	onStatusParameters,
	onSyncedParameters
} from '@hocuspocus/provider';

interface ProviderConfig {
	wsUrl: string;
	documentId: string;
	userId: string;
	token: string;
}

interface ProviderInstance {
	provider: HocuspocusProvider | null;
	doc: Y.Doc | null;
	awareness: Awareness | null;
	isConnected: boolean;
	isSynced: boolean;
	error: string | null;
}

class YjsProviderManager {
	private instances = new Map<string, ProviderInstance>();
	private readonly CONNECTION_TIMEOUT = 10000;

	async createProvider(config: ProviderConfig): Promise<ProviderInstance> {
		const docId = config.documentId;

		if (this.instances.has(docId)) {
			return this.instances.get(docId)!;
		}

		const instance: ProviderInstance = {
			provider: null,
			doc: null,
			awareness: null,
			isConnected: false,
			isSynced: false,
			error: null
		};

		try {
			const ydoc = new Y.Doc();
			const websocketProvider = new HocuspocusProviderWebsocket({
				url: this.buildWebSocketUrl(config.wsUrl),
				autoConnect: false,
				maxAttempts: 2,
				initialDelay: 0,
				delay: 1000,
				minDelay: 1000,
				factor: 1.5
			});
			const provider = new HocuspocusProvider({
				websocketProvider,
				name: `doc:${docId}`,
				document: ydoc,
				token: config.token,
				onStatus: ({ status }: onStatusParameters) => {
					instance.isConnected = status === 'connected';
					if (status === 'connected') {
						instance.error = null;
					} else if (status === 'disconnected') {
						console.warn(`[Yjs] Disconnected from collaboration server for ${docId}`);
					}
				},
				onSynced: ({ state }: onSyncedParameters) => {
					instance.isSynced = state;
					void state;
				},
				onAuthenticationFailed: ({ reason }: onAuthenticationFailedParameters) => {
					instance.error = reason || 'Authentication failed';
					instance.isConnected = false;
					console.error(`[Yjs] Authentication failed for ${docId}: ${instance.error}`);
				}
			});

			instance.provider = provider;
			instance.doc = ydoc;
			instance.awareness = provider.awareness;

			provider.setAwarenessField('user', {
				id: config.userId
			});
			provider.attach();

			this.instances.set(docId, instance);
			void websocketProvider.connect().catch((error: unknown) => {
				const errorMsg = error instanceof Error ? error.message : String(error);
				instance.error = errorMsg;
				instance.isConnected = false;
				console.error(`[Yjs] Websocket connect failed for ${docId}: ${errorMsg}`);
			});
			await this.waitForConnection(provider, this.CONNECTION_TIMEOUT);

			return instance;
		} catch (error) {
			const errorMsg = error instanceof Error ? error.message : 'Unknown error';
			instance.error = errorMsg;
			instance.isConnected = false;
			instance.isSynced = false;

			console.warn(
				`[Yjs] Failed to create Hocuspocus provider for ${docId}: ${errorMsg}. Falling back to local mode.`
			);

			if (instance.provider) {
				instance.provider.destroy();
				instance.provider = null;
				instance.awareness = null;
			}

			try {
				const ydoc = new Y.Doc();
				instance.doc = ydoc;
			} catch (fallbackError) {
				console.error('[Yjs] Fallback failed:', fallbackError);
			}

			this.instances.set(docId, instance);
			return instance;
		}
	}

	private async waitForConnection(provider: HocuspocusProvider, timeout: number): Promise<void> {
		if (provider.configuration.websocketProvider.status === 'connected') {
			return;
		}

		return new Promise((resolve, reject) => {
			const timer = setTimeout(() => {
				cleanup();
				console.warn(`[Yjs] Websocket timed out for ${provider.configuration.name}`);
				reject(new Error('WebSocket connection timeout'));
			}, timeout);

			const handleStatus = ({ status }: { status: string }) => {
				if (status === 'connected') {
					cleanup();
					resolve();
				}
			};

			const handleAuthenticationFailed = ({ reason }: { reason: string }) => {
				cleanup();
				console.warn(`[Yjs] Authentication rejected for ${provider.configuration.name}: ${reason}`);
				reject(new Error(reason || 'Authentication failed'));
			};

			const cleanup = () => {
				clearTimeout(timer);
				provider.off('status', handleStatus);
				provider.off('authenticationFailed', handleAuthenticationFailed);
			};

			provider.on('status', handleStatus);
			provider.on('authenticationFailed', handleAuthenticationFailed);
		});
	}

	private buildWebSocketUrl(baseUrl: string): string {
		let url = baseUrl;

		if (
			!url.startsWith('http://') &&
			!url.startsWith('https://') &&
			!url.startsWith('ws://') &&
			!url.startsWith('wss://')
		) {
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
			url = `${protocol}//${window.location.host}${url}`;
		}

		return url.replace(/^https:/, 'wss:').replace(/^http:/, 'ws:');
	}

	stopReconnects(documentId: string): void {
		const instance = this.instances.get(documentId);
		instance?.provider?.configuration.websocketProvider.disconnect();
	}

	getProvider(documentId: string): ProviderInstance | undefined {
		return this.instances.get(documentId);
	}

	destroyProvider(documentId: string): void {
		const instance = this.instances.get(documentId);
		if (!instance) {
			return;
		}

		instance.provider?.destroy();
		instance.doc?.destroy();
		this.instances.delete(documentId);
	}

	destroyAll(): void {
		for (const [docId] of this.instances) {
			this.destroyProvider(docId);
		}
	}
}

export const yjsProvider = new YjsProviderManager();
export type { ProviderConfig, ProviderInstance };
