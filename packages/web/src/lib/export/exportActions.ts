export type ExportAction = 'download-html' | 'copy-markdown' | 'download-markdown';

export type ExportActionCapability = {
	requiresPublicImageURLs: boolean;
};

export const exportActionCapabilities: Record<ExportAction, ExportActionCapability> = {
	'download-html': {
		requiresPublicImageURLs: true
	},
	'copy-markdown': {
		requiresPublicImageURLs: true
	},
	'download-markdown': {
		requiresPublicImageURLs: true
	}
};

export function exportActionRequiresPublicImageURLs(action: ExportAction): boolean {
	return exportActionCapabilities[action].requiresPublicImageURLs;
}
