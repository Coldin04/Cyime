<script lang="ts">
	import { tick } from 'svelte';
	import { clickOutside } from '$lib/actions/clickOutside';
	import { fade } from 'svelte/transition';
	import type { ExportAction } from '$lib/export/exportActions';
	import * as m from '$paraglide/messages';
	import ShareNetwork from '~icons/ph/share-network';

	interface Props {
		onAction: (action: ExportAction) => void | Promise<unknown>;
	}

	let { onAction }: Props = $props();

	let triggerElement: HTMLButtonElement | null = null;
	let panelElement = $state<HTMLDivElement | null>(null);
	let open = $state(false);
	let panelStyle = $state('');
	const viewportMargin = 12;

	const inactiveToggleClass =
		'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800';
	const iconButtonBaseClass =
		'inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md leading-none transition-colors disabled:cursor-not-allowed disabled:opacity-50';

	function closePanel() {
		open = false;
	}

	function updatePanelPosition() {
		if (!triggerElement) return;
		const rect = triggerElement.getBoundingClientRect();
		const panelWidth = panelElement?.offsetWidth ?? 200;
		const preferredLeft = rect.left + rect.width / 2 - panelWidth / 2;
		const left = Math.max(
			viewportMargin,
			Math.min(preferredLeft, window.innerWidth - panelWidth - viewportMargin)
		);
		panelStyle = `position: fixed; left: ${Math.round(left)}px; top: ${Math.round(rect.bottom + 8)}px;`;
	}

	async function togglePanel() {
		open = !open;
		if (!open) return;
		await tick();
		updatePanelPosition();
	}

	function handleAction(action: ExportAction) {
		void onAction(action);
		open = false;
	}
</script>

<div
	class="shrink-0"
	use:clickOutside={{
		enabled: open,
		handler: closePanel
	}}
>
	<button
		bind:this={triggerElement}
		type="button"
		title={m.editor_toolbar_share()}
		aria-label={m.editor_toolbar_share()}
		aria-haspopup="menu"
		aria-expanded={open}
		class={`${iconButtonBaseClass} ${inactiveToggleClass}`}
		onclick={() => void togglePanel()}
	>
		<ShareNetwork class="h-4 w-4" />
	</button>

	{#if open}
		<div
			bind:this={panelElement}
			in:fade={{ duration: 120 }}
			out:fade={{ duration: 100 }}
			style={panelStyle}
			class="z-40 rounded-xl border border-zinc-200 bg-white p-2 shadow-xl shadow-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-900 dark:shadow-black/30"
		>
			<div class="flex min-w-[13rem] flex-col gap-1">
				<button
					type="button"
					class="inline-flex items-center rounded-md px-2 py-1.5 text-left text-xs text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
					onclick={() => handleAction('download-html')}
				>
					{m.editor_export_download_html()}
				</button>
				<button
					type="button"
					class="inline-flex items-center rounded-md px-2 py-1.5 text-left text-xs text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
					onclick={() => handleAction('copy-markdown')}
				>
					{m.editor_export_copy_markdown()}
				</button>
				<button
					type="button"
					class="inline-flex items-center rounded-md px-2 py-1.5 text-left text-xs text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
					onclick={() => handleAction('download-markdown')}
				>
					{m.editor_export_download_markdown()}
				</button>
			</div>
		</div>
	{/if}
</div>
