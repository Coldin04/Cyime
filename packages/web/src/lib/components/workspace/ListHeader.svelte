<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	let {
		allSelected,
		someSelected
	}: {
		allSelected: boolean;
		someSelected: boolean;
	} = $props();

	let inputElement: HTMLInputElement;
	const dispatch = createEventDispatcher();

	$effect(() => {
		if (inputElement) {
			inputElement.indeterminate = someSelected;
		}
	});

	function toggle() {
		dispatch('toggleAll');
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
				class="h-4 w-4 rounded border-zinc-300 opacity-0 transition-opacity group-hover:opacity-100 dark:border-zinc-600"
				checked={allSelected}
				onclick={toggle}
			/>
			<span>名称</span>
		</div>
		<div class="flex items-center justify-end gap-x-4 sm:gap-x-6">
			<div class="hidden w-28 text-right sm:block">上次修改</div>
			<div class="hidden w-24 text-right md:block">创建者</div>
			<div class="w-10" />
		</div>
	</div>
</div>
