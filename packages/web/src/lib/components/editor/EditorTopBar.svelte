<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import { auth } from '$lib/stores/auth';
	import { updateMarkdownTitle } from '$lib/api/workspace';
	import { toast } from 'svelte-sonner';

	// Icons
	import Home from '~icons/ph/house';
	import Search from '~icons/ph/magnifying-glass';
	import User from '~icons/ph/user';
	import SignOut from '~icons/ph/sign-out';
	import Trash from '~icons/ph/trash';
	import FileMd from '~icons/ph/file-md';
	import Pencil from '~icons/ph/pencil';
	import Check from '~icons/ph/check';
	import X from '~icons/ph/x';

	const {
		markdownId,
		initialTitle,
		isSaving,
		lastSaved,
		hasUnsavedChanges
	}: {
		markdownId: string;
		initialTitle: string;
		isSaving: boolean;
		lastSaved: Date | null;
		hasUnsavedChanges: boolean;
	} = $props();

	let showUserMenu = $state(false);
	let title = $state(initialTitle);

	// Title editing state
	let isEditingTitle = $state(false);
	let editingTitle = $state('');
	let titleInput: HTMLInputElement | null = null;

	$effect(() => {
		// When the initial title from the parent changes (e.g., on new doc load),
		// update the component's internal title state.
		if (initialTitle !== title) {
			title = initialTitle;
		}
	});

	async function startEditingTitle() {
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
			await updateMarkdownTitle(markdownId!, editingTitle.trim());
			title = editingTitle.trim();
			toast.success('标题已更新');
		} catch (error) {
			console.error('Failed to update title:', error);
			toast.error('更新标题失败');
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

	function toggleUserMenu() {
		showUserMenu = !showUserMenu;
	}

	function handleLogout() {
		auth.logout();
		showUserMenu = false;
	}
</script>

<!-- Top Bar -->
<header
	class="relative z-30 flex h-16 shrink-0 items-center justify-center border-b border-black/10 bg-white/80 backdrop-blur-md dark:border-white/10 dark:bg-zinc-900/80"
>
	<!-- Left Controls: Absolutely positioned -->
	<div class="absolute left-4 top-1/2 flex -translate-y-1/2 items-center gap-2">
		<!-- Home Button -->
		<a
			href="/workspace"
			class="grid h-8 w-8 shrink-0 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
			title="返回工作台"
		>
			<Home class="h-5 w-5" />
		</a>

		<!-- Divider -->
		<div class="h-5 w-px bg-zinc-200 dark:bg-zinc-700"></div>
	</div>

	<!-- Center: Title Section -->
	<div class="flex min-w-0 flex-1 items-center gap-2 px-20">
		<FileMd class="h-5 w-5 shrink-0 text-zinc-400 self-center" />

		<!-- Container for Title and Status -->
		<div class="flex flex-col">
			{#if isEditingTitle}
				<div class="flex items-center">
					<input
						bind:this={titleInput}
						type="text"
						value={editingTitle}
						oninput={(e) => (editingTitle = e.currentTarget.value)}
						onkeydown={handleTitleKeydown}
						onblur={saveTitle}
						class="w-full max-w-xl bg-transparent text-base text-zinc-900 placeholder-zinc-400 focus:outline-none dark:text-zinc-100"
						placeholder="文档标题"
					/>
					<div class="flex items-center gap-1">
						<button
							onclick={saveTitle}
							class="grid h-8 w-8 place-content-center rounded-full text-green-600 transition-colors hover:bg-green-100 dark:text-green-400 dark:hover:bg-green-900/30"
							title="保存标题"
						>
							<Check class="h-5 w-5" />
						</button>
						<button
							onclick={cancelEditingTitle}
							class="grid h-8 w-8 place-content-center rounded-full text-red-600 transition-colors hover:bg-red-100 dark:text-red-400 dark:hover:bg-red-900/30"
							title="取消"
						>
							<X class="h-5 w-5" />
						</button>
					</div>
				</div>
			{:else}
				<button
					onclick={startEditingTitle}
					class="group flex min-w-0 items-center gap-2"
					title="点击编辑标题"
				>
					<h1
						class="truncate rounded bg-transparent px-2 text-sm text-zinc-900 placeholder-zinc-400 transition-colors group-hover:bg-zinc-100 dark:text-zinc-100 dark:group-hover:bg-zinc-800"
					>
						{title || '未命名文档'}
					</h1>
					<Pencil
						class="h-4 w-4 shrink-0 text-zinc-400 opacity-0 transition-opacity group-hover:opacity-100"
					/>
				</button>
			{/if}

			<!-- Save Status -->
			<div class="px-2 py-0 text-left leading-3">
				{#if isSaving}
					<span class="text-xs text-zinc-400 py-0">保存中...</span>
				{:else if hasUnsavedChanges}
					<span class="text-xs text-zinc-400 py-0">未保存</span>
				{:else if lastSaved}
					<span class="text-xs text-zinc-400 py-0">
						已保存 {lastSaved.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
					</span>
				{:else}
					<span class="text-xs text-zinc-400 py-0">待修改</span>
				{/if}
			</div>
		</div>
	</div>

	<!-- Right Controls: Absolutely positioned -->
	<div class="absolute right-4 top-1/2 flex -translate-y-1/2 items-center gap-4">
		<button
			class="grid h-8 w-8 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
			title="搜索（开发中）"
		>
			<Search class="h-5 w-5" />
		</button>
		<div class="relative">
			<button
				onclick={toggleUserMenu}
				class="grid h-8 w-8 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
			>
				<User class="h-5 w-5" />
			</button>
			{#if showUserMenu}
				<div
					class="absolute top-full right-0 z-10 mt-2 w-48 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-zinc-800 dark:ring-zinc-700"
				>
					<a
						href="/workspace"
						class="block px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
						>返回工作区</a
					>
					<a
						href="/workspace/trash"
						class="flex items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
					>
						<Trash class="h-4 w-4" />
						<span>回收站</span>
					</a>
					<div class="my-1 h-px bg-zinc-200 dark:bg-zinc-700"></div>
					<button
						onclick={handleLogout}
						class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-red-600 hover:bg-zinc-100 dark:text-red-400 dark:hover:bg-zinc-700"
					>
						<SignOut class="h-4 w-4" />
						<span>登出</span>
					</button>
				</div>
			{/if}
		</div>
	</div>
</header>

