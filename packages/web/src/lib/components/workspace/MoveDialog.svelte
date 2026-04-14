<script lang="ts">
	import { batchMoveFiles, getAllFolders, type FileItem } from '$lib/api/workspace';
	import { createEventDispatcher } from 'svelte';
	import Folder from '~icons/ph/folder';
	import { toast } from 'svelte-sonner';
	import * as m from '$paraglide/messages';

	type ItemToMove = {
		id: string;
		type: 'folder' | 'document';
	};

	let {
		items = []
	}: {
		items: ItemToMove[];
	} = $props();

	const dispatch = createEventDispatcher();

	let isMoving = $state(false);
	let selectedFolderId = $state<string | null>(null);
	let allFolders = $state<FileItem[]>([]);
	let isLoadingFolders = $state(true);

	// Get all folders on mount
	$effect(() => {
		(async () => {
			try {
				isLoadingFolders = true;
				const folders = await getAllFolders({});

				// If moving folders, filter out themselves and all their descendants from the possible destinations
				const foldersToMove = items.filter((i) => i.type === 'folder');
				if (foldersToMove.length > 0) {
					const foldersToExclude = new Set<string>();
					const queue: string[] = [];

					// Add all folders being moved to the exclusion list and the initial queue
					for (const folder of foldersToMove) {
						foldersToExclude.add(folder.id);
						queue.push(folder.id);
					}

					// Create a map for efficient child lookup
					const parentToChildrenMap = new Map<string, string[]>();
					for (const f of folders) {
						if (f.parentId) {
							if (!parentToChildrenMap.has(f.parentId)) {
								parentToChildrenMap.set(f.parentId, []);
							}
							parentToChildrenMap.get(f.parentId)!.push(f.id);
						}
					}

					// BFS to find all descendants of all folders being moved
					let head = 0;
					while (head < queue.length) {
						const currentId = queue[head++];
						const children = parentToChildrenMap.get(currentId) || [];
						for (const childId of children) {
							if (!foldersToExclude.has(childId)) {
								foldersToExclude.add(childId);
								queue.push(childId);
							}
						}
					}

					allFolders = folders.filter((f) => !foldersToExclude.has(f.id));
				} else {
					allFolders = folders;
				}
			} catch (error) {
				console.error('Failed to load folders:', error);
				toast.error(m.move_dialog_load_failed());
			} finally {
				isLoadingFolders = false;
			}
		})();
	});

	// Build folder tree for display
	type FolderTreeNode = {
		id: string;
		name: string;
		children: FolderTreeNode[];
		level: number;
	};

	function buildFolderTree(folders: FileItem[]): FolderTreeNode[] {
		const folderMap = new Map<string, FolderTreeNode>();
		const roots: FolderTreeNode[] = [];

		// Initialize all folders
		folders.forEach((f) => {
			folderMap.set(f.id, {
				id: f.id,
				name: f.name,
				children: [],
				level: 0
			});
		});

		// Build tree
		folders.forEach((f) => {
			const node = folderMap.get(f.id)!;
			if (f.parentId) {
				const parent = folderMap.get(f.parentId);
				if (parent) {
					parent.children.push(node);
				} else {
					roots.push(node);
				}
			} else {
				roots.push(node);
			}
		});

		// Calculate levels
		function setLevels(nodes: FolderTreeNode[], level: number) {
			nodes.forEach((node) => {
				node.level = level;
				setLevels(node.children, level + 1);
			});
		}
		setLevels(roots, 0);

		return roots;
	}

	const folderTree = $derived(buildFolderTree(allFolders));

	function flattenTree(nodes: FolderTreeNode[], result: FolderTreeNode[] = []): FolderTreeNode[] {
		nodes.forEach((node) => {
			result.push(node);
			flattenTree(node.children, result);
		});
		return result;
	}

	const flatFolders = $derived(flattenTree(folderTree));

	function handleCancel() {
		dispatch('cancel');
	}

	async function handleMove() {
		if (isMoving || items.length === 0) return;

		isMoving = true;
		try {
			const result = await batchMoveFiles(items, selectedFolderId);

			if (result.success) {
				toast.success(result.message || m.move_dialog_success());
			} else {
				// Show a generic error message, with details for failed items
				let errorMessage = result.message || m.move_dialog_failed();
				if (result.failedItems && result.failedItems.length > 0) {
					const failedReasons = result.failedItems.map((item) => item.reason).join(', ');
					errorMessage += `: ${failedReasons}`;
				}
				toast.error(errorMessage);
			}

			dispatch('move', { targetId: selectedFolderId });
		} catch (error: any) {
			console.error('Failed to move:', error);
			toast.error(error.message || m.move_dialog_failed());
		} finally {
			isMoving = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			handleCancel();
		}
	}
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<div
	role="button"
	tabindex="0"
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
	onclick={handleCancel}
	onkeydown={(e) => {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			handleCancel();
		}
		handleKeydown(e);
	}}
