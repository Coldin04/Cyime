<script lang="ts">
	import { clickOutside } from '$lib/actions/clickOutside';
	import type { DocumentImageTargetOption } from '$lib/components/editor/documentImageTargets';
	import * as m from '$paraglide/messages';

	let {
		open = false,
		imageCount = 0,
		targetOptions = [],
		selectedTargetId = '',
		busy = false,
		onTargetChange,
		onCancel,
		onSaveAs,
		onReplace
	}: {
		open?: boolean;
		imageCount?: number;
		targetOptions?: DocumentImageTargetOption[];
		selectedTargetId?: string;
		busy?: boolean;
		onTargetChange?: (targetId: string) => void;
		onCancel?: () => void;
		onSaveAs?: () => void | Promise<void>;
		onReplace?: () => void | Promise<void>;
	} = $props();

	function handleKeydown(event: KeyboardEvent) {
		if (!open || busy) {
			return;
		}
		if (event.key === 'Escape') {
			event.preventDefault();
			onCancel?.();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<div class="fixed inset-0 z-[100] flex items-center justify-center bg-black/55 p-4 backdrop-blur-[1px]">
		<div
			role="dialog"
			aria-modal="true"
			aria-label={m.editor_export_private_images_dialog_label()}
			tabindex="-1"
			class="w-full max-w-md rounded-xl border border-zinc-200 bg-white p-5 shadow-2xl dark:border-zinc-700 dark:bg-zinc-900"
			use:clickOutside={{
				enabled: open && !busy,
				handler: () => onCancel?.()
			}}
		>
			<h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
				{m.editor_export_private_images_dialog_title()}
			</h3>
			<p class="mt-2 text-sm leading-6 text-zinc-600 dark:text-zinc-300">
				{m.editor_export_private_images_dialog_message({ count: imageCount })}
			</p>

			<label class="mt-4 flex flex-col gap-2">
				<span class="text-sm font-medium text-zinc-800 dark:text-zinc-200">{m.editor_export_private_images_dialog_target()}</span>
				<select
					class="rounded-2xl border border-zinc-200 bg-white px-4 py-3 text-sm text-zinc-900 outline-none transition focus:border-sky-400 focus:ring-2 focus:ring-sky-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-sky-500 dark:focus:ring-sky-900/60"
					value={selectedTargetId}
					disabled={busy}
					onchange={(event) => onTargetChange?.((event.currentTarget as HTMLSelectElement).value)}
				>
					{#each targetOptions as option (option.id)}
						<option value={option.id}>{option.label}</option>
					{/each}
				</select>
			</label>

			<p class="mt-3 text-xs leading-5 text-zinc-500 dark:text-zinc-400">
				{m.editor_export_private_images_dialog_help()}
			</p>

			<div class="mt-5 flex justify-end gap-2">
				<button
					type="button"
					class="rounded-md px-4 py-2 text-sm text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
					disabled={busy}
					onclick={() => onCancel?.()}
				>
					{m.common_cancel()}
				</button>
				<button
					type="button"
					class="rounded-md px-4 py-2 text-sm text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
					disabled={busy}
					onclick={() => onSaveAs?.()}
				>
					{busy ? m.common_processing() : m.common_save_as()}
				</button>
				<button
					type="button"
					class="rounded-2xl bg-sky-500 px-4 py-3 text-sm font-medium text-white shadow-sm transition hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:text-white dark:hover:bg-sky-400"
					disabled={busy}
					onclick={() => onReplace?.()}
				>
					{busy ? m.common_processing() : m.common_replace()}
				</button>
			</div>
		</div>
	</div>
{/if}
