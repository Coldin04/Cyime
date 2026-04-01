<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import type { JSONContent } from '@tiptap/core';
	import { browser } from '$app/environment';
	import { beforeNavigate, goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import Editor from '$lib/components/editor/Editor.svelte';
	import EditorTopBar from '$lib/components/editor/EditorTopBar.svelte';
	import ConfirmDialog from '$lib/components/common/ConfirmDialog.svelte';
	import {
		defaultAutoSaveEnabled,
		defaultAutoSaveIntervalSeconds,
		readAutoSaveEnabled,
		readAutoSaveIntervalSeconds
	} from '$lib/components/editor/autoSave';
	import { auth } from '$lib/stores/auth';
	import { resolveApiUrl } from '$lib/config/api';
	import { realtimeConfig } from '$lib/stores/realtime';
	import { yjsProvider, type ProviderInstance } from '$lib/utils/yjsProvider';
	import { getDocumentContent, resolveAssetReadURLs, updateDocumentContent } from '$lib/api/editor';
	import { getDocumentDetails, updateDocumentImageTarget } from '$lib/api/workspace';
	import { getImageBedConfigs, type ImageBedConfig } from '$lib/api/user';
	import {
		getDocumentImageTargetLabel,
		getDocumentImageTargetOptions
	} from '$lib/components/editor/documentImageTargets';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';

	let title = $state('');
	let manualExcerpt = $state('');
	let myRole = $state<'owner' | 'collaborator' | 'editor' | 'viewer' | string>('owner');
	let publicAccess = $state<'private' | 'authenticated' | 'public' | string>('private');
	let publicUrl = $state('');
	const EMPTY_DOC: JSONContent = {
		type: 'doc',
		content: [{ type: 'paragraph' }]
	};

	let content = $state<JSONContent>(EMPTY_DOC);
	let documentType = $state<'rich_text' | 'table' | string>('rich_text');
	let preferredImageTargetId = $state('managed-r2');
	let imageBedConfigs = $state<ImageBedConfig[]>([]);
	let isUpdatingImageTarget = $state(false);
	let isSaving = $state(false);
	let lastSaved = $state<Date | null>(null);
	let hasUnsavedChanges = $state(false);
	let isLoading = $state(true);
	let collaboration = $state<ProviderInstance | null>(null);
	let collaborationError = $state<string | null>(null);
	let collaborationIndicator = $state<
		{ kind: 'single' | 'single-offline' | 'multi-pending' | 'multi'; label: string } | null
	>(null);
	let detachCollaborationListeners: (() => void) | null = null;
	let presenceCount = $state(0);
	let presenceConnected = $state(false);
	let hasAttemptedPresence = $state(false);
	let presenceSessionId = $state('');
	let presenceHeartbeatTimer: number | null = null;
	let isInitializingCollaboration = $state(false);
	let lastCollaborationAttemptAt = $state(0);
	let isYjsConnected = $state(false);
	let isLeaveConfirmOpen = $state(false);
	let pendingNavigationUrl = $state<string | null>(null);
	let bypassLeaveGuard = $state(false);
	let autoSaveEnabled = $state(defaultAutoSaveEnabled);
	let autoSaveIntervalSeconds = $state(defaultAutoSaveIntervalSeconds);
	const availableImageTargets = $derived(getDocumentImageTargetOptions(imageBedConfigs));
	const currentImageTargetLabel = $derived(
		getDocumentImageTargetLabel(preferredImageTargetId, availableImageTargets)
	);

	function cloneContentJson(value: JSONContent): JSONContent {
		return JSON.parse(JSON.stringify(value)) as JSONContent;
	}

	type ImageNodeRecord = Record<string, unknown> & {
		attrs?: Record<string, unknown>;
	};

	function collectImageNodes(value: unknown, nodes: ImageNodeRecord[]) {
		if (!value || typeof value !== 'object') {
			return;
		}
		const node = value as ImageNodeRecord;
		if (node.type === 'image') {
			nodes.push(node);
		}
		const children = node.content;
		if (Array.isArray(children)) {
			for (const child of children) {
				collectImageNodes(child, nodes);
			}
		}
	}

	function getManagedAssetId(attrs: Record<string, unknown>): string | null {
		const raw = attrs.assetId;
		return typeof raw === 'string' && raw.trim() !== '' ? raw.trim() : null;
	}

	async function refreshSignedImageSources(input: JSONContent): Promise<JSONContent> {
		const cloned = cloneContentJson(input);
		const imageNodes: ImageNodeRecord[] = [];
		collectImageNodes(cloned, imageNodes);
		if (imageNodes.length === 0) {
			return cloned;
		}

		const assetIds = Array.from(
			new Set(
				imageNodes
					.map((node) => getManagedAssetId((node.attrs ?? {}) as Record<string, unknown>))
					.filter((value): value is string => value !== null)
			)
		);
		if (assetIds.length === 0) {
			return cloned;
		}

		let resolved: Awaited<ReturnType<typeof resolveAssetReadURLs>> | null = null;
		try {
			resolved = await resolveAssetReadURLs(assetIds);
		} catch (error) {
			console.error('[Load] Failed to resolve image URLs:', error);
			return cloned;
		}
		if (!resolved) {
			return cloned;
		}
		const resolvedMap = new Map(
			resolved.items
				.filter((item) => item.assetId && item.url)
				.map((item) => [item.assetId, item.url as string])
		);

		for (const node of imageNodes) {
			const attrs = (node.attrs ?? {}) as Record<string, unknown>;
			const assetId = getManagedAssetId(attrs);
			if (!assetId) continue;
			const resolvedURL = resolvedMap.get(assetId);
			if (!resolvedURL) {
				console.error('[Load] Failed to resolve image URL for asset:', assetId);
				continue;
			}
			attrs.src = resolvedURL;
			node.attrs = attrs;
		}

		return cloned;
	}

	function normalizeManagedImagesForSave(input: JSONContent): JSONContent {
		const cloned = cloneContentJson(input);
		const imageNodes: ImageNodeRecord[] = [];
		collectImageNodes(cloned, imageNodes);

		for (const node of imageNodes) {
			const attrs = (node.attrs ?? {}) as Record<string, unknown>;
			const assetId = getManagedAssetId(attrs);
			if (!assetId) {
				continue;
			}

			delete attrs.src;
			node.attrs = attrs;
		}

		return cloned;
	}

	// Manually bridge the SvelteKit `page` store to a Svelte 5 signal
	// since this environment is in runes-mode but likely on an older Svelte 5 version.
	let pageSignal = $state(get(page));
	page.subscribe((p) => (pageSignal = p));
	let authSignal = $state(get(auth));
	auth.subscribe((state) => (authSignal = state));
	const documentId = $derived(pageSignal.params?.id);

	beforeNavigate((navigation) => {
		if (!browser || !hasUnsavedChanges || bypassLeaveGuard) {
			return;
		}
		if (!navigation.to?.url) return;

		navigation.cancel();
		pendingNavigationUrl = `${navigation.to.url.pathname}${navigation.to.url.search}${navigation.to.url.hash}`;
		isLeaveConfirmOpen = true;
	});

	function handleCancelLeave() {
		isLeaveConfirmOpen = false;
		pendingNavigationUrl = null;
	}

	async function handleConfirmLeave() {
		if (!pendingNavigationUrl) {
			handleCancelLeave();
			return;
		}

		const target = pendingNavigationUrl;
		isLeaveConfirmOpen = false;
		pendingNavigationUrl = null;
		bypassLeaveGuard = true;
		await goto(target);
		bypassLeaveGuard = false;
	}

	async function handleLeaveWithoutSave() {
		await handleConfirmLeave();
	}

	function handleContentChange(newContent: JSONContent) {
		if (isLoading) return;
		hasUnsavedChanges = true;
		content = newContent;
		if (documentId) {
			void connectPresenceSocket(documentId);
			if (!collaboration) {
				void startCollaboration(documentId, 'editing');
			}
		}
	}

	function handleTitleChange(newTitle: string) {
		title = newTitle;
	}

	function handleExcerptChange(newExcerpt: string) {
		manualExcerpt = newExcerpt;
	}

	function handlePublicAccessChange(nextPublicAccess: string, nextPublicURL: string) {
		publicAccess = nextPublicAccess;
		publicUrl = nextPublicURL;
	}

	function updateCollaborationIndicator() {
		if (presenceCount > 1) {
			if (isYjsConnected) {
				collaborationIndicator = { kind: 'multi', label: `协作已连接，当前有 ${presenceCount} 个会话在线` };
				return;
			}

			collaborationIndicator = {
				kind: 'multi-pending',
				label: `检测到 ${presenceCount} 个会话在线，但协作连接尚未建立`
			};
			return;
		}

		if (collaborationError || (hasAttemptedPresence && !presenceConnected && !collaboration)) {
			collaborationIndicator = { kind: 'single-offline', label: '协作连接已断开，当前为单人模式' };
			return;
		}

		collaborationIndicator = { kind: 'single', label: '当前仅你在线编辑' };
	}

	function ensurePresenceSessionId() {
		if (presenceSessionId !== '') {
			return presenceSessionId;
		}
		presenceSessionId = crypto.randomUUID();
		return presenceSessionId;
	}

	async function resolveRealtimeWsUrl(): Promise<string> {
		if (!authSignal.token) {
			throw new Error('Missing access token');
		}

		let wsUrl = get(realtimeConfig).config?.realtimeWsUrl ?? '';
		if (!wsUrl) {
			await realtimeConfig.reload();
			wsUrl = get(realtimeConfig).config?.realtimeWsUrl ?? '';
		}
		if (!wsUrl) {
			throw new Error('Realtime WebSocket URL is not configured');
		}

		return wsUrl;
	}

	function buildRealtimePresenceURL(nextDocumentId: string): string {
		const url = new URL(resolveApiUrl('/api/v1/workspace/documents/_/presence'), window.location.origin);
		url.pathname = url.pathname.replace('/_', `/${nextDocumentId}`);
		url.search = '';
		return url.toString();
	}

	async function fetchCollaborationPresence(nextDocumentId: string): Promise<number> {
		const response = await fetch(buildRealtimePresenceURL(nextDocumentId), {
			headers: {
				Authorization: `Bearer ${authSignal.token}`
			},
			credentials: 'include'
		});
		if (!response.ok) {
			throw new Error(`Failed to fetch collaboration presence: ${response.status}`);
		}

		const payload = (await response.json()) as { connectedCount?: number };
		return typeof payload.connectedCount === 'number' ? payload.connectedCount : 0;
	}

	function clearPresenceSocket() {
		if (presenceHeartbeatTimer !== null) {
			window.clearInterval(presenceHeartbeatTimer);
			presenceHeartbeatTimer = null;
		}
	}

	async function connectPresenceSocket(nextDocumentId: string) {
		if (!browser || presenceHeartbeatTimer !== null) {
			return;
		}

		const token = authSignal.token;
		if (!token) {
			return;
		}

		hasAttemptedPresence = true;
		const sessionId = ensurePresenceSessionId();
		const heartbeat = async () => {
			try {
				const response = await fetch(buildRealtimePresenceURL(nextDocumentId), {
					method: 'PUT',
					headers: {
						Authorization: `Bearer ${token}`,
						'Content-Type': 'application/json',
						'X-Presence-Session-Id': sessionId
					},
					credentials: 'include',
					body: JSON.stringify({ sessionId })
				});
				if (!response.ok) {
					throw new Error(`Presence heartbeat failed: ${response.status}`);
				}
				const payload = (await response.json()) as { connectedCount?: number };
				presenceCount = typeof payload.connectedCount === 'number' ? payload.connectedCount : 0;
				presenceConnected = true;
				collaborationError = isYjsConnected ? null : collaborationError;
				console.log(`[Presence] ${nextDocumentId} has ${presenceCount} active session(s)`);
				updateCollaborationIndicator();
			} catch (error) {
				presenceConnected = false;
				console.warn(`[Presence] Heartbeat failed for ${nextDocumentId}:`, error);
				updateCollaborationIndicator();
			}
		};

		console.log(`[Presence] Starting backend heartbeat for ${nextDocumentId}`);
		await heartbeat();
		presenceHeartbeatTimer = window.setInterval(() => {
			void heartbeat();
		}, 5000);
	}

	async function initializeCollaboration(nextDocumentId: string): Promise<ProviderInstance> {
		const wsUrl = await resolveRealtimeWsUrl();
		const token = authSignal.token;
		if (!token) {
			throw new Error('Missing access token');
		}

		const instance = await yjsProvider.createProvider({
			wsUrl,
			documentId: nextDocumentId,
			userId: authSignal.user?.id ?? 'unknown',
			token
		});

		if (instance.error) {
			console.warn('[Collaboration] Falling back to local Y.js mode:', instance.error);
		}

		return instance;
	}

	async function startCollaboration(nextDocumentId: string, reason: 'presence' | 'editing') {
		const now = Date.now();
		if (
			isInitializingCollaboration ||
			(collaboration && !collaborationError) ||
			(now - lastCollaborationAttemptAt < 10000 && reason !== 'presence')
		) {
			console.log(
				`[Collaboration] Start skipped for ${nextDocumentId}: initializing=${isInitializingCollaboration}, existing=${Boolean(collaboration && !collaborationError)}, recent=${now - lastCollaborationAttemptAt < 10000 && reason !== 'presence'}`
			);
			return;
		}

		if (collaborationError && collaboration) {
			yjsProvider.destroyProvider(nextDocumentId);
			collaboration = null;
			clearCollaborationListeners();
		}

		isInitializingCollaboration = true;
		lastCollaborationAttemptAt = now;
		try {
			console.log(`[Collaboration] Starting Yjs for ${nextDocumentId} via ${reason}`);
			const collaborationInstance = await initializeCollaboration(nextDocumentId);
			if (collaborationInstance.error || !collaborationInstance.provider) {
				collaboration = null;
				collaborationError = collaborationInstance.error || '协作连接失败';
				isYjsConnected = false;
				console.warn(
					`[Collaboration] Yjs unavailable for ${nextDocumentId}: ${collaborationError}`
				);
				updateCollaborationIndicator();
				yjsProvider.destroyProvider(nextDocumentId);
				return;
			}

			collaboration = collaborationInstance;
			collaborationError = null;
			console.log(`[Collaboration] Yjs provider ready for ${nextDocumentId}`);
			attachCollaborationListeners(collaborationInstance);
		} catch (collaborationInitError) {
			console.error('[Collaboration] Failed to initialize realtime collaboration:', collaborationInitError);
			collaboration = null;
			collaborationError =
				collaborationInitError instanceof Error
					? collaborationInitError.message
					: 'Unknown collaboration error';
			isYjsConnected = false;
			updateCollaborationIndicator();
			clearCollaborationListeners();
		} finally {
			isInitializingCollaboration = false;
		}
	}

	function clearCollaborationListeners() {
		detachCollaborationListeners?.();
		detachCollaborationListeners = null;
	}

	function attachCollaborationListeners(instance: ProviderInstance) {
		clearCollaborationListeners();

		if (!instance.provider) {
			collaborationError = '协作连接已断开';
			isYjsConnected = false;
			updateCollaborationIndicator();
			return;
		}

		const syncPresence = (states?: Array<{ clientId: number }>) => {
			const peerCount =
				states?.length ??
				Array.from(instance.provider?.awareness?.getStates().values?.() ?? []).length;
			presenceCount = Math.max(peerCount, 1);
			updateCollaborationIndicator();
		};

		const handleStatus = ({ status }: { status: string }) => {
			if (status === 'connected') {
				collaborationError = null;
				isYjsConnected = true;
				console.log(`[Yjs] Connected for ${documentId}`);
				syncPresence();
				return;
			}

			if (status === 'disconnected') {
				collaborationError = '协作连接已断开';
				isYjsConnected = false;
				console.warn(`[Yjs] Disconnected for ${documentId}`);
				updateCollaborationIndicator();
			}
		};

		const handleAuthenticationFailed = ({ reason }: { reason: string }) => {
			collaborationError = reason || '协作鉴权失败';
			isYjsConnected = false;
			console.warn(`[Yjs] Authentication failed for ${documentId}: ${collaborationError}`);
			updateCollaborationIndicator();
			if (documentId) {
				yjsProvider.stopReconnects(documentId);
			}
		};

		const handleAwarenessChange = ({ states }: { states: Array<{ clientId: number }> }) => {
			syncPresence(states);
		};

		instance.provider.on('status', handleStatus);
		instance.provider.on('authenticationFailed', handleAuthenticationFailed);
		instance.provider.on('awarenessChange', handleAwarenessChange);
		detachCollaborationListeners = () => {
			instance.provider?.off('status', handleStatus);
			instance.provider?.off('authenticationFailed', handleAuthenticationFailed);
			instance.provider?.off('awarenessChange', handleAwarenessChange);
		};

		if (instance.error) {
			collaborationError = instance.error;
			isYjsConnected = false;
			updateCollaborationIndicator();
		} else if (instance.isConnected) {
			isYjsConnected = true;
			syncPresence();
		} else {
			isYjsConnected = false;
			updateCollaborationIndicator();
		}
	}

	async function saveContent(reason: 'manual' | 'auto' = 'manual'): Promise<boolean> {
		if (!documentId || isLoading || isSaving || !hasUnsavedChanges) {
			return !hasUnsavedChanges;
		}

		isSaving = true;
		try {
			await updateDocumentContent(documentId, normalizeManagedImagesForSave(content));
			lastSaved = new Date();
			hasUnsavedChanges = false;
			return true;
		} catch (error) {
			console.error('[Save] Failed to save content:', error);
			if (reason === 'manual') {
				toast.error(m.editor_save_failed());
			}
			return false;
		} finally {
			isSaving = false;
		}
	}

	async function handleSaveAndLeave() {
		const saved = await saveContent();
		if (!saved) {
			return;
		}
		await handleConfirmLeave();
	}

	async function handleImageTargetChange(nextTargetId: string) {
		if (!documentId || isUpdatingImageTarget || nextTargetId === preferredImageTargetId) {
			return;
		}

		isUpdatingImageTarget = true;
		try {
			const updated = await updateDocumentImageTarget(documentId, nextTargetId);
			preferredImageTargetId = updated.preferredImageTargetId;
			toast.success(m.editor_image_target_updated());
		} catch (error) {
			console.error('[Document] Failed to update image target:', error);
			toast.error(
				error instanceof Error && error.message.trim() !== ''
					? error.message
					: m.editor_image_target_update_failed()
			);
		} finally {
			isUpdatingImageTarget = false;
		}
	}

	// Load document content when ID becomes available
	$effect(() => {
		if (documentId && !authSignal.loading) {
			isLoading = true;
			const loadContent = async () => {
				try {
					console.log('[Load] Loading document for ID:', documentId);
					// Load document details (for title) and content in parallel
					const [details, data, configs] = await Promise.all([
						getDocumentDetails(documentId),
						getDocumentContent(documentId),
						getImageBedConfigs().catch((error) => {
							console.error('[Load] Failed to load image bed configs:', error);
							return [] as ImageBedConfig[];
						})
					]);

					if (details.myRole === 'viewer') {
						await goto(`/view/documents/${documentId}`);
						return;
					}

					const loadedContent = data.contentJson ?? EMPTY_DOC;
					imageBedConfigs = configs;
					content = await refreshSignedImageSources(loadedContent);
					// Use the title from the API
					title = details.title ?? '';
					manualExcerpt = details.manualExcerpt ?? '';
					myRole = details.myRole ?? 'owner';
					publicAccess = details.publicAccess ?? 'private';
					publicUrl = details.publicUrl ?? `/view/documents/${documentId}`;
					documentType = details.documentType ?? 'rich_text';
					preferredImageTargetId = details.preferredImageTargetId ?? 'managed-r2';
					hasUnsavedChanges = false;
					lastSaved = null;
					presenceCount = 0;
					presenceConnected = false;
					hasAttemptedPresence = false;
					collaborationError = null;
					isYjsConnected = false;
					clearPresenceSocket();
					clearCollaborationListeners();
					if (documentId) {
						yjsProvider.destroyProvider(documentId);
					}
					collaboration = null;
					updateCollaborationIndicator();
					console.log('[Load] Title loaded:', title);
					isLoading = false;

					void (async () => {
						try {
							presenceCount = await fetchCollaborationPresence(documentId);
							updateCollaborationIndicator();
							await connectPresenceSocket(documentId);
							await startCollaboration(documentId, 'presence');
						} catch (presenceError) {
							console.error('[Collaboration] Failed to fetch presence:', presenceError);
							presenceCount = 0;
							presenceConnected = false;
							hasAttemptedPresence = true;
							collaborationError = 'presence-disconnected';
							isYjsConnected = false;
							updateCollaborationIndicator();
						}
					})();
				} catch (error) {
					console.error('[Load] Failed to load document:', error);
					collaboration = null;
					collaborationIndicator = null;
					isYjsConnected = false;
					clearPresenceSocket();
					clearCollaborationListeners();
					toast.error(
						error instanceof Error && error.message.trim() !== ''
							? error.message
							: '加载文档失败'
					);
					goto('/workspace');
				} finally {
					if (isLoading) {
						isLoading = false;
					}
				}
			};
			loadContent();
		}
	});

	onDestroy(() => {
		clearPresenceSocket();
		clearCollaborationListeners();
		if (documentId) {
			yjsProvider.destroyProvider(documentId);
		}
	});

	$effect(() => {
		if (!browser) {
			return;
		}

		// 当前先从本地偏好读取自动保存策略，后续可以直接换成个人中心设置源。
		autoSaveEnabled = readAutoSaveEnabled();
		autoSaveIntervalSeconds = readAutoSaveIntervalSeconds();
	});

	$effect(() => {
		if (!browser || !documentId || isLoading || !autoSaveEnabled) {
			return;
		}

		// 自动保存只负责兜底落盘，不额外维护独立状态指示。
		const timer = window.setInterval(() => {
			if (!hasUnsavedChanges || isSaving) {
				return;
			}

			void saveContent('auto');
		}, autoSaveIntervalSeconds * 1000);

		return () => {
			window.clearInterval(timer);
		};
	});


	onMount(() => {
		const handleKeydown = (event: KeyboardEvent) => {
			const isSaveKey = (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 's';
			if (!isSaveKey) return;
			event.preventDefault();
			void saveContent();
		};

		const handleBeforeUnload = (event: BeforeUnloadEvent) => {
			if (!hasUnsavedChanges) {
				return;
			}

			event.preventDefault();
			event.returnValue = '';
		};

		window.addEventListener('keydown', handleKeydown);
		window.addEventListener('beforeunload', handleBeforeUnload);
		return () => {
			window.removeEventListener('keydown', handleKeydown);
			window.removeEventListener('beforeunload', handleBeforeUnload);
		};
	});
</script>

<svelte:head>
  <title>{m.page_title_edit_document({ title })}</title>
</svelte:head>

<div class="flex h-screen flex-col bg-white dark:bg-zinc-900">
	{#if documentId}
			<EditorTopBar
				{documentId}
				initialTitle={title}
				initialExcerpt={manualExcerpt}
				{documentType}
				{preferredImageTargetId}
				{availableImageTargets}
				{myRole}
				{publicAccess}
				{publicUrl}
				{collaborationIndicator}
				readOnly={false}
				showEditShortcut={false}
				{isUpdatingImageTarget}
				{isSaving}
				{lastSaved}
				{hasUnsavedChanges}
				onTitleChange={handleTitleChange}
				onManualExcerptChange={handleExcerptChange}
				onImageTargetChange={handleImageTargetChange}
				onPublicAccessChange={(nextPublicAccess, nextPublicURL) =>
					handlePublicAccessChange(nextPublicAccess, nextPublicURL)}
			/>
	{/if}

	<!-- Editor -->
	<main class="flex-1 overflow-hidden">
		<div class="h-full w-full">
			{#if browser && !isLoading}
				{#if documentType === 'table'}
					<div class="prose dark:prose-invert p-6">
						<p>{m.edit_document_editor_under_construction()}</p>
					</div>
				{:else}
					{#key collaboration?.doc ? `collab:${documentId}` : `local:${documentId}`}
						<Editor
							documentId={documentId!}
							{content}
							currentImageTargetLabel={currentImageTargetLabel}
							{collaboration}
							{isSaving}
							{hasUnsavedChanges}
							hydrateManagedContent={refreshSignedImageSources}
							onSave={saveContent}
							onContentChange={handleContentChange}
						/>
					{/key}
				{/if}
			{:else}
				<div class="prose dark:prose-invert">
					<p>{m.workspace_loading()}</p>
				</div>
			{/if}
		</div>
	</main>
</div>

<ConfirmDialog
	open={isLeaveConfirmOpen}
	title={m.common_unsaved_changes()}
	message={m.editor_unsaved_confirm_leave()}
	confirmText={m.common_save()}
	secondaryText={m.common_dont_save()}
	confirmVariant="primary"
	onCancel={handleCancelLeave}
	onSecondary={handleLeaveWithoutSave}
	onConfirm={handleSaveAndLeave}
/>
