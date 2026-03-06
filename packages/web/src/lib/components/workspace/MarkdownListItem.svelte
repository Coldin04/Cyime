<script lang="ts">
	import FileMd from '~icons/ph/file-md';
	import type { FileItem } from '$lib/api/workspace';
	import DotsThreeVertical from '~icons/ph/dots-three-vertical';
	import Pencil from '~icons/ph/pencil';
	import Trash from '~icons/ph/trash';
	import FolderOpen from '~icons/ph/folder-open';
	import { deleteFile, updateFileName, moveFile } from '$lib/api/workspace';
	import { toast } from 'svelte-sonner';
	import MoveDialog from '$lib/components/workspace/MoveDialog.svelte';
	import * as m from '$paraglide/messages';

	let {
		item,
		selectedItems,
		bulkMode = false,
		onToggle,
		onRefresh
	}: {
		item: FileItem;
		selectedItems: { [key:string]: boolean };
		bulkMode?: boolean;
		onToggle: (id: string) => void;
		onRefresh?: () => void;
	} = $props();

	const isSelected = $derived(!!selectedItems[item.id]);
	const checkboxClasses = $derived(
		`h-4 w-4 rounded border-zinc-400 transition-opacity dark:border-zinc-600 ${
			bulkMode || isSelected ? 'opacity-100' : 'opacity-0'
		}`
	);

	let showMenu = $state(false);
	let isEditing = $state(false);
	let editingName = $state('');
	let isMoving = $state(false);
	const isMovingItem = $derived(isMoving && item.type === 'markdown');

	function formatRelativeTime(dateString: string): string {
		const date = new Date(dateString);
		const now = new Date();
		const diffInSeconds = Math.floor((now.getTime() - date.getTime()) / 1000);

		if (diffInSeconds < 60) {
			return m.time_just_now();
		} else if (diffInSeconds < 3600) {
			const minutes = Math.floor(diffInSeconds / 60);
			return m.time_minutes_ago({ minutes });
		} else if (diffInSeconds < 86400) {
			const hours = Math.floor(diffInSeconds / 3600);
			return m.time_hours_ago({ hours });
		} else if (diffInSeconds < 604800) {
			const days = Math.floor(diffInSeconds / 86400);
			return m.time_days_ago({ days });
		} else {
			return date.toLocaleDateString('zh-CN', {
				year: 'numeric',
				month: 'short',
				day: 'numeric'
			});
		}
	}

	function handleKeyDown(event: KeyboardEvent) {
		if (event.key === ' ' || event.key === 'Enter') {
			event.preventDefault();
			onToggle(item.id);
		}
	}

	function toggleMenu() {
		showMenu = !showMenu;
	}

	function closeMenu() {
		showMenu = false;
	}

	function startEditing() {
		editingName = item.title || '';
		isEditing = true;
		showMenu = false;
	}

	async function saveEditing() {
		if (!editingName.trim() || editingName === item.title) {
			isEditing = false;
			return;
		}

		try {
			await updateFileName(item.id, 'markdown', editingName.trim());
			toast.success(m.markdown_rename_success());
			onRefresh?.();
		} catch (error) {
			console.error('Failed to rename:', error);
			toast.error(m.folder_rename_failed());
		} finally {
			isEditing = false;
		}
	}

	function cancelEditing() {
		isEditing = false;
	}

	function handleEditingKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			saveEditing();
		} else if (e.key === 'Escape') {
			cancelEditing();
		}
	}

	async function handleDelete() {
		if (!confirm(m.markdown_delete_confirm())) {
			return;
		}

		try {
			await deleteFile(item.id, 'markdown');
			toast.success(m.folder_delete_success());
			onRefresh?.();
		} catch (error) {
			console.error('Failed to delete:', error);
			toast.error(m.folder_delete_failed());
		}
		showMenu = false;
	}

	function startMoving() {
		isMoving = true;
		showMenu = false;
	}

	function handleMoveComplete() {
		isMoving = false;
		onRefresh?.();
	}

	function handleMoveCancel() {
		isMoving = false;
	}
</script>

