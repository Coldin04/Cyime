<script lang="ts">
	import { tick } from 'svelte';
	import { clickOutside } from '$lib/actions/clickOutside';
	import { fade } from 'svelte/transition';
	import * as m from '$paraglide/messages';
	import Paragraph from '~icons/ph/paragraph';
	import CaretDown from '~icons/ph/caret-down';

	interface Props {
		currentValue: string;
		onSelect: (value: string) => void;
	}

	let { currentValue, onSelect }: Props = $props();

	const headingLevels = [1, 2, 3, 4, 5, 6] as const;

	let menuElement: HTMLDivElement | null = null;
	let triggerElement: HTMLButtonElement | null = null;
	let panelElement = $state<HTMLDivElement | null>(null);
	let open = $state(false);
	let panelStyle = $state('');
	const viewportMargin = 12;

	function headingLabel(level: number) {
		switch (level) {
			case 1:
				return m.editor_toolbar_heading_1();
			case 2:
				return m.editor_toolbar_heading_2();
			case 3:
				return m.editor_toolbar_heading_3();
			case 4:
				return m.editor_toolbar_heading_4();
			case 5:
				return m.editor_toolbar_heading_5();
			default:
				return m.editor_toolbar_heading_6();
		}
	}

	function handleSelect(value: string) {
		open = false;
		onSelect(value);
	}

	function closeMenu() {
		open = false;
	}

	function updatePanelPosition() {
		if (!triggerElement) return;
		const rect = triggerElement.getBoundingClientRect();
		const panelWidth = panelElement?.offsetWidth ?? 176;
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
		title={m.editor_toolbar_heading_level()}
		aria-label={m.editor_toolbar_heading_level()}
		aria-haspopup="menu"
		aria-expanded={open}
		class={`flex h-8 shrink-0 items-center gap-1.5 rounded-md border px-2 text-xs transition-colors ${
			currentValue === 'paragraph'
				? 'border-zinc-200 bg-white text-zinc-700 hover:border-zinc-300 hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:border-zinc-600 dark:hover:bg-zinc-800'
				: 'border-zinc-900 bg-zinc-900 text-white hover:bg-zinc-800 dark:border-zinc-100 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200'
		}`}
		onclick={toggleMenu}
	>
		{#if currentValue === 'paragraph'}
			<Paragraph class="h-4 w-4 shrink-0" />
		{:else}
			<span class="inline-flex min-w-6 items-center justify-center text-[11px] font-semibold">
				{currentValue.toUpperCase()}
			</span>
		{/if}
		<CaretDown class={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-180' : ''}`} />
	</button>

	{#if open}
		<div
			bind:this={panelElement}
			in:fade={{ duration: 120 }}
			out:fade={{ duration: 100 }}
			role="menu"
			style={panelStyle}
			class="z-40 min-w-[11rem] rounded-xl border border-zinc-200 bg-white p-1.5 shadow-xl shadow-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-900 dark:shadow-black/30"
		>
			<button
				type="button"
				role="menuitem"
				class={`flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition-colors ${
					currentValue === 'paragraph'
						? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'
						: 'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800'
				}`}
				onclick={() => handleSelect('paragraph')}
			>
				<Paragraph class="h-4 w-4 shrink-0" />
				<span>{m.editor_toolbar_paragraph()}</span>
			</button>
			{#each headingLevels as level}
				<button
					type="button"
					role="menuitem"
					class={`mt-1 flex w-full items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition-colors ${
						currentValue === `h${level}`
							? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900'
							: 'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800'
					}`}
					onclick={() => handleSelect(`h${level}`)}
				>
					<span class="inline-flex h-4 min-w-6 items-center justify-center text-[11px] font-semibold">
						H{level}
					</span>
					<span>{headingLabel(level)}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>
