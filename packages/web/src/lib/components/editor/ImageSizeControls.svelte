<script lang="ts">
	import { onMount } from 'svelte';
	import * as m from '$paraglide/messages';
	import Resize from '~icons/ph/resize';
	import CaretDown from '~icons/ph/caret-down';

	interface Props {
		currentWidth: string;
		onSelect: (width: string) => void;
	}

	let { currentWidth, onSelect }: Props = $props();
	let menuElement: HTMLDivElement | null = null;
	let open = $state(false);

	const options = [
		{ value: 'auto', title: m.editor_image_size_auto() },
		{ value: '40%', title: m.editor_image_size_small() },
		{ value: '60%', title: m.editor_image_size_medium() },
		{ value: '80%', title: m.editor_image_size_large() },
		{ value: '100%', title: m.editor_image_size_full() }
	] as const;

	function currentLabel() {
		return options.find((option) => option.value === currentWidth)?.title || m.editor_image_size_auto();
	}

	function handleSelect(width: string) {
		open = false;
		onSelect(width);
	}

	onMount(() => {
		const handlePointerDown = (event: PointerEvent) => {
			if (!open || !menuElement) return;
			const target = event.target;
			if (target instanceof Node && menuElement.contains(target)) return;
			open = false;
		};

		document.addEventListener('pointerdown', handlePointerDown);
		return () => {
			document.removeEventListener('pointerdown', handlePointerDown);
		};
	});
</script>

<div bind:this={menuElement} class="relative shrink-0">
	<button
		type="button"
		title={currentLabel()}
		aria-label={currentLabel()}
		aria-haspopup="menu"
		aria-expanded={open}
		class="flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-zinc-200 bg-white px-2 text-xs text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:border-zinc-600 dark:hover:bg-zinc-800"
		onclick={() => {
			open = !open;
		}}
	>
		<Resize class="h-4 w-4" />
		<CaretDown class={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-180' : ''}`} />
	</button>

	{#if open}
		<div
			role="menu"
			class="absolute left-0 top-[calc(100%+0.4rem)] z-20 min-w-[10rem] rounded-xl border border-zinc-200 bg-white p-1.5 shadow-xl shadow-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-900 dark:shadow-black/30"
		>
			{#each options as option}
				<button
					type="button"
					role="menuitem"
					class={`flex w-full items-center rounded-lg px-2.5 py-2 text-left text-sm transition-colors ${
						currentWidth === option.value
							? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'
							: 'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800'
					}`}
					onclick={() => handleSelect(option.value)}
				>
					{option.title}
				</button>
			{/each}
		</div>
	{/if}
</div>