<div
	role="button"
	tabindex="0"
	class="group flex cursor-pointer items-center justify-between border-b border-zinc-200 px-4 py-3 transition-colors hover:bg-gradient-to-r hover:from-blue-50/50 hover:to-transparent dark:border-zinc-700 dark:hover:bg-zinc-800/60 {isSelected
		? 'bg-blue-50 dark:bg-blue-900/30'
		: ''}"
	onclick={() => onToggle(item.id)}
	onkeydown={handleKeyDown}
>
	<!-- Left Side: Name -->
	<div class="flex min-w-0 items-center gap-3 pr-4">
		<input
			type="checkbox"
			class={checkboxClasses}
			checked={isSelected}
			onclick={(e) => e.stopPropagation()}
			onchange={() => onToggle(item.id)}
		/>
		<FileMd class="h-5 w-5 flex-shrink-0 text-blue-500 dark:text-blue-400" />
		<a
			href="/edit/md/{item.id}"
			class="truncate font-normal text-zinc-800 dark:text-zinc-200"
			onclick={(e) => e.stopPropagation()}
		>
			{item.title}
		</a>
	</div>

	<!-- Right Side: Metadata -->
	<div class="flex flex-shrink-0 items-center justify-end gap-x-4 sm:gap-x-6">
		<div class="hidden w-28 text-right text-sm text-zinc-600 dark:text-zinc-400 sm:block">
			{formatRelativeTime(item.updatedAt)}
		</div>
		<div class="hidden w-24 text-right text-sm text-zinc-600 dark:text-zinc-400 md:block pr-0.5">
			{item.creator.displayName || 'You'}
		</div>
		<div class="relative w-10 flex justify-center">
			<button
				class="rounded-full p-2 text-zinc-500 transition-colors hover:bg-zinc-200 dark:text-zinc-400 dark:hover:bg-zinc-700"
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
					class="absolute top-full right-0 z-20 mt-1 w-40 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-zinc-800 dark:ring-zinc-700"
					onclick={(e) => e.stopPropagation()}
					onkeydown={(e) => {
						if (e.key === 'Escape') {
							closeMenu();
						}
					}}
					tabindex="-1"
				>
					<button
						onclick={startEditing}
						class="flex w-full items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
						role="menuitem"
					>
						<Pencil class="h-4 w-4" />
						<span>{m.common_rename()}</span>
					</button>
					<button
						onclick={startMoving}
						class="flex w-full items-center gap-2 px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-700"
						role="menuitem"
					>
						<FolderOpen class="h-4 w-4" />
						<span>{m.common_move_to()}</span>
					</button>
					<button
						onclick={handleDelete}
						class="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-zinc-100 dark:text-red-400 dark:hover:bg-zinc-700"
						role="menuitem"
					>
						<Trash class="h-4 w-4" />
						<span>{m.common_delete()}</span>
					</button>
				</div>
			{/if}
		</div>
	</div>
</div>

{#if isMoving && isMovingItem}
	<MoveDialog
		itemId={item.id}
		itemType={item.type}
		currentParentId={item.folderId ?? null}
		on:cancel={handleMoveCancel}
		on:move={handleMoveComplete}
	/>
{/if}

{#if isEditing}
	<div
		role="button"
		tabindex="0"
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={cancelEditing}
		onkeydown={(e) => {
			if (e.key === 'Escape' || e.key === 'Enter') {
				cancelEditing();
			}
		}}
	>
		<div
			role="presentation"
			class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-zinc-800"
			onclick={(e) => e.stopPropagation()}
		>
			<h3 class="mb-4 text-lg font-medium text-zinc-900 dark:text-zinc-100">{m.markdown_rename_title()}</h3>
			<input
				type="text"
				value={editingName}
				oninput={(e) => editingName = e.currentTarget.value}
				onkeydown={handleEditingKeydown}
				class="mb-4 w-full rounded-md border border-zinc-300 px-3 py-2 text-base text-zinc-900 focus:border-blue-500 focus:outline-none dark:border-zinc-600 dark:bg-zinc-700 dark:text-zinc-100"
				placeholder={m.markdown_name_placeholder()}
			/>
			<div class="flex justify-end gap-2">
				<button
					onclick={cancelEditing}
					class="rounded-md px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-700"
				>
					{m.common_cancel()}
				</button>
				<button
					onclick={saveEditing}
					class="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700"
				>
					{m.common_save()}
				</button>
			</div>
		</div>
	</div>
{/if}
