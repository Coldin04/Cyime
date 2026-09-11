import type { JSONContent } from '@tiptap/core';

type MarkdownNode = {
	type?: string;
	text?: string;
	attrs?: Record<string, unknown>;
	content?: MarkdownNode[];
	marks?: Array<{ type?: string; attrs?: Record<string, unknown> }>;
};

function escapeMarkdown(value: string): string {
	return value.replaceAll('[', '\\[').replaceAll(']', '\\]');
}

function normalizeCodeBlockLanguageInfo(value: unknown): string {
	if (typeof value !== 'string') {
		return '';
	}
	return value
		.trim()
		.replace(/[\s`]+/g, '')
		.slice(0, 32);
}

function getText(node: MarkdownNode | undefined): string {
	if (!node) return '';
	if (node.type === 'text') return node.text ?? '';
	return (node.content ?? []).map((child) => getText(child)).join('');
}

export function exportMarkdown(contentJson: JSONContent): string {
	const lines: string[] = [];

	function renderInline(nodes: MarkdownNode[] = []): string {
		let output = '';

		for (const node of nodes) {
			switch (node.type) {
				case 'text': {
					let text = node.text ?? '';
					const marks = node.marks ?? [];
					const isCode = marks.some((mark) => mark.type === 'code');
					const isBold = marks.some((mark) => mark.type === 'bold');
					const isItalic = marks.some((mark) => mark.type === 'italic');
					const linkMark = marks.find((mark) => mark.type === 'link');

					if (isCode) {
						text = `\`${text.replaceAll('`', '\\`')}\``;
					} else {
						if (isBold) text = `**${text}**`;
						if (isItalic) text = `*${text}*`;
					}

					if (linkMark?.attrs?.href && typeof linkMark.attrs.href === 'string') {
						text = `[${text}](${linkMark.attrs.href})`;
					}

					output += text;
					break;
				}
				case 'hardBreak':
					output += '  \n';
					break;
				case 'inlineMath':
					output += `$${typeof node.attrs?.latex === 'string' ? node.attrs.latex : ''}$`;
					break;
				case 'image': {
					const src = typeof node.attrs?.src === 'string' ? node.attrs.src : '';
					const title = typeof node.attrs?.title === 'string' ? node.attrs.title : '';
					const alt = typeof node.attrs?.alt === 'string' ? node.attrs.alt : title;
					output += `![${escapeMarkdown(alt)}](${src})`;
					break;
				}
				default:
					output += renderInline(node.content ?? []);
			}
		}

		return output;
	}

	function renderTableCell(node: MarkdownNode | undefined): string {
		if (!node) return '';

		return (node.content ?? [])
			.map((block) => renderInline(block.content ?? []))
			.join('<br>')
			.replace(/ {2}\n|\n/g, '<br>')
.replaceAll('\\', '\\\\').replaceAll('|', '\\|')
			.trim();
	}

	function renderTable(node: MarkdownNode): void {
		const rows = (node.content ?? []).filter((child) => child.type === 'tableRow');
		if (rows.length === 0) return;

		const renderedRows = rows.map((row) =>
			(row.content ?? [])
				.filter((cell) => cell.type === 'tableHeader' || cell.type === 'tableCell')
				.map((cell) => renderTableCell(cell))
		);
		const columnCount = Math.max(...renderedRows.map((row) => row.length));
		if (columnCount === 0) return;

		const padRow = (row: string[]) => [
			...row,
			...Array.from({ length: columnCount - row.length }, () => '')
		];
		const firstRowIsHeader = (rows[0].content ?? []).some((cell) => cell.type === 'tableHeader');
		const header = firstRowIsHeader
			? padRow(renderedRows[0])
			: Array.from({ length: columnCount }, () => '');
		const body = firstRowIsHeader ? renderedRows.slice(1) : renderedRows;
		const renderRow = (row: string[]) => `| ${padRow(row).join(' | ')} |`;

		lines.push(renderRow(header));
		lines.push(renderRow(Array.from({ length: columnCount }, () => '---')));
		for (const row of body) lines.push(renderRow(row));
		lines.push('');
	}

	function renderNode(node: MarkdownNode | undefined, depth = 0, orderedIndex = 1) {
		if (!node) return;

		switch (node.type) {
			case 'doc':
				for (const child of node.content ?? []) renderNode(child, depth);
				return;
			case 'paragraph':
				lines.push(renderInline(node.content ?? []));
				lines.push('');
				return;
			case 'heading': {
				const level = typeof node.attrs?.level === 'number' ? node.attrs.level : 1;
				lines.push(`${'#'.repeat(Math.min(6, Math.max(1, level)))} ${renderInline(node.content ?? [])}`);
				lines.push('');
				return;
			}
			case 'bulletList':
				for (const child of node.content ?? []) renderNode(child, depth + 1);
				lines.push('');
				return;
			case 'orderedList': {
				let nextIndex = 1;
				for (const child of node.content ?? []) {
					renderNode(child, depth + 1, nextIndex);
					nextIndex += 1;
				}
				lines.push('');
				return;
			}
			case 'listItem': {
				const firstParagraph = (node.content ?? []).find((child) => child.type === 'paragraph');
				const fallbackContent = firstParagraph?.content ?? node.content ?? [];
				const marker = orderedIndex > 0 ? `${orderedIndex}.` : '-';
				lines.push(`${'  '.repeat(Math.max(0, depth - 1))}${marker} ${renderInline(fallbackContent)}`);
				for (const child of node.content ?? []) {
					if (child.type === 'paragraph') continue;
					renderNode(child, depth);
				}
				return;
			}
			case 'blockquote': {
				const nestedLines: string[] = [];
				const originalLines = lines.splice(0, lines.length);
				for (const child of node.content ?? []) renderNode(child, depth);
				nestedLines.push(...lines);
				lines.splice(0, lines.length, ...originalLines);
				for (const line of nestedLines.filter((entry) => entry !== '')) {
					lines.push(`> ${line}`);
				}
				lines.push('');
				return;
			}
			case 'codeBlock':
				lines.push(`\`\`\`${normalizeCodeBlockLanguageInfo(node.attrs?.language)}`);
				lines.push(getText(node));
				lines.push('```');
				lines.push('');
				return;
			case 'horizontalRule':
				lines.push('---');
				lines.push('');
				return;
			case 'blockMath':
				lines.push('$$');
				lines.push(typeof node.attrs?.latex === 'string' ? node.attrs.latex : '');
				lines.push('$$');
				lines.push('');
				return;
			case 'table':
				renderTable(node);
				return;
			case 'image': {
				const src = typeof node.attrs?.src === 'string' ? node.attrs.src : '';
				if (src) {
					const title = typeof node.attrs?.title === 'string' ? node.attrs.title : '';
					const alt = typeof node.attrs?.alt === 'string' ? node.attrs.alt : title;
					lines.push(`![${escapeMarkdown(alt)}](${src})`);
					lines.push('');
				}
				return;
			}
			default: {
				const inline = renderInline(node.content ?? []);
				if (inline.trim() !== '') {
					lines.push(inline);
					lines.push('');
				}
			}
		}
	}

	renderNode(contentJson as MarkdownNode);

	while (lines.length > 0 && lines[lines.length - 1].trim() === '') {
		lines.pop();
	}

	return lines.join('\n');
}
