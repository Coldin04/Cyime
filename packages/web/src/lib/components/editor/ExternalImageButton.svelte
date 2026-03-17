<script lang="ts">
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import * as m from '$paraglide/messages';
	import LinkSimple from '~icons/ph/link-simple';

	interface Props {
		onInsert: (src: string) => boolean;
	}

	let { onInsert }: Props = $props();

	let panelElement: HTMLDivElement | null = null;
	let draft = $state('');
	let open = $state(false);

	function closePanel() {
		draft = '';
		open = false;
	}

	function handleInsert() {
		const inserted = onInsert(draft);
		if (inserted) {
			closePanel();
		}
	}

	onMount(() => {
		const handlePointerDown = (event: PointerEvent) => {
			if (!open || !panelElement) return;
			const target = event.target;
			if (target instanceof Node && panelElement.contains(target)) return;
			closePanel();
		};

		document.addEventListener('pointerdown', handlePointerDown);
		return () => {
			document.removeEventListener('pointerdown', handlePointerDown);
		};
	});
</script>

<div bind:this={panelElement} class="relative flex shrink-0 items-center">
	<button
		type="button"
		title={m.editor_toolbar_insert_external_image()}
		aria-label={m.editor_toolbar_insert_external_image()}
		class="inline-flex h-8 shrink-0 items-center justify-center rounded-md px-2 text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
		onclick={() => {
			open = !open;
		}}
	>
		<LinkSimple class="h-4 w-4" />
	</button>

	{#if open}
		<div
			in:fade={{ duration: 120 }}
			out:fade={{ duration: 100 }}
			class="absolute left-0 top-[calc(100%+0.4rem)] z-20 rounded-xl border border-zinc-200 bg-white p-2 shadow-xl shadow-zinc-900/10 dark:border-zinc-700 dark:bg-zinc-900 dark:shadow-black/30"
		>
			<div class="flex min-w-[16rem] items-center gap-2 rounded-lg border border-zinc-200 bg-zinc-50 px-2 py-2 text-xs text-zinc-700 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-200">
				<input
					type="url"
					class="min-w-0 flex-1 bg-transparent text-xs outline-none placeholder:text-zinc-400 dark:placeholder:text-zinc-500"
					placeholder={m.editor_external_image_placeholder()}
					bind:value={draft}
				/>
				<button
					type="button"
					class="inline-flex h-7 shrink-0 items-center rounded-md bg-zinc-900 px-2 text-xs font-medium text-white transition-colors hover:bg-zinc-800 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
					onclick={handleInsert}
				>
					{m.editor_external_image_insert()}
				</button>
			</div>
		</div>
	{/if}
</div>
