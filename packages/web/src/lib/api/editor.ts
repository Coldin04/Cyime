import { apiFetch } from '$lib/api';

export type MarkdownContent = {
	id: string;
	markdownId: string;
	version: number;
	content: string;
	createdAt: string;
};

export type UpdateContentRequest = {
	content: string;
};

export type UpdateContentResponse = {
	success: boolean;
	version: number;
	updatedAt: string;
};

export type VersionInfo = {
	id: string;
	version: number;
	createdAt: string;
};

export type VersionsResponse = {
	versions: VersionInfo[];
};

/**
 * Get the latest content of a markdown document
 */
export async function getMarkdownContent(markdownId: string): Promise<MarkdownContent> {
	const response = await apiFetch(`/api/v1/workspace/markdowns/${markdownId}/content`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch markdown content');
	}

	return response.json();
}

/**
 * Update markdown content (creates a new version)
 */
export async function updateMarkdownContent(
	markdownId: string,
	content: string
): Promise<UpdateContentResponse> {
	const response = await apiFetch(`/api/v1/workspace/markdowns/${markdownId}/content`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ content })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to update markdown content');
	}

	return response.json();
}

/**
 * Get all versions of a markdown document
 */
export async function getMarkdownVersions(markdownId: string): Promise<VersionsResponse> {
	const response = await apiFetch(`/api/v1/workspace/markdowns/${markdownId}/versions`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch markdown versions');
	}

	return response.json();
}

/**
 * Get a specific version of markdown content
 */
export async function getMarkdownContentByVersion(
	markdownId: string,
	version: number
): Promise<MarkdownContent> {
	const response = await apiFetch(
		`/api/v1/workspace/markdowns/${markdownId}/versions/${version}`
	);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch markdown content by version');
	}

	return response.json();
}
