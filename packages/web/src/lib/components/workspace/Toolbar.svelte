<script lang="ts">
	import Plus from '~icons/ph/plus';
	import DotsThreeVertical from '~icons/ph/dots-three-vertical';
	import FolderPlus from '~icons/ph/folder-plus';
	import Trash from '~icons/ph/trash';
	import X from '~icons/ph/x';
	import { createMarkdown } from '$lib/api/workspace';
	import { goto } from '$app/navigation';
	import Breadcrumb from './Breadcrumb.svelte';
	import { breadcrumbItems } from '$lib/stores/workspace';
	import * as m from '$paraglide/messages';

	const {
		bulkMode = false,
		selectedItemsCount = 0,
		currentFolderId,
		onToggleBulk,
		onBulkDelete,
		onCreateFolder,
		onNavigate
	}: {
		bulkMode?: boolean;
		selectedItemsCount?: number;
		currentFolderId: string | null;
		onToggleBulk: () => void;
		onBulkDelete: () => void;
		onCreateFolder: () => void;
		onNavigate: (id: string | null) => void;
	} = $props();

	let showMenu = $state(false);

	function toggleMenu() {
		showMenu = !showMenu;
	}

	async function handleCreateDocument() {
		try {
			const newDoc = await createMarkdown({
				title: m.edit_document_title(),
				content: '',
				folderId: currentFolderId
			});
			goto(`/edit/md/${newDoc.id}`);
		} catch (error) {
			console.error('创建文档失败:', error);
		}
	}
</script>

<div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4">
	<div class="min-w-0 truncate">
		<Breadcrumb onNavigate={onNavigate} items={$breadcrumbItems} />
	</div>

	<!-- Action Buttons -->
	<div class="relative flex flex-shrink-0 items-center">
		{#if bulkMode}
			<!-- Bulk Mode Actions -->
			<div class="flex items-center gap-2">
				<span class="text-sm text-zinc-600 dark:text-zinc-400">
					{m.toolbar_selected_count({ count: selectedItemsCount })}
				</span>
				<button
					onclick={onBulkDelete}
					class="inline-flex h-10 items-center gap-2 rounded-lg bg-red-500 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-red-600"
				>
					<Trash class="h-4 w-4" />
					<span class="hidden sm:inline">{m.common_delete()}</span>
				</button>
				<button
					onclick={onToggleBulk}
					class="inline-flex h-10 items-center justify-center rounded-lg border border-zinc-300 bg-white px-3 text-sm font-semibold text-zinc-700 shadow-sm transition-colors hover:bg-zinc-50 dark:border-zinc-600 dark:bg-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-700"
				>
					<X class="h-4 w-4" />
				</button>
			</div>
		{:else}
			<!-- Normal Mode -->
			<button
				onclick={handleCreateDocument}
				class="inline-flex h-10 items-center justify-center gap-2 rounded-l-lg bg-riptide-500 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800 disabled:opacity-50 sm:px-4"
			>
				<Plus class="h-4 w-4" />
				<span class="hidden sm:inline">{m.common_new_document()}</span>
			</button>
			<button
				onclick={toggleMenu}
				class="inline-flex h-10 w-10 items-center justify-center rounded-r-lg border-l border-riptide-400 bg-riptide-500 p-2 text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800"
				aria-label={m.common_more_options()}
			>
				<DotsThreeVertical class="h-5 w-5" />
			</button>

			{#if showMenu}
				<div
					class="absolute top-full right-0 z-10 mt-2 w-48 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-zinc-800 dark:ring-zinc-700"
					role="menu"
					aria-orientation="vertical"
					aria-labelledby="menu-button"
				>
					<button
						onclick={onToggleBulk}
						class="flex w-full items-center gap-3 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
						role="menuitem"
					>
						<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
							/>
						</svg>
						<span>{m.common_bulk_select()}</span>
					</button>
					<button
						onclick={() => {
							onCreateFolder();
							showMenu = false;
						}}
						class="flex w-full items-center gap-3 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
						role="menuitem"
					>
						<FolderPlus class="h-4 w-4" />
						<span>{m.common_new_folder()}</span>
					</button>
				</div>
			{/if}
		{/if}
	</div>
</div>

