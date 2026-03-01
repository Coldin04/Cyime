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

<div class="flex items-center justify-between pt-4 pb-8">
	<!-- Breadcrumbs -->
	<nav aria-label="Breadcrumb" class="flex items-center text-zinc-500 dark:text-zinc-400">
		<a href="/workspace" class="hover:underline">Workspace</a>
		<CaretRight class="mx-1 h-4 w-4" />
		<a href="/workspace" class="hover:underline">Learning</a>
		<CaretRight class="mx-1 h-4 w-4" />
		<span class="font-medium text-zinc-800 dark:text-zinc-200">Svelte</span>
	</nav>

	<!-- Action Buttons -->
	<div class="relative flex items-center" bind:this={menuElement}>
		<button
			onclick={handleCreateDocument}
			class="inline-flex items-center justify-center gap-2 rounded-l-lg bg-riptide-500 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800 disabled:opacity-50"
		>
			<Plus class="h-4 w-4" />
			<span>新建文档</span>
		</button>
		<button
			onclick={toggleMenu}
			class="rounded-r-lg border-l border-riptide-400 bg-riptide-500 p-2 text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800"
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
