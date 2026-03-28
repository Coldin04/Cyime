<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import type { MediaAssetItem, MediaAssetReferencesResponse } from '$lib/api/media';
	import { deleteMediaAsset, getMediaAssetReferences, getMediaAssetURL, listMediaAssets } from '$lib/api/media';
	import { portal } from '$lib/actions/portal';
	import CopySimple from '~icons/ph/copy-simple';
	import Check from '~icons/ph/check';
	import FileImage from '~icons/ph/file-image';
	import FileVideo from '~icons/ph/file-video';
	import File from '~icons/ph/file';

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
	let referencesHint = $state('');
	let referencesAsset: MediaAssetItem | null = $state(null);
	let referencesData: MediaAssetReferencesResponse | null = $state(null);
	let listRequestId = 0;
	let refsRequestId = 0;
	let copiedDocumentID = $state('');
	let previewURLByAssetID = $state<Record<string, string>>({});
	let previewLoadingByAssetID = $state<Record<string, boolean>>({});
	let previewFailedByAssetID = $state<Record<string, boolean>>({});

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
				return m.user_media_status_ready();
			case 'pending_delete':
				return m.user_media_status_pending_delete();
			case 'deleted':
				return m.user_media_status_deleted();
			case 'failed':
				return m.user_media_status_failed();
			default:
				return statusValue;
		}
	}

	function isImageAsset(item: MediaAssetItem): boolean {
		return item.kind === 'image' && item.status !== 'deleted';
	}

	function hasReferences(item: MediaAssetItem): boolean {
		return item.referenceCount > 0;
	}

	async function loadPreviewURL(item: MediaAssetItem) {
		if (!isImageAsset(item)) return;
		if (previewURLByAssetID[item.id] || previewLoadingByAssetID[item.id]) return;

		previewLoadingByAssetID = { ...previewLoadingByAssetID, [item.id]: true };
		try {
			const result = await getMediaAssetURL(item.id);
			previewURLByAssetID = { ...previewURLByAssetID, [item.id]: result.url };
			if (previewFailedByAssetID[item.id]) {
				const next = { ...previewFailedByAssetID };
				delete next[item.id];
				previewFailedByAssetID = next;
			}
		} catch (error) {
			console.error('load media preview failed', error);
			previewFailedByAssetID = { ...previewFailedByAssetID, [item.id]: true };
		} finally {
			const next = { ...previewLoadingByAssetID };
			delete next[item.id];
			previewLoadingByAssetID = next;
		}
	}

	function onPreviewImageError(itemID: string) {
		previewFailedByAssetID = { ...previewFailedByAssetID, [itemID]: true };
	}

	function deleteDisabledReason(item: MediaAssetItem): string {
		if (item.deletable) return '';
		if (item.status === 'deleted') return m.user_media_delete_disabled_deleted();
		if (item.referenceCount > 0) return m.user_media_delete_disabled_referenced();
		return m.user_media_delete_disabled_unavailable();
	}

	function deleteButtonLabel(item: MediaAssetItem): string {
		if (item.status === 'deleted') return m.user_media_status_deleted();
		if (item.deletable) return m.user_media_action_delete_permanently();
		return m.user_media_action_delete();
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
			for (const item of result.items.slice(0, 6)) {
				void loadPreviewURL(item);
			}
		} catch (error) {
			if (requestId !== listRequestId) return;
			errorMessage = error instanceof Error ? error.message : m.user_media_error_load_assets();
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

	async function openReferences(asset: MediaAssetItem, hint = '') {
		const requestId = ++refsRequestId;
		referencesOpen = true;
		referencesAsset = asset;
		referencesLoading = true;
		referencesError = '';
		referencesHint = hint;
		referencesData = null;
		try {
			const data = await getMediaAssetReferences(asset.id);
			if (requestId !== refsRequestId) return;
			referencesData = data;
		} catch (error) {
			if (requestId !== refsRequestId) return;
			referencesError = error instanceof Error ? error.message : m.user_media_error_load_references();
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
		referencesHint = '';
	}

	async function copyDocumentID(documentID: string) {
		try {
			await navigator.clipboard.writeText(documentID);
			copiedDocumentID = documentID;
			toast.success(m.user_media_toast_document_id_copied());
			setTimeout(() => {
				if (copiedDocumentID === documentID) copiedDocumentID = '';
			}, 1200);
		} catch {
			toast.error(m.user_media_toast_copy_failed());
		}
	}

	async function handleDeleteAsset(asset: MediaAssetItem) {
		if (asset.status === 'deleted') return;
		if (!asset.deletable) return;
		const ok = confirm(m.user_media_confirm_delete_asset({ filename: asset.filename }));
		if (!ok) return;

		try {
			await deleteMediaAsset(asset.id);
			toast.success(m.user_media_toast_deleted());
			if (items.length === 1 && offset > 0) {
				offset = Math.max(0, offset - PAGE_SIZE);
			}
			await loadAssets();
		} catch (error) {
			const message = error instanceof Error ? error.message : m.user_media_toast_delete_failed();
			if (message.includes('referenced')) {
				toast.error(m.user_media_toast_cannot_delete_referenced());
				await loadAssets();
				return;
			}
			toast.error(message);
		}
	}
</script>

<div class="space-y-4">
	<div class="grid gap-3 rounded-xl border border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-700/50 dark:bg-zinc-800/20 sm:grid-cols-[minmax(0,1fr)_150px_170px_auto]">
		<input
			bind:value={queryInput}
			type="text"
			placeholder={m.user_media_search_placeholder()}
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
			<option value="all">{m.user_media_filter_kind_all()}</option>
			<option value="image">{m.user_media_filter_kind_image()}</option>
			<option value="video">{m.user_media_filter_kind_video()}</option>
			<option value="file">{m.user_media_filter_kind_file()}</option>
		</select>
		<select
			bind:value={status}
			class="rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-800 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
			onchange={applyFilters}
		>
			<option value="all">{m.user_media_filter_status_all()}</option>
			<option value="ready">{m.user_media_status_ready()}</option>
			<option value="pending_delete">{m.user_media_status_pending_delete()}</option>
			<option value="deleted">{m.user_media_status_deleted()}</option>
			<option value="failed">{m.user_media_status_failed()}</option>
		</select>
		<button
			type="button"
			class="rounded-lg bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
			onclick={applyFilters}
		>
			{m.user_media_action_search()}
		</button>
	</div>

	{#if errorMessage}
		<div class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
			{errorMessage}
		</div>
	{/if}

	<div class="rounded-xl border border-zinc-200 dark:border-zinc-700/50">
		{#if loading}
			<div class="p-8 text-center text-sm text-zinc-500 dark:text-zinc-400">{m.user_media_loading()}</div>
		{:else if items.length === 0}
			<div class="p-8 text-center text-sm text-zinc-500 dark:text-zinc-400">{m.user_media_empty()}</div>
		{:else}
			<div class="divide-y divide-zinc-100 dark:divide-zinc-800/70">
				{#each items as item (item.id)}
					<div class="grid gap-3 p-4 sm:grid-cols-[minmax(0,1fr)_240px] sm:items-center sm:gap-4">
						<div class="min-w-0 flex items-start gap-3">
							<div class="h-12 w-12 shrink-0 overflow-hidden rounded-lg border border-zinc-200 bg-zinc-100 dark:border-zinc-700 dark:bg-zinc-800">
								{#if isImageAsset(item) && previewURLByAssetID[item.id] && !previewFailedByAssetID[item.id]}
									<img
										src={previewURLByAssetID[item.id]}
										alt={item.filename}
										class="h-full w-full object-cover"
										loading="lazy"
										onerror={() => onPreviewImageError(item.id)}
									/>
								{:else}
									<div class="grid h-full w-full place-content-center text-zinc-500 dark:text-zinc-400">
										{#if item.kind === 'image'}
											<FileImage class="h-5 w-5" />
										{:else if item.kind === 'video'}
											<FileVideo class="h-5 w-5" />
										{:else}
											<File class="h-5 w-5" />
										{/if}
									</div>
								{/if}
							</div>

							<div class="min-w-0">
								<p class="truncate text-sm font-medium text-zinc-900 dark:text-zinc-100">{item.filename}</p>
								<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
									{item.kind === 'image'
										? m.user_media_filter_kind_image()
										: item.kind === 'video'
											? m.user_media_filter_kind_video()
											: item.kind === 'file'
												? m.user_media_filter_kind_file()
												: item.kind.toUpperCase()} · {formatBytes(item.fileSize)} · {item.mimeType}
								</p>
								<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
									{m.user_media_meta_status()}：{statusLabel(item.status)} · {m.user_media_meta_reference()}：{item.referenceCount} · {m.user_media_meta_uploaded_at()}：{formatDate(item.createdAt)}
								</p>
							</div>
						</div>
						<div class="flex flex-wrap items-center gap-2 sm:justify-end sm:pl-2">
							{#if isImageAsset(item) && !previewURLByAssetID[item.id] && !previewLoadingByAssetID[item.id]}
								<button
									type="button"
									class="rounded-lg border border-zinc-200 px-3 py-1.5 text-xs font-medium text-zinc-700 transition hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
									onclick={() => loadPreviewURL(item)}
								>
									{m.user_media_action_load_preview()}
								</button>
							{/if}
							{#if hasReferences(item)}
								<button
									type="button"
									class="rounded-lg border border-zinc-200 px-3 py-1.5 text-xs font-medium text-zinc-700 transition hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
									onclick={() => openReferences(item, m.user_media_delete_hint_remove_references())}
								>
									{m.user_media_action_view_references()}
								</button>
							{/if}
							{#if !hasReferences(item)}
								<button
									type="button"
									class="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-700 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-900/50 dark:text-red-300 dark:hover:bg-red-950/30"
									disabled={item.status === 'deleted'}
									title={item.status === 'deleted' ? deleteDisabledReason(item) : m.user_media_action_delete()}
									onclick={() => handleDeleteAsset(item)}
								>
									{deleteButtonLabel(item)}
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="flex items-center justify-between text-sm">
		<p class="text-zinc-500 dark:text-zinc-400">{m.user_media_total_items({ count: total })}</p>
		<div class="flex gap-2">
			<button
				type="button"
				class="rounded-lg border border-zinc-200 px-3 py-1.5 text-zinc-700 transition hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
				disabled={offset === 0 || loading}
				onclick={goPrevPage}
			>
				{m.user_media_action_previous_page()}
			</button>
			<button
				type="button"
				class="rounded-lg border border-zinc-200 px-3 py-1.5 text-zinc-700 transition hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
				disabled={!hasMore || loading}
				onclick={goNextPage}
			>
				{m.user_media_action_next_page()}
			</button>
		</div>
	</div>
</div>

{#if referencesOpen}
	<div
		use:portal
		class="fixed inset-0 z-[100] min-h-dvh w-screen overflow-y-auto bg-black/40"
		role="presentation"
		onclick={closeReferences}
	>
		<div class="flex min-h-dvh w-full items-center justify-center p-2 sm:p-4">
			<div
				class="w-full max-w-lg rounded-2xl border border-zinc-200 bg-white p-5 shadow-xl dark:border-zinc-800 dark:bg-zinc-900"
				role="dialog"
				aria-modal="true"
				tabindex="-1"
				onclick={(event) => event.stopPropagation()}
				onkeydown={(event) => {
					if (event.key === 'Escape') {
						closeReferences();
					}
				}}
			>
				<div class="flex items-start justify-between gap-3">
					<div>
						<h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{m.user_media_references_title()}</h3>
						<p class="mt-1 truncate text-xs text-zinc-500 dark:text-zinc-400">
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
					{#if referencesHint}
						<p class="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-300">
							{referencesHint}
						</p>
					{/if}
					{#if referencesLoading}
						<p class="text-sm text-zinc-500 dark:text-zinc-400">{m.user_media_loading()}</p>
					{:else if referencesError}
						<p class="text-sm text-red-600 dark:text-red-300">{referencesError}</p>
					{:else if referencesData}
						<p class="text-sm text-zinc-600 dark:text-zinc-300">{m.user_media_references_count({ count: referencesData.referenceCount })}</p>
						{#if referencesData.documents.length === 0}
							<p class="mt-3 text-sm text-zinc-500 dark:text-zinc-400">{m.user_media_references_empty()}</p>
						{:else}
							<ul class="mt-3 space-y-2">
								{#each referencesData.documents as doc (doc.documentId)}
									<li class="rounded-lg border border-zinc-200 px-3 py-2 dark:border-zinc-700">
										<div class="flex items-center justify-between gap-3">
											<div class="min-w-0">
												<p class="truncate text-sm font-medium text-zinc-900 dark:text-zinc-100">{doc.title}</p>
												<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">{formatDate(doc.updatedAt)}</p>
											</div>
											<div class="shrink-0 flex items-center gap-2">
												<button
													type="button"
													class="inline-flex items-center gap-1 rounded-md border border-zinc-200 px-2 py-1 text-xs font-medium text-zinc-700 transition hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
													onclick={() => copyDocumentID(doc.documentId)}
												>
													{#if copiedDocumentID === doc.documentId}
														<Check class="h-3.5 w-3.5" />
														{m.user_media_action_copied()}
													{:else}
														<CopySimple class="h-3.5 w-3.5" />
														{m.user_media_action_copy_id()}
													{/if}
												</button>
												<a
													href={`/edit/documents/${doc.documentId}`}
													class="text-xs font-medium text-riptide-700 hover:underline dark:text-riptide-300"
												>
													{m.user_media_action_open_document()}
												</a>
											</div>
										</div>
									</li>
								{/each}
							</ul>
						{/if}
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}
