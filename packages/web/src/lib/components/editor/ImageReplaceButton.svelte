<script lang="ts">
	import ArrowsClockwise from '~icons/ph/arrows-clockwise';

	interface Props {
		accept: string;
		disabled?: boolean;
		label: string;
		onFileSelected: (file: File) => void;
	}

	let { accept, disabled = false, label, onFileSelected }: Props = $props();

	let inputElement: HTMLInputElement | null = null;

	function handleChange(event: Event) {
		const target = event.currentTarget as HTMLInputElement;
		const file = target.files?.[0];
		if (!file) {
			return;
		}

		onFileSelected(file);
		target.value = '';
	}
</script>

<input
	bind:this={inputElement}
	type="file"
	accept={accept}
	class="hidden"
	onchange={handleChange}
/>

<button
	type="button"
	title={label}
	aria-label={label}
	disabled={disabled}
	class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
	onclick={() => inputElement?.click()}
>
	<ArrowsClockwise class="h-4 w-4" />
</button>
