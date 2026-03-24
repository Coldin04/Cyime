<script lang="ts">
	import ImageSquare from '~icons/ph/image-square';

	interface Props {
		accept: string;
		disabled?: boolean;
		label: string;
		uploadingLabel?: string;
		isUploading?: boolean;
		onFilesSelected: (files: FileList) => void;
	}

	let {
		accept,
		disabled = false,
		label,
		uploadingLabel = label,
		isUploading = false,
		onFilesSelected
	}: Props = $props();

	let inputElement: HTMLInputElement | null = null;

	function handleChange(event: Event) {
		const target = event.currentTarget as HTMLInputElement;
		if (!target.files || target.files.length === 0) {
			return;
		}

		onFilesSelected(target.files);
		target.value = '';
	}
</script>

<input
	bind:this={inputElement}
	type="file"
	accept={accept}
	multiple
	class="hidden"
	onchange={handleChange}
/>

<button
	type="button"
	title={isUploading ? uploadingLabel : label}
	aria-label={isUploading ? uploadingLabel : label}
	disabled={disabled || isUploading}
	class="inline-flex h-8 shrink-0 items-center justify-center gap-1.5 rounded-md px-2 leading-none text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
	onclick={() => inputElement?.click()}
>
	<ImageSquare class="h-4 w-4" />
	{#if isUploading}
		<span class="text-xs font-medium">{uploadingLabel}</span>
	{/if}
</button>
