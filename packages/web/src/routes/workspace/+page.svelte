<script lang="ts">
	import TopBar from '$lib/components/workspace/TopBar.svelte';
	import Toolbar from '$lib/components/workspace/Toolbar.svelte';
	import ListHeader from '$lib/components/workspace/ListHeader.svelte';
	import FolderListItem from '$lib/components/workspace/FolderListItem.svelte';
	import MarkdownListItem from '$lib/components/workspace/MarkdownListItem.svelte';
	import FolderListItemSkeleton from '$lib/components/workspace/FolderListItemSkeleton.svelte';
	import MarkdownListItemSkeleton from '$lib/components/workspace/MarkdownListItemSkeleton.svelte';
	import GreetingHeader from '$lib/components/workspace/GreetingHeader.svelte';
	import NewFolderItem from '$lib/components/workspace/NewFolderItem.svelte';
	import { onMount } from 'svelte';
	import { getFiles, type FileItem } from '$lib/api/workspace';

	let items = $state<FileItem[]>([]);
	let selectedItems = $state<{ [key: string]: boolean }>({});
	let hasMore = $state(false);
	let sortBy = $state('updated_at');
	let order = $state('desc');
	let filterType = $state<'all' | 'folders' | 'markdowns'>('all');
	let isLoading = $state(false);
	let currentFolderId = $state<string | null>(null);
	let isCreatingFolder = $state(false);
	let bulkMode = $state(false);

	const selectedItemsCount = $derived(Object.keys(selectedItems).length);
	const allSelected = $derived(items.length > 0 && selectedItemsCount === items.length);
	const someSelected = $derived(selectedItemsCount > 0 && !allSelected);

	async function loadFiles(reset = false) {
		if (isLoading) return;

		isLoading = true;
		try {
			const data = await getFiles({
				parent_id: currentFolderId,
				limit: 50,
				offset: reset ? 0 : items.length,
				sort_by: sortBy,
				order: order,
				type: filterType
			});

			if (reset) {
				items = data.items;
				selectedItems = {};
			} else {
				items = [...items, ...data.items];
			}
			hasMore = data.hasMore;
		} catch (error) {
			console.error('加载失败:', error);
		} finally {
			isLoading = false;
		}
	}

	function handleFolderCreated() {
		isCreatingFolder = false;
		loadFiles(true);
	}

	function toggleBulkMode() {
		bulkMode = !bulkMode;
		if (!bulkMode) {
			selectedItems = {};
		}
	}

	function toggleSelectAll() {
		if (allSelected) {
			selectedItems = {};
		} else {
			const newSelected: { [key: string]: boolean } = {};
			for (const item of items) {
				newSelected[item.id] = true;
			}
			selectedItems = newSelected;
			// 全选时自动进入批量模式
			if (!bulkMode) {
				bulkMode = true;
			}
		}
	}

	function handleBulkDelete() {
		console.log('Delete selected items:', Object.keys(selectedItems));
		selectedItems = {};
		bulkMode = false;
	}

	function toggleItem(id: string) {
		if (selectedItems[id]) {
			delete selectedItems[id];
		} else {
			selectedItems[id] = true;
		}
	}

	onMount(() => {
		loadFiles(true);
	});
</script>

<TopBar />

<main class="max-w-5xl mx-auto px-4 sm:px-6 py-8">
	<GreetingHeader />
	<Toolbar
		{bulkMode}
		selectedItemsCount={selectedItemsCount}
		on:createfolder={() => (isCreatingFolder = true)}
		on:togglebulk={toggleBulkMode}
		on:bulkdelete={handleBulkDelete}
	/>

	<div class="my-6 border-t border-zinc-200 dark:border-zinc-700">
		<ListHeader
			{allSelected}
			{someSelected}
			{bulkMode}
			selectedItemsCount={selectedItemsCount}
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
		{#each items as item (item.id)}
			{#if item.type === 'folder'}
				<FolderListItem {item} {selectedItems} {bulkMode} onToggle={() => toggleItem(item.id)} />
			{:else if item.type === 'markdown'}
				<MarkdownListItem {item} {selectedItems} {bulkMode} onToggle={() => toggleItem(item.id)} />
			{/if}
		{/each}

		{#if isLoading}
			<FolderListItemSkeleton />
			<MarkdownListItemSkeleton />
		{/if}
	</div>
</main>
