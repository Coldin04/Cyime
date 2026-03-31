<script lang="ts">
	import type { JSONContent } from '@tiptap/core';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import { auth } from '$lib/stores/auth';
	import Editor from '$lib/components/editor/Editor.svelte';
	import EditorTopBar from '$lib/components/editor/EditorTopBar.svelte';
	import {
		getDocumentContent,
		getPublicDocumentContent,
		resolveAssetReadURLs
	} from '$lib/api/editor';
	import { getDocumentDetails, getPublicDocumentDetails } from '$lib/api/workspace';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';

	const EMPTY_DOC: JSONContent = {
		type: 'doc',
		content: [{ type: 'paragraph' }]
	};

	let title = $state('');
	let manualExcerpt = $state('');
	let myRole = $state<'owner' | 'collaborator' | 'editor' | 'viewer' | string>('viewer');
	let content = $state<JSONContent>(EMPTY_DOC);
	let documentType = $state<'rich_text' | 'table' | string>('rich_text');
	let preferredImageTargetId = $state('managed-r2');
	let isLoading = $state(true);
	let requireSignIn = $state(false);

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
			console.error('[View] Failed to resolve image URLs:', error);
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
				continue;
			}
			attrs.src = resolvedURL;
			node.attrs = attrs;
		}

		return cloned;
	}

	let pageSignal = $state(get(page));
	page.subscribe((p) => (pageSignal = p));
	const documentId = $derived(pageSignal.params?.id);
	const canOpenEditor = $derived(myRole !== 'viewer');

	$effect(() => {
		if (!documentId || $auth.loading) {
			return;
		}

		isLoading = true;
		const loadContent = async () => {
			try {
				requireSignIn = false;
				if ($auth.token) {
					const [details, data] = await Promise.all([
						getDocumentDetails(documentId),
						getDocumentContent(documentId)
					]);

					title = details.title ?? '';
					manualExcerpt = details.manualExcerpt ?? '';
					myRole = details.myRole ?? 'owner';
					documentType = details.documentType ?? 'rich_text';
					preferredImageTargetId = details.preferredImageTargetId ?? 'managed-r2';
					content = await refreshSignedImageSources(data.contentJson ?? EMPTY_DOC);
					return;
				}

				const [details, data] = await Promise.all([
					getPublicDocumentDetails(documentId),
					getPublicDocumentContent(documentId)
				]);

				title = details.title ?? '';
				manualExcerpt = details.manualExcerpt ?? '';
				myRole = 'viewer';
				documentType = details.documentType ?? 'rich_text';
				preferredImageTargetId = details.preferredImageTargetId ?? 'managed-r2';
				content = data.contentJson ?? EMPTY_DOC;
			} catch (error) {
				console.error('[View] Failed to load document:', error);
				if (
					!$auth.token &&
					error instanceof Error &&
					(error as Error & { status?: number }).status === 401
				) {
					requireSignIn = true;
					return;
				}
				toast.error(
					error instanceof Error && error.message.trim() !== ''
						? error.message
						: '加载文档失败'
				);
				goto('/workspace');
			} finally {
				isLoading = false;
			}
		};
		loadContent();
	});
</script>

<svelte:head>
	<title>{m.page_title_view_document({ title })}</title>
</svelte:head>

<div class="flex h-screen flex-col bg-white dark:bg-zinc-900">
	{#if documentId}
		<EditorTopBar
			{documentId}
			initialTitle={title}
			initialExcerpt={manualExcerpt}
			{documentType}
			{preferredImageTargetId}
			availableImageTargets={[]}
			readOnly={true}
			showEditShortcut={canOpenEditor}
			editHref={`/edit/documents/${documentId}`}
			isSaving={false}
			lastSaved={null}
			hasUnsavedChanges={false}
		/>
	{/if}

	<main class="flex-1 overflow-hidden">
		<div class="h-full w-full">
			{#if requireSignIn}
				<div class="flex h-full items-center justify-center px-6">
					<div class="max-w-md text-center">
						<p class="text-sm text-zinc-600 dark:text-zinc-300">{m.viewer_sign_in_required()}</p>
						<a
							href="/login"
							class="mt-4 inline-flex items-center rounded-md bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
						>
							{m.viewer_sign_in_action()}
						</a>
					</div>
				</div>
			{:else if browser && !isLoading}
				{#if documentType === 'table'}
					<div class="prose dark:prose-invert p-6">
						<p>{m.edit_document_editor_under_construction()}</p>
					</div>
				{:else}
					<Editor documentId={documentId!} {content} readOnly={true} isSaving={false} hasUnsavedChanges={false} />
				{/if}
			{:else}
				<div class="prose dark:prose-invert">
					<p>{m.workspace_loading()}</p>
				</div>
			{/if}
		</div>
	</main>
</div>
