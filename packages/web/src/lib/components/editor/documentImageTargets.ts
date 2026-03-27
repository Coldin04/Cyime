import * as m from '$paraglide/messages';
import type { ImageBedConfig } from '$lib/api/user';

export type DocumentImageTargetOption = {
	id: string;
	label: string;
	description: string;
	providerType?: string;
};

const managedTargetId = 'managed-r2';

export function getBuiltInImageTargetOptions(): DocumentImageTargetOption[] {
	return [
		{
			id: managedTargetId,
			label: m.editor_image_target_managed_r2_label(),
			description: m.editor_image_target_managed_r2_description(),
			providerType: 'managed'
		}
	];
}

export function getUserImageTargetOptions(configs: ImageBedConfig[]): DocumentImageTargetOption[] {
	return configs
		.filter((config) => config.isEnabled)
		.map((config) => ({
			id: config.id,
			label: config.name,
			description: getImageBedProviderDescription(config),
			providerType: config.providerType
		}));
}

export function getDocumentImageTargetOptions(configs: ImageBedConfig[]): DocumentImageTargetOption[] {
	return [...getBuiltInImageTargetOptions(), ...getUserImageTargetOptions(configs)];
}

export function getDocumentImageTargetLabel(targetId: string, options: DocumentImageTargetOption[]): string {
	return options.find((option) => option.id === targetId)?.label ?? targetId;
}

function getImageBedProviderDescription(config: ImageBedConfig): string {
	switch (config.providerType) {
		case 'see':
			return m.editor_image_target_see_public_description();
		case 'lsky':
			return m.editor_image_target_lsky_public_description();
		default:
			return config.providerType;
	}
}

