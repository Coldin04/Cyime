<script lang="ts">
	import { onMount } from 'svelte';
	import {
		getOutgoingSharedDocuments,
		updateDocumentPublicAccess,
		type OutgoingSharedDocumentItem
	} from '$lib/api/workspace';
	import OutgoingSharedDocumentListItem from '$lib/components/user/OutgoingSharedDocumentListItem.svelte';
	import DocumentCollaborationSettings from '$lib/components/editor/DocumentCollaborationSettings.svelte';
	import SharedDocumentListItemSkeleton from '$lib/components/workspace/SharedDocumentListItemSkeleton.svelte';
	import { portal } from '$lib/actions/portal';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';
	import X from '~icons/ph/x';
	import Copy from '~icons/ph/copy-simple';

	const PAGE_SIZE = 50;

	let items = $state<OutgoingSharedDocumentItem[]>([]);
	let hasMore = $state(false);
	let isLoading = $state(true);
	let isLoadingMore = $state(false);
	let loadError = $state('');
	let offset = $state(0);

	let manageMembersDoc = $state<OutgoingSharedDocumentItem | null>(null);
	let publicAccessDoc = $state<OutgoingSharedDocumentItem | null>(null);
	let draftPublicAccess = $state<'private' | 'authenticated' | 'public' | string>('private');
	let isSavingPublicAccess = $state(false);
	let copiedPublicURL = $state(false);

	onMount(() => {
		void loadInitial();
	});

	async function loadInitial() {
		isLoading = true;
		loadError = '';
		try {
			const data = await getOutgoingSharedDocuments({ limit: PAGE_SIZE, offset: 0 });
			items = data.items;
			hasMore = data.hasMore;
			offset = data.items.length;
		} catch (error) {
			loadError = error instanceof Error ? error.message : m.user_sharing_error_load_failed();
			items = [];
			hasMore = false;
			offset = 0;
		} finally {
			isLoading = false;
		}
	}

	async function loadMore() {
		if (isLoading || isLoadingMore || !hasMore) return;
		isLoadingMore = true;
		try {
			const data = await getOutgoingSharedDocuments({ limit: PAGE_SIZE, offset });
			items = [...items, ...data.items];
			hasMore = data.hasMore;
			offset += data.items.length;
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_sharing_error_load_more_failed());
		} finally {
			isLoadingMore = false;
		}
	}

	function openManageMembers(doc: OutgoingSharedDocumentItem) {
		manageMembersDoc = doc;
	}

	async function closeManageMembers() {
		manageMembersDoc = null;
		await loadInitial();
	}

	function openPublicAccess(doc: OutgoingSharedDocumentItem) {
		publicAccessDoc = doc;
		draftPublicAccess = doc.publicAccess;
		copiedPublicURL = false;
	}

	function closePublicAccess() {
		publicAccessDoc = null;
		copiedPublicURL = false;
	}

	function resolveAbsolutePublicURL(doc: OutgoingSharedDocumentItem) {
		try {
			return new URL(doc.publicUrl || `/view/documents/${doc.documentId}`, window.location.origin).toString();
		} catch {
			return doc.publicUrl || `/view/documents/${doc.documentId}`;
		}
	}

	async function copyPublicURL(doc: OutgoingSharedDocumentItem) {
		try {
			await navigator.clipboard.writeText(resolveAbsolutePublicURL(doc));
			copiedPublicURL = true;
			setTimeout(() => {
				copiedPublicURL = false;
			}, 1200);
			toast.success(m.user_sharing_toast_link_copied());
		} catch {
			toast.error(m.user_sharing_toast_copy_failed());
		}
	}

	async function savePublicAccess() {
		if (!publicAccessDoc || isSavingPublicAccess) {
			return;
		}

		isSavingPublicAccess = true;
		try {
			const response = await updateDocumentPublicAccess(publicAccessDoc.documentId, draftPublicAccess);
			const updatedItem = {
				...publicAccessDoc,
				publicAccess: response.publicAccess,
				publicUrl: response.publicUrl
			};

			if (response.publicAccess === 'private' && updatedItem.sharedMemberCount === 0) {
				items = items.filter((item) => item.documentId !== updatedItem.documentId);
				closePublicAccess();
			} else {
				items = items.map((item) => (item.documentId === updatedItem.documentId ? updatedItem : item));
				publicAccessDoc = updatedItem;
			}

			toast.success(response.publicAccess === 'private' ? m.user_sharing_toast_public_access_disabled() : m.user_sharing_toast_public_access_updated());
		} catch (error) {
			toast.error(error instanceof Error ? error.message : m.user_sharing_toast_public_access_update_failed());
		} finally {
			isSavingPublicAccess = false;
		}
	}

	function publicAccessLabel(access: string) {
		switch (access) {
			case 'public':
				return m.user_sharing_public_access_desc_public();
			case 'authenticated':
				return m.user_sharing_public_access_desc_authenticated();
			default:
				return m.user_sharing_public_access_desc_private();
		}
	}
