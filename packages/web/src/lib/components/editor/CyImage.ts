import Image from '@tiptap/extension-image';

export const cyImageWidths = ['auto', '40%', '60%', '80%', '100%'] as const;

export const CyImage = Image.extend({
	addAttributes() {
		return {
			...this.parent?.(),
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
			}
		};
	}
});
