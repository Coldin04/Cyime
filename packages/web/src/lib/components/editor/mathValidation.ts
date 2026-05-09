import type { JSONContent } from '@tiptap/core';

export const MAX_MATH_LATEX_LENGTH = 2000;
export const KATEX_MAX_SIZE = 10;
export const KATEX_MAX_EXPAND = 200;

const mathNodeTypes = new Set(['inlineMath', 'blockMath']);

export function normalizeMathLatexInput(value: unknown): string | null {
	if (typeof value !== 'string') {
		return null;
	}

	const latex = value.trim();
	if (latex === '' || latex.length > MAX_MATH_LATEX_LENGTH) {
		return null;
	}

	return latex;
}

export function sanitizeMathLatexAttr(value: unknown): string {
	return normalizeMathLatexInput(value) ?? '';
}

export function sanitizeMathContent(value: JSONContent): JSONContent {
	let changed = false;
	const next: JSONContent = { ...value };

	if (Array.isArray(value.content)) {
		const content = value.content.map((child) => {
			const sanitizedChild = sanitizeMathContent(child);
			if (sanitizedChild !== child) {
				changed = true;
			}
			return sanitizedChild;
		});
		if (changed) {
			next.content = content;
		}
	}

	if (value.type && mathNodeTypes.has(value.type)) {
		const latex = sanitizeMathLatexAttr(value.attrs?.latex);
		if (latex !== value.attrs?.latex) {
			next.attrs = { ...value.attrs, latex };
			changed = true;
		}
	}

	return changed ? next : value;
}
