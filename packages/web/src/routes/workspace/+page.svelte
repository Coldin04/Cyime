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
	let selectedItems = $state(new Set<string>());
	let hasMore = $state(false);
	let sortBy = $state('updated_at');
	let order = $state('desc');
	let filterType = $state<'all' | 'folders' | 'markdowns'>('all');
	let isLoading = $state(false);
	let currentFolderId = $state<string | null>(null);
	let isCreatingFolder = $state(false);

	const allSelected = $derived(items.length > 0 && selectedItems.size === items.length);
	const someSelected = $derived(selectedItems.size > 0 && !allSelected);

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
				selectedItems.clear();
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

	function toggleSelectAll() {
		if (allSelected) {
			selectedItems.clear();
		} else {
			items.forEach((item) => selectedItems.add(item.id));
		}
		// Force reactivity since we are mutating a set
		selectedItems = selectedItems;
	}

	onMount(() => {
		loadFiles(true);
	});
</script>

<TopBar />

<main class="max-w-5xl mx-auto px-4 sm:px-6 py-8">
	<GreetingHeader />
	<Toolbar on:createfolder={() => (isCreatingFolder = true)} />

	<div class="my-6 border-t border-zinc-200 dark:border-zinc-700">
		<ListHeader
			{allSelected}
			{someSelected}
			on:toggleAll={toggleSelectAll}
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
				<FolderListItem {item} {selectedItems} />
			{:else if item.type === 'markdown'}
				<MarkdownListItem {item} {selectedItems} />
			{/if}
		{/each}

		{#if isLoading}
			<FolderListItemSkeleton />
			<MarkdownListItemSkeleton />
		{/if}
	</div>
</main>
