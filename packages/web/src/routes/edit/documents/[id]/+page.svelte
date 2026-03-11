<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { beforeNavigate, goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import Editor from '$lib/components/editor/Editor.svelte';
	import EditorTopBar from '$lib/components/editor/EditorTopBar.svelte';
	import ConfirmDialog from '$lib/components/common/ConfirmDialog.svelte';
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
	let isLeaveConfirmOpen = $state(false);
	let pendingNavigationUrl = $state<string | null>(null);
	let bypassLeaveGuard = $state(false);

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

	function handleContentChange(newContent: string) {
		if (isLoading) return;
		hasUnsavedChanges = true;
		content = newContent;
	}

	function handleTitleChange(newTitle: string) {
		title = newTitle;
	}

	async function saveContent(): Promise<boolean> {
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
			toast.error(m.editor_save_failed());
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
					<Editor
						{content}
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
