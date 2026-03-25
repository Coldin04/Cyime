<script lang="ts">
	import { clickOutside } from '$lib/actions/clickOutside';
	import type { DocumentImageTargetOption } from '$lib/components/editor/documentImageTargets';
	import * as m from '$paraglide/messages';
	import SlidersHorizontal from '~icons/ph/sliders-horizontal';
	import Check from '~icons/ph/check';

	type Props = {
		currentTargetId: string;
		options: DocumentImageTargetOption[];
		isUpdating?: boolean;
		onSelect: (targetId: string) => void | Promise<unknown>;
	};

	let { currentTargetId, options, isUpdating = false, onSelect }: Props = $props();

	let open = $state(false);

	function closeMenu() {
		open = false;
	}

	function handleSelect(targetId: string) {
		if (isUpdating || targetId === currentTargetId) {
			closeMenu();
			return;
		}
		void onSelect(targetId);
		closeMenu();
	}
</script>

<div
	class="relative"
	use:clickOutside={{
		enabled: open,
		handler: closeMenu
	}}
>
	<button
		type="button"
		class="grid h-8 w-8 shrink-0 place-content-center rounded-full text-zinc-500 transition-colors hover:bg-black/10 hover:text-zinc-800 disabled:opacity-50 dark:text-zinc-400 dark:hover:bg-white/10 dark:hover:text-zinc-200"
		title={m.editor_topbar_image_target_settings()}
		aria-label={m.editor_topbar_image_target_settings()}
		disabled={isUpdating}
		onclick={() => (open = !open)}
	>
		<SlidersHorizontal class="h-5 w-5" />
	</button>

	{#if open}
		<div
			class="absolute top-full right-0 z-20 mt-2 w-80 rounded-2xl border border-zinc-200 bg-white p-3 shadow-xl dark:border-zinc-800 dark:bg-zinc-950"
			role="menu"
		>
			<div class="px-2 pb-2">
				<p class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
					{m.editor_image_target_menu_title()}
				</p>
				<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
					{m.editor_image_target_menu_description()}
				</p>
			</div>

			<div class="space-y-2">
				{#each options as option (option.id)}
					<button
						type="button"
						class={`flex w-full items-start justify-between gap-3 rounded-2xl border px-3 py-3 text-left transition ${
							option.id === currentTargetId
								? 'border-riptide-300 bg-riptide-50 dark:border-riptide-800 dark:bg-riptide-950/30'
								: 'border-zinc-200 bg-white hover:border-zinc-300 hover:bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-950 dark:hover:border-zinc-700 dark:hover:bg-zinc-900'
						}`}
						onclick={() => handleSelect(option.id)}
					>
						<div class="min-w-0">
							<p class="text-sm font-medium text-zinc-900 dark:text-zinc-100">{option.label}</p>
							<p class="mt-1 text-xs leading-5 text-zinc-500 dark:text-zinc-400">
								{option.description}
							</p>
						</div>
						{#if option.id === currentTargetId}
							<div class="grid h-7 w-7 shrink-0 place-content-center rounded-full bg-riptide-500 text-white">
								<Check class="h-4 w-4" />
							</div>
						{/if}
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
