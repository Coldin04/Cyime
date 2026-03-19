<script lang="ts">
	import { tick } from 'svelte';
	import { clickOutside } from '$lib/actions/clickOutside';
	import { fade } from 'svelte/transition';
	import * as m from '$paraglide/messages';
	import Resize from '~icons/ph/resize';
	import CaretDown from '~icons/ph/caret-down';

	interface Props {
		currentWidth: string;
		onSelect: (width: string) => void;
	}

	let { currentWidth, onSelect }: Props = $props();
	let menuElement: HTMLDivElement | null = null;
	let triggerElement: HTMLButtonElement | null = null;
	let panelElement: HTMLDivElement | null = null;
	let open = $state(false);
	let panelStyle = $state('');
	const viewportMargin = 12;

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

	function closeMenu() {
		open = false;
	}

	function updatePanelPosition() {
		if (!triggerElement) return;
		const rect = triggerElement.getBoundingClientRect();
		const panelWidth = panelElement?.offsetWidth ?? 160;
		const left = Math.max(
			viewportMargin,
			Math.min(rect.left, window.innerWidth - panelWidth - viewportMargin)
		);
		panelStyle = `position: fixed; left: ${Math.round(left)}px; top: ${Math.round(rect.bottom + 8)}px;`;
	}

	async function toggleMenu() {
		open = !open;
		if (!open) return;
		await tick();
		updatePanelPosition();
	}
</script>

<div
	bind:this={menuElement}
	class="shrink-0"
	use:clickOutside={{
		enabled: open,
		handler: closeMenu
	}}
>
	<button
		bind:this={triggerElement}
		type="button"
		title={currentLabel()}
		aria-label={currentLabel()}
		aria-haspopup="menu"
		aria-expanded={open}
		class="flex h-8 shrink-0 items-center gap-1.5 rounded-md border border-zinc-200 bg-white px-2 text-xs text-zinc-700 transition-colors hover:border-zinc-300 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:border-zinc-600 dark:hover:bg-zinc-800"
		onclick={toggleMenu}
	>
		<Resize class="h-4 w-4" />
		<CaretDown class={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-180' : ''}`} />
	</button>

	{#if open}
		<div
			bind:this={panelElement}
			in:fade={{ duration: 120 }}
			out:fade={{ duration: 100 }}
			role="menu"
			style={panelStyle}
			class="z-40 min-w-[10rem] rounded-xl border border-zinc-200 bg-white p-1.5 shadow-xl shadow-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-900 dark:shadow-black/30"
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
