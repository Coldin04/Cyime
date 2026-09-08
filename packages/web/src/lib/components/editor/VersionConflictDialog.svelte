<script lang="ts">
	import * as m from '$paraglide/messages';
	import ModalDialog from '$lib/components/common/ModalDialog.svelte';
	import { exportMarkdown } from '$lib/export/documentExport';
	import type { ConflictResolution, MergeChunk } from '$lib/components/editor/blockMerge';
	import type { JSONContent } from '@tiptap/core';

	let {
		open,
		status,
		cloudUpdatedAt,
		chunks,
		resolutions,
		onResolutionChange,
		onApply,
		onDiscardLocal,
		onExportDraft,
		onReturnWorkspace,
		onOpenReader
	}: {
		open: boolean;
		status: 'loading' | 'ready' | 'saving';
		cloudUpdatedAt: string | null;
		chunks: MergeChunk[];
		resolutions: Record<string, ConflictResolution>;
		onResolutionChange: (id: string, resolution: ConflictResolution) => void;
		onApply: () => void | Promise<void>;
		onDiscardLocal: () => void | Promise<void>;
		onExportDraft: () => void | Promise<void>;
		onReturnWorkspace: () => void;
		onOpenReader: () => void;
	} = $props();

	function previewOf(blocks: JSONContent[]): string {
		if (blocks.length === 0) {
			return m.editor_version_conflict_empty_run();
		}
		const markdown = exportMarkdown({ type: 'doc', content: blocks }).trim();
		return markdown === '' ? m.editor_version_conflict_empty_run() : markdown;
	}

	const conflicts = $derived(chunks.filter((chunk): chunk is Extract<MergeChunk, { kind: 'conflict' }> => chunk.kind === 'conflict'));
	const autoLocalCount = $derived(
		chunks.filter((chunk) => chunk.kind === 'auto' && chunk.source === 'local').length
	);
	const autoCloudCount = $derived(
		chunks.filter((chunk) => chunk.kind === 'auto' && chunk.source === 'cloud').length
	);
	const busy = $derived(status === 'loading' || status === 'saving');

	const formattedCloudUpdatedAt = $derived.by(() => {
		if (!cloudUpdatedAt) return '';
		try {
			return new Date(cloudUpdatedAt).toLocaleString();
		} catch {
			return cloudUpdatedAt;
		}
	});
</script>