</script>

<section class="space-y-6">
	<div>
		<h2 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{m.user_sharing_section_title()}</h2>
		<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
			{m.user_sharing_section_description()}
		</p>
	</div>

	<div class="border-t border-zinc-200 dark:border-zinc-700">
		{#if isLoading}
			<div>
				<SharedDocumentListItemSkeleton />
				<SharedDocumentListItemSkeleton />
				<SharedDocumentListItemSkeleton />
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
						{m.user_sharing_retry()}
					</button>
				</div>
			</div>
		{:else if items.length === 0}
			<div class="px-6 py-14 text-center">
				<h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{m.user_sharing_empty_title()}</h3>
				<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
					{m.user_sharing_empty_description()}
				</p>
			</div>
		{:else}
			<div>
				{#each items as doc (doc.documentId)}
					<OutgoingSharedDocumentListItem
						{doc}
						onManageMembers={() => openManageMembers(doc)}
						onManagePublicAccess={() => openPublicAccess(doc)}
					/>
				{/each}
			</div>

			{#if hasMore}
				<div class="border-b border-zinc-200 px-4 py-3 dark:border-zinc-700">
					<button
						type="button"
						class="w-full rounded-lg border border-zinc-200 bg-white px-4 py-2 text-sm font-medium text-zinc-700 transition hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:bg-zinc-800"
						onclick={() => void loadMore()}
						disabled={isLoadingMore}
					>
						{isLoadingMore ? m.user_sharing_loading() : m.user_sharing_load_more()}
					</button>
				</div>
			{/if}
		{/if}
	</div>
</section>

{#if manageMembersDoc}
	<div
		use:portal
		class="fixed inset-0 z-[140] flex items-center justify-center bg-black/45 p-4"
		role="presentation"
		onclick={() => void closeManageMembers()}
	>
		<div
			class="w-full max-w-2xl overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-950"
			role="dialog"
			aria-modal="true"
			aria-label={m.user_sharing_dialog_manage_members_title()}
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => {
				if (event.key === 'Escape') {
					void closeManageMembers();
				}
			}}
		>
			<header class="flex items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
				<div class="min-w-0">
					<p class="truncate text-sm font-semibold text-zinc-900 dark:text-zinc-100">{m.user_sharing_dialog_manage_members_title()}</p>
					<p class="truncate text-xs text-zinc-500 dark:text-zinc-400">{manageMembersDoc.title}</p>
				</div>
				<button
					type="button"
					class="rounded-md p-1 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
					onclick={() => void closeManageMembers()}
				>
					<X class="h-4 w-4" />
				</button>
			</header>
			<div class="p-4">
				<DocumentCollaborationSettings documentId={manageMembersDoc.documentId} enabled={true} />
			</div>
		</div>
	</div>
{/if}

{#if publicAccessDoc}
	<div
		use:portal
		class="fixed inset-0 z-[140] flex items-center justify-center bg-black/45 p-4"
		role="presentation"
		onclick={closePublicAccess}
	>
		<div
			class="w-full max-w-xl overflow-hidden rounded-xl border border-zinc-200 bg-white shadow-2xl dark:border-zinc-800 dark:bg-zinc-950"
			role="dialog"
			aria-modal="true"
			aria-label={m.user_sharing_dialog_public_access_title()}
			tabindex="-1"
			onclick={(event) => event.stopPropagation()}
			onkeydown={(event) => {
				if (event.key === 'Escape') {
					closePublicAccess();
				}
			}}
		>
			<header class="flex items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3 dark:border-zinc-800">
				<div class="min-w-0">
					<p class="truncate text-sm font-semibold text-zinc-900 dark:text-zinc-100">{m.user_sharing_dialog_public_access_title()}</p>
					<p class="truncate text-xs text-zinc-500 dark:text-zinc-400">{publicAccessDoc.title}</p>
				</div>
				<button
					type="button"
					class="rounded-md p-1 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
					onclick={closePublicAccess}
				>
					<X class="h-4 w-4" />
				</button>
			</header>
			<div class="space-y-6 p-4">
				<div class="space-y-2">
					<label for="sharing-public-access" class="text-xs font-medium text-zinc-500 dark:text-zinc-400">
						{m.user_sharing_public_access_scope_label()}
					</label>
					<select
						id="sharing-public-access"
						bind:value={draftPublicAccess}
						class="w-full rounded-lg border border-zinc-200 bg-white px-3 py-2 text-sm text-zinc-900 outline-none focus:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-zinc-500"
					>
						<option value="private">{m.user_sharing_public_access_private_option()}</option>
						<option value="authenticated">{m.user_sharing_public_access_authenticated_option()}</option>
						<option value="public">{m.user_sharing_public_access_public_option()}</option>
					</select>
					<p class="text-xs text-zinc-500 dark:text-zinc-400">{publicAccessLabel(draftPublicAccess)}</p>
				</div>

				{#if draftPublicAccess !== 'private'}
					<div class="space-y-2">
						<p class="text-xs font-medium text-zinc-500 dark:text-zinc-400">{m.user_sharing_public_link_label()}</p>
						<div class="flex items-center gap-2 rounded-xl border border-zinc-200 bg-zinc-50 p-2 dark:border-zinc-700 dark:bg-zinc-900">
							<div class="min-w-0 flex-1 truncate px-2 text-sm text-zinc-700 dark:text-zinc-200">
								{resolveAbsolutePublicURL(publicAccessDoc)}
							</div>
							<button
								type="button"
								class="inline-flex h-10 w-10 items-center justify-center rounded-lg border border-zinc-200 text-zinc-600 transition hover:bg-white dark:border-zinc-700 dark:text-zinc-300 dark:hover:bg-zinc-800"
								onclick={() => {
									if (publicAccessDoc) {
										void copyPublicURL(publicAccessDoc);
									}
								}}
								title={copiedPublicURL ? m.user_sharing_copied_tooltip() : m.user_sharing_copy_link_tooltip()}
							>
								<Copy class="h-4 w-4" />
							</button>
						</div>
					</div>
				{/if}

				<div class="flex justify-end gap-2">
					<button
						type="button"
						class="rounded-lg px-4 py-2 text-sm text-zinc-700 transition hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
						onclick={closePublicAccess}
					>
						{m.user_sharing_action_cancel()}
					</button>
					<button
						type="button"
						class="rounded-lg bg-zinc-900 px-4 py-2 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
						onclick={() => void savePublicAccess()}
						disabled={isSavingPublicAccess}
					>
						{isSavingPublicAccess ? m.user_sharing_saving() : m.user_sharing_action_save()}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}
