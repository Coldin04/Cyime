<script lang="ts">
	import TopBar from '$lib/components/workspace/TopBar.svelte';
	import Toolbar from '$lib/components/workspace/Toolbar.svelte';
	import ListHeader from '$lib/components/workspace/ListHeader.svelte';
	import FolderListItem from '$lib/components/workspace/FolderListItem.svelte';
	import MarkdownListItem from '$lib/components/workspace/MarkdownListItem.svelte';
	import FolderListItemSkeleton from '$lib/components/workspace/FolderListItemSkeleton.svelte';
	import MarkdownListItemSkeleton from '$lib/components/workspace/MarkdownListItemSkeleton.svelte';
	import GreetingHeader from '$lib/components/workspace/GreetingHeader.svelte';
	import type { Component } from 'svelte';

	// 1. Component Mapping for dynamic rendering
	const componentMap: Record<string, Component<any>> = {
		folder: FolderListItem,
		markdown: MarkdownListItem
	};

	// 2. Placeholder data simulating a backend response
	const items = [
		{ id: 1, type: 'folder', name: 'Project Documents' },
		{ id: 2, type: 'folder', name: 'Meeting Notes' },
		{ id: 3, type: 'markdown', name: 'Q1 Roadmap.md' },
		{ id: 4, type: 'markdown', name: 'Component Design Ideas.md' }
	];
</script>

<TopBar />

<main class="container mx-auto px-4 pt-16">
	<GreetingHeader />
	<Toolbar />

	<div class="rounded-lg border border-zinc-200 bg-white dark:border-zinc-700 dark:bg-zinc-800/20">
		<ListHeader />
		<div class="border-t border-zinc-200 dark:border-zinc-700">
			<!-- 3. Dynamic rendering using <svelte:component> -->
			{#each items as item (item.id)}
				<svelte:component this={componentMap[item.type]} {item} />
			{/each}

			<!-- Skeleton loaders for loading state -->
			<FolderListItemSkeleton />
			<MarkdownListItemSkeleton />
		</div>
	</div>
</main>
