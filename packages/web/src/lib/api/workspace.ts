import { apiFetch } from '$lib/api';

export type FileItem = {
	id: string;
	type: 'folder' | 'document';
	documentType?: 'rich_text' | 'table' | string;
	preferredImageTargetId?: 'managed-r2' | string;
	myRole?: 'owner' | 'collaborator' | 'editor' | 'viewer' | string | null;
	name: string;
	description?: string | null;
	parentId?: string | null;
	folderId?: string | null;
	title?: string | null;
	excerpt?: string | null;
	manualExcerpt?: string | null;
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

export type CreateDocumentRequest = {
	title: string;
	contentJson: { [key: string]: unknown };
	folderId?: string | null;
	documentType?: 'rich_text' | 'table';
	preferredImageTargetId?: 'managed-r2' | string;
};

export type CreateDocumentResponse = FileItem;

export type UpdateDocumentImageTargetResponse = {
	success: boolean;
	preferredImageTargetId: string;
};

export type DeleteResponse = {
	success: boolean;
	message: string;
};

export type ShareDocumentMember = {
	userId: string;
	role: 'owner' | 'collaborator' | 'editor' | 'viewer' | string;
	displayName?: string | null;
	email?: string | null;
};

export type ShareDocumentResponse = {
	documentId: string;
	members: ShareDocumentMember[];
};

export type NotificationItem = {
	id: string;
	userId: string;
	type: string;
	groupKey: string;
	data: Record<string, unknown>;
	readAt?: string | null;
	createdAt: string;
	updatedAt: string;
};

export type NotificationListResponse = {
	items: NotificationItem[];
	hasMore: boolean;
	total: number;
	unreadCount: number;
};

export type SharedDocumentItem = {
	documentId: string;
	title: string;
	excerpt: string;
	documentType: 'rich_text' | 'table' | string;
	preferredImageTargetId: 'managed-r2' | string;
	folderId?: string | null;
	ownerUserId: string;
	ownerDisplayName?: string | null;
	myRole: 'owner' | 'collaborator' | 'editor' | 'viewer' | string;
	createdAt: string;
	updatedAt: string;
};

export type SharedDocumentListResponse = {
	items: SharedDocumentItem[];
	hasMore: boolean;
	total: number;
};

export type SharedDocumentSummaryResponse = {
	hasSharedDocuments: boolean;
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
	type?: 'all' | 'folders' | 'documents';
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
		throw new Error(error.Message || error.message || 'Failed to create folder');
	}

	return response.json();
}

/**
 * Creates a new document document
 */
export async function createDocument(request: CreateDocumentRequest): Promise<CreateDocumentResponse> {
	const response = await apiFetch('/api/v1/workspace/documents', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify(request)
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.Message || error.message || 'Failed to create document');
	}

	return response.json();
}

/**
 * Deletes a file (soft delete)
 */
export async function deleteFile(id: string, type: 'folder' | 'document'): Promise<DeleteResponse> {
	const response = await apiFetch(`/api/v1/workspace/files/${id}?type=${type}`, {
		method: 'DELETE'
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to delete file');
	}

	return response.json();
}

/**
 * Batch deletes multiple files (folders and documents)
 */
export async function batchDeleteFiles(items: { id: string; type: 'folder' | 'document' }[]): Promise<{ success: boolean; message: string; failedItems?: { id: string; type: string; reason: string }[] }> {
	const response = await apiFetch('/api/v1/workspace/files/batch-delete', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ items })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to delete files');
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
	type: 'folder' | 'document';
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
export async function restoreItems(items: { id: string; type: 'folder' | 'document' }[]) {
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
export async function permanentDeleteItems(items: { id: string; type: 'folder' | 'document' }[]) {
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

/**
 * Get document details by ID
 */
export async function getDocumentDetails(id: string): Promise<FileItem> {
	const response = await apiFetch(`/api/v1/workspace/files/${id}?type=document`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch document details');
	}

	return response.json();
}

export async function getSharedDocumentSummary(): Promise<SharedDocumentSummaryResponse> {
	const response = await apiFetch('/api/v1/workspace/shared/summary');
	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch shared document summary');
	}
	return response.json();
}

export async function getSharedDocuments(params: {
	limit?: number;
	offset?: number;
}): Promise<SharedDocumentListResponse> {
	const queryParams = new URLSearchParams();
	if (params.limit !== undefined) {
		queryParams.set('limit', String(params.limit));
	}
	if (params.offset !== undefined) {
		queryParams.set('offset', String(params.offset));
	}
	const query = queryParams.toString();
	const response = await apiFetch(
		query ? `/api/v1/workspace/shared/documents?${query}` : '/api/v1/workspace/shared/documents'
	);
	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch shared documents');
	}
	return response.json();
}

export async function leaveSharedDocument(documentId: string): Promise<{ success?: boolean; message?: string }> {
	const response = await apiFetch(`/api/v1/workspace/documents/${documentId}/shares/me`, {
		method: 'DELETE'
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to leave shared document');
	}

	if (response.status === 204) {
		return { success: true };
	}

	return response.json();
}

export async function listDocumentMembers(documentId: string): Promise<ShareDocumentResponse> {
	const response = await apiFetch(`/api/v1/workspace/documents/${documentId}/shares`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch document members');
	}

	return response.json();
}

export async function inviteDocumentByEmail(
	documentId: string,
	email: string,
	role: 'viewer' | 'editor' | 'collaborator'
): Promise<ShareDocumentResponse> {
	const response = await apiFetch(`/api/v1/workspace/documents/${documentId}/invites`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ email, role })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to invite collaborator');
	}

	return response.json();
}

