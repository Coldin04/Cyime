<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import Editor from '$lib/components/editor/Editor.svelte';
	import EditorTopBar from '$lib/components/editor/EditorTopBar.svelte';
	import { getDocumentContent, updateDocumentContent } from '$lib/api/editor';
	import { getDocumentDetails } from '$lib/api/workspace';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';

	let title = $state('');
	let content = $state('');
	let documentType = $state<'rich_text' | 'table' | string>('rich_text');
	let isSaving = $state(false);
	let lastSaved = $state<Date | null>(null);
	let hasUnsavedChanges = $state(false);
	let isLoading = $state(true);
	let saveTimer: ReturnType<typeof setTimeout> | null = null;

	// Manually bridge the SvelteKit `page` store to a Svelte 5 signal
	// since this environment is in runes-mode but likely on an older Svelte 5 version.
	let pageSignal = $state(get(page));
	page.subscribe((p) => (pageSignal = p));
	const documentId = $derived(pageSignal.params?.id);

	// Auto-save function with debounce
	function scheduleSave(newContent: string) {
		if (saveTimer) {
			clearTimeout(saveTimer);
		}

		saveTimer = setTimeout(async () => {
			await saveContent(newContent);
		}, 1000); // 1 second debounce
	}

	async function saveContent(newContent: string) {
		if (!hasUnsavedChanges) {
			console.log('[Save] No unsaved changes, skipping');
			return;
		}

		console.log('[Save] Saving content, length:', newContent?.length, 'documentId:', documentId);
		isSaving = true;
		try {
			const result = await updateDocumentContent(documentId!, newContent);
			console.log('[Save] Save successful:', result);
			lastSaved = new Date();
			hasUnsavedChanges = false;
			// 自动保存不弹窗
		} catch (error) {
			console.error('[Save] Failed to save content:', error);
			toast.error(m.folder_delete_failed());
		} finally {
			isSaving = false;
		}
	}

	function handleContentChange(newContent: string) {
		// Skip if currently loading content
		if (isLoading) return;

		hasUnsavedChanges = true;
		content = newContent;
		scheduleSave(newContent);
	}

	function handleTitleChange(newTitle: string) {
		title = newTitle;
	}

	// Load document content when ID becomes available
	$effect(() => {
		if (documentId) {
			isLoading = true;
			const loadContent = async () => {
				try {
					console.log('[Load] Loading document for ID:', documentId);
					// Load document details (for title) and content in parallel
					const [details, data] = await Promise.all([
						getDocumentDetails(documentId),
						getDocumentContent(documentId)
					]);

					console.log('[Load] Content loaded, length:', data.content?.length);
					content = data.content;
					// Use the title from the API
					title = details.title ?? '';
					documentType = details.documentType ?? 'rich_text';
					console.log('[Load] Title loaded:', title);
					// Reset state for the new document
					hasUnsavedChanges = false;
					lastSaved = null;
					console.log('[Load] Content set, hasUnsavedChanges:', hasUnsavedChanges);
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

	// Cleanup timer on unmount
	onMount(() => {
		return () => {
			if (saveTimer) {
				clearTimeout(saveTimer);
			}
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
			{isSaving}
			{lastSaved}
			{hasUnsavedChanges}
			onTitleChange={handleTitleChange}
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
					<Editor {content} onContentChange={handleContentChange} />
				{/if}
			{:else}
				<div class="prose dark:prose-invert">
					<p>{m.workspace_loading()}</p>
				</div>
			{/if}
		</div>
	</main>
</div>
