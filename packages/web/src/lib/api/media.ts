import { apiFetch } from '$lib/api';

export type MediaAssetItem = {
	id: string;
	kind: 'image' | 'video' | 'file' | string;
	filename: string;
	mimeType: string;
	fileSize: number;
	visibility: 'private' | 'public' | string;
	status: 'ready' | 'pending_delete' | 'deleted' | 'failed' | string;
	referenceCount: number;
	deletable: boolean;
	documentId?: string | null;
	createdAt: string;
	updatedAt: string;
};

export type MediaAssetListResponse = {
	items: MediaAssetItem[];
	hasMore: boolean;
	total: number;
};

export type MediaAssetReferenceDocument = {
	documentId: string;
	title: string;
	updatedAt: string;
};

export type MediaAssetReferencesResponse = {
	assetId: string;
	referenceCount: number;
	documents: MediaAssetReferenceDocument[];
};

export type MediaAssetURLResponse = {
	assetId: string;
	url: string;
	expiresAt: string;
};

type ListMediaAssetsParams = {
	kind?: 'all' | 'image' | 'video' | 'file';
	status?: 'all' | 'ready' | 'pending_delete' | 'deleted' | 'failed';
	q?: string;
	limit?: number;
	offset?: number;
};

async function parseJSONOrThrow<T>(response: Response, fallbackMessage: string): Promise<T> {
	if (!response.ok) {
		const error = await response.json().catch(() => ({}));
		throw new Error(error.message || error.error || fallbackMessage);
	}
	return response.json() as Promise<T>;
}

export async function listMediaAssets(params: ListMediaAssetsParams): Promise<MediaAssetListResponse> {
	const query = new URLSearchParams();
	if (params.kind) query.set('kind', params.kind);
	if (params.status) query.set('status', params.status);
	if (params.q) query.set('q', params.q);
	if (typeof params.limit === 'number') query.set('limit', String(params.limit));
	if (typeof params.offset === 'number') query.set('offset', String(params.offset));

	const response = await apiFetch(`/api/v1/media/assets?${query.toString()}`);
	return parseJSONOrThrow<MediaAssetListResponse>(response, 'Failed to fetch media assets');
}

export async function getMediaAssetReferences(assetId: string): Promise<MediaAssetReferencesResponse> {
	const response = await apiFetch(`/api/v1/media/assets/${assetId}/references`);
	return parseJSONOrThrow<MediaAssetReferencesResponse>(response, 'Failed to fetch asset references');
}

export async function getMediaAssetURL(assetId: string): Promise<MediaAssetURLResponse> {
	const response = await apiFetch(`/api/v1/media/assets/${assetId}/url`);
	return parseJSONOrThrow<MediaAssetURLResponse>(response, 'Failed to fetch media url');
}

export async function deleteMediaAsset(assetId: string): Promise<void> {
	const response = await apiFetch(`/api/v1/media/assets/${assetId}`, {
		method: 'DELETE'
	});
	if (!response.ok) {
		const error = await response.json().catch(() => ({}));
		throw new Error(error.message || error.error || 'Failed to delete asset');
	}
}
