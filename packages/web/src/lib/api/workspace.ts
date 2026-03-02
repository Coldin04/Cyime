import { apiFetch } from '$lib/api';

export type FileItem = {
	id: string;
	type: 'folder' | 'markdown';
	name: string;
	description?: string | null;
	parentId?: string | null;
	folderId?: string | null;
	title?: string | null;
	excerpt?: string | null;
	createdAt: string;
	updatedAt: string;
	creator: {
		id: string;
		displayName: string | null;
	};
};

export type FileListResponse = {
	items: FileItem[];
	hasMore: boolean;
	total: number;
};

export type CreateFolderRequest = {
	name: string;
	description?: string | null;
	parentId?: string | null;
};

export type CreateFolderResponse = FileItem;

export type CreateMarkdownRequest = {
	title: string;
	content: string;
	folderId?: string | null;
};

export type CreateMarkdownResponse = FileItem;

export type DeleteResponse = {
	success: boolean;
	message: string;
};

/**
 * Fetches the file list from the workspace
 */
export async function getFiles(params: {
	parent_id?: string | null;
	limit?: number;
	offset?: number;
	sort_by?: string;
	order?: string;
	type?: 'all' | 'folders' | 'markdowns';
}): Promise<FileListResponse> {
	const queryParams = new URLSearchParams();

	if (params.parent_id !== undefined && params.parent_id !== null) {
		queryParams.set('parent_id', params.parent_id);
	}
	if (params.limit !== undefined) {
		queryParams.set('limit', params.limit.toString());
	}
	if (params.offset !== undefined) {
		queryParams.set('offset', params.offset.toString());
	}
	if (params.sort_by !== undefined) {
		queryParams.set('sort_by', params.sort_by);
	}
	if (params.order !== undefined) {
		queryParams.set('order', params.order);
	}
	if (params.type !== undefined) {
		queryParams.set('type', params.type);
	}

	const response = await apiFetch(`/api/v1/workspace/files?${queryParams.toString()}`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch files');
	}

	return response.json();
}

/**
 * Creates a new folder
 */
export async function createFolder(request: CreateFolderRequest): Promise<CreateFolderResponse> {
	const response = await apiFetch('/api/v1/workspace/folders', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to create folder');
	}

	return response.json();
}

/**
 * Creates a new markdown document
 */
export async function createMarkdown(request: CreateMarkdownRequest): Promise<CreateMarkdownResponse> {
	const response = await apiFetch('/api/v1/workspace/markdowns', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to create markdown');
	}

	return response.json();
}

/**
 * Deletes a file (soft delete)
 */
export async function deleteFile(id: string, type: 'folder' | 'markdown'): Promise<DeleteResponse> {
	const response = await apiFetch(`/api/v1/workspace/files/${id}?type=${type}`, {
		method: 'DELETE'
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to delete file');
	}

	return response.json();
}

export type AncestorItem = {
	id: string;
	name: string;
};

/**
 * Fetches the ancestor path for a folder
 */
export async function getFolderAncestors(id: string): Promise<AncestorItem[]> {
	const response = await apiFetch(`/api/v1/workspace/folders/${id}/ancestors`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch folder ancestors');
	}

	return response.json();
}

// --- Trash API Types and Functions ---

export type TrashItem = {
	id: string;
	type: 'folder' | 'markdown';
	name: string;
	deletedAt: string;
};

export type TrashListResponse = {
	items: TrashItem[];
	hasMore: boolean;
	total: number;
};

/**
 * Fetches the file list from the trash
 */
export async function getTrashedFiles(params: {
	limit?: number;
	offset?: number;
	sort_by?: string;
	order?: string;
}): Promise<TrashListResponse> {
	const queryParams = new URLSearchParams();

	if (params.limit !== undefined) {
		queryParams.set('limit', params.limit.toString());
	}
	if (params.offset !== undefined) {
		queryParams.set('offset', params.offset.toString());
	}
	if (params.sort_by !== undefined) {
		queryParams.set('sort_by', params.sort_by);
	}
	if (params.order !== undefined) {
		queryParams.set('order', params.order);
	}

	const response = await apiFetch(`/api/v1/workspace/trash?${queryParams.toString()}`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch trashed files');
	}

	return response.json();
}

/**
 * Restores a list of items from the trash
 */
export async function restoreItems(items: { id: string; type: 'folder' | 'markdown' }[]) {
	const response = await apiFetch('/api/v1/workspace/trash/restore', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ items })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to restore items');
	}

	return response.json();
}

/**
 * Permanently deletes a list of items from the trash
 */
export async function permanentDeleteItems(items: { id: string; type: 'folder' | 'markdown' }[]) {
	const response = await apiFetch('/api/v1/workspace/trash', {
		method: 'DELETE',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ items })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to permanently delete items');
	}

	return response.json();
}