>
	<div
		role="dialog"
		aria-modal="true"
		aria-labelledby="move-dialog-title"
		tabindex="-1"
		class="w-full max-w-md rounded-lg bg-white p-6 shadow-xl dark:bg-zinc-800"
		onclick={(e) => e.stopPropagation()}
		onkeydown={(e) => e.stopPropagation()}
	>
		<h3 id="move-dialog-title" class="mb-4 text-lg font-medium text-zinc-900 dark:text-zinc-100">
			{m.move_dialog_title()}
		</h3>

		{#if isLoadingFolders}
			<div class="mb-4 flex items-center justify-center py-8">
				<div class="h-6 w-6 animate-spin rounded-full border-2 border-zinc-300 border-t-blue-500"></div>
			</div>
		{:else if flatFolders.length === 0}
			<div class="mb-4 flex flex-col items-center justify-center py-8 text-center">
				<Folder class="mb-2 h-8 w-8 text-zinc-400" />
				<p class="text-sm text-zinc-500 dark:text-zinc-400">{m.move_dialog_no_folders()}</p>
			</div>
		{:else}
			<div class="mb-4 max-h-64 overflow-y-auto rounded-md border border-zinc-200 dark:border-zinc-700">
				<!-- Root option -->
				<button
					type="button"
					class="flex w-full items-center gap-2 border-b border-zinc-100 px-4 py-2 text-left text-sm text-zinc-800 transition-colors hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-700 {selectedFolderId ===
					null
						? 'bg-blue-50 dark:bg-blue-900/30'
						: ''}"
					onclick={() => (selectedFolderId = null)}
					onkeydown={(e) => {
						if (e.key === 'Enter' || e.key === ' ') {
							e.preventDefault();
							selectedFolderId = null;
						}
					}}
				>
					<Folder class="h-4 w-4 flex-shrink-0 text-zinc-500" />
					<span class="truncate">{m.move_dialog_root_folder()}</span>
				</button>

				{#each flatFolders as folder (folder.id)}
					<button
						type="button"
						class="flex w-full items-center gap-2 border-b border-zinc-100 px-4 py-2 text-left text-sm text-zinc-800 transition-colors hover:bg-zinc-50 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-700 {selectedFolderId ===
						folder.id
							? 'bg-blue-50 dark:bg-blue-900/30'
							: ''}"
						onclick={() => (selectedFolderId = folder.id)}
						onkeydown={(e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								e.preventDefault();
								selectedFolderId = folder.id;
							}
						}}
						style="padding-left: {1 + folder.level * 1.5}rem"
					>
						<Folder class="h-4 w-4 flex-shrink-0 text-sky-500" />
						<span class="truncate">{folder.name}</span>
					</button>
				{/each}
			</div>
		{/if}

		<div class="flex justify-end gap-2">
			<button
				type="button"
				onclick={handleCancel}
				class="rounded-md px-4 py-2 text-sm text-zinc-700 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-700"
			>
				{m.common_cancel()}
			</button>
			<button
				type="button"
				onclick={handleMove}
				disabled={isMoving || (isLoadingFolders || (flatFolders.length === 0 && selectedFolderId === null))}
				class="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
			>
				{isMoving ? m.move_dialog_moving() : m.common_move()}
			</button>
		</div>
	</div>
</div>
