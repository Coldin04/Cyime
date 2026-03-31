<script lang="ts">
	import CaretLeft from '~icons/ph/caret-left';
	import UsersThree from '~icons/ph/users-three';
	import * as m from '$paraglide/messages';
	import { toast } from 'svelte-sonner';
	import {
		getSharedDocuments,
		leaveSharedDocument,
		type SharedDocumentItem
	} from '$lib/api/workspace';
	import SharedDocumentListItem from '$lib/components/workspace/SharedDocumentListItem.svelte';
	import DocumentListItemSkeleton from '$lib/components/workspace/DocumentListItemSkeleton.svelte';
	import ListHeader from '$lib/components/workspace/ListHeader.svelte';
	import DocumentCollaborationSettings from '$lib/components/editor/DocumentCollaborationSettings.svelte';
	import X from '~icons/ph/x';
	import { portal } from '$lib/actions/portal';

	const PAGE_SIZE = 50;

	let items = $state<SharedDocumentItem[]>([]);
	let hasMore = $state(false);
	let isLoading = $state(true);
	let isLoadingMore = $state(false);
	let loadError = $state('');
	let offset = $state(0);

	let manageMembersDoc = $state<{ id: string; title: string } | null>(null);

	$effect(() => {
		void loadInitial();
	});

	async function loadInitial() {
		isLoading = true;
		loadError = '';
		try {
			const data = await getSharedDocuments({ limit: PAGE_SIZE, offset: 0 });
			items = data.items;
			hasMore = data.hasMore;
			offset = data.items.length;
		} catch (error) {
			loadError = error instanceof Error ? error.message : m.common_unknown_error();
			items = [];
			hasMore = false;
			offset = 0;
		} finally {
			isLoading = false;
		}
	}

	async function loadMore() {
		if (isLoadingMore || isLoading || !hasMore) return;
		isLoadingMore = true;
		try {
			const data = await getSharedDocuments({ limit: PAGE_SIZE, offset });
			items = [...items, ...data.items];
			hasMore = data.hasMore;
			offset += data.items.length;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.common_unknown_error());
		} finally {
			isLoadingMore = false;
		}
	}

	async function handleLeave(doc: SharedDocumentItem) {
		const ok = window.confirm(m.workspace_shared_leave_confirm({ title: doc.title }));
		if (!ok) return;

		try {
			await leaveSharedDocument(doc.documentId);
			items = items.filter((it) => it.documentId !== doc.documentId);
			toast.success(m.workspace_shared_leave_success());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.workspace_shared_leave_failed());
		}
	}
</script>

<svelte:head>
	<title>{m.page_title_workspace_shared()}</title>
</svelte:head>

<div class="space-y-6">
	<header class="space-y-3">
		<a
			href="/workspace"
			class="inline-flex items-center gap-2 rounded-md px-2 py-1 text-sm font-medium text-zinc-600 transition hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
		>
			<CaretLeft class="h-4 w-4" />
			<span>{m.topbar_back_to_workspace()}</span>
		</a>

		<div class="flex items-start gap-3">
			<div class="grid h-10 w-10 place-content-center rounded-xl bg-teal-50 text-teal-600 dark:bg-teal-900/30 dark:text-teal-300">
				<UsersThree class="h-5 w-5" />
			</div>
			<div class="min-w-0">
				<h1 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
					{m.workspace_shared_page_title()}
				</h1>
				<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
					{m.workspace_shared_page_description()}
				</p>
			</div>
		</div>
	</header>

	<section class="my-6 border-t border-zinc-200 dark:border-zinc-700">
		<ListHeader
			allSelected={false}
			someSelected={false}
			bulkMode={false}
			selectedItemsCount={0}
			on:toggleAll={() => {}}
			on:bulkdelete={() => {}}
		/>

		{#if isLoading}
			<div>
				<DocumentListItemSkeleton />
				<DocumentListItemSkeleton />
				<DocumentListItemSkeleton />
			</div>
		{:else if loadError}
			<div class="p-4">
				<p class="text-sm text-rose-600 dark:text-rose-300">{loadError}</p>
				<div class="mt-3">
					<button
						type="button"
						class="rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-700 dark:border-zinc-700 dark:text-zinc-300"
						onclick={() => void loadInitial()}
					>
						{m.workspace_shared_retry()}
					</button>
				</div>
			</div>
		{:else if items.length === 0}
			<div class="flex flex-col items-center justify-center px-6 py-14 text-center">
				<div class="mb-4 grid h-12 w-12 place-content-center rounded-2xl bg-zinc-100 text-zinc-400 dark:bg-zinc-900 dark:text-zinc-500">
					<UsersThree class="h-6 w-6" />
				</div>
				<h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
					{m.workspace_shared_empty_title()}
				</h3>
				<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
					{m.workspace_shared_empty_description()}
				</p>
			</div>
		{:else}
			<div>
				{#each items as doc (doc.documentId)}
					<SharedDocumentListItem
						{doc}
						onLeave={() => void handleLeave(doc)}
						onManageMembers={() => (manageMembersDoc = { id: doc.documentId, title: doc.title })}
					/>
				{/each}
			</div>

			{#if hasMore}
				<div class="border-t border-zinc-200 px-4 py-3 dark:border-zinc-700">
					<button
						type="button"
						class="w-full rounded-lg border border-zinc-200 bg-white px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:bg-zinc-800"
						onclick={() => void loadMore()}
						disabled={isLoadingMore}
					>
						{isLoadingMore ? m.workspace_shared_loading_more() : m.workspace_shared_load_more()}
					</button>
				</div>
			{/if}
		{/if}
	</section>
</div>

{#if manageMembersDoc}
	<div
		use:portal
		class="fixed inset-0 z-[140] flex items-center justify-center bg-black/45 p-4"
		role="presentation"
		onclick={() => (manageMembersDoc = null)}
	>
		<div
			class="w-full max-w-2xl overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-950"
			role="dialog"
			aria-modal="true"
			aria-label={m.workspace_shared_manage_members()}
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => {
				if (event.key === 'Escape') manageMembersDoc = null;
			}}
		>
			<header class="flex items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
				<div class="min-w-0">
					<p class="truncate text-sm font-semibold text-zinc-900 dark:text-zinc-100">
						{m.workspace_shared_manage_members()}
					</p>
					<p class="truncate text-xs text-zinc-500 dark:text-zinc-400">{manageMembersDoc.title}</p>
				</div>
				<button
					type="button"
					class="rounded-md p-1 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
					onclick={() => (manageMembersDoc = null)}
				>
					<X class="h-4 w-4" />
				</button>
			</header>
			<div class="p-4">
				<DocumentCollaborationSettings documentId={manageMembersDoc.id} enabled={true} />
			</div>
		</div>
	</div>
{/if}
