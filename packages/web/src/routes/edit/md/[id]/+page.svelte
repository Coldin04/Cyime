<script lang="ts">
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { get } from 'svelte/store';
	import Editor from '$lib/components/editor/Editor.svelte';
	import Search from '~icons/ph/magnifying-glass';
	import User from '~icons/ph/user';
	import SignOut from '~icons/ph/sign-out';
	import Trash from '~icons/ph/trash';
	import FileText from '~icons/ph/file-text';
	import Pencil from '~icons/ph/pencil';
	import Check from '~icons/ph/check';
	import X from '~icons/ph/x';
	import { auth } from '$lib/stores/auth';
	import { getMarkdownContent, updateMarkdownContent } from '$lib/api/editor';
	import { getMarkdownDetails, updateMarkdownTitle } from '$lib/api/workspace';
	import { toast } from 'svelte-sonner';

	let showUserMenu = $state(false);
	let title = $state('');
	let content = $state('');
	let isSaving = $state(false);
	let lastSaved = $state<Date | null>(null);
	let hasUnsavedChanges = $state(false);
	let isLoading = $state(true);
	let saveTimer: ReturnType<typeof setTimeout> | null = null;
	
	// Title editing state
	let isEditingTitle = $state(false);
	let editingTitle = $state('');
	let titleInput: HTMLInputElement | null = null;

	// Manually bridge the SvelteKit `page` store to a Svelte 5 signal
	// since this environment is in runes-mode but likely on an older Svelte 5 version.
	let pageSignal = $state(get(page));
	page.subscribe((p) => (pageSignal = p));
	const markdownId = $derived(pageSignal.params?.id);

	// Auto-save function with debounce
	function scheduleSave(newContent: string) {
		if (saveTimer) {
			clearTimeout(saveTimer);
		}

		saveTimer = setTimeout(async () => {
			await saveContent(newContent);
		}, 1000); // 1 second debounce
	}

	async function saveContent(newContent: string) {
		if (!hasUnsavedChanges) {
			console.log('[Save] No unsaved changes, skipping');
			return;
		}

		console.log('[Save] Saving content, length:', newContent?.length, 'markdownId:', markdownId);
		isSaving = true;
		try {
			const result = await updateMarkdownContent(markdownId!, newContent);
			console.log('[Save] Save successful:', result);
			lastSaved = new Date();
			hasUnsavedChanges = false;
			// 自动保存不弹窗
		} catch (error) {
			console.error('[Save] Failed to save content:', error);
			toast.error('保存失败');
		} finally {
			isSaving = false;
		}
	}

	function handleContentChange(newContent: string) {
		// Skip if currently loading content
		if (isLoading) return;
		
		hasUnsavedChanges = true;
		content = newContent;
		scheduleSave(newContent);
	}

	function handleTitleChange(newTitle: string) {
		title = newTitle;
		// TODO: Implement title update API
	}

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

	// Load markdown content when ID becomes available
	$effect(() => {
		if (markdownId) {
			isLoading = true;
			const loadContent = async () => {
				try {
					console.log('[Load] Loading markdown for ID:', markdownId);
					// Load markdown details (for title) and content in parallel
					const [details, data] = await Promise.all([
						getMarkdownDetails(markdownId),
						getMarkdownContent(markdownId)
					]);
					
					console.log('[Load] Content loaded, length:', data.content?.length);
					content = data.content;
					// Use the title from the API
					title = details.title || '未命名文档';
					console.log('[Load] Title loaded:', title);
					// Reset state for the new document
					hasUnsavedChanges = false;
					lastSaved = null;
					console.log('[Load] Content set, hasUnsavedChanges:', hasUnsavedChanges);
				} catch (error) {
					console.error('[Load] Failed to load markdown:', error);
					toast.error('加载文档失败');
					goto('/workspace');
				} finally {
					isLoading = false;
				}
			};
			loadContent();
		}
	});

	// Cleanup timer on unmount
	onMount(() => {
		return () => {
			if (saveTimer) {
				clearTimeout(saveTimer);
			}
		};
	});
</script>

<div class="flex h-screen flex-col bg-white dark:bg-zinc-900">
	<!-- Top Bar -->
	<header
		class="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-black/10 bg-white/80 px-4 backdrop-blur-md dark:border-white/10 dark:bg-zinc-900/80"
	>
		<!-- Left: Title -->
		<div class="flex flex-1 items-center gap-4">
			<FileText class="h-5 w-5 text-zinc-400" />
			{#if isEditingTitle}
				<input
					bind:this={titleInput}
					type="text"
					value={editingTitle}
					oninput={(e) => editingTitle = e.currentTarget.value}
					onkeydown={handleTitleKeydown}
					onblur={saveTitle}
					class="w-full max-w-xl bg-transparent text-lg font-medium text-zinc-900 placeholder-zinc-400 focus:outline-none dark:text-zinc-100"
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
			{:else}
				<button
					onclick={startEditingTitle}
					class="flex flex-1 items-center gap-2 group"
					title="点击编辑标题"
				>
					<input
						type="text"
						value={title}
						readonly
						class="w-full max-w-xl bg-transparent text-lg font-medium text-zinc-900 placeholder-zinc-400 focus:outline-none dark:text-zinc-100 cursor-pointer group-hover:bg-zinc-100 dark:group-hover:bg-zinc-800 rounded px-2 py-1 transition-colors"
						placeholder="文档标题"
					/>
					<Pencil class="h-4 w-4 text-zinc-400 opacity-0 group-hover:opacity-100 transition-opacity" />
				</button>
			{/if}
			{#if hasUnsavedChanges}
				<span class="text-sm text-zinc-400">未保存</span>
			{:else if isSaving}
				<span class="text-sm text-zinc-400">保存中...</span>
			{:else if lastSaved}
				<span class="text-sm text-zinc-400">
					已保存 {lastSaved.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
				</span>
			{/if}
		</div>

		<!-- Right: Search and User Menu -->
		<div class="flex items-center gap-4">
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

	<!-- Editor -->
	<main class="flex-1 overflow-hidden">
		<div class="h-full w-full">
			{#if browser && !isLoading}
				<Editor {content} onContentChange={handleContentChange} />
			{:else}
				<div class="prose dark:prose-invert">
					<p>正在加载编辑器...</p>
				</div>
			{/if}
		</div>
	</main>
</div>