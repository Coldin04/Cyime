<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import type { MediaAssetItem, MediaAssetReferencesResponse } from '$lib/api/media';
	import { deleteMediaAsset, getMediaAssetReferences, listMediaAssets } from '$lib/api/media';

	const PAGE_SIZE = 20;

	let items = $state<MediaAssetItem[]>([]);
	let total = $state(0);
	let hasMore = $state(false);
	let loading = $state(false);
	let errorMessage = $state('');

	let queryInput = $state('');
	let query = $state('');
	let kind = $state<'all' | 'image' | 'video' | 'file'>('all');
	let status = $state<'all' | 'ready' | 'pending_delete' | 'deleted' | 'failed'>('all');
	let offset = $state(0);

	let referencesOpen = $state(false);
	let referencesLoading = $state(false);
	let referencesError = $state('');
	let referencesAsset: MediaAssetItem | null = $state(null);
	let referencesData: MediaAssetReferencesResponse | null = $state(null);
	let listRequestId = 0;
	let refsRequestId = 0;

	onMount(() => {
		void loadAssets();
	});

	function formatBytes(bytes: number): string {
		if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
		const units = ['B', 'KB', 'MB', 'GB'];
		let value = bytes;
		let idx = 0;
		while (value >= 1024 && idx < units.length - 1) {
			value /= 1024;
			idx += 1;
		}
		return `${value.toFixed(value >= 10 || idx === 0 ? 0 : 1)} ${units[idx]}`;
	}

	function formatDate(iso: string): string {
		const d = new Date(iso);
		if (Number.isNaN(d.getTime())) return iso;
		return d.toLocaleString();
	}

	function statusLabel(statusValue: string): string {
		switch (statusValue) {
			case 'ready':
				return '正常';
			case 'pending_delete':
				return '待清理';
			case 'deleted':
				return '已删除';
			case 'failed':
				return '异常';
			default:
				return statusValue;
		}
	}

	async function loadAssets() {
		const requestId = ++listRequestId;
		loading = true;
		errorMessage = '';
		try {
			const result = await listMediaAssets({
				kind,
				status,
				q: query.trim(),
				limit: PAGE_SIZE,
				offset
			});
			if (requestId !== listRequestId) return;
			items = result.items;
			total = result.total;
			hasMore = result.hasMore;
		} catch (error) {
			if (requestId !== listRequestId) return;
			errorMessage = error instanceof Error ? error.message : '加载媒体资源失败';
			items = [];
			total = 0;
			hasMore = false;
		} finally {
			if (requestId !== listRequestId) return;
			loading = false;
		}
	}

	async function applyFilters() {
		offset = 0;
		query = queryInput.trim();
		await loadAssets();
	}

	async function goPrevPage() {
		if (offset <= 0) return;
		offset = Math.max(0, offset - PAGE_SIZE);
		await loadAssets();
	}

	async function goNextPage() {
		if (!hasMore) return;
		offset += PAGE_SIZE;
		await loadAssets();
	}

	async function openReferences(asset: MediaAssetItem) {
		const requestId = ++refsRequestId;
		referencesOpen = true;
		referencesAsset = asset;
		referencesLoading = true;
		referencesError = '';
		referencesData = null;
		try {
			const data = await getMediaAssetReferences(asset.id);
			if (requestId !== refsRequestId) return;
			referencesData = data;
		} catch (error) {
			if (requestId !== refsRequestId) return;
			referencesError = error instanceof Error ? error.message : '加载引用失败';
		} finally {
			if (requestId !== refsRequestId) return;
			referencesLoading = false;
		}
	}

	function closeReferences() {
		refsRequestId += 1;
		referencesOpen = false;
		referencesAsset = null;
		referencesData = null;
		referencesError = '';
	}

	async function handleDeleteAsset(asset: MediaAssetItem) {
		if (!asset.deletable) return;
		const ok = confirm(`确认删除资源 "${asset.filename}" 吗？此操作不可恢复。`);
		if (!ok) return;

		try {
			await deleteMediaAsset(asset.id);
			toast.success('资源已删除');
			if (items.length === 1 && offset > 0) {
				offset = Math.max(0, offset - PAGE_SIZE);
			}
			await loadAssets();
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '删除失败');
		}
	}
</script>

