<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Editor, rootCtx, defaultValueCtx } from '@milkdown/kit/core';
	import { commonmark } from '@milkdown/kit/preset/commonmark';
	import { nord } from '@milkdown/theme-nord';
	import { listener, listenerCtx } from '@milkdown/kit/plugin/listener';

	interface Props {
		content: string;
		onContentChange?: (content: string) => void;
	}

	let { content, onContentChange }: Props = $props();

	let editorElement: HTMLDivElement | undefined = $state();
	let editor: Editor | null = null;

	onMount(async () => {
		if (!editorElement) return;

		editor = await Editor.make()
			.config((ctx) => {
				ctx.set(rootCtx, editorElement);
				ctx.set(defaultValueCtx, content || '');
				ctx.get(listenerCtx).markdownUpdated((_, markdown) => {
					onContentChange?.(markdown);
				});
			})
			.config(nord)
			.use(commonmark)
			.use(listener)
			.create();
	});

	onDestroy(() => {
		if (editor) {
			editor.destroy();
			editor = null;
		}
	});
</script>

<div bind:this={editorElement} class="milkdown-editor h-full w-full"></div>

<style>
	:global(.milkdown-editor .ProseMirror) {
		height: 100%;
		width: 100%;
		max-width: none;
		min-height: 100%;
		outline: none;
	}

	:global(.milkdown-editor) {
		max-width: none;
		height: 100%;
		outline: none;
	}

	:global(.milkdown-editor > div) {
		height: 100%;
		outline: none;
	}
</style>
