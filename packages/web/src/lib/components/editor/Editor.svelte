<script lang="ts">
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { Editor } from '@tiptap/core';
	import type { Content, JSONContent } from '@tiptap/core';
	import Link from '@tiptap/extension-link';
	import Placeholder from '@tiptap/extension-placeholder';
	import StarterKit from '@tiptap/starter-kit';
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
	import { CyImage, cyImageAlignments, cyImageWidths } from '$lib/components/editor/CyImage';
	import ImageTitleControls from '$lib/components/editor/ImageTitleControls.svelte';
	import ExternalImageButton from '$lib/components/editor/ExternalImageButton.svelte';
	import HeadingLevelMenu from '$lib/components/editor/HeadingLevelMenu.svelte';
	import ImageLayoutControls from '$lib/components/editor/ImageLayoutControls.svelte';
	import ImageReplaceButton from '$lib/components/editor/ImageReplaceButton.svelte';
	import LinkControls from '$lib/components/editor/LinkControls.svelte';
	import ImageSizeControls from '$lib/components/editor/ImageSizeControls.svelte';
	import ImageUploadButton from '$lib/components/editor/ImageUploadButton.svelte';
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
	let uploadingImageCount = $state(0);
	const imageUploadToastId = 'editor-image-upload';

	const allowedImageMimeTypes = new Set([
		'image/png',
		'image/jpeg',
		'image/webp',
		'image/gif'
	]);
	const allowedImageExtensions = new Set(['png', 'jpg', 'jpeg', 'webp', 'gif']);
	const imageUploadAccept = '.png,.jpg,.jpeg,.webp,.gif,image/png,image/jpeg,image/webp,image/gif';
	const headingLevels = [1, 2, 3, 4, 5, 6] as const;
	const externalImagePathPattern = /\.(avif|gif|jpe?g|png|svg|webp)(?:$|[?#])/i;

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

	function isSupportedImageFile(file: File): boolean {
		const type = file.type.trim().toLowerCase();
		if (type && allowedImageMimeTypes.has(type)) {
			return true;
		}

		const ext = file.name.split('.').pop()?.trim().toLowerCase() ?? '';
		return ext !== '' && allowedImageExtensions.has(ext);
	}

	function showUnsupportedImageUploadToast(file: File) {
		const ext = file.name.split('.').pop()?.trim().toLowerCase() ?? 'unknown';
		toast.error(`暂不支持上传 ${ext.toUpperCase()}，请使用 PNG/JPG/WebP/GIF`);
	}

	function insertUploadedImage(attrs: Record<string, unknown>) {
		if (!editor) return;

		editor
			.chain()
			.focus()
			.insertContent([
				{
					type: 'image',
					attrs
				},
				{
					type: 'paragraph'
				}
			])
			.run();
	}

	function buildExternalImageTitle(src: string): string {
		try {
			const parsed = new URL(src);
			const filename = parsed.pathname.split('/').pop()?.trim() ?? '';
			return filename || parsed.hostname;
		} catch {
			return '';
		}
	}

	function normalizeExternalImageURL(raw: string): string {
		const trimmed = raw.trim();
		if (!trimmed) return '';

		const candidate = /^[a-zA-Z][a-zA-Z\d+\-.]*:/.test(trimmed) ? trimmed : `https://${trimmed}`;
		try {
			const parsed = new URL(candidate);
			if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
				return '';
			}
			return parsed.toString();
		} catch {
			return '';
		}
	}

	function isExternalImageURL(raw: string): boolean {
		const normalized = normalizeExternalImageURL(raw);
		if (!normalized) return false;

		try {
			const parsed = new URL(normalized);
			return externalImagePathPattern.test(`${parsed.pathname}${parsed.search}${parsed.hash}`);
		} catch {
			return false;
		}
	}

	function insertExternalImage(src: string): boolean {
		const normalized = normalizeExternalImageURL(src);
		if (!normalized || !isExternalImageURL(normalized)) {
			toast.error(m.editor_external_image_invalid());
			return false;
		}

		insertUploadedImage({
			src: normalized,
			title: buildExternalImageTitle(normalized),
			alt: buildExternalImageTitle(normalized)
		});
		return true;
	}

	function beginImageUpload() {
		uploadingImageCount += 1;
		toast.loading(
			uploadingImageCount > 1
				? `正在上传 ${uploadingImageCount} 张图片...`
				: m.common_uploading(),
			{ id: imageUploadToastId, duration: Infinity }
		);
	}

	function endImageUpload() {
		uploadingImageCount = Math.max(0, uploadingImageCount - 1);
		if (uploadingImageCount > 0) {
			toast.loading(`正在上传 ${uploadingImageCount} 张图片...`, {
				id: imageUploadToastId,
				duration: Infinity
			});
			return;
		}

		toast.dismiss(imageUploadToastId);
	}

	async function uploadAndInsertImage(
		file: File,
		source: 'picker' | 'paste' = 'picker'
	): Promise<boolean> {
		if (!editor) return false;
		if (!isSupportedImageFile(file)) {
			showUnsupportedImageUploadToast(file);
			return false;
		}
		beginImageUpload();
		try {
			const uploaded = await uploadDocumentAsset(documentId, file, 'private');
			insertUploadedImage({
				src: uploaded.url,
				alt: file.name,
				title: file.name,
				assetId: uploaded.assetId
			});
			return true;
		} catch (error) {
			console.error(`[${source === 'paste' ? 'Paste' : 'Upload'}] Failed to upload image:`, error);
			toast.error(error instanceof Error ? error.message : '上传图片失败');
			return false;
		} finally {
			endImageUpload();
		}
	}

	async function uploadAndInsertImages(files: Iterable<File>, source: 'picker' | 'paste' = 'picker') {
		const supportedFiles: File[] = [];
		let blockedCount = 0;
		let uploadedCount = 0;

		for (const file of files) {
			if (!isSupportedImageFile(file)) {
				blockedCount += 1;
				continue;
			}
			supportedFiles.push(file);
		}

		if (blockedCount > 0) {
			toast.error(
				blockedCount === 1
					? '检测到 1 个不支持的图片文件，已跳过。仅支持 PNG/JPG/WebP/GIF。'
					: `检测到 ${blockedCount} 个不支持的图片文件，已跳过。仅支持 PNG/JPG/WebP/GIF。`
			);
		}

		for (const file of supportedFiles) {
			const uploaded = await uploadAndInsertImage(file, source);
			if (uploaded) {
				uploadedCount += 1;
			}
		}

		if (uploadedCount > 0 && uploadingImageCount === 0) {
			toast.success(
				uploadedCount === 1 ? '图片上传完成' : `${uploadedCount} 张图片上传完成`
			);
		}
	}

	function hasClipboardFiles(clipboard: DataTransfer): boolean {
		return Array.from(clipboard.items).some((item) => item.kind === 'file') || clipboard.files.length > 0;
	}

	function collectClipboardImageFiles(clipboard: DataTransfer): File[] {
		const imageTypes = new Set(['image/png', 'image/jpeg', 'image/webp', 'image/gif']);

		const files = [
			...Array.from(clipboard.items)
				.filter((item) => item.kind === 'file' && imageTypes.has(item.type))
				.map((item) => item.getAsFile())
				.filter((file): file is File => file !== null),
			...Array.from(clipboard.files).filter((file) => imageTypes.has(file.type))
		];

		const uniqueFiles = new Map<string, File>();
		for (const file of files) {
			const key = `${file.name}:${file.size}:${file.type}:${file.lastModified}`;
			if (!uniqueFiles.has(key)) {
				uniqueFiles.set(key, file);
			}
		}

		return [...uniqueFiles.values()];
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
				CyImage.configure({
					inline: false,
					allowBase64: true
				}),
				Link.configure({
					openOnClick: false,
					autolink: true,
					defaultProtocol: 'https'
				}),
				Table.configure({
					resizable: true,
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

						const clipboardFiles = collectClipboardImageFiles(clipboard);
						if (clipboardFiles.length > 0) {
							clipboardEvent.preventDefault();
							void (async () => {
								await uploadAndInsertImages(clipboardFiles, 'paste');
							})();
							return true;
						}

						if (hasClipboardFiles(clipboard)) {
							clipboardEvent.preventDefault();
							toast.error('当前仅支持粘贴 PNG/JPG/WebP/GIF 图片，其他文件请先导出为图片后再上传。');
							return true;
						}

						const html = clipboard.getData('text/html');
						const imageSources = extractImageSourcesFromHTML(html).filter((src) =>
							src.startsWith('data:image/') || src.startsWith('http://') || src.startsWith('https://')
						);
						if (imageSources.length > 0) {
							clipboardEvent.preventDefault();
							void (async () => {
								let blockedSourceCount = 0;
								for (const src of imageSources) {
									const file = await srcToUploadFile(src);
									if (!file) {
										blockedSourceCount += 1;
										continue;
									}
									await uploadAndInsertImage(file, 'paste');
								}

								if (blockedSourceCount > 0) {
									toast.error(
										'检测到不支持或无法读取的粘贴图片内容，已跳过。仅支持 PNG/JPG/WebP/GIF。'
									);
								}
							})();
							return true;
						}

						const text = clipboard.getData('text/plain');
						if (isExternalImageURL(text.trim())) {
							clipboardEvent.preventDefault();
							insertExternalImage(text);
							return true;
						}

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

	function currentImageWidth() {
		editorRevision;
		if (!editor || !editor.isActive('image')) return 'auto';
		const attrs = editor.getAttributes('image');
		const width = typeof attrs.width === 'string' ? attrs.width : '';
		return cyImageWidths.includes(width as (typeof cyImageWidths)[number]) ? width : 'auto';
	}

	function currentImageAlign() {
		editorRevision;
		if (!editor || !editor.isActive('image')) return 'content';
		const attrs = editor.getAttributes('image');
		const align = typeof attrs.align === 'string' ? attrs.align : 'content';
		return cyImageAlignments.includes(align as (typeof cyImageAlignments)[number]) ? align : 'content';
	}

	function currentImageTitle() {
		editorRevision;
		if (!editor || !editor.isActive('image')) return '';
		const attrs = editor.getAttributes('image');
		return typeof attrs.title === 'string' ? attrs.title : '';
	}

	function currentLinkHref() {
		editorRevision;
		if (!editor || !editor.isActive('link')) return '';
		const attrs = editor.getAttributes('link');
		return typeof attrs.href === 'string' ? attrs.href : '';
	}

	function applyImageWidth(width: string) {
		if (!editor || !editor.isActive('image')) {
			return;
		}

		editor
			.chain()
			.focus()
			.updateAttributes('image', {
				width: width === 'auto' ? null : width
			})
			.run();
		editorRevision += 1;
	}

	function applyImageAlign(align: string) {
		if (!editor || !editor.isActive('image')) {
			return;
		}

		editor
			.chain()
			.focus()
			.updateAttributes('image', {
				align
			})
			.run();
		editorRevision += 1;
	}

	function applyImageTitle(title: string) {
		if (!editor || !editor.isActive('image')) {
			return;
		}

		editor
			.chain()
			.focus()
			.updateAttributes('image', {
				title
			})
			.run();
		editorRevision += 1;
	}

	async function replaceCurrentImage(file: File) {
		if (!editor || !editor.isActive('image')) {
			return;
		}
		if (!isSupportedImageFile(file)) {
			showUnsupportedImageUploadToast(file);
			return;
		}

		beginImageUpload();
		try {
			const uploaded = await uploadDocumentAsset(documentId, file, 'private');
			editor
				.chain()
				.focus()
				.updateAttributes('image', {
					src: uploaded.url,
					assetId: uploaded.assetId,
					title: file.name
				})
				.run();
			editorRevision += 1;
			toast.success('图片替换完成');
		} catch (error) {
			console.error('[Replace] Failed to replace image:', error);
			toast.error(error instanceof Error ? error.message : '替换图片失败');
		} finally {
			endImageUpload();
		}
	}

	function normalizeLinkHref(href: string) {
		const trimmed = href.trim();
		if (!trimmed) return '';

		// If the href already has a scheme, only allow a safe subset.
		const schemeMatch = trimmed.match(/^([a-zA-Z][a-zA-Z\d+\-.]*:)/);
		if (schemeMatch) {
			const scheme = schemeMatch[1].toLowerCase();
			const allowedSchemes = new Set(['http:', 'https:', 'mailto:']);
			if (!allowedSchemes.has(scheme)) {
				// Reject unsafe or unknown schemes like javascript:, data:, etc.
				return '';
			}
			return trimmed;
		}

		// No explicit scheme: default to https.
		return `https://${trimmed}`;
	}

	function applyLinkHref(href: string) {
		if (!editor) return;
		const normalizedHref = normalizeLinkHref(href);
		if (!normalizedHref) {
			editor.chain().focus().unsetLink().run();
			editorRevision += 1;
			return;
		}

		editor
			.chain()
			.focus()
			.extendMarkRange('link')
			.setLink({ href: normalizedHref })
			.run();
		editorRevision += 1;
	}

	function removeLink() {
		if (!editor) return;
		editor.chain().focus().extendMarkRange('link').unsetLink().run();
		editorRevision += 1;
	}

	const activeToggleClass = 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900';
	const inactiveToggleClass =
		'text-zinc-700 hover:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800';
	const iconButtonBaseClass =
		'inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md leading-none transition-colors disabled:cursor-not-allowed disabled:opacity-50';
</script>

<div class="flex h-full w-full flex-col">
	<div class="relative z-10 border-b border-zinc-200 px-3 py-2 dark:border-zinc-800">
		<div class="overflow-visible">
			<div
				class="mx-auto flex w-full max-w-6xl flex-nowrap items-center justify-start gap-2 overflow-x-auto whitespace-nowrap scrollbar-none md:justify-center"
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
			<LinkControls href={currentLinkHref()} onSave={applyLinkHref} onRemove={removeLink} />
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
				class={`rounded-md px-2 py-1 text-xs leading-none transition-colors ${inactiveToggleClass}`}
				onclick={() =>
					apply((instance) => {
						instance.chain().focus().setHorizontalRule().run();
					})}
			>
				<Minus class="h-4 w-4" />
			</button>
			<div class="mx-0.5 h-5 w-px shrink-0 bg-zinc-200 dark:bg-zinc-700 md:mx-1"></div>
			<ImageUploadButton
				accept={imageUploadAccept}
				label={m.editor_toolbar_upload_image()}
				uploadingLabel={m.common_uploading()}
				isUploading={uploadingImageCount > 0}
				onFilesSelected={(files) => {
					void uploadAndInsertImages(Array.from(files), 'picker');
				}}
			/>
			<ExternalImageButton
				onInsert={(src) => insertExternalImage(src)}
			/>
			{#if isActive('image')}
				<div
					in:fade={{ duration: 120 }}
					out:fade={{ duration: 120 }}
					class="inline-flex shrink-0 items-center gap-1 rounded-lg px-1 outline outline-1 -outline-offset-1 outline-zinc-200 dark:outline-zinc-700"
				>
					<ImageReplaceButton
						accept={imageUploadAccept}
						label={m.editor_image_replace()}
						onFileSelected={(file) => {
							void replaceCurrentImage(file);
						}}
					/>
					<ImageTitleControls value={currentImageTitle()} onSave={applyImageTitle} />
					<ImageSizeControls currentWidth={currentImageWidth()} onSelect={applyImageWidth} />
					<ImageLayoutControls currentAlign={currentImageAlign()} onSelect={applyImageAlign} />
				</div>
			{/if}
				<TableToolbarControls
				isTableActive={isActive('table')}
				isHeaderRowActive={isActive('tableHeader')}
				canInsertTable={canApply((instance) =>
					instance.can().chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
				)}
				canAddRow={canApply((instance) => instance.can().chain().focus().addRowAfter().run())}
				canDeleteRow={canApply((instance) => instance.can().chain().focus().deleteRow().run())}
				canAddColumn={canApply((instance) => instance.can().chain().focus().addColumnAfter().run())}
				canDeleteColumn={canApply((instance) => instance.can().chain().focus().deleteColumn().run())}
				canToggleHeaderRow={canApply((instance) => instance.can().chain().focus().toggleHeaderRow().run())}
				canDeleteTable={canApply((instance) => instance.can().chain().focus().deleteTable().run())}
				onInsertTable={(rows, cols) =>
					apply((instance) => {
						instance.chain().focus().insertTable({ rows, cols, withHeaderRow: true }).run();
					})}
				onAddRow={() =>
					apply((instance) => {
						instance.chain().focus().addRowAfter().run();
					})}
				onDeleteRow={() =>
					apply((instance) => {
						instance.chain().focus().deleteRow().fixTables().run();
					})}
				onAddColumn={() =>
					apply((instance) => {
						instance.chain().focus().addColumnAfter().run();
					})}
				onDeleteColumn={() =>
					apply((instance) => {
						instance.chain().focus().deleteColumn().fixTables().run();
					})}
				onToggleHeaderRow={() =>
					apply((instance) => {
						instance.chain().focus().toggleHeaderRow().fixTables().run();
					})}
				onDeleteTable={() =>
					apply((instance) => {
						instance.chain().focus().deleteTable().run();
					})}
				/>
			</div>
		</div>
	</div>

	<div class="h-full w-full overflow-y-auto">
		<div bind:this={editorElement} class="h-full w-full"></div>
	</div>
</div>