<ModalDialog {open} title={m.editor_version_conflict_dialog_title()} maxWidthClass="max-w-4xl">
	<div class="flex max-h-[80vh] flex-col gap-4">
		<div>
			<h2 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
				{m.editor_version_conflict_dialog_title()}
			</h2>
			<p class="mt-1 text-sm leading-6 text-zinc-600 dark:text-zinc-300">
				{m.editor_version_conflict_dialog_description()}
			</p>
			{#if formattedCloudUpdatedAt}
				<p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">
					{m.editor_version_conflict_cloud_updated_at({ time: formattedCloudUpdatedAt })}
				</p>
			{/if}
		</div>

		{#if status === 'loading'}
			<div class="flex items-center justify-center py-10 text-sm text-zinc-500 dark:text-zinc-400">
				{m.editor_version_conflict_loading()}
			</div>
		{:else}
			{#if autoLocalCount + autoCloudCount > 0}
				<p class="rounded-md bg-zinc-50 px-3 py-2 text-xs leading-5 text-zinc-600 dark:bg-zinc-800/60 dark:text-zinc-300">
					{m.editor_version_conflict_auto_summary({ local: autoLocalCount, cloud: autoCloudCount })}
				</p>
			{/if}

			{#if conflicts.length === 0}
				<p class="text-sm text-zinc-600 dark:text-zinc-300">
					{m.editor_version_conflict_no_conflicts()}
				</p>
			{:else}
				<p class="text-sm font-medium text-zinc-800 dark:text-zinc-100">
					{m.editor_version_conflict_conflict_count({ count: conflicts.length })}
				</p>
				<div class="flex-1 space-y-3 overflow-y-auto pr-1">
					{#each conflicts as conflict (conflict.id)}
						{@const resolution = resolutions[conflict.id] ?? 'local'}
						<div class="rounded-lg border border-zinc-200 p-3 dark:border-zinc-700">
							<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
								<div
									class={`rounded-md border p-2 ${resolution === 'local' ? 'border-sky-400 bg-sky-50 dark:border-sky-500 dark:bg-sky-950/40' : 'border-zinc-200 dark:border-zinc-700'}`}
								>
									<div class="mb-1 text-xs font-semibold text-zinc-500 dark:text-zinc-400">
										{m.editor_version_conflict_local_label()}
									</div>
									<pre class="whitespace-pre-wrap break-words font-sans text-xs leading-5 text-zinc-800 dark:text-zinc-100">{previewOf(conflict.localBlocks)}</pre>
								</div>
								<div
									class={`rounded-md border p-2 ${resolution === 'cloud' ? 'border-sky-400 bg-sky-50 dark:border-sky-500 dark:bg-sky-950/40' : 'border-zinc-200 dark:border-zinc-700'}`}
								>
									<div class="mb-1 text-xs font-semibold text-zinc-500 dark:text-zinc-400">
										{m.editor_version_conflict_cloud_label()}
									</div>
									<pre class="whitespace-pre-wrap break-words font-sans text-xs leading-5 text-zinc-800 dark:text-zinc-100">{previewOf(conflict.cloudBlocks)}</pre>
								</div>
							</div>
							<div class="mt-2 flex flex-wrap gap-2">
								<button
									type="button"
									class={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${resolution === 'local' ? 'bg-sky-500 text-white' : 'bg-zinc-100 text-zinc-700 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700'}`}
									disabled={busy}
									onclick={() => onResolutionChange(conflict.id, 'local')}
								>
									{m.editor_version_conflict_use_local()}
								</button>
								<button
									type="button"
									class={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${resolution === 'cloud' ? 'bg-sky-500 text-white' : 'bg-zinc-100 text-zinc-700 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700'}`}
									disabled={busy}
									onclick={() => onResolutionChange(conflict.id, 'cloud')}
								>
									{m.editor_version_conflict_use_cloud()}
								</button>
								{#if conflict.allowBoth}
									<button
										type="button"
										class={`rounded-md px-2.5 py-1 text-xs font-medium transition-colors ${resolution === 'both' ? 'bg-sky-500 text-white' : 'bg-zinc-100 text-zinc-700 hover:bg-zinc-200 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700'}`}
										disabled={busy}
										onclick={() => onResolutionChange(conflict.id, 'both')}
									>
										{m.editor_version_conflict_use_both()}
									</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			{/if}
		{/if}

		<div class="flex flex-wrap items-center justify-between gap-2 border-t border-zinc-200 pt-3 dark:border-zinc-700">
			<div class="flex flex-wrap gap-2">
				<button
					type="button"
					class="rounded-md px-3 py-1.5 text-xs text-zinc-600 underline-offset-2 hover:underline dark:text-zinc-300"
					disabled={busy}
					onclick={() => void onExportDraft()}
				>
					{m.editor_version_conflict_export_draft_action()}
				</button>
				<button
					type="button"
					class="rounded-md px-3 py-1.5 text-xs text-zinc-600 underline-offset-2 hover:underline dark:text-zinc-300"
					disabled={busy}
					onclick={onOpenReader}
				>
					{m.editor_open_reader_action()}
				</button>
			</div>
			<div class="flex flex-wrap gap-2">
				<button
					type="button"
					class="rounded-md px-3 py-2 text-sm text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
					disabled={busy}
					onclick={onReturnWorkspace}
				>
					{m.editor_return_workspace_action()}
				</button>
				<button
					type="button"
					class="rounded-md bg-zinc-100 px-3 py-2 text-sm font-medium text-zinc-700 transition-colors hover:bg-zinc-200 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-800 dark:text-zinc-200 dark:hover:bg-zinc-700"
					disabled={busy}
					onclick={() => void onDiscardLocal()}
				>
					{m.editor_version_conflict_discard_local_action()}
				</button>
				<button
					type="button"
					class="rounded-md bg-sky-500 px-3 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-sky-500 dark:hover:bg-sky-400"
					disabled={busy}
					onclick={() => void onApply()}
				>
					{status === 'saving' ? m.common_loading() : m.editor_version_conflict_apply_action()}
				</button>
			</div>
		</div>
	</div>
</ModalDialog>
