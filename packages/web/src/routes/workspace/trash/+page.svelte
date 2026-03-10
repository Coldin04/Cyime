<script lang="ts">
	import {
		getTrashedFiles,
		restoreItems,
		permanentDeleteItems,
		type TrashItem
	} from '$lib/api/workspace';
	import Folder from '~icons/ph/folder';
	import File from '~icons/ph/file';
	import FolderListItemSkeleton from '$lib/components/workspace/FolderListItemSkeleton.svelte';
	import GreetingHeader from '$lib/components/workspace/GreetingHeader.svelte';
	import { toast } from 'svelte-sonner';
	import ClockClockwise from '~icons/ph/clock-clockwise';
	import TrashSimple from '~icons/ph/trash-simple';
	import * as m from '$paraglide/messages';

	let items = $state<TrashItem[]>([]);
	let isLoading = $state(true);
	let refreshTrigger = $state(0);

	$effect(() => {
		const trigger = refreshTrigger;
		(async () => {
			isLoading = true;
			try {
				const result = await getTrashedFiles({});
				items = result.items || [];
			} catch (error) {
				console.error('Failed to load trashed items:', error);
				items = [];
				toast.error(
					m.trash_restore_failed({ error: error instanceof Error ? error.message : '未知错误' })
				);
			} finally {
				isLoading = false;
			}
		})();
	});

async function handleRestore(item: TrashItem) {
	try {
		await restoreItems([{ id: item.id, type: item.type }]);
		toast.success(m.trash_restore_success({ name: item.name }));
		refreshTrigger++;
	} catch (error) {
		toast.error(m.trash_restore_failed({ error: error instanceof Error ? error.message : '未知错误' }));
	}
}

async function handlePermanentDelete(item: TrashItem) {
	if (!confirm(m.trash_delete_permanent_confirm({ name: item.name }))) {
		return;
	}
	try {
		await permanentDeleteItems([{ id: item.id, type: item.type }]);
		toast.success(m.trash_delete_permanent_success({ name: item.name }));
		refreshTrigger++;
	} catch (error) {
		toast.error(m.trash_delete_permanent_failed({ error: error instanceof Error ? error.message : '未知错误' }));
	}
}

async function handleEmptyTrash() {
	if (
		!confirm(
			m.trash_empty_confirm({ count: items.length })
		)
	) {
		return;
	}
	try {
		await permanentDeleteItems([]); // Pass empty array to delete all
		toast.success(m.trash_empty_success());
		refreshTrigger++;
	} catch (error) {
		toast.error(m.trash_empty_failed({ error: error instanceof Error ? error.message : '未知错误' }));
	}
}
</script>

<svelte:head>
  <title>{m.page_title_trash()}</title>
</svelte:head>

<div>
	<GreetingHeader mode="trash" />

	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold text-zinc-900 dark:text-zinc-100">{m.trash_title()}</h1>
			<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
				{m.trash_description()}
			</p>
		</div>
		<button
			onclick={handleEmptyTrash}
			disabled={items.length === 0}
			class="inline-flex h-10 items-center gap-2 rounded-lg bg-red-500 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-50"
		>
			<span>{m.trash_empty_button()}</span>
		</button>
	</div>

	<div class="my-6 border-t border-zinc-200 dark:border-zinc-700">
		{#if isLoading}
			<FolderListItemSkeleton />
			<FolderListItemSkeleton />
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
					<path d="M3 6h18" />
					<path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
				</svg>
				<h3 class="text-lg font-semibold text-zinc-800 dark:text-zinc-200">{m.trash_empty_state_title()}</h3>
				<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">{m.trash_empty_state_description()}</p>
			</div>
		{:else}
			<!-- Header -->
			<div class="grid grid-cols-[1fr_120px_auto] gap-4 px-4 py-2 text-sm font-medium text-zinc-500">
				<span>{m.common_name()}</span>
				<span>{m.common_deleted_at()}</span>
			</div>
			<!-- List -->
			{#each items as item (item.id)}
				<div
					class="group grid grid-cols-[1fr_120px_auto] items-center gap-4 rounded-lg px-4 py-2 hover:bg-zinc-100 dark:hover:bg-zinc-800"
				>
					<div class="flex items-center gap-3 truncate">
						{#if item.type === 'folder'}
							<Folder class="h-5 w-5 flex-shrink-0 text-teal-500" />
						{:else}
							<File class="h-5 w-5 flex-shrink-0 text-blue-500" />
						{/if}
						<span class="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">
							{item.name}
						</span>
					</div>
					<div class="text-sm text-zinc-500">
						{new Date(item.deletedAt).toLocaleDateString()}
					</div>
					<div
						class="flex items-center justify-end gap-2 opacity-0 transition-opacity group-hover:opacity-100"
					>
						<button
							onclick={() => handleRestore(item)}
							class="grid h-8 w-8 place-content-center rounded-md text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
							title={m.common_restore()}
						>
							<ClockClockwise class="h-5 w-5" />
						</button>
						<button
							onclick={() => handlePermanentDelete(item)}
							class="grid h-8 w-8 place-content-center rounded-md text-red-500 transition-colors hover:bg-red-500/10"
							title={m.common_permanent_delete()}
						>
							<TrashSimple class="h-5 w-5" />
						</button>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>
