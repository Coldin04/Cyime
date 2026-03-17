<script lang="ts">
	import * as m from '$paraglide/messages';

	interface Props {
		currentWidth: string;
		onSelect: (width: string) => void;
	}

	let { currentWidth, onSelect }: Props = $props();

	const options = [
		{ value: 'auto', label: 'Auto', title: m.editor_image_size_auto() },
		{ value: '40%', label: 'S', title: m.editor_image_size_small() },
		{ value: '60%', label: 'M', title: m.editor_image_size_medium() },
		{ value: '80%', label: 'L', title: m.editor_image_size_large() },
		{ value: '100%', label: 'Full', title: m.editor_image_size_full() }
	] as const;

	const activeToggleClass = 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900';
	const inactiveToggleClass =
		'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800';
</script>

{#each options as option}
	<button
		type="button"
		title={option.title}
		aria-label={option.title}
		class={`inline-flex h-8 shrink-0 items-center rounded-md px-2 text-[11px] font-medium leading-none transition-colors ${
			currentWidth === option.value ? activeToggleClass : inactiveToggleClass
		}`}
		onclick={() => onSelect(option.value)}
	>
		{option.label}
	</button>
{/each}
