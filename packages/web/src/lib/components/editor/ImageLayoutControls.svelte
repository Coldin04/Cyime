<script lang="ts">
	import * as m from '$paraglide/messages';
	import TextAlignLeft from '~icons/ph/text-align-left';
	import ArrowsOutHorizontal from '~icons/ph/arrows-out-line-horizontal';

	interface Props {
		currentAlign: string;
		onSelect: (align: string) => void;
	}

	let { currentAlign, onSelect }: Props = $props();

	const options = [
		{ value: 'content', title: m.editor_image_align_content(), icon: TextAlignLeft },
		{ value: 'full', title: m.editor_image_align_full(), icon: ArrowsOutHorizontal }
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
		class={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md transition-colors ${
			currentAlign === option.value ? activeToggleClass : inactiveToggleClass
		}`}
		onclick={() => onSelect(option.value)}
	>
		<option.icon class="h-4 w-4" />
	</button>
{/each}
