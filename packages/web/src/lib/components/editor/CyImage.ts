import Image from '@tiptap/extension-image';

export const cyImageWidths = ['auto', '40%', '60%', '80%', '100%'] as const;
export const cyImageAlignments = ['content', 'full'] as const;

export const CyImage = Image.extend({
	addAttributes() {
		return {
			...this.parent?.(),
			alt: {
				default: null,
				parseHTML: (element) => element.getAttribute('alt'),
				renderHTML: (attributes) => {
					const alt = typeof attributes.alt === 'string' ? attributes.alt.trim() : '';
					if (alt !== '') {
						return { alt };
					}

					const title = typeof attributes.title === 'string' ? attributes.title.trim() : '';
					if (title !== '') {
						return { alt: title };
					}

					return {};
				}
			},
			width: {
				default: null,
				parseHTML: (element) =>
					element.getAttribute('data-display-width') ||
					element.getAttribute('width') ||
					(element instanceof HTMLElement ? element.style.width || null : null),
				renderHTML: (attributes) => {
					const width =
						typeof attributes.width === 'string' && attributes.width.trim() !== ''
							? attributes.width.trim()
							: null;

					if (!width || width === 'auto') {
						return {};
					}

					return {
						'data-display-width': width,
						style: `width: ${width};`
					};
				}
			},
			align: {
				default: 'content',
				parseHTML: (element) => element.getAttribute('data-display-align') || 'content',
				renderHTML: (attributes) => {
					const align =
						typeof attributes.align === 'string' && attributes.align.trim() !== ''
							? attributes.align.trim()
							: 'content';

					if (align === 'content') {
						return {};
					}

					return {
						'data-display-align': align
					};
				}
			}
		};
	}
});
