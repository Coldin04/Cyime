<script lang="ts">
	import { updateDocumentTitle } from '$lib/api/workspace';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import UserMenuDropdown from '$lib/components/common/UserMenuDropdown.svelte';
	import EditorDocumentSettingsDialog from '$lib/components/editor/EditorDocumentSettingsDialog.svelte';
	import type { DocumentImageTargetOption } from '$lib/components/editor/documentImageTargets';

	// Icons
	import Home from '~icons/ph/house';
	import Search from '~icons/ph/magnifying-glass';
	import FileText from '~icons/ph/file-text';
	import PencilSimple from '~icons/ph/pencil-simple';
	import Check from '~icons/ph/check';
	import X from '~icons/ph/x';
	import User from '~icons/ph/user';
	import UsersThree from '~icons/ph/users-three';
	import WarningCircle from '~icons/ph/warning-circle';

let {
	documentId,
	initialTitle,
	initialExcerpt = '',
	documentType = 'rich_text',
	preferredImageTargetId,
	availableImageTargets,
	myRole = 'owner',
	publicAccess = 'private',
	publicUrl = '',
	collaborationIndicator = null,
	readOnly = false,
	showEditShortcut = false,
	editHref = '',
	isUpdatingImageTarget = false,
	isSaving,
	lastSaved,
	hasUnsavedChanges,
	onTitleChange,
	onManualExcerptChange,
	onImageTargetChange,
	onPublicAccessChange
}: {
	documentId: string;
	initialTitle: string;
	initialExcerpt?: string;
	documentType?: string;
	preferredImageTargetId: string;
	availableImageTargets: DocumentImageTargetOption[];
	myRole?: 'owner' | 'collaborator' | 'editor' | 'viewer' | string;
	publicAccess?: 'private' | 'authenticated' | 'public' | string;
	publicUrl?: string;
	collaborationIndicator?: { kind: 'offline' | 'single' | 'multi'; label: string } | null;
	readOnly?: boolean;
	showEditShortcut?: boolean;
	editHref?: string;
	isUpdatingImageTarget?: boolean;
	isSaving: boolean;
	lastSaved: Date | null;
	hasUnsavedChanges: boolean;
	onTitleChange?: (title: string) => void;
	onManualExcerptChange?: (excerpt: string) => void;
	onImageTargetChange?: (targetId: string) => void | Promise<unknown>;
	onPublicAccessChange?: (publicAccess: string, publicUrl: string) => void;
} = $props();

let title = $state('');
let excerpt = $state('');
const canEditDocumentMeta = $derived(myRole === 'owner' || myRole === 'collaborator');
const canOpenDocumentSettings = $derived(canEditDocumentMeta);

	// Title editing state
	let isEditingTitle = $state(false);
	let editingTitle = $state('');
	let titleInput: HTMLInputElement | null = $state(null);

	$effect(() => {
		// When the initial title from the parent changes (e.g., on new doc load),
		// update the component's internal title state.
	if (initialTitle !== title) {
		title = initialTitle;
	}
});

$effect(() => {
	if (initialExcerpt !== excerpt) {
		excerpt = initialExcerpt;
	}
});

	async function startEditingTitle() {
		if (readOnly || !canEditDocumentMeta) return;
		editingTitle = title;
		isEditingTitle = true;
		// Focus the input after render
		setTimeout(() => titleInput?.focus(), 0);
	}

	async function saveTitle() {
		if (!editingTitle.trim() || editingTitle === title) {
			isEditingTitle = false;
			return;
		}

		try {
			await updateDocumentTitle(documentId, editingTitle.trim());
			title = editingTitle.trim();
			onTitleChange?.(title);
			toast.success(m.editor_topbar_title_updated());
		} catch (error) {
			console.error('Failed to update title:', error);
			toast.error(m.editor_topbar_title_update_failed());
		} finally {
			isEditingTitle = false;
		}
	}

	function cancelEditingTitle() {
		isEditingTitle = false;
	}

	function handleTitleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			saveTitle();
		} else if (e.key === 'Escape') {
			cancelEditingTitle();
		}
	}

</script>

<!-- Top Bar -->
<header
	class="z-30 flex h-16 shrink-0 items-center justify-between border-b border-black/10 bg-white/80 backdrop-blur-md dark:border-white/10 dark:bg-zinc-900/80"