<div class="space-y-4">
	<div class="grid gap-3 rounded-xl border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-700/50 dark:bg-zinc-800/20 sm:grid-cols-[minmax(0,1fr)_150px_170px_auto]">
		<input
			bind:value={queryInput}
			type="text"
			placeholder="搜索文件名"
			class="w-full rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-800 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
			onkeydown={async (e) => {
				if (e.key === 'Enter') {
					e.preventDefault();
					await applyFilters();
				}
			}}
		/>
		<select
			bind:value={kind}
			class="rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-800 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
			onchange={applyFilters}
		>
			<option value="all">全部类型</option>
			<option value="image">图片</option>
			<option value="video">视频</option>
			<option value="file">文件</option>
		</select>
		<select
			bind:value={status}
			class="rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-800 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
			onchange={applyFilters}
		>
			<option value="all">全部状态</option>
			<option value="ready">正常</option>
			<option value="pending_delete">待清理</option>
			<option value="deleted">已删除</option>
			<option value="failed">异常</option>
		</select>
		<button
			type="button"
			class="rounded-lg bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
			onclick={applyFilters}
		>
			搜索
		</button>
	</div>

	{#if errorMessage}
		<div class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
			{errorMessage}
		</div>
	{/if}

	<div class="rounded-xl border border-zinc-200 dark:border-zinc-700/50">
		{#if loading}
			<div class="p-8 text-center text-sm text-zinc-500 dark:text-zinc-400">加载中...</div>
		{:else if items.length === 0}
			<div class="p-8 text-center text-sm text-zinc-500 dark:text-zinc-400">暂无媒体资源</div>
		{:else}
			<div class="divide-y divide-zinc-100 dark:divide-zinc-800/70">
				{#each items as item (item.id)}
					<div class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
						<div class="min-w-0">
							<p class="truncate text-sm font-medium text-zinc-900 dark:text-zinc-100">{item.filename}</p>
							<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
								{item.kind.toUpperCase()} · {formatBytes(item.fileSize)} · {item.mimeType}
							</p>
							<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
								状态：{statusLabel(item.status)} · 引用：{item.referenceCount} · 上传时间：{formatDate(item.createdAt)}
							</p>
						</div>
						<div class="flex flex-wrap items-center gap-2">
							<button
								type="button"
								class="rounded-lg border border-zinc-200 px-3 py-1.5 text-xs font-medium text-zinc-700 transition hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
								onclick={() => openReferences(item)}
							>
								查看引用
							</button>
							<button
								type="button"
								class="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-700 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-900/50 dark:text-red-300 dark:hover:bg-red-950/30"
								disabled={!item.deletable}
								onclick={() => handleDeleteAsset(item)}
							>
								删除
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="flex items-center justify-between text-sm">
		<p class="text-zinc-500 dark:text-zinc-400">共 {total} 项</p>
		<div class="flex gap-2">
			<button
				type="button"
				class="rounded-lg border border-zinc-200 px-3 py-1.5 text-zinc-700 transition hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
				disabled={offset === 0 || loading}
				onclick={goPrevPage}
			>
				上一页
			</button>
			<button
				type="button"
				class="rounded-lg border border-zinc-200 px-3 py-1.5 text-zinc-700 transition hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
				disabled={!hasMore || loading}
				onclick={goNextPage}
			>
				下一页
			</button>
		</div>
	</div>
</div>

{#if referencesOpen}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
		<div class="w-full max-w-lg rounded-2xl border border-zinc-200 bg-white p-5 shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
			<div class="flex items-start justify-between gap-3">
				<div>
					<h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">资源引用</h3>
					<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400 truncate">
						{referencesAsset?.filename}
					</p>
				</div>
				<button
					type="button"
					class="rounded-full p-1.5 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
					onclick={closeReferences}
				>
					✕
				</button>
			</div>

			<div class="mt-4">
				{#if referencesLoading}
					<p class="text-sm text-zinc-500 dark:text-zinc-400">加载中...</p>
				{:else if referencesError}
					<p class="text-sm text-red-600 dark:text-red-300">{referencesError}</p>
				{:else if referencesData}
					<p class="text-sm text-zinc-600 dark:text-zinc-300">被 {referencesData.referenceCount} 篇文档引用</p>
					{#if referencesData.documents.length === 0}
						<p class="mt-3 text-sm text-zinc-500 dark:text-zinc-400">暂无引用文档</p>
					{:else}
						<ul class="mt-3 space-y-2">
							{#each referencesData.documents as doc (doc.documentId)}
								<li class="rounded-lg border border-zinc-200 px-3 py-2 dark:border-zinc-700">
									<div class="flex items-center justify-between gap-3">
										<div class="min-w-0">
											<p class="truncate text-sm font-medium text-zinc-900 dark:text-zinc-100">{doc.title}</p>
											<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">{formatDate(doc.updatedAt)}</p>
										</div>
										<a
											href={`/edit/documents/${doc.documentId}`}
											class="shrink-0 text-xs font-medium text-riptide-700 hover:underline dark:text-riptide-300"
										>
											前往文档
										</a>
									</div>
								</li>
							{/each}
						</ul>
					{/if}
				{/if}
			</div>
		</div>
	</div>
{/if}
