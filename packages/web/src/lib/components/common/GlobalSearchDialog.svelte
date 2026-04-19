<script lang="ts">
	import { goto } from '$app/navigation';
	import { searchWorkspace, type SearchDocumentItem, type SearchFolderItem, type SearchMediaItem, type WorkspaceSearchResponse } from '$lib/api/workspace';
	import { clickOutside } from '$lib/actions/clickOutside';
	import { workspaceContext } from '$lib/stores/workspace';
	import * as m from '$paraglide/messages';
	import MagnifyingGlass from '~icons/ph/magnifying-glass';
	import X from '~icons/ph/x';
	import FileText from '~icons/ph/file-text';
	import Folder from '~icons/ph/folder';
	import Image from '~icons/ph/image';
	import VideoCamera from '~icons/ph/video-camera';
	import Paperclip from '~icons/ph/paperclip';
	import ArrowBendDownLeft from '~icons/ph/arrow-bend-down-left';

	type FlatResult =
		| {
				key: string;
				group: 'documents';
				item: SearchDocumentItem;
		  }
		| {
				key: string;
				group: 'folders';
				item: SearchFolderItem;
		  }
		| {
				key: string;
				group: 'media';
				item: SearchMediaItem;
		  };

	let {
		open = false,
		onClose
	}: {
		open?: boolean;
		onClose?: () => void;
	} = $props();

	const emptyResults = (query = ''): WorkspaceSearchResponse => ({
		query,
		documents: [],
		folders: [],
		media: [],
		total: 0
	});

	let query = $state('');
	let results = $state<WorkspaceSearchResponse>(emptyResults());
	let isLoading = $state(false);
	let errorMessage = $state('');
	let activeIndex = $state(0);
	let inputEl: HTMLInputElement | null = $state(null);
	let requestToken = 0;

	const flatResults = $derived.by<FlatResult[]>(() => [
		...results.documents.map((item) => ({ key: `document:${item.id}`, group: 'documents' as const, item })),
		...results.folders.map((item) => ({ key: `folder:${item.id}`, group: 'folders' as const, item })),
		...results.media.map((item) => ({ key: `media:${item.id}`, group: 'media' as const, item }))
	]);

	$effect(() => {
		if (!open) {
			query = '';
			results = emptyResults();
			errorMessage = '';
			isLoading = false;
			activeIndex = 0;
			return;
		}

		setTimeout(() => inputEl?.focus(), 0);
	});

	$effect(() => {
		if (!open) {
			return;
		}

		const trimmed = query.trim();
		errorMessage = '';

		if (!trimmed) {
			results = emptyResults();
			isLoading = false;
			activeIndex = 0;
			return;
		}

		const currentToken = ++requestToken;
		const timer = setTimeout(async () => {
			isLoading = true;
			try {
				const next = await searchWorkspace({ q: trimmed, limit: 5 });
				if (currentToken !== requestToken) return;
				results = next;
				activeIndex = 0;
			} catch (error) {
				if (currentToken !== requestToken) return;
				results = emptyResults(trimmed);
				errorMessage = error instanceof Error ? error.message : 'Search failed';
			} finally {
				if (currentToken === requestToken) {
					isLoading = false;
				}
			}
		}, 120);

		return () => clearTimeout(timer);
	});

	$effect(() => {
		if (flatResults.length === 0) {
			activeIndex = 0;
			return;
		}
		if (activeIndex < 0) {
			activeIndex = 0;
			return;
		}
		if (activeIndex >= flatResults.length) {
			activeIndex = flatResults.length - 1;
		}
	});

	function closeDialog() {
		onClose?.();
	}

	function formatDocumentSubtitle(item: SearchDocumentItem): string {
		return item.excerpt?.trim() || item.myRole || item.documentType;
	}

	function formatFolderSubtitle(): string {
		return m.global_search_open_folder();
	}

	function formatMediaSubtitle(item: SearchMediaItem): string {
		return item.documentTitle?.trim() || item.mimeType;
	}

	function escapeHtml(value: string): string {
		return value
			.replaceAll('&', '&amp;')
			.replaceAll('<', '&lt;')
			.replaceAll('>', '&gt;')
			.replaceAll('"', '&quot;')
			.replaceAll("'", '&#39;');
	}

	function escapeRegExp(value: string): string {
		return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
	}

	function highlightMatch(value: string, selected: boolean): string {
		const text = value.trim();
		const keyword = query.trim();
		if (!text) {
			return '';
		}
		if (!keyword) {
			return escapeHtml(text);
		}

		const pattern = new RegExp(`(${escapeRegExp(keyword)})`, 'gi');
		const markClass = selected
			? 'rounded bg-sky-200 px-0.5 text-sky-900 dark:bg-sky-900/60 dark:text-sky-100'
			: 'rounded bg-sky-100 px-0.5 text-sky-700 dark:bg-sky-950/70 dark:text-sky-300';

		return escapeHtml(text).replace(pattern, `<mark class="${markClass}">$1</mark>`);
	}

	async function selectResult(entry: FlatResult) {
		closeDialog();

		if (entry.group === 'documents') {
			await goto(`/edit/documents/${entry.item.id}`);
			return;
		}

		if (entry.group === 'folders') {
			workspaceContext.update((ctx) => ({
				...ctx,
				currentFolderId: entry.item.id,
				bulkMode: false
			}));
			await goto('/workspace');
			return;
		}

		if (entry.item.documentId) {
			await goto(`/edit/documents/${entry.item.documentId}`);
			return;
		}

		await goto('/user/media');
	}

	function handleKeydown(event: KeyboardEvent) {
		if (!open) {
			return;
		}
		if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
			event.preventDefault();
			return;
		}
		if (event.key === 'Escape') {
			event.preventDefault();
			closeDialog();
			return;
		}
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			if (flatResults.length > 0) {
				activeIndex = (activeIndex + 1) % flatResults.length;
			}
			return;
		}
		if (event.key === 'ArrowUp') {
			event.preventDefault();
			if (flatResults.length > 0) {
				activeIndex = (activeIndex - 1 + flatResults.length) % flatResults.length;
			}
			return;
		}
		if (event.key === 'Enter') {
			const target = flatResults[activeIndex];
			if (!target) {
				return;
			}
			event.preventDefault();
			void selectResult(target);
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div class="fixed inset-0 z-[110] flex items-start justify-center bg-black/55 p-3 pt-[8vh] backdrop-blur-[1px] sm:p-6">
		<div
			role="dialog"
			aria-modal="true"
			aria-label={m.common_search_placeholder()}
			tabindex="-1"
			class="w-full max-w-4xl overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-700 dark:bg-zinc-900"
			use:clickOutside={{
				enabled: open,
				handler: closeDialog
			}}
		>
			<div class="border-b border-zinc-200 p-3 sm:p-4 dark:border-zinc-700">
				<div class="flex items-center gap-2.5 rounded-lg border-2 border-sky-500/80 bg-white px-3 py-2 shadow-sm dark:bg-zinc-950">
					<MagnifyingGlass class="h-5 w-5 shrink-0 text-sky-500" />
					<input
						bind:this={inputEl}
						bind:value={query}
						type="text"
						autocomplete="off"
						spellcheck="false"
						placeholder={m.global_search_input_placeholder()}
						class="min-w-0 flex-1 bg-transparent text-base text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100"
					/>
					<button
						type="button"
						class="grid h-8 w-8 shrink-0 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
						aria-label={m.global_search_close()}
						onclick={closeDialog}
					>
						<X class="h-5 w-5" />
					</button>
				</div>
			</div>

			<div class="max-h-[65vh] overflow-y-auto px-3 pb-4 pt-3 sm:px-5">
				{#if errorMessage}
					<p class="px-2 py-3 text-sm text-red-600 dark:text-red-400">{errorMessage}</p>
				{:else if !query.trim()}
					<p class="px-2 py-3 text-sm text-zinc-500 dark:text-zinc-400">
						{m.global_search_hint_idle()}
					</p>
				{:else if isLoading}
					<p class="px-2 py-3 text-sm text-zinc-500 dark:text-zinc-400">{m.global_search_loading()}</p>
				{:else if flatResults.length === 0}
					<p class="px-2 py-3 text-sm text-zinc-500 dark:text-zinc-400">{m.global_search_empty()}</p>
				{:else}
					{#if results.documents.length > 0}
						<section class="mb-4">
							<h3 class="px-2 pb-2 text-sm font-semibold text-sky-600 dark:text-sky-400">{m.global_search_group_documents()}</h3>
							<div class="space-y-2">
								{#each results.documents as item}
									<button
										type="button"
										class={`flex w-full items-center gap-3 rounded-lg px-4 py-3 text-left shadow-sm ring-1 ring-inset transition ${
											flatResults[activeIndex]?.key === `document:${item.id}`
												? 'bg-sky-100 text-zinc-900 ring-sky-200 dark:bg-sky-950/40 dark:text-zinc-100 dark:ring-sky-900/60'
												: 'bg-white text-zinc-900 ring-zinc-200 hover:bg-zinc-50 dark:bg-zinc-950 dark:text-zinc-100 dark:ring-zinc-800 dark:hover:bg-zinc-900'
										}`}
										onmousemove={() => {
											const index = flatResults.findIndex((entry) => entry.key === `document:${item.id}`);
											if (index >= 0) activeIndex = index;
										}}
										onclick={() =>
											void selectResult({
												key: `document:${item.id}`,
												group: 'documents',
												item
											})}
									>
										<FileText class="h-5 w-5 shrink-0" />
										<div class="min-w-0 flex-1">
											<p class="truncate text-base">
												{@html highlightMatch(item.title, flatResults[activeIndex]?.key === `document:${item.id}`)}
											</p>
											<p class={`truncate text-sm ${flatResults[activeIndex]?.key === `document:${item.id}` ? 'text-zinc-600 dark:text-zinc-300' : 'text-zinc-500 dark:text-zinc-400'}`}>
												{@html highlightMatch(formatDocumentSubtitle(item), flatResults[activeIndex]?.key === `document:${item.id}`)}
											</p>
										</div>
										<ArrowBendDownLeft class="h-5 w-5 shrink-0 opacity-80" />
									</button>
								{/each}
							</div>
						</section>
					{/if}

					{#if results.folders.length > 0}
						<section class="mb-4">
							<h3 class="px-2 pb-2 text-sm font-semibold text-sky-600 dark:text-sky-400">{m.global_search_group_folders()}</h3>
							<div class="space-y-2">
								{#each results.folders as item}
									<button
										type="button"
										class={`flex w-full items-center gap-3 rounded-lg px-4 py-3 text-left shadow-sm ring-1 ring-inset transition ${
											flatResults[activeIndex]?.key === `folder:${item.id}`
												? 'bg-sky-100 text-zinc-900 ring-sky-200 dark:bg-sky-950/40 dark:text-zinc-100 dark:ring-sky-900/60'
												: 'bg-white text-zinc-900 ring-zinc-200 hover:bg-zinc-50 dark:bg-zinc-950 dark:text-zinc-100 dark:ring-zinc-800 dark:hover:bg-zinc-900'
										}`}
										onmousemove={() => {
											const index = flatResults.findIndex((entry) => entry.key === `folder:${item.id}`);
											if (index >= 0) activeIndex = index;
										}}
										onclick={() =>
											void selectResult({
												key: `folder:${item.id}`,
												group: 'folders',
												item
											})}
									>
										<Folder class="h-5 w-5 shrink-0" />
										<div class="min-w-0 flex-1">
											<p class="truncate text-base">
												{@html highlightMatch(item.name, flatResults[activeIndex]?.key === `folder:${item.id}`)}
											</p>
											<p class={`truncate text-sm ${flatResults[activeIndex]?.key === `folder:${item.id}` ? 'text-zinc-600 dark:text-zinc-300' : 'text-zinc-500 dark:text-zinc-400'}`}>
												{@html highlightMatch(formatFolderSubtitle(), flatResults[activeIndex]?.key === `folder:${item.id}`)}
											</p>
										</div>
										<ArrowBendDownLeft class="h-5 w-5 shrink-0 opacity-80" />
									</button>
								{/each}
							</div>
						</section>
					{/if}

					{#if results.media.length > 0}
						<section>
							<h3 class="px-2 pb-2 text-sm font-semibold text-sky-600 dark:text-sky-400">{m.global_search_group_media()}</h3>
							<div class="space-y-2">
								{#each results.media as item}
									<button
										type="button"
										class={`flex w-full items-center gap-3 rounded-lg px-4 py-3 text-left shadow-sm ring-1 ring-inset transition ${
											flatResults[activeIndex]?.key === `media:${item.id}`
												? 'bg-sky-100 text-zinc-900 ring-sky-200 dark:bg-sky-950/40 dark:text-zinc-100 dark:ring-sky-900/60'
												: 'bg-white text-zinc-900 ring-zinc-200 hover:bg-zinc-50 dark:bg-zinc-950 dark:text-zinc-100 dark:ring-zinc-800 dark:hover:bg-zinc-900'
										}`}
										onmousemove={() => {
											const index = flatResults.findIndex((entry) => entry.key === `media:${item.id}`);
											if (index >= 0) activeIndex = index;
										}}
										onclick={() =>
											void selectResult({
												key: `media:${item.id}`,
												group: 'media',
												item
											})}
									>
										{#if item.kind === 'image'}
											<Image class="h-5 w-5 shrink-0" />
										{:else if item.kind === 'video'}
											<VideoCamera class="h-5 w-5 shrink-0" />
										{:else}
											<Paperclip class="h-5 w-5 shrink-0" />
										{/if}
										<div class="min-w-0 flex-1">
											<p class="truncate text-base">
												{@html highlightMatch(item.filename, flatResults[activeIndex]?.key === `media:${item.id}`)}
											</p>
											<p class={`truncate text-sm ${flatResults[activeIndex]?.key === `media:${item.id}` ? 'text-zinc-600 dark:text-zinc-300' : 'text-zinc-500 dark:text-zinc-400'}`}>
												{@html highlightMatch(formatMediaSubtitle(item), flatResults[activeIndex]?.key === `media:${item.id}`)}
											</p>
										</div>
										<ArrowBendDownLeft class="h-5 w-5 shrink-0 opacity-80" />
									</button>
								{/each}
							</div>
						</section>
					{/if}
				{/if}
			</div>

			<div class="flex items-center justify-between border-t border-zinc-200 px-4 py-3 text-xs text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
				<div class="flex items-center gap-2">
					<span>{m.global_search_footer_select()}</span>
					<span>{m.global_search_footer_navigate()}</span>
				</div>
				<span>{m.global_search_footer_close()}</span>
			</div>
		</div>
	</div>
{/if}
