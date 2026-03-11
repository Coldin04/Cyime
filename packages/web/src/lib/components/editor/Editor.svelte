<script lang="ts">
	interface Props {
		content: string;
		onContentChange?: (content: string) => void;
	}

	let { content, onContentChange }: Props = $props();
	let localContent = $state('');

	$effect(() => {
		if (content !== localContent) {
			localContent = content;
		}
	});

	function handleInput(event: Event) {
		const target = event.currentTarget as HTMLTextAreaElement;
		localContent = target.value;
		onContentChange?.(target.value);
	}
</script>

<div class="h-full w-full p-4">
	<textarea
		class="h-full w-full resize-none rounded-md border border-zinc-200 bg-white p-4 font-mono text-sm text-zinc-800 outline-none focus:border-zinc-400 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-100 dark:focus:border-zinc-500"
		value={localContent}
		oninput={handleInput}
		placeholder="Tiptap migration in progress..."
	></textarea>
</div>
