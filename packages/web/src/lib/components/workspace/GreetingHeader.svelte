<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { auth } from '$lib/stores/auth';
	import * as m from '$paraglide/messages';
	import Plus from '~icons/ph/plus';
	import DotsThreeVertical from '~icons/ph/dots-three-vertical';
	import FolderPlus from '~icons/ph/folder-plus';
	import CheckSquare from '~icons/ph/check-square';
	import House from '~icons/ph/house';
	import { createDocument } from '$lib/api/workspace';
	import { goto } from '$app/navigation';
	import { toast } from 'svelte-sonner';
	import { workspaceContext } from '$lib/stores/workspace';
	import { clickOutside } from '$lib/actions/clickOutside';

	let { mode = 'workspace' }: { mode?: 'workspace' | 'trash' } = $props();
	const isTrashMode = $derived(mode === 'trash');

	function getGreeting(): string {
		const hour = new Date().getHours();
		if (hour < 6) {
			return m.greeting_night();
		} else if (hour < 12) {
			return m.greeting_morning();
		} else if (hour < 14) {
			return m.greeting_noon();
		} else if (hour < 18) {
			return m.greeting_afternoon();
		} else {
			return m.greeting_evening();
		}
	}

	function getInitial(name: string | null): string {
		if (!name || name.trim() === '') {
			return m.common_user().charAt(0).toUpperCase();
		}
		return name.charAt(0).toUpperCase();
	}

	let showMenu = $state(false);
	let isLoading = $state(false);
	let avatarLoadFailed = $state(false);
	let avatarLoaded = $state(false);
	let avatarImgEl = $state<HTMLImageElement | null>(null);
	const avatarUrl = $derived(($auth.user?.avatarUrl || '').trim());

	$effect(() => {
		const _avatar = avatarUrl;
		avatarLoadFailed = false;
		avatarLoaded = false;
	});

	$effect(() => {
		if (avatarImgEl && avatarImgEl.complete && avatarImgEl.naturalWidth > 0) {
			avatarLoaded = true;
		}
	});

	function toggleMenu() {
		showMenu = !showMenu;
	}

	function closeMenu() {
		showMenu = false;
	}

	async function handleCreateDocument() {
		if (isLoading) return;
		
		isLoading = true;
		try {
			const newDoc = await createDocument({
				title: '',
				contentJson: {
					type: 'doc',
					content: [{ type: 'paragraph' }]
				},
				folderId: $workspaceContext.currentFolderId,
				documentType: 'rich_text'
			});
			goto(`/edit/documents/${newDoc.id}`);
		} catch (error) {
			console.error('创建文档失败:', error);
			toast.error(
				m.document_create_failed({
					error: error instanceof Error ? error.message : m.common_unknown_error()
				})
			);
		} finally {
			isLoading = false;
		}
	}

	function handleGoToWorkspaceRoot() {
		goto('/workspace');
	}

	function handleCreateFolder() {
		workspaceContext.update((ctx) => ({ ...ctx, isCreatingFolder: true }));
		closeMenu();
	}

	function handleToggleBulk() {
		workspaceContext.update((ctx) => ({
			...ctx,
			bulkMode: !ctx.bulkMode
		}));
		closeMenu();
	}

	afterNavigate(() => {
		closeMenu();
	});
</script>

<section class="mb-6 flex items-center justify-between gap-4">
	<div class="flex items-center gap-4">
		<div
			class="relative grid h-16 w-16 flex-shrink-0 place-content-center overflow-hidden rounded-full bg-riptide-100 dark:bg-riptide-900"
		>
			{#if avatarUrl && !avatarLoadFailed}
				{#if !avatarLoaded}
					<div
						class="absolute inset-0 animate-pulse bg-riptide-200/80 dark:bg-riptide-800/70"
						aria-hidden="true"
					></div>
				{/if}
				<img
					bind:this={avatarImgEl}
					src={avatarUrl}
					alt={m.greeting_avatar_alt({ name: $auth.user?.displayName || m.common_user() })}
					class="h-full w-full rounded-full object-cover transition-opacity duration-200"
					class:opacity-0={!avatarLoaded}
					class:opacity-100={avatarLoaded}
					decoding="async"
					fetchpriority="low"
					referrerpolicy="no-referrer"
					onload={() => {
						avatarLoaded = true;
					}}
					onerror={() => {
						avatarLoadFailed = true;
					}}
				/>
			{:else}
				<span class="text-3xl font-bold text-riptide-600 dark:text-riptide-300">
					{getInitial($auth.user?.displayName || null)}
				</span>
			{/if}
		</div>
			<div>
			<h2 class="text-2xl font-bold text-zinc-800 dark:text-zinc-200">
				{getGreeting()}, {$auth.user?.displayName || m.common_user()}
			</h2>
			<p class="text-zinc-500 dark:text-zinc-400">{m.greeting_question()}</p>
		</div>
	</div>

	<!-- Action Buttons -->
	<div
		class="relative flex flex-shrink-0 items-center"
		use:clickOutside={{
			enabled: showMenu && !isTrashMode,
			handler: closeMenu
		}}
	>
		<button
			onclick={isTrashMode ? handleGoToWorkspaceRoot : handleCreateDocument}
			disabled={isLoading}
			class="inline-flex h-10 items-center justify-center gap-2 bg-riptide-500 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800 disabled:opacity-50 sm:px-4 {isTrashMode
				? 'rounded-lg'
				: 'rounded-l-lg'}"
		>
			{#if isLoading}
				<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
					<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
				</svg>
			{:else if isTrashMode}
				<House class="h-4 w-4" />
			{:else}
				<Plus class="h-4 w-4" />
			{/if}
			<span class="hidden sm:inline">
				{isTrashMode ? m.topbar_back_to_workspace() : m.common_new_document()}
			</span>
		</button>
		{#if !isTrashMode}
			<button
				onclick={toggleMenu}
				class="inline-flex h-10 w-10 items-center justify-center rounded-r-lg border-l border-riptide-400 bg-riptide-500 p-2 text-white shadow-sm transition-colors hover:bg-riptide-600 active:bg-riptide-800"
				aria-label={m.common_more_options()}
			>
				<DotsThreeVertical class="h-5 w-5" />
			</button>
		{/if}

		{#if showMenu && !isTrashMode}
			<div
				class="absolute top-full right-0 z-10 mt-2 w-48 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-zinc-800 dark:ring-zinc-700"
				role="menu"
				aria-orientation="vertical"
				aria-labelledby="menu-button"
			>
				<button
					onclick={handleToggleBulk}
					class="flex w-full items-center gap-3 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
					role="menuitem"
				>
					<CheckSquare class="h-4 w-4" />
					<span>{m.common_bulk_select()}</span>
				</button>
				<button
					onclick={handleCreateFolder}
					class="flex w-full items-center gap-3 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
					role="menuitem"
				>
					<FolderPlus class="h-4 w-4" />
					<span>{m.common_new_folder()}</span>
				</button>
			</div>
		{/if}
	</div>
</section>
