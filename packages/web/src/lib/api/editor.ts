import { apiFetch } from '$lib/api';

export type DocumentContent = {
	id: string;
	documentId: string;
	content: string;
	contentJson: string;
	contentMarkdown: string;
	plainText: string;
	createdAt: string;
	updatedAt: string;
};

export type UpdateContentResponse = {
	success: boolean;
	updatedAt: string;
};

export async function getDocumentContent(documentId: string): Promise<DocumentContent> {
	const response = await apiFetch(`/api/v1/edit/documents/${documentId}/content`);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to fetch document content');
	}

	return response.json();
}

export async function updateDocumentContent(
	documentId: string,
	content: string
): Promise<UpdateContentResponse> {
	const response = await apiFetch(`/api/v1/edit/documents/${documentId}/content`, {
		method: 'PUT',
		headers: {
			'Content-Type': 'application/json'
		},
		body: JSON.stringify({ content })
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Failed to update document content');
	}

	return response.json();
}