export async function removeDocumentMember(
	documentId: string,
	userId: string
): Promise<ShareDocumentResponse> {
	const response = await apiFetch(`/api/v1/workspace/documents/${documentId}/shares/${userId}`, {
		method: 'DELETE'
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to remove member');
	}

	return response.json();
}

export async function updateDocumentMemberRole(
	documentId: string,
	userId: string,
	role: 'viewer' | 'editor' | 'collaborator'
): Promise<ShareDocumentResponse> {
	const response = await apiFetch(`/api/v1/workspace/documents/${documentId}/shares`, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ userId, role })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to update member role');
	}

	return response.json();
}

export async function listNotifications(params?: {
	type?: string;
	unread?: boolean;
	limit?: number;
	offset?: number;
}): Promise<NotificationListResponse> {
	const query = new URLSearchParams();
	if (params?.type) query.set('type', params.type);
	if (params?.unread) query.set('unread', '1');
	if (params?.limit !== undefined) query.set('limit', String(params.limit));
	if (params?.offset !== undefined) query.set('offset', String(params.offset));

	const url = query.toString()
		? `/api/v1/notifications?${query.toString()}`
		: '/api/v1/notifications';
	const response = await apiFetch(url);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch notifications');
	}

	return response.json();
}

export async function acceptDocumentInvite(inviteId: string): Promise<{ success?: boolean; message?: string }> {
	const response = await apiFetch(`/api/v1/workspace/document-invites/${inviteId}/accept`, {
		method: 'POST'
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to accept invite');
	}

	if (response.status === 204) {
		return { success: true };
	}

	return response.json();
}

export async function declineDocumentInvite(inviteId: string): Promise<{ success?: boolean; message?: string }> {
	const response = await apiFetch(`/api/v1/workspace/document-invites/${inviteId}/decline`, {
		method: 'POST'
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to decline invite');
	}

	if (response.status === 204) {
		return { success: true };
	}

	return response.json();
}

export async function clearNotifications(): Promise<{ success: boolean; clearedCount: number }> {
	const response = await apiFetch('/api/v1/notifications', {
		method: 'DELETE'
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to clear notifications');
	}

	return response.json();
}

/**
 * Update document title
 */
export async function updateDocumentTitle(id: string, title: string): Promise<{ success: boolean }> {
	const response = await apiFetch(`/api/v1/workspace/documents/${id}/title`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ title })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to update title');
	}

	return response.json();
}

export async function updateDocumentExcerpt(
	id: string,
	excerpt: string
): Promise<{ success: boolean; excerpt: string; manualExcerpt: string }> {
	const response = await apiFetch(`/api/v1/workspace/documents/${id}/excerpt`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ excerpt })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to update excerpt');
	}

	return response.json();
}

export async function updateDocumentImageTarget(
	id: string,
	preferredImageTargetId: string
): Promise<UpdateDocumentImageTargetResponse> {
	const response = await apiFetch(`/api/v1/workspace/documents/${id}/image-target`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ preferredImageTargetId })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to update image target');
	}

	return response.json();
}

/**
 * Update folder name
 */
export async function updateFolderName(id: string, name: string): Promise<{ success: boolean }> {
	const response = await apiFetch(`/api/v1/workspace/folders/${id}/name`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ name })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to update folder name');
	}

	return response.json();
}

/**
 * Update file name (unified API for both folder and document)
 */
export async function updateFileName(id: string, type: 'folder' | 'document', name: string): Promise<{ success: boolean }> {
	if (type === 'folder') {
		return updateFolderName(id, name);
	}
	return updateDocumentTitle(id, name);
}

/**
 * Move document document to a different folder
 */
export async function moveDocument(id: string, folderId: string | null): Promise<{ success: boolean; message: string; updatedAt: string }> {
	const response = await apiFetch(`/api/v1/workspace/documents/${id}/move`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ folderId })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to move document');
	}

	return response.json();
}

/**
 * Move folder to a different parent folder
 */
export async function moveFolder(id: string, parentId: string | null): Promise<{ success: boolean; message: string; updatedAt: string }> {
	const response = await apiFetch(`/api/v1/workspace/folders/${id}/move`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ parentId })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to move folder');
	}

	return response.json();
}

/**
 * Unified API for moving both folder and document
 */
export async function moveFile(id: string, type: 'folder' | 'document', targetId: string | null): Promise<{ success: boolean; message: string; updatedAt: string }> {
	if (type === 'folder') {
		return moveFolder(id, targetId);
	}
	return moveDocument(id, targetId);
}

/**
 * Batch moves multiple files and folders to a new destination
 */
export async function batchMoveFiles(
	items: { id: string; type: 'folder' | 'document' }[],
	destinationFolderId: string | null
): Promise<{
	success: boolean;
	message: string;
	movedCount: number;
	failedItems?: { id: string; type: string; reason: string }[];
}> {
	const response = await apiFetch('/api/v1/workspace/files/batch-move', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ items, destinationFolderId })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to move files');
	}

	return response.json();
}

/**
 * Fetches all folders for the current user (for move dialog)
 */
export async function getAllFolders(params: {
	parent_id?: string | null;
}): Promise<FileItem[]> {
	const queryParams = new URLSearchParams();
	queryParams.set('type', 'folders');
	
	if (params.parent_id !== undefined && params.parent_id !== null) {
		queryParams.set('parent_id', params.parent_id);
	}
	
	// Fetch all items by using a large limit
	queryParams.set('limit', '1000');
	queryParams.set('offset', '0');

	const response = await apiFetch(`/api/v1/workspace/files?${queryParams.toString()}`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch folders');
	}

	const data: FileListResponse = await response.json();
	return data.items || [];
}
