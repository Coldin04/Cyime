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
		class="container mx-auto flex items-center px-4 py-2 text-sm font-semibold text-zinc-500"
	>
		<div class="flex flex-1 items-center gap-3">
			<input
				bind:this={inputElement}
				type="checkbox"
				class="h-4 w-4 rounded border-zinc-300 opacity-0 transition-opacity group-hover:opacity-100 dark:border-zinc-600"
				checked={allSelected}
				onclick={toggle}
			/>
			<span>名称</span>
		</div>
		<div class="w-48 hidden sm:block">上次修改</div>
		<div class="w-32 hidden md:block">创建者</div>
	</div>
</div>
