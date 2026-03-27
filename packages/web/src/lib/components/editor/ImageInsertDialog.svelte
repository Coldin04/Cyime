<script lang="ts">
	import { browser } from '$app/environment';
	import { portal } from '$lib/actions/portal';
	import * as m from '$paraglide/messages';
	import UploadSimple from '~icons/ph/upload-simple';
	import LinkSimple from '~icons/ph/link-simple';

	type Props = {
		open: boolean;
		accept: string;
		isUploading?: boolean;
		currentTargetLabel: string;
		onFilesSelected: (files: File[]) => void;
		onInsertLink: (src: string) => boolean | Promise<boolean>;
	};

	let {
		open = $bindable(false),
		accept,
		isUploading = false,
		currentTargetLabel,
		onFilesSelected,
		onInsertLink
	}: Props = $props();

	let fileInput = $state<HTMLInputElement | null>(null);
	let linkValue = $state('');
	let isDragging = $state(false);
	let isSubmittingLink = $state(false);

	function closeDialog() {
		open = false;
		isDragging = false;
		linkValue = '';
	}

	function handleFileInputChange(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const files = input.files ? Array.from(input.files) : [];
		if (files.length === 0) {
			return;
		}
		onFilesSelected(files);
		input.value = '';
		closeDialog();
	}

	function handleDrop(event: DragEvent) {
		event.preventDefault();
		isDragging = false;
		const files = Array.from(event.dataTransfer?.files ?? []).filter((file) =>
			file.type.startsWith('image/')
		);
		if (files.length === 0) {
			return;
		}
		onFilesSelected(files);
		closeDialog();
	}

	async function handleInsertLink() {
		if (isSubmittingLink) {
			return;
		}
		isSubmittingLink = true;
		try {
			const inserted = await onInsertLink(linkValue);
			if (inserted) {
				closeDialog();
			}
		} finally {
			isSubmittingLink = false;
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
				class="w-full max-w-lg rounded-[28px] border border-zinc-200 bg-white p-5 shadow-2xl dark:border-zinc-800 dark:bg-zinc-950"
				role="dialog"
				aria-modal="true"
				tabindex="-1"
				onclick={(event) => event.stopPropagation()}
				onkeydown={(event) => {
					if (event.key === 'Escape') {
						closeDialog();
					}
				}}
			>
				<div class="flex items-start justify-between gap-4">
					<div>
						<h2 class="text-lg font-semibold text-zinc-900 dark:text-zinc-100">
							{m.editor_image_insert_title()}
						</h2>
						<p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
							{m.editor_image_insert_description()}
						</p>
					</div>
					<button
						type="button"
						class="rounded-full p-2 text-zinc-500 transition hover:bg-zinc-100 hover:text-zinc-900 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
						onclick={closeDialog}
					>
						✕
					</button>
				</div>

				<p class="mt-4 rounded-2xl bg-riptide-50 px-3 py-2 text-xs font-medium text-riptide-900 dark:bg-riptide-950/40 dark:text-riptide-200">
					{m.editor_image_insert_current_target({ target: currentTargetLabel })}
				</p>

				<div class="mt-4 space-y-4">
					<input
						bind:this={fileInput}
						type="file"
						accept={accept}
						multiple
						class="hidden"
						onchange={handleFileInputChange}
					/>

					<button
						type="button"
						class={`flex w-full flex-col items-center justify-center gap-3 rounded-[24px] border border-dashed px-5 py-8 text-center transition ${
							isDragging
								? 'border-riptide-500 bg-riptide-50 dark:border-riptide-400 dark:bg-riptide-950/30'
								: 'border-zinc-300 bg-zinc-50 hover:border-zinc-400 hover:bg-zinc-100 dark:border-zinc-700 dark:bg-zinc-900 dark:hover:border-zinc-600 dark:hover:bg-zinc-900/80'
						}`}
						onclick={() => fileInput?.click()}
						ondragenter={(event) => {
							event.preventDefault();
							isDragging = true;
						}}
						ondragover={(event) => {
							event.preventDefault();
							isDragging = true;
						}}
						ondragleave={(event) => {
							event.preventDefault();
							if (event.currentTarget === event.target) {
								isDragging = false;
							}
						}}
						ondrop={handleDrop}
					>
						<div class="grid h-12 w-12 place-content-center rounded-2xl bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900">
							<UploadSimple class="h-6 w-6" />
						</div>
						<div class="space-y-1">
							<p class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
								{m.editor_image_insert_drop_title()}
							</p>
							<p class="text-xs text-zinc-500 dark:text-zinc-400">
								{m.editor_image_insert_drop_hint()}
							</p>
						</div>
					</button>

					<div class="rounded-[24px] border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
						<div class="flex items-center gap-2">
							<div class="grid h-9 w-9 shrink-0 place-content-center rounded-xl bg-zinc-100 text-zinc-700 dark:bg-zinc-900 dark:text-zinc-200">
								<LinkSimple class="h-4 w-4" />
							</div>
							<div>
								<p class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">
									{m.editor_image_insert_link_label()}
								</p>
								<p class="text-xs text-zinc-500 dark:text-zinc-400">
									{m.editor_external_image_invalid()}
								</p>
							</div>
						</div>

						<div class="mt-3 flex flex-col gap-2 sm:flex-row">
							<input
								bind:value={linkValue}
								type="url"
								inputmode="url"
								autocapitalize="off"
								autocomplete="off"
								spellcheck="false"
								class="min-w-0 flex-1 rounded-2xl border border-zinc-200 bg-white px-4 py-3 text-sm text-zinc-900 outline-none transition focus:border-riptide-400 focus:ring-2 focus:ring-riptide-200 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-riptide-500 dark:focus:ring-riptide-900/60"
								placeholder={m.editor_image_insert_link_placeholder()}
								onkeydown={(event) => {
									if (event.key === 'Enter') {
										event.preventDefault();
										void handleInsertLink();
									}
								}}
							/>
							<button
								type="button"
								class="rounded-2xl bg-zinc-900 px-4 py-3 text-sm font-medium text-white transition hover:bg-zinc-800 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-200"
								disabled={isSubmittingLink || isUploading}
								onclick={() => {
									void handleInsertLink();
								}}
							>
								{m.editor_image_insert_link_submit()}
							</button>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}
