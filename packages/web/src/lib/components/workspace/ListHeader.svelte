<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import Trash from '~icons/ph/trash';
	import * as m from '$paraglide/messages';

	let {
		allSelected,
		someSelected,
		bulkMode = false,
		selectedItemsCount = 0
	}: {
		allSelected: boolean;
		someSelected: boolean;
		bulkMode?: boolean;
		selectedItemsCount?: number;
	} = $props();

	let inputElement: HTMLInputElement;
	const dispatch = createEventDispatcher();

	const hasSelection = $derived(allSelected || someSelected);
	const checkboxClasses = $derived(
		`h-4 w-4 rounded border-zinc-300 transition-opacity dark:border-zinc-600 ${
			bulkMode || hasSelection || selectedItemsCount > 0 ? 'opacity-100' : 'opacity-0'
		}`
	);

	$effect(() => {
		if (inputElement) {
			inputElement.indeterminate = someSelected;
		}
	});

	function toggle() {
		dispatch('toggleAll');
	}

	function handleBulkDelete() {
		dispatch('bulkdelete');
	}
</script>

<div class="group border-b border-zinc-200 dark:border-zinc-700">
	<div
		class="container mx-auto flex items-center justify-between px-4 py-2 text-sm font-semibold text-zinc-500"
	>
		<div class="flex items-center gap-3">
			<input
				bind:this={inputElement}
				type="checkbox"
				class={checkboxClasses}
				class:cursor-pointer={bulkMode}
				checked={allSelected}
				onclick={toggle}
			/>
			<span>{m.common_name()}</span>
		</div>
		<div class="flex items-center justify-end gap-x-4 sm:gap-x-6">
			{#if !bulkMode}
				<div class="hidden w-28 text-right sm:block">{m.common_last_modified()}</div>
				<div class="hidden w-24 text-right md:block">{m.common_creator()}</div>
				<div class="w-10"></div>
			{/if}
		</div>
	</div>
</div>
