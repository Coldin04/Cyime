<script lang="ts">
	import { createFolder } from '$lib/api/workspace';
	import { createEventDispatcher } from 'svelte';
	import Folder from '~icons/ph/folder';
	import { onMount } from 'svelte';
	import * as m from '$paraglide/messages';

	let { parentId }: { parentId: string | null } = $props();

	let name = $state('');
	let isCreating = $state(false);
	let inputElement: HTMLInputElement;

	const dispatch = createEventDispatcher();

	async function handleCreate() {
		if (!name.trim() || isCreating) return;

		isCreating = true;
		try {
			await createFolder({ name: name.trim(), parentId });
			dispatch('create');
		} catch (error) {
			console.error('创建文件夹失败:', error);
			// Optionally, show an error message to the user
		} finally {
			isCreating = false;
			// The component will be removed by the parent, so no need to reset state here
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

	onMount(() => {
		inputElement?.focus();
	});
</script>

<div
	class="border-b border-zinc-200 bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800/50"
>
	<div class="container mx-auto flex items-center gap-4 px-4 py-2">
		<Folder class="h-5 w-5 flex-shrink-0 text-teal-500" />
		<div class="flex-grow">
			<input
				bind:this={inputElement}
				type="text"
				bind:value={name}
				onkeydown={handleKeydown}
				placeholder={m.new_folder_placeholder()}
				class="w-full bg-transparent py-1 text-sm text-zinc-900 placeholder-zinc-400 focus:outline-none dark:text-zinc-100"
				disabled={isCreating}
			/>
		</div>
		<div class="flex items-center gap-2">
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
