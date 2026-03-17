<script lang="ts">
	import { onMount } from 'svelte';
	import { Editor } from '@tiptap/core';
	import type { Content, JSONContent } from '@tiptap/core';
	import Placeholder from '@tiptap/extension-placeholder';
	import StarterKit from '@tiptap/starter-kit';
	import Image from '@tiptap/extension-image';
	import { Table } from '@tiptap/extension-table';
	import { TableCell } from '@tiptap/extension-table-cell';
	import { TableHeader } from '@tiptap/extension-table-header';
	import { TableRow } from '@tiptap/extension-table-row';
	import { marked } from 'marked';
	import * as m from '$paraglide/messages';
	import TextB from '~icons/ph/text-b';
	import TextItalic from '~icons/ph/text-italic';
	import ListBullets from '~icons/ph/list-bullets';
	import ListNumbers from '~icons/ph/list-numbers';
	import FloppyDisk from '~icons/ph/floppy-disk';
	import ArrowCounterClockwise from '~icons/ph/arrow-counter-clockwise';
	import ArrowClockwise from '~icons/ph/arrow-clockwise';
	import Quotes from '~icons/ph/quotes';
	import Code from '~icons/ph/code';
	import Minus from '~icons/ph/minus';
	import HeadingLevelMenu from '$lib/components/editor/HeadingLevelMenu.svelte';
	import TableToolbarControls from '$lib/components/editor/TableToolbarControls.svelte';
	import { uploadDocumentAsset } from '$lib/api/editor';
	import { toast } from 'svelte-sonner';

	interface Props {
		documentId: string;
		content: JSONContent;
		isSaving?: boolean;
		hasUnsavedChanges?: boolean;
		onContentChange?: (content: JSONContent) => void;
		onSave?: () => void | Promise<unknown>;
	}

	let {
		documentId,
		content,
		isSaving = false,
		hasUnsavedChanges = false,
		onContentChange,
		onSave
	}: Props = $props();

	const EMPTY_DOC: JSONContent = {
		type: 'doc',
		content: [{ type: 'paragraph' }]
	};

	let editorElement: HTMLDivElement | null = null;
	let editor: Editor | null = null;
	let lastSyncedContent = '';
	let editorRevision = $state(0);

	const allowedUploadMimeTypes = new Set([
		'image/png',
		'image/jpeg',
		'image/webp',
		'image/gif',
		'video/mp4',
		'video/webm'
	]);

	const allowedUploadExtensions = new Set(['png', 'jpg', 'jpeg', 'webp', 'gif', 'mp4', 'webm']);
	const headingLevels = [1, 2, 3, 4, 5, 6] as const;

	function sanitizePastedHTML(html: string): string {
		const parser = new DOMParser();
		const doc = parser.parseFromString(html, 'text/html');

		doc.querySelectorAll('script, style, meta, link').forEach((node) => node.remove());

		doc.querySelectorAll('img').forEach((img) => {
			const src = img.getAttribute('src')?.trim();
			if (!src) {
				img.remove();
				return;
			}
		});

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

	function looksLikeMarkdown(text: string): boolean {
		const sample = text.trim();
		if (!sample) return false;

		const markdownPatterns = [
			/^#{1,6}\s/m,
			/^\s*[-*+]\s+/m,
			/^\s*\d+\.\s+/m,
			/```[\s\S]*```/m,
			/`[^`\n]+`/,
			/\[[^\]]+\]\([^)]+\)/,
			/!\[[^\]]*\]\([^)]+\)/,
			/\*\*[^*\n]+\*\*/,
			/\*[^*\n]+\*/,
			/^>\s+/m
		];

		return markdownPatterns.some((pattern) => pattern.test(sample));
	}

	function normalizeDoc(value: JSONContent | null | undefined): JSONContent {
		if (!value || value.type !== 'doc') {
			return EMPTY_DOC;
		}
		if (!Array.isArray(value.content) || value.content.length === 0) {
			return EMPTY_DOC;
		}
		return value;
	}

	function toTiptapContent(value: JSONContent): Content {
		return normalizeDoc(value);
	}

	function serializeDoc(value: JSONContent): string {
		return JSON.stringify(normalizeDoc(value));
	}

	function isSupportedUploadFile(file: File): boolean {
		const type = file.type.trim().toLowerCase();
		if (type && allowedUploadMimeTypes.has(type)) {
			return true;
		}

		const ext = file.name.split('.').pop()?.trim().toLowerCase() ?? '';
		return ext !== '' && allowedUploadExtensions.has(ext);
	}

	function showUnsupportedUploadToast(file: File) {
		const ext = file.name.split('.').pop()?.trim().toLowerCase() ?? 'unknown';
		toast.error(`暂不支持上传 ${ext.toUpperCase()}，请使用 PNG/JPG/WebP/GIF/MP4/WebM`);
	}

	async function uploadAndInsertImage(file: File) {
		if (!editor) return;
		if (!isSupportedUploadFile(file)) {
			showUnsupportedUploadToast(file);
			return;
		}
		try {
			const uploaded = await uploadDocumentAsset(documentId, file, 'private');
			editor
				.chain()
				.focus()
				.insertContent({
					type: 'image',
					attrs: {
						src: uploaded.url,
						alt: file.name,
						title: file.name,
						assetId: uploaded.assetId
					}
				})
				.run();
		} catch (error) {
			console.error('[Upload] Failed to upload image:', error);
			toast.error(error instanceof Error ? error.message : '上传资源失败');
		}
	}

	function extractImageSourcesFromHTML(html: string): string[] {
		if (!html) return [];
		const parser = new DOMParser();
		const doc = parser.parseFromString(html, 'text/html');
		return Array.from(doc.querySelectorAll('img'))
			.map((img) => img.getAttribute('src')?.trim() ?? '')
			.filter((src) => src.length > 0);
	}

	async function srcToUploadFile(src: string): Promise<File | null> {
		try {
			const response = await fetch(src);
			if (!response.ok) return null;
			const blob = await response.blob();
			const ext = blob.type.split('/')[1] || 'png';
			return new File([blob], `pasted-image.${ext}`, { type: blob.type || 'image/png' });
		} catch {
			return null;
		}
	}

	onMount(() => {
		if (!editorElement) {
			return;
		}

		lastSyncedContent = serializeDoc(content);
		editor = new Editor({
			element: editorElement,
			extensions: [
				StarterKit.configure({
					heading: {
						levels: [...headingLevels]
					}
				}),
				Image.configure({
					inline: false,
					allowBase64: true
				}),
				Table.configure({
					resizable: false,
					HTMLAttributes: {
						class: 'cw-editor-table'
					}
				}),
				TableRow,
				TableHeader,
				TableCell,
				Placeholder.configure({
					placeholder: m.editor_placeholder()
				})
			],
			content: toTiptapContent(content),
			editorProps: {
				transformPastedHTML: (html) => sanitizePastedHTML(html),
				handleDOMEvents: {
					paste: (_view, event) => {
						const clipboardEvent = event as ClipboardEvent;
						const clipboard = clipboardEvent.clipboardData;
						if (!clipboard) return false;

						const imageFiles = [
							...Array.from(clipboard.items)
								.filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
								.map((item) => item.getAsFile())
								.filter((file): file is File => file !== null),
							...Array.from(clipboard.files).filter((file) => file.type.startsWith('image/'))
						];

						if (imageFiles.length > 0) {
							clipboardEvent.preventDefault();
							void (async () => {
								for (const file of imageFiles) {
									try {
										await uploadAndInsertImage(file);
									} catch (error) {
										console.error('[Paste] Failed to upload pasted image file:', error);
									}
								}
							})();
							return true;
						}

						const html = clipboard.getData('text/html');
						const imageSources = extractImageSourcesFromHTML(html).filter((src) =>
							src.startsWith('data:image/') || src.startsWith('http://') || src.startsWith('https://')
						);
						if (imageSources.length > 0) {
							clipboardEvent.preventDefault();
							void (async () => {
								for (const src of imageSources) {
									const file = await srcToUploadFile(src);
									if (!file) continue;
									try {
										await uploadAndInsertImage(file);
									} catch (error) {
										console.error('[Paste] Failed to upload pasted image element:', error);
									}
								}
							})();
							return true;
						}

						const text = clipboard.getData('text/plain');
						if (!looksLikeMarkdown(text)) return false;

						const rendered = marked.parse(text, {
							async: false,
							gfm: true,
							breaks: true
						});
						if (typeof rendered !== 'string') return false;

						clipboardEvent.preventDefault();
						editor
							?.chain()
							.focus()
							.insertContent(sanitizePastedHTML(rendered))
							.run();
						return true;
					}
				},
				attributes: {
					class:
						'tiptap min-h-full w-full px-4 py-6 text-base text-zinc-800 outline-none dark:text-zinc-100 sm:px-8 lg:px-[14%]'
				}
			},
			onUpdate: ({ editor }) => {
				const nextContent = editor.getJSON();
				lastSyncedContent = serializeDoc(nextContent);
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

		if (serializeDoc(content) === lastSyncedContent) {
			return;
		}

		lastSyncedContent = serializeDoc(content);
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

	function canApply(action: (instance: Editor) => boolean) {
		editorRevision;
		if (!editor) return false;
		return action(editor);
	}

	function currentHeadingValue() {
		editorRevision;
		if (!editor) return 'paragraph';
		for (const level of headingLevels) {
			if (editor.isActive('heading', { level })) {
				return `h${level}`;
			}
		}
		return 'paragraph';
	}

	function applyHeadingValue(value: string) {
		if (!editor) return;
		if (value === 'paragraph') {
			editor.chain().focus().setParagraph().run();
			editorRevision += 1;
			return;
		}

		const level = Number.parseInt(value.replace('h', ''), 10);
		if (!headingLevels.includes(level as (typeof headingLevels)[number])) {
			return;
		}

		editor.chain().focus().setHeading({ level: level as (typeof headingLevels)[number] }).run();
		editorRevision += 1;
	}

	const activeToggleClass = 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900';
	const inactiveToggleClass =
		'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800';
	const iconButtonBaseClass =
		'inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md leading-none transition-colors disabled:cursor-not-allowed disabled:opacity-50';
</script>

<div class="flex h-full w-full flex-col">
	<div class="border-b border-zinc-200 px-3 py-2 dark:border-zinc-800">
		<div
			class="mx-auto flex w-full max-w-4xl flex-nowrap items-center justify-start gap-2 overflow-x-auto whitespace-nowrap scrollbar-none md:flex-wrap md:justify-center md:overflow-visible md:whitespace-normal"
		>
			<button
				type="button"
				title={m.editor_toolbar_save_with_shortcut()}
				aria-label={m.editor_toolbar_save_with_shortcut()}
				disabled={isSaving || !hasUnsavedChanges}
				onclick={() => onSave?.()}
				class={`${iconButtonBaseClass} text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800`}
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
				class={`${iconButtonBaseClass} text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800`}
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
				class={`${iconButtonBaseClass} text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800`}
			>
				<ArrowClockwise class="h-4 w-4" />
			</button>
			<div class="mx-0.5 h-5 w-px shrink-0 bg-zinc-200 dark:bg-zinc-700 md:mx-1"></div>
			<HeadingLevelMenu currentValue={currentHeadingValue()} onSelect={applyHeadingValue} />
			<div class="mx-0.5 h-5 w-px shrink-0 bg-zinc-200 dark:bg-zinc-700 md:mx-1"></div>
			<button
				type="button"
				title={m.editor_toolbar_bold()}
				aria-label={m.editor_toolbar_bold()}
				class={`rounded-md px-2 py-1 text-xs font-semibold leading-none transition-colors ${
					isActive('bold') ? activeToggleClass : inactiveToggleClass
				}`}
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
				class={`rounded-md px-2 py-1 text-xs italic leading-none transition-colors ${
					isActive('italic') ? activeToggleClass : inactiveToggleClass
				}`}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleItalic().run();
					})}
			>
				<TextItalic class="h-4 w-4" />
			</button>
			<div class="mx-0.5 h-5 w-px shrink-0 bg-zinc-200 dark:bg-zinc-700 md:mx-1"></div>
			<button
				type="button"
				title={m.editor_toolbar_bullet_list()}
				aria-label={m.editor_toolbar_bullet_list()}
				class={`rounded-md px-2 py-1 text-xs leading-none transition-colors ${
					isActive('bulletList') ? activeToggleClass : inactiveToggleClass
				}`}
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
				class={`rounded-md px-2 py-1 text-xs leading-none transition-colors ${
					isActive('orderedList') ? activeToggleClass : inactiveToggleClass
				}`}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleOrderedList().run();
					})}
			>
				<ListNumbers class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_blockquote()}
				aria-label={m.editor_toolbar_blockquote()}
				class={`rounded-md px-2 py-1 text-xs leading-none transition-colors ${
					isActive('blockquote') ? activeToggleClass : inactiveToggleClass
				}`}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleBlockquote().run();
					})}
			>
				<Quotes class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_code_block()}
				aria-label={m.editor_toolbar_code_block()}
				class={`rounded-md px-2 py-1 text-xs leading-none transition-colors ${
					isActive('codeBlock') ? activeToggleClass : inactiveToggleClass
				}`}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().toggleCodeBlock().run();
					})}
			>
				<Code class="h-4 w-4" />
			</button>
			<button
				type="button"
				title={m.editor_toolbar_divider()}
				aria-label={m.editor_toolbar_divider()}
				class={`rounded-md px-2 py-1 text-xs leading-none ${inactiveToggleClass}`}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().setHorizontalRule().run();
					})}
			>
				<Minus class="h-4 w-4" />
			</button>
			<div class="mx-0.5 h-5 w-px shrink-0 bg-zinc-200 dark:bg-zinc-700 md:mx-1"></div>
			<TableToolbarControls
				isTableActive={isActive('table')}
				canInsertTable={canApply((instance) =>
					instance.can().chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
				)}
				canAddRow={canApply((instance) => instance.can().chain().focus().addRowAfter().run())}
				canAddColumn={canApply((instance) => instance.can().chain().focus().addColumnAfter().run())}
				canDeleteTable={canApply((instance) => instance.can().chain().focus().deleteTable().run())}
				onInsertTable={() =>
					apply((instance) => {
						instance.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run();
					})}
				onAddRow={() =>
					apply((instance) => {
						instance.chain().focus().addRowAfter().run();
					})}
				onAddColumn={() =>
					apply((instance) => {
						instance.chain().focus().addColumnAfter().run();
					})}
				onDeleteTable={() =>
					apply((instance) => {
						instance.chain().focus().deleteTable().run();
					})}
			/>
		</div>
	</div>

	<div class="h-full w-full overflow-y-auto">
		<div bind:this={editorElement} class="h-full w-full"></div>
	</div>
</div>
