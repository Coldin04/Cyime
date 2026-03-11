<script lang="ts">
	import { onMount } from 'svelte';
	import { Editor } from '@tiptap/core';
	import type { Content, JSONContent } from '@tiptap/core';
	import Placeholder from '@tiptap/extension-placeholder';
	import StarterKit from '@tiptap/starter-kit';
	import * as m from '$paraglide/messages';
	import TextHOne from '~icons/ph/text-h-one';
	import TextHTwo from '~icons/ph/text-h-two';
	import Paragraph from '~icons/ph/paragraph';
	import TextB from '~icons/ph/text-b';
	import TextItalic from '~icons/ph/text-italic';
	import ListBullets from '~icons/ph/list-bullets';
	import ListNumbers from '~icons/ph/list-numbers';
	import FloppyDisk from '~icons/ph/floppy-disk';
	import ArrowCounterClockwise from '~icons/ph/arrow-counter-clockwise';
	import ArrowClockwise from '~icons/ph/arrow-clockwise';

	interface Props {
		content: string;
		isSaving?: boolean;
		hasUnsavedChanges?: boolean;
		onContentChange?: (content: string) => void;
		onSave?: () => void | Promise<void>;
	}

	let { content, isSaving = false, hasUnsavedChanges = false, onContentChange, onSave }: Props = $props();

	let editorElement: HTMLDivElement | null = null;
	let editor: Editor | null = null;
	let lastSyncedContent = '';
	let editorRevision = $state(0);

	function sanitizePastedHTML(html: string): string {
		const parser = new DOMParser();
		const doc = parser.parseFromString(html, 'text/html');

		doc.querySelectorAll('script, style, meta, link').forEach((node) => node.remove());

		doc.querySelectorAll('*').forEach((element) => {
			for (const attr of [...element.attributes]) {
				const attrName = attr.name.toLowerCase();
				const isUnsafe =
					attrName === 'style' ||
					attrName === 'class' ||
					attrName === 'id' ||
					attrName.startsWith('data-') ||
					attrName.startsWith('aria-');
				if (isUnsafe) {
					element.removeAttribute(attr.name);
				}
			}
		});

		// Flatten styling-only wrappers so pasted content keeps structure but not noisy spans.
		doc.querySelectorAll('span').forEach((span) => {
			const parent = span.parentNode;
			if (!parent) return;
			while (span.firstChild) {
				parent.insertBefore(span.firstChild, span);
			}
			parent.removeChild(span);
		});

		return doc.body.innerHTML;
	}

	function createParagraphNode(text: string): JSONContent {
		if (!text) {
			return { type: 'paragraph' };
		}

		return {
			type: 'paragraph',
			content: [
				{
					type: 'text',
					text
				}
			]
		};
	}

	function toTiptapContent(value: string): Content {
		const trimmed = value.trim();

		if (!trimmed) {
			return {
				type: 'doc',
				content: [{ type: 'paragraph' }]
			};
		}

		// Preserve existing HTML documents if they already come from a rich-text source.
		if (trimmed.startsWith('<') && trimmed.endsWith('>')) {
			return value;
		}

		return {
			type: 'doc',
			content: value.split(/\n{2,}/).map((block) => createParagraphNode(block.replace(/\n/g, ' ')))
		};
	}

	onMount(() => {
		if (!editorElement) {
			return;
		}

		lastSyncedContent = content;
		editor = new Editor({
			element: editorElement,
			extensions: [
				StarterKit,
				Placeholder.configure({
					placeholder: m.editor_placeholder()
				})
			],
			content: toTiptapContent(content),
			editorProps: {
				transformPastedHTML: (html) => sanitizePastedHTML(html),
				attributes: {
					class:
						'tiptap min-h-full w-full px-4 py-6 text-base text-zinc-800 outline-none dark:text-zinc-100 sm:px-8 lg:px-[14%]'
				}
			},
			onUpdate: ({ editor }) => {
				const nextContent = editor.getHTML();
				lastSyncedContent = nextContent;
				onContentChange?.(nextContent);
				editorRevision += 1;
			},
			onSelectionUpdate: () => {
				editorRevision += 1;
			}
		});

		return () => {
			editor?.destroy();
			editor = null;
		};
	});

	$effect(() => {
		if (!editor) {
			return;
		}

		if (content === lastSyncedContent) {
			return;
		}

		lastSyncedContent = content;
		editor.commands.setContent(toTiptapContent(content), { emitUpdate: false });
	});

	function apply(action: (instance: Editor) => void) {
		if (!editor) return;
		action(editor);
		editorRevision += 1;
	}

	function isActive(name: string, attributes?: Record<string, unknown>) {
		editorRevision;
		if (!editor) return false;
		return editor.isActive(name, attributes);
	}

	function canUndo() {
		editorRevision;
		if (!editor) return false;
		return editor.can().chain().focus().undo().run();
	}

	function canRedo() {
		editorRevision;
		if (!editor) return false;
		return editor.can().chain().focus().redo().run();
	}