>
	<!-- Left Controls -->
	<div class="flex items-center gap-2 px-4">
		<!-- Home Button -->
		<a
			href="/workspace"
			class="grid h-8 w-8 shrink-0 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
			title={m.topbar_back_to_workspace()}
		>
			<Home class="h-5 w-5" />
		</a>

		<!-- Divider -->
		<div class="h-5 w-px bg-zinc-200 dark:bg-zinc-700"></div>
	</div>

	<!-- Center: Title Section -->
	<div class="flex min-w-0 flex-1 items-center gap-2 px-0">
		<FileText class="h-5 w-5 shrink-0 text-zinc-400 self-center" />

		<div class="flex min-w-0 flex-col">
			{#if isEditingTitle}
				<div class="flex items-center gap-1">
					<input
						bind:this={titleInput}
						type="text"
						value={editingTitle}
						oninput={(e) => (editingTitle = e.currentTarget.value)}
						onkeydown={handleTitleKeydown}
						onblur={saveTitle}
						class="w-full max-w-xl bg-transparent text-sm text-zinc-900 placeholder-zinc-400 focus:outline-none dark:text-zinc-100 px-2 py-0"
						placeholder={m.document_name_placeholder()}
					/>
					<button
						onclick={saveTitle}
						class="grid h-4 w-4 place-content-center rounded text-green-600 transition-colors hover:bg-green-100 dark:text-green-400 dark:hover:bg-green-900/30"
						title={m.editor_topbar_save_title()}
					>
						<Check class="h-4 w-4" />
					</button>
					<button
						onclick={cancelEditingTitle}
						class="grid h-4 w-4 place-content-center rounded text-red-600 transition-colors hover:bg-red-100 dark:text-red-400 dark:hover:bg-red-900/30"
						title={m.common_cancel()}
					>
						<X class="h-4 w-4" />
					</button>
				</div>
			{:else}
				{#if readOnly || !canEditDocumentMeta}
					<h1 class="truncate rounded bg-transparent px-2 text-sm text-zinc-900 dark:text-zinc-100" title={title}>
						{title}
					</h1>
				{:else}
					<button
						onclick={startEditingTitle}
						class="group flex min-w-0 items-center"
						title={m.editor_topbar_edit_title_tooltip()}
					>
						<h1
							class="truncate rounded bg-transparent px-2 text-sm text-zinc-900 placeholder-zinc-400 transition-colors group-hover:bg-zinc-100 dark:text-zinc-100 dark:group-hover:bg-zinc-800"
							title={title}
						>
							{title}
						</h1>
					</button>
				{/if}
			{/if}

			<div class="flex items-center gap-2 px-2 py-0 text-left leading-3">
				{#if collaborationIndicator}
					<div class="group relative flex items-center">
						<div
							class={`grid h-5 w-5 place-content-center ${
								collaborationIndicator.kind === 'offline'
									? 'text-amber-500 dark:text-amber-300'
									: collaborationIndicator.kind === 'multi'
										? 'text-emerald-600 dark:text-emerald-400'
										: 'text-zinc-400 dark:text-zinc-500'
							}`}
							title={collaborationIndicator.label}
							aria-label={collaborationIndicator.label}
						>
							{#if collaborationIndicator.kind === 'multi'}
								<UsersThree class="h-3.5 w-3.5" />
							{:else if collaborationIndicator.kind === 'offline'}
								<WarningCircle class="h-3.5 w-3.5" />
							{:else}
								<User class="h-3.5 w-3.5" />
							{/if}
						</div>
						<div
							class="pointer-events-none absolute left-0 top-7 z-40 w-max max-w-xs rounded-md bg-zinc-900 px-2 py-1 text-[11px] text-white opacity-0 shadow-lg transition-opacity group-hover:opacity-100 dark:bg-zinc-100 dark:text-zinc-900"
						>
							{collaborationIndicator.label}
						</div>
					</div>
				{/if}

				{#if readOnly}
					<span class="text-xs text-zinc-400 py-0">{m.editor_topbar_read_only()}</span>
				{:else if isSaving}
					<span class="text-xs text-zinc-400 py-0">{m.editor_topbar_saving()}</span>
				{:else if hasUnsavedChanges}
					<span class="text-xs text-zinc-400 py-0">{m.editor_topbar_unsaved()}</span>
				{:else if lastSaved}
					<span class="text-xs text-zinc-400 py-0">
						{m.editor_topbar_saved_at({ time: lastSaved.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) })}
					</span>
				{:else}
					<span class="text-xs text-zinc-400 py-0">{m.editor_topbar_pending_changes()}</span>
				{/if}
			</div>
		</div>
	</div>

	<!-- Right Controls -->
	<div class="flex items-center gap-4 pr-4">
		{#if !readOnly && canOpenDocumentSettings}
			<EditorDocumentSettingsDialog
				{documentId}
				documentTitle={title}
				documentManualExcerpt={excerpt}
				{documentType}
				currentTargetId={preferredImageTargetId}
				options={availableImageTargets}
				canEditBasic={canEditDocumentMeta}
				canManageMembers={myRole === 'owner' || myRole === 'collaborator'}
				canEditImageSettings={canEditDocumentMeta}
				canManagePublic={myRole === 'owner'}
				{publicAccess}
				{publicUrl}
				isUpdating={isUpdatingImageTarget}
				onSelect={(targetId) => onImageTargetChange?.(targetId)}
				onTitleChange={(nextTitle) => {
					title = nextTitle;
					onTitleChange?.(nextTitle);
				}}
				onManualExcerptChange={(nextExcerpt) => {
					excerpt = nextExcerpt;
					onManualExcerptChange?.(nextExcerpt);
				}}
				onPublicAccessChange={(nextPublicAccess, nextPublicURL) =>
					onPublicAccessChange?.(nextPublicAccess, nextPublicURL)}
			/>
		{/if}
		{#if showEditShortcut && editHref}
			<a
				href={editHref}
				class="grid h-8 w-8 shrink-0 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
				title={m.editor_topbar_open_editor()}
				aria-label={m.editor_topbar_open_editor()}
			>
				<PencilSimple class="h-5 w-5" />
			</a>
		{/if}
		<button
			class="grid h-8 w-8 shrink-0 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
			title={m.common_search_placeholder()}
		>
			<Search class="h-5 w-5" />
		</button>
		<UserMenuDropdown profileHref="/user" trashHref="/workspace/trash" showTrash={true} />
	</div>
</header>
