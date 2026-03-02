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
	import { toast } from 'svelte-sonner';
	import ClockClockwise from '~icons/ph/clock-clockwise';
	import TrashSimple from '~icons/ph/trash-simple';

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
					`加载回收站项目失败: ${error instanceof Error ? error.message : '未知错误'}`
				);
			} finally {
				isLoading = false;
			}
		})();
	});

	async function handleRestore(item: TrashItem) {
		try {
			await restoreItems([{ id: item.id, type: item.type }]);
			toast.success(`“${item.name}” 已恢复`);
			refreshTrigger++;
		} catch (error) {
			toast.error(`恢复失败: ${error instanceof Error ? error.message : '未知错误'}`);
		}
	}

	async function handlePermanentDelete(item: TrashItem) {
		if (!confirm(`您确定要永久删除 “${item.name}” 吗？此操作无法撤销。`)) {
			return;
		}
		try {
			await permanentDeleteItems([{ id: item.id, type: item.type }]);
			toast.success(`“${item.name}” 已被永久删除`);
			refreshTrigger++;
		} catch (error) {
			toast.error(`永久删除失败: ${error instanceof Error ? error.message : '未知错误'}`);
		}
	}

	async function handleEmptyTrash() {
		if (
			!confirm(
				`您确定要清空回收站吗？所有 ${items.length} 个项目都将被永久删除，此操作无法撤销。`
			)
		) {
			return;
		}
		try {
			await permanentDeleteItems([]); // Pass empty array to delete all
			toast.success('回收站已清空');
			refreshTrigger++;
		} catch (error) {
			toast.error(`清空回收站失败: ${error instanceof Error ? error.message : '未知错误'}`);
		}
	}
</script>

<div>
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold text-zinc-900 dark:text-zinc-100">回收站</h1>
			<p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
				这里的项目将在 30 天后被自动永久删除。
			</p>
		</div>
		<button
			onclick={handleEmptyTrash}
			disabled={items.length === 0}
			class="inline-flex h-10 items-center gap-2 rounded-lg bg-red-500 px-3 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-50"
		>
			<span>清空回收站</span>
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
				<h3 class="text-lg font-semibold text-zinc-800 dark:text-zinc-200">回收站是空的</h3>
				<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">这里没有已删除的项目。</p>
			</div>
		{:else}
			<!-- Header -->
			<div class="grid grid-cols-[1fr_120px_auto] gap-4 px-4 py-2 text-sm font-medium text-zinc-500">
				<span>名称</span>
				<span>删除于</span>
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
							title="恢复"
						>
							<ClockClockwise class="h-5 w-5" />
						</button>
						<button
							onclick={() => handlePermanentDelete(item)}
							class="grid h-8 w-8 place-content-center rounded-md text-red-500 transition-colors hover:bg-red-500/10"
							title="永久删除"
						>
							<TrashSimple class="h-5 w-5" />
						</button>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>
