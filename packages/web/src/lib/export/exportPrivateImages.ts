import type { JSONContent } from '@tiptap/core';
import {
	copyToClipboard,
	downloadTextFile,
	exportBBCode,
	exportHtmlDocument,
	exportMarkdown
} from '$lib/export/documentExport';
import type { ExportAction } from '$lib/export/exportActions';

type ImageNodeRecord = Record<string, unknown> & {
	attrs?: Record<string, unknown>;
	content?: unknown[];
};

export type ManagedImageUsage = {
	assetId: string;
	src: string | null;
	title: string | null;
	alt: string | null;
};

export function cloneContentJson(value: JSONContent): JSONContent {
	return JSON.parse(JSON.stringify(value)) as JSONContent;
}

export function collectImageNodes(value: unknown, nodes: ImageNodeRecord[]) {
	if (!value || typeof value !== 'object') {
		return;
	}

	const node = value as ImageNodeRecord;
	if (node.type === 'image') {
		nodes.push(node);
	}

	const children = node.content;
	if (Array.isArray(children)) {
		for (const child of children) {
			collectImageNodes(child, nodes);
		}
	}
}

export function getManagedAssetId(attrs: Record<string, unknown>): string | null {
	const raw = attrs.assetId;
	return typeof raw === 'string' && raw.trim() !== '' ? raw.trim() : null;
}

export function collectManagedImages(content: JSONContent): ManagedImageUsage[] {
	const imageNodes: ImageNodeRecord[] = [];
	collectImageNodes(content, imageNodes);

	const usages: ManagedImageUsage[] = [];
	const seen = new Set<string>();
	for (const node of imageNodes) {
		const attrs = (node.attrs ?? {}) as Record<string, unknown>;
		const assetId = getManagedAssetId(attrs);
		if (!assetId || seen.has(assetId)) {
			continue;
		}

		seen.add(assetId);
		usages.push({
			assetId,
			src: typeof attrs.src === 'string' && attrs.src.trim() !== '' ? attrs.src.trim() : null,
			title: typeof attrs.title === 'string' && attrs.title.trim() !== '' ? attrs.title.trim() : null,
			alt: typeof attrs.alt === 'string' && attrs.alt.trim() !== '' ? attrs.alt.trim() : null
		});
	}

	return usages;
}

export function replaceManagedImagesWithPublicURLs(
	content: JSONContent,
	publicURLByAssetID: Map<string, string>
): JSONContent {
	const cloned = cloneContentJson(content);
	const imageNodes: ImageNodeRecord[] = [];
	collectImageNodes(cloned, imageNodes);

	for (const node of imageNodes) {
		const attrs = (node.attrs ?? {}) as Record<string, unknown>;
		const assetId = getManagedAssetId(attrs);
		if (!assetId) {
			continue;
		}

		const publicURL = publicURLByAssetID.get(assetId);
		if (!publicURL) {
			continue;
		}

		attrs.src = publicURL;
		delete attrs.assetId;
		node.attrs = attrs;
	}

	return cloned;
}

export function buildExportFilename(title: string, extension: 'html' | 'md'): string {
	const rawTitle = title.trim();
	const safeTitle = rawTitle.replace(/[\\/:*?"<>|]+/g, ' ').replace(/\s+/g, ' ').trim();
	return `${safeTitle || 'cyimewrite-export'}.${extension}`;
}

export async function runExportAction(action: ExportAction, options: {
	title: string;
	contentJson: JSONContent;
}): Promise<'copied' | 'downloaded'> {
	if (action === 'download-pdf') {
		throw new Error('download_pdf_requires_print_html');
	}

	if (action === 'copy-bbcode') {
		const bbcode = exportBBCode(options.contentJson);
		const copied = await copyToClipboard(bbcode);
		if (!copied) {
			throw new Error('copy_bbcode_failed');
		}
		return 'copied';
	}

	if (action === 'download-html') {
		const html = exportHtmlDocument({
			title: options.title.trim() || 'CyimeWrite Export',
			contentJson: options.contentJson
		});
		downloadTextFile(buildExportFilename(options.title, 'html'), html, 'text/html;charset=utf-8');
		return 'downloaded';
	}

	const markdown = exportMarkdown(options.contentJson);
	if (action === 'copy-markdown') {
		const copied = await copyToClipboard(markdown);
		if (!copied) {
			throw new Error('copy_markdown_failed');
		}
		return 'copied';
	}

	downloadTextFile(buildExportFilename(options.title, 'md'), markdown, 'text/markdown;charset=utf-8');
	return 'downloaded';
}

export async function inlineManagedImagesAsDataURLs(
	content: JSONContent,
	resolveAssetContentURL: (assetId: string) => string
): Promise<JSONContent> {
	const managedImages = collectManagedImages(content);
	if (managedImages.length === 0) {
		return cloneContentJson(content);
	}

	const dataURLByAssetID = new Map<string, string>();
	for (const item of managedImages) {
		const response = await fetch(resolveAssetContentURL(item.assetId), {
			credentials: 'include'
		});
		if (!response.ok) {
			throw new Error(`Failed to fetch private image ${item.assetId}`);
		}

		const blob = await response.blob();
		const dataURL = await new Promise<string>((resolve, reject) => {
			const reader = new FileReader();
			reader.onload = () => {
				if (typeof reader.result === 'string') {
					resolve(reader.result);
					return;
				}
				reject(new Error('Failed to encode image as data URL'));
			};
			reader.onerror = () => reject(reader.error ?? new Error('Failed to read image blob'));
			reader.readAsDataURL(blob);
		});
		dataURLByAssetID.set(item.assetId, dataURL);
	}

	return replaceManagedImagesWithPublicURLs(content, dataURLByAssetID);
}

export async function exportPdfDocument(options: {
	title: string;
	html: string;
}): Promise<void> {
	const iframe = document.createElement('iframe');
	iframe.style.position = 'fixed';
	iframe.style.right = '0';
	iframe.style.bottom = '0';
	iframe.style.width = '0';
	iframe.style.height = '0';
	iframe.style.border = '0';
	iframe.setAttribute('aria-hidden', 'true');
	document.body.appendChild(iframe);

	const cleanup = () => {
		window.setTimeout(() => {
			iframe.remove();
		}, 1000);
	};

	// Temporary browser-side PDF export: render a print-friendly HTML snapshot
	// and rely on the browser's native Print to PDF flow. This avoids adding
	// Chromium or a heavy server-side PDF stack for now.
	iframe.onload = () => {
		iframe.contentWindow?.focus();
		iframe.contentWindow?.print();
		cleanup();
	};
	iframe.srcdoc = options.html;
}

export { exportHtmlDocument };
