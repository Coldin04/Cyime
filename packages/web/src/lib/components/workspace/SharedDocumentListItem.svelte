<script lang="ts">
	import type { SharedDocumentItem } from '$lib/api/workspace';
	import FileText from '~icons/ph/file-text';
	import Table from '~icons/ph/table';
	import DotsThreeVertical from '~icons/ph/dots-three-vertical';
	import UsersThree from '~icons/ph/users-three';
	import SignOut from '~icons/ph/sign-out';
	import Pencil from '~icons/ph/pencil';
	import Eye from '~icons/ph/eye';
	import { goto } from '$app/navigation';
	import { clickOutside } from '$lib/actions/clickOutside';
	import * as m from '$paraglide/messages';
	import { getLocale } from '$paraglide/runtime';

	let {
		doc,
		onLeave,
		onManageMembers
	}: {
		doc: SharedDocumentItem;
		onLeave: () => void;
		onManageMembers?: () => void;
	} = $props();

	let showMenu = $state(false);

	function roleLabel(role: string) {
		switch (role) {
			case 'collaborator':
				return m.workspace_shared_role_collaborator();
			case 'editor':
				return m.workspace_shared_role_editor();
			default:
				return m.workspace_shared_role_viewer();
		}
	}

	function roleClass(role: string) {
		switch (role) {
			case 'collaborator':
				return 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/40 dark:text-cyan-300';
			case 'editor':
				return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300';
			default:
				return 'bg-zinc-200 text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300';
		}
	}

	function formatRelativeTime(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

		if (diffInSeconds < 60) {
			return m.time_just_now();
		} else if (diffInSeconds < 3600) {
			return m.time_minutes_ago({ minutes: Math.floor(diffInSeconds / 60) });
		} else if (diffInSeconds < 86400) {
			return m.time_hours_ago({ hours: Math.floor(diffInSeconds / 3600) });
		} else if (diffInSeconds < 604800) {
			return m.time_days_ago({ days: Math.floor(diffInSeconds / 86400) });
		} else {
			return date.toLocaleDateString(getLocale(), {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		}
	}

	function openDocument() {
		const targetRoute =
			doc.myRole === 'viewer'
				? `/view/documents/${doc.documentId}`
				: `/edit/documents/${doc.documentId}`;
		goto(targetRoute);
	}

	function handleClick() {
		openDocument();
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === ' ' || event.key === 'Enter') {
			event.preventDefault();
			handleClick();
		}
	}

	function toggleMenu() {
		showMenu = !showMenu;
	}

	function closeMenu() {
		showMenu = false;
	}
</script>

<div
	role="button"
	tabindex="0"
	class="group flex cursor-pointer items-center justify-between border-b border-zinc-200 px-4 py-3 transition-colors hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800/60"
	onclick={handleClick}
	onkeydown={handleKeyDown}
>
	<div class="flex min-w-0 items-start gap-3 pr-4">
		{#if doc.documentType === 'table'}
			<Table class="mt-0.5 h-5 w-5 flex-shrink-0 text-blue-500 dark:text-blue-400" />
		{:else}
			<FileText class="mt-0.5 h-5 w-5 flex-shrink-0 text-blue-500 dark:text-blue-400" />
		{/if}

		<div class="min-w-0">
			<div class="flex items-center gap-2">
				<span class="truncate font-normal text-zinc-800 dark:text-zinc-200">{doc.title}</span>
				<span class={`shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium ${roleClass(doc.myRole)}`}>
					{roleLabel(doc.myRole)}
				</span>
			</div>
			{#if doc.excerpt}
				<p class="mt-0.5 line-clamp-1 text-xs text-zinc-500 dark:text-zinc-400">{doc.excerpt}</p>
			{/if}
		</div>
	</div>

	<div class="flex flex-shrink-0 items-center justify-end gap-x-4 sm:gap-x-6">
		<div class="hidden w-28 text-right text-sm text-zinc-600 dark:text-zinc-400 sm:block">
			{formatRelativeTime(doc.updatedAt)}
		</div>
		<div class="hidden w-28 text-right text-sm text-zinc-600 dark:text-zinc-400 md:block">
			{doc.ownerDisplayName || m.workspace_shared_owner_unknown()}
		</div>

		<div
			class="relative z-10 w-10 flex justify-center"
			use:clickOutside={{
				enabled: showMenu,
				handler: closeMenu
			}}
		>
			<button
				type="button"
				class="rounded-full p-2 text-zinc-500 transition-colors hover:bg-zinc-200 dark:text-zinc-400 dark:hover:bg-zinc-700"
				title={m.common_more_options()}
				aria-label={m.common_more_options()}
				onclick={(e) => {
					e.stopPropagation();
					toggleMenu();
				}}
			>
				<DotsThreeVertical class="h-5 w-5" />
			</button>

			{#if showMenu}
				<div
					role="menu"
					class="absolute top-full right-0 z-50 mt-1 w-48 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black/5 focus:outline-none dark:bg-zinc-900 dark:ring-zinc-800"
					onclick={(e) => e.stopPropagation()}
					onkeydown={(e) => {
						if (e.key === 'Escape') closeMenu();
					}}
					tabindex="-1"
				>
					<button
						type="button"
						class="flex w-full items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
						role="menuitem"
						onclick={() => {
							closeMenu();
							openDocument();
						}}
					>
						{#if doc.myRole === 'viewer'}
							<Eye class="h-4 w-4" />
							<span>{m.common_open()}</span>
						{:else}
							<Pencil class="h-4 w-4" />
							<span>{m.common_edit()}</span>
						{/if}
					</button>

					{#if doc.myRole === 'collaborator'}
						<button
							type="button"
							class="flex w-full items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
							role="menuitem"
							onclick={() => {
								closeMenu();
								onManageMembers?.();
							}}
						>
							<UsersThree class="h-4 w-4" />
							<span>{m.workspace_shared_manage_members()}</span>
						</button>
					{/if}

					<button
						type="button"
						class="flex w-full items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
						role="menuitem"
						onclick={() => {
							closeMenu();
							onLeave();
						}}
					>
						<SignOut class="h-4 w-4" />
						<span>{m.common_leave()}</span>
					</button>
				</div>
			{/if}
		</div>
	</div>
</div>
