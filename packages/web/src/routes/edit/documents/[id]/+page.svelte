<script lang="ts">
	import { onDestroy, onMount, tick } from 'svelte';
	import type { JSONContent } from '@tiptap/core';
	import { browser } from '$app/environment';
	import { beforeNavigate, goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import type { ExportAction } from '$lib/export/exportActions';
	import Editor from '$lib/components/editor/Editor.svelte';
	import EditorTopBar from '$lib/components/editor/EditorTopBar.svelte';
	import ExportPrivateImagesDialog from '$lib/components/editor/ExportPrivateImagesDialog.svelte';
	import ConfirmDialog from '$lib/components/common/ConfirmDialog.svelte';
	import ModalDialog from '$lib/components/common/ModalDialog.svelte';
	import {
		defaultAutoSaveEnabled,
		defaultAutoSaveIntervalSeconds,
		readAutoSaveEnabled,
		readAutoSaveIntervalSeconds
	} from '$lib/components/editor/autoSave';
	import { auth } from '$lib/stores/auth';
	import { apiFetch } from '$lib/api';
	import { resolveApiUrl } from '$lib/config/api';
	import {
		getDocumentContent,
		pasteDocumentImage,
		resolveAssetReadURLs,
		updateDocumentContent
	} from '$lib/api/editor';
	import {
		createDocument,
		getDocumentDetails,
		updateDocumentImageTarget
	} from '$lib/api/workspace';
	import { getImageBedConfigs, type ImageBedConfig } from '$lib/api/user';
	import {
		getDocumentImageTargetLabel,
		getDocumentImageTargetOptions
	} from '$lib/components/editor/documentImageTargets';
	import {
		buildExportAssetFilename,
		collectImageNodes,
		collectManagedImages,
		cloneContentJson,
		getManagedAssetId,
		inferExportAssetMimeType,
		normalizeManagedImagesForSave,
		replaceManagedImagesWithPublicURLs
	} from '$lib/export/managedImages';
	import {
		ExportCopyError,
		exportHtmlDocument,
		exportPdfDocument,
		inlineManagedImagesAsDataURLs,
		runExportAction
	} from '$lib/export/exportPrivateImages';
	import { exportActionRequiresPublicImageURLs } from '$lib/export/exportActions';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';

	let title = $state('');
	let manualExcerpt = $state('');
	let myRole = $state<'owner' | 'collaborator' | 'editor' | 'viewer' | string>('owner');
	let publicAccess = $state<'private' | 'authenticated' | 'public' | string>('private');
	let publicUrl = $state('');
	let folderId = $state<string | null>(null);
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
	let documentLoadSequence = 0;
	let isLeaveConfirmOpen = $state(false);
	let pendingNavigationUrl = $state<string | null>(null);
	let bypassLeaveGuard = $state(false);
	let isExportPrivateImagesDialogOpen = $state(false);
	let isPreparingExport = $state(false);
	let pendingExportAction = $state<ExportAction | null>(null);
	let exportTargetId = $state('');
	let manualCopyContent = $state('');
	let manualCopyTitle = $state('');
	let autoSaveEnabled = $state(defaultAutoSaveEnabled);
	let autoSaveIntervalSeconds = $state(defaultAutoSaveIntervalSeconds);
	let editorContentOverride = $state<{ token: number; content: JSONContent } | null>(null);
	type SaveReason = 'manual' | 'auto' | 'leave' | 'export';
	type EditorOverrideWaiter = {
		expectedSerializedContent: string;
		resolve: (value: boolean) => void;
		timer: number;
	};
	let pageSignal = $state(get(page));
	const unsubscribePage = page.subscribe((p) => (pageSignal = p));
	let authSignal = $state(get(auth));
	const unsubscribeAuth = auth.subscribe((state) => (authSignal = state));
	const documentId = $derived(pageSignal.params?.id);
	let editorContentOverrideWaiter: EditorOverrideWaiter | null = null;
	let nextEditorContentOverrideToken = 0;
	const availableImageTargets = $derived(getDocumentImageTargetOptions(imageBedConfigs));
	const exportImageTargetOptions = $derived(
		availableImageTargets.filter((option) => option.id !== 'managed-r2')
	);
	const currentImageTargetLabel = $derived(
		getDocumentImageTargetLabel(preferredImageTargetId, availableImageTargets)
	);

	type ImageNodeRecord = Record<string, unknown> & {
		attrs?: Record<string, unknown>;
	};

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

	function serializeComparableContent(input: JSONContent): string {
		return JSON.stringify(normalizeManagedImagesForSave(input));
	}

	function logSaveDebug(event: string, details: Record<string, unknown> = {}) {
		console.debug('[SaveDebug]', event, {
			at: new Date().toISOString(),
			documentId,
			hasUnsavedChanges,
			isSaving,
			...details
		});
	}

	function createExportCopyTitle(value: string): string {
		const trimmed = value.trim();
		return trimmed === '' ? 'Untitled Export' : `${trimmed} (Export)`;
	}

	async function performExport(action: ExportAction, exportContent: JSONContent) {
		try {
			if (action === 'download-pdf') {
				const printContent = await inlineManagedImagesAsDataURLs(exportContent, (assetId) =>
					resolveApiUrl(`/api/v1/media/assets/${assetId}/content`)
				);
				const html = await exportHtmlDocument({
					title: title.trim() || 'Cyime Export',
					contentJson: printContent,
					colorMode: 'light'
				});
				await exportPdfDocument({
					title: title.trim() || 'Cyime Export',
					html
				});
				return;
			}

			const result = await runExportAction(action, {
				title,
				contentJson: exportContent
			});
			if (action === 'copy-markdown' && result === 'copied') {
				toast.success(m.editor_export_markdown_copied());
				return;
			}
			if (action === 'copy-bbcode' && result === 'copied') {
				toast.success(m.editor_export_bbcode_copied());
			}
		} catch (error) {
			console.error('[Export] Failed to export document:', error);
			if (error instanceof ExportCopyError && error.message === 'copy_markdown_failed') {
				manualCopyTitle = m.editor_export_copy_markdown();
				manualCopyContent = error.content;
				toast.error(m.editor_export_markdown_copy_failed());
				return;
			}
			if (error instanceof ExportCopyError && error.message === 'copy_bbcode_failed') {
				manualCopyTitle = m.editor_export_copy_bbcode();
				manualCopyContent = error.content;
				toast.error(m.editor_export_bbcode_copy_failed());
				return;
			}
			if (error instanceof Error && error.message === 'copy_markdown_failed') {
				toast.error(m.editor_export_markdown_copy_failed());
				return;
			}
			if (error instanceof Error && error.message === 'copy_bbcode_failed') {
				toast.error(m.editor_export_bbcode_copy_failed());
				return;
			}
			toast.error(m.editor_export_failed());
		}
	}

	function closeManualCopyDialog() {
		manualCopyContent = '';
		manualCopyTitle = '';
	}

	async function retryManualCopy() {
		try {
			await navigator.clipboard.writeText(manualCopyContent);
			toast.success(m.editor_export_manual_copy_success());
			closeManualCopyDialog();
		} catch {
			toast.error(m.editor_export_manual_copy_failed());
		}
	}

	async function prepareExportContentWithPublicImages(targetId: string): Promise<JSONContent> {
		if (!documentId) {
			throw new Error('Missing document id');
		}

		const contentSnapshot = cloneContentJson(content);
		const managedImages = collectManagedImages(contentSnapshot);
		if (managedImages.length === 0) {
			return contentSnapshot;
		}

		toast.loading(m.editor_export_prepare_images({ current: 0, total: managedImages.length }), {
			id: 'export-private-images',
			duration: Infinity
		});

		try {
			const publicURLByAssetID = new Map<string, string>();

			for (let index = 0; index < managedImages.length; index += 1) {
				const item = managedImages[index];
				toast.loading(m.editor_export_prepare_images({ current: index + 1, total: managedImages.length }), {
					id: 'export-private-images',
					duration: Infinity
				});

				const response = await apiFetch(`/api/v1/media/assets/${item.assetId}/content`);
				if (!response.ok) {
					throw new Error(`Failed to fetch private image ${item.assetId}`);
				}

				const blob = await response.blob();
				const mimeType = inferExportAssetMimeType(
					blob.type || response.headers.get('content-type') || '',
					item.title ?? item.alt,
					item.src
				);
				const file = new File(
					[blob],
					buildExportAssetFilename(item.assetId, mimeType, item.title ?? item.alt),
					{ type: mimeType }
				);
				const uploaded = await pasteDocumentImage(documentId, file, { targetId });
				publicURLByAssetID.set(item.assetId, uploaded.url);
			}

			return replaceManagedImagesWithPublicURLs(contentSnapshot, publicURLByAssetID);
		} finally {
			toast.dismiss('export-private-images');
		}
	}

	function resolveExportErrorMessage(error: unknown): string {
		const apiError = error as { code?: string; status?: number; message?: string } | undefined;
		const message = error instanceof Error ? error.message : (apiError?.message ?? '');
		if (
			apiError?.code === 'DOCUMENT_IMAGE_PROVIDER_UPLOAD_FAILED' &&
			/timeout|awaiting response headers|deadline exceeded/i.test(message)
		) {
			return m.editor_export_image_bed_timeout();
		}
		if (apiError?.code === 'DOCUMENT_IMAGE_PROVIDER_UPLOAD_FAILED') {
			return m.editor_export_image_bed_upload_failed();
		}
		return message.trim() !== '' ? message : m.editor_export_failed();
	}

	function closeExportPrivateImagesDialog() {
		if (isPreparingExport) {
			return;
		}
		isExportPrivateImagesDialogOpen = false;
		pendingExportAction = null;
		exportTargetId = '';
	}

	async function finalizeExportWithProcessedContent(exportContent: JSONContent) {
		if (!pendingExportAction) {
			return;
		}
		const action = pendingExportAction;
		isExportPrivateImagesDialogOpen = false;
		pendingExportAction = null;
		exportTargetId = '';
		await performExport(action, exportContent);
	}

	async function handleExportWithSaveAs() {
		if (!pendingExportAction || !documentId || !exportTargetId || isPreparingExport) {
			return;
		}

		isPreparingExport = true;
		try {
			const exportContent = await prepareExportContentWithPublicImages(exportTargetId);
			await createDocument({
				title: createExportCopyTitle(title),
				contentJson: normalizeManagedImagesForSave(exportContent) as { [key: string]: unknown },
				folderId,
				documentType: documentType === 'table' ? 'table' : 'rich_text',
				preferredImageTargetId: exportTargetId
			});
			await finalizeExportWithProcessedContent(exportContent);
		} catch (error) {
			console.error('[Export] Failed to create export copy:', error);
			toast.error(resolveExportErrorMessage(error));
		} finally {
			isPreparingExport = false;
		}
	}

	async function handleExportWithReplace() {
		if (!pendingExportAction || !documentId || !exportTargetId || isPreparingExport) {
			return;
		}

		isPreparingExport = true;
		try {
			const exportContent = await prepareExportContentWithPublicImages(exportTargetId);
			const applied = await applyEditorContentOverride(exportContent);
			if (!applied) {
				throw new Error('Failed to apply export content into the editor');
			}

			const saved = await requestDocumentSave('export');
			if (!saved) {
				throw new Error('Failed to persist export content');
			}

			const targetResult = await updateDocumentImageTarget(documentId, exportTargetId);
			preferredImageTargetId = targetResult.preferredImageTargetId;
			await finalizeExportWithProcessedContent(exportContent);
		} catch (error) {
			console.error('[Export] Failed to replace private images for export:', error);
			toast.error(resolveExportErrorMessage(error));
		} finally {
			isPreparingExport = false;
		}
	}

	async function handleExportAction(action: ExportAction) {
		if (hasUnsavedChanges || isSaving) {
			const saved = await requestDocumentSave('export');
			if (!saved) {
				toast.error(m.editor_save_failed());
				return;
			}
		}

		const managedImages = collectManagedImages(content);
		if (!exportActionRequiresPublicImageURLs(action) || managedImages.length === 0) {
			await performExport(action, content);
			return;
		}

		if (exportImageTargetOptions.length === 0) {
			toast.error(m.editor_export_private_images_config_required());
			return;
		}

		const preferredTarget = exportImageTargetOptions.find((option) => option.id === preferredImageTargetId);
		exportTargetId = preferredTarget?.id ?? exportImageTargetOptions[0].id;
		pendingExportAction = action;
		isExportPrivateImagesDialogOpen = true;
	}

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
		if (serializeComparableContent(content) === serializeComparableContent(newContent)) {
			return;
		}
		content = newContent;
		if (
			editorContentOverrideWaiter &&
			serializeComparableContent(newContent) === editorContentOverrideWaiter.expectedSerializedContent
		) {
			window.clearTimeout(editorContentOverrideWaiter.timer);
			editorContentOverrideWaiter.resolve(true);
			editorContentOverrideWaiter = null;
		}
		hasUnsavedChanges = true;
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

	function settleEditorContentOverrideWaiter(value: boolean) {
		if (!editorContentOverrideWaiter) {
			return;
		}

		window.clearTimeout(editorContentOverrideWaiter.timer);
		editorContentOverrideWaiter.resolve(value);
		editorContentOverrideWaiter = null;
	}

	async function applyEditorContentOverride(nextContent: JSONContent): Promise<boolean> {
		if (serializeComparableContent(content) === serializeComparableContent(nextContent)) {
			return true;
		}

		settleEditorContentOverrideWaiter(false);

		const result = new Promise<boolean>((resolve) => {
			const timer = window.setTimeout(() => {
				editorContentOverrideWaiter = null;
				resolve(false);
			}, 5000);
			editorContentOverrideWaiter = {
				expectedSerializedContent: serializeComparableContent(nextContent),
				resolve,
				timer
			};
		});

		const token = ++nextEditorContentOverrideToken;
		editorContentOverride = { token, content: nextContent };
		await tick();
		window.setTimeout(() => {
			if (
				editorContentOverrideWaiter &&
				editorContentOverrideWaiter.expectedSerializedContent === serializeComparableContent(nextContent)
			) {
				content = nextContent;
				hasUnsavedChanges = true;
				settleEditorContentOverrideWaiter(true);
			}
		}, 0);
		return result;
	}

	async function saveLocalContent(reason: SaveReason = 'manual'): Promise<boolean> {
		if (!documentId || isLoading || isSaving || !hasUnsavedChanges) {
			return !hasUnsavedChanges;
		}

		isSaving = true;
		logSaveDebug('local-save-start', { reason });
		try {
			const contentSnapshot = normalizeManagedImagesForSave(content);
			const serializedSnapshot = JSON.stringify(contentSnapshot);
			await updateDocumentContent(documentId, contentSnapshot);
			lastSaved = new Date();
			hasUnsavedChanges = serializeComparableContent(content) !== serializedSnapshot;
			logSaveDebug('local-save-success', { reason });
			return !hasUnsavedChanges;
		} catch (error) {
			console.error('[Save] Failed to save content:', error);
			logSaveDebug('local-save-failed', { reason, error: error instanceof Error ? error.message : String(error) });
			if (reason === 'manual') {
				toast.error(m.editor_save_failed());
			}
			return false;
		} finally {
			isSaving = false;
		}
	}

	async function requestDocumentSave(reason: SaveReason = 'manual'): Promise<boolean> {
		return saveLocalContent(reason);
	}

	async function handleSaveAndLeave() {
		const saved = await requestDocumentSave('leave');
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
			const targetDocumentId = documentId;
			const loadSequence = ++documentLoadSequence;
			isLoading = true;

			const isCurrentLoad = () => loadSequence === documentLoadSequence && documentId === targetDocumentId;

			const loadContent = async () => {
				try {
					console.log('[Load] Loading document for ID:', targetDocumentId);
					// Load document details (for title) and content in parallel
					const [details, data, configs] = await Promise.all([
						getDocumentDetails(targetDocumentId),
						getDocumentContent(targetDocumentId),
						getImageBedConfigs().catch((error) => {
							console.error('[Load] Failed to load image bed configs:', error);
							return [] as ImageBedConfig[];
						})
					]);

					if (!isCurrentLoad()) {
						return;
					}

					if (details.myRole !== 'owner') {
						await goto(`/view/documents/${targetDocumentId}`);
						return;
					}

					const loadedContent = data.contentJson ?? EMPTY_DOC;
					const hydratedContent = await refreshSignedImageSources(loadedContent);
					if (!isCurrentLoad()) {
						return;
					}

					imageBedConfigs = configs;
					content = hydratedContent;
					// Use the title from the API
					title = details.title ?? '';
					manualExcerpt = details.manualExcerpt ?? '';
					folderId = details.folderId ?? null;
					myRole = details.myRole ?? 'owner';
					publicAccess = details.publicAccess ?? 'private';
					publicUrl = details.publicUrl ?? `/view/documents/${targetDocumentId}`;
					documentType = details.documentType ?? 'rich_text';
					preferredImageTargetId = details.preferredImageTargetId ?? 'managed-r2';
					hasUnsavedChanges = false;
					lastSaved = null;
					isSaving = false;
					console.log('[Load] Title loaded:', title);
					isLoading = false;
				} catch (error) {
					if (!isCurrentLoad()) {
						return;
					}
					console.error('[Load] Failed to load document:', error);
					toast.error(
						error instanceof Error && error.message.trim() !== ''
							? error.message
							: '加载文档失败'
					);
					goto('/workspace');
				} finally {
					if (isCurrentLoad() && isLoading) {
						isLoading = false;
					}
				}
			};
			loadContent();
		}
	});

	onDestroy(() => {
		unsubscribePage();
		unsubscribeAuth();
		settleEditorContentOverrideWaiter(false);
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

		const timer = window.setInterval(() => {
			if (!hasUnsavedChanges || isSaving) {
				return;
			}

			void requestDocumentSave('auto');
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
			void requestDocumentSave('manual');
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
					<Editor
						documentId={documentId!}
						{content}
						externalContentOverride={editorContentOverride}
						currentImageTargetId={preferredImageTargetId}
						currentImageTargetLabel={currentImageTargetLabel}
						imageTargetOptions={availableImageTargets}
						{isUpdatingImageTarget}
						{isSaving}
						{hasUnsavedChanges}
						onImageTargetChange={handleImageTargetChange}
						onSave={() => requestDocumentSave('manual')}
						onExportAction={handleExportAction}
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

<ExportPrivateImagesDialog
	open={isExportPrivateImagesDialogOpen}
	imageCount={collectManagedImages(content).length}
	targetOptions={exportImageTargetOptions}
	selectedTargetId={exportTargetId}
	busy={isPreparingExport}
	onTargetChange={(nextTargetId) => {
		exportTargetId = nextTargetId;
	}}
	onCancel={closeExportPrivateImagesDialog}
	onSaveAs={handleExportWithSaveAs}
	onReplace={handleExportWithReplace}
/>

<ModalDialog
	open={manualCopyContent.trim() !== ''}
	title={manualCopyTitle || m.editor_export_manual_copy_title()}
	maxWidthClass="max-w-3xl"
	onClose={closeManualCopyDialog}
>
	<div class="space-y-4">
		<div>
			<h2 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
				{m.editor_export_manual_copy_title()}
			</h2>
			<p class="mt-1 text-sm leading-6 text-zinc-600 dark:text-zinc-300">
				{m.editor_export_manual_copy_description()}
			</p>
		</div>
		<textarea
			readonly
			spellcheck="false"
			value={manualCopyContent}
			class="h-72 w-full resize-none rounded-md border border-zinc-200 bg-zinc-50 p-3 font-mono text-xs leading-5 text-zinc-800 outline-none dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-100"
			onfocus={(event) => event.currentTarget.select()}
		></textarea>
		<div class="flex justify-end gap-2">
			<button
				type="button"
				class="inline-flex h-8 items-center rounded-md px-3 text-sm text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				onclick={closeManualCopyDialog}
			>
				{m.common_cancel()}
			</button>
			<button
				type="button"
				class="inline-flex h-8 items-center rounded-md bg-zinc-900 px-3 text-sm font-medium text-white transition-colors hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
				onclick={retryManualCopy}
			>
				{m.editor_export_manual_copy_action()}
			</button>
		</div>
	</div>
</ModalDialog>
