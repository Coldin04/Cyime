<script lang="ts">
	import { clickOutside } from '$lib/actions/clickOutside';
	import type { DocumentImageTargetOption } from '$lib/components/editor/documentImageTargets';

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
			aria-label="导出私有图片处理"
			tabindex="-1"
			class="w-full max-w-md rounded-xl border border-zinc-200 bg-white p-5 shadow-2xl dark:border-zinc-700 dark:bg-zinc-900"
			use:clickOutside={{
				enabled: open && !busy,
				handler: () => onCancel?.()
			}}
		>
			<h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">
				私有图片无法直接导出
			</h3>
			<p class="mt-2 text-sm leading-6 text-zinc-600 dark:text-zinc-300">
				当前文档包含 {imageCount} 张私有图片。Markdown 和 HTML 导出无法稳定访问这些签名链接，请先上传到图床再继续导出。
			</p>

			<label class="mt-4 flex flex-col gap-2">
				<span class="text-sm font-medium text-zinc-800 dark:text-zinc-200">导出时上传到</span>
				<select
					class="rounded-2xl border border-zinc-200 bg-white px-4 py-3 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
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
				“另存为”会复制一份新文档并替换图片链接；“替换”会直接修改当前文档内容和图片上传偏好，然后继续导出。
			</p>

			<div class="mt-5 flex justify-end gap-2">
				<button
					type="button"
					class="rounded-md px-4 py-2 text-sm text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
					disabled={busy}
					onclick={() => onCancel?.()}
				>
					取消
				</button>
				<button
					type="button"
					class="rounded-md px-4 py-2 text-sm text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
					disabled={busy}
					onclick={() => onSaveAs?.()}
				>
					{busy ? '处理中...' : '另存为'}
				</button>
				<button
					type="button"
					class="rounded-2xl bg-zinc-900 px-4 py-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
					disabled={busy}
					onclick={() => onReplace?.()}
				>
					{busy ? '处理中...' : '替换'}
				</button>
			</div>
		</div>
	</div>
{/if}
