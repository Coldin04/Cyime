import { apiFetch } from '$lib/api';
import type { JSONContent } from '@tiptap/core';

export type DocumentContent = {
	id: string;
	documentId: string;
	contentJson: JSONContent;
	plainText: string;
	contentVersion: number;
	createdAt: string;
	updatedAt: string;
};

export type UpdateContentResponse = {
	success: boolean;
	contentVersion: number;
	updatedAt: string;
};

export type UploadAssetResponse = {
	id: string;
	assetId: string;
	documentId: string;
	kind: 'image' | 'video' | 'file' | string;
	filename: string;
	mimeType: string;
	fileSize: number;
	storageProvider: string;
	objectKey: string;
	url: string;
	expiresAt?: string;
	visibility: 'private' | 'public' | string;
};

export type AssetReadURLResponse = {
	assetId: string;
	url: string;
	expiresAt: string;
};

export type ResolveAssetURLsItem = {
	assetId: string;
	url?: string;
	expiresAt?: string;
	error?: string;
	code?: string;
};

export type ResolveAssetURLsResponse = {
	items: ResolveAssetURLsItem[];
};

export type UploadDocumentImageResponse = {
	targetId: 'managed-r2' | string;
	mode: 'managed_asset' | 'external_url' | string;
	url: string;
	assetId?: string;
	expiresAt?: string;
};

export type EditorAPIError = Error & {
	code?: string;
};

async function parseJSONResponse<T>(response: Response, fallbackMessage: string): Promise<T> {
	const raw = await response.text();
	if (!raw) {
		throw new Error(`${fallbackMessage} (status ${response.status}, empty response body)`);
	}

	try {
		return JSON.parse(raw) as T;
	} catch {
		throw new Error(
			`${fallbackMessage} (status ${response.status}, body: ${raw.slice(0, 240)})`
		);
	}
}

export async function getDocumentContent(documentId: string): Promise<DocumentContent> {
	const response = await apiFetch(`/api/v1/edit/documents/${documentId}/content`);

	if (!response.ok) {
		const error = await parseJSONResponse<{ message?: string }>(
			response,
			'Failed to fetch document content'
		);
		throw new Error(error.message || 'Failed to fetch document content');
	}

	return parseJSONResponse<DocumentContent>(response, 'Failed to parse document content response');
}

export async function getPublicDocumentContent(documentId: string): Promise<DocumentContent> {
	const response = await apiFetch(`/api/v1/public/documents/${documentId}/content`);

	if (!response.ok) {
		const error = await parseJSONResponse<{ message?: string }>(
			response,
			'Failed to fetch public document content'
		);
		const err = new Error(error.message || 'Failed to fetch public document content') as Error & {
			status?: number;
		};
		err.status = response.status;
		throw err;
	}

	return parseJSONResponse<DocumentContent>(
		response,
		'Failed to parse public document content response'
	);
}

export async function updateDocumentContent(
	documentId: string,
	contentJson: JSONContent
): Promise<UpdateContentResponse> {
	const response = await apiFetch(`/api/v1/edit/documents/${documentId}/content`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ contentJson })
	});

	if (!response.ok) {
		const error = await parseJSONResponse<{ message?: string }>(
			response,
			'Failed to update document content'
		);
		throw new Error(error.message || 'Failed to update document content');
	}

	return parseJSONResponse<UpdateContentResponse>(
		response,
		'Failed to parse document update response'
	);
}

export async function uploadDocumentAsset(
	documentId: string,
	file: File,
	visibility: 'private' | 'public' = 'private'
): Promise<UploadAssetResponse> {
	const formData = new FormData();
	formData.append('file', file);
	formData.append('visibility', visibility);

	const response = await apiFetch(`/api/v1/edit/documents/${documentId}/assets`, {
		method: 'POST',
		body: formData
	});

	if (!response.ok) {
		const error = await parseJSONResponse<{ message?: string }>(response, 'Failed to upload asset');
		throw new Error(error.message || 'Failed to upload asset');
	}

	return parseJSONResponse<UploadAssetResponse>(response, 'Failed to parse upload response');
}

export async function pasteDocumentImage(
	documentId: string,
	file: File
): Promise<UploadDocumentImageResponse> {
	const formData = new FormData();
	formData.append('file', file);

	const response = await apiFetch(`/api/v1/edit/documents/${documentId}/paste-image`, {
		method: 'POST',
		body: formData
	});

	if (!response.ok) {
		const error = await parseJSONResponse<{ message?: string; code?: string }>(
			response,
			'Failed to upload pasted image'
		);
		const apiError = new Error(
			error.message || 'Failed to upload pasted image'
		) as EditorAPIError;
		apiError.code = error.code;
		throw apiError;
	}

	return parseJSONResponse<UploadDocumentImageResponse>(
		response,
		'Failed to parse pasted image upload response'
	);
}

export async function getAssetReadURL(assetId: string): Promise<AssetReadURLResponse> {
	const response = await apiFetch(`/api/v1/media/assets/${assetId}/url`);
	if (!response.ok) {
		const error = await parseJSONResponse<{ message?: string }>(
			response,
			'Failed to get asset read URL'
		);
		throw new Error(error.message || 'Failed to get asset read URL');
	}
	return parseJSONResponse<AssetReadURLResponse>(response, 'Failed to parse asset read URL response');
}

export async function resolveAssetReadURLs(assetIds: string[]): Promise<ResolveAssetURLsResponse> {
	const response = await apiFetch('/api/v1/media/assets/resolve', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ assetIds })
	});
	if (!response.ok) {
		const error = await parseJSONResponse<{ message?: string }>(
			response,
			'Failed to resolve asset read URLs'
		);
		throw new Error(error.message || 'Failed to resolve asset read URLs');
	}
	return parseJSONResponse<ResolveAssetURLsResponse>(
		response,
		'Failed to parse asset read URL resolve response'
	);
}
