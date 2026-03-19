<script lang="ts">
	import { tick } from 'svelte';
	import { clickOutside } from '$lib/actions/clickOutside';
	import { fade } from 'svelte/transition';
	import * as m from '$paraglide/messages';

	interface Props {
		value: string;
		onSave: (value: string) => void;
	}

	let { value, onSave }: Props = $props();

	let panelElement: HTMLDivElement | null = null;
	let triggerElement: HTMLButtonElement | null = null;
	let panelContentElement: HTMLDivElement | null = null;
	let draft = $state('');
	let open = $state(false);
	let panelStyle = $state('');
	const viewportMargin = 12;

	$effect(() => {
		draft = value;
	});

	function closeWithoutSaving() {
		draft = value;
		open = false;
	}

	function handleSave() {
		onSave(draft);
		open = false;
	}

	function updatePanelPosition() {
		if (!triggerElement) return;
		const rect = triggerElement.getBoundingClientRect();
		const panelWidth = panelContentElement?.offsetWidth ?? 240;
		const left = Math.max(
			viewportMargin,
			Math.min(rect.left, window.innerWidth - panelWidth - viewportMargin)
		);
		panelStyle = `position: fixed; left: ${Math.round(left)}px; top: ${Math.round(rect.bottom + 8)}px;`;
	}

	async function togglePanel() {
		open = !open;
		if (!open) return;
		await tick();
		updatePanelPosition();
	}
</script>

<div
	bind:this={panelElement}
	class="flex shrink-0 items-center"
	use:clickOutside={{
		enabled: open,
		handler: closeWithoutSaving
	}}
>
	<button
		bind:this={triggerElement}
		type="button"
		title={m.editor_image_title_label()}
		aria-label={m.editor_image_title_label()}
		class="inline-flex h-8 shrink-0 items-center justify-center rounded-md px-2 text-xs text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
		onclick={togglePanel}
	>
		<span class="text-[11px] font-semibold tracking-[0.02em]">Title</span>
	</button>

	{#if open}
		<div
			bind:this={panelContentElement}
			in:fade={{ duration: 120 }}
			out:fade={{ duration: 100 }}
			style={panelStyle}
			class="z-40 rounded-xl border border-zinc-200 bg-white p-2 shadow-xl shadow-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-900 dark:shadow-black/30"
		>
			<div class="flex min-w-[15rem] items-center gap-2 rounded-lg border border-zinc-200 bg-zinc-50 px-2 py-2 text-xs text-zinc-700 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-200">
				<label class="flex min-w-0 flex-1 items-center gap-2">
					<span class="shrink-0 text-zinc-500 dark:text-zinc-400">{m.editor_image_title_label()}</span>
					<input
						type="text"
						class="min-w-0 flex-1 bg-transparent text-xs outline-none placeholder:text-zinc-400 dark:placeholder:text-zinc-500"
						placeholder={m.editor_image_title_placeholder()}
						bind:value={draft}
					/>
				</label>
				<button
					type="button"
					class="inline-flex h-7 shrink-0 items-center rounded-md bg-zinc-900 px-2 text-xs font-medium text-white transition-colors hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
					onclick={handleSave}
				>
					{m.common_save()}
				</button>
			</div>
		</div>
	{/if}
</div>
