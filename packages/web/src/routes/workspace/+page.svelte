<script lang="ts">
	import Toolbar from '$lib/components/workspace/Toolbar.svelte';
	import ListHeader from '$lib/components/workspace/ListHeader.svelte';
	import FolderListItem from '$lib/components/workspace/FolderListItem.svelte';
	import MarkdownListItem from '$lib/components/workspace/MarkdownListItem.svelte';
	import FolderListItemSkeleton from '$lib/components/workspace/FolderListItemSkeleton.svelte';
	import MarkdownListItemSkeleton from '$lib/components/workspace/MarkdownListItemSkeleton.svelte';
	import NewFolderItem from '$lib/components/workspace/NewFolderItem.svelte';
	import { getFiles, getFolderAncestors, deleteFile, type FileItem } from '$lib/api/workspace';
	import { breadcrumbItems } from '$lib/stores/workspace';

	let items = $state<FileItem[]>([]);
	let selectedItems = $state<{ [key: string]: boolean }>({});
	let hasMore = $state(false);
	let sortBy = $state('updated_at');
	let order = $state('desc');
	let filterType = $state<'all' | 'folders' | 'markdowns'>('all');
	let isLoading = $state(true);
	let currentFolderId = $state<string | null>(null);
	let isCreatingFolder = $state(false);
	let bulkMode = $state(false);
	let refreshTrigger = $state(0);

	const selectedItemsCount = $derived(Object.keys(selectedItems).length);
	const allSelected = $derived(items.length > 0 && selectedItemsCount === items.length);
	const someSelected = $derived(selectedItemsCount > 0 && !allSelected);

	// Centralized, cancellable data loading effect
	$effect(() => {
		// Add refreshTrigger to the dependencies
		const trigger = refreshTrigger;
		let aborted = false;

		(async () => {
			isLoading = true;
			try {
				// Fetch files and ancestors in parallel for better performance
				const filesPromise = getFiles({
					parent_id: currentFolderId,
					limit: 50,
					offset: 0,
					sort_by: sortBy,
					order: order,
					type: filterType
				});

				const ancestorsPromise = currentFolderId
					? getFolderAncestors(currentFolderId)
					: Promise.resolve([]); // At root, ancestors are an empty array

				// Await both promises simultaneously
				const [fileData, ancestorData] = await Promise.all([filesPromise, ancestorsPromise]);

				if (aborted) return; // Don't update state if effect has been re-run

				// Atomically update state after all data is successfully fetched
				items = fileData.items || []; // Guard against null from API response
				hasMore = fileData.hasMore;
				breadcrumbItems.set(ancestorData);
			} catch (error) {
				if (aborted) return;
				console.error('Failed to load workspace data:', error);
				// On error, reset to a clean empty state
				items = [];
				hasMore = false;
				breadcrumbItems.set([]);
			} finally {
				if (aborted) return;
				// This will always run, ensuring the loading spinner doesn't get stuck
				isLoading = false;
			}
		})();

		return () => {
			aborted = true;
		};
	});

	function handleNavigate(id: string | null) {
		if (currentFolderId === id) return;
		currentFolderId = id;
		bulkMode = false;
		for (const key in selectedItems) {
			delete selectedItems[key];
		}
	}

	function handleFolderCreated() {
		isCreatingFolder = false;
		refreshTrigger++; // Trigger the effect to refresh data
	}

	function toggleBulkMode() {
		bulkMode = !bulkMode;
		if (!bulkMode) {
			for (const key in selectedItems) {
				delete selectedItems[key];
			}
		}
	}

	function toggleSelectAll() {
		if (allSelected) {
			for (const key in selectedItems) {
				delete selectedItems[key];
			}
		} else {
			for (const item of items) {
				selectedItems[item.id] = true;
			}
			if (!bulkMode) {
				bulkMode = true;
			}
		}
	}

	import { toast } from 'svelte-sonner';

	async function handleBulkDelete() {
		const itemsToDelete = Object.keys(selectedItems);
		if (itemsToDelete.length === 0) return;

		try {
			const deletePromises = itemsToDelete.map((id) => {
				const item = items.find((i) => i.id === id);
				if (item) {
					return deleteFile(id, item.type);
				}
				return Promise.resolve(); // Should not happen, but as a safeguard
			});

			await Promise.all(deletePromises);
			toast.success(
				itemsToDelete.length > 1
					? `已成功删除 ${itemsToDelete.length} 个项目`
					: '已成功删除 1 个项目'
			);
		} catch (error) {
			console.error('Failed to delete items:', error);
			toast.error(`删除失败: ${error instanceof Error ? error.message : '未知错误'}`);
		} finally {
			// Clear selection and refresh the list
			for (const key in selectedItems) {
				delete selectedItems[key];
			}
			bulkMode = false;
			refreshTrigger++;
		}
	}

	function toggleItem(id: string) {
		if (selectedItems[id]) {
			delete selectedItems[id];
		} else {
			selectedItems[id] = true;
		}
	}
</script>

<div>
	<Toolbar
		{bulkMode}
		{selectedItemsCount}
		{currentFolderId}
		onCreateFolder={() => (isCreatingFolder = true)}
		onToggleBulk={toggleBulkMode}
		onBulkDelete={handleBulkDelete}
		onNavigate={handleNavigate}
	/>

	<div class="my-6 border-t border-zinc-200 dark:border-zinc-700">
		<ListHeader
			{allSelected}
			{someSelected}
			{bulkMode}
			{selectedItemsCount}
			on:toggleAll={toggleSelectAll}
			on:bulkdelete={handleBulkDelete}
		/>

		<!-- 新建文件夹组件 -->
		{#if isCreatingFolder}
			<NewFolderItem
				parentId={currentFolderId}
				on:create={handleFolderCreated}
				on:cancel={() => (isCreatingFolder = false)}
			/>
		{/if}

		<!-- 文件列表 -->
		{#if isLoading}
			<FolderListItemSkeleton />
			<MarkdownListItemSkeleton />
		{:else if items.length === 0}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					width="48"
					height="48"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
					stroke-linejoin="round"
					class="mb-4 text-zinc-400 dark:text-zinc-500"
				>
					<path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z" />
				</svg>
				<h3 class="text-lg font-semibold text-zinc-800 dark:text-zinc-200">此文件夹为空</h3>
				<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
					点击“新建文档”或“新建文件夹”来开始创作。
				</p>
			</div>
		{:else}
			{#each items as item (item.id)}
				{#if item.type === 'folder'}
					<FolderListItem
						{item}
						{selectedItems}
						{bulkMode}
						onToggle={toggleItem}
						onNavigate={handleNavigate}
						onRefresh={() => refreshTrigger++}
					/>
				{:else if item.type === 'markdown'}
					<MarkdownListItem {item} {selectedItems} {bulkMode} onToggle={toggleItem} onRefresh={() => refreshTrigger++} />
				{/if}
			{/each}
		{/if}
	</div>
</div>
