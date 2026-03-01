<script lang="ts">
	import CaretRight from '~icons/ph/caret-right';
	import Plus from '~icons/ph/plus';
	import DotsThreeVertical from '~icons/ph/dots-three-vertical';
	import FolderPlus from '~icons/ph/folder-plus';
	import { createEventDispatcher } from 'svelte';
	import { createMarkdown } from '$lib/api/workspace';
	import { goto } from '$app/navigation';
	import { onMount, onDestroy } from 'svelte';

	let currentFolderId: string | null = null;
	let showMenu = $state(false);
	let menuElement: HTMLElement;

	// A simple representation of the breadcrumb path.
	// In a real app, this would be derived from the current route or a store.
	const breadcrumbPath = $state([
		{ name: 'Workspace', href: '/workspace' },
		{ name: 'Learning', href: '/workspace' },
		{ name: 'Svelte', href: null }
	]);

	const dispatch = createEventDispatcher();

	function toggleMenu() {
		showMenu = !showMenu;
	}

	function handleCreateFolder() {
		dispatch('createfolder');
		showMenu = false;
	}

	function handleClickOutside(event: MouseEvent) {
		if (showMenu && menuElement && !menuElement.contains(event.target as Node)) {
			showMenu = false;
		}
	}

	onMount(() => {
		document.addEventListener('click', handleClickOutside, true);
	});

	onDestroy(() => {
		document.removeEventListener('click', handleClickOutside, true);
	});

	async function handleCreateDocument() {
		try {
			const newDoc = await createMarkdown({
				title: '未命名文档',
				content: '',
				folderId: currentFolderId
			});
			// 创建成功后跳转到编辑器
			goto(`/edit/md/${newDoc.id}`);
		} catch (error) {
			console.error('创建文档失败:', error);
		}
	}
</script>

<div class="flex items-center justify-between">
	<!-- Breadcrumbs -->
	<nav
		aria-label="Breadcrumb"
		class="flex min-w-0 items-center text-zinc-500 dark:text-zinc-400"
	>
		{#each breadcrumbPath as segment, i}
			{#if segment.href}
				<a href={segment.href} class="truncate hover:underline">
					{segment.name}
				</a>
			{:else}
				<span class="truncate font-medium text-zinc-800 dark:text-zinc-200">{segment.name}</span>
			{/if}

			{#if i < breadcrumbPath.length - 1}
				<CaretRight class="mx-1 h-4 w-4 flex-shrink-0" />
			{/if}
		{/each}
	</nav>

	<!-- Action Buttons -->
	<div class="relative ml-4 flex flex-shrink-0 items-center" bind:this={menuElement}>
		<button
			onclick={handleCreateDocument}
			class="inline-flex h-10 items-center justify-center gap-2 rounded-l-lg bg-riptide-500 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800 disabled:opacity-50 sm:px-4"
		>
			<Plus class="h-4 w-4" />
			<span class="hidden sm:inline">新建文档</span>
		</button>
		<button
			onclick={toggleMenu}
			class="inline-flex h-10 w-10 items-center justify-center rounded-r-lg border-l border-riptide-400 bg-riptide-500 p-2 text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800"
			aria-label="更多选项"
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
					onclick={handleCreateFolder}
					class="flex w-full items-center gap-3 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
					role="menuitem"
				>
					<FolderPlus class="h-4 w-4" />
					<span>新建文件夹</span>
				</button>
			</div>
		{/if}
	</div>
</div>

