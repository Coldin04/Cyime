<script lang="ts">
	import { browser } from '$app/environment';
	import { portal } from '$lib/actions/portal';
	import * as m from '$paraglide/messages';

	type MathMode = 'inline' | 'block';

	type Props = {
		open: boolean;
		mode: MathMode;
		initialValue?: string;
		showDelete?: boolean;
		onSubmit: (latex: string) => boolean | Promise<boolean>;
		onDelete?: () => boolean | Promise<boolean>;
	};

	let {
		open = $bindable(false),
		mode,
		initialValue = '',
		showDelete = false,
		onSubmit,
		onDelete
	}: Props = $props();

	let latexValue = $state('');
	let isSubmitting = $state(false);
	const inlinePlaceholder = '\\alpha + \\beta = \\gamma';
	const blockPlaceholder = '\\int_0^1 x^2 \\\\, dx = \\\\frac{1}{3}';

	$effect(() => {
		latexValue = initialValue;
	});

	function closeDialog() {
		open = false;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			closeDialog();
			return;
		}
		if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
			event.preventDefault();
			void handleSubmit();
			return;
		}
		if (mode === 'inline' && event.key === 'Enter') {
			event.preventDefault();
			void handleSubmit();
		}
	}

	async function handleSubmit() {
		if (isSubmitting) {
			return;
		}

		isSubmitting = true;
		try {
			const submitted = await onSubmit(latexValue);
			if (submitted) {
				closeDialog();
			}
		} finally {
			isSubmitting = false;
		}
	}

	async function handleDelete() {
		if (!onDelete || isSubmitting) {
			return;
		}
		isSubmitting = true;
		try {
			const deleted = await onDelete();
			if (deleted) {
				closeDialog();
			}
		} finally {
			isSubmitting = false;
		}
	}

	$effect(() => {
		if (!browser) {
			return;
		}
		document.body.style.overflow = open ? 'hidden' : '';
		return () => {
			document.body.style.overflow = '';
		};
	});
</script>

{#if open}
	<div
		use:portal
		class="fixed inset-0 z-[120] min-h-dvh w-screen overflow-y-auto bg-black/45"
		role="presentation"
		onclick={closeDialog}
	>
		<div class="flex min-h-dvh w-full items-center justify-center p-3 sm:p-5">
			<div
				class="w-full max-w-lg rounded-[18px] border border-zinc-200 bg-white p-5 shadow-2xl dark:border-zinc-800 dark:bg-zinc-950"
				role="dialog"
				aria-modal="true"
				aria-label={mode === 'inline' ? m.editor_math_inline_title() : m.editor_math_block_title()}
				tabindex="-1"
				onclick={(event) => event.stopPropagation()}
				onkeydown={handleKeydown}
			>
				<div class="flex items-start justify-between gap-4">
					<div>
						<h2 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
							{mode === 'inline' ? m.editor_math_inline_title() : m.editor_math_block_title()}
						</h2>
						<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
							{mode === 'inline'
								? m.editor_math_inline_description()
								: m.editor_math_block_description()}
						</p>
					</div>
					<button
						type="button"
						class="inline-flex h-8 w-8 items-center justify-center rounded-full text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
						onclick={closeDialog}
					>
						✕
					</button>
				</div>

				<div class="mt-4">
					{#if mode === 'inline'}
						<div class="flex items-center gap-2 border border-zinc-200 bg-zinc-50 px-3 py-2 text-xs text-zinc-700 dark:border-zinc-700 dark:bg-zinc-950 dark:text-zinc-200">
							<input
								id="math-latex-input"
								bind:value={latexValue}
								type="text"
								autocapitalize="off"
								autocomplete="off"
								spellcheck="false"
								class="min-w-0 flex-1 bg-transparent font-mono text-sm text-zinc-900 outline-none placeholder:text-zinc-400 dark:text-zinc-100 dark:placeholder:text-zinc-500"
								placeholder={inlinePlaceholder}
							/>
							<button
								type="button"
								class="inline-flex h-7 shrink-0 items-center bg-zinc-900 px-2 text-xs font-medium text-white transition-colors hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
								disabled={isSubmitting}
								onclick={() => {
									void handleSubmit();
								}}
							>
								{m.common_save()}
							</button>
						</div>
					{:else}
						<textarea
							id="math-latex-input"
							bind:value={latexValue}
							rows={7}
							autocapitalize="off"
							autocomplete="off"
							spellcheck="false"
							class="min-h-[8rem] w-full resize-y rounded-none border border-zinc-200 bg-white px-4 py-3 font-mono text-sm text-zinc-900 outline-none transition focus:border-sky-400 focus:ring-2 focus:ring-sky-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-sky-500 dark:focus:ring-sky-900/60"
							placeholder={blockPlaceholder}
						></textarea>
					{/if}
					<p class="mt-2 text-xs text-zinc-500 dark:text-zinc-400">
						{m.editor_math_submit_hint()}
					</p>
				</div>

				<div class="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
					{#if mode === 'block' && showDelete}
						<button
							type="button"
							class="h-8 rounded-md border border-red-200 px-3 text-sm font-medium text-red-600 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-red-900/60 dark:text-red-300 dark:hover:bg-red-950/30"
							disabled={isSubmitting}
							onclick={() => {
								void handleDelete();
							}}
						>
							{m.common_delete()}
						</button>
					{/if}
					<button
						type="button"
						class="h-8 rounded-md border border-zinc-200 px-3 text-sm font-medium text-zinc-700 transition hover:bg-zinc-100 dark:border-zinc-700 dark:text-zinc-200 dark:hover:bg-zinc-800"
						onclick={closeDialog}
					>
						{m.common_cancel()}
					</button>
					{#if mode === 'block'}
						<button
							type="button"
							class="h-8 rounded-md bg-zinc-900 px-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
							disabled={isSubmitting}
							onclick={() => {
								void handleSubmit();
							}}
						>
							{m.common_save()}
						</button>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}
