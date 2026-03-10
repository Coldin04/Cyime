<script lang="ts">
	import { createFolder } from '$lib/api/workspace';
	import { createEventDispatcher } from 'svelte';
	import Folder from '~icons/ph/folder';
	import { onMount } from 'svelte';
	import * as m from '$paraglide/messages';
	import { toast } from 'svelte-sonner';

	let { parentId }: { parentId: string | null } = $props();

	let name = $state('');
	let isCreating = $state(false);
	let inputElement: HTMLInputElement;
	let containerElement: HTMLDivElement;

	const dispatch = createEventDispatcher();

	async function handleCreate() {
		if (!name.trim() || isCreating) return;

		isCreating = true;
		try {
			await createFolder({ name: name.trim(), parentId });
			dispatch('create');
		} catch (error) {
			console.error('创建文件夹失败:', error);
			toast.error(
				m.folder_create_failed({
					error: error instanceof Error ? error.message : m.common_unknown_error()
				})
			);
		} finally {
			isCreating = false;
		}
	}

	function handleCancel() {
		dispatch('cancel');
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			handleCreate();
		} else if (event.key === 'Escape') {
			handleCancel();
		}
	}

	function handleClickOutside(event: MouseEvent) {
		if (containerElement && !containerElement.contains(event.target as Node)) {
			handleCancel();
		}
	}

	onMount(() => {
		inputElement?.focus();
		document.addEventListener('mousedown', handleClickOutside);
		return () => {
			document.removeEventListener('mousedown', handleClickOutside);
		};
	});
</script>

<div
	bind:this={containerElement}
	class="border-b border-zinc-200 bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800/50"
>
	<div class="container mx-auto flex items-center gap-3 px-4 py-2">
		<div class="flex min-w-0 items-center gap-3 pr-4 flex-1">
			<div class="h-4 w-4"></div>
			<Folder class="h-5 w-5 flex-shrink-0 text-teal-500" />
			<input
				bind:this={inputElement}
				type="text"
				bind:value={name}
				onkeydown={handleKeydown}
				placeholder={m.new_folder_placeholder()}
				class="w-full max-w-md bg-transparent py-1 text-base text-zinc-900 placeholder-zinc-400 focus:outline-none dark:text-zinc-100"
				disabled={isCreating}
			/>
		</div>
		<div class="flex flex-shrink-0 items-center gap-2">
			<button
				onclick={handleCreate}
				disabled={!name.trim() || isCreating}
				class="rounded bg-riptide-500 px-3 py-1 text-xs font-semibold text-white hover:bg-riptide-600 disabled:cursor-not-allowed disabled:opacity-50"
			>
				{isCreating ? m.new_folder_creating() : m.common_create()}
			</button>
			<button
				onclick={handleCancel}
				class="rounded bg-zinc-200 px-3 py-1 text-xs font-semibold text-zinc-700 hover:bg-zinc-300 dark:bg-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-600"
			>
				{m.common_cancel()}
			</button>
		</div>
	</div>
</div>