</script>

<div class="flex h-full w-full flex-col">
	<div class="border-b border-zinc-200 px-3 py-2 dark:border-zinc-800">
		<div class="mx-auto flex w-full max-w-4xl flex-wrap items-center justify-center gap-2">
			<button
				type="button"
				title={m.editor_toolbar_save_with_shortcut()}
				aria-label={m.editor_toolbar_save_with_shortcut()}
				disabled={isSaving || !hasUnsavedChanges}
				onclick={() => onSave?.()}
				class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md leading-none text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
			>
				<FloppyDisk class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_undo_with_shortcut()}
				aria-label={m.editor_toolbar_undo_with_shortcut()}
				disabled={!canUndo()}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().undo().run();
					})}
				class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md leading-none text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
			>
				<ArrowCounterClockwise class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_redo_with_shortcut()}
				aria-label={m.editor_toolbar_redo_with_shortcut()}
				disabled={!canRedo()}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().redo().run();
					})}
				class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md leading-none text-zinc-700 transition-colors hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-50 dark:text-zinc-200 dark:hover:bg-zinc-800"
			>
				<ArrowClockwise class="h-4 w-4" />
			</button>
			<div class="mx-1 h-5 w-px bg-zinc-200 dark:bg-zinc-700"></div>
			<div class="flex flex-wrap items-center gap-2">
			<button
				type="button"
				title={m.editor_toolbar_heading_1()}
				aria-label={m.editor_toolbar_heading_1()}
				class="rounded-md px-2 py-1 text-xs leading-none text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				class:bg-zinc-900={isActive('heading', { level: 1 })}
				class:text-white={isActive('heading', { level: 1 })}
				class:dark:bg-zinc-100={isActive('heading', { level: 1 })}
				class:dark:text-zinc-900={isActive('heading', { level: 1 })}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleHeading({ level: 1 }).run();
					})}
			>
				<TextHOne class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_heading_2()}
				aria-label={m.editor_toolbar_heading_2()}
				class="rounded-md px-2 py-1 text-xs leading-none text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				class:bg-zinc-900={isActive('heading', { level: 2 })}
				class:text-white={isActive('heading', { level: 2 })}
				class:dark:bg-zinc-100={isActive('heading', { level: 2 })}
				class:dark:text-zinc-900={isActive('heading', { level: 2 })}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleHeading({ level: 2 }).run();
					})}
			>
				<TextHTwo class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_paragraph()}
				aria-label={m.editor_toolbar_paragraph()}
				class="rounded-md px-2 py-1 text-xs leading-none text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().setParagraph().run();
					})}
			>
				<Paragraph class="h-4 w-4" />
			</button>
			<div class="mx-1 h-5 w-px bg-zinc-200 dark:bg-zinc-700"></div>
			<button
				type="button"
				title={m.editor_toolbar_bold()}
				aria-label={m.editor_toolbar_bold()}
				class="rounded-md px-2 py-1 text-xs font-semibold leading-none text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				class:bg-zinc-900={isActive('bold')}
				class:text-white={isActive('bold')}
				class:dark:bg-zinc-100={isActive('bold')}
				class:dark:text-zinc-900={isActive('bold')}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleBold().run();
					})}
			>
				<TextB class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_italic()}
				aria-label={m.editor_toolbar_italic()}
				class="rounded-md px-2 py-1 text-xs italic leading-none text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				class:bg-zinc-900={isActive('italic')}
				class:text-white={isActive('italic')}
				class:dark:bg-zinc-100={isActive('italic')}
				class:dark:text-zinc-900={isActive('italic')}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleItalic().run();
					})}
			>
				<TextItalic class="h-4 w-4" />
			</button>
			<div class="mx-1 h-5 w-px bg-zinc-200 dark:bg-zinc-700"></div>
			<button
				type="button"
				title={m.editor_toolbar_bullet_list()}
				aria-label={m.editor_toolbar_bullet_list()}
				class="rounded-md px-2 py-1 text-xs leading-none text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				class:bg-zinc-900={isActive('bulletList')}
				class:text-white={isActive('bulletList')}
				class:dark:bg-zinc-100={isActive('bulletList')}
				class:dark:text-zinc-900={isActive('bulletList')}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleBulletList().run();
					})}
			>
				<ListBullets class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_numbered_list()}
				aria-label={m.editor_toolbar_numbered_list()}
				class="rounded-md px-2 py-1 text-xs leading-none text-zinc-700 transition-colors hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800"
				class:bg-zinc-900={isActive('orderedList')}
				class:text-white={isActive('orderedList')}
				class:dark:bg-zinc-100={isActive('orderedList')}
				class:dark:text-zinc-900={isActive('orderedList')}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleOrderedList().run();
					})}
			>
				<ListNumbers class="h-4 w-4" />
			</button>
			</div>
		</div>
	</div>

	<div class="h-full w-full overflow-y-auto">
		<div bind:this={editorElement} class="h-full w-full"></div>
	</div>
</div>
