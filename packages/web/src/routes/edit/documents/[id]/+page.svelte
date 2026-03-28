<script lang="ts">
	import { onMount } from 'svelte';
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
	import { getAssetReadURL, getDocumentContent, updateDocumentContent } from '$lib/api/editor';
	import { getDocumentDetails, updateDocumentImageTarget } from '$lib/api/workspace';
	import { getImageBedConfigs, type ImageBedConfig } from '$lib/api/user';
	import {
		getDocumentImageTargetLabel,
		getDocumentImageTargetOptions
	} from '$lib/components/editor/documentImageTargets';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';

	let title = $state('');
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
	let isLeaveConfirmOpen = $state(false);
	let pendingNavigationUrl = $state<string | null>(null);
	let bypassLeaveGuard = $state(false);
	let autoSaveEnabled = $state(defaultAutoSaveEnabled);
	let autoSaveIntervalSeconds = $state(defaultAutoSaveIntervalSeconds);
	const availableImageTargets = $derived(getDocumentImageTargetOptions(imageBedConfigs));
	const currentImageTargetLabel = $derived(
		getDocumentImageTargetLabel(preferredImageTargetId, availableImageTargets)
	);

	const assetPathPattern =
		/\/api\/v1\/media\/assets\/([0-9a-fA-F-]{36})\/content(?:\?.*)?$/;

	function extractAssetIdFromSrc(src: string): string | null {
		try {
			const parsed = new URL(src, browser ? window.location.origin : 'http://localhost');
			const match = parsed.pathname.match(assetPathPattern);
			return match?.[1] ?? null;
		} catch {
			const match = src.match(assetPathPattern);
			return match?.[1] ?? null;
		}
	}

	function cloneContentJson(value: JSONContent): JSONContent {
		return JSON.parse(JSON.stringify(value)) as JSONContent;
	}

	function collectImageNodes(value: unknown, nodes: Array<Record<string, unknown>>) {
		if (!value || typeof value !== 'object') {
			return;
		}
		const node = value as Record<string, unknown>;
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

	async function refreshSignedImageSources(input: JSONContent): Promise<JSONContent> {
		const cloned = cloneContentJson(input);
		const imageNodes: Array<Record<string, unknown>> = [];
		collectImageNodes(cloned, imageNodes);
		if (imageNodes.length === 0) {
			return cloned;
		}

		for (const node of imageNodes) {
			const attrs = (node.attrs ?? {}) as Record<string, unknown>;
			const src = typeof attrs.src === 'string' ? attrs.src : '';
			if (!src) continue;
			const assetId = extractAssetIdFromSrc(src);
			if (!assetId) continue;

			try {
				const resolved = await getAssetReadURL(assetId);
				attrs.src = resolved.url;
				node.attrs = attrs;
			} catch (error) {
				console.error('[Load] Failed to refresh image URL for asset:', assetId, error);
			}
		}

		return cloned;
	}

	// Manually bridge the SvelteKit `page` store to a Svelte 5 signal
	// since this environment is in runes-mode but likely on an older Svelte 5 version.
	let pageSignal = $state(get(page));
	page.subscribe((p) => (pageSignal = p));
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
	}

	function handleTitleChange(newTitle: string) {
		title = newTitle;
	}

	async function saveContent(reason: 'manual' | 'auto' = 'manual'): Promise<boolean> {
		if (!documentId || isLoading || isSaving || !hasUnsavedChanges) {
			return !hasUnsavedChanges;
		}

		isSaving = true;
		try {
			await updateDocumentContent(documentId, content);
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
		if (documentId) {
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

					const loadedContent = data.contentJson ?? EMPTY_DOC;
					imageBedConfigs = configs;
					content = await refreshSignedImageSources(loadedContent);
					// Use the title from the API
					title = details.title ?? '';
					documentType = details.documentType ?? 'rich_text';
					preferredImageTargetId = details.preferredImageTargetId ?? 'managed-r2';
					hasUnsavedChanges = false;
					lastSaved = null;
					console.log('[Load] Title loaded:', title);
				} catch (error) {
					console.error('[Load] Failed to load document:', error);
					toast.error(m.move_dialog_load_failed());
					goto('/workspace');
				} finally {
					isLoading = false;
				}
			};
			loadContent();
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
				{documentType}
				{preferredImageTargetId}
				{availableImageTargets}
				{isUpdatingImageTarget}
				{isSaving}
				{lastSaved}
				{hasUnsavedChanges}
				onTitleChange={handleTitleChange}
				onImageTargetChange={handleImageTargetChange}
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
					<Editor
						documentId={documentId!}
						{content}
						currentImageTargetLabel={currentImageTargetLabel}
						{isSaving}
						{hasUnsavedChanges}
						onSave={saveContent}
						onContentChange={handleContentChange}
					/>
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
