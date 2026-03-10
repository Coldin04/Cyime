import { writable } from 'svelte/store';

export type BreadcrumbItem = {
	id: string;
	name: string;
};

export const breadcrumbItems = writable<BreadcrumbItem[]>([]);

// Workspace context store for sharing state between layout and pages
export const workspaceContext = writable<{
	currentFolderId: string | null;
	isCreatingFolder: boolean;
	bulkMode: boolean;
}>({
	currentFolderId: null,
	isCreatingFolder: false,
	bulkMode: false
});
